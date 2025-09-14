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
	"github.com/ecosystem-manager/api/pkg/tasks"
	"github.com/ecosystem-manager/api/pkg/websocket"
)

var (
	// Core components
	storage   *tasks.Storage
	assembler *prompts.Assembler
	processor *queue.Processor
	wsManager *websocket.Manager

	// Handlers
	taskHandlers      *handlers.TaskHandlers
	queueHandlers     *handlers.QueueHandlers
	discoveryHandlers *handlers.DiscoveryHandlers
	healthHandlers    *handlers.HealthHandlers
	settingsHandlers  *handlers.SettingsHandlers
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("🚀 Starting Ecosystem Manager API...")

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
	queueDir := filepath.Join(scenarioRoot, "queue")
	promptsDir := filepath.Join(scenarioRoot, "prompts")

	// Ensure queue directories exist
	dirs := []string{"pending", "in-progress", "review", "completed", "failed"}
	for _, dir := range dirs {
		fullPath := filepath.Join(queueDir, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return err
		}
	}

	// Initialize storage
	storage = tasks.NewStorage(queueDir)
	log.Println("✅ Task storage initialized")

	// Initialize prompts assembler
	assembler, err = prompts.NewAssembler(promptsDir)
	if err != nil {
		return err
	}
	log.Println("✅ Prompt assembler initialized")

	// Initialize WebSocket manager
	wsManager = websocket.NewManager()
	log.Println("✅ WebSocket manager initialized")

	// Initialize queue processor
	processor = queue.NewProcessor(
		30*time.Second, // 30-second processing interval
		storage,
		assembler,
		wsManager.GetBroadcastChannel(),
	)
	log.Println("✅ Queue processor initialized")

	// Initialize handlers
	taskHandlers = handlers.NewTaskHandlers(storage, assembler, processor, wsManager)
	queueHandlers = handlers.NewQueueHandlers(processor, wsManager)
	discoveryHandlers = handlers.NewDiscoveryHandlers(assembler)
	healthHandlers = handlers.NewHealthHandlers(processor)
	settingsHandlers = handlers.NewSettingsHandlers(processor, wsManager)
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
	api.HandleFunc("/tasks/{id}", taskHandlers.GetTaskHandler).Methods("GET")
	api.HandleFunc("/tasks/{id}", taskHandlers.UpdateTaskHandler).Methods("PUT")
	api.HandleFunc("/tasks/{id}", taskHandlers.DeleteTaskHandler).Methods("DELETE")
	api.HandleFunc("/tasks/{id}/status", taskHandlers.UpdateTaskStatusHandler).Methods("PUT") // Missing route

	// Task prompt operations
	api.HandleFunc("/tasks/{id}/prompt", taskHandlers.GetTaskPromptHandler).Methods("GET")
	api.HandleFunc("/tasks/{id}/prompt/assembled", taskHandlers.GetAssembledPromptHandler).Methods("GET")

	// Prompt viewer (no task ID required)
	api.HandleFunc("/prompt-viewer", taskHandlers.PromptViewerHandler).Methods("GET")

	// Queue management routes
	api.HandleFunc("/queue/status", queueHandlers.GetQueueStatusHandler).Methods("GET")
	api.HandleFunc("/queue/trigger", queueHandlers.TriggerQueueProcessingHandler).Methods("POST")
	api.HandleFunc("/queue/reset-rate-limit", queueHandlers.ResetRateLimitHandler).Methods("POST")

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

	// Discovery routes
	api.HandleFunc("/resources", discoveryHandlers.GetResourcesHandler).Methods("GET")
	api.HandleFunc("/scenarios", discoveryHandlers.GetScenariosHandler).Methods("GET")
	api.HandleFunc("/resources/{name}/status", discoveryHandlers.GetResourceStatusHandler).Methods("GET") // Missing route
	api.HandleFunc("/scenarios/{name}/status", discoveryHandlers.GetScenarioStatusHandler).Methods("GET") // Missing route
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
