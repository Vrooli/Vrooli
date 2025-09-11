package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

const (
	apiVersion  = "1.0.0"
	serviceName = "maintenance-orchestrator"
)

var (
	logger     *log.Logger
	startTime  = time.Now()
)

func main() {
	// Protect against direct execution - must be run through lifecycle system
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start maintenance-orchestrator

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Change working directory to project root for scenario discovery
	// API runs from scenarios/maintenance-orchestrator/api/, so go up 3 levels to project root
	if err := os.Chdir("../../../"); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to change to project root directory: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger = log.New(os.Stdout, "[maintenance-orchestrator] ", log.LstdFlags)

	// Log current working directory for transparency
	if cwd, err := os.Getwd(); err == nil {
		logger.Printf("📁 Working directory: %s", cwd)
		logger.Printf("🔍 Scenario discovery will be performed relative to this directory")
	}

	// Get port from environment (set by lifecycle system) - NO FALLBACK VALUES
	port := os.Getenv("API_PORT")
	if port == "" {
		logger.Fatal("❌ API_PORT environment variable is required")
	}

	// Initialize orchestrator
	orchestrator := NewOrchestrator()
	initializeDefaultPresets(orchestrator)

	// Start discovery goroutine
	go func() {
		for {
			discoverScenarios(orchestrator, logger)
			time.Sleep(60 * time.Second)
		}
	}()

	// Setup router
	r := mux.NewRouter()
	
	// Apply CORS middleware first
	r.Use(corsMiddleware)
	
	// Health endpoint (outside versioning for simplicity)
	r.HandleFunc("/health", healthHandler(startTime)).Methods("GET")

	// API v1 routes
	v1 := r.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/scenarios", handleGetScenarios(orchestrator)).Methods("GET")
	v1.HandleFunc("/scenarios/{id}/activate", handleActivateScenario(orchestrator)).Methods("POST")
	v1.HandleFunc("/scenarios/{id}/deactivate", handleDeactivateScenario(orchestrator)).Methods("POST")
	v1.HandleFunc("/presets", handleGetPresets(orchestrator)).Methods("GET")
	v1.HandleFunc("/presets/{id}/apply", handleApplyPreset(orchestrator)).Methods("POST")
	v1.HandleFunc("/status", handleGetStatus(orchestrator, startTime)).Methods("GET")
	v1.HandleFunc("/stop-all", handleStopAll(orchestrator)).Methods("POST")

	// Options handlers for CORS
	v1.HandleFunc("/scenarios", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/scenarios/{id}/activate", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/scenarios/{id}/deactivate", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/presets", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/presets/{id}/apply", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/status", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/stop-all", optionsHandler).Methods("OPTIONS")

	logger.Printf("🚀 %s API v%s starting on port %s", serviceName, apiVersion, port)
	logger.Printf("📊 Endpoints available at http://localhost:%s/api/v1", port)
	logger.Printf("❤️ Health check at http://localhost:%s/health", port)

	if err := http.ListenAndServe(":"+port, r); err != nil {
		logger.Fatalf("Server failed to start: %v", err)
	}
}