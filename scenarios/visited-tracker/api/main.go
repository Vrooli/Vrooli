package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	repocontract "github.com/vrooli/repo-contract-go"
)

var (
	logger        *log.Logger
	healthHandler = newHealthHandler()
)

func newHealthHandler() http.HandlerFunc {
	return health.New(serviceName).
		Version(apiVersion).
		Check(health.Func("storage", storageHealthCheck), health.Optional).
		Handler()
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "visited-tracker",
	}) {
		return // Process was re-exec'd after rebuild
	}

	projectRoot, err := resolveProjectRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to determine project root directory: %v\n", err)
		os.Exit(1)
	}

	// Change working directory to project root for file pattern resolution
	if err := os.Chdir(projectRoot); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to change to project root directory: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger = log.New(os.Stdout, "[visited-tracker] ", log.LstdFlags)

	// Log current working directory for transparency
	if cwd, err := os.Getwd(); err == nil {
		logger.Printf("📁 Working directory: %s", cwd)
		logger.Printf("💡 File patterns will be resolved relative to this directory")
	}

	// Initialize JSON file storage
	if err := initFileStorage(); err != nil {
		logger.Fatalf("File storage initialization failed: %v", err)
	}

	// Get port from environment
	port := os.Getenv("API_PORT")
	if port == "" {
		logger.Fatal("❌ API_PORT environment variable is required")
	}

	// Setup router
	r := mux.NewRouter()

	// Apply CORS middleware first
	r.Use(corsMiddleware)

	// API v1 routes
	v1 := r.PathPrefix("/api/v1").Subrouter()

	// Health endpoint using api-core/health for standardized response format
	// No database check since visited-tracker uses file-based storage
	r.HandleFunc("/health", healthHandler).Methods("GET")

	// Campaign management endpoints
	v1.HandleFunc("/campaigns", listCampaignsHandler).Methods("GET")
	v1.HandleFunc("/campaigns", createCampaignHandler).Methods("POST")
	v1.HandleFunc("/campaigns", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/campaigns/find-or-create", findOrCreateCampaignHandler).Methods("POST")
	v1.HandleFunc("/campaigns/find-or-create", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/campaigns/{id}", getCampaignHandler).Methods("GET")
	v1.HandleFunc("/campaigns/{id}", updateCampaignHandler).Methods("PATCH")
	v1.HandleFunc("/campaigns/{id}", deleteCampaignHandler).Methods("DELETE")
	v1.HandleFunc("/campaigns/{id}", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/campaigns/{id}/files/by-path", getFileByPathHandler).Methods("GET")
	v1.HandleFunc("/campaigns/{id}/files/exclude", bulkExcludeFilesHandler).Methods("POST")
	v1.HandleFunc("/campaigns/{id}/files/exclude", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/campaigns/{id}/files/{file_id}/notes", updateFileNotesHandler).Methods("PATCH")
	v1.HandleFunc("/campaigns/{id}/files/{file_id}/priority", updateFilePriorityHandler).Methods("PATCH")
	v1.HandleFunc("/campaigns/{id}/files/{file_id}/exclude", toggleFileExclusionHandler).Methods("PATCH")
	v1.HandleFunc("/campaigns/{id}/reset", resetCampaignHandler).Methods("POST")
	v1.HandleFunc("/campaigns/{id}/reset", optionsHandler).Methods("OPTIONS")

	// Campaign-specific visit tracking endpoints
	v1.HandleFunc("/campaigns/{id}/visit", visitHandler).Methods("POST")
	v1.HandleFunc("/campaigns/{id}/visit", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/campaigns/{id}/adjust-visit", adjustVisitHandler).Methods("POST")
	v1.HandleFunc("/campaigns/{id}/adjust-visit", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/campaigns/{id}/structure/sync", structureSyncHandler).Methods("POST")
	v1.HandleFunc("/campaigns/{id}/structure/sync", optionsHandler).Methods("OPTIONS")

	// Campaign prioritization and analytics endpoints
	v1.HandleFunc("/campaigns/{id}/prioritize/least-visited", leastVisitedHandler).Methods("GET")
	v1.HandleFunc("/campaigns/{id}/prioritize/most-stale", mostStaleHandler).Methods("GET")
	v1.HandleFunc("/campaigns/{id}/coverage", coverageHandler).Methods("GET")
	v1.HandleFunc("/campaigns/{id}/export", exportHandler).Methods("GET")
	v1.HandleFunc("/campaigns/import", importHandler).Methods("POST")
	v1.HandleFunc("/campaigns/import", optionsHandler).Methods("OPTIONS")

	logger.Printf("🚀 %s API v%s starting on port %s", serviceName, apiVersion, port)
	logger.Printf("📊 Endpoints available at http://localhost:%s/api/v1", port)
	logger.Printf("💾 Data stored in JSON files at: %s", filepath.Join("scenarios", "visited-tracker", dataDir))

	if err := http.ListenAndServe(":"+port, r); err != nil {
		logger.Fatalf("Server failed to start: %v", err)
	}
}

func resolveProjectRoot() (string, error) {
	if projectRoot := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); projectRoot != "" {
		return repocontract.FindRepoRootFromPath(projectRoot)
	}
	return repocontract.ResolveRepoRoot()
}

// corsMiddleware handles Cross-Origin Resource Sharing (CORS) for all requests
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get allowed origins from environment or use localhost defaults
		allowedOrigins := getAllowedOrigins()
		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		if isOriginAllowed(origin, allowedOrigins) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getAllowedOrigins returns the list of allowed CORS origins
func getAllowedOrigins() []string {
	// Check environment variable first
	if origins := os.Getenv("CORS_ALLOWED_ORIGINS"); origins != "" {
		return strings.Split(origins, ",")
	}

	// Default to localhost origins for development
	// In production, CORS_ALLOWED_ORIGINS should be explicitly set
	uiPort := os.Getenv("UI_PORT")
	if uiPort == "" {
		uiPort = "38440" // fallback to default
	}

	return []string{
		"http://localhost:" + uiPort,
		"http://127.0.0.1:" + uiPort,
	}
}

// isOriginAllowed checks if an origin is in the allowed list
func isOriginAllowed(origin string, allowed []string) bool {
	for _, o := range allowed {
		if o == origin {
			return true
		}
	}
	return false
}
