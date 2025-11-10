package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/ecosystem-manager/api/pkg/handlers"
	"github.com/ecosystem-manager/api/pkg/prompts"
	"github.com/ecosystem-manager/api/pkg/queue"
	"github.com/ecosystem-manager/api/pkg/recycler"
	"github.com/ecosystem-manager/api/pkg/systemlog"
	"github.com/ecosystem-manager/api/pkg/tasks"
	"github.com/ecosystem-manager/api/pkg/websocket"
)

// LOGGING STRATEGY:
//
// This codebase uses TWO intentionally separate logging systems:
//
// 1. Standard library log.* (stdout/stderr) - Real-time Operational Logs
//    - Purpose: Immediate feedback for operators/developers watching the process
//    - Destination: stdout/stderr (captured by process managers, systemd, Docker, kubectl)
//    - Use for: Startup messages, progress indicators, debugging info, operational observability
//    - Volume: Verbose, ephemeral
//    - Examples: "🚀 Starting API...", "Processing task...", "Warning: retrying..."
//
// 2. Custom systemlog.* (file-based) - Historical Audit Trail
//    - Purpose: Persistent, structured logs for UI consumers and post-mortem analysis
//    - Destination: Date-stamped files in ../logs/ directory
//    - Served via: /api/logs HTTP endpoint (see pkg/handlers/logs.go)
//    - Use for: Significant business events, errors, state changes worth persisting
//    - Volume: Selective, permanent (only ~28% of log.* call volume)
//    - Severity levels: Debug, Info, Warn, Error
//    - Examples: Task state transitions, configuration changes, critical errors
//
// GUIDELINES:
// - Use log.* for operational observability (what's happening right now)
// - Use systemlog.* for audit trail (what happened that matters historically)
// - Avoid logging the same message to both systems (choose the right destination)
// - systemlog.* powers the UI log viewer; log.* powers real-time monitoring
//

var (
	// Core components
	storage      *tasks.Storage
	assembler    *prompts.Assembler
	processor    *queue.Processor
	wsManager    *websocket.Manager
	taskRecycler *recycler.Recycler

	// Handlers
	taskHandlers      *handlers.TaskHandlers
	queueHandlers     *handlers.QueueHandlers
	discoveryHandlers *handlers.DiscoveryHandlers
	healthHandlers    *handlers.HealthHandlers
	settingsHandlers  *handlers.SettingsHandlers
)

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start ecosystem-manager

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("🚀 Starting Ecosystem Manager API...")
	systemlog.Init()
	systemlog.Info("API startup initiated")

	// Initialize core components
	if err := initializeComponents(); err != nil {
		log.Fatalf("Failed to initialize components: %v", err)
	}

	// SAFETY FEATURE: Always start paused on API startup
	// The processor will be started when user enables it in UI
	// This ensures tasks don't execute automatically on startup
	log.Println("⚠️  Queue processor initialized but NOT started (safety feature)")
	log.Println("💡  Enable the processor in the UI settings to start processing tasks")

	// Set up HTTP server
	router := setupRoutes()

	// Get port from environment - fail if not set
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("API_PORT environment variable is required but not set")
	}

	// Validate port
	if _, err := strconv.Atoi(port); err != nil {
		log.Fatalf("Invalid API_PORT '%s': %v", port, err)
	}

	log.Printf("✅ Ecosystem Manager API starting on port %s", port)
	log.Printf("🔗 WebSocket endpoint: ws://localhost:%s/ws", port)
	log.Printf("🏥 Health check: http://localhost:%s/health", port)
	log.Printf("📋 Queue status: http://localhost:%s/api/queue/status", port)
	systemlog.Info(fmt.Sprintf("HTTP server listening on port %s", port))

	// Start server
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// initializeComponents initializes all core system components
func initializeComponents() error {
	// Determine directories relative to the executable
	// This ensures the paths work correctly regardless of where the binary is run from
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}
	execDir := filepath.Dir(execPath)

	// The ecosystem-manager structure has queue and prompts at the scenario root
	// The API binary is in scenarios/ecosystem-manager/api/
	scenarioRoot := filepath.Join(execDir, "..")
	projectRoot := filepath.Clean(filepath.Join(scenarioRoot, "..", ".."))
	queueDir := filepath.Join(scenarioRoot, "queue")
	promptsDir := filepath.Join(scenarioRoot, "prompts")

	// Ensure queue directories exist
	dirs := []string{"pending", "in-progress", "review", "completed", "failed", "completed-finalized", "failed-blocked", "archived"}
	for _, dir := range dirs {
		fullPath := filepath.Join(queueDir, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return err
		}
	}

	// Initialize storage
	storage = tasks.NewStorage(queueDir)
	log.Println("✅ Task storage initialized")
	systemlog.Info("Task storage initialized")
	if err := storage.CleanupDuplicates(); err != nil {
		log.Printf("Warning: duplicate task cleanup encountered issues: %v", err)
		systemlog.Warnf("Duplicate task cleanup encountered issues: %v", err)
	}

	// Initialize prompts assembler
	assembler, err = prompts.NewAssembler(promptsDir, projectRoot)
	if err != nil {
		return err
	}
	log.Println("✅ Prompt assembler initialized")
	systemlog.Info("Prompt assembler initialized")

	// Initialize WebSocket manager
	wsManager = websocket.NewManager()
	log.Println("✅ WebSocket manager initialized")
	systemlog.Info("WebSocket manager initialized")

	// Initialize recycler
	taskRecycler = recycler.New(storage, wsManager)
	taskRecycler.Start()
	log.Println("✅ Recycler daemon started")
	systemlog.Info("Recycler daemon started")

	// Initialize queue processor
	processor = queue.NewProcessor(
		30*time.Second, // 30-second processing interval
		storage,
		assembler,
		wsManager.GetBroadcastChannel(),
	)
	log.Println("✅ Queue processor initialized")
	systemlog.Info("Queue processor initialized")

	// Initialize handlers
	taskHandlers = handlers.NewTaskHandlers(storage, assembler, processor, wsManager)
	queueHandlers = handlers.NewQueueHandlers(processor, wsManager, storage)
	discoveryHandlers = handlers.NewDiscoveryHandlers(assembler)
	healthHandlers = handlers.NewHealthHandlers(processor)
	settingsHandlers = handlers.NewSettingsHandlers(processor, wsManager, taskRecycler)
	log.Println("✅ HTTP handlers initialized")

	return nil
}

// setupRoutes configures all HTTP routes
func setupRoutes() http.Handler {
	router := mux.NewRouter()

	// WebSocket endpoint
	router.HandleFunc("/ws", wsManager.HandleWebSocket)

	// Health check endpoint
	router.HandleFunc("/health", healthHandlers.HealthCheckHandler).Methods("GET")

	// Task management routes
	api := router.PathPrefix("/api").Subrouter()

	// Task CRUD operations
	api.HandleFunc("/tasks", taskHandlers.GetTasksHandler).Methods("GET")
	api.HandleFunc("/tasks", taskHandlers.CreateTaskHandler).Methods("POST")
	api.HandleFunc("/tasks/active-targets", taskHandlers.GetActiveTargetsHandler).Methods("GET")
	api.HandleFunc("/tasks/{id}", taskHandlers.GetTaskHandler).Methods("GET")
	api.HandleFunc("/tasks/{id}", taskHandlers.UpdateTaskHandler).Methods("PUT")
	api.HandleFunc("/tasks/{id}", taskHandlers.DeleteTaskHandler).Methods("DELETE")
	api.HandleFunc("/tasks/{id}/status", taskHandlers.UpdateTaskStatusHandler).Methods("PUT")
	api.HandleFunc("/tasks/{id}/logs", taskHandlers.GetTaskLogsHandler).Methods("GET")
	api.HandleFunc("/tasks/{id}/executions", taskHandlers.GetExecutionHistoryHandler).Methods("GET")
	api.HandleFunc("/tasks/{id}/executions/{execution_id}/prompt", taskHandlers.GetExecutionPromptHandler).Methods("GET")
	api.HandleFunc("/tasks/{id}/executions/{execution_id}/output", taskHandlers.GetExecutionOutputHandler).Methods("GET")
	api.HandleFunc("/tasks/{id}/executions/{execution_id}/metadata", taskHandlers.GetExecutionMetadataHandler).Methods("GET")

	// Global execution history (all tasks)
	api.HandleFunc("/executions", taskHandlers.GetAllExecutionHistoryHandler).Methods("GET")

	// Task prompt operations
	api.HandleFunc("/tasks/{id}/prompt", taskHandlers.GetTaskPromptHandler).Methods("GET")
	api.HandleFunc("/tasks/{id}/prompt/assembled", taskHandlers.GetAssembledPromptHandler).Methods("GET")

	// Prompt viewer (no task ID required)
	api.HandleFunc("/prompt-viewer", taskHandlers.PromptViewerHandler).Methods("POST")

	// Queue management routes
	api.HandleFunc("/queue/status", queueHandlers.GetQueueStatusHandler).Methods("GET")
	api.HandleFunc("/queue/resume-diagnostics", queueHandlers.GetResumeDiagnosticsHandler).Methods("GET")
	api.HandleFunc("/queue/trigger", queueHandlers.TriggerQueueProcessingHandler).Methods("POST")
	api.HandleFunc("/queue/reset-rate-limit", queueHandlers.ResetRateLimitHandler).Methods("POST")
	api.HandleFunc("/recycler/test", settingsHandlers.TestRecyclerHandler).Methods("POST")
	api.HandleFunc("/recycler/preview-prompt", settingsHandlers.PreviewRecyclerPromptHandler).Methods("POST")

	// System logs
	api.HandleFunc("/logs", handlers.LogsHandler).Methods("GET")

	// Process management (match original path)
	api.HandleFunc("/processes/running", queueHandlers.GetRunningProcessesHandler).Methods("GET")

	// Maintenance (match original path)
	api.HandleFunc("/maintenance/state", queueHandlers.SetMaintenanceStateHandler).Methods("POST")

	// Queue control endpoints (these are new/different from original)
	api.HandleFunc("/queue/stop", queueHandlers.StopQueueProcessorHandler).Methods("POST")
	api.HandleFunc("/queue/start", queueHandlers.StartQueueProcessorHandler).Methods("POST")
	api.HandleFunc("/queue/processes/terminate", queueHandlers.TerminateProcessHandler).Methods("POST")

	// Settings routes
	api.HandleFunc("/settings", settingsHandlers.GetSettingsHandler).Methods("GET")
	api.HandleFunc("/settings", settingsHandlers.UpdateSettingsHandler).Methods("PUT")
	api.HandleFunc("/settings/reset", settingsHandlers.ResetSettingsHandler).Methods("POST")
	api.HandleFunc("/settings/recycler/models", settingsHandlers.GetRecyclerModelsHandler).Methods("GET")

	// Discovery routes
	api.HandleFunc("/resources", discoveryHandlers.GetResourcesHandler).Methods("GET")
	api.HandleFunc("/scenarios", discoveryHandlers.GetScenariosHandler).Methods("GET")
	api.HandleFunc("/resources/{name}/status", discoveryHandlers.GetResourceStatusHandler).Methods("GET")
	api.HandleFunc("/scenarios/{name}/status", discoveryHandlers.GetScenarioStatusHandler).Methods("GET")
	api.HandleFunc("/operations", discoveryHandlers.GetOperationsHandler).Methods("GET")
	api.HandleFunc("/categories", discoveryHandlers.GetCategoriesHandler).Methods("GET")

	// Enable CORS for all routes
	corsHandler := gorillaHandlers.CORS(
		gorillaHandlers.AllowedOrigins([]string{"*"}), // Allow all origins for development
		gorillaHandlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		gorillaHandlers.AllowedHeaders([]string{"*"}),
	)(router)

	// Add request logging
	loggedHandler := gorillaHandlers.LoggingHandler(os.Stdout, corsHandler)

	log.Println("✅ HTTP routes configured")
	return loggedHandler
}
