package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/gridforce/core/internal/platform/container"
	"github.com/gridforce/core/pkg/protocol"
)

// Helper for contains check
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func main() {
	flag.String("wallet", "ignored", "Flag ignored, using hardcoded wallet")
	serverAddr := flag.String("server", "46.101.96.91:8080", "Server address (e.g. 46.101.96.91:8080 or xxx.ngrok-free.app)")
	flag.Parse()

	// FORCE WALLET ADDRESS for Demo/testing
	forcedWallet := "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
	wallet := &forcedWallet
	fmt.Printf("FORCE WALLET: %s\n", *wallet)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	scheme := "ws"
	// Auto-detect secure scheme for public endpoints
	if contains(*serverAddr, "ngrok") || contains(*serverAddr, "https") {
		scheme = "wss"
	}

	u := url.URL{Scheme: scheme, Host: *serverAddr, Path: "/ws"}
	log.Printf("Connecting to Server: %s with wallet %s", u.String(), *wallet)

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	// 1. Construct Auth Payload
	authPayload := protocol.AuthPayload{
		DeviceID:      "gpu-node-01", // Static for now, could be dynamic
		WalletAddress: *wallet,
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		CpuCores:      runtime.NumCPU(),
	}

	// 2. Marshal Payload
	payloadBytes, err := json.Marshal(authPayload)
	if err != nil {
		log.Fatal("marshal auth:", err)
	}
	log.Printf("CLIENT DEBUG: Generated JSON: %s", string(payloadBytes))

	// 3. Construct Message
	authMsg := protocol.Message{
		Type:    protocol.TypeAuth,
		Payload: payloadBytes,
	}

	// 4. Send Message
	if err := c.WriteJSON(authMsg); err != nil {
		log.Println("write auth:", err)
		return
	}
	fmt.Printf("Sent AUTH message: OS=%s Arch=%s Cores=%d\n", authPayload.OS, authPayload.Arch, authPayload.CpuCores)

	done := make(chan struct{})

	// Listen for messages
	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s", message)

			var msg protocol.Message
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Println("unmarshal:", err)
				continue
			}

			if msg.Type == protocol.TypeJobOffer {
				var offer struct {
					Image string   `json:"image"`
					Cmd   []string `json:"cmd"`
				}
				if err := json.Unmarshal(msg.Payload, &offer); err != nil {
					log.Println("unmarshal offer:", err)
					continue
				}

				fmt.Printf("Received Job Offer: %s %v\n", offer.Image, offer.Cmd)
				
				// Execute container
				logs, err := container.RunContainer(context.Background(), offer.Image, offer.Cmd)
				if err != nil {
					log.Printf("Container run failed: %v\n", err)
					logs = fmt.Sprintf("Error: %v", err)
				}
				
				fmt.Printf("Job Completed. Result: %s\n", logs)

				// Send Result
				resultPayload, _ := json.Marshal(logs)
				resultMsg := protocol.Message{
					Type:    protocol.TypeJobResult,
					Payload: resultPayload,
				}
				c.WriteJSON(resultMsg)
			}
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println("interrupt")
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}
