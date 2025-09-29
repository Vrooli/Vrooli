package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// Config holds application configuration
type Config struct {
	Port        string
	DatabaseURL string
	N8NURL      string
	WindmillURL string
	APIToken    string
}

// Server holds server dependencies
type Server struct {
	config *Config
	db     *sql.DB
	router *mux.Router
}

// Response is a generic API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// NewServer creates a new server instance
func NewServer() (*Server, error) {
	config := &Config{
		Port:        getEnv("API_PORT", "15001"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:password@localhost:5433/crypto_tools?sslmode=disable"),
		N8NURL:      getEnv("N8N_BASE_URL", "http://localhost:5678"),
		WindmillURL: getEnv("WINDMILL_BASE_URL", "http://localhost:5681"),
		APIToken:    getEnv("API_TOKEN", "crypto-tools-api-key-2024"),
	}

	// Connect to database
	var db *sql.DB
	var err error
	
	// Try to connect to database, but allow running without it for testing
	if config.DatabaseURL != "" && config.DatabaseURL != "mock" {
		db, err = sql.Open("postgres", config.DatabaseURL)
		if err != nil {
			log.Printf("Warning: Database connection failed: %v", err)
			log.Println("Running in mock mode without database")
			db = nil
		} else if err := db.Ping(); err != nil {
			log.Printf("Warning: Database ping failed: %v", err)
			log.Println("Running in mock mode without database")
			db = nil
		}
	}

	server := &Server{
		config: config,
		db:     db,
		router: mux.NewRouter(),
	}

	server.setupRoutes()
	return server, nil
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Middleware
	s.router.Use(loggingMiddleware)
	s.router.Use(corsMiddleware)
	s.router.Use(s.authMiddleware)

	// Health check (no auth)
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET", "OPTIONS")

	// Crypto-specific routes
	s.setupCryptoRoutes()

	// API routes
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Example resource routes - customize for your scenario
	api.HandleFunc("/resources", s.handleListResources).Methods("GET")
	api.HandleFunc("/resources", s.handleCreateResource).Methods("POST")
	api.HandleFunc("/resources/{id}", s.handleGetResource).Methods("GET")
	api.HandleFunc("/resources/{id}", s.handleUpdateResource).Methods("PUT")
	api.HandleFunc("/resources/{id}", s.handleDeleteResource).Methods("DELETE")

	// Workflow execution
	api.HandleFunc("/execute", s.handleExecuteWorkflow).Methods("POST")
	api.HandleFunc("/executions", s.handleListExecutions).Methods("GET")
	api.HandleFunc("/executions/{id}", s.handleGetExecution).Methods("GET")

	// Documentation
	s.router.HandleFunc("/docs", s.handleDocs).Methods("GET")
}

// Middleware functions
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("[%s] %s %s", r.Method, r.RequestURI, time.Since(start))
		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Configure allowed origins based on environment
		allowedOrigins := []string{
			"http://localhost:3000",
			"http://localhost:5173",
			"http://localhost:15001",
			"https://vrooli.com",
		}
		
		// Check if the request origin is in the allowed list
		origin := r.Header.Get("Origin")
		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				allowed = true
				break
			}
		}
		
		// Set CORS headers only for allowed origins
		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else if origin == "" {
			// Allow requests with no origin (e.g., same-origin requests, CLI tools)
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:15001")
		}
		
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "3600")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health check and docs
		if r.URL.Path == "/health" || r.URL.Path == "/docs" {
			next.ServeHTTP(w, r)
			return
		}

		// Check authorization header
		token := r.Header.Get("Authorization")
		if token == "" || token != "Bearer "+s.config.APIToken {
			s.sendError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Handler functions
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	overallStatus := "healthy"
	var errors []map[string]interface{}
	readiness := true
	
	// Schema-compliant health response structure
	healthResponse := map[string]interface{}{
		"status":    overallStatus,
		"service":   "crypto-tools-api",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"readiness": true,
		"version":   "1.0.0",
		"dependencies": map[string]interface{}{},
	}
	
	// Check database connectivity
	dbHealth := s.checkDatabaseHealth()
	healthResponse["dependencies"].(map[string]interface{})["database"] = dbHealth
	if dbHealth["status"] != "healthy" {
		overallStatus = "degraded"
		if dbHealth["status"] == "unhealthy" {
			readiness = false
			overallStatus = "unhealthy"
		}
		if dbHealth["error"] != nil {
			errors = append(errors, dbHealth["error"].(map[string]interface{}))
		}
	}
	
	// Check Windmill integration
	windmillHealth := s.checkWindmillHealth()
	healthResponse["dependencies"].(map[string]interface{})["windmill"] = windmillHealth
	if windmillHealth["status"] != "healthy" {
		if overallStatus != "unhealthy" {
			overallStatus = "degraded"
		}
		if windmillHealth["error"] != nil {
			errors = append(errors, windmillHealth["error"].(map[string]interface{}))
		}
	}
	
	// Check N8N integration
	n8nHealth := s.checkN8NHealth()
	healthResponse["dependencies"].(map[string]interface{})["n8n"] = n8nHealth
	if n8nHealth["status"] != "healthy" {
		if overallStatus != "unhealthy" {
			overallStatus = "degraded"
		}
		if n8nHealth["error"] != nil {
			errors = append(errors, n8nHealth["error"].(map[string]interface{}))
		}
	}
	
	// Update final status
	healthResponse["status"] = overallStatus
	healthResponse["readiness"] = readiness
	
	// Add errors if any
	if len(errors) > 0 {
		healthResponse["errors"] = errors
	}
	
	// Add metrics
	healthResponse["metrics"] = map[string]interface{}{
		"total_dependencies": 3,
		"healthy_dependencies": s.countHealthyDependencies(healthResponse["dependencies"].(map[string]interface{})),
		"response_time_ms": time.Since(start).Milliseconds(),
	}
	
	// Add crypto-tools specific stats
	cryptoStats := s.getCryptoStats()
	healthResponse["crypto_stats"] = cryptoStats
	
	// Return appropriate HTTP status
	statusCode := http.StatusOK
	if overallStatus == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}
	
	s.sendJSON(w, statusCode, healthResponse)
}

// Health check helper methods
func (s *Server) checkDatabaseHealth() map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"checks": map[string]interface{}{},
	}
	
	if s.db == nil {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code": "DATABASE_NOT_INITIALIZED",
			"message": "Database connection not initialized",
			"category": "configuration",
			"retryable": false,
		}
		return health
	}
	
	// Test database connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := s.db.PingContext(ctx); err != nil {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code": "DATABASE_CONNECTION_FAILED",
			"message": "Failed to ping database: " + err.Error(),
			"category": "resource",
			"retryable": true,
		}
		return health
	}
	health["checks"].(map[string]interface{})["ping"] = "ok"
	
	// Check crypto-tools tables
	var tableCount int
	tableQuery := `SELECT COUNT(*) FROM information_schema.tables 
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE'`
	if err := s.db.QueryRowContext(ctx, tableQuery).Scan(&tableCount); err != nil {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code": "DATABASE_SCHEMA_CHECK_FAILED",
			"message": "Failed to verify database schema: " + err.Error(),
			"category": "resource",
			"retryable": true,
		}
	} else {
		health["checks"].(map[string]interface{})["table_count"] = tableCount
		if tableCount == 0 {
			health["status"] = "degraded"
			health["error"] = map[string]interface{}{
				"code": "DATABASE_SCHEMA_EMPTY",
				"message": "No tables found in database",
				"category": "configuration",
				"retryable": false,
			}
		}
	}
	
	// Check connection pool
	stats := s.db.Stats()
	health["checks"].(map[string]interface{})["open_connections"] = stats.OpenConnections
	health["checks"].(map[string]interface{})["in_use"] = stats.InUse
	health["checks"].(map[string]interface{})["idle"] = stats.Idle
	
	if stats.OpenConnections > stats.MaxOpenConnections * 9 / 10 {
		if health["status"] == "healthy" {
			health["status"] = "degraded"
		}
		if health["error"] == nil {
			health["error"] = map[string]interface{}{
				"code": "DATABASE_CONNECTION_POOL_HIGH",
				"message": fmt.Sprintf("Connection pool usage high: %d/%d", stats.OpenConnections, stats.MaxOpenConnections),
				"category": "resource",
				"retryable": false,
			}
		}
	}
	
	return health
}

func (s *Server) checkWindmillHealth() map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"checks": map[string]interface{}{},
	}
	
	if s.config.WindmillURL == "" {
		health["status"] = "not_configured"
		health["checks"].(map[string]interface{})["automation"] = "disabled"
		return health
	}
	
	// Test Windmill connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", s.config.WindmillURL+"/api/version", nil)
	if err != nil {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code": "WINDMILL_REQUEST_FAILED",
			"message": "Failed to create request to Windmill: " + err.Error(),
			"category": "internal",
			"retryable": false,
		}
		return health
	}
	
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code": "WINDMILL_CONNECTION_FAILED",
			"message": "Failed to connect to Windmill: " + err.Error(),
			"category": "network",
			"retryable": true,
		}
		return health
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized {
		health["checks"].(map[string]interface{})["connectivity"] = "ok"
		health["checks"].(map[string]interface{})["automation"] = "enabled"
	} else {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code": "WINDMILL_UNHEALTHY",
			"message": fmt.Sprintf("Windmill returned status %d", resp.StatusCode),
			"category": "resource",
			"retryable": true,
		}
	}
	
	return health
}

func (s *Server) checkN8NHealth() map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"checks": map[string]interface{}{},
	}
	
	if s.config.N8NURL == "" {
		health["status"] = "not_configured"
		health["checks"].(map[string]interface{})["workflows"] = "disabled"
		return health
	}
	
	// Test N8N connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", s.config.N8NURL+"/healthz", nil)
	if err != nil {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code": "N8N_REQUEST_FAILED",
			"message": "Failed to create request to N8N: " + err.Error(),
			"category": "internal",
			"retryable": false,
		}
		return health
	}
	
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code": "N8N_CONNECTION_FAILED",
			"message": "Failed to connect to N8N: " + err.Error(),
			"category": "network",
			"retryable": true,
		}
		return health
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusOK {
		health["checks"].(map[string]interface{})["connectivity"] = "ok"
		health["checks"].(map[string]interface{})["workflows"] = "enabled"
	} else {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code": "N8N_UNHEALTHY",
			"message": fmt.Sprintf("N8N returned status %d", resp.StatusCode),
			"category": "resource",
			"retryable": true,
		}
	}
	
	return health
}

func (s *Server) countHealthyDependencies(deps map[string]interface{}) int {
	count := 0
	for _, dep := range deps {
		if depMap, ok := dep.(map[string]interface{}); ok {
			if status, exists := depMap["status"]; exists && (status == "healthy" || status == "not_configured") {
				count++
			}
		}
	}
	return count
}

func (s *Server) getCryptoStats() map[string]interface{} {
	stats := map[string]interface{}{
		"total_operations": 0,
		"hash_operations": 0,
		"encryption_operations": 0,
		"signature_operations": 0,
		"key_operations": 0,
	}
	
	if s.db == nil {
		return stats
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	
	// These are example queries - adjust based on actual crypto-tools schema
	var totalOps, hashOps, encryptOps, signOps, keyOps int
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM crypto_operations").Scan(&totalOps)
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM crypto_operations WHERE operation_type = 'hash'").Scan(&hashOps)
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM crypto_operations WHERE operation_type = 'encrypt'").Scan(&encryptOps)
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM crypto_operations WHERE operation_type = 'sign'").Scan(&signOps)
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM crypto_keys").Scan(&keyOps)
	
	stats["total_operations"] = totalOps
	stats["hash_operations"] = hashOps
	stats["encryption_operations"] = encryptOps
	stats["signature_operations"] = signOps
	stats["key_operations"] = keyOps
	
	return stats
}

func (s *Server) handleListResources(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement based on your scenario needs
	// Example: List resources from database

	query := `SELECT id, name, description, created_at FROM resources ORDER BY created_at DESC LIMIT 100`
	rows, err := s.db.Query(query)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to query resources")
		return
	}
	defer rows.Close()

	var resources []map[string]interface{}
	for rows.Next() {
		var id, name, description string
		var createdAt time.Time

		if err := rows.Scan(&id, &name, &description, &createdAt); err != nil {
			continue
		}

		resources = append(resources, map[string]interface{}{
			"id":          id,
			"name":        name,
			"description": description,
			"created_at":  createdAt,
		})
	}

	s.sendJSON(w, http.StatusOK, resources)
}

func (s *Server) handleCreateResource(w http.ResponseWriter, r *http.Request) {
	var input map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Generate ID
	id := uuid.New().String()

	// TODO: Validate input and insert into database
	// This is a template - customize for your needs

	query := `INSERT INTO resources (id, name, description, config, created_at) 
	          VALUES ($1, $2, $3, $4, $5)`

	_, err := s.db.Exec(query,
		id,
		input["name"],
		input["description"],
		input["config"],
		time.Now(),
	)

	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to create resource")
		return
	}

	s.sendJSON(w, http.StatusCreated, map[string]interface{}{
		"id":         id,
		"created_at": time.Now(),
	})
}

func (s *Server) handleGetResource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// TODO: Query resource from database
	query := `SELECT id, name, description, config, created_at FROM resources WHERE id = $1`

	var resource map[string]interface{}
	row := s.db.QueryRow(query, id)

	var name, description string
	var config json.RawMessage
	var createdAt time.Time

	err := row.Scan(&id, &name, &description, &config, &createdAt)
	if err == sql.ErrNoRows {
		s.sendError(w, http.StatusNotFound, "resource not found")
		return
	}
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to query resource")
		return
	}

	resource = map[string]interface{}{
		"id":          id,
		"name":        name,
		"description": description,
		"config":      config,
		"created_at":  createdAt,
	}

	s.sendJSON(w, http.StatusOK, resource)
}

func (s *Server) handleUpdateResource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var input map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// TODO: Update resource in database
	query := `UPDATE resources SET name = $2, description = $3, config = $4, updated_at = $5 
	          WHERE id = $1`

	result, err := s.db.Exec(query,
		id,
		input["name"],
		input["description"],
		input["config"],
		time.Now(),
	)

	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to update resource")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		s.sendError(w, http.StatusNotFound, "resource not found")
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"id":         id,
		"updated_at": time.Now(),
	})
}

func (s *Server) handleDeleteResource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	query := `DELETE FROM resources WHERE id = $1`
	result, err := s.db.Exec(query, id)

	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to delete resource")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		s.sendError(w, http.StatusNotFound, "resource not found")
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"deleted": true,
		"id":      id,
	})
}

func (s *Server) handleExecuteWorkflow(w http.ResponseWriter, r *http.Request) {
	var input map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// TODO: Trigger workflow execution via n8n or Windmill
	// This is a template - customize based on your workflow platform

	executionID := uuid.New().String()

	// Example: Call n8n webhook
	// webhookURL := fmt.Sprintf("%s/webhook/%s", s.config.N8NURL, input["workflow_id"])
	// resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))

	s.sendJSON(w, http.StatusAccepted, map[string]interface{}{
		"execution_id": executionID,
		"status":       "pending",
		"started_at":   time.Now(),
	})
}

func (s *Server) handleListExecutions(w http.ResponseWriter, r *http.Request) {
	// TODO: List workflow executions from database
	query := `SELECT id, workflow_id, status, started_at, completed_at 
	          FROM executions 
	          ORDER BY started_at DESC 
	          LIMIT 100`

	rows, err := s.db.Query(query)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to query executions")
		return
	}
	defer rows.Close()

	var executions []map[string]interface{}
	for rows.Next() {
		var id, workflowID, status string
		var startedAt time.Time
		var completedAt sql.NullTime

		if err := rows.Scan(&id, &workflowID, &status, &startedAt, &completedAt); err != nil {
			continue
		}

		execution := map[string]interface{}{
			"id":          id,
			"workflow_id": workflowID,
			"status":      status,
			"started_at":  startedAt,
		}

		if completedAt.Valid {
			execution["completed_at"] = completedAt.Time
		}

		executions = append(executions, execution)
	}

	s.sendJSON(w, http.StatusOK, executions)
}

func (s *Server) handleGetExecution(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// TODO: Get execution details from database
	query := `SELECT id, workflow_id, status, input_data, output_data, error_message, 
	                 started_at, completed_at 
	          FROM executions 
	          WHERE id = $1`

	row := s.db.QueryRow(query, id)

	var workflowID, status string
	var inputData, outputData json.RawMessage
	var errorMessage sql.NullString
	var startedAt time.Time
	var completedAt sql.NullTime

	err := row.Scan(&id, &workflowID, &status, &inputData, &outputData,
		&errorMessage, &startedAt, &completedAt)

	if err == sql.ErrNoRows {
		s.sendError(w, http.StatusNotFound, "execution not found")
		return
	}
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to query execution")
		return
	}

	execution := map[string]interface{}{
		"id":          id,
		"workflow_id": workflowID,
		"status":      status,
		"input_data":  inputData,
		"output_data": outputData,
		"started_at":  startedAt,
	}

	if errorMessage.Valid {
		execution["error_message"] = errorMessage.String
	}
	if completedAt.Valid {
		execution["completed_at"] = completedAt.Time
	}

	s.sendJSON(w, http.StatusOK, execution)
}

func (s *Server) handleDocs(w http.ResponseWriter, r *http.Request) {
	docs := map[string]interface{}{
		"name":        "Crypto-Tools API",
		"version":     "1.0.0",
		"description": "SCENARIO_DESCRIPTION_PLACEHOLDER",
		"endpoints": []map[string]string{
			{"method": "GET", "path": "/health", "description": "Health check"},
			{"method": "GET", "path": "/api/v1/resources", "description": "List resources"},
			{"method": "POST", "path": "/api/v1/resources", "description": "Create resource"},
			{"method": "GET", "path": "/api/v1/resources/{id}", "description": "Get resource"},
			{"method": "PUT", "path": "/api/v1/resources/{id}", "description": "Update resource"},
			{"method": "DELETE", "path": "/api/v1/resources/{id}", "description": "Delete resource"},
			{"method": "POST", "path": "/api/v1/execute", "description": "Execute workflow"},
			{"method": "GET", "path": "/api/v1/executions", "description": "List executions"},
			{"method": "GET", "path": "/api/v1/executions/{id}", "description": "Get execution"},
		},
	}

	s.sendJSON(w, http.StatusOK, docs)
}

// Helper functions
func (s *Server) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{
		Success: status < 400,
		Data:    data,
	})
}

func (s *Server) sendError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{
		Success: false,
		Error:   message,
	})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Run starts the server
func (s *Server) Run() error {
	srv := &http.Server{
		Addr:         ":" + s.config.Port,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}

		s.db.Close()
	}()

	log.Printf("Server starting on port %s", s.config.Port)
	log.Printf("API documentation available at http://localhost:%s/docs", s.config.Port)

	return srv.ListenAndServe()
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start crypto-tools

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	log.Println("Starting Crypto-Tools API...")

	server, err := NewServer()
	if err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}

	if err := server.Run(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}
