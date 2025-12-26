package main

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

// Issue represents a captured issue
type Issue struct {
	ID              string                 `json:"id"`
	Timestamp       time.Time              `json:"timestamp"`
	ScreenshotPath  string                 `json:"screenshot_path"`
	ScenarioName    string                 `json:"scenario_name"`
	URL             string                 `json:"url"`
	Description     string                 `json:"description"`
	ContextData     map[string]interface{} `json:"context_data"`
	AgentSessionID  string                 `json:"agent_session_id,omitempty"`
	Status          string                 `json:"status"`
	ResolutionNotes string                 `json:"resolution_notes,omitempty"`
	Tags            []string               `json:"tags,omitempty"`
}

// AgentSpawnRequest represents a request to spawn an agent
type AgentSpawnRequest struct {
	IssueID     string                 `json:"issue_id"`
	AgentType   string                 `json:"agent_type"`
	Context     map[string]interface{} `json:"context"`
	Screenshot  string                 `json:"screenshot"`
	Description string                 `json:"description"`
}

// AgentSession represents an agent working session
type AgentSession struct {
	ID        string     `json:"id"`
	IssueID   string     `json:"issue_id"`
	AgentType string     `json:"agent_type"`
	StartTime time.Time  `json:"start_time"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	Status    string     `json:"status"`
	Output    string     `json:"output,omitempty"`
}

//go:embed webui/*
var embeddedWebUI embed.FS

var db *sql.DB

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "vrooli-assistant",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// Initialize database connection
	initDB()

	// Create tables if they don't exist
	createTables()

	// Serve embedded web UI assets
	webFS, err := fs.Sub(embeddedWebUI, "webui")
	if err != nil {
		log.Fatalf("Failed to load embedded web UI assets: %v", err)
	}
	spa := newSPAFileServer(http.FS(webFS), "index.html")

	// Set up routes
	router := newRouter(spa)

	// Enable CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:*", "file://*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(router)

	// Start server with graceful shutdown
	if err := server.Run(server.Config{
		Handler: handler,
		Cleanup: func(ctx context.Context) error {
			if db != nil {
				return db.Close()
			}
			return nil
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func newRouter(staticHandler http.Handler) *mux.Router {
	router := mux.NewRouter()

	// Health check - both paths for compatibility
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/api/health", healthHandler).Methods("GET")
	router.HandleFunc("/api/v1/assistant/status", statusHandler).Methods("GET")

	// Issue capture
	router.HandleFunc("/api/v1/assistant/capture", captureHandler).Methods("POST")

	// Agent spawning
	router.HandleFunc("/api/v1/assistant/spawn-agent", spawnAgentHandler).Methods("POST")

	// History
	router.HandleFunc("/api/v1/assistant/history", historyHandler).Methods("GET")

	// Issue details
	router.HandleFunc("/api/v1/assistant/issues/{id}", issueHandler).Methods("GET")

	// Update issue status
	router.HandleFunc("/api/v1/assistant/issues/{id}/status", updateStatusHandler).Methods("PUT")

	if staticHandler != nil {
		router.PathPrefix("/").Handler(staticHandler)
	}

	return router
}

type spaFileServer struct {
	fs    http.FileSystem
	index string
}

func newSPAFileServer(fileSystem http.FileSystem, index string) http.Handler {
	return spaFileServer{fs: fileSystem, index: index}
}

func (s spaFileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")

	if path == "" {
		s.serveIndex(w, r)
		return
	}

	file, err := s.fs.Open(path)
	if err != nil {
		s.serveIndex(w, r)
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		s.serveIndex(w, r)
		return
	}

	if info.IsDir() {
		nestedPath := strings.TrimSuffix(path, "/") + "/" + s.index
		nestedFile, nestedErr := s.fs.Open(nestedPath)
		if nestedErr != nil {
			s.serveIndex(w, r)
			return
		}
		defer nestedFile.Close()

		nestedInfo, nestedInfoErr := nestedFile.Stat()
		if nestedInfoErr != nil {
			s.serveIndex(w, r)
			return
		}

		http.ServeContent(w, r, nestedInfo.Name(), nestedInfo.ModTime(), nestedFile)
		return
	}

	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}

func (s spaFileServer) serveIndex(w http.ResponseWriter, r *http.Request) {
	indexFile, err := s.fs.Open(s.index)
	if err != nil {
		http.Error(w, "Vrooli Assistant web UI is not built yet. Run `npm install && npm run build` inside ui/.", http.StatusServiceUnavailable)
		return
	}
	defer indexFile.Close()

	info, err := indexFile.Stat()
	if err != nil {
		http.Error(w, "Vrooli Assistant web UI is unavailable.", http.StatusInternalServerError)
		return
	}

	http.ServeContent(w, r, info.Name(), info.ModTime(), indexFile)
}

func initDB() {
	var err error
	db, err = database.Connect(context.Background(), database.Config{
		Driver: "postgres",
	})
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}
}

func createTables() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS issues (
			id UUID PRIMARY KEY,
			timestamp TIMESTAMP NOT NULL,
			screenshot_path TEXT,
			scenario_name VARCHAR(255),
			url TEXT,
			description TEXT NOT NULL,
			context_data JSONB,
			agent_session_id UUID,
			status VARCHAR(50) NOT NULL,
			resolution_notes TEXT,
			tags TEXT[]
		)`,
		`CREATE TABLE IF NOT EXISTS agent_sessions (
			id UUID PRIMARY KEY,
			issue_id UUID REFERENCES issues(id),
			agent_type VARCHAR(50) NOT NULL,
			start_time TIMESTAMP NOT NULL,
			end_time TIMESTAMP,
			status VARCHAR(50) NOT NULL,
			output TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_issues_status ON issues(status)`,
		`CREATE INDEX IF NOT EXISTS idx_issues_scenario ON issues(scenario_name)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_sessions_issue ON agent_sessions(issue_id)`,
	}

	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			log.Printf("Failed to create table: %v", err)
		}
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	// Get counts from database
	var issueCount, sessionCount int
	db.QueryRow("SELECT COUNT(*) FROM issues").Scan(&issueCount)
	db.QueryRow("SELECT COUNT(*) FROM agent_sessions").Scan(&sessionCount)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":          "operational",
		"issues_captured": issueCount,
		"agents_spawned":  sessionCount,
		"uptime":          time.Since(startTime).String(),
	})
}

func captureHandler(w http.ResponseWriter, r *http.Request) {
	var issue Issue
	err := json.NewDecoder(r.Body).Decode(&issue)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Generate ID and set defaults
	issue.ID = uuid.New().String()
	issue.Timestamp = time.Now()
	issue.Status = "captured"

	// Store in database
	contextJSON, _ := json.Marshal(issue.ContextData)
	_, err = db.Exec(`
		INSERT INTO issues (id, timestamp, screenshot_path, scenario_name, url, description, context_data, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		issue.ID, issue.Timestamp, issue.ScreenshotPath, issue.ScenarioName,
		issue.URL, issue.Description, contextJSON, issue.Status)

	if err != nil {
		log.Printf("Failed to store issue: %v", err)
		http.Error(w, "Failed to store issue", http.StatusInternalServerError)
		return
	}

	// Create task in backlog
	createBacklogTask(issue)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"issue_id": issue.ID,
		"status":   "captured",
		"message":  "Issue captured successfully",
	})
}

func spawnAgentHandler(w http.ResponseWriter, r *http.Request) {
	var req AgentSpawnRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create agent session
	sessionID := uuid.New().String()
	session := AgentSession{
		ID:        sessionID,
		IssueID:   req.IssueID,
		AgentType: req.AgentType,
		StartTime: time.Now(),
		Status:    "running",
	}

	// Store session
	_, err = db.Exec(`
		INSERT INTO agent_sessions (id, issue_id, agent_type, start_time, status)
		VALUES ($1, $2, $3, $4, $5)`,
		session.ID, session.IssueID, session.AgentType, session.StartTime, session.Status)

	if err != nil {
		log.Printf("Failed to create agent session: %v", err)
		http.Error(w, "Failed to create agent session", http.StatusInternalServerError)
		return
	}

	// Update issue with agent session
	db.Exec("UPDATE issues SET agent_session_id = $1, status = 'assigned' WHERE id = $2",
		sessionID, req.IssueID)

	// Spawn agent asynchronously
	go spawnAgent(session, req)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"session_id": sessionID,
		"status":     "spawned",
		"message":    fmt.Sprintf("Agent %s spawned successfully", req.AgentType),
	})
}

func historyHandler(w http.ResponseWriter, r *http.Request) {
	// Get recent issues
	rows, err := db.Query(`
		SELECT id, timestamp, scenario_name, description, status, agent_session_id
		FROM issues
		ORDER BY timestamp DESC
		LIMIT 50`)

	if err != nil {
		http.Error(w, "Failed to fetch history", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var issues []Issue
	for rows.Next() {
		var issue Issue
		var agentSessionID sql.NullString
		err := rows.Scan(&issue.ID, &issue.Timestamp, &issue.ScenarioName,
			&issue.Description, &issue.Status, &agentSessionID)
		if err != nil {
			continue
		}
		if agentSessionID.Valid {
			issue.AgentSessionID = agentSessionID.String
		}
		issues = append(issues, issue)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"issues": issues,
		"count":  len(issues),
	})
}

func issueHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	issueID := vars["id"]

	var issue Issue
	var contextJSON []byte
	err := db.QueryRow(`
		SELECT id, timestamp, screenshot_path, scenario_name, url, description, 
		       context_data, agent_session_id, status, resolution_notes
		FROM issues WHERE id = $1`, issueID).Scan(
		&issue.ID, &issue.Timestamp, &issue.ScreenshotPath, &issue.ScenarioName,
		&issue.URL, &issue.Description, &contextJSON, &issue.AgentSessionID,
		&issue.Status, &issue.ResolutionNotes)

	if err != nil {
		http.Error(w, "Issue not found", http.StatusNotFound)
		return
	}

	json.Unmarshal(contextJSON, &issue.ContextData)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(issue)
}

func updateStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	issueID := vars["id"]

	var update struct {
		Status string `json:"status"`
		Notes  string `json:"notes,omitempty"`
	}

	err := json.NewDecoder(r.Body).Decode(&update)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = db.Exec("UPDATE issues SET status = $1, resolution_notes = $2 WHERE id = $3",
		update.Status, update.Notes, issueID)

	if err != nil {
		http.Error(w, "Failed to update issue", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// Helper functions

func createBacklogTask(issue Issue) {
	// Create task file in backlog
	taskContent := fmt.Sprintf(`# Issue: %s

**Scenario**: %s
**URL**: %s
**Captured**: %s

## Description
%s

## Context
- Issue ID: %s
- Screenshot: %s
- Status: %s

## Resolution
_Pending agent assignment_
`, issue.Description, issue.ScenarioName, issue.URL, issue.Timestamp.Format(time.RFC3339),
		issue.Description, issue.ID, issue.ScreenshotPath, issue.Status)

	// Store task in data directory for now
	taskDir := "../data/tasks"
	os.MkdirAll(taskDir, 0755)

	taskFile := fmt.Sprintf("%s/issue-%s.md", taskDir, issue.ID)
	if err := os.WriteFile(taskFile, []byte(taskContent), 0644); err != nil {
		log.Printf("Failed to create task file: %v", err)
	} else {
		log.Printf("Task created at %s", taskFile)
	}
}

func spawnAgent(session AgentSession, req AgentSpawnRequest) {
	// Prepare context for agent
	context := fmt.Sprintf(`Issue ID: %s
Description: %s
Screenshot: %s
Scenario: %s`, req.IssueID, req.Description, req.Screenshot, req.Context["scenario"])

	// Store context in a file for the agent
	contextFile := fmt.Sprintf("../data/contexts/%s.txt", session.ID)
	os.MkdirAll("../data/contexts", 0755)
	os.WriteFile(contextFile, []byte(context), 0644)

	// Log agent spawn attempt
	log.Printf("Spawning %s agent for issue %s", req.AgentType, req.IssueID)

	// For now, just mark as spawned - actual agent integration would happen here
	// In production, this would use vrooli resource claude-code or similar
	status := "spawned"
	output := fmt.Sprintf("Agent %s spawned for issue %s\nContext saved to: %s",
		req.AgentType, req.IssueID, contextFile)

	// Update session
	endTime := time.Now()
	db.Exec(`UPDATE agent_sessions SET end_time = $1, status = $2, output = $3 WHERE id = $4`,
		endTime, status, output, session.ID)

	// Update issue status
	db.Exec("UPDATE issues SET status = $1, agent_session_id = $2 WHERE id = $3",
		"assigned", session.ID, req.IssueID)

	log.Printf("Agent session %s created successfully", session.ID)
}

var startTime = time.Now()
