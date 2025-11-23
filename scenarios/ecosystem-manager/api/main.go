package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"

	"github.com/ecosystem-manager/api/pkg/autosteer"
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
	db           *sql.DB

	// Auto Steer components
	autoSteerProfileService  *autosteer.ProfileService
	autoSteerExecutionEngine *autosteer.ExecutionEngine
	autoSteerHistoryService  *autosteer.HistoryService
	autoSteerMetricsCollector *autosteer.MetricsCollector

	// Handlers
	taskHandlers      *handlers.TaskHandlers
	queueHandlers     *handlers.QueueHandlers
	discoveryHandlers *handlers.DiscoveryHandlers
	healthHandlers    *handlers.HealthHandlers
	settingsHandlers  *handlers.SettingsHandlers
	autoSteerHandlers *autosteer.AutoSteerHandlers
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

	// Initialize database connection
	if err := initializeDatabase(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

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

// initializeDatabase initializes the PostgreSQL database connection
func initializeDatabase() error {
	// Get database connection details from environment
	dbHost := os.Getenv("POSTGRES_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}

	dbPort := os.Getenv("POSTGRES_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}

	dbUser := os.Getenv("POSTGRES_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}

	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	if dbPassword == "" {
		dbPassword = "postgres"
	}

	dbName := os.Getenv("POSTGRES_DB")
	if dbName == "" {
		dbName = "ecosystem_manager"
	}

	// Build connection string
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName,
	)

	// Open database connection
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Println("✅ Database connection established")
	systemlog.Info("Database connection established")

	return nil
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
	if err := storage.ResyncStatuses(); err != nil {
		log.Printf("Warning: status resync encountered issues: %v", err)
		systemlog.Warnf("Status resync encountered issues: %v", err)
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

	// Initialize Auto Steer components
	projectRoot2 := filepath.Clean(filepath.Join(scenarioRoot, "..", ".."))

	// Verify Auto Steer database tables exist
	if err := autosteer.EnsureTablesExist(db); err != nil {
		log.Fatalf("Auto Steer database tables missing: %v", err)
	}

	// Log table counts for debugging
	counts, err := autosteer.GetTableCounts(db)
	if err != nil {
		log.Printf("Warning: Could not get table counts: %v", err)
	} else {
		log.Printf("Auto Steer table counts: profiles=%d, executions=%d, active_states=%d",
			counts["auto_steer_profiles"],
			counts["profile_executions"],
			counts["profile_execution_state"])
	}

	autoSteerMetricsCollector = autosteer.NewMetricsCollector(projectRoot2)
	autoSteerProfileService = autosteer.NewProfileService(db)
	autoSteerExecutionEngine = autosteer.NewExecutionEngine(db, autoSteerProfileService, autoSteerMetricsCollector)
	autoSteerHistoryService = autosteer.NewHistoryService(db)
	log.Println("✅ Auto Steer components initialized")
	systemlog.Info("Auto Steer components initialized")

	// Connect Auto Steer to queue processor
	autoSteerIntegration := queue.NewAutoSteerIntegration(autoSteerExecutionEngine)
	processor.SetAutoSteerIntegration(autoSteerIntegration)

	// Initialize handlers
	taskHandlers = handlers.NewTaskHandlers(storage, assembler, processor, wsManager)
	queueHandlers = handlers.NewQueueHandlers(processor, wsManager, storage)
	discoveryHandlers = handlers.NewDiscoveryHandlers(assembler)
	healthHandlers = handlers.NewHealthHandlers(processor)
	settingsHandlers = handlers.NewSettingsHandlers(processor, wsManager, taskRecycler)
	autoSteerHandlers = autosteer.NewAutoSteerHandlers(autoSteerProfileService, autoSteerExecutionEngine, autoSteerHistoryService)
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

	// Auto Steer routes
	// Profile management
	api.HandleFunc("/auto-steer/profiles", autoSteerHandlers.CreateProfile).Methods("POST")
	api.HandleFunc("/auto-steer/profiles", autoSteerHandlers.ListProfiles).Methods("GET")
	api.HandleFunc("/auto-steer/profiles/{id}", autoSteerHandlers.GetProfile).Methods("GET")
	api.HandleFunc("/auto-steer/profiles/{id}", autoSteerHandlers.UpdateProfile).Methods("PUT")
	api.HandleFunc("/auto-steer/profiles/{id}", autoSteerHandlers.DeleteProfile).Methods("DELETE")

	// Profile templates
	api.HandleFunc("/auto-steer/templates", autoSteerHandlers.GetTemplates).Methods("GET")

	// Execution management
	api.HandleFunc("/auto-steer/execution/start", autoSteerHandlers.StartExecution).Methods("POST")
	api.HandleFunc("/auto-steer/execution/evaluate", autoSteerHandlers.EvaluateIteration).Methods("POST")
	api.HandleFunc("/auto-steer/execution/advance", autoSteerHandlers.AdvancePhase).Methods("POST")
	api.HandleFunc("/auto-steer/execution/{taskId}", autoSteerHandlers.GetExecutionState).Methods("GET")

	// Metrics
	api.HandleFunc("/auto-steer/metrics/{taskId}", autoSteerHandlers.GetMetrics).Methods("GET")

	// Historical performance
	api.HandleFunc("/auto-steer/history", autoSteerHandlers.GetHistory).Methods("GET")
	api.HandleFunc("/auto-steer/history/{executionId}", autoSteerHandlers.GetExecution).Methods("GET")
	api.HandleFunc("/auto-steer/history/{executionId}/feedback", autoSteerHandlers.SubmitFeedback).Methods("POST")

	// Analytics
	api.HandleFunc("/auto-steer/analytics/{profileId}", autoSteerHandlers.GetProfileAnalytics).Methods("GET")

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
