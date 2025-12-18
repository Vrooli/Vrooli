package main

import (
	"github.com/vrooli/api-core/preflight"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

const (
	apiVersion  = "1.0.0"
	serviceName = "ai-model-orchestra-controller"
)

// validateEnvironment checks all required environment variables at startup
func validateEnvironment(logger *log.Logger) error {
	required := map[string]string{
		"API_PORT":                "API server port",
		"ORCHESTRATOR_HOST":       "Host for orchestrator services",
		"RESOURCE_PORTS_POSTGRES": "PostgreSQL port",
		"RESOURCE_PORTS_REDIS":    "Redis port",
		"RESOURCE_PORTS_OLLAMA":   "Ollama port",
		"POSTGRES_USER":           "PostgreSQL username",
		"POSTGRES_PASSWORD":       "PostgreSQL password",
	}
	
	optional := map[string]string{
		"POSTGRES_DB":                "PostgreSQL database name (defaults to 'orchestrator')",
		"REDIS_PASSWORD":             "Redis password (optional)",
		"ORCHESTRATOR_LOG_LEVEL":     "Log level (defaults to 'info')",
		"RESOURCE_MONITOR_INTERVAL":  "Resource monitoring interval (defaults to 5000ms)",
		"MODEL_HEALTH_CHECK_INTERVAL": "Model health check interval (defaults to 30000ms)",
		"MAX_RETRY_ATTEMPTS":         "Maximum retry attempts (defaults to 3)",
		"REQUEST_TIMEOUT":            "Request timeout (defaults to 30000ms)",
		"UI_PORT":                    "UI server port",
	}
	
	missingRequired := []string{}
	
	// Check required variables
	for env, description := range required {
		if os.Getenv(env) == "" {
			missingRequired = append(missingRequired, fmt.Sprintf("  - %s: %s", env, description))
		}
	}
	
	if len(missingRequired) > 0 {
		logger.Printf("❌ Missing required environment variables:")
		for _, missing := range missingRequired {
			logger.Println(missing)
		}
		return fmt.Errorf("missing %d required environment variables", len(missingRequired))
	}
	
	// Log optional variables status
	logger.Printf("✅ All required environment variables are set")
	logger.Printf("📋 Optional environment variables:")
	for env, description := range optional {
		value := os.Getenv(env)
		if value != "" {
			// Don't log sensitive values
			if strings.Contains(strings.ToLower(env), "password") || strings.Contains(strings.ToLower(env), "secret") {
				logger.Printf("  ✓ %s: [SET] - %s", env, description)
			} else {
				logger.Printf("  ✓ %s: %s - %s", env, value, description)
			}
		} else {
			logger.Printf("  ○ %s: [NOT SET] - %s", env, description)
		}
	}
	
	return nil
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "ai-model-orchestra-controller",
	}) {
		return // Process was re-exec'd after rebuild
	}

	logger := log.New(os.Stdout, "[ai-orchestrator] ", log.LstdFlags)
	
	logger.Printf("🚀 Starting AI Model Orchestra Controller v%s", apiVersion)
	
	// Validate all required environment variables
	if err := validateEnvironment(logger); err != nil {
		logger.Fatalf("❌ Environment validation failed: %v", err)
	}
	
	port := os.Getenv("API_PORT")
	
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
	
	// Start background system monitoring with health checks
	go startSystemMonitoring(app)
	
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
	logger.Printf("📊 Dashboard endpoint: /dashboard")
	logger.Printf("🔗 Health check endpoints: /health and /api/v1/health")
	logger.Printf("🚀 Service: %s v%s", serviceName, apiVersion)
	
	if err := http.ListenAndServe(":"+port, r); err != nil {
		logger.Fatalf("❌ Server failed to start: %v", err)
	}
}