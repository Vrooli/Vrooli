package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
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

	// Build allowed origins from environment variables
	uiPort := os.Getenv("UI_PORT")
	if uiPort == "" {
		log.Fatal("❌ UI_PORT environment variable is required")
	}
	
	// Get UI host from environment, default to localhost for dev
	uiHost := os.Getenv("UI_HOST")
	if uiHost == "" {
		uiHost = "localhost"
	}
	
	allowedOrigins := []string{
		fmt.Sprintf("http://%s:%s", uiHost, uiPort),
	}
	
	// Add custom origins if specified
	if customOrigins := os.Getenv("ALLOWED_ORIGINS"); customOrigins != "" {
		// Split by comma and add each origin
		for _, origin := range strings.Split(customOrigins, ",") {
			if trimmed := strings.TrimSpace(origin); trimmed != "" {
				allowedOrigins = append(allowedOrigins, trimmed)
			}
		}
	}

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:          300,
	}))

	// Routes
	r.Get("/health", handler.Health)
	r.Get("/ws", handler.HandleWebSocket) // WebSocket endpoint
	
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/workflows/create", handler.CreateWorkflow)
		r.Get("/workflows", handler.ListWorkflows)
		r.Get("/workflows/{id}", handler.GetWorkflow)
		r.Post("/workflows/{id}/execute", handler.ExecuteWorkflow)
		r.Get("/executions/{id}/screenshots", handler.GetExecutionScreenshots)
		
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
		"ui_port":  uiPort,
		"api_host": apiHost,
		"ui_host": uiHost,
		"cors_origins": len(allowedOrigins),
	}).Info("🚀 Browser Automation Studio API starting")
	log.WithField("endpoint", fmt.Sprintf("http://%s:%s/api/v1", apiHost, port)).Info("📊 API endpoint ready")
	
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.WithError(err).Fatal("❌ Server failed to start")
	}
}