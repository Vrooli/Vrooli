package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

// Issue represents a captured issue
type Issue struct {
	ID             string                 `json:"id"`
	Timestamp      time.Time              `json:"timestamp"`
	ScreenshotPath string                 `json:"screenshot_path"`
	ScenarioName   string                 `json:"scenario_name"`
	URL            string                 `json:"url"`
	Description    string                 `json:"description"`
	ContextData    map[string]interface{} `json:"context_data"`
	AgentSessionID string                 `json:"agent_session_id,omitempty"`
	Status         string                 `json:"status"`
	ResolutionNotes string                `json:"resolution_notes,omitempty"`
	Tags           []string               `json:"tags,omitempty"`
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
	ID        string    `json:"id"`
	IssueID   string    `json:"issue_id"`
	AgentType string    `json:"agent_type"`
	StartTime time.Time `json:"start_time"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	Status    string    `json:"status"`
	Output    string    `json:"output,omitempty"`
}

var db *sql.DB

func main() {
	// Initialize database connection
	initDB()
	defer db.Close()

	// Create tables if they don't exist
	createTables()

	// Set up routes
	router := mux.NewRouter()
	
	// Health check
	router.HandleFunc("/health", healthHandler).Methods("GET")
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

	// Enable CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:*", "file://*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})
	
	handler := c.Handler(router)
	
	// Get port from environment or use default
	port := getEnv("API_PORT", getEnv("PORT", ""))
	
	log.Printf("Vrooli Assistant API starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func initDB() {
	// Get database connection from environment
	dbHost := os.Getenv("POSTGRES_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	
	dbPort := os.Getenv("POSTGRES_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	
	dbUser := os.Getenv("POSTGRES_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}
	
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	if dbPassword == "" {
		dbPassword = "postgres"
	}
	
	dbName := os.Getenv("POSTGRES_DB")
	if dbName == "" {
		dbName = "vrooli_assistant"
	}
	
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)
	
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	
	// Test connection
	err = db.Ping()
	if err != nil {
		log.Fatal("Failed to ping database:", err)
	}
	
	log.Println("Database connected successfully")
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
		"status": "operational",
		"issues_captured": issueCount,
		"agents_spawned": sessionCount,
		"uptime": time.Since(startTime).String(),
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
		"status": "captured",
		"message": "Issue captured successfully",
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
		"status": "spawned",
		"message": fmt.Sprintf("Agent %s spawned successfully", req.AgentType),
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
		"count": len(issues),
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
	
	// Write to backlog directory (simplified - would integrate with task system)
	backlogPath := "/home/matthalloran8/Vrooli/docs/tasks/backlog.md"
	if _, err := os.Stat(backlogPath); err == nil {
		// Append to existing backlog
		f, _ := os.OpenFile(backlogPath, os.O_APPEND|os.O_WRONLY, 0644)
		defer f.Close()
		f.WriteString("\n\n" + taskContent)
	}
}

func spawnAgent(session AgentSession, req AgentSpawnRequest) {
	// Prepare context for agent
	context := fmt.Sprintf(`Issue ID: %s
Description: %s
Screenshot: %s
Scenario: %s`, req.IssueID, req.Description, req.Screenshot, req.Context["scenario"])
	
	// Spawn agent based on type
	var cmd *exec.Cmd
	switch req.AgentType {
	case "claude-code":
		// Spawn claude-code with context
		cmd = exec.Command("claude-code", "--context", context)
	case "agent-s2":
		// Spawn agent-s2
		cmd = exec.Command("agent-s2", "fix", "--issue", req.IssueID)
	default:
		log.Printf("Unknown agent type: %s", req.AgentType)
		return
	}
	
	// Run agent (simplified - would capture output properly)
	output, err := cmd.CombinedOutput()
	
	// Update session
	endTime := time.Now()
	status := "completed"
	if err != nil {
		status = "failed"
		log.Printf("Agent failed: %v", err)
	}
	
	db.Exec(`UPDATE agent_sessions SET end_time = $1, status = $2, output = $3 WHERE id = $4`,
		endTime, status, string(output), session.ID)
	
	// Update issue status
	issueStatus := "resolved"
	if status == "failed" {
		issueStatus = "failed"
	}
	db.Exec("UPDATE issues SET status = $1 WHERE id = $2", issueStatus, session.IssueID)
}

var startTime = time.Now()
