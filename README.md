# Distributed Heartbeat Monitoring System
**Khalid Bin Abdullah — ID: 7200671**

A distributed heartbeat monitoring system built in Go. Multiple client nodes send periodic heartbeat messages to a central server. The server tracks each client's status in real time, detects failures when a client goes silent, and provides a secure web dashboard for authenticated users to monitor clients and manage registered devices.

---

## Project Structure

```
heartbeat-monitor/
├── go.mod                  — Go module and dependencies
├── monitor.db              — SQLite database (auto-created on first run)
├── server/
│   └── main.go             — Central monitoring server
├── client/
│   └── main.go             — Client node (run multiple instances)
├── auth/
│   └── auth.go             — Authentication: hashing, sessions, tokens
├── database/
│   └── db.go               — SQLite: users and devices tables
└── dashboard/
    ├── index.html          — Main dashboard (protected)
    └── login.html          — Login page (public)
```

---

## How to Run

### 1. Install dependencies
```
go get modernc.org/sqlite
go get golang.org/x/crypto
go mod tidy
```

### 2. Start the server
```
go run server/main.go
```
Server starts at http://localhost:8080
Default login — **username:** `admin` **password:** `admin123`

### 3. Start client nodes
Open a new terminal for each client:
```
go run client/main.go -id=client-1
go run client/main.go -id=client-2
go run client/main.go -id=client-3
```

### 4. Open the dashboard
```
http://localhost:8080
```
Log in, watch clients appear as online. Stop a client and it turns offline after 10 seconds.

---

## API Endpoints

| Method | Endpoint           | Auth | Description                  |
|--------|--------------------|------|------------------------------|
| GET    | /                  | ✅   | Web dashboard                |
| GET    | /login             | ❌   | Login page                   |
| POST   | /login/submit      | ❌   | Process login form           |
| GET    | /logout            | ✅   | Log out and clear session    |
| POST   | /heartbeat         | ❌   | Client sends heartbeat       |
| GET    | /status            | ✅   | Get all client statuses      |
| GET    | /devices           | ✅   | List all registered devices  |
| POST   | /devices/add       | ✅   | Register a new device        |
| POST   | /devices/delete    | ✅   | Remove a device by ID        |

---

## Course Topics Covered

| Topic                        | Where in the project                              |
|------------------------------|---------------------------------------------------|
| Distributed systems          | Multiple client nodes reporting to one server     |
| Client-server communication  | HTTP REST API between clients and server          |
| Concurrent programming       | `sync.Mutex`, goroutines, `go offlineDetector()`  |
| Failure detection            | Background goroutine marks silent clients offline |
| Web services in Go           | `net/http` handlers, REST API                     |
| Authentication               | bcrypt hashing, session tokens, cookie middleware |
| Database storage             | SQLite via `modernc.org/sqlite`                   |
| Network communication        | HTTP over TCP, JSON payloads                      |

---

## Technologies Used
- **Go (Golang)** — server and client
- **modernc.org/sqlite** — pure Go SQLite driver (no CGO required)
- **golang.org/x/crypto** — bcrypt password hashing
- **net/http** — HTTP server and REST API
- **goroutines & sync.Mutex** — concurrent request handling
- **HTML / CSS / JavaScript** — web dashboard
