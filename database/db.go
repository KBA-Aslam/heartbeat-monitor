// ============================================================
// Distributed Heartbeat Monitoring System
// Khalid Bin Abdullah — ID: 7200671
//
// database/db.go
// Handles all SQLite database operations.
// Creates tables for users and devices on startup.
// Provides functions to query and modify both tables.
// ============================================================

package database

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

// DB is the shared database connection used across the app
var DB *sql.DB

// User represents a row in the users table
type User struct {
	ID       int
	Username string
	Password string // stored as bcrypt hash
}

// Device represents a row in the devices table
type Device struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	RegisteredBy string `json:"registered_by"`
}

// Init opens the SQLite file and creates tables if they don't exist yet
func Init(path string) {
	var err error
	DB, err = sql.Open("sqlite", path)
	if err != nil {
		log.Fatalf("[DB] Failed to open database: %v", err)
	}

	createTables()
	log.Println("[DB] Database ready:", path)
}

// createTables sets up users and devices tables
func createTables() {
	userTable := `
	CREATE TABLE IF NOT EXISTS users (
		id       INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL
	);`

	deviceTable := `
	CREATE TABLE IF NOT EXISTS devices (
		id             INTEGER PRIMARY KEY AUTOINCREMENT,
		name           TEXT UNIQUE NOT NULL,
		description    TEXT,
		registered_by  TEXT NOT NULL
	);`

	if _, err := DB.Exec(userTable); err != nil {
		log.Fatalf("[DB] Failed to create users table: %v", err)
	}
	if _, err := DB.Exec(deviceTable); err != nil {
		log.Fatalf("[DB] Failed to create devices table: %v", err)
	}
}

// ── User operations ─────────────────────────────────────────

// CreateUser inserts a new user with an already-hashed password
func CreateUser(username, hashedPassword string) error {
	_, err := DB.Exec(`INSERT INTO users (username, password) VALUES (?, ?)`,
		username, hashedPassword)
	return err
}

// GetUser fetches a user by username, returns nil if not found
func GetUser(username string) (*User, error) {
	row := DB.QueryRow(`SELECT id, username, password FROM users WHERE username = ?`, username)
	u := &User{}
	err := row.Scan(&u.ID, &u.Username, &u.Password)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

// ── Device operations ────────────────────────────────────────

// AddDevice inserts a new device into the database
func AddDevice(name, description, registeredBy string) error {
	_, err := DB.Exec(
		`INSERT INTO devices (name, description, registered_by) VALUES (?, ?, ?)`,
		name, description, registeredBy)
	return err
}

// GetAllDevices returns every device in the database
func GetAllDevices() ([]Device, error) {
	rows, err := DB.Query(`SELECT id, name, description, registered_by FROM devices`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []Device
	for rows.Next() {
		var d Device
		if err := rows.Scan(&d.ID, &d.Name, &d.Description, &d.RegisteredBy); err != nil {
			return nil, err
		}
		devices = append(devices, d)
	}
	return devices, nil
}

// DeleteDevice removes a device by ID
func DeleteDevice(id int) error {
	_, err := DB.Exec(`DELETE FROM devices WHERE id = ?`, id)
	return err
}

// DeviceExists checks if a device name is already registered
func DeviceExists(name string) bool {
	var count int
	DB.QueryRow(`SELECT COUNT(*) FROM devices WHERE name = ?`, name).Scan(&count)
	return count > 0
}
