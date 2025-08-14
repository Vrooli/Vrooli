package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type Config struct {
	Port         string
	PostgresURL  string
	QdrantURL    string
	RedisURL     string
	MinioURL     string
	OllamaURL    string
	N8NBaseURL   string
	WindmillURL  string
}

type Server struct {
	config *Config
	db     *sql.DB
}

type Issue struct {
	ID               string    `json:"id"`
	AppID            string    `json:"app_id"`
	ExternalID       *string   `json:"external_id,omitempty"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	Status           string    `json:"status"`
	Priority         string    `json:"priority"`
	Type             string    `json:"type"`
	ReporterName     *string   `json:"reporter_name,omitempty"`
	ReporterEmail    *string   `json:"reporter_email,omitempty"`
	AssignedAgentID  *string   `json:"assigned_agent_id,omitempty"`
	InvestigationReport *string `json:"investigation_report,omitempty"`
	RootCause        *string   `json:"root_cause,omitempty"`
	SuggestedFix     *string   `json:"suggested_fix,omitempty"`
	ConfidenceScore  *float64  `json:"confidence_score,omitempty"`
	ErrorMessage     *string   `json:"error_message,omitempty"`
	StackTrace       *string   `json:"stack_trace,omitempty"`
	Tags             []string  `json:"tags"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type Agent struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	DisplayName      string    `json:"display_name"`
	Description      *string   `json:"description,omitempty"`
	SystemPrompt     string    `json:"system_prompt"`
	UserPromptTemplate string  `json:"user_prompt_template"`
	Capabilities     []string  `json:"capabilities"`
	MaxTokens        int       `json:"max_tokens"`
	Temperature      float64   `json:"temperature"`
	Model            string    `json:"model"`
	SuccessRate      float64   `json:"success_rate"`
	TotalRuns        int       `json:"total_runs"`
	SuccessfulRuns   int       `json:"successful_runs"`
	IsActive         bool      `json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
}

type App struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	DisplayName   string    `json:"display_name"`
	ScenarioName  *string   `json:"scenario_name,omitempty"`
	Type          string    `json:"type"`
	APIToken      string    `json:"api_token"`
	Status        string    `json:"status"`
	TotalIssues   int       `json:"total_issues"`
	OpenIssues    int       `json:"open_issues"`
	CreatedAt     time.Time `json:"created_at"`
}

type ApiResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func loadConfig() *Config {
	return &Config{
		Port:         getEnv("PORT", "8090"),
		PostgresURL:  getEnv("POSTGRES_URL", "postgres://postgres:postgres@localhost:5432/issue_tracker?sslmode=disable"),
		QdrantURL:    getEnv("QDRANT_URL", "http://localhost:6333"),
		RedisURL:     getEnv("REDIS_URL", "redis://localhost:6379"),
		MinioURL:     getEnv("MINIO_URL", "http://localhost:9000"),
		OllamaURL:    getEnv("OLLAMA_URL", "http://localhost:11434"),
		N8NBaseURL:   getEnv("N8N_BASE_URL", "http://localhost:5678"),
		WindmillURL:  getEnv("WINDMILL_BASE_URL", "http://localhost:8000"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	response := ApiResponse{
		Success: true,
		Message: "App Issue Tracker API is healthy",
		Data: map[string]interface{}{
			"timestamp": time.Now().UTC(),
			"version":   "1.0.0",
			"services": map[string]string{
				"database": s.config.PostgresURL,
				"qdrant":   s.config.QdrantURL,
				"n8n":      s.config.N8NBaseURL,
				"windmill": s.config.WindmillURL,
			},
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) getIssuesHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	status := r.URL.Query().Get("status")
	priority := r.URL.Query().Get("priority")
	issueType := r.URL.Query().Get("type")
	limitStr := r.URL.Query().Get("limit")
	
	limit := 20
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil {
			limit = parsed
		}
	}
	
	// Build dynamic query
	query := `SELECT id, app_id, external_id, title, description, status, priority, type, 
			 reporter_name, reporter_email, assigned_agent_id, investigation_report, 
			 root_cause, suggested_fix, confidence_score, error_message, stack_trace, 
			 tags, created_at, updated_at FROM issues WHERE 1=1`
	
	args := []interface{}{}
	argCount := 1
	
	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
		argCount++
	}
	
	if priority != "" {
		query += fmt.Sprintf(" AND priority = $%d", argCount)
		args = append(args, priority)
		argCount++
	}
	
	if issueType != "" {
		query += fmt.Sprintf(" AND type = $%d", argCount)
		args = append(args, issueType)
		argCount++
	}
	
	query += " ORDER BY created_at DESC"
	query += fmt.Sprintf(" LIMIT $%d", argCount)
	args = append(args, limit)
	
	rows, err := s.db.Query(query, args...)
	if err != nil {
		http.Error(w, "Database query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var issues []Issue
	for rows.Next() {
		var issue Issue
		var tagsJSON string
		
		err := rows.Scan(
			&issue.ID, &issue.AppID, &issue.ExternalID, &issue.Title, &issue.Description,
			&issue.Status, &issue.Priority, &issue.Type, &issue.ReporterName, &issue.ReporterEmail,
			&issue.AssignedAgentID, &issue.InvestigationReport, &issue.RootCause, &issue.SuggestedFix,
			&issue.ConfidenceScore, &issue.ErrorMessage, &issue.StackTrace, &tagsJSON,
			&issue.CreatedAt, &issue.UpdatedAt,
		)
		if err != nil {
			continue
		}
		
		// Parse tags from JSON array
		if tagsJSON != "" {
			json.Unmarshal([]byte(tagsJSON), &issue.Tags)
		}
		
		issues = append(issues, issue)
	}
	
	response := ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"issues": issues,
			"count":  len(issues),
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) createIssueHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title        string   `json:"title"`
		Description  string   `json:"description"`
		Type         string   `json:"type"`
		Priority     string   `json:"priority"`
		ErrorMessage string   `json:"error_message"`
		StackTrace   string   `json:"stack_trace"`
		Tags         []string `json:"tags"`
		AppToken     string   `json:"app_token"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// Validate required fields
	if req.Title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}
	
	// Default values
	if req.Type == "" {
		req.Type = "bug"
	}
	if req.Priority == "" {
		req.Priority = "medium"
	}
	if req.Description == "" {
		req.Description = req.Title
	}
	
	// Get app ID from token (simplified - in production would validate token)
	appID := "00000000-0000-0000-0000-000000000001" // Default app ID
	
	// Insert issue
	tagsJSON, _ := json.Marshal(req.Tags)
	var issueID string
	
	err := s.db.QueryRow(`
		INSERT INTO issues (app_id, title, description, type, priority, error_message, stack_trace, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`, appID, req.Title, req.Description, req.Type, req.Priority, req.ErrorMessage, req.StackTrace, string(tagsJSON)).Scan(&issueID)
	
	if err != nil {
		log.Printf("Error creating issue: %v", err)
		http.Error(w, "Failed to create issue", http.StatusInternalServerError)
		return
	}
	
	response := ApiResponse{
		Success: true,
		Message: "Issue created successfully",
		Data: map[string]interface{}{
			"issue_id": issueID,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) getAgentsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(`
		SELECT id, name, display_name, description, system_prompt, user_prompt_template,
			   capabilities, max_tokens, temperature, model, success_rate, total_runs,
			   successful_runs, is_active, created_at
		FROM agents WHERE is_active = true
		ORDER BY display_name
	`)
	if err != nil {
		http.Error(w, "Database query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var agents []Agent
	for rows.Next() {
		var agent Agent
		var capabilitiesStr string
		
		err := rows.Scan(
			&agent.ID, &agent.Name, &agent.DisplayName, &agent.Description,
			&agent.SystemPrompt, &agent.UserPromptTemplate, &capabilitiesStr,
			&agent.MaxTokens, &agent.Temperature, &agent.Model, &agent.SuccessRate,
			&agent.TotalRuns, &agent.SuccessfulRuns, &agent.IsActive, &agent.CreatedAt,
		)
		if err != nil {
			continue
		}
		
		// Parse capabilities
		if capabilitiesStr != "" {
			capabilitiesStr = strings.Trim(capabilitiesStr, "{}")
			if capabilitiesStr != "" {
				agent.Capabilities = strings.Split(capabilitiesStr, ",")
			}
		}
		
		agents = append(agents, agent)
	}
	
	response := ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"agents": agents,
			"count":  len(agents),
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) getAppsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(`
		SELECT id, name, display_name, scenario_name, type, api_token, status,
			   total_issues, open_issues, created_at
		FROM apps
		ORDER BY display_name
	`)
	if err != nil {
		http.Error(w, "Database query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var apps []App
	for rows.Next() {
		var app App
		
		err := rows.Scan(
			&app.ID, &app.Name, &app.DisplayName, &app.ScenarioName, &app.Type,
			&app.APIToken, &app.Status, &app.TotalIssues, &app.OpenIssues, &app.CreatedAt,
		)
		if err != nil {
			continue
		}
		
		apps = append(apps, app)
	}
	
	response := ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"apps":  apps,
			"count": len(apps),
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) triggerInvestigationHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IssueID  string `json:"issue_id"`
		AgentID  string `json:"agent_id"`
		Priority string `json:"priority"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	if req.IssueID == "" {
		http.Error(w, "Issue ID is required", http.StatusBadRequest)
		return
	}
	
	// Default priority if not specified
	if req.Priority == "" {
		req.Priority = "normal"
	}
	
	// In a real implementation, this would trigger the n8n workflow
	// For now, we'll just return a mock response
	response := ApiResponse{
		Success: true,
		Message: "Investigation triggered successfully",
		Data: map[string]interface{}{
			"run_id":           fmt.Sprintf("run_%d", time.Now().Unix()),
			"investigation_id": fmt.Sprintf("inv_%d", time.Now().Unix()),
			"issue_id":         req.IssueID,
			"agent_id":         req.AgentID,
			"priority":         req.Priority,
			"status":          "queued",
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) searchIssuesHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}
	
	// Simple text search implementation
	// In production, this would use vector similarity search with Qdrant
	rows, err := s.db.Query(`
		SELECT id, app_id, title, description, status, priority, type, created_at
		FROM issues 
		WHERE title ILIKE $1 OR description ILIKE $1
		ORDER BY created_at DESC
		LIMIT 10
	`, "%"+query+"%")
	
	if err != nil {
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var results []map[string]interface{}
	for rows.Next() {
		var issue struct {
			ID          string
			AppID       string
			Title       string
			Description string
			Status      string
			Priority    string
			Type        string
			CreatedAt   time.Time
		}
		
		err := rows.Scan(&issue.ID, &issue.AppID, &issue.Title, &issue.Description,
			&issue.Status, &issue.Priority, &issue.Type, &issue.CreatedAt)
		if err != nil {
			continue
		}
		
		result := map[string]interface{}{
			"id":          issue.ID,
			"app_id":      issue.AppID,
			"title":       issue.Title,
			"description": issue.Description,
			"status":      issue.Status,
			"priority":    issue.Priority,
			"type":        issue.Type,
			"created_at":  issue.CreatedAt,
			"similarity":  0.8, // Mock similarity score
		}
		
		results = append(results, result)
	}
	
	response := ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"results": results,
			"count":   len(results),
			"query":   query,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) getStatsHandler(w http.ResponseWriter, r *http.Request) {
	// Get basic statistics
	var totalIssues, openIssues, inProgress, fixedToday int
	
	s.db.QueryRow("SELECT COUNT(*) FROM issues").Scan(&totalIssues)
	s.db.QueryRow("SELECT COUNT(*) FROM issues WHERE status IN ('open', 'investigating')").Scan(&openIssues)
	s.db.QueryRow("SELECT COUNT(*) FROM issues WHERE status = 'in_progress'").Scan(&inProgress)
	s.db.QueryRow("SELECT COUNT(*) FROM issues WHERE status = 'fixed' AND DATE(resolved_at) = CURRENT_DATE").Scan(&fixedToday)
	
	response := ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"stats": map[string]interface{}{
				"total_issues":           totalIssues,
				"open_issues":            openIssues,
				"in_progress":            inProgress,
				"fixed_today":            fixedToday,
				"avg_resolution_hours":   24.5, // Mock value
				"top_apps": []map[string]interface{}{
					{"app_name": "vrooli-core", "issue_count": 15},
					{"app_name": "retro-game-launcher", "issue_count": 8},
					{"app_name": "agent-metareasoning", "issue_count": 3},
				},
			},
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Token")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func main() {
	config := loadConfig()
	
	// Connect to database
	db, err := sql.Open("postgres", config.PostgresURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()
	
	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatal("Database ping failed:", err)
	}
	
	log.Printf("Connected to database successfully")
	
	server := &Server{
		config: config,
		db:     db,
	}
	
	// Setup routes
	r := mux.NewRouter()
	
	// Health check
	r.HandleFunc("/health", server.healthHandler).Methods("GET")
	
	// API routes
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/issues", server.getIssuesHandler).Methods("GET")
	api.HandleFunc("/issues", server.createIssueHandler).Methods("POST")
	api.HandleFunc("/issues/search", server.searchIssuesHandler).Methods("GET")
	api.HandleFunc("/agents", server.getAgentsHandler).Methods("GET")
	api.HandleFunc("/apps", server.getAppsHandler).Methods("GET")
	api.HandleFunc("/investigate", server.triggerInvestigationHandler).Methods("POST")
	api.HandleFunc("/stats", server.getStatsHandler).Methods("GET")
	
	// Apply CORS middleware
	handler := corsMiddleware(r)
	
	log.Printf("Starting App Issue Tracker API server on port %s", config.Port)
	log.Printf("Health check: http://localhost:%s/health", config.Port)
	log.Printf("API base URL: http://localhost:%s/api", config.Port)
	
	if err := http.ListenAndServe(":"+config.Port, handler); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}