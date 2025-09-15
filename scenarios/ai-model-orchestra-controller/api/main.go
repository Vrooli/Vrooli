package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

const (
	apiVersion  = "1.0.0"
	serviceName = "ai-model-orchestra-controller"
)

func main() {
	logger := log.New(os.Stdout, "[ai-orchestrator] ", log.LstdFlags)
	
	logger.Printf("🚀 Starting AI Model Orchestra Controller v%s", apiVersion)
	
	// Validate required environment variables
	port := os.Getenv("API_PORT")
	if port == "" {
		logger.Fatalf("❌ API_PORT environment variable is required")
	}
	
	// Initialize application state
	app := &AppState{
		Logger: logger,
	}
	
	// Initialize services
	var err error
	
	// Initialize database
	app.DB, err = initDatabase(logger)
	if err != nil {
		logger.Printf("❌ Database initialization failed: %v", err)
		// Continue without database for development
	} else {
		if err := initSchema(app.DB); err != nil {
			logger.Printf("⚠️  Schema initialization failed: %v", err)
		}
	}
	
	// Initialize Redis
	app.Redis, err = initRedis(logger)
	if err != nil {
		logger.Printf("❌ Redis initialization failed: %v", err)
		// Continue without Redis for development
	}
	
	// Initialize Ollama
	app.OllamaClient, err = initOllama(logger)
	if err != nil {
		logger.Printf("❌ Ollama initialization failed: %v", err)
		// Continue without Ollama for development (will fall back to simulation)
	}
	
	// Initialize Docker
	app.DockerClient, err = initDocker(logger)
	if err != nil {
		logger.Printf("❌ Docker initialization failed: %v", err)
		// Continue without Docker for development
	}
	
	// Start background system monitoring
	go startSystemMonitoring(app.DB, logger)
	
	// Initialize handlers
	handlers := NewHandlers(app)
	
	// Setup routes
	r := mux.NewRouter()
	
	// Health check endpoints (both versioned and legacy)
	r.HandleFunc("/health", handlers.handleHealthCheck).Methods("GET")
	r.HandleFunc("/api/v1/health", handlers.handleHealthCheck).Methods("GET")
	
	// API v1 routes - all endpoints under /api/v1
	api := r.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/ai/select-model", handlers.handleModelSelect).Methods("POST")
	api.HandleFunc("/ai/route-request", handlers.handleRouteRequest).Methods("POST")
	api.HandleFunc("/ai/models/status", handlers.handleModelsStatus).Methods("GET")
	api.HandleFunc("/ai/resources/metrics", handlers.handleResourceMetrics).Methods("GET")
	
	// Dashboard and static files
	r.HandleFunc("/dashboard", handlers.handleDashboard).Methods("GET")
	r.PathPrefix("/ui/").Handler(http.StripPrefix("/ui/", http.FileServer(http.Dir("../ui/"))))
	
	// Apply middleware
	r.Use(corsMiddleware)
	
	logger.Printf("🎛️  API server starting on port %s", port)
	logger.Printf("📊 Dashboard available at: http://localhost:%s/dashboard", port)
	logger.Printf("🔗 Health check: http://localhost:%s/health", port)
	logger.Printf("🔗 Health check (v1): http://localhost:%s/api/v1/health", port)
	
	if err := http.ListenAndServe(":"+port, r); err != nil {
		logger.Fatalf("❌ Server failed to start: %v", err)
	}
}