package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
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

// Entity type constants
const (
	EntityTypeScenario = "scenario"
	EntityTypeResource = "resource"
)

// Draft status constants
const (
	DraftStatusDraft     = "draft"
	DraftStatusPublished = "published"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status       string         `json:"status"`
	Service      string         `json:"service"`
	Timestamp    string         `json:"timestamp"`
	Readiness    bool           `json:"readiness"`
	Dependencies map[string]any `json:"dependencies"`
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
		slog.Error("API_PORT environment variable must be set")
		os.Exit(1)
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

	// Global middleware
	router.Use(corsMiddleware)
	router.Use(jsonMiddleware)

	// Health check (standard endpoint for ecosystem interoperability)
	router.HandleFunc("/health", handleHealth).Methods("GET")

	// API v1 routes with JSON and database middleware
	apiV1 := router.PathPrefix("/api/v1").Subrouter()
	apiV1.Use(requireDBMiddleware)

	// Legacy health check endpoint for backward compatibility
	apiV1.HandleFunc("/health", handleHealth).Methods("GET")

	// Catalog endpoints
	apiV1.HandleFunc("/catalog", handleGetCatalog).Methods("GET")
	apiV1.HandleFunc("/catalog/{type}/{name}", handleGetPublishedPRD).Methods("GET")
	apiV1.HandleFunc("/catalog/{type}/{name}/draft", handleEnsureDraftFromPublishedPRD).Methods("POST")
	apiV1.HandleFunc("/catalog/{type}/{name}/requirements", handleGetRequirements).Methods("GET")
	apiV1.HandleFunc("/catalog/{type}/{name}/requirements/{requirement_id}", handleUpdateRequirement).Methods("PUT")
	apiV1.HandleFunc("/catalog/{type}/{name}/targets", handleGetOperationalTargets).Methods("GET")

	// Draft endpoints
	apiV1.HandleFunc("/drafts", handleListDrafts).Methods("GET")
	apiV1.HandleFunc("/drafts", handleCreateDraft).Methods("POST")
	apiV1.HandleFunc("/drafts/{id}", handleGetDraft).Methods("GET")
	apiV1.HandleFunc("/drafts/{id}", handleUpdateDraft).Methods("PUT")
	apiV1.HandleFunc("/drafts/{id}", handleDeleteDraft).Methods("DELETE")
	apiV1.HandleFunc("/drafts/{id}/targets", handleGetDraftTargets).Methods("GET")
	apiV1.HandleFunc("/drafts/{id}/targets", handleUpdateDraftTargets).Methods("PUT")

	// Backlog endpoints
	apiV1.HandleFunc("/backlog", handleListBacklog).Methods("GET")
	apiV1.HandleFunc("/backlog", handleCreateBacklogEntries).Methods("POST")
	apiV1.HandleFunc("/backlog/convert", handleConvertBacklogEntries).Methods("POST")
	apiV1.HandleFunc("/backlog/{id}/convert", handleConvertSingleBacklogEntry).Methods("POST")
	apiV1.HandleFunc("/backlog/{id}", handleDeleteBacklogEntry).Methods("DELETE")

	// Validation and AI endpoints
	apiV1.HandleFunc("/drafts/validate", handleValidatePRD).Methods("POST")
	apiV1.HandleFunc("/drafts/{id}/validate", handleValidateDraft).Methods("POST")
	apiV1.HandleFunc("/drafts/{id}/ai/generate-section", handleAIGenerateSection).Methods("POST")
	apiV1.HandleFunc("/drafts/{id}/publish", handlePublishDraft).Methods("POST")

	// Quality insights endpoints
	apiV1.HandleFunc("/quality/{type}/{name}", handleGetQualityReport).Methods("GET")
	apiV1.HandleFunc("/quality/scan", handleQualityScan).Methods("POST")
	apiV1.HandleFunc("/quality/summary", handleQualitySummary).Methods("GET")

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

// jsonMiddleware sets JSON content type for API responses
func jsonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// requireDBMiddleware checks database availability before processing requests
func requireDBMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if db == nil {
			respondServiceUnavailable(w, "Database not available")
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
	health := HealthResponse{
		Status:       "healthy",
		Service:      "prd-control-tower-api",
		Timestamp:    time.Now().Format(time.RFC3339),
		Readiness:    true,
		Dependencies: map[string]any{},
	}

	// Check database
	if db != nil {
		if err := db.Ping(); err != nil {
			health.Dependencies["database"] = map[string]any{
				"status": "error",
				"error":  err.Error(),
			}
			health.Status = "degraded"
		} else {
			health.Dependencies["database"] = map[string]any{
				"status": "healthy",
			}
		}
	} else {
		health.Dependencies["database"] = map[string]any{
			"status": "not_initialized",
		}
		health.Status = "degraded"
	}

	// Check draft directory (relative to scenario root, one level up from api/)
	draftDir := "../data/prd-drafts"
	if _, err := os.Stat(draftDir); os.IsNotExist(err) {
		health.Dependencies["draft_storage"] = map[string]any{
			"status": "error",
			"error":  "Draft directory does not exist",
		}
		health.Status = "degraded"
	} else {
		health.Dependencies["draft_storage"] = map[string]any{
			"status": "healthy",
		}
	}

	respondJSON(w, http.StatusOK, health)
}

// respondJSON encodes v as JSON and writes it to w with the specified status code, logging any errors
func respondJSON(w http.ResponseWriter, statusCode int, v any) {
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

// isValidEntityType checks if the given entity type is valid (scenario or resource)
func isValidEntityType(entityType string) bool {
	return entityType == EntityTypeScenario || entityType == EntityTypeResource
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
