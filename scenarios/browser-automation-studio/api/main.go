package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/handlers"
	"github.com/vrooli/browser-automation-studio/middleware"
	"github.com/vrooli/browser-automation-studio/performance"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
	"github.com/vrooli/browser-automation-studio/services/recovery"
	"github.com/vrooli/browser-automation-studio/services/scheduler"
	"github.com/vrooli/browser-automation-studio/services/uxmetrics"
	uxanalyzer "github.com/vrooli/browser-automation-studio/services/uxmetrics/analyzer"
	uxrepository "github.com/vrooli/browser-automation-studio/services/uxmetrics/repository"
	"github.com/vrooli/browser-automation-studio/sidecar"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
)

const globalRequestTimeout = 15 * time.Minute

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "browser-automation-studio",
	}) {
		return // Process was re-exec'd after rebuild
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
	logrus.SetFormatter(log.Formatter)
	logrus.SetLevel(log.Level)
	logrus.SetOutput(log.Out)

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

	// Initialize repository
	repo := database.NewRepository(db, log)

	// Initialize WebSocket hub
	hub := wsHub.NewHub(log)
	go hub.Run()

	// Load configuration
	cfg := config.Load()

	// Initialize entitlement services
	entitlementSvc := entitlement.NewService(cfg.Entitlement, log)
	usageTracker := entitlement.NewUsageTracker(db.RawDB(), log)
	aiCreditsTracker := entitlement.NewAICreditsTracker(db.RawDB(), log)
	entitlementHandler := handlers.NewEntitlementHandler(entitlementSvc, usageTracker, aiCreditsTracker, repo)
	entitlementMiddleware := middleware.NewEntitlementMiddleware(entitlementSvc, log, cfg.Entitlement, repo)

	if cfg.Entitlement.Enabled {
		log.WithFields(logrus.Fields{
			"service_url":     cfg.Entitlement.ServiceURL,
			"cache_ttl":       cfg.Entitlement.CacheTTL,
			"default_tier":    cfg.Entitlement.DefaultTier,
			"watermark_tiers": cfg.Entitlement.WatermarkTiers,
		}).Info("✅ Entitlement system enabled")
	} else {
		log.Info("⚠️  Entitlement system disabled - all features available without restrictions")
	}

	// Initialize UX metrics repository (used by both handler wiring and API endpoints)
	uxRepo := uxrepository.NewPostgresRepository(db.DB)

	// Resolve allowed origins before constructing handlers
	corsCfg := middleware.GetCachedCorsConfig()

	// Initialize handlers with UX metrics integration and entitlement services
	// The UX metrics collector wraps the event sink to passively capture interaction data
	// The entitlement services enable tier-based feature gating and AI credits tracking
	deps := handlers.InitDefaultDepsWithOptions(repo, hub, log, handlers.DepsOptions{
		UXMetricsRepo:      uxRepo,
		EntitlementService: entitlementSvc,
		AICreditsTracker:   aiCreditsTracker,
	})
	handler := handlers.NewHandlerWithDeps(repo, hub, log, corsCfg.AllowAll, corsCfg.AllowedOrigins, deps)

	// Initialize UX metrics service for API endpoints
	// The collector is integrated in the workflow service via InitDefaultDepsWithUXMetrics
	uxAnalyzer := uxanalyzer.NewAnalyzer(uxRepo, nil)
	uxService := uxmetrics.NewService(nil, uxAnalyzer, uxRepo)
	uxHandler := handlers.NewUXMetricsHandler(uxService, log)
	log.Info("✅ UX metrics service initialized with event pipeline integration")

	// Wire up WebSocket input forwarding for low-latency input events
	// This allows the UI to send input via WebSocket instead of HTTP POST
	hub.SetInputForwarder(handler.CreateInputForwarder())

	// Initialize playwright-driver sidecar management
	// This enables automatic restart on crashes, health monitoring, and recording recovery
	var sidecarDeps *sidecar.Dependencies
	driverClient, err := driver.NewClient(driver.WithLogger(log))
	if err != nil {
		log.WithError(err).Warn("⚠️  Failed to create driver client - sidecar management disabled")
	} else {
		sidecarDeps, err = sidecar.BuildDependencies(db.DB, driverClient, hub, log)
		if err != nil {
			log.WithError(err).Warn("⚠️  Failed to initialize sidecar management")
		} else if sidecarDeps.IsEnabled() {
			// Start sidecar services (spawns playwright-driver)
			startCtx, startCancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := sidecarDeps.Start(startCtx); err != nil {
				log.WithError(err).Warn("⚠️  Sidecar failed to start - playwright-driver may not be available")
			} else {
				log.Info("✅ Playwright-driver sidecar started")
			}
			startCancel()
		}
	}

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

	// Setup router
	r := chi.NewRouter()

	// Middleware
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(globalRequestTimeout))

	// CORS middleware - secure by default, configurable via environment
	r.Use(middleware.CorsMiddleware(log))

	// Entitlement middleware - injects user identity and entitlement into context
	r.Use(entitlementMiddleware.InjectEntitlement)

	// Routes
	// Health endpoint using api-core/health for standardized response format
	healthHandler := health.New().
		Version("1.0.0").
		Check(health.DB(db.RawDB()), health.Critical).
		Handler()
	r.Get("/health", healthHandler)
	r.Get("/ws", handler.HandleWebSocket)                                                 // WebSocket endpoint for browser clients
	r.Get("/ws/recording/{sessionId}/frames", handler.HandleDriverFrameStream)            // WebSocket for playwright-driver binary frame streaming (recording mode)
	r.Get("/ws/execution/{executionId}/frames", handler.HandleDriverExecutionFrameStream) // WebSocket for playwright-driver binary frame streaming (execution mode)

	r.Route("/api/v1", func(r chi.Router) {
		// Health endpoint under /api/v1 for consistency
		r.Get("/health", healthHandler)
		// Project routes
		r.Post("/projects", handler.CreateProject)
		r.Post("/projects/inspect-folder", handler.InspectProjectFolder)
		r.Post("/projects/import", handler.ImportProject)
		r.Get("/projects", handler.ListProjects)
		r.Get("/projects/{id}", handler.GetProject)
		r.Put("/projects/{id}", handler.UpdateProject)
		r.Delete("/projects/{id}", handler.DeleteProject)
		r.Get("/projects/{id}/workflows", handler.GetProjectWorkflows)
		r.Post("/projects/{id}/workflows/bulk-delete", handler.BulkDeleteProjectWorkflows)
		r.Post("/projects/{id}/execute-all", handler.ExecuteAllProjectWorkflows)
		r.Get("/projects/{id}/files/tree", handler.GetProjectFileTree)
		r.Get("/projects/{id}/files/read", handler.ReadProjectFile)
		r.Post("/projects/{id}/files/mkdir", handler.MkdirProjectPath)
		r.Post("/projects/{id}/files/write", handler.WriteProjectWorkflowFile)
		r.Post("/projects/{id}/files/move", handler.MoveProjectFile)
		r.Post("/projects/{id}/files/delete", handler.DeleteProjectFile)
		r.Post("/projects/{id}/files/resync", handler.ResyncProjectFiles)

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
		r.Get("/executions/{id}/recorded-videos", handler.GetExecutionRecordedVideos)
		r.Get("/executions/{id}/recorded-traces", handler.GetExecutionRecordedTraces)
		r.Get("/executions/{id}/recorded-har", handler.GetExecutionRecordedHar)
		r.Post("/executions/{executionId}/frames", handler.ReceiveExecutionFrame) // Frame streaming callback from playwright-driver
		r.Post("/executions/{id}/seed-cleanup", handler.ScheduleExecutionSeedCleanup)

		// Export library routes
		r.Get("/exports", handler.ListExports)
		r.Post("/exports", handler.CreateExport)
		r.Get("/exports/{id}", handler.GetExport)
		r.Patch("/exports/{id}", handler.UpdateExport)
		r.Delete("/exports/{id}", handler.DeleteExport)
		r.Post("/exports/{id}/generate-caption", handler.GenerateExportCaption)
		r.Get("/replay-config", handler.GetReplayConfig)
		r.Put("/replay-config", handler.PutReplayConfig)
		r.Delete("/replay-config", handler.DeleteReplayConfig)

		// Scenario routes
		r.Get("/scenarios", handler.ListScenarios)
		r.Get("/scenarios/{name}/port", handler.GetScenarioPort)

		// Recording session profiles
		r.Get("/recordings/sessions", handler.ListRecordingSessionProfiles)
		r.Post("/recordings/sessions", handler.CreateRecordingSessionProfile)
		r.Patch("/recordings/sessions/{profileId}", handler.UpdateRecordingSessionProfile)
		r.Delete("/recordings/sessions/{profileId}", handler.DeleteRecordingSessionProfile)
		r.Get("/recordings/sessions/{profileId}/storage", handler.GetStorageState)
		r.Delete("/recordings/sessions/{profileId}/storage", handler.ClearAllStorage)
		r.Delete("/recordings/sessions/{profileId}/storage/cookies", handler.ClearAllCookies)
		r.Delete("/recordings/sessions/{profileId}/storage/cookies/{domain}", handler.DeleteCookiesByDomain)
		r.Delete("/recordings/sessions/{profileId}/storage/cookies/{domain}/{name}", handler.DeleteCookie)
		r.Delete("/recordings/sessions/{profileId}/storage/origins", handler.ClearAllLocalStorage)
		r.Delete("/recordings/sessions/{profileId}/storage/origins/{origin}", handler.DeleteLocalStorageByOrigin)
		r.Delete("/recordings/sessions/{profileId}/storage/origins/{origin}/{name}", handler.DeleteLocalStorageItem)

		// Service worker management (live session)
		r.Get("/recordings/sessions/{profileId}/service-workers", handler.GetServiceWorkers)
		r.Delete("/recordings/sessions/{profileId}/service-workers", handler.ClearAllServiceWorkers)
		r.Delete("/recordings/sessions/{profileId}/service-workers/{scopeURL}", handler.DeleteServiceWorker)

		// Screenshot serving routes
		r.Get("/screenshots/*", handler.ServeScreenshot)
		r.Get("/screenshots/thumbnail/*", handler.ServeThumbnail)
		r.Get("/artifacts/*", handler.ServeArtifact)

		// Preview route for taking screenshots of URLs
		// POST /api/v1/preview-screenshot
		r.Post("/preview-screenshot", handler.TakePreviewScreenshot)

		// Link preview route for fetching OpenGraph metadata
		// GET /api/v1/link-preview?url=<encoded-url>
		r.Get("/link-preview", handler.GetLinkPreview)

		// Element analysis route for intelligent selector detection
		r.Post("/analyze-elements", handler.AnalyzeElements)

		// Element at coordinate route for click-based selector detection
		r.Post("/element-at-coordinate", handler.GetElementAtCoordinate)

		// AI-powered element analysis route using Ollama text models with DOM
		r.Post("/ai-analyze-elements", handler.AIAnalyzeElements)

		// AI Vision Navigation routes
		r.Post("/ai-navigate", handler.AINavigate)
		r.Get("/ai-navigate/{navigationId}/status", handler.AINavigateStatus)
		r.Post("/ai-navigate/{navigationId}/abort", handler.AINavigateAbort)
		r.Post("/ai-navigate/{navigationId}/resume", handler.AINavigateResume)

		// Internal callback route for playwright-driver step events
		r.Post("/internal/ai-navigate/callback", handler.AINavigateCallback)

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
		r.Post("/recordings/live/{sessionId}/frame", handler.ReceiveRecordingFrame)   // Callback for driver frame streaming
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

		// Multi-tab/page support endpoints
		r.Get("/recordings/live/{sessionId}/pages", handler.GetRecordingPages)
		r.Post("/recordings/live/{sessionId}/pages", handler.CreateRecordingPage)
		r.Post("/recordings/live/{sessionId}/pages/{pageId}/activate", handler.ActivateRecordingPage)
		r.Post("/recordings/live/{sessionId}/pages/{pageId}/close", handler.CloseRecordingPage)
		r.Post("/recordings/live/{sessionId}/page-event", handler.ReceivePageEvent) // Callback for driver page events
		r.Get("/recordings/live/{sessionId}/timeline", handler.GetRecordingTimeline)

		// DOM tree extraction for Browser Inspector tab
		r.Post("/dom-tree", handler.GetDOMTree)

		// Filesystem helper routes (UI support utilities)
		r.Post("/fs/list-directories", handler.ListDirectories)

		// Entitlement routes for subscription management
		r.Get("/entitlement/status", entitlementHandler.GetEntitlementStatus)
		r.Get("/entitlement/identity", entitlementHandler.GetUserIdentity)
		r.Post("/entitlement/identity", entitlementHandler.SetUserIdentity)
		r.Delete("/entitlement/identity", entitlementHandler.ClearUserIdentity)
		r.Get("/entitlement/usage", entitlementHandler.GetUsageSummary)
		r.Post("/entitlement/refresh", entitlementHandler.RefreshEntitlement)
		r.Get("/entitlement/override", entitlementHandler.GetEntitlementOverride)
		r.Post("/entitlement/override", entitlementHandler.SetEntitlementOverride)
		r.Delete("/entitlement/override", entitlementHandler.ClearEntitlementOverride)

		// UX metrics routes (Pro tier and above)
		r.Get("/executions/{id}/ux-metrics", uxHandler.GetExecutionMetrics)
		r.Get("/executions/{id}/ux-metrics/steps/{stepIndex}", uxHandler.GetStepMetrics)
		r.Post("/executions/{id}/ux-metrics/compute", uxHandler.ComputeMetrics)
		r.Get("/workflows/{id}/ux-metrics/aggregate", uxHandler.GetWorkflowMetricsAggregate)

		// Schedule management routes
		r.Post("/workflows/{workflowID}/schedules", handler.CreateSchedule)
		r.Get("/workflows/{workflowID}/schedules", handler.ListWorkflowSchedules)
		r.Get("/schedules", handler.ListAllSchedules)
		r.Get("/schedules/occurrences", handler.GetScheduleOccurrences)
		r.Get("/schedules/{scheduleID}", handler.GetSchedule)
		r.Patch("/schedules/{scheduleID}", handler.UpdateSchedule)
		r.Delete("/schedules/{scheduleID}", handler.DeleteSchedule)
		r.Post("/schedules/{scheduleID}/trigger", handler.TriggerSchedule)
		r.Post("/schedules/{scheduleID}/toggle", handler.ToggleSchedule)

		// Observability routes (proxied to playwright-driver)
		r.Get("/observability", handler.GetObservability)
		r.Post("/observability/refresh", handler.RefreshObservability)
		r.Post("/observability/diagnostics/run", handler.RunDiagnostics)
	})

	// Initialize and start the workflow scheduler
	// The scheduler loads active schedules from the database and triggers workflow executions
	// at the configured cron times
	scheduleNotifier := scheduler.NewWSNotifier(hub, log)
	schedulerSvc := scheduler.New(repo, handler.GetExecutionService(), scheduleNotifier, log)
	if err := schedulerSvc.Start(); err != nil {
		log.WithError(err).Warn("⚠️  Scheduler failed to start - scheduled workflows will not run")
	} else {
		log.WithField("scheduled_count", schedulerSvc.RegisteredCount()).Info("✅ Scheduler service started")
		// Scheduler is stopped during graceful shutdown signal handling
	}

	// Register debug performance endpoints (when enabled in config)
	if cfg.Performance.ExposeEndpoint {
		perfEndpoints := performance.NewEndpoints(handler.GetPerfRegistry(), log)
		perfEndpoints.RegisterRoutes(r)
		log.Info("✅ Debug performance endpoints registered at /debug/performance")
	}

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

	log.WithFields(logrus.Fields{
		"api_host":    apiHost,
		"cors_policy": corsPolicy,
	}).Info("🚀 Vrooli Ascension API starting")

	// Start server with graceful shutdown
	// WriteTimeout is extended to allow long-running automation requests
	if err := server.Run(server.Config{
		Handler:      r,
		WriteTimeout: globalRequestTimeout + 30*time.Second,
		Cleanup: func(ctx context.Context) error {
			// Stop the scheduler first to prevent new executions
			if schedulerSvc != nil {
				log.Info("Stopping scheduler...")
				if err := schedulerSvc.Stop(); err != nil {
					log.WithError(err).Error("Failed to stop scheduler cleanly")
				}
			}
			// Stop sidecar (stops playwright-driver and health monitor)
			if sidecarDeps != nil && sidecarDeps.IsEnabled() {
				log.Info("Stopping sidecar...")
				if err := sidecarDeps.Stop(ctx); err != nil {
					log.WithError(err).Error("Failed to stop sidecar cleanly")
				}
			}
			// Close database connection
			if db != nil {
				db.Close()
			}
			log.Info("✅ Server stopped gracefully")
			return nil
		},
	}); err != nil {
		log.WithError(err).Fatal("Server error")
	}
}

// performStartupHealthCheck validates critical dependencies are available before accepting requests.
// This catches configuration issues early rather than having workflows fail at runtime.
func performStartupHealthCheck(log *logrus.Logger) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var errors []string

	// Check 1: Playwright driver health
	playwrightURL := os.Getenv(driver.PlaywrightDriverEnv)
	if playwrightURL == "" {
		playwrightURL = driver.DefaultDriverURL
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
