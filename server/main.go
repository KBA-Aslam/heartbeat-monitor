// ============================================================
// Distributed Heartbeat Monitoring System
// Khalid Bin Abdullah — ID: 7200671
//
// server/main.go
// Central monitoring server. Handles heartbeats, device status,
// user authentication, device management, and serves the dashboard.
// ============================================================

package main

import (
	"encoding/json"
	"fmt"
	"heartbeat-monitor/auth"
	"heartbeat-monitor/database"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// ClientInfo holds the runtime heartbeat state of a client node
type ClientInfo struct {
	ID         string    `json:"id"`
	LastSeen   time.Time `json:"last_seen"`
	Status     string    `json:"status"`
	LastStatus string    `json:"last_status"`
}

// HeartbeatRequest is the JSON payload a client sends to /heartbeat
type HeartbeatRequest struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// Store is a thread-safe in-memory store for heartbeat states
type Store struct {
	mu      sync.Mutex
	clients map[string]*ClientInfo
}

var store = &Store{
	clients: make(map[string]*ClientInfo),
}

const offlineTimeout = 10 * time.Second

// ── Middleware ───────────────────────────────────────────────

// withCORS adds cross-origin headers to a handler
func withCORS(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		h(w, r)
	}
}

// requireAuth checks for a valid session cookie before allowing access
func requireAuth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		if _, ok := auth.GetSession(cookie.Value); !ok {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		h(w, r)
	}
}

// ── Auth handlers ────────────────────────────────────────────

func loginPageHandler(w http.ResponseWriter, r *http.Request) {
	html, err := os.ReadFile("dashboard/login.html")
	if err != nil {
		http.Error(w, "Login page not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write(html)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	user, err := database.GetUser(username)
	if err != nil || user == nil || !auth.CheckPassword(password, user.Password) {
		http.Redirect(w, r, "/login?error=1", http.StatusFound)
		return
	}

	token, err := auth.GenerateToken()
	if err != nil {
		http.Error(w, "Could not create session", http.StatusInternalServerError)
		return
	}
	auth.SaveSession(token, username)

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   7200,
	})

	log.Printf("[Auth] User logged in: %s", username)
	http.Redirect(w, r, "/", http.StatusFound)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		auth.DeleteSession(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:   "session_token",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	http.Redirect(w, r, "/login", http.StatusFound)
}

// ── Dashboard handler ────────────────────────────────────────

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	html, err := os.ReadFile("dashboard/index.html")
	if err != nil {
		http.Error(w, "Dashboard not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write(html)
}

// ── Heartbeat handlers ───────────────────────────────────────

func heartbeatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	store.mu.Lock()
	if _, exists := store.clients[req.ID]; !exists {
		store.clients[req.ID] = &ClientInfo{ID: req.ID}
		log.Printf("[+] New client registered: %s", req.ID)
	}
	store.clients[req.ID].LastSeen = time.Now()
	store.clients[req.ID].Status = "online"
	store.clients[req.ID].LastStatus = req.Status
	store.mu.Unlock()

	log.Printf("[♥] Heartbeat from %s", req.ID)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, `{"message": "heartbeat received"}`)
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	store.mu.Lock()
	clients := make([]*ClientInfo, 0, len(store.clients))
	for _, c := range store.clients {
		clients = append(clients, c)
	}
	store.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clients)
}

func offlineDetector() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		store.mu.Lock()
		for _, client := range store.clients {
			if client.Status == "online" && time.Since(client.LastSeen) > offlineTimeout {
				client.Status = "offline"
				log.Printf("[!] Client went offline: %s", client.ID)
			}
		}
		store.mu.Unlock()
	}
}

// ── Device handlers ──────────────────────────────────────────

// devicesHandler handles GET /devices — returns all registered devices as JSON
func devicesHandler(w http.ResponseWriter, r *http.Request) {
	devices, err := database.GetAllDevices()
	if err != nil {
		http.Error(w, "Failed to fetch devices", http.StatusInternalServerError)
		return
	}
	if devices == nil {
		devices = []database.Device{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(devices)
}

// addDeviceHandler handles POST /devices/add — registers a new device
func addDeviceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get logged-in username from session
	cookie, _ := r.Cookie("session_token")
	session, _ := auth.GetSession(cookie.Value)

	name := r.FormValue("name")
	description := r.FormValue("description")

	if name == "" {
		http.Error(w, "Device name is required", http.StatusBadRequest)
		return
	}
	if database.DeviceExists(name) {
		http.Error(w, "Device name already exists", http.StatusConflict)
		return
	}

	if err := database.AddDevice(name, description, session.Username); err != nil {
		http.Error(w, "Failed to add device", http.StatusInternalServerError)
		return
	}

	log.Printf("[Device] Added: %s by %s", name, session.Username)
	http.Redirect(w, r, "/", http.StatusFound)
}

// deleteDeviceHandler handles POST /devices/delete — removes a device by ID
func deleteDeviceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == 0 {
		http.Error(w, "Invalid device ID", http.StatusBadRequest)
		return
	}

	if err := database.DeleteDevice(req.ID); err != nil {
		http.Error(w, "Failed to delete device", http.StatusInternalServerError)
		return
	}

	log.Printf("[Device] Deleted ID: %d", req.ID)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, `{"message": "device deleted"}`)
}

// ── Main ─────────────────────────────────────────────────────

func main() {
	database.Init("monitor.db")
	auth.SeedAdmin("admin", "admin123")

	go offlineDetector()

	// Public routes
	http.HandleFunc("/login", loginPageHandler)
	http.HandleFunc("/login/submit", loginHandler)
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/heartbeat", withCORS(heartbeatHandler))

	// Protected routes
	http.HandleFunc("/", requireAuth(dashboardHandler))
	http.HandleFunc("/status", requireAuth(withCORS(statusHandler)))
	http.HandleFunc("/devices", requireAuth(withCORS(devicesHandler)))
	http.HandleFunc("/devices/add", requireAuth(addDeviceHandler))
	http.HandleFunc("/devices/delete", requireAuth(deleteDeviceHandler))

	log.Println("========================================")
	log.Println("  Heartbeat Monitor Server running")
	log.Println("  Address  : http://localhost:8080")
	log.Println("  Login    : admin / admin123")
	log.Println("  GET  /            — dashboard (auth)")
	log.Println("  POST /heartbeat   — receive heartbeats")
	log.Println("  GET  /status      — client statuses (auth)")
	log.Println("  GET  /devices     — list devices (auth)")
	log.Println("  POST /devices/add — add device (auth)")
	log.Println("  POST /devices/delete — remove device (auth)")
	log.Println("========================================")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
