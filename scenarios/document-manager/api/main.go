package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type Config struct {
	Port            string
	PostgresURL     string
	RedisURL        string
	QdrantURL       string
	OllamaURL       string
	N8NURL          string
	WindmillURL     string
	UnstructuredURL string
	CORSOrigins     string
}

type Application struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	RepositoryURL    string    `json:"repository_url"`
	DocumentationPath string   `json:"documentation_path"`
	HealthScore      float64   `json:"health_score"`
	Active           bool      `json:"active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	AgentCount       int       `json:"agent_count,omitempty"`
	Status           string    `json:"status,omitempty"`
}

type Agent struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	Type                 string    `json:"type"`
	ApplicationID        string    `json:"application_id"`
	Configuration        string    `json:"configuration"`
	ScheduleCron         string    `json:"schedule_cron"`
	AutoApplyThreshold   float64   `json:"auto_apply_threshold"`
	LastPerformanceScore float64   `json:"last_performance_score"`
	Enabled              bool      `json:"enabled"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
	ApplicationName      string    `json:"application_name,omitempty"`
	Status               string    `json:"status,omitempty"`
}

type ImprovementQueue struct {
	ID            string    `json:"id"`
	AgentID       string    `json:"agent_id"`
	ApplicationID string    `json:"application_id"`
	Type          string    `json:"type"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Severity      string    `json:"severity"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	AgentName     string    `json:"agent_name,omitempty"`
	ApplicationName string  `json:"application_name,omitempty"`
}

type SystemStatus struct {
	Service string `json:"service"`
	Status  string `json:"status"`
	Details string `json:"details,omitempty"`
}

type SearchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

type SearchResult struct {
	ID          string                 `json:"id"`
	Score       float64                `json:"score"`
	DocumentID  string                 `json:"document_id,omitempty"`
	Content     string                 `json:"content,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ApplicationName string             `json:"application_name,omitempty"`
}

type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Query   string         `json:"query"`
	Total   int            `json:"total"`
}

// Document represents a document to be indexed
type Document struct {
	ID              string                 `json:"id"`
	ApplicationID   string                 `json:"application_id"`
	ApplicationName string                 `json:"application_name"`
	Path            string                 `json:"path"`
	Content         string                 `json:"content"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// IndexRequest represents a request to index documents
type IndexRequest struct {
	ApplicationID string     `json:"application_id"`
	Documents     []Document `json:"documents"`
}

// IndexResponse represents the response from indexing
type IndexResponse struct {
	Indexed int      `json:"indexed"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors,omitempty"`
}

// QdrantPoint represents a point to be inserted into Qdrant
type QdrantPoint struct {
	ID      string                 `json:"id"`
	Vector  []float64              `json:"vector"`
	Payload map[string]interface{} `json:"payload"`
}

// QdrantSearchRequest represents a Qdrant search request
type QdrantSearchRequest struct {
	Vector []float64 `json:"vector"`
	Limit  int       `json:"limit"`
	WithPayload bool  `json:"with_payload"`
}

// QdrantSearchResponse represents a Qdrant search response
type QdrantSearchResponse struct {
	Result []struct {
		ID      string                 `json:"id"`
		Score   float64                `json:"score"`
		Payload map[string]interface{} `json:"payload"`
	} `json:"result"`
}

var db *sql.DB
var config Config

func loadConfig() Config {
	// API_PORT is required - no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}
	
	// Database configuration - support both POSTGRES_URL and individual components
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		// Try to build from individual components
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")
		
		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			log.Fatal("❌ Missing database configuration. Provide POSTGRES_URL or individual connection parameters (host, port, user, password, database)")
		}
		
		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}
	
	// Optional service URLs - no defaults
	// CORS configuration - default to localhost:UI_PORT if not specified
	corsOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if corsOrigins == "" {
		uiPort := os.Getenv("UI_PORT")
		if uiPort != "" {
			corsOrigins = fmt.Sprintf("http://localhost:%s", uiPort)
		} else {
			corsOrigins = "http://localhost:3000"
		}
	}

	return Config{
		Port:            port,
		PostgresURL:     postgresURL,
		RedisURL:        os.Getenv("REDIS_URL"),
		QdrantURL:       os.Getenv("QDRANT_URL"),
		OllamaURL:       os.Getenv("OLLAMA_URL"),
		N8NURL:          os.Getenv("N8N_URL"),
		WindmillURL:     os.Getenv("WINDMILL_URL"),
		UnstructuredURL: os.Getenv("UNSTRUCTURED_URL"),
		CORSOrigins:     corsOrigins,
	}
}

func initDB() error {
	var err error
	db, err = sql.Open("postgres", config.PostgresURL)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	log.Println("🔄 Attempting database connection with exponential backoff...")
	log.Printf("📆 Database URL configured")

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

		// Add random jitter to prevent thundering herd (±25% of delay)
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * (2*rand.Float64() - 1))
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
		return fmt.Errorf("❌ Database connection failed after %d attempts: %w", maxRetries, pingErr)
	}
	
	log.Println("🎉 Database connection pool established successfully!")
	return nil
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check database connectivity
	dbConnected := false
	var dbLatency *float64
	var dbError map[string]interface{}
	if db != nil {
		start := time.Now()
		err := db.Ping()
		latencyMs := float64(time.Since(start).Microseconds()) / 1000.0
		if err == nil {
			dbConnected = true
			dbLatency = &latencyMs
		} else {
			dbError = map[string]interface{}{
				"code":      "CONNECTION_FAILED",
				"message":   err.Error(),
				"category":  "network",
				"retryable": true,
			}
		}
	}

	// Determine overall status
	status := "healthy"
	if !dbConnected {
		status = "degraded"
	}

	response := map[string]interface{}{
		"status":    status,
		"service":   "document-manager-api",
		"timestamp": time.Now().Format(time.RFC3339),
		"readiness": true,
		"version":   "2.0.0",
		"dependencies": map[string]interface{}{
			"database": map[string]interface{}{
				"connected":  dbConnected,
				"latency_ms": dbLatency,
				"error":      dbError,
			},
		},
	}

	json.NewEncoder(w).Encode(response)
}

func dbStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	status := SystemStatus{Service: "postgres", Status: "healthy"}
	
	if err := db.Ping(); err != nil {
		status.Status = "unhealthy"
		status.Details = err.Error()
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	
	json.NewEncoder(w).Encode(status)
}

func vectorStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	status := SystemStatus{Service: "qdrant", Status: "healthy"}
	
	resp, err := http.Get(config.QdrantURL + "/readyz")
	if err != nil {
		status.Status = "unhealthy"
		status.Details = err.Error()
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(status)
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) // Drain body to allow connection reuse
	
	if resp.StatusCode != 200 {
		status.Status = "unhealthy"
		status.Details = fmt.Sprintf("HTTP %d", resp.StatusCode)
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	
	json.NewEncoder(w).Encode(status)
}

func aiStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	status := SystemStatus{Service: "ollama", Status: "healthy"}
	
	resp, err := http.Get(config.OllamaURL + "/api/tags")
	if err != nil {
		status.Status = "unhealthy"
		status.Details = err.Error()
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(status)
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) // Drain body to allow connection reuse
	
	if resp.StatusCode != 200 {
		status.Status = "unhealthy"
		status.Details = fmt.Sprintf("HTTP %d", resp.StatusCode)
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	
	json.NewEncoder(w).Encode(status)
}

func applicationsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		getApplications(w, r)
	case "POST":
		createApplication(w, r)
	case "DELETE":
		deleteApplication(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
	}
}

func getApplications(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT 
			a.id, a.name, a.repository_url, a.documentation_path, 
			a.health_score, a.active, a.created_at, a.updated_at,
			COUNT(ag.id) as agent_count,
			CASE WHEN a.active = true THEN 'active' ELSE 'inactive' END as status
		FROM applications a
		LEFT JOIN agents ag ON a.id = ag.application_id AND ag.enabled = true
		GROUP BY a.id
		ORDER BY a.updated_at DESC
	`
	
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Error querying applications: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Database query failed"})
		return
	}
	defer rows.Close()
	
	var applications []Application
	for rows.Next() {
		var app Application
		err := rows.Scan(
			&app.ID, &app.Name, &app.RepositoryURL, &app.DocumentationPath,
			&app.HealthScore, &app.Active, &app.CreatedAt, &app.UpdatedAt,
			&app.AgentCount, &app.Status,
		)
		if err != nil {
			log.Printf("Error scanning application: %v", err)
			continue
		}
		applications = append(applications, app)
	}
	
	json.NewEncoder(w).Encode(applications)
}

func createApplication(w http.ResponseWriter, r *http.Request) {
	var app Application
	if err := json.NewDecoder(r.Body).Decode(&app); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}
	
	query := `
		INSERT INTO applications (name, repository_url, documentation_path, health_score, active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	
	err := db.QueryRow(query, app.Name, app.RepositoryURL, app.DocumentationPath, 0.0, true).Scan(
		&app.ID, &app.CreatedAt, &app.UpdatedAt,
	)
	if err != nil {
		log.Printf("Error creating application: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create application"})
		return
	}
	
	app.HealthScore = 0.0
	app.Active = true
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(app)

	// Publish real-time event
	publishEvent(EventApplicationCreated, app)
}

func deleteApplication(w http.ResponseWriter, r *http.Request) {
	// Get application ID from query parameter
	appID := r.URL.Query().Get("id")
	if appID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing application ID"})
		return
	}

	// Delete related records first (foreign key constraints)
	// Delete agents
	_, err := db.Exec("DELETE FROM agents WHERE application_id = $1", appID)
	if err != nil {
		log.Printf("Error deleting related agents: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to delete related agents"})
		return
	}

	// Delete queue items
	_, err = db.Exec("DELETE FROM improvement_queue WHERE application_id = $1", appID)
	if err != nil {
		log.Printf("Error deleting related queue items: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to delete related queue items"})
		return
	}

	// Delete the application
	result, err := db.Exec("DELETE FROM applications WHERE id = $1", appID)
	if err != nil {
		log.Printf("Error deleting application: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to delete application"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Application not found"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Application deleted successfully"})

	// Publish real-time event
	publishEvent(EventApplicationDeleted, map[string]string{"id": appID})
}

func agentsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		getAgents(w, r)
	case "POST":
		createAgent(w, r)
	case "DELETE":
		deleteAgent(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
	}
}

func getAgents(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT 
			ag.id, ag.name, ag.type, ag.application_id, ag.config,
			ag.schedule_cron, ag.auto_apply_threshold, ag.last_performance_score,
			ag.enabled, ag.created_at, COALESCE(ag.last_run, ag.created_at) as updated_at,
			a.name as application_name,
			CASE WHEN ag.enabled = true THEN 'enabled' ELSE 'disabled' END as status
		FROM agents ag
		JOIN applications a ON ag.application_id = a.id
		ORDER BY ag.created_at DESC
	`
	
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Error querying agents: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Database query failed"})
		return
	}
	defer rows.Close()
	
	var agents []Agent
	for rows.Next() {
		var agent Agent
		err := rows.Scan(
			&agent.ID, &agent.Name, &agent.Type, &agent.ApplicationID, &agent.Configuration,
			&agent.ScheduleCron, &agent.AutoApplyThreshold, &agent.LastPerformanceScore,
			&agent.Enabled, &agent.CreatedAt, &agent.UpdatedAt,
			&agent.ApplicationName, &agent.Status,
		)
		if err != nil {
			log.Printf("Error scanning agent: %v", err)
			continue
		}
		agents = append(agents, agent)
	}
	
	json.NewEncoder(w).Encode(agents)
}

func createAgent(w http.ResponseWriter, r *http.Request) {
	var agent Agent
	if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}
	
	// Default configuration to empty JSON object if not provided
	configJSON := agent.Configuration
	if configJSON == "" {
		configJSON = "{}"
	}
	
	query := `
		INSERT INTO agents (name, type, application_id, config, schedule_cron, auto_apply_threshold, enabled)
		VALUES ($1, $2, $3, $4::jsonb, $5, $6, $7)
		RETURNING id, created_at, COALESCE(last_run, created_at) as updated_at
	`
	
	err := db.QueryRow(query, agent.Name, agent.Type, agent.ApplicationID, 
		configJSON, agent.ScheduleCron, agent.AutoApplyThreshold, true).Scan(
		&agent.ID, &agent.CreatedAt, &agent.UpdatedAt,
	)
	if err != nil {
		log.Printf("Error creating agent: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create agent"})
		return
	}
	
	agent.Enabled = true
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(agent)

	// Publish real-time event
	publishEvent(EventAgentCreated, agent)
}

func deleteAgent(w http.ResponseWriter, r *http.Request) {
	// Get agent ID from query parameter
	agentID := r.URL.Query().Get("id")
	if agentID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing agent ID"})
		return
	}

	// Delete related queue items first (foreign key constraints)
	_, err := db.Exec("DELETE FROM improvement_queue WHERE agent_id = $1", agentID)
	if err != nil {
		log.Printf("Error deleting related queue items: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to delete related queue items"})
		return
	}

	// Delete the agent
	result, err := db.Exec("DELETE FROM agents WHERE id = $1", agentID)
	if err != nil {
		log.Printf("Error deleting agent: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to delete agent"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Agent not found"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Agent deleted successfully"})

	// Publish real-time event
	publishEvent(EventAgentDeleted, map[string]string{"id": agentID})
}

func queueHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		getQueue(w, r)
	case "POST":
		createQueueItem(w, r)
	case "DELETE":
		deleteQueueItem(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
	}
}

func getQueue(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT 
			iq.id, iq.agent_id, iq.application_id, iq.type, iq.title,
			iq.description, iq.severity, iq.status, iq.created_at, iq.created_at as updated_at,
			a.name as application_name, ag.name as agent_name
		FROM improvement_queue iq
		JOIN applications a ON iq.application_id = a.id
		JOIN agents ag ON iq.agent_id = ag.id
		ORDER BY 
			CASE iq.severity
				WHEN 'critical' THEN 1
				WHEN 'high' THEN 2
				WHEN 'medium' THEN 3
				WHEN 'low' THEN 4
			END,
			iq.created_at DESC
	`
	
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Error querying queue: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Database query failed"})
		return
	}
	defer rows.Close()
	
	var items []ImprovementQueue
	for rows.Next() {
		var item ImprovementQueue
		err := rows.Scan(
			&item.ID, &item.AgentID, &item.ApplicationID, &item.Type, &item.Title,
			&item.Description, &item.Severity, &item.Status, &item.CreatedAt, &item.UpdatedAt,
			&item.ApplicationName, &item.AgentName,
		)
		if err != nil {
			log.Printf("Error scanning queue item: %v", err)
			continue
		}
		items = append(items, item)
	}
	
	json.NewEncoder(w).Encode(items)
}

func createQueueItem(w http.ResponseWriter, r *http.Request) {
	var item ImprovementQueue
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}
	
	query := `
		INSERT INTO improvement_queue (agent_id, application_id, type, title, description, severity, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, created_at as updated_at
	`
	
	err := db.QueryRow(query, item.AgentID, item.ApplicationID, item.Type, 
		item.Title, item.Description, item.Severity, "pending").Scan(
		&item.ID, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		log.Printf("Error creating queue item: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create queue item"})
		return
	}
	
	item.Status = "pending"
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(item)

	// Publish real-time event
	publishEvent(EventQueueItemCreated, item)
}

func deleteQueueItem(w http.ResponseWriter, r *http.Request) {
	// Get queue item ID from query parameter
	itemID := r.URL.Query().Get("id")
	if itemID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing queue item ID"})
		return
	}

	// Delete the queue item
	result, err := db.Exec("DELETE FROM improvement_queue WHERE id = $1", itemID)
	if err != nil {
		log.Printf("Error deleting queue item: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to delete queue item"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Queue item not found"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Queue item deleted successfully"})

	// Publish real-time event
	publishEvent(EventQueueItemDeleted, map[string]string{"id": itemID})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Use configured CORS origins instead of wildcard
		w.Header().Set("Access-Control-Allow-Origin", config.CORSOrigins)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// vectorSearchHandler performs similarity search using Qdrant
func vectorSearchHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	var searchReq SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&searchReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}

	if searchReq.Query == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Query parameter is required"})
		return
	}

	// Default limit to 10 results
	limit := searchReq.Limit
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	// Generate embedding using Ollama's nomic-embed-text model
	embedding, err := generateOllamaEmbedding(searchReq.Query)
	if err != nil {
		log.Printf("Error generating embedding: %v, falling back to mock", err)
		embedding = generateMockEmbedding(searchReq.Query)
	}

	// Query Qdrant for similar documents
	results, err := queryQdrantSimilarity(embedding, limit)
	if err != nil {
		log.Printf("Error querying Qdrant: %v", err)
		// Return 503 Service Unavailable when vector search is unavailable
		w.WriteHeader(http.StatusServiceUnavailable)
		response := SearchResponse{
			Results: []SearchResult{},
			Query:   searchReq.Query,
			Total:   0,
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	response := SearchResponse{
		Results: results,
		Query:   searchReq.Query,
		Total:   len(results),
	}

	json.NewEncoder(w).Encode(response)
}

// indexHandler indexes documents into Qdrant for similarity search
func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	var indexReq IndexRequest
	if err := json.NewDecoder(r.Body).Decode(&indexReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}

	if indexReq.ApplicationID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "application_id is required"})
		return
	}

	if len(indexReq.Documents) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "documents array cannot be empty"})
		return
	}

	// Look up application name
	var appName string
	row := db.QueryRow("SELECT name FROM applications WHERE id = $1", indexReq.ApplicationID)
	if err := row.Scan(&appName); err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Application not found"})
		return
	}

	// Set application name on all documents
	for i := range indexReq.Documents {
		indexReq.Documents[i].ApplicationID = indexReq.ApplicationID
		indexReq.Documents[i].ApplicationName = appName
	}

	// Index documents into Qdrant
	indexed, errors, err := indexDocuments(indexReq.Documents)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	response := IndexResponse{
		Indexed: indexed,
		Failed:  len(errors),
		Errors:  errors,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// generateOllamaEmbedding generates embeddings using Ollama's nomic-embed-text model
func generateOllamaEmbedding(text string) ([]float64, error) {
	if config.OllamaURL == "" {
		return nil, fmt.Errorf("Ollama URL not configured")
	}

	// Prepare request payload
	payload := map[string]interface{}{
		"model":  "nomic-embed-text",
		"prompt": text,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make request to Ollama API
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(config.OllamaURL+"/api/embeddings", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to call Ollama API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Embedding) == 0 {
		return nil, fmt.Errorf("received empty embedding from Ollama")
	}

	return result.Embedding, nil
}

// generateMockEmbedding creates a simple embedding for demonstration
// Used as fallback when Ollama is unavailable
func generateMockEmbedding(text string) []float64 {
	// Create a deterministic but varied embedding based on text
	rand.Seed(int64(len(text)))
	embedding := make([]float64, 384) // Common embedding dimension
	for i := range embedding {
		embedding[i] = rand.Float64()*2 - 1 // Values between -1 and 1
	}
	return embedding
}

// ensureQdrantCollection ensures the collection exists in Qdrant
func ensureQdrantCollection() error {
	collectionName := "document-manager-docs"

	// Check if collection exists
	client := &http.Client{Timeout: 10 * time.Second}
	checkURL := fmt.Sprintf("%s/collections/%s", config.QdrantURL, collectionName)

	resp, err := client.Get(checkURL)
	if err == nil && resp.StatusCode == http.StatusOK {
		resp.Body.Close()
		return nil // Collection exists
	}
	if resp != nil {
		resp.Body.Close()
	}

	// Create collection with 768 dimensions (nomic-embed-text model)
	createReq := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     768,
			"distance": "Cosine",
		},
	}

	reqBody, err := json.Marshal(createReq)
	if err != nil {
		return fmt.Errorf("failed to marshal create request: %w", err)
	}

	createURL := fmt.Sprintf("%s/collections/%s", config.QdrantURL, collectionName)
	req, err := http.NewRequest("PUT", createURL, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create collection, status %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("Created Qdrant collection: %s", collectionName)
	return nil
}

// indexDocuments indexes documents into Qdrant
func indexDocuments(documents []Document) (int, []string, error) {
	collectionName := "document-manager-docs"

	// Ensure collection exists
	if err := ensureQdrantCollection(); err != nil {
		return 0, nil, fmt.Errorf("failed to ensure collection: %w", err)
	}

	indexed := 0
	var errors []string
	var points []QdrantPoint

	// Generate embeddings for all documents
	for _, doc := range documents {
		// Generate embedding for document content
		embedding, err := generateOllamaEmbedding(doc.Content)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to generate embedding for %s: %v", doc.ID, err))
			continue
		}

		// Create payload with document metadata
		payload := map[string]interface{}{
			"document_id":      doc.ID,
			"application_id":   doc.ApplicationID,
			"application_name": doc.ApplicationName,
			"path":             doc.Path,
			"content":          doc.Content,
		}

		if doc.Metadata != nil {
			for k, v := range doc.Metadata {
				payload[k] = v
			}
		}

		// Generate UUID for Qdrant from document ID
		// Use deterministic UUID v5 based on document ID
		namespace := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
		pointID := uuid.NewSHA1(namespace, []byte(doc.ID)).String()

		points = append(points, QdrantPoint{
			ID:      pointID,
			Vector:  embedding,
			Payload: payload,
		})
	}

	// Batch insert points into Qdrant
	if len(points) > 0 {
		upsertReq := map[string]interface{}{
			"points": points,
		}

		reqBody, err := json.Marshal(upsertReq)
		if err != nil {
			return 0, errors, fmt.Errorf("failed to marshal upsert request: %w", err)
		}

		client := &http.Client{Timeout: 30 * time.Second}
		upsertURL := fmt.Sprintf("%s/collections/%s/points", config.QdrantURL, collectionName)

		req, err := http.NewRequest("PUT", upsertURL, bytes.NewReader(reqBody))
		if err != nil {
			return 0, errors, fmt.Errorf("failed to create upsert request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return 0, errors, fmt.Errorf("failed to upsert points: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return 0, errors, fmt.Errorf("failed to upsert points, status %d: %s", resp.StatusCode, string(body))
		}

		indexed = len(points)
	}

	return indexed, errors, nil
}

// queryQdrantSimilarity searches Qdrant for similar vectors
func queryQdrantSimilarity(embedding []float64, limit int) ([]SearchResult, error) {
	collectionName := "document-manager-docs"

	// Check if Qdrant is available
	if config.QdrantURL == "" {
		return []SearchResult{}, fmt.Errorf("Qdrant URL not configured")
	}

	// Build search request
	searchReq := QdrantSearchRequest{
		Vector:      embedding,
		Limit:       limit,
		WithPayload: true,
	}

	reqBody, err := json.Marshal(searchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search request: %w", err)
	}

	// Make search request to Qdrant
	client := &http.Client{Timeout: 10 * time.Second}
	searchURL := fmt.Sprintf("%s/collections/%s/points/search", config.QdrantURL, collectionName)

	req, err := http.NewRequest("POST", searchURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create search request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed, status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var qdrantResp QdrantSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&qdrantResp); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	// Convert Qdrant results to SearchResults
	results := make([]SearchResult, 0, len(qdrantResp.Result))
	for _, item := range qdrantResp.Result {
		result := SearchResult{
			ID:    item.ID,
			Score: item.Score,
		}

		// Extract payload fields
		if docID, ok := item.Payload["document_id"].(string); ok {
			result.DocumentID = docID
		}
		if content, ok := item.Payload["content"].(string); ok {
			result.Content = content
		}
		if appName, ok := item.Payload["application_name"].(string); ok {
			result.ApplicationName = appName
		}

		// Store full payload as metadata
		result.Metadata = item.Payload

		results = append(results, result)
	}

	return results, nil
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start document-manager

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	config = loadConfig()

	if err := initDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize Redis for real-time updates
	if err := initRedis(config.RedisURL); err != nil {
		log.Printf("Warning: Redis initialization failed: %v", err)
	}
	defer closeRedis()
	
	r := mux.NewRouter()
	
	// Add middleware
	r.Use(corsMiddleware)
	r.Use(loggingMiddleware)
	
	// Health and system endpoints
	r.HandleFunc("/health", healthHandler).Methods("GET")
	r.HandleFunc("/api/system/db-status", dbStatusHandler).Methods("GET")
	r.HandleFunc("/api/system/vector-status", vectorStatusHandler).Methods("GET")
	r.HandleFunc("/api/system/ai-status", aiStatusHandler).Methods("GET")
	
	// Main API endpoints
	r.HandleFunc("/api/applications", applicationsHandler).Methods("GET", "POST", "DELETE", "OPTIONS")
	r.HandleFunc("/api/agents", agentsHandler).Methods("GET", "POST", "DELETE", "OPTIONS")
	r.HandleFunc("/api/queue", queueHandler).Methods("GET", "POST", "DELETE", "OPTIONS")
	r.HandleFunc("/api/queue/batch", batchQueueHandler).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/search", vectorSearchHandler).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/index", indexHandler).Methods("POST", "OPTIONS")

	// Static file serving for any additional assets
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	
	port := config.Port
	log.Printf("Document Manager API starting on port %s", port)
	log.Printf("Database: %s", config.PostgresURL)
	log.Printf("Qdrant: %s", config.QdrantURL)
	log.Printf("Ollama: %s", config.OllamaURL)
	log.Printf("Windmill: %s", config.WindmillURL)
	log.Printf("n8n: %s", config.N8NURL)
	
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}