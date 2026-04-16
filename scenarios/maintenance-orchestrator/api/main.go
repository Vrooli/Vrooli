package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	repocontract "github.com/vrooli/repo-contract-go"
)

const (
	apiVersion  = "1.0.0"
	serviceName = "maintenance-orchestrator"
)

var (
	logger    *log.Logger
	startTime = time.Now()
)

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "maintenance-orchestrator",
	}) {
		return // Process was re-exec'd after rebuild
	}

	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to resolve project root directory: %v\n", err)
		os.Exit(1)
	}

	if err := os.Chdir(root); err != nil {
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

	// Perform initial discovery before starting server
	logger.Printf("🔍 Performing initial scenario discovery...")
	discoverScenarios(orchestrator, logger)
	logger.Printf("✅ Initial discovery complete: %d scenarios found", len(orchestrator.GetScenarios()))

	// Start periodic discovery goroutine
	go func() {
		for {
			time.Sleep(60 * time.Second)
			discoverScenarios(orchestrator, logger)
		}
	}()

	// Start resource monitoring goroutine for active scenarios
	go func() {
		for {
			time.Sleep(30 * time.Second)
			scenarios := orchestrator.GetScenarios()
			for _, scenario := range scenarios {
				if scenario.IsActive && scenario.Port > 0 {
					usage := getScenarioResourceUsage(scenario.Name, scenario.Port)
					if usage != nil {
						orchestrator.UpdateResourceUsage(scenario.ID, usage)
					}
				}
			}
		}
	}()

	// Setup router
	r := mux.NewRouter()

	// Apply CORS middleware first
	r.Use(corsMiddleware)

	// Health endpoint (outside versioning for simplicity)
	r.HandleFunc("/health", health.Handler()).Methods("GET")

	// API v1 routes
	v1 := r.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/scenarios", handleGetScenarios(orchestrator)).Methods("GET")
	v1.HandleFunc("/scenarios/{id}/activate", handleActivateScenario(orchestrator)).Methods("POST")
	v1.HandleFunc("/scenarios/{id}/deactivate", handleDeactivateScenario(orchestrator)).Methods("POST")
	v1.HandleFunc("/presets", handleGetPresets(orchestrator)).Methods("GET")
	v1.HandleFunc("/presets", handleCreatePreset(orchestrator)).Methods("POST")
	v1.HandleFunc("/presets/active", handleGetActivePresets(orchestrator)).Methods("GET")
	v1.HandleFunc("/presets/{id}/apply", handleApplyPreset(orchestrator)).Methods("POST")
	v1.HandleFunc("/status", handleGetStatus(orchestrator, startTime)).Methods("GET")
	v1.HandleFunc("/stop-all", handleStopAll(orchestrator)).Methods("POST")
	v1.HandleFunc("/scenario-statuses", handleGetScenarioStatuses()).Methods("GET")
	v1.HandleFunc("/all-scenarios", handleListAllScenarios()).Methods("GET")
	v1.HandleFunc("/scenarios/{name}/add-tag", handleAddMaintenanceTag()).Methods("POST")
	v1.HandleFunc("/scenarios/{name}/remove-tag", handleRemoveMaintenanceTag()).Methods("POST")
	v1.HandleFunc("/scenarios/{name}/preset-assignments", handleGetScenarioPresetAssignments(orchestrator)).Methods("GET")
	v1.HandleFunc("/scenarios/{name}/preset-assignments", handleUpdateScenarioPresetAssignments(orchestrator)).Methods("POST")
	v1.HandleFunc("/scenarios/{name}/port", handleGetScenarioPort()).Methods("GET")
	v1.HandleFunc("/scenarios/{id}/start", handleStartScenario()).Methods("POST")
	v1.HandleFunc("/scenarios/{id}/stop", handleStopScenario()).Methods("POST")

	// Options handlers for CORS
	v1.HandleFunc("/scenarios", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/scenarios/{id}/activate", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/scenarios/{id}/deactivate", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/presets", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/presets/active", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/presets/{id}/apply", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/status", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/stop-all", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/scenario-statuses", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/all-scenarios", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/scenarios/{name}/add-tag", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/scenarios/{name}/remove-tag", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/scenarios/{name}/preset-assignments", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/scenarios/{id}/start", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/scenarios/{id}/stop", optionsHandler).Methods("OPTIONS")

	logger.Printf("🚀 %s API v%s starting on port %s", serviceName, apiVersion, port)
	logger.Printf("📊 Endpoints available at http://localhost:%s/api/v1", port)
	logger.Printf("❤️ Health check at http://localhost:%s/health", port)

	if err := http.ListenAndServe(":"+port, r); err != nil {
		logger.Fatalf("Server failed to start: %v", err)
	}
}
