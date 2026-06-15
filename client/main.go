package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

// HeartbeatRequest matches what the server expects
type HeartbeatRequest struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

const (
	serverURL       = "http://localhost:8080/heartbeat"
	heartbeatEvery  = 3 * time.Second // send a heartbeat every 3 seconds
)

func sendHeartbeat(clientID string) {
	payload := HeartbeatRequest{
		ID:     clientID,
		Status: "running",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[!] Failed to encode heartbeat: %v", err)
		return
	}

	resp, err := http.Post(serverURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("[!] Could not reach server: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.Printf("[♥] Heartbeat sent — %s", clientID)
	} else {
		log.Printf("[!] Server responded with status: %d", resp.StatusCode)
	}
}

func main() {
	// -id flag lets you set a unique client name when launching
	clientID := flag.String("id", "client-1", "Unique ID for this client")
	flag.Parse()

	fmt.Printf("========================================\n")
	fmt.Printf("  Client Node started\n")
	fmt.Printf("  ID      : %s\n", *clientID)
	fmt.Printf("  Server  : %s\n", serverURL)
	fmt.Printf("  Interval: every %v\n", heartbeatEvery)
	fmt.Printf("========================================\n")

	ticker := time.NewTicker(heartbeatEvery)
	defer ticker.Stop()

	// Send one immediately, then keep ticking
	sendHeartbeat(*clientID)

	for range ticker.C {
		sendHeartbeat(*clientID)
	}
}
