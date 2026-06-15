// ============================================================
// Distributed Heartbeat Monitoring System
// Khalid Bin Abdullah — ID: 7200671
//
// auth/auth.go
// Handles user authentication: password hashing, login
// validation, session token generation and verification.
// Sessions are stored in memory with an expiry time.
// ============================================================

package auth

import (
	"crypto/rand"
	"encoding/hex"
	"heartbeat-monitor/database"
	"log"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Session represents a logged-in user's active session
type Session struct {
	Username  string
	ExpiresAt time.Time
}

// sessionStore holds all active sessions in memory
// protected by a mutex for concurrent access
var (
	sessionStore = make(map[string]Session)
	sessionMu    sync.Mutex
)

const sessionDuration = 2 * time.Hour

// ── Password helpers ─────────────────────────────────────────

// HashPassword takes a plain password and returns a bcrypt hash
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword compares a plain password against a stored hash
func CheckPassword(plain, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// ── Session helpers ──────────────────────────────────────────

// GenerateToken creates a cryptographically random session token
func GenerateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// SaveSession stores a new session token for the given username
func SaveSession(token, username string) {
	sessionMu.Lock()
	defer sessionMu.Unlock()
	sessionStore[token] = Session{
		Username:  username,
		ExpiresAt: time.Now().Add(sessionDuration),
	}
}

// GetSession returns the session for a token, or false if invalid/expired
func GetSession(token string) (Session, bool) {
	sessionMu.Lock()
	defer sessionMu.Unlock()
	s, ok := sessionStore[token]
	if !ok || time.Now().After(s.ExpiresAt) {
		delete(sessionStore, token)
		return Session{}, false
	}
	return s, true
}

// DeleteSession removes a session (logout)
func DeleteSession(token string) {
	sessionMu.Lock()
	defer sessionMu.Unlock()
	delete(sessionStore, token)
}

// ── Seed default admin ───────────────────────────────────────

// SeedAdmin creates a default admin user if no users exist yet.
// This runs once at server startup so you can log in immediately.
func SeedAdmin(username, password string) {
	existing, err := database.GetUser(username)
	if err != nil {
		log.Printf("[Auth] Error checking for admin: %v", err)
		return
	}
	if existing != nil {
		log.Printf("[Auth] Admin user already exists: %s", username)
		return
	}

	hashed, err := HashPassword(password)
	if err != nil {
		log.Printf("[Auth] Failed to hash admin password: %v", err)
		return
	}

	if err := database.CreateUser(username, hashed); err != nil {
		log.Printf("[Auth] Failed to create admin user: %v", err)
		return
	}

	log.Printf("[Auth] Default admin created — username: %s  password: %s", username, password)
}
