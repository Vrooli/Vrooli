package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

var db *sql.DB

// HealthResponse represents the health check response
type HealthResponse struct {
	Status       string                 `json:"status"`
	Service      string                 `json:"service"`
	Timestamp    string                 `json:"timestamp"`
	Readiness    bool                   `json:"readiness"`
	Dependencies map[string]interface{} `json:"dependencies"`
}

func main() {
	// Lifecycle protection check
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start prd-control-tower

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Validate required environment variables
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("API_PORT environment variable must be set")
	}

	// Validate RESOURCE_OPENROUTER_URL if AI features are expected
	// Note: This is optional - AI features will gracefully degrade without it
	openrouterURL := os.Getenv("RESOURCE_OPENROUTER_URL")
	if openrouterURL == "" {
		slog.Warn("RESOURCE_OPENROUTER_URL not set - AI assistance features will be unavailable")
	}

	// Initialize database
	if err := initDB(); err != nil {
		slog.Warn("Database initialization failed", "error", err)
	}

	router := mux.NewRouter()

	// CORS middleware
	router.Use(corsMiddleware)

	// Health check (standard endpoint for ecosystem interoperability)
	router.HandleFunc("/health", handleHealth).Methods("GET")
	// Legacy health check endpoint for backward compatibility
	router.HandleFunc("/api/v1/health", handleHealth).Methods("GET")

	// Catalog endpoints
	router.HandleFunc("/api/v1/catalog", handleGetCatalog).Methods("GET")
	router.HandleFunc("/api/v1/catalog/{type}/{name}", handleGetPublishedPRD).Methods("GET")

	// Draft endpoints
	router.HandleFunc("/api/v1/drafts", handleListDrafts).Methods("GET")
	router.HandleFunc("/api/v1/drafts", handleCreateDraft).Methods("POST")
	router.HandleFunc("/api/v1/drafts/{id}", handleGetDraft).Methods("GET")
	router.HandleFunc("/api/v1/drafts/{id}", handleUpdateDraft).Methods("PUT")
	router.HandleFunc("/api/v1/drafts/{id}", handleDeleteDraft).Methods("DELETE")

	// Validation and AI endpoints
	router.HandleFunc("/api/v1/drafts/validate", handleValidatePRD).Methods("POST")
	router.HandleFunc("/api/v1/drafts/{id}/validate", handleValidateDraft).Methods("POST")
	router.HandleFunc("/api/v1/drafts/{id}/ai/generate-section", handleAIGenerateSection).Methods("POST")
	router.HandleFunc("/api/v1/drafts/{id}/publish", handlePublishDraft).Methods("POST")

	slog.Info("PRD Control Tower API starting", "port", port, "service", "prd-control-tower")
	if err := http.ListenAndServe(":"+port, router); err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}

func initDB() error {
	dbHost := os.Getenv("POSTGRES_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("POSTGRES_PORT")
	if dbPort == "" {
		return fmt.Errorf("POSTGRES_PORT environment variable must be set")
	}
	dbName := os.Getenv("POSTGRES_DB")
	if dbName == "" {
		dbName = "vrooli"
	}
	dbUser := os.Getenv("POSTGRES_USER")
	if dbUser == "" {
		dbUser = "vrooli"
	}
	dbPass := os.Getenv("POSTGRES_PASSWORD")
	if dbPass == "" {
		return fmt.Errorf("POSTGRES_PASSWORD environment variable must be set")
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPass, dbName)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err = db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get allowed origins from environment or use default localhost origins for development
		allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
		if allowedOrigins == "" {
			// Default to localhost origins for development
			allowedOrigins = "http://localhost:36300,http://localhost:36301,http://127.0.0.1:36300,http://127.0.0.1:36301"
		}

		origin := r.Header.Get("Origin")
		// Check if origin is in allowed list
		if origin != "" {
			for _, allowed := range splitOrigins(allowedOrigins) {
				if origin == allowed {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					break
				}
			}
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// splitOrigins splits a comma-separated list of origins
func splitOrigins(origins string) []string {
	var result []string
	for _, origin := range strings.Split(origins, ",") {
		trimmed := strings.TrimSpace(origin)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	health := HealthResponse{
		Status:       "healthy",
		Service:      "prd-control-tower-api",
		Timestamp:    time.Now().Format(time.RFC3339),
		Readiness:    true,
		Dependencies: map[string]interface{}{},
	}

	// Check database
	if db != nil {
		if err := db.Ping(); err != nil {
			health.Dependencies["database"] = map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			}
			health.Status = "degraded"
		} else {
			health.Dependencies["database"] = map[string]interface{}{
				"status": "healthy",
			}
		}
	} else {
		health.Dependencies["database"] = map[string]interface{}{
			"status": "not_initialized",
		}
		health.Status = "degraded"
	}

	// Check draft directory (relative to scenario root, one level up from api/)
	draftDir := "../data/prd-drafts"
	if _, err := os.Stat(draftDir); os.IsNotExist(err) {
		health.Dependencies["draft_storage"] = map[string]interface{}{
			"status": "error",
			"error":  "Draft directory does not exist",
		}
		health.Status = "degraded"
	} else {
		health.Dependencies["draft_storage"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	json.NewEncoder(w).Encode(health)
}

// getVrooliRoot returns the Vrooli root directory from environment or default
func getVrooliRoot() (string, error) {
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		home := os.Getenv("HOME")
		if home == "" {
			return "", fmt.Errorf("neither VROOLI_ROOT nor HOME environment variable is set")
		}
		vrooliRoot = filepath.Join(home, "Vrooli")
	}
	return vrooliRoot, nil
}
