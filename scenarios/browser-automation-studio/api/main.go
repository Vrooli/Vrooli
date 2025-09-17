package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/browserless"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/handlers"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
)

func main() {
	// Protect against direct execution - must be run through lifecycle system
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start browser-automation-studio

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Change working directory to project root for proper resource access
	// API runs from scenarios/browser-automation-studio/api/, so go up 3 levels to project root
	if err := os.Chdir("../../../"); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to change to project root directory: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})
	
	logLevel := os.Getenv("LOG_LEVEL")
	switch logLevel {
	case "debug":
		log.SetLevel(logrus.DebugLevel)
	case "warn":
		log.SetLevel(logrus.WarnLevel)
	case "error":
		log.SetLevel(logrus.ErrorLevel)
	default:
		log.SetLevel(logrus.InfoLevel)
	}

	// Log current working directory for transparency
	if cwd, err := os.Getwd(); err == nil {
		log.WithField("cwd", cwd).Info("Starting Browser Automation Studio API")
	} else {
		log.Info("Starting Browser Automation Studio API")
	}

	// Initialize database connection
	db, err := database.NewConnection(log)
	if err != nil {
		log.WithError(err).Fatal("Failed to connect to database")
	}
	defer db.Close()

	// Initialize repository
	repo := database.NewRepository(db, log)

	// Initialize WebSocket hub
	hub := wsHub.NewHub(log)
	go hub.Run()

	// Initialize browserless client
	browserlessClient := browserless.NewClient(log, repo)

	// Initialize handlers
	handler := handlers.NewHandler(repo, browserlessClient, hub, log)

	// Get port configuration - required from lifecycle system
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}

	// Setup router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Simple CORS middleware - allows all origins for development
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set CORS headers
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token, X-Requested-With")
			w.Header().Set("Access-Control-Expose-Headers", "Link")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "300")
			
			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	})

	// Routes
	r.Get("/health", handler.Health)
	r.Get("/ws", handler.HandleWebSocket) // WebSocket endpoint
	
	r.Route("/api/v1", func(r chi.Router) {
		// Project routes
		r.Post("/projects", handler.CreateProject)
		r.Get("/projects", handler.ListProjects)
		r.Get("/projects/{id}", handler.GetProject)
		r.Put("/projects/{id}", handler.UpdateProject)
		r.Delete("/projects/{id}", handler.DeleteProject)
		r.Get("/projects/{id}/workflows", handler.GetProjectWorkflows)
		r.Post("/projects/{id}/execute-all", handler.ExecuteAllProjectWorkflows)
		
		// Workflow routes
		r.Post("/workflows/create", handler.CreateWorkflow)
		r.Get("/workflows", handler.ListWorkflows)
		r.Get("/workflows/{id}", handler.GetWorkflow)
		r.Post("/workflows/{id}/execute", handler.ExecuteWorkflow)
		
		// Execution routes
		r.Get("/executions", handler.ListExecutions)
		r.Get("/executions/{id}", handler.GetExecution)
		r.Post("/executions/{id}/stop", handler.StopExecution)
		r.Get("/executions/{id}/screenshots", handler.GetExecutionScreenshots)
		
		// Scenario routes
		r.Get("/scenarios/{name}/port", handler.GetScenarioPort)
		
		// Screenshot serving routes
		r.Get("/screenshots/*", handler.ServeScreenshot)
		r.Get("/screenshots/thumbnail/*", handler.ServeThumbnail)
	})

	// Get API host for logging
	apiHost := os.Getenv("API_HOST")
	if apiHost == "" {
		apiHost = "localhost"
	}
	
	log.WithFields(logrus.Fields{
		"api_port": port,
		"api_host": apiHost,
		"cors_policy": "allow_all",
	}).Info("🚀 Browser Automation Studio API starting")
	log.WithField("endpoint", fmt.Sprintf("http://%s:%s/api/v1", apiHost, port)).Info("📊 API endpoint ready")
	
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.WithError(err).Fatal("❌ Server failed to start")
	}
}