package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type Config struct {
	Port         string
	PostgresURL  string
	RedisURL     string
	MinioURL     string
	QdrantURL    string
	N8nURL       string
	WindmillURL  string
}

type ScrapingAgent struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Platform     string                 `json:"platform"`
	AgentType    string                 `json:"agent_type"`
	Configuration map[string]interface{} `json:"configuration"`
	ScheduleCron *string                `json:"schedule_cron"`
	Enabled      bool                   `json:"enabled"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	LastRun      *time.Time             `json:"last_run"`
	NextRun      *time.Time             `json:"next_run"`
	Tags         []string               `json:"tags"`
}

type ScrapingTarget struct {
	ID             string                 `json:"id"`
	AgentID        string                 `json:"agent_id"`
	URL            string                 `json:"url"`
	SelectorConfig map[string]interface{} `json:"selector_config"`
	Authentication map[string]interface{} `json:"authentication"`
	Headers        map[string]interface{} `json:"headers"`
	ProxyConfig    map[string]interface{} `json:"proxy_config"`
	RateLimitMs    int                    `json:"rate_limit_ms"`
	MaxRetries     int                    `json:"max_retries"`
	CreatedAt      time.Time              `json:"created_at"`
}

type ScrapingResult struct {
	ID              string                 `json:"id"`
	AgentID         string                 `json:"agent_id"`
	TargetID        string                 `json:"target_id"`
	RunID           string                 `json:"run_id"`
	Status          string                 `json:"status"`
	Data            map[string]interface{} `json:"data"`
	Screenshots     []string               `json:"screenshots"`
	ExtractedCount  *int                   `json:"extracted_count"`
	ErrorMessage    *string                `json:"error_message"`
	StartedAt       time.Time              `json:"started_at"`
	CompletedAt     *time.Time             `json:"completed_at"`
	ExecutionTimeMs *int                   `json:"execution_time_ms"`
}

type PlatformCapabilities struct {
	Platform     string                 `json:"platform"`
	AgentTypes   []string               `json:"agent_types,omitempty"`
	Capabilities []string               `json:"capabilities,omitempty"`
	Features     map[string]interface{} `json:"features"`
	BestFor      string                 `json:"best_for"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

var db *sql.DB
var config Config

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start web-scraper-manager

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	config = loadConfig()
	
	// Initialize database connection
	var err error
	db, err = sql.Open("postgres", config.PostgresURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second
	
	log.Println("🔄 Attempting database connection with exponential backoff...")
	log.Printf("🕸️ Database URL configured")
	
	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = db.Ping()
		if pingErr == nil {
			log.Printf("✅ Database connected successfully on attempt %d", attempt + 1)
			break
		}
		
		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay) * math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))
		
		// Add progressive jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * (float64(attempt) / float64(maxRetries)))
		actualDelay := delay + jitter
		
		log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt + 1, maxRetries, pingErr)
		log.Printf("⏳ Waiting %v before next attempt", actualDelay)
		
		// Provide detailed status every few attempts
		if attempt > 0 && attempt % 3 == 0 {
			log.Printf("📈 Retry progress:")
			log.Printf("   - Attempts made: %d/%d", attempt + 1, maxRetries)
			log.Printf("   - Total wait time: ~%v", time.Duration(attempt * 2) * baseDelay)
			log.Printf("   - Current delay: %v (with jitter: %v)", delay, jitter)
		}
		
		time.Sleep(actualDelay)
	}
	
	if pingErr != nil {
		log.Fatalf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}
	
	log.Println("🎉 Database connection pool established successfully!")

	// Setup routes
	router := mux.NewRouter()
	
	// Health endpoint
	router.HandleFunc("/health", healthHandler).Methods("GET")
	
	// API routes
	api := router.PathPrefix("/api").Subrouter()
	
	// Agents endpoints
	api.HandleFunc("/agents", getAgentsHandler).Methods("GET")
	api.HandleFunc("/agents", createAgentHandler).Methods("POST")
	api.HandleFunc("/agents/{id}", getAgentHandler).Methods("GET")
	api.HandleFunc("/agents/{id}", updateAgentHandler).Methods("PUT")
	api.HandleFunc("/agents/{id}", deleteAgentHandler).Methods("DELETE")
	
	// Targets endpoints
	api.HandleFunc("/targets", getTargetsHandler).Methods("GET")
	api.HandleFunc("/targets", createTargetHandler).Methods("POST")
	api.HandleFunc("/agents/{agentId}/targets", getAgentTargetsHandler).Methods("GET")
	
	// Results endpoints
	api.HandleFunc("/results", getResultsHandler).Methods("GET")
	api.HandleFunc("/agents/{agentId}/results", getAgentResultsHandler).Methods("GET")
	
	// Platform capabilities
	api.HandleFunc("/platforms", getPlatformsHandler).Methods("GET")
	
	// Execution endpoints
	api.HandleFunc("/agents/{id}/execute", executeAgentHandler).Methods("POST")
	api.HandleFunc("/workflows/{id}/execute", executeWorkflowHandler).Methods("POST")
	
	// Export endpoints
	api.HandleFunc("/export", exportDataHandler).Methods("POST")
	
	// Monitoring endpoints
	api.HandleFunc("/metrics", getMetricsHandler).Methods("GET")
	api.HandleFunc("/status", getStatusHandler).Methods("GET")

	// Enable CORS
	router.Use(corsMiddleware)

	log.Printf("Web Scraper Manager API starting on port %s", config.Port)
	log.Fatal(http.ListenAndServe(":"+config.Port, router))
}

func loadConfig() Config {
	// Database configuration - support both POSTGRES_URL and individual components
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		// Try to build from individual components - REQUIRED, no defaults
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")
		
		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			log.Fatal("❌ Database configuration missing. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}
		
		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}
	
	// Port configuration - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		log.Fatal("❌ API_PORT or PORT environment variable is required")
	}
	
	return Config{
		Port:        port,
		PostgresURL: postgresURL,
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379"),
		MinioURL:    getEnv("MINIO_URL", "http://localhost:9000"),
		QdrantURL:   getEnv("QDRANT_URL", "http://localhost:6333"),
		N8nURL:      getEnv("N8N_BASE_URL", "http://localhost:5678"),
		WindmillURL: getEnv("WINDMILL_BASE_URL", "http://localhost:8000"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func respondJSON(w http.ResponseWriter, status int, response APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	// Check database connection
	if err := db.Ping(); err != nil {
		respondJSON(w, http.StatusServiceUnavailable, APIResponse{
			Success: false,
			Error:   "Database connection failed",
		})
		return
	}
	
	respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Web Scraper Manager API is healthy",
		Data: map[string]interface{}{
			"timestamp": time.Now().UTC(),
			"version":   "1.0.0",
			"database":  "connected",
		},
	})
}

func getAgentsHandler(w http.ResponseWriter, r *http.Request) {
	platform := r.URL.Query().Get("platform")
	enabled := r.URL.Query().Get("enabled")
	
	query := `
		SELECT id, name, platform, agent_type, configuration, schedule_cron, 
		       enabled, created_at, updated_at, last_run, next_run, tags
		FROM scraping_agents 
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1
	
	if platform != "" {
		query += fmt.Sprintf(" AND platform = $%d", argIndex)
		args = append(args, platform)
		argIndex++
	}
	
	if enabled != "" {
		if enabledBool, err := strconv.ParseBool(enabled); err == nil {
			query += fmt.Sprintf(" AND enabled = $%d", argIndex)
			args = append(args, enabledBool)
			argIndex++
		}
	}
	
	query += " ORDER BY created_at DESC"
	
	rows, err := db.Query(query, args...)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to query agents: " + err.Error(),
		})
		return
	}
	defer rows.Close()
	
	var agents []ScrapingAgent
	for rows.Next() {
		var agent ScrapingAgent
		var configJSON []byte
		var tagsJSON []byte
		
		err := rows.Scan(
			&agent.ID, &agent.Name, &agent.Platform, &agent.AgentType,
			&configJSON, &agent.ScheduleCron, &agent.Enabled,
			&agent.CreatedAt, &agent.UpdatedAt, &agent.LastRun,
			&agent.NextRun, &tagsJSON,
		)
		if err != nil {
			log.Printf("Error scanning agent row: %v", err)
			continue
		}
		
		// Parse JSON fields
		if len(configJSON) > 0 {
			json.Unmarshal(configJSON, &agent.Configuration)
		}
		if len(tagsJSON) > 0 {
			json.Unmarshal(tagsJSON, &agent.Tags)
		}
		
		agents = append(agents, agent)
	}
	
	if agents == nil {
		agents = []ScrapingAgent{}
	}
	
	respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    agents,
	})
}

func createAgentHandler(w http.ResponseWriter, r *http.Request) {
	var agent ScrapingAgent
	if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
		respondJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid JSON: " + err.Error(),
		})
		return
	}
	
	// Validate required fields
	if agent.Name == "" || agent.Platform == "" {
		respondJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Name and platform are required",
		})
		return
	}
	
	configJSON, _ := json.Marshal(agent.Configuration)
	tagsJSON, _ := json.Marshal(agent.Tags)
	
	query := `
		INSERT INTO scraping_agents (name, platform, agent_type, configuration, 
		                           schedule_cron, enabled, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`
	
	err := db.QueryRow(query, agent.Name, agent.Platform, agent.AgentType,
		configJSON, agent.ScheduleCron, agent.Enabled, tagsJSON).Scan(
		&agent.ID, &agent.CreatedAt, &agent.UpdatedAt)
	
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to create agent: " + err.Error(),
		})
		return
	}
	
	respondJSON(w, http.StatusCreated, APIResponse{
		Success: true,
		Data:    agent,
		Message: "Agent created successfully",
	})
}

func getAgentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]
	
	var agent ScrapingAgent
	var configJSON []byte
	var tagsJSON []byte
	
	query := `
		SELECT id, name, platform, agent_type, configuration, schedule_cron,
		       enabled, created_at, updated_at, last_run, next_run, tags
		FROM scraping_agents WHERE id = $1
	`
	
	err := db.QueryRow(query, agentID).Scan(
		&agent.ID, &agent.Name, &agent.Platform, &agent.AgentType,
		&configJSON, &agent.ScheduleCron, &agent.Enabled,
		&agent.CreatedAt, &agent.UpdatedAt, &agent.LastRun,
		&agent.NextRun, &tagsJSON,
	)
	
	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusNotFound, APIResponse{
			Success: false,
			Error:   "Agent not found",
		})
		return
	} else if err != nil {
		respondJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Database error: " + err.Error(),
		})
		return
	}
	
	// Parse JSON fields
	if len(configJSON) > 0 {
		json.Unmarshal(configJSON, &agent.Configuration)
	}
	if len(tagsJSON) > 0 {
		json.Unmarshal(tagsJSON, &agent.Tags)
	}
	
	respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    agent,
	})
}

func updateAgentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]
	
	var agent ScrapingAgent
	if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
		respondJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid JSON: " + err.Error(),
		})
		return
	}
	
	configJSON, _ := json.Marshal(agent.Configuration)
	tagsJSON, _ := json.Marshal(agent.Tags)
	
	query := `
		UPDATE scraping_agents 
		SET name = $1, platform = $2, agent_type = $3, configuration = $4,
		    schedule_cron = $5, enabled = $6, tags = $7, updated_at = CURRENT_TIMESTAMP
		WHERE id = $8
		RETURNING updated_at
	`
	
	err := db.QueryRow(query, agent.Name, agent.Platform, agent.AgentType,
		configJSON, agent.ScheduleCron, agent.Enabled, tagsJSON, agentID).Scan(&agent.UpdatedAt)
	
	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusNotFound, APIResponse{
			Success: false,
			Error:   "Agent not found",
		})
		return
	} else if err != nil {
		respondJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to update agent: " + err.Error(),
		})
		return
	}
	
	agent.ID = agentID
	respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    agent,
		Message: "Agent updated successfully",
	})
}

func deleteAgentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]
	
	result, err := db.Exec("DELETE FROM scraping_agents WHERE id = $1", agentID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to delete agent: " + err.Error(),
		})
		return
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondJSON(w, http.StatusNotFound, APIResponse{
			Success: false,
			Error:   "Agent not found",
		})
		return
	}
	
	respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Agent deleted successfully",
	})
}

func getTargetsHandler(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, agent_id, url, selector_config, authentication, 
		       headers, proxy_config, rate_limit_ms, max_retries, created_at
		FROM scraping_targets 
		ORDER BY created_at DESC
	`
	
	rows, err := db.Query(query)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to query targets: " + err.Error(),
		})
		return
	}
	defer rows.Close()
	
	var targets []ScrapingTarget
	for rows.Next() {
		var target ScrapingTarget
		var selectorJSON, authJSON, headersJSON, proxyJSON []byte
		
		err := rows.Scan(
			&target.ID, &target.AgentID, &target.URL,
			&selectorJSON, &authJSON, &headersJSON, &proxyJSON,
			&target.RateLimitMs, &target.MaxRetries, &target.CreatedAt,
		)
		if err != nil {
			log.Printf("Error scanning target row: %v", err)
			continue
		}
		
		// Parse JSON fields
		if len(selectorJSON) > 0 {
			json.Unmarshal(selectorJSON, &target.SelectorConfig)
		}
		if len(authJSON) > 0 {
			json.Unmarshal(authJSON, &target.Authentication)
		}
		if len(headersJSON) > 0 {
			json.Unmarshal(headersJSON, &target.Headers)
		}
		if len(proxyJSON) > 0 {
			json.Unmarshal(proxyJSON, &target.ProxyConfig)
		}
		
		targets = append(targets, target)
	}
	
	if targets == nil {
		targets = []ScrapingTarget{}
	}
	
	respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    targets,
	})
}

func createTargetHandler(w http.ResponseWriter, r *http.Request) {
	var target ScrapingTarget
	if err := json.NewDecoder(r.Body).Decode(&target); err != nil {
		respondJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid JSON: " + err.Error(),
		})
		return
	}
	
	// Validate required fields
	if target.AgentID == "" || target.URL == "" {
		respondJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Agent ID and URL are required",
		})
		return
	}
	
	selectorJSON, _ := json.Marshal(target.SelectorConfig)
	authJSON, _ := json.Marshal(target.Authentication)
	headersJSON, _ := json.Marshal(target.Headers)
	proxyJSON, _ := json.Marshal(target.ProxyConfig)
	
	query := `
		INSERT INTO scraping_targets (agent_id, url, selector_config, authentication,
		                            headers, proxy_config, rate_limit_ms, max_retries)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`
	
	err := db.QueryRow(query, target.AgentID, target.URL, selectorJSON,
		authJSON, headersJSON, proxyJSON, target.RateLimitMs, target.MaxRetries).Scan(
		&target.ID, &target.CreatedAt)
	
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to create target: " + err.Error(),
		})
		return
	}
	
	respondJSON(w, http.StatusCreated, APIResponse{
		Success: true,
		Data:    target,
		Message: "Target created successfully",
	})
}

func getAgentTargetsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["agentId"]
	
	query := `
		SELECT id, agent_id, url, selector_config, authentication,
		       headers, proxy_config, rate_limit_ms, max_retries, created_at
		FROM scraping_targets 
		WHERE agent_id = $1
		ORDER BY created_at DESC
	`
	
	rows, err := db.Query(query, agentID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to query targets: " + err.Error(),
		})
		return
	}
	defer rows.Close()
	
	var targets []ScrapingTarget
	for rows.Next() {
		var target ScrapingTarget
		var selectorJSON, authJSON, headersJSON, proxyJSON []byte
		
		err := rows.Scan(
			&target.ID, &target.AgentID, &target.URL,
			&selectorJSON, &authJSON, &headersJSON, &proxyJSON,
			&target.RateLimitMs, &target.MaxRetries, &target.CreatedAt,
		)
		if err != nil {
			log.Printf("Error scanning target row: %v", err)
			continue
		}
		
		// Parse JSON fields
		if len(selectorJSON) > 0 {
			json.Unmarshal(selectorJSON, &target.SelectorConfig)
		}
		if len(authJSON) > 0 {
			json.Unmarshal(authJSON, &target.Authentication)
		}
		if len(headersJSON) > 0 {
			json.Unmarshal(headersJSON, &target.Headers)
		}
		if len(proxyJSON) > 0 {
			json.Unmarshal(proxyJSON, &target.ProxyConfig)
		}
		
		targets = append(targets, target)
	}
	
	if targets == nil {
		targets = []ScrapingTarget{}
	}
	
	respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    targets,
	})
}

func getResultsHandler(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	limit := r.URL.Query().Get("limit")
	
	query := `
		SELECT id, agent_id, target_id, run_id, status, data, screenshots,
		       extracted_count, error_message, started_at, completed_at, execution_time_ms
		FROM scraping_results 
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1
	
	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}
	
	query += " ORDER BY started_at DESC"
	
	if limit != "" {
		if limitInt, err := strconv.Atoi(limit); err == nil && limitInt > 0 {
			query += fmt.Sprintf(" LIMIT $%d", argIndex)
			args = append(args, limitInt)
		}
	}
	
	rows, err := db.Query(query, args...)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to query results: " + err.Error(),
		})
		return
	}
	defer rows.Close()
	
	var results []ScrapingResult
	for rows.Next() {
		var result ScrapingResult
		var dataJSON, screenshotsJSON []byte
		
		err := rows.Scan(
			&result.ID, &result.AgentID, &result.TargetID, &result.RunID,
			&result.Status, &dataJSON, &screenshotsJSON, &result.ExtractedCount,
			&result.ErrorMessage, &result.StartedAt, &result.CompletedAt,
			&result.ExecutionTimeMs,
		)
		if err != nil {
			log.Printf("Error scanning result row: %v", err)
			continue
		}
		
		// Parse JSON fields
		if len(dataJSON) > 0 {
			json.Unmarshal(dataJSON, &result.Data)
		}
		if len(screenshotsJSON) > 0 {
			json.Unmarshal(screenshotsJSON, &result.Screenshots)
		}
		
		results = append(results, result)
	}
	
	if results == nil {
		results = []ScrapingResult{}
	}
	
	respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    results,
	})
}

func getAgentResultsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["agentId"]
	
	query := `
		SELECT id, agent_id, target_id, run_id, status, data, screenshots,
		       extracted_count, error_message, started_at, completed_at, execution_time_ms
		FROM scraping_results 
		WHERE agent_id = $1
		ORDER BY started_at DESC
	`
	
	rows, err := db.Query(query, agentID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to query results: " + err.Error(),
		})
		return
	}
	defer rows.Close()
	
	var results []ScrapingResult
	for rows.Next() {
		var result ScrapingResult
		var dataJSON, screenshotsJSON []byte
		
		err := rows.Scan(
			&result.ID, &result.AgentID, &result.TargetID, &result.RunID,
			&result.Status, &dataJSON, &screenshotsJSON, &result.ExtractedCount,
			&result.ErrorMessage, &result.StartedAt, &result.CompletedAt,
			&result.ExecutionTimeMs,
		)
		if err != nil {
			log.Printf("Error scanning result row: %v", err)
			continue
		}
		
		// Parse JSON fields
		if len(dataJSON) > 0 {
			json.Unmarshal(dataJSON, &result.Data)
		}
		if len(screenshotsJSON) > 0 {
			json.Unmarshal(screenshotsJSON, &result.Screenshots)
		}
		
		results = append(results, result)
	}
	
	if results == nil {
		results = []ScrapingResult{}
	}
	
	respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    results,
	})
}

func getPlatformsHandler(w http.ResponseWriter, r *http.Request) {
	// Return static platform capabilities for now
	// In a real implementation, this could be loaded from the database or config
	platforms := []PlatformCapabilities{
		{
			Platform:     "huginn",
			AgentTypes:   []string{"WebsiteAgent", "RssAgent", "TwitterStreamAgent", "EmailAgent", "JsonAgent", "DataOutputAgent"},
			Features:     map[string]interface{}{"scheduling": true, "chaining": true, "filtering": true, "templating": true, "webhooks": true},
			BestFor:      "RSS feeds, social media monitoring, scheduled scraping",
		},
		{
			Platform:     "browserless",
			Capabilities: []string{"screenshot", "pdf", "content", "scrape", "function"},
			Features:     map[string]interface{}{"javascript_rendering": true, "cookie_management": true, "proxy_support": true, "stealth_mode": true, "parallel_execution": true},
			BestFor:      "JavaScript-heavy sites, screenshots, PDF generation",
		},
		{
			Platform:     "agent-s2",
			Capabilities: []string{"navigate", "interact", "extract", "monitor", "automate"},
			Features:     map[string]interface{}{"ai_powered": true, "visual_recognition": true, "natural_language": true, "adaptive_learning": true, "complex_workflows": true},
			BestFor:      "Complex interactions, dynamic content, AI-guided scraping",
		},
	}
	
	respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    platforms,
	})
}

func executeAgentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]
	
	// This would trigger execution via n8n workflow
	// For now, return a simple response
	respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Agent %s execution triggered", agentID),
		Data: map[string]interface{}{
			"agent_id":    agentID,
			"run_id":      fmt.Sprintf("run_%d", time.Now().Unix()),
			"status":      "queued",
			"started_at":  time.Now().UTC(),
		},
	})
}

func executeWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowID := vars["id"]
	
	// This would trigger execution via n8n workflow
	respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Workflow %s execution triggered", workflowID),
		Data: map[string]interface{}{
			"workflow_id": workflowID,
			"run_id":      fmt.Sprintf("run_%d", time.Now().Unix()),
			"status":      "queued",
			"started_at":  time.Now().UTC(),
		},
	})
}

func exportDataHandler(w http.ResponseWriter, r *http.Request) {
	var exportRequest struct {
		AgentIDs []string `json:"agent_ids"`
		Format   string   `json:"format"`
		Filters  map[string]interface{} `json:"filters"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&exportRequest); err != nil {
		respondJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid JSON: " + err.Error(),
		})
		return
	}
	
	// This would trigger export workflow
	respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Export job queued",
		Data: map[string]interface{}{
			"export_id":  fmt.Sprintf("export_%d", time.Now().Unix()),
			"format":     exportRequest.Format,
			"agent_ids":  exportRequest.AgentIDs,
			"status":     "queued",
			"created_at": time.Now().UTC(),
		},
	})
}

func getMetricsHandler(w http.ResponseWriter, r *http.Request) {
	// Return basic metrics
	var totalAgents, activeAgents, totalResults int
	
	db.QueryRow("SELECT COUNT(*) FROM scraping_agents").Scan(&totalAgents)
	db.QueryRow("SELECT COUNT(*) FROM scraping_agents WHERE enabled = true").Scan(&activeAgents)
	db.QueryRow("SELECT COUNT(*) FROM scraping_results").Scan(&totalResults)
	
	respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"total_agents":  totalAgents,
			"active_agents": activeAgents,
			"total_results": totalResults,
			"timestamp":     time.Now().UTC(),
		},
	})
}

func getStatusHandler(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"api":         "healthy",
		"database":    "connected",
		"timestamp":   time.Now().UTC(),
		"version":     "1.0.0",
		"uptime":      "running",
	}
	
	// Check database
	if err := db.Ping(); err != nil {
		status["database"] = "disconnected"
		status["api"] = "degraded"
	}
	
	respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    status,
	})
}