package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/handlers"
	"github.com/vrooli/browser-automation-studio/middleware"
	"github.com/vrooli/browser-automation-studio/services/recovery"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
)

const globalRequestTimeout = 15 * time.Minute

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
		log.WithField("cwd", cwd).Info("Starting Vrooli Ascension API")
	} else {
		log.Info("Starting Vrooli Ascension API")
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

	// Resolve allowed origins before constructing handlers
	corsCfg := middleware.GetCachedCorsConfig()

	// Initialize handlers
	handler := handlers.NewHandler(repo, hub, log, corsCfg.AllowAll, corsCfg.AllowedOrigins)

	// Wire up WebSocket input forwarding for low-latency input events
	// This allows the UI to send input via WebSocket instead of HTTP POST
	hub.SetInputForwarder(handler.CreateInputForwarder())

	// Startup health check - validate critical dependencies before accepting requests
	// This prevents the scenario where the API starts but all workflow executions fail
	if err := performStartupHealthCheck(log); err != nil {
		log.WithError(err).Warn("⚠️  Startup health check failed - some features may be unavailable")
		// Continue startup but log the warning - this allows the API to serve health endpoints
		// and provide diagnostic information even when the automation engine is unavailable
	}

	// Recover stale executions from previous runs (progress continuity)
	recoverySvc := recovery.NewService(repo, log)
	recoverCtx, recoverCancel := context.WithTimeout(context.Background(), 30*time.Second)
	if result, err := recoverySvc.RecoverStaleExecutions(recoverCtx); err != nil {
		log.WithError(err).Warn("⚠️  Stale execution recovery failed - some executions may show incorrect status")
	} else if result.TotalStale > 0 {
		log.WithFields(logrus.Fields{
			"recovered":   result.Recovered,
			"resumable":   result.Resumable,
			"total_stale": result.TotalStale,
		}).Info("✅ Stale execution recovery completed")
	}
	recoverCancel()

	// Get port configuration - required from lifecycle system
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}

	// Setup router
	r := chi.NewRouter()

	// Middleware
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(globalRequestTimeout))

	// CORS middleware - secure by default, configurable via environment
	r.Use(middleware.CorsMiddleware(log))

	// Routes
	r.Get("/health", handler.Health)
	r.Get("/ws", handler.HandleWebSocket)                                     // WebSocket endpoint for browser clients
	r.Get("/ws/recording/{sessionId}/frames", handler.HandleDriverFrameStream) // WebSocket for playwright-driver binary frame streaming

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
		r.Post("/workflows/validate", handler.ValidateWorkflow)
		r.Post("/workflows/validate-resolved", handler.ValidateResolvedWorkflow)
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
		r.Post("/executions/{id}/resume", handler.ResumeExecution)
		r.Get("/executions/{id}/screenshots", handler.GetExecutionScreenshots)

		// Export library routes
		r.Get("/exports", handler.ListExports)
		r.Post("/exports", handler.CreateExport)
		r.Get("/exports/{id}", handler.GetExport)
		r.Patch("/exports/{id}", handler.UpdateExport)
		r.Delete("/exports/{id}", handler.DeleteExport)
		r.Post("/exports/{id}/generate-caption", handler.GenerateExportCaption)

		// Scenario routes
		r.Get("/scenarios", handler.ListScenarios)
		r.Get("/scenarios/{name}/port", handler.GetScenarioPort)

		// Recording session profiles
		r.Get("/recordings/sessions", handler.ListRecordingSessionProfiles)
		r.Post("/recordings/sessions", handler.CreateRecordingSessionProfile)
		r.Patch("/recordings/sessions/{profileId}", handler.UpdateRecordingSessionProfile)
		r.Delete("/recordings/sessions/{profileId}", handler.DeleteRecordingSessionProfile)

		// Screenshot serving routes
		r.Get("/screenshots/*", handler.ServeScreenshot)
		r.Get("/screenshots/thumbnail/*", handler.ServeThumbnail)

		// Preview route for taking screenshots of URLs
		// POST /api/v1/preview-screenshot
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

		// Live recording routes (Record Mode)
		r.Post("/recordings/live/session", handler.CreateRecordingSession)
		r.Post("/recordings/live/session/{sessionId}/close", handler.CloseRecordingSession)
		r.Post("/recordings/live/start", handler.StartLiveRecording)
		r.Post("/recordings/live/{sessionId}/stop", handler.StopLiveRecording)
		r.Get("/recordings/live/{sessionId}/status", handler.GetRecordingStatus)
		r.Get("/recordings/live/{sessionId}/actions", handler.GetRecordedActions)
		r.Post("/recordings/live/{sessionId}/action", handler.ReceiveRecordingAction) // Callback for driver action streaming
		r.Post("/recordings/live/{sessionId}/frame", handler.ReceiveRecordingFrame)  // Callback for driver frame streaming
		r.Post("/recordings/live/{sessionId}/navigate", handler.NavigateRecordingSession)
		r.Post("/recordings/live/{sessionId}/viewport", handler.UpdateRecordingViewport)
		r.Post("/recordings/live/{sessionId}/input", handler.ForwardRecordingInput)
		r.Post("/recordings/live/{sessionId}/stream-settings", handler.UpdateStreamSettings)
		r.Get("/recordings/live/{sessionId}/frame", handler.GetRecordingFrame)
		r.Post("/recordings/live/{sessionId}/screenshot", handler.CaptureRecordingScreenshot)
		r.Post("/recordings/live/{sessionId}/generate-workflow", handler.GenerateWorkflowFromRecording)
		r.Post("/recordings/live/{sessionId}/validate-selector", handler.ValidateSelector)
		r.Post("/recordings/live/{sessionId}/replay-preview", handler.ReplayRecordingPreview)
		r.Post("/recordings/live/{sessionId}/persist", handler.PersistRecordingSession)

		// DOM tree extraction for Browser Inspector tab
		r.Post("/dom-tree", handler.GetDOMTree)
	})

	// Get API host for logging
	apiHost := os.Getenv("API_HOST")
	if apiHost == "" {
		apiHost = "localhost"
	}

	corsPolicy := "restricted"
	if corsCfg.AllowAll {
		corsPolicy = "allow_all"
	} else {
		corsPolicy = strings.Join(corsCfg.AllowedOrigins, ",")
	}

	// Check if port is available before attempting to bind
	if err := checkPortAvailable(port); err != nil {
		log.WithFields(logrus.Fields{
			"port":  port,
			"error": err.Error(),
		}).Fatal("❌ Port unavailable - another process may be using it")
	}

	log.WithFields(logrus.Fields{
		"api_port":    port,
		"api_host":    apiHost,
		"cors_policy": corsPolicy,
	}).Info("🚀 Vrooli Ascension API starting")
	log.WithField("endpoint", fmt.Sprintf("http://%s:%s/api/v1", apiHost, port)).Info("📊 API endpoint ready")

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.WithError(err).Fatal("❌ Server failed to start")
	}
}

// checkPortAvailable verifies a port is not already in use before binding.
// Returns nil if port is available, error if port is in use or check fails.
func checkPortAvailable(port string) error {
	addr := "127.0.0.1:" + port
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		// Port is in use or some other error
		return fmt.Errorf("port %s is unavailable: %w (hint: check if another instance is running with 'lsof -i :%s' or 'ss -tlnp | grep %s')", port, err, port, port)
	}
	// Successfully bound, port is available - close immediately so actual server can bind
	listener.Close()
	return nil
}

// performStartupHealthCheck validates critical dependencies are available before accepting requests.
// This catches configuration issues early rather than having workflows fail at runtime.
func performStartupHealthCheck(log *logrus.Logger) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var errors []string

	// Check 1: Playwright driver health
	playwrightURL := os.Getenv("PLAYWRIGHT_DRIVER_URL")
	if playwrightURL == "" {
		playwrightURL = "http://127.0.0.1:39400"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, playwrightURL+"/health", nil)
	if err != nil {
		errors = append(errors, fmt.Sprintf("playwright driver: failed to create request: %v", err))
	} else {
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			errors = append(errors, fmt.Sprintf("playwright driver at %s: %v (ensure playwright-driver is running)", playwrightURL, err))
		} else {
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				errors = append(errors, fmt.Sprintf("playwright driver at %s: unhealthy (status %d)", playwrightURL, resp.StatusCode))
			} else {
				log.WithField("url", playwrightURL).Info("✅ Playwright driver healthy")
			}
		}
	}

	// Check 2: MinIO storage health (if configured)
	minioEndpoint := os.Getenv("MINIO_ENDPOINT")
	if minioEndpoint != "" {
		// MinIO health check is optional - storage issues are handled gracefully at runtime
		log.WithField("endpoint", minioEndpoint).Info("✅ MinIO endpoint configured")
	}

	if len(errors) > 0 {
		for _, e := range errors {
			log.Warn("⚠️  " + e)
		}
		return fmt.Errorf("%d startup health check(s) failed", len(errors))
	}

	return nil
}
