package server

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"

	"github.com/ecosystem-manager/api/pkg/autosteer"
	"github.com/ecosystem-manager/api/pkg/handlers"
	"github.com/ecosystem-manager/api/pkg/prompts"
	"github.com/ecosystem-manager/api/pkg/queue"
	"github.com/ecosystem-manager/api/pkg/recycler"
	"github.com/ecosystem-manager/api/pkg/settings"
	"github.com/ecosystem-manager/api/pkg/systemlog"
	"github.com/ecosystem-manager/api/pkg/tasks"
	"github.com/ecosystem-manager/api/pkg/websocket"
)

type Config struct {
	Port           string
	AllowedOrigins []string
}

type Application struct {
	// Core components
	storage      *tasks.Storage
	assembler    *prompts.Assembler
	processor    *queue.Processor
	wsManager    *websocket.Manager
	taskRecycler *recycler.Recycler
	db           *sql.DB

	// Auto Steer components
	autoSteerProfileService   *autosteer.ProfileService
	autoSteerExecutionEngine  *autosteer.ExecutionEngine
	autoSteerHistoryService   *autosteer.HistoryService
	autoSteerMetricsCollector *autosteer.MetricsCollector

	// Handlers
	taskHandlers      *handlers.TaskHandlers
	queueHandlers     *handlers.QueueHandlers
	discoveryHandlers *handlers.DiscoveryHandlers
	healthHandlers    *handlers.HealthHandlers
	settingsHandlers  *handlers.SettingsHandlers
	promptsHandlers   *handlers.PromptsHandlers
	autoSteerHandlers *autosteer.AutoSteerHandlers

	// Paths
	scenarioRoot string
	projectRoot  string

	// Server settings
	port           string
	allowedOrigins []string

	shutdownOnce sync.Once
}

// New builds an Application with initialized dependencies.
func New(cfg Config) (*Application, error) {
	if cfg.Port == "" {
		return nil, fmt.Errorf("API_PORT is required")
	}
	if _, err := strconv.Atoi(cfg.Port); err != nil {
		return nil, fmt.Errorf("invalid API_PORT '%s': %w", cfg.Port, err)
	}
	if len(cfg.AllowedOrigins) == 0 {
		cfg.AllowedOrigins = []string{"*"}
	}

	scenarioRoot, projectRoot, err := resolvePaths()
	if err != nil {
		return nil, err
	}

	systemlog.InitWithBaseDir(scenarioRoot)
	systemlog.Info("API startup initiated")

	app := &Application{
		scenarioRoot:   scenarioRoot,
		projectRoot:    projectRoot,
		port:           cfg.Port,
		allowedOrigins: cfg.AllowedOrigins,
		shutdownOnce:   sync.Once{},
	}

	if err := app.initializeDatabase(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	if err := app.initializeComponents(); err != nil {
		app.shutdown()
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}

	// SAFETY FEATURE: Always start paused on API startup
	// The processor will be started when user enables it in UI
	log.Println("⚠️  Queue processor initialized but NOT started (safety feature)")
	log.Println("💡  Enable the processor in the UI settings to start processing tasks")

	return app, nil
}

// Run starts the HTTP server and shuts it down when the context is cancelled.
func (a *Application) Run(ctx context.Context) error {
	router := a.setupRoutes()

	log.Printf("✅ Ecosystem Manager API starting on port %s", a.port)
	log.Printf("🔗 WebSocket endpoint: ws://localhost:%s/ws", a.port)
	log.Printf("🏥 Health check: http://localhost:%s/health", a.port)
	log.Printf("📋 Queue status: http://localhost:%s/api/queue/status", a.port)
	systemlog.Info(fmt.Sprintf("HTTP server listening on port %s", a.port))

	server := &http.Server{
		Addr:         ":" + a.port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			a.shutdown()
			return fmt.Errorf("server failed to start: %w", err)
		}
	case <-ctx.Done():
		log.Println("Shutdown signal received, stopping server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil && err != http.ErrServerClosed {
			a.shutdown()
			return fmt.Errorf("server shutdown failed: %w", err)
		}

		// Wait for the server goroutine to exit after shutdown
		<-errCh
	}

	a.shutdown()
	return nil
}

func resolvePaths() (string, string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", "", fmt.Errorf("failed to get executable path: %v", err)
	}
	execDir := filepath.Dir(execPath)

	scenarioRoot := filepath.Join(execDir, "..")
	projectRoot := filepath.Clean(filepath.Join(scenarioRoot, "..", ".."))

	return scenarioRoot, projectRoot, nil
}

// initializeDatabase initializes the PostgreSQL database connection
func (a *Application) initializeDatabase() error {
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
	a.db, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := a.db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	a.db.SetMaxOpenConns(25)
	a.db.SetMaxIdleConns(5)
	a.db.SetConnMaxLifetime(5 * time.Minute)

	log.Println("✅ Database connection established")
	systemlog.Info("Database connection established")

	return nil
}

// initializeComponents initializes all core system components
func (a *Application) initializeComponents() error {
	queueDir := filepath.Join(a.scenarioRoot, "queue")
	promptsDir := filepath.Join(a.scenarioRoot, "prompts")

	// Ensure queue directories exist (aligned with valid queue statuses)
	for _, dir := range tasks.GetValidStatuses() {
		fullPath := filepath.Join(queueDir, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return err
		}
	}

	settings.SetPersistencePath(filepath.Join(a.scenarioRoot, "config", "settings.json"))
	if err := settings.LoadFromDisk(); err != nil {
		log.Printf("Warning: could not load persisted settings, using defaults: %v", err)
	}

	// Initialize storage
	a.storage = tasks.NewStorage(queueDir)
	log.Println("✅ Task storage initialized")
	systemlog.Info("Task storage initialized")
	if err := a.storage.CleanupDuplicates(); err != nil {
		log.Printf("Warning: duplicate task cleanup encountered issues: %v", err)
		systemlog.Warnf("Duplicate task cleanup encountered issues: %v", err)
	}
	if err := a.storage.ResyncStatuses(); err != nil {
		log.Printf("Warning: status resync encountered issues: %v", err)
		systemlog.Warnf("Status resync encountered issues: %v", err)
	}

	// Initialize prompts assembler
	var err error
	a.assembler, err = prompts.NewAssembler(promptsDir, a.projectRoot)
	if err != nil {
		return err
	}
	log.Println("✅ Prompt assembler initialized")
	systemlog.Info("Prompt assembler initialized")

	// Initialize WebSocket manager
	a.wsManager = websocket.NewManager()
	log.Println("✅ WebSocket manager initialized")
	systemlog.Info("WebSocket manager initialized")

	// Initialize recycler
	a.taskRecycler = recycler.New(a.storage, a.wsManager)
	a.taskRecycler.Start()
	log.Println("✅ Recycler daemon started")
	systemlog.Info("Recycler daemon started")

	// Initialize queue processor
	a.processor = queue.NewProcessor(
		a.storage,
		a.assembler,
		a.wsManager.GetBroadcastChannel(),
		a.taskRecycler,
	)
	a.taskRecycler.SetWakeFunc(a.processor.Wake)
	log.Println("✅ Queue processor initialized")
	systemlog.Info("Queue processor initialized")

	// Initialize Auto Steer components
	if err := autosteer.EnsureTablesExist(a.db); err != nil {
		return fmt.Errorf("auto steer database tables missing: %w", err)
	}

	// Log table counts for debugging
	counts, err := autosteer.GetTableCounts(a.db)
	if err != nil {
		log.Printf("Warning: Could not get table counts: %v", err)
	} else {
		log.Printf("Auto Steer table counts: profiles=%d, executions=%d, active_states=%d",
			counts["auto_steer_profiles"],
			counts["profile_executions"],
			counts["profile_execution_state"])
	}

	a.autoSteerMetricsCollector = autosteer.NewMetricsCollector(a.projectRoot)
	a.autoSteerProfileService = autosteer.NewProfileService(a.db)
	a.autoSteerExecutionEngine = autosteer.NewExecutionEngine(
		a.db,
		a.autoSteerProfileService,
		a.autoSteerMetricsCollector,
		filepath.Join(promptsDir, "phases"),
	)
	a.autoSteerHistoryService = autosteer.NewHistoryService(a.db)
	log.Println("✅ Auto Steer components initialized")
	systemlog.Info("Auto Steer components initialized")

	// Connect Auto Steer to queue processor
	autoSteerIntegration := queue.NewAutoSteerIntegration(a.autoSteerExecutionEngine)
	a.processor.SetAutoSteerIntegration(autoSteerIntegration)

	// Initialize handlers
	a.taskHandlers = handlers.NewTaskHandlers(a.storage, a.assembler, a.processor, a.wsManager, a.autoSteerProfileService)
	a.queueHandlers = handlers.NewQueueHandlers(a.processor, a.wsManager, a.storage)
	a.discoveryHandlers = handlers.NewDiscoveryHandlers(a.assembler)
	a.healthHandlers = handlers.NewHealthHandlers(a.processor, a.taskRecycler)
	a.settingsHandlers = handlers.NewSettingsHandlers(a.processor, a.wsManager, a.taskRecycler)
	a.promptsHandlers = handlers.NewPromptsHandlers(a.assembler)
	a.autoSteerHandlers = autosteer.NewAutoSteerHandlers(a.autoSteerProfileService, a.autoSteerExecutionEngine, a.autoSteerHistoryService)
	log.Println("✅ HTTP handlers initialized")

	return nil
}

// setupRoutes configures all HTTP routes
func (a *Application) setupRoutes() http.Handler {
	router := mux.NewRouter()

	// WebSocket endpoint
	router.HandleFunc("/ws", a.wsManager.HandleWebSocket)

	// Health check endpoint
	router.HandleFunc("/health", a.healthHandlers.HealthCheckHandler).Methods("GET")

	// Task management routes
	api := router.PathPrefix("/api").Subrouter()

	// Task CRUD operations
	api.HandleFunc("/tasks", a.taskHandlers.GetTasksHandler).Methods("GET")
	api.HandleFunc("/tasks", a.taskHandlers.CreateTaskHandler).Methods("POST")
	api.HandleFunc("/tasks/active-targets", a.taskHandlers.GetActiveTargetsHandler).Methods("GET")
	api.HandleFunc("/tasks/{id}", a.taskHandlers.GetTaskHandler).Methods("GET")
	api.HandleFunc("/tasks/{id}", a.taskHandlers.UpdateTaskHandler).Methods("PUT")
	api.HandleFunc("/tasks/{id}", a.taskHandlers.DeleteTaskHandler).Methods("DELETE")
	api.HandleFunc("/tasks/{id}/status", a.taskHandlers.UpdateTaskStatusHandler).Methods("PUT")
	api.HandleFunc("/tasks/{id}/logs", a.taskHandlers.GetTaskLogsHandler).Methods("GET")
	api.HandleFunc("/tasks/{id}/executions", a.taskHandlers.GetExecutionHistoryHandler).Methods("GET")
	api.HandleFunc("/tasks/{id}/executions/{execution_id}/prompt", a.taskHandlers.GetExecutionPromptHandler).Methods("GET")
	api.HandleFunc("/tasks/{id}/executions/{execution_id}/output", a.taskHandlers.GetExecutionOutputHandler).Methods("GET")
	api.HandleFunc("/tasks/{id}/executions/{execution_id}/metadata", a.taskHandlers.GetExecutionMetadataHandler).Methods("GET")

	// Global execution history (all tasks)
	api.HandleFunc("/executions", a.taskHandlers.GetAllExecutionHistoryHandler).Methods("GET")

	// Task prompt operations
	api.HandleFunc("/tasks/{id}/prompt", a.taskHandlers.GetTaskPromptHandler).Methods("GET")
	api.HandleFunc("/tasks/{id}/prompt/assembled", a.taskHandlers.GetAssembledPromptHandler).Methods("GET")

	// Prompt viewer (no task ID required)
	api.HandleFunc("/prompt-viewer", a.taskHandlers.PromptViewerHandler).Methods("POST")
	api.HandleFunc("/prompts", a.promptsHandlers.ListPromptFilesHandler).Methods("GET")
	api.HandleFunc("/prompts/{path:.*}", a.promptsHandlers.GetPromptFileHandler).Methods("GET")
	api.HandleFunc("/prompts/{path:.*}", a.promptsHandlers.UpdatePromptFileHandler).Methods("PUT")

	// Queue management routes
	api.HandleFunc("/queue/status", a.queueHandlers.GetQueueStatusHandler).Methods("GET")
	api.HandleFunc("/queue/resume-diagnostics", a.queueHandlers.GetResumeDiagnosticsHandler).Methods("GET")
	api.HandleFunc("/queue/trigger", a.queueHandlers.TriggerQueueProcessingHandler).Methods("POST")
	api.HandleFunc("/queue/reset-rate-limit", a.queueHandlers.ResetRateLimitHandler).Methods("POST")
	// System logs
	api.HandleFunc("/logs", handlers.LogsHandler).Methods("GET")

	// Process management (match original path)
	api.HandleFunc("/processes/running", a.queueHandlers.GetRunningProcessesHandler).Methods("GET")

	// Maintenance (match original path)
	api.HandleFunc("/maintenance/state", a.queueHandlers.SetMaintenanceStateHandler).Methods("POST")

	// Queue control endpoints (these are new/different from original)
	api.HandleFunc("/queue/stop", a.queueHandlers.StopQueueProcessorHandler).Methods("POST")
	api.HandleFunc("/queue/start", a.queueHandlers.StartQueueProcessorHandler).Methods("POST")
	api.HandleFunc("/queue/processes/terminate", a.queueHandlers.TerminateProcessHandler).Methods("POST")

	// Settings routes
	api.HandleFunc("/settings", a.settingsHandlers.GetSettingsHandler).Methods("GET")
	api.HandleFunc("/settings", a.settingsHandlers.UpdateSettingsHandler).Methods("PUT")
	api.HandleFunc("/settings/reset", a.settingsHandlers.ResetSettingsHandler).Methods("POST")
	api.HandleFunc("/settings/recycler/models", a.settingsHandlers.GetRecyclerModelsHandler).Methods("GET")

	// Discovery routes
	api.HandleFunc("/resources", a.discoveryHandlers.GetResourcesHandler).Methods("GET")
	api.HandleFunc("/scenarios", a.discoveryHandlers.GetScenariosHandler).Methods("GET")
	api.HandleFunc("/resources/{name}/status", a.discoveryHandlers.GetResourceStatusHandler).Methods("GET")
	api.HandleFunc("/scenarios/{name}/status", a.discoveryHandlers.GetScenarioStatusHandler).Methods("GET")
	api.HandleFunc("/operations", a.discoveryHandlers.GetOperationsHandler).Methods("GET")
	api.HandleFunc("/categories", a.discoveryHandlers.GetCategoriesHandler).Methods("GET")

	// Auto Steer routes
	// Profile management
	api.HandleFunc("/auto-steer/profiles", a.autoSteerHandlers.CreateProfile).Methods("POST")
	api.HandleFunc("/auto-steer/profiles", a.autoSteerHandlers.ListProfiles).Methods("GET")
	api.HandleFunc("/auto-steer/profiles/{id}", a.autoSteerHandlers.GetProfile).Methods("GET")
	api.HandleFunc("/auto-steer/profiles/{id}", a.autoSteerHandlers.UpdateProfile).Methods("PUT")
	api.HandleFunc("/auto-steer/profiles/{id}", a.autoSteerHandlers.DeleteProfile).Methods("DELETE")

	// Profile templates
	api.HandleFunc("/auto-steer/templates", a.autoSteerHandlers.GetTemplates).Methods("GET")

	// Execution management
	api.HandleFunc("/auto-steer/execution/start", a.autoSteerHandlers.StartExecution).Methods("POST")
	api.HandleFunc("/auto-steer/execution/evaluate", a.autoSteerHandlers.EvaluateIteration).Methods("POST")
	api.HandleFunc("/auto-steer/execution/reset", a.autoSteerHandlers.ResetExecution).Methods("POST")
	api.HandleFunc("/auto-steer/execution/advance", a.autoSteerHandlers.AdvancePhase).Methods("POST")
	api.HandleFunc("/auto-steer/execution/seek", a.autoSteerHandlers.SeekExecution).Methods("POST")
	api.HandleFunc("/auto-steer/execution/{taskId}", a.autoSteerHandlers.GetExecutionState).Methods("GET")

	// Metrics
	api.HandleFunc("/auto-steer/metrics/{taskId}", a.autoSteerHandlers.GetMetrics).Methods("GET")

	// Historical performance
	api.HandleFunc("/auto-steer/history", a.autoSteerHandlers.GetHistory).Methods("GET")
	api.HandleFunc("/auto-steer/history/{executionId}", a.autoSteerHandlers.GetExecution).Methods("GET")
	api.HandleFunc("/auto-steer/history/{executionId}/feedback", a.autoSteerHandlers.SubmitFeedback).Methods("POST")

	// Analytics
	api.HandleFunc("/auto-steer/analytics/{profileId}", a.autoSteerHandlers.GetProfileAnalytics).Methods("GET")

	origins := a.allowedOrigins
	if len(origins) == 0 {
		origins = []string{"*"}
	}

	// Enable CORS for all routes
	corsHandler := gorillaHandlers.CORS(
		gorillaHandlers.AllowedOrigins(origins),
		gorillaHandlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		gorillaHandlers.AllowedHeaders([]string{"*"}),
	)(router)

	// Add request logging
	loggedHandler := gorillaHandlers.LoggingHandler(os.Stdout, corsHandler)

	log.Println("✅ HTTP routes configured")
	return loggedHandler
}

func (a *Application) shutdown() {
	a.shutdownOnce.Do(func() {
		if a.processor != nil {
			a.processor.Stop()
			a.processor.Shutdown()
		}
		if a.taskRecycler != nil {
			a.taskRecycler.Stop()
		}
		if a.db != nil {
			if err := a.db.Close(); err != nil {
				log.Printf("Warning: failed to close database: %v", err)
			}
		}
		systemlog.Close()
	})
}
