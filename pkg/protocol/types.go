package protocol

import "encoding/json"

// Message Types
const (
	TypeAuth      = "AUTH"
	TypeJobOffer  = "JOB_OFFER"
	TypeJobResult = "JOB_RESULT"
	TypeHeartbeat = "HEARTBEAT"
)

// Message is the standard wrapper for all WebSocket communications
type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// AuthPayload represents the payload for AUTH messages
type AuthPayload struct {
	DeviceID      string `json:"device_id"`
	WalletAddress string `json:"wallet_address"`
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	CpuCores      int    `json:"cpu_cores"`
}
