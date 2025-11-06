package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

	// Determine project root securely
	// API runs from scenarios/browser-automation-studio/api/ directory
	projectRoot := os.Getenv("VROOLI_ROOT")
	if projectRoot == "" {
		// Fallback: resolve from current executable location
		if execPath, err := os.Executable(); err == nil {
			// Expected structure: {VROOLI_ROOT}/scenarios/browser-automation-studio/api/binary
			// So go up 3 directories to reach VROOLI_ROOT
			projectRoot = filepath.Join(filepath.Dir(execPath), "..", "..", "..")
			projectRoot, _ = filepath.Abs(projectRoot)
		} else {
			fmt.Fprintf(os.Stderr, "❌ VROOLI_ROOT not set and cannot determine project root: %v\n", err)
			os.Exit(1)
		}
	}

	// Validate and change to project root
	if absRoot, err := filepath.Abs(projectRoot); err == nil {
		// Security check: ensure path doesn't contain suspicious patterns
		if filepath.Clean(absRoot) != absRoot {
			fmt.Fprintf(os.Stderr, "❌ Invalid project root path (potential path traversal): %s\n", projectRoot)
			os.Exit(1)
		}
		if err := os.Chdir(absRoot); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to change to project root directory %s: %v\n", absRoot, err)
			os.Exit(1)
		}
	} else {
		fmt.Fprintf(os.Stderr, "❌ Failed to resolve project root path %s: %v\n", projectRoot, err)
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

	// CORS middleware - secure by default, configurable via environment
	r.Use(corsMiddleware(log))

	// Routes
	r.Get("/health", handler.Health)
	r.Get("/ws", handler.HandleWebSocket) // WebSocket endpoint

	r.Route("/api/v1", func(r chi.Router) {
		// Health endpoint under /api/v1 for consistency
		r.Get("/health", handler.Health)
		// Project routes
		r.Post("/projects", handler.CreateProject)
		r.Get("/projects", handler.ListProjects)
		r.Get("/projects/{id}", handler.GetProject)
		r.Put("/projects/{id}", handler.UpdateProject)
		r.Delete("/projects/{id}", handler.DeleteProject)
		r.Get("/projects/{id}/workflows", handler.GetProjectWorkflows)
		r.Post("/projects/{id}/workflows/bulk-delete", handler.BulkDeleteProjectWorkflows)
		r.Post("/projects/{id}/execute-all", handler.ExecuteAllProjectWorkflows)

		// Workflow routes
		r.Post("/workflows/create", handler.CreateWorkflow)
		r.Post("/workflows/execute-adhoc", handler.ExecuteAdhocWorkflow)
		r.Get("/workflows", handler.ListWorkflows)
		r.Get("/workflows/{id}", handler.GetWorkflow)
		r.Put("/workflows/{id}", handler.UpdateWorkflow)
		r.Get("/workflows/{id}/versions", handler.ListWorkflowVersions)
		r.Get("/workflows/{id}/versions/{version}", handler.GetWorkflowVersion)
		r.Post("/workflows/{id}/versions/{version}/restore", handler.RestoreWorkflowVersion)
		r.Post("/workflows/{id}/execute", handler.ExecuteWorkflow)
		r.Post("/workflows/{id}/modify", handler.ModifyWorkflow)

		// Execution routes
		r.Get("/executions", handler.ListExecutions)
		r.Get("/executions/{id}", handler.GetExecution)
		r.Get("/executions/{id}/timeline", handler.GetExecutionTimeline)
		r.Post("/executions/{id}/export", handler.PostExecutionExport)
		r.Post("/executions/{id}/stop", handler.StopExecution)
		r.Get("/executions/{id}/screenshots", handler.GetExecutionScreenshots)

		// Scenario routes
		r.Get("/scenarios", handler.ListScenarios)
		r.Get("/scenarios/{name}/port", handler.GetScenarioPort)

		// Screenshot serving routes
		r.Get("/screenshots/*", handler.ServeScreenshot)
		r.Get("/screenshots/thumbnail/*", handler.ServeThumbnail)

		// Preview route for taking screenshots of URLs
		r.Post("/preview-screenshot", handler.TakePreviewScreenshot)

		// Element analysis route for intelligent selector detection
		r.Post("/analyze-elements", handler.AnalyzeElements)

		// Element at coordinate route for click-based selector detection
		r.Post("/element-at-coordinate", handler.GetElementAtCoordinate)

		// AI-powered element analysis route using Ollama text models with DOM
		r.Post("/ai-analyze-elements", handler.AIAnalyzeElements)

		// Recording ingestion and asset serving
		r.Post("/recordings/import", handler.ImportRecording)
		r.Get("/recordings/assets/{executionID}/*", handler.ServeRecordingAsset)

		// DOM tree extraction for Browser Inspector tab
		r.Post("/dom-tree", handler.GetDOMTree)
	})

	// Get API host for logging
	apiHost := os.Getenv("API_HOST")
	if apiHost == "" {
		apiHost = "localhost"
	}

	corsCfg := resolveAllowedOrigins()
	corsPolicy := "restricted"
	if corsCfg.allowAll {
		corsPolicy = "allow_all"
	} else {
		corsPolicy = strings.Join(corsCfg.allowedOrigins, ",")
	}

	log.WithFields(logrus.Fields{
		"api_port":    port,
		"api_host":    apiHost,
		"cors_policy": corsPolicy,
	}).Info("🚀 Browser Automation Studio API starting")
	log.WithField("endpoint", fmt.Sprintf("http://%s:%s/api/v1", apiHost, port)).Info("📊 API endpoint ready")

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.WithError(err).Fatal("❌ Server failed to start")
	}
}

// Test change for auto-rebuild validation
