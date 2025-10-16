package main

import (
	"fmt"
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

	// Issue count tracking for max_issues limit
	issuesProcessedMutex sync.RWMutex
	issuesProcessedCount int

	// Running process tracking
	runningProcessesMutex sync.RWMutex
	runningProcesses      map[string]*RunningProcess

	// WebSocket hub for real-time updates
	hub *Hub
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

	qdrantURL := strings.TrimSpace(os.Getenv("QDRANT_URL"))
	if qdrantURL == "" {
		logInfo("Qdrant URL not provided; semantic search disabled")
	}

	return &Config{
		Port:         getEnv("API_PORT", getEnv("PORT", "")),
		QdrantURL:    qdrantURL,
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
	if root, ok := os.LookupEnv("VROOLI_ROOT"); ok {
		trimmed := strings.TrimSpace(root)
		if trimmed == "" {
			logError("VROOLI_ROOT environment variable is set but empty")
			os.Exit(1)
		}
		return trimmed
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
	lifecycleManaged, ok := os.LookupEnv("VROOLI_LIFECYCLE_MANAGED")
	if !ok || lifecycleManaged != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start app-issue-tracker

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	config := loadConfig()

	// Load agent settings (ecosystem-manager pattern)
	_ = LoadAgentSettings(config.ScenarioRoot)
	logInfo("Agent settings loaded", "scenario_root", config.ScenarioRoot)

	// Ensure issues directory structure exists
	folders := []string{"open", "active", "waiting", "completed", "failed", "templates"}
	for _, folder := range folders {
		folderPath := filepath.Join(config.IssuesDir, folder)
		if err := os.MkdirAll(folderPath, 0o755); err != nil {
			logErrorErr("Failed to ensure issue folder exists", err, "folder", folderPath)
			os.Exit(1)
		}
	}

	logInfo("Using file-based storage", "issues_dir", config.IssuesDir)

	// Initialize WebSocket hub
	hub := NewHub()
	go hub.Run()

	server := &Server{
		config:           config,
		runningProcesses: make(map[string]*RunningProcess),
		hub:              hub,
	}

	// Setup routes
	r := mux.NewRouter()

	// Health check
	r.HandleFunc("/health", server.healthHandler).Methods("GET")

	// API routes (v1)
	v1 := r.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/ws", server.handleWebSocket).Methods("GET")
	v1.HandleFunc("/issues", server.getIssuesHandler).Methods("GET")
	v1.HandleFunc("/issues", server.createIssueHandler).Methods("POST")
	v1.HandleFunc("/issues/{id}/attachments/{attachment:.*}", server.getIssueAttachmentHandler).Methods("GET")
	v1.HandleFunc("/issues/{id}", server.getIssueHandler).Methods("GET")
	v1.HandleFunc("/issues/{id}", server.updateIssueHandler).Methods("PUT", "PATCH")
	v1.HandleFunc("/issues/{id}", server.deleteIssueHandler).Methods("DELETE")
	v1.HandleFunc("/issues/{id}/agent/conversation", server.getIssueAgentConversationHandler).Methods("GET")
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
	v1.HandleFunc("/automation/processor/reset-counter", server.resetIssueCounterHandler).Methods("POST")
	v1.HandleFunc("/rate-limit-status", server.getRateLimitStatusHandler).Methods("GET")
	v1.HandleFunc("/processes/running", server.getRunningProcessesHandler).Methods("GET")

	// Initialize processor state and start background loop
	server.initializeProcessorState()
	go server.runProcessorLoop()

	// Apply CORS middleware
	handler := corsMiddleware(r)

	logInfo("Starting App Issue Tracker API", "port", config.Port)
	logInfo("API health endpoint configured", "url", fmt.Sprintf("http://localhost:%s/health", config.Port))
	logInfo("API base URL", "url", fmt.Sprintf("http://localhost:%s/api/v1", config.Port))
	logInfo("Scenario configuration", "issues_dir", config.IssuesDir, "scenario_root", config.ScenarioRoot)
	logInfo("Processor loop state", "active", false)

	if err := http.ListenAndServe(":"+config.Port, handler); err != nil {
		logErrorErr("Server failed to start", err, "port", config.Port)
		os.Exit(1)
	}
}
