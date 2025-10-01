package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gorilla/mux"
)

// Server holds the application state
type Server struct {
	config         *Config
	processorState ProcessorState
	processorMutex sync.RWMutex
}

// loadConfig loads application configuration from environment variables
func loadConfig() *Config {
	vrooliRoot := getVrooliRoot()
	scenarioRoot := filepath.Join(vrooliRoot, "scenarios/app-issue-tracker")

	// Default to the actual scenario issues directory if not specified
	defaultIssuesDir := filepath.Join(scenarioRoot, "data/issues")
	if _, err := os.Stat("./data/issues"); err == nil {
		// If local issues directory exists, use it
		defaultIssuesDir = "./data/issues"
	}

	return &Config{
		Port:         getEnv("API_PORT", getEnv("PORT", "")),
		QdrantURL:    getEnv("QDRANT_URL", "http://localhost:6333"),
		IssuesDir:    getEnv("ISSUES_DIR", defaultIssuesDir),
		ScenarioRoot: scenarioRoot,
	}
}

// getEnv retrieves an environment variable with a default fallback
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getVrooliRoot determines the Vrooli project root directory
func getVrooliRoot() string {
	if root := os.Getenv("VROOLI_ROOT"); root != "" {
		return root
	}
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	if out, err := cmd.Output(); err == nil {
		return strings.TrimSpace(string(out))
	}
	ex, _ := os.Executable()
	return filepath.Dir(filepath.Dir(ex))
}

// corsMiddleware adds CORS headers to all responses
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
	// Protect against direct execution - must be run through lifecycle system
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start app-issue-tracker

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	config := loadConfig()

	// Ensure issues directory structure exists
	folders := []string{"open", "active", "waiting", "completed", "failed", "templates"}
	for _, folder := range folders {
		folderPath := filepath.Join(config.IssuesDir, folder)
		if err := os.MkdirAll(folderPath, 0755); err != nil {
			log.Fatalf("Failed to create folder %s: %v", folder, err)
		}
	}

	log.Printf("Using file-based storage at: %s", config.IssuesDir)

	server := &Server{config: config}

	// Setup routes
	r := mux.NewRouter()

	// Health check
	r.HandleFunc("/health", server.healthHandler).Methods("GET")

	// API routes (v1)
	v1 := r.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/issues", server.getIssuesHandler).Methods("GET")
	v1.HandleFunc("/issues", server.createIssueHandler).Methods("POST")
	v1.HandleFunc("/issues/{id}/attachments/{attachment:.*}", server.getIssueAttachmentHandler).Methods("GET")
	v1.HandleFunc("/issues/{id}", server.getIssueHandler).Methods("GET")
	v1.HandleFunc("/issues/{id}", server.updateIssueHandler).Methods("PUT", "PATCH")
	v1.HandleFunc("/issues/{id}", server.deleteIssueHandler).Methods("DELETE")
	v1.HandleFunc("/issues/search", server.searchIssuesHandler).Methods("GET")
	// v1.HandleFunc("/issues/search/similar", server.vectorSearchHandler).Methods("POST") // TODO: Implement vector search
	v1.HandleFunc("/agents", server.getAgentsHandler).Methods("GET")
	v1.HandleFunc("/agent/settings", server.getAgentSettingsHandler).Methods("GET")
	v1.HandleFunc("/agent/settings", server.updateAgentSettingsHandler).Methods("PATCH")
	v1.HandleFunc("/apps", server.getAppsHandler).Methods("GET")
	v1.HandleFunc("/investigate", server.triggerInvestigationHandler).Methods("POST")
	v1.HandleFunc("/investigate/preview", server.previewInvestigationPromptHandler).Methods("POST")
	v1.HandleFunc("/generate-fix", server.triggerFixGenerationHandler).Methods("POST")
	v1.HandleFunc("/stats", server.getStatsHandler).Methods("GET")
	v1.HandleFunc("/export", server.exportIssuesHandler).Methods("GET")
	v1.HandleFunc("/automation/processor", server.getProcessorHandler).Methods("GET")
	v1.HandleFunc("/automation/processor", server.updateProcessorHandler).Methods("PATCH")
	v1.HandleFunc("/rate-limit-status", server.getRateLimitStatusHandler).Methods("GET")

	// Initialize processor state and start background loop
	server.initializeProcessorState()
	go server.runProcessorLoop()

	// Apply CORS middleware
	handler := corsMiddleware(r)

	log.Printf("Starting File-Based App Issue Tracker API on port %s", config.Port)
	log.Printf("Health check: http://localhost:%s/health", config.Port)
	log.Printf("API base URL: http://localhost:%s/api/v1", config.Port)
	log.Printf("Issues directory: %s", config.IssuesDir)
	log.Printf("Scenario root: %s", config.ScenarioRoot)
	log.Printf("Processor loop: Running (inactive by default)")

	if err := http.ListenAndServe(":"+config.Port, handler); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
