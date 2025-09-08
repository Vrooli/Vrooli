package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"time"

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
}

type Application struct {
	ID               int       `json:"id"`
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
	ID                   int       `json:"id"`
	Name                 string    `json:"name"`
	Type                 string    `json:"type"`
	ApplicationID        int       `json:"application_id"`
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
	ID            int       `json:"id"`
	AgentID       int       `json:"agent_id"`
	ApplicationID int       `json:"application_id"`
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
			log.Fatal("❌ Missing database configuration. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}
		
		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}
	
	// Optional service URLs - no defaults
	return Config{
		Port:            port,
		PostgresURL:     postgresURL,
		RedisURL:        os.Getenv("REDIS_URL"),
		QdrantURL:       os.Getenv("QDRANT_URL"),
		OllamaURL:       os.Getenv("OLLAMA_URL"),
		N8NURL:          os.Getenv("N8N_URL"),
		WindmillURL:     os.Getenv("WINDMILL_URL"),
		UnstructuredURL: os.Getenv("UNSTRUCTURED_URL"),
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
		return fmt.Errorf("❌ Database connection failed after %d attempts: %w", maxRetries, pingErr)
	}
	
	log.Println("🎉 Database connection pool established successfully!")
	return nil
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   "document-manager-api",
		"version":   "2.0.0",
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
	
	resp, err := http.Get(config.QdrantURL + "/health")
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
}

func agentsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	switch r.Method {
	case "GET":
		getAgents(w, r)
	case "POST":
		createAgent(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
	}
}

func getAgents(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT 
			ag.id, ag.name, ag.type, ag.application_id, ag.configuration,
			ag.schedule_cron, ag.auto_apply_threshold, ag.last_performance_score,
			ag.enabled, ag.created_at, ag.updated_at,
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
	
	query := `
		INSERT INTO agents (name, type, application_id, configuration, schedule_cron, auto_apply_threshold, enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`
	
	err := db.QueryRow(query, agent.Name, agent.Type, agent.ApplicationID, 
		agent.Configuration, agent.ScheduleCron, agent.AutoApplyThreshold, true).Scan(
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
}

func queueHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	switch r.Method {
	case "GET":
		getQueue(w, r)
	case "POST":
		createQueueItem(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
	}
}

func getQueue(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT 
			iq.id, iq.agent_id, iq.application_id, iq.type, iq.title,
			iq.description, iq.severity, iq.status, iq.created_at, iq.updated_at,
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
		RETURNING id, created_at, updated_at
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

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

func main() {
	config = loadConfig()
	
	if err := initDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	
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
	r.HandleFunc("/api/applications", applicationsHandler).Methods("GET", "POST", "OPTIONS")
	r.HandleFunc("/api/agents", agentsHandler).Methods("GET", "POST", "OPTIONS")
	r.HandleFunc("/api/queue", queueHandler).Methods("GET", "POST", "OPTIONS")
	
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