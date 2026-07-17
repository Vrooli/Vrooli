package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/handlers"
	aiserviceconnect "github.com/vrooli/browser-automation-studio/handlers/ai_service"
	captureconnect "github.com/vrooli/browser-automation-studio/handlers/capture"
	entitlementconnect "github.com/vrooli/browser-automation-studio/handlers/entitlement"
	executionsconnect "github.com/vrooli/browser-automation-studio/handlers/executions"
	exportsserviceconnect "github.com/vrooli/browser-automation-studio/handlers/exports_service"
	observabilityconnect "github.com/vrooli/browser-automation-studio/handlers/observability"
	projectfilesconnect "github.com/vrooli/browser-automation-studio/handlers/project_files"
	projectsconnect "github.com/vrooli/browser-automation-studio/handlers/projects"
	recordingsconnect "github.com/vrooli/browser-automation-studio/handlers/recordings"
	replayconfigconnect "github.com/vrooli/browser-automation-studio/handlers/replay_config"
	scenariosconnect "github.com/vrooli/browser-automation-studio/handlers/scenarios"
	schedulesconnect "github.com/vrooli/browser-automation-studio/handlers/schedules"
	schemaconnect "github.com/vrooli/browser-automation-studio/handlers/schema"
	sessionprofilesconnect "github.com/vrooli/browser-automation-studio/handlers/session_profiles"
	uxmetricsconnect "github.com/vrooli/browser-automation-studio/handlers/uxmetrics"
	visionnavconnect "github.com/vrooli/browser-automation-studio/handlers/vision_navigation"
	workflowsconnect "github.com/vrooli/browser-automation-studio/handlers/workflows"
	"github.com/vrooli/browser-automation-studio/middleware"
	"github.com/vrooli/browser-automation-studio/performance"
	"github.com/vrooli/browser-automation-studio/services/ai"
	"github.com/vrooli/browser-automation-studio/services/credits"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
	"github.com/vrooli/browser-automation-studio/services/recovery"
	"github.com/vrooli/browser-automation-studio/services/retention"
	"github.com/vrooli/browser-automation-studio/services/scheduler"
	"github.com/vrooli/browser-automation-studio/services/testgenie"
	"github.com/vrooli/browser-automation-studio/services/uxmetrics"
	uxanalyzer "github.com/vrooli/browser-automation-studio/services/uxmetrics/analyzer"
	uxrepository "github.com/vrooli/browser-automation-studio/services/uxmetrics/repository"
	"github.com/vrooli/browser-automation-studio/services/vision"
	"github.com/vrooli/browser-automation-studio/sidecar"
	"github.com/vrooli/browser-automation-studio/usecases/import/adapters"
	importassets "github.com/vrooli/browser-automation-studio/usecases/import/assets"
	importprojects "github.com/vrooli/browser-automation-studio/usecases/import/projects"
	importroutines "github.com/vrooli/browser-automation-studio/usecases/import/routines"
	importscan "github.com/vrooli/browser-automation-studio/usecases/import/scan"
	"github.com/vrooli/browser-automation-studio/usecases/import/shared"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"

	// Unified recording service for timeline persistence
	unifiedrecording "github.com/vrooli/browser-automation-studio/services/recording"
	unifiedpersistence "github.com/vrooli/browser-automation-studio/services/recording/persistence"

	repocontract "github.com/vrooli/repo-contract-go"
)

const globalRequestTimeout = 15 * time.Minute

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "browser-automation-studio",
	}) {
		return // Process was re-exec'd after rebuild
	}

	projectRoot, err := resolveProjectRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to determine project root via repo contract: %v\n", err)
		os.Exit(1)
	}

	if err := os.Chdir(projectRoot); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to change to project root directory %s: %v\n", projectRoot, err)
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

	// Initialize import usecase handlers
	fsScanner := shared.NewFilesystemScanner(log)
	projectAdapter := adapters.NewProjectAdapter(repo)
	workflowAdapter := adapters.NewWorkflowAdapter(repo)
	assetAdapter := adapters.NewAssetAdapter(repo)
	routineImportSvc := importroutines.NewService(fsScanner, workflowAdapter, projectAdapter, log)
	routineImportHandler := importroutines.NewHandler(routineImportSvc, log)
	assetImportSvc := importassets.NewService(fsScanner, projectAdapter, assetAdapter, log)
	assetImportHandler := importassets.NewHandler(assetImportSvc, log)
	scanImportSvc := importscan.NewService(fsScanner, projectAdapter, workflowAdapter, log)
	scanImportHandler := importscan.NewHandler(scanImportSvc, log)

	// Initialize WebSocket hub
	hub := wsHub.NewHub(log)
	go hub.Run()

	// Initialize unified recording service for timeline persistence
	// This service manages all recorded actions and page events across the application
	unifiedRecordingRepo := unifiedpersistence.NewSQLiteRepository(db.RawDB(), log)
	unifiedRecordingSvc := unifiedrecording.NewService(
		unifiedRecordingRepo, hub, log, unifiedrecording.ServiceConfig{},
	)
	log.Info("✅ Unified recording service initialized")

	// Load configuration
	cfg := config.Load()

	// Initialize entitlement service
	entitlementSvc := entitlement.NewService(cfg.Entitlement, log)
	entitlementMiddleware := middleware.NewEntitlementMiddleware(entitlementSvc, log, cfg.Entitlement, repo)
	byokMiddleware := middleware.NewBYOKMiddleware()

	if cfg.Entitlement.ServiceURL != "" {
		log.WithFields(logrus.Fields{
			"service_url":     cfg.Entitlement.ServiceURL,
			"cache_ttl":       cfg.Entitlement.CacheTTL,
			"default_tier":    cfg.Entitlement.DefaultTier,
			"watermark_tiers": cfg.Entitlement.WatermarkTiers,
		}).Info("✅ Entitlement service configured")
	} else {
		log.WithField("default_tier", cfg.Entitlement.DefaultTier).Info("ℹ️  No entitlement service URL - using default tier")
	}

	// Initialize unified credits service (always enabled for tracking)
	creditService := credits.NewService(credits.ServiceOptions{
		DB:             db.RawDB(),
		Logger:         log,
		EntitlementSvc: entitlementSvc,
		// LPBS integration for centralized usage reporting
		LPBSURL:    cfg.Entitlement.ServiceURL,
		LPBSSecret: cfg.Entitlement.ServiceSecret,
	})
	if cfg.Entitlement.ServiceURL != "" && cfg.Entitlement.ServiceSecret != "" {
		log.Info("✅ Unified credits service initialized with LPBS reporting")
	} else {
		log.Info("✅ Unified credits service initialized (LPBS reporting disabled - no secret configured)")
	}

	// Initialize entitlement handler (uses unified credit service)

	// Initialize AI provider chain and factory
	aiProviderChain := ai.NewAIProviderChain(ai.AIProviderChainOptions{
		Logger:        log,
		CreditService: creditService,
		EnableBYOK:    cfg.AIProvider.EnableBYOK,
		EnableVrooli:  cfg.AIProvider.EnableVrooliAPI,
		EnableDevMode: cfg.AIProvider.EnableDevMode,
		VrooliAPIURL:  cfg.AIProvider.VrooliAPIURL,
		DefaultModel:  cfg.AIProvider.DefaultModel,
		Role:          cfg.AIProvider.DefaultRole,
	})
	aiClientFactory := ai.NewAIClientFactory(ai.AIClientFactoryOptions{
		Chain:  aiProviderChain,
		Logger: log,
	})
	log.WithFields(logrus.Fields{
		"byok_enabled":     cfg.AIProvider.EnableBYOK,
		"vrooli_enabled":   cfg.AIProvider.EnableVrooliAPI,
		"dev_mode_enabled": cfg.AIProvider.EnableDevMode,
	}).Info("✅ AI provider chain initialized")

	// Initialize UX metrics repository (used by both handler wiring and API endpoints)
	uxRepo := uxrepository.NewRepository(db.DB)

	// Initialize navigator registry for vision navigation
	navigatorRegistry := vision.NewNavigatorRegistry()

	// Create and register playwright navigator
	playwrightNav := vision.NewPlaywrightVisionNavigator(log,
		vision.WithPlaywrightHub(hub),
		vision.WithPlaywrightCreditService(creditService),
	)
	navigatorRegistry.Register(playwrightNav)

	// Create and register claude code navigator (stub for future use)
	claudeCodeNav := vision.NewClaudeCodeVisionNavigator(log)
	navigatorRegistry.Register(claudeCodeNav)

	log.WithField("navigator_count", navigatorRegistry.Count()).Info("✅ Vision navigator registry initialized")

	// Resolve allowed origins before constructing handlers
	corsCfg := middleware.GetCachedCorsConfig()

	// Initialize handlers with UX metrics integration and entitlement services
	// The UX metrics collector wraps the event sink to passively capture interaction data
	// The entitlement services enable tier-based feature gating and credit tracking
	deps := handlers.InitDefaultDepsWithOptions(repo, hub, log, handlers.DepsOptions{
		UXMetricsRepo:           uxRepo,
		EntitlementService:      entitlementSvc,
		CreditService:           creditService,
		AIClientFactory:         aiClientFactory,
		NavigatorRegistry:       navigatorRegistry,
		PlaywrightNavigator:     playwrightNav,
		UnifiedRecordingService: unifiedRecordingSvc,
	})
	handler := handlers.NewHandlerWithDeps(repo, hub, log, corsCfg.AllowAll, corsCfg.AllowedOrigins, deps)

	// Initialize project import usecase handler (needs deps.CatalogService for workflow sync)
	workflowSyncAdapter := adapters.NewWorkflowSyncAdapter(deps.CatalogService)
	projectImportSvc := importprojects.NewService(fsScanner, projectAdapter, workflowSyncAdapter, log)
	projectImportHandler := importprojects.NewHandler(projectImportSvc, log)

	// Initialize UX metrics service for API endpoints
	// The collector is integrated in the workflow service via InitDefaultDepsWithUXMetrics
	uxAnalyzer := uxanalyzer.NewAnalyzer(uxRepo, nil)
	uxService := uxmetrics.NewService(nil, uxAnalyzer, uxRepo)
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

	// BYOK middleware - extracts OpenRouter API key from request header for AI requests
	r.Use(byokMiddleware.InjectBYOKKey)

	// Correlation ID middleware - generates and propagates correlation IDs for request tracing
	correlationMiddleware := middleware.NewCorrelationMiddleware(log)
	r.Use(correlationMiddleware.InjectCorrelationID)

	// === Connect-RPC services (side-by-side with chi REST routes) ===
	// Each new Connect service appends one line to connectMounts. The
	// chi REST routes below are unaffected; Connect routes get the full
	// middleware stack above. Do not migrate REST endpoints here — that
	// is a separate, per-endpoint decision.
	connectMounts := []connectx.ServiceMount{
		captureconnect.Module(captureconnect.Deps{
			Executor:          deps.ExecutionService,
			Storage:           deps.Storage,
			Resolver:          discovery.NewResolver(discovery.ResolverConfig{}),
			ReadinessResolver: captureconnect.NewReadinessProfileResolver(),
			Logger:            log,
		}),
		scenariosconnect.Module(scenariosconnect.Deps{
			Logger: log,
		}),
		entitlementconnect.Module(entitlementconnect.Deps{
			Provider: entitlementSvc,
			Credits:  creditService,
			Settings: repo,
			Logger:   log,
		}),
		projectfilesconnect.Module(projectfilesconnect.Deps{
			Repo:    repo,
			Catalog: deps.CatalogService,
			Logger:  log,
		}),
		projectsconnect.Module(projectsconnect.Deps{
			Catalog:  deps.CatalogService,
			Executor: deps.ExecutionService,
			Logger:   log,
		}),
		executionsconnect.Module(executionsconnect.Deps{
			Executor:       handler.ExecutionService(),
			SeedScheduler:  handler.SeedCleanupManager(),
			RecordingsRoot: handler.RecordingsRoot(),
			Retention:      retention.NewService(repo, retention.OSFileSystem{}, handler.RecordingsRoot(), log),
			Logger:         log,
		}),
		replayconfigconnect.Module(replayconfigconnect.Deps{
			Store:  repo,
			Logger: log,
		}),
		uxmetricsconnect.Module(uxmetricsconnect.Deps{
			Service: uxService,
			Logger:  log,
		}),
		schedulesconnect.Module(schedulesconnect.Deps{
			Repo:     repo,
			Catalog:  deps.CatalogService,
			Executor: deps.ExecutionService,
			Logger:   log,
		}),
		sessionprofilesconnect.Module(sessionprofilesconnect.Deps{
			Repo:   deps.SessionProfileService,
			Logger: log,
		}),
		recordingsconnect.Module(recordingsconnect.Deps{
			Repo:       deps.SessionProfileService,
			RecordMode: deps.RecordModeService,
			Logger:     log,
		}),
		aiserviceconnect.Module(aiserviceconnect.Deps{
			Screenshot:      handler.ScreenshotHandler(),
			DOM:             handler.DOMHandler(),
			ElementAnalysis: handler.ElementAnalysisHandler(),
			AIAnalysis:      handler.AIAnalysisHandler(),
			LinkPreview: func(ctx context.Context, u string) (*aiserviceconnect.LinkPreviewData, bool, error) {
				preview, found, err := handlers.FetchLinkPreview(ctx, u)
				if err != nil || !found || preview == nil {
					return nil, found, err
				}
				return &aiserviceconnect.LinkPreviewData{
					Title:       preview.Title,
					Description: preview.Description,
					Image:       preview.Image,
					Favicon:     preview.Favicon,
					SiteName:    preview.SiteName,
				}, true, nil
			},
			Logger: log,
		}),
		workflowsconnect.Module(workflowsconnect.Deps{
			Catalog:       deps.CatalogService,
			Executor:      deps.ExecutionService,
			Validator:     workflowsconnect.NewDefaultValidator(handler.WorkflowValidator()),
			SeedRunner:    workflowsconnect.NewDefaultSeedRunner(testgenie.NewClient(nil, nil)),
			SeedScheduler: handler.SeedCleanupManager(),
			CreditService: creditService,
			UserIdentity:  entitlement.UserIdentityFromContext,
			Logger:        log,
		}),
		exportsserviceconnect.Module(exportsserviceconnect.Deps{
			Repo:            repo,
			Executor:        deps.ExecutionService,
			Catalog:         deps.CatalogService,
			AIClientFactory: aiClientFactory,
			Logger:          log,
		}),
		observabilityconnect.Module(observabilityconnect.Deps{
			Proxy:  handler,
			Logger: log,
		}),
		visionnavconnect.Module(visionnavconnect.Deps{
			Logger:   log,
			Registry: navigatorRegistry,
			Credits:  creditService,
			Tracker:  playwrightNav,
		}),
	}
	schemaMount, err := schemaconnect.Module(schemaconnect.Deps{Logger: log})
	if err != nil {
		log.WithError(err).Warn("⚠️  Failed to initialize schema handler; SchemaService disabled")
	} else {
		connectMounts = append(connectMounts, schemaMount)
	}
	connectx.RegisterChi(r, connectMounts...)

	// Routes
	// Health endpoint using api-core/health for standardized response format
	playwrightURL := os.Getenv(driver.PlaywrightDriverEnv)
	if playwrightURL == "" {
		playwrightURL = driver.DefaultDriverURL
	}

	healthHandler := health.New().
		Version("1.0.0").
		Check(health.DB(db.RawDB()), health.Critical).
		Check(health.Func("storage", func(ctx context.Context) error {
			if deps.Storage == nil {
				return health.NewErrorDetail(
					"STORAGE_NOT_INITIALIZED",
					"storage client not initialized - screenshot/artifact storage unavailable",
					"internal",
					false,
				)
			}
			if err := deps.Storage.HealthCheck(ctx); err != nil {
				return health.NewErrorDetail(
					"STORAGE_CONNECTION_ERROR",
					fmt.Sprintf("storage health check failed: %v", err),
					"resource",
					true,
				)
			}
			return nil
		}), health.Optional).
		Check(health.CheckerFunc(func(ctx context.Context) health.CheckResult {
			if deps.CatalogService == nil {
				return health.CheckResult{
					Name:      "automation_engine",
					Connected: false,
					Error: health.NewErrorDetail(
						"AUTOMATION_ENGINE_NOT_INITIALIZED",
						"automation workflow service not initialized",
						"internal",
						false,
					),
				}
			}
			ok, err := deps.CatalogService.CheckAutomationHealth(ctx)
			if err != nil {
				return health.CheckResult{
					Name:      "automation_engine",
					Connected: false,
					Error: health.NewErrorDetail(
						"AUTOMATION_ENGINE_ERROR",
						fmt.Sprintf("automation engine health check failed: %v", err),
						"resource",
						true,
					),
				}
			}
			if !ok {
				return health.CheckResult{
					Name:      "automation_engine",
					Connected: false,
					Error: health.NewErrorDetail(
						"AUTOMATION_ENGINE_UNHEALTHY",
						"automation engine reported unhealthy",
						"resource",
						true,
					),
				}
			}
			return health.CheckResult{
				Name:      "automation_engine",
				Connected: true,
			}
		}), health.Optional).
		Check(health.HTTP("playwright_driver", playwrightURL+"/health"), health.Optional).
		Handler()
	r.Get("/health", healthHandler)
	// RESTException: WebSocket endpoints are not RPC and stay on chi.
	// RESTReason: third_party_shape (browser WebSocket transport + binary
	// playwright-driver frame stream). Tracked in docs/internal/REST_EXCEPTIONS.md.
	r.Get("/ws", handler.HandleWebSocket)                                                 // WebSocket endpoint for browser clients
	r.Get("/ws/recording/{sessionId}/frames", handler.HandleDriverFrameStream)            // WebSocket for playwright-driver binary frame streaming (recording mode)
	r.Get("/ws/execution/{executionId}/frames", handler.HandleDriverExecutionFrameStream) // WebSocket for playwright-driver binary frame streaming (execution mode)

	r.Route("/api/v1", func(r chi.Router) {
		// Health endpoint under /api/v1 for consistency
		r.Get("/health", healthHandler)

		// Project CRUD + project-scoped workflow operations are owned by
		// ProjectsService (Connect-RPC); see handlers/projects/.
		// RESTException: GET /projects/{id}/files/* streams arbitrary file
		// bytes with MIME types decided by extension; consumed by the browser
		// via <img>, <a download>, and file viewers.
		// RESTReason: third_party_shape (browser-native binary streaming).
		// Tracked in docs/internal/REST_EXCEPTIONS.md.
		r.Get("/projects/{id}/files/*", handler.ServeProjectFile)

		// Import usecase routes (project, routine/workflow, and asset import handlers)
		projectImportHandler.RegisterRoutes(r)
		routineImportHandler.RegisterRoutes(r)
		assetImportHandler.RegisterRoutes(r)
		scanImportHandler.RegisterRoutes(r)

		// Workflow routes are served by WorkflowsService via Connect-RPC; see
		// connectMounts above. The legacy REST routes were removed during the
		// Phase 7 Connect-RPC migration.

		// Execution JSON queries / lifecycle controls are served by
		// ExecutionsService via Connect-RPC; see connectMounts above. The
		// legacy REST routes were removed during the Phase 8 migration.
		//
		// Two endpoints intentionally remain on chi REST:
		//   - POST /executions/{id}/export — multipart-ish replay export.
		//     RESTException: third_party_shape — streams binary mp4/gif/webm,
		//     HTML zip bundles, or writes files to a caller-supplied
		//     output_dir on the server filesystem. Not RPC-shaped.
		r.Post("/executions/{id}/export", handler.PostExecutionExport)
		//   - POST /executions/{executionId}/frames — playwright-driver frame
		//     callback. RESTException: webhook_receiver — same shape as the
		//     /internal/history-callback sink; bound to the driver protocol,
		//     not RPC-shaped. Tracked in docs/internal/REST_EXCEPTIONS.md.
		r.Post("/executions/{executionId}/frames", handler.ReceiveExecutionFrame)

		// Export library routes are served by ExportsService via Connect-RPC;
		// see connectMounts above. The legacy REST routes were removed during
		// the Phase 9 proto+Connect migration.
		//
		// One endpoint intentionally remains on chi REST:
		//   - POST /executions/{id}/export — multipart-ish replay export.
		//     RESTException: third_party_shape — streams binary mp4/gif/webm
		//     or writes files to a caller-supplied output_dir on the server
		//     filesystem. Not RPC-shaped. Already declared above.
		// Replay configuration is served via Connect-RPC (ReplayConfigService);
		// see connectMounts above. The legacy REST surface was removed in
		// the Phase 9 proto+Connect migration.

		// Recording session profile CRUD is served via Connect-RPC
		// (SessionProfilesService); the storage/cookies/service-workers/
		// history/tabs sub-resources are served by RecordingsService —
		// see connectMounts above. The legacy REST routes were removed
		// during the Phase 8 proto+Connect migration.

		// Internal callback for history events from playwright driver.
		// RESTException: webhook_receiver — fire-and-forget history-entry sink
		// posted by the out-of-process playwright-driver; bound to the driver
		// protocol, not RPC-shaped. Tracked in docs/internal/REST_EXCEPTIONS.md.
		r.Post("/internal/history-callback", handler.HistoryCallback)

		// Screenshot and artifact binary-serve routes.
		//
		// These endpoints stream image/binary bytes from the storage backend
		// (MinIO) directly to <img src=...> and <a download> consumers in the
		// UI. There is no JSON metadata sibling surface: no list/get/delete
		// metadata operations exist for these resources today (storage paths
		// are addressed by name, not by RPC entity). If/when a metadata
		// catalog is added it will be served via Connect-RPC; the byte-streaming
		// sub-routes below will remain REST.
		//
		// RESTException: third_party_shape — binary media serve, not RPC.
		// Tracked in docs/internal/REST_EXCEPTIONS.md.
		r.Get("/screenshots/*", handler.ServeScreenshot)
		// RESTException: third_party_shape — binary media serve, not RPC.
		r.Get("/screenshots/thumbnail/*", handler.ServeThumbnail)
		// RESTException: third_party_shape — binary media serve, not RPC.
		r.Get("/artifacts/*", handler.ServeArtifact)

		// AI helper routes (preview-screenshot, link-preview, analyze-elements,
		// element-at-coordinate, ai-analyze-elements, dom-tree) are served via
		// Connect-RPC by AIService; see connectMounts above. The legacy REST
		// surface was removed in the Phase 9 proto+Connect migration.

		// AI Vision Navigation is served via Connect-RPC by
		// VisionNavigationService; see connectMounts above. The legacy REST
		// surface (/ai-navigate/*) was removed in the proto+Connect migration.
		//
		// RESTException: webhook_receiver — playwright driver POSTs step and
		// completion events to this endpoint; the driver protocol is not
		// RPC-shaped. Tracked in docs/internal/REST_EXCEPTIONS.md.
		r.Post("/internal/ai-navigate/callback", handler.AINavigateCallback)

		// Recording ingestion and asset serving
		// RESTException: multipart upload of recording archive (zip bytes);
		// proto JSON would force base64 round-trip. Stays REST.
		// RESTReason: multipart_upload. Tracked in docs/internal/REST_EXCEPTIONS.md.
		r.Post("/recordings/import", handler.ImportRecording)
		// RESTException: streams stored recording asset bytes (frames, screenshots,
		// video) from disk directly to <img>/<video>/<a download> consumers in the
		// UI Replay tab and CLI renderer. Path-addressed by storage filename, not
		// by RPC entity.
		// RESTReason: third_party_shape. Tracked in docs/internal/REST_EXCEPTIONS.md.
		r.Get("/recordings/assets/{executionID}/*", handler.ServeRecordingAsset)

		// Live recording routes (Record Mode)
		r.Post("/recordings/live/session", handler.CreateRecordingSession)
		r.Post("/recordings/live/session/{sessionId}/close", handler.CloseRecordingSession)
		r.Post("/recordings/live/start", handler.StartLiveRecording)
		r.Post("/recordings/live/{sessionId}/stop", handler.StopLiveRecording)
		r.Get("/recordings/live/{sessionId}/status", handler.GetRecordingStatus)
		r.Get("/recordings/live/{sessionId}/actions", handler.GetRecordedActions)
		r.Get("/recordings/live/{sessionId}/debug", handler.GetRecordingDebug)
		r.Post("/recordings/live/{sessionId}/action", handler.ReceiveRecordingAction) // Callback for driver action streaming
		r.Post("/recordings/live/{sessionId}/frame", handler.ReceiveRecordingFrame)   // Callback for driver frame streaming
		r.Post("/recordings/live/{sessionId}/navigate", handler.NavigateRecordingSession)
		r.Post("/recordings/live/{sessionId}/reload", handler.ReloadRecordingSession)
		r.Post("/recordings/live/{sessionId}/go-back", handler.GoBackRecordingSession)
		r.Post("/recordings/live/{sessionId}/go-forward", handler.GoForwardRecordingSession)
		r.Get("/recordings/live/{sessionId}/navigation-state", handler.GetNavigationState)
		r.Get("/recordings/live/{sessionId}/navigation-stack", handler.GetNavigationStack)
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

		// DOM tree extraction is served by AIService.GetDOMTree (Connect-RPC).

		// Entitlement routes are served via Connect-RPC (EntitlementService);
		// see connectMounts above. The legacy REST surface was removed in
		// the Phase 4 proto+Connect migration.

		// UX metrics routes are served via Connect-RPC (UXMetricsService);
		// see connectMounts above. The legacy REST surface was removed in
		// the Phase 9 proto+Connect migration.

		// Schedule routes are served by SchedulesService (Connect-RPC).

		// Observability routes are served by ObservabilityService (Connect-RPC).
	})

	// Initialize and start the workflow scheduler
	// The scheduler loads active schedules from the database and triggers workflow executions
	// at the configured cron times
	scheduleNotifier := scheduler.NewWSNotifier(hub, log)
	schedulerSvc := scheduler.New(scheduler.SchedulerOptions{
		Repo:          repo,
		Executor:      handler.GetExecutionService(),
		Notifier:      scheduleNotifier,
		Log:           log,
		CreditService: creditService,
		SettingsRepo:  repo,
	})
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

	var corsPolicy string
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

func resolveProjectRoot() (string, error) {
	if root := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); root != "" {
		return repocontract.FindRepoRootFromPath(root)
	}
	return repocontract.ResolveRepoRoot()
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

// checkPortAvailable checks if a TCP port is available for binding.
// Returns nil if available, or an error with diagnostic hints if unavailable.
func checkPortAvailable(port string) error {
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("invalid port number: %s", port)
	}
	if portNum < 0 || portNum > 65535 {
		return fmt.Errorf("port out of valid range (0-65535): %d", portNum)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		return fmt.Errorf("port %s unavailable (hint: use 'lsof -i :%s' or 'ss -tlnp | grep %s' to find the process)", port, port, port)
	}
	listener.Close()
	return nil
}
