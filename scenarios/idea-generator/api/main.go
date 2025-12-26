package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"

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
	// Connect to database using api-core with automatic retry
	db, err := initDB()
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
		postgresURL:     os.Getenv("POSTGRES_URL"),
		qdrantURL:       qdrantURL,
		minioURL:        minioURL,
		redisURL:        redisURL,
		ollamaURL:       ollamaURL,
		unstructuredURL: unstructuredURL,
	}, nil
}

func initDB() (*sql.DB, error) {
	db, err := database.Connect(context.Background(), database.Config{
		Driver: "postgres",
	})
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Println("🎉 Database connection pool established successfully!")
	return db, nil
}

// Handler functions are now in separate files:
// - handlers_health.go: Health, status, and workflow capability handlers
// - handlers_campaign.go: Campaign CRUD handlers
// - handlers_idea.go: Idea generation, refinement, search, and document processing handlers

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "idea-generator",
	}) {
		return // Process was re-exec'd after rebuild
	}

	apiServer, err := NewApiServer()
	if err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}

	r := mux.NewRouter()

	// API routes - all under /api prefix for consistency
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/ideas/generate", apiServer.generateIdeasHandler).Methods("POST")
	api.HandleFunc("/ideas", apiServer.ideasHandler).Methods("GET", "POST")
	api.HandleFunc("/ideas/refine", apiServer.refineIdeaHandler).Methods("POST")
	api.HandleFunc("/campaigns", apiServer.campaignsHandler).Methods("GET", "POST")
	api.HandleFunc("/campaigns/{id}", apiServer.campaignByIDHandler).Methods("GET", "DELETE")
	api.HandleFunc("/search", apiServer.searchHandler).Methods("POST")
	api.HandleFunc("/documents/process", apiServer.processDocumentHandler).Methods("POST")
	api.HandleFunc("/workflows", apiServer.workflowsHandler).Methods("GET")

	// Health and status endpoints at root for standard compliance
	r.HandleFunc("/health", apiServer.healthHandler).Methods("GET")
	r.HandleFunc("/status", apiServer.statusHandler).Methods("GET")

	// Enable CORS
	corsHandler := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"*"}),
	)(r)

	log.Printf("Idea Generator API server starting")
	log.Printf("Services:")
	log.Printf("  Database: Connected")
	log.Printf("  Qdrant: %s", apiServer.qdrantURL)
	log.Printf("  MinIO: %s", apiServer.minioURL)
	log.Printf("  Ollama: %s", apiServer.ollamaURL)
	log.Printf("  Unstructured: %s", apiServer.unstructuredURL)

	if err := server.Run(server.Config{
		Handler: corsHandler,
		Cleanup: func(ctx context.Context) error {
			if apiServer.db != nil {
				return apiServer.db.Close()
			}
			return nil
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
