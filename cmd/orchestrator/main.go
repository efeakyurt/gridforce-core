package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/gridforce/core/internal/core/blockchain"
	"github.com/gridforce/core/internal/core/db"
	"github.com/gridforce/core/pkg/protocol"
)

// ProviderSession holds information about a connected provider
type ProviderSession struct {
	Conn           *websocket.Conn
	DeviceID       string
	WalletAddress  string
	Specs          string
	IP             string
	Status         string
	LastSeen       time.Time
	Tokens         int64
	BenchmarkScore int
}

var (
	// Store active connections: RemoteAddr -> *ProviderSession
	providers = make(map[string]*ProviderSession)
	mu        sync.RWMutex

	// Blockchain Client
	chainClient *blockchain.Client

	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for dev
		},
	}
)

func generateRandomKey() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback if rand fails
		return fmt.Sprintf("sk_live_%d", time.Now().UnixNano())
	}
	return "sk_live_" + hex.EncodeToString(bytes)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading:", err)
		return
	}
	
	addr := conn.RemoteAddr().String()
	log.Printf("New Provider Connected: %s\n", addr)

	// cleanup handler
	defer func() {
		conn.Close()
		mu.Lock()
		delete(providers, addr)
		mu.Unlock()
		log.Printf("Provider Disconnected: %s\n", addr)
	}()

	// Initial placeholder registration (unauthenticated)
	mu.Lock()
	providers[addr] = &ProviderSession{
		Conn:     conn,
		IP:       addr,
		Status:   "CONNECTED", 
		LastSeen: time.Now(),
	}
	mu.Unlock()

	// Update DB Status - Initial
	node := db.Node{
		ID:        addr,
		IPAddress: addr,
		Status:    "CONNECTED",
		LastSeen:  time.Now(),
	}
	db.DB.Save(&node)

	// Listen for messages
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}

		var msg protocol.Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Error unmarshalling message: %v\n", err)
			continue
		}

		// Handle Auth to capture DeviceID and Wallet
		if msg.Type == protocol.TypeAuth {
			// DEBUG: Print Raw Payload
			log.Printf("DEBUG: Raw Auth Payload: %s", string(msg.Payload))

			var authPayload protocol.AuthPayload
			if err := json.Unmarshal(msg.Payload, &authPayload); err == nil {
				// DEBUG: Print Parsed Values
				log.Printf("DEBUG: Parsed Specs: OS=%s Arch=%s Cores=%d", authPayload.OS, authPayload.Arch, authPayload.CpuCores)

				// Construct dynamic specs string
				specsStr := fmt.Sprintf("%s/%s - %d Cores", authPayload.OS, authPayload.Arch, authPayload.CpuCores)

				mu.Lock()
				if session, ok := providers[addr]; ok {
					session.DeviceID = authPayload.DeviceID
					session.WalletAddress = authPayload.WalletAddress
					session.Specs = specsStr
					session.Status = "ONLINE"
					session.LastSeen = time.Now()

					// Restore tokens & benchmark from DB if checking against existing record
					var storedNode db.Node
					if err := db.DB.First(&storedNode, "id = ?", addr).Error; err == nil {
						session.Tokens = storedNode.Tokens
						session.BenchmarkScore = storedNode.BenchmarkScore
					}
				}
				mu.Unlock()

				// Update DB with DeviceID, Wallet, and Specs
				node.WalletAddress = authPayload.WalletAddress
				node.Specs = specsStr
				node.Status = "ONLINE"
				node.LastSeen = time.Now()
				db.DB.Save(&node)
				
				fmt.Printf("Provider Authenticated: %s | Wallet: %s | Specs: %s\n", authPayload.DeviceID, authPayload.WalletAddress, specsStr)
			} else {
				log.Printf("Error unmarshalling auth payload: %v", err)
			}
		}

		if msg.Type == protocol.TypeJobResult {
			var result string
			json.Unmarshal(msg.Payload, &result)
			fmt.Printf("Job Result: %s\n", result)

			// Determine DeviceID for record and reward
			mu.Lock()
			deviceID := "unknown"
			walletAddr := ""
			
			// Benchmark Logic
			isBenchmark := false
			benchmarkScore := 0
			if strings.Contains(result, "BENCHMARK_SCORE:") {
				isBenchmark = true
				parts := strings.Split(result, "BENCHMARK_SCORE:")
				if len(parts) > 1 {
					scoreStr := strings.TrimSpace(parts[1])
					if s, err := strconv.Atoi(scoreStr); err == nil {
						benchmarkScore = s
					}
				}
			}

			if sess, ok := providers[addr]; ok {
				deviceID = sess.DeviceID
				walletAddr = sess.WalletAddress
				
				// Reward Tokens (DB Tracking) ONLY if NOT a benchmark or maybe small reward?
				// User said: "Do NOT mint tokens for benchmark jobs (optional)"
				// Let's Skip token increase for benchmarks to avoid spamming
				if !isBenchmark {
					sess.Tokens += 10
				} else {
					// Update Session Score
					sess.BenchmarkScore = benchmarkScore
				}
			}
			mu.Unlock()

			// Save Job Result
			job := db.Job{
				NodeID: deviceID, // Use DeviceID if available, else unknown
				Image:  "alpine",
				Status: "COMPLETED",
				Result: result,
			}
			db.DB.Create(&job)

			// Update Node Tokens/Score in DB
			var dbNode db.Node
			if err := db.DB.First(&dbNode, "id = ?", addr).Error; err == nil {
				if isBenchmark {
					dbNode.BenchmarkScore = benchmarkScore
					log.Printf("Node %s Benchmark Updated: %d\n", deviceID, benchmarkScore)
				} else {
					dbNode.Tokens += 10
				}
				db.DB.Save(&dbNode)
			}
			
			if !isBenchmark {
				log.Printf("Job Saved: %d | Node Rewarded: %s (+10 Tokens)\n", job.ID, deviceID)

				// Mint Tokens on Blockchain
				if chainClient != nil && walletAddr != "" && walletAddr != "0x000000000000000000000000000000000000dead" {
					tx, err := chainClient.MintToken(walletAddr, 10)
					if err != nil {
						log.Printf("Blockchain Error: Failed to mint tokens: %v\n", err)
					} else {
						log.Printf("Blockchain Tx Sent: %s | Minted 10 GRID to %s\n", tx, walletAddr)
					}
				} else {
					log.Println("Skipping Blockchain Mint: Client not init or invalid wallet")
				}
			} else {
				log.Printf("Benchmark Job Completed: Score %d", benchmarkScore)
			}
		}
	}
}

// Auth Middleware
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-KEY")
		if apiKey == "" {
			http.Error(w, "Unauthorized: Validation Failed", http.StatusUnauthorized)
			return
		}

		var customer db.Customer
		if err := db.DB.Where("api_key = ?", apiKey).First(&customer).Error; err != nil {
			http.Error(w, "Unauthorized: Invalid API Key", http.StatusUnauthorized)
			return
		}

		if customer.Credits <= 0 {
			http.Error(w, "Payment Required: Insufficient Credits", http.StatusPaymentRequired)
			return
		}

		// Deduct Credit
		customer.Credits -= 1
		db.DB.Save(&customer)

		// Proceed
		next(w, r)
	}
}

func handleJobDispatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Image string   `json:"image"`
		Cmd   []string `json:"cmd"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	mu.RLock()
	if len(providers) == 0 {
		mu.RUnlock()
		http.Error(w, "No providers available", http.StatusServiceUnavailable)
		return
	}

	// Pick first available provider
	var targetConn *websocket.Conn
	for _, sess := range providers {
		targetConn = sess.Conn
		break
	}
	mu.RUnlock()

	// Create Offer
	offerPayload, _ := json.Marshal(req)
	msg := protocol.Message{
		Type:    protocol.TypeJobOffer,
		Payload: offerPayload,
	}

	if err := targetConn.WriteJSON(msg); err != nil {
		log.Println("Failed to send job offer:", err)
		http.Error(w, "Failed to dispatch job", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Job dispatched"))
	log.Printf("Job dispatched to %s\n", targetConn.RemoteAddr())
}

// API: Get Active Nodes
func handleGetNodes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	type NodeResponse struct {
		Device          string `json:"device"`
		Wallet          string `json:"wallet"`
		Specs           string `json:"specs"`
		IP              string `json:"ip"`
		Status          string `json:"status"`
		Tokens          int64  `json:"tokens"`
		BenchmarkScore  int    `json:"benchmark_score"`
	}

	mu.RLock()
	defer mu.RUnlock()

	var nodes []NodeResponse
	for _, sess := range providers {
		nodes = append(nodes, NodeResponse{
			Device:          sess.DeviceID,
			Wallet:          sess.WalletAddress,
			Specs:           sess.Specs,
			IP:              sess.IP,
			Status:          sess.Status,
			Tokens:          sess.Tokens,
			BenchmarkScore:  sess.BenchmarkScore,
		})
	}
	json.NewEncoder(w).Encode(nodes)
}

// API: Get Last Jobs
func handleGetJobs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var jobs []db.Job
	// Get last 10 jobs order by ID desc
	if result := db.DB.Order("id desc").Limit(10).Find(&jobs); result.Error != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	
	json.NewEncoder(w).Encode(jobs)
}

// API: Admin Create Customer
func handleCreateCustomer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	newKey := generateRandomKey()
	customer := db.Customer{
		ID:      uuid.New().String(),
		ApiKey:  newKey,
		Credits: 1000,
	}

	if err := db.DB.Create(&customer).Error; err != nil {
		http.Error(w, "Failed to create customer", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"api_key": newKey,
		"credits": 1000,
		"message": "Customer created successfully",
	})
}

func main() {
	// Database Configuration
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dsn := fmt.Sprintf("host=%s user=gridforce password=secret dbname=gridforce_core port=5432 sslmode=disable", dbHost)
	db.InitDB(dsn)

	// Blockchain Configuration
	var err error
	rpcURL := os.Getenv("BLOCKCHAIN_RPC")
	if rpcURL == "" {
		rpcURL = "http://127.0.0.1:8545"
	}
	
	privKey := os.Getenv("BLOCKCHAIN_PRIVATE_KEY")
	if privKey == "" {
		log.Fatal("BLOCKCHAIN_PRIVATE_KEY is missing")
	}

	contractAddr := os.Getenv("BLOCKCHAIN_CONTRACT_ADDRESS")
	if contractAddr == "" {
		log.Fatal("BLOCKCHAIN_CONTRACT_ADDRESS is missing")
	}
	
	chainClient, err = blockchain.NewClient(rpcURL, privKey, contractAddr)
	if err != nil {
		log.Printf("Warning: Failed to initialize blockchain client: %v. Minting disabled.\n", err)
	} else {
		log.Println("Blockchain Client Initialized Successfully")
	}

	// Static Files
	fs := http.FileServer(http.Dir("./web"))
	http.Handle("/", fs)
	
	// Download Center
	http.Handle("/downloads/", http.StripPrefix("/downloads/", http.FileServer(http.Dir("./downloads"))))

	// API Endpoints
	http.HandleFunc("/ws", handleWebSocket)
	// Apply Auth Middleware to job dispatch
	http.HandleFunc("/jobs", authMiddleware(handleJobDispatch))
	http.HandleFunc("/api/nodes", handleGetNodes)
	http.HandleFunc("/api/jobs", handleGetJobs)
	// Admin API
	http.HandleFunc("/api/admin/create-customer", handleCreateCustomer)

	fmt.Println("Orchestrator running on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
