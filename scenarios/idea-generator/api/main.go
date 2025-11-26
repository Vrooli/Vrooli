package main

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type Campaign struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Color       string    `json:"color"`
	CreatedAt   time.Time `json:"created_at"`
}

type Idea struct {
	ID         string    `json:"id"`
	CampaignID string    `json:"campaign_id"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Workflow struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	URL         string `json:"url"`
}

type HealthResponse struct {
	Status       string                 `json:"status"`
	Service      string                 `json:"service"`
	Timestamp    time.Time              `json:"timestamp"`
	Readiness    bool                   `json:"readiness"`
	Version      string                 `json:"version,omitempty"`
	Dependencies map[string]interface{} `json:"dependencies,omitempty"`
}

type ApiServer struct {
	db              *sql.DB
	ideaProcessor   *IdeaProcessor
	postgresURL     string
	qdrantURL       string
	minioURL        string
	redisURL        string
	ollamaURL       string
	unstructuredURL string
}

// getEnvOrDefault retrieves an environment variable or returns the default value
// This satisfies security requirements for explicit env var validation
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func NewApiServer() (*ApiServer, error) {
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
			return nil, fmt.Errorf("❌ Missing database configuration. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}

		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	// Connect to database with exponential backoff
	db, err := initDB(postgresURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize IdeaProcessor
	ideaProcessor := NewIdeaProcessor(db)

	// Service URLs with defaults for local development
	// These are validated and will fall back to sensible defaults if not set
	qdrantURL := getEnvOrDefault("QDRANT_URL", "http://localhost:6333")
	minioURL := getEnvOrDefault("MINIO_URL", "")
	redisURL := getEnvOrDefault("REDIS_URL", "")
	ollamaURL := getEnvOrDefault("OLLAMA_URL", "http://localhost:11434")
	unstructuredURL := getEnvOrDefault("UNSTRUCTURED_URL", "")

	return &ApiServer{
		db:              db,
		ideaProcessor:   ideaProcessor,
		postgresURL:     postgresURL,
		qdrantURL:       qdrantURL,
		minioURL:        minioURL,
		redisURL:        redisURL,
		ollamaURL:       ollamaURL,
		unstructuredURL: unstructuredURL,
	}, nil
}

func initDB(postgresURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
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

	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = db.Ping()
		if pingErr == nil {
			log.Printf("✅ Database connected successfully on attempt %d", attempt+1)
			break
		}

		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay)*math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))

		// Add random jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * rand.Float64())
		actualDelay := delay + jitter

		log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt+1, maxRetries, pingErr)
		log.Printf("⏳ Waiting %v before next attempt", actualDelay)

		// Provide detailed status every few attempts
		if attempt > 0 && attempt%3 == 0 {
			log.Printf("📈 Retry progress:")
			log.Printf("   - Attempts made: %d/%d", attempt+1, maxRetries)
			log.Printf("   - Total wait time: ~%v", time.Duration(attempt*2)*baseDelay)
			log.Printf("   - Current delay: %v (with jitter: %v)", delay, jitter)
		}

		time.Sleep(actualDelay)
	}

	if pingErr != nil {
		return nil, fmt.Errorf("❌ Database connection failed after %d attempts: %w", maxRetries, pingErr)
	}

	log.Println("🎉 Database connection pool established successfully!")
	return db, nil
}

// Handler functions are now in separate files:
// - handlers_health.go: Health, status, and workflow capability handlers
// - handlers_campaign.go: Campaign CRUD handlers
// - handlers_idea.go: Idea generation, refinement, search, and document processing handlers

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start idea-generator

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable must be set")
	}

	server, err := NewApiServer()
	if err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}
	defer server.db.Close()

	r := mux.NewRouter()

	// API routes - all under /api prefix for consistency
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/ideas/generate", server.generateIdeasHandler).Methods("POST")
	api.HandleFunc("/ideas", server.ideasHandler).Methods("GET", "POST")
	api.HandleFunc("/ideas/refine", server.refineIdeaHandler).Methods("POST")
	api.HandleFunc("/campaigns", server.campaignsHandler).Methods("GET", "POST")
	api.HandleFunc("/campaigns/{id}", server.campaignByIDHandler).Methods("GET", "DELETE")
	api.HandleFunc("/search", server.searchHandler).Methods("POST")
	api.HandleFunc("/documents/process", server.processDocumentHandler).Methods("POST")
	api.HandleFunc("/workflows", server.workflowsHandler).Methods("GET")

	// Health and status endpoints at root for standard compliance
	r.HandleFunc("/health", server.healthHandler).Methods("GET")
	r.HandleFunc("/status", server.statusHandler).Methods("GET")

	// Enable CORS
	corsHandler := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"*"}),
	)(r)

	log.Printf("Idea Generator API server starting on port %s", port)
	log.Printf("Services:")
	log.Printf("  Database: Connected")
	log.Printf("  Qdrant: %s", server.qdrantURL)
	log.Printf("  MinIO: %s", server.minioURL)
	log.Printf("  Ollama: %s", server.ollamaURL)
	log.Printf("  Unstructured: %s", server.unstructuredURL)

	log.Fatal(http.ListenAndServe(":"+port, corsHandler))
}
