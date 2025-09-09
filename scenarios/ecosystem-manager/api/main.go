package main

import (
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
	storage     *tasks.Storage
	assembler   *prompts.Assembler
	processor   *queue.Processor
	wsManager   *websocket.Manager
	
	// Handlers
	taskHandlers      *handlers.TaskHandlers
	queueHandlers     *handlers.QueueHandlers
	discoveryHandlers *handlers.DiscoveryHandlers
	healthHandlers    *handlers.HealthHandlers
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("🚀 Starting Ecosystem Manager API...")
	
	// Initialize core components
	if err := initializeComponents(); err != nil {
		log.Fatalf("Failed to initialize components: %v", err)
	}
	
	// Start queue processor
	processor.Start()
	defer processor.Stop()
	
	// Set up HTTP server
	router := setupRoutes()
	
	// Get port from environment
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "5020" // Default port
	}
	
	// Validate port
	if _, err := strconv.Atoi(port); err != nil {
		log.Printf("Invalid API_PORT '%s', using default 5020", port)
		port = "5020"
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
	// Determine directories
	queueDir := filepath.Join("..", "queue")
	promptsDir := filepath.Join("..", "prompts")
	
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
	var err error
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
	
	// Queue management routes  
	api.HandleFunc("/queue/status", queueHandlers.GetQueueStatusHandler).Methods("GET")
	api.HandleFunc("/queue/trigger", queueHandlers.TriggerQueueProcessingHandler).Methods("POST")
	
	// Process management (match original path)
	api.HandleFunc("/processes/running", queueHandlers.GetRunningProcessesHandler).Methods("GET")
	
	// Maintenance (match original path)
	api.HandleFunc("/maintenance/state", queueHandlers.SetMaintenanceStateHandler).Methods("POST")
	
	// Queue control endpoints (these are new/different from original)
	api.HandleFunc("/queue/stop", queueHandlers.StopQueueProcessorHandler).Methods("POST")
	api.HandleFunc("/queue/start", queueHandlers.StartQueueProcessorHandler).Methods("POST")
	api.HandleFunc("/queue/processes/terminate", queueHandlers.TerminateProcessHandler).Methods("POST")
	
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