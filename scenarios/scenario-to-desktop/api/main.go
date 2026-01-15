package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"

	"scenario-to-desktop-api/agentmanager"
	"scenario-to-desktop-api/build"
	"scenario-to-desktop-api/bundle"
	"scenario-to-desktop-api/distribution"
	"scenario-to-desktop-api/generation"
	"scenario-to-desktop-api/persistence"
	"scenario-to-desktop-api/pipeline"
	preflightdomain "scenario-to-desktop-api/preflight"
	"scenario-to-desktop-api/records"
	"scenario-to-desktop-api/scenario"
	httputil "scenario-to-desktop-api/shared/http"
	"scenario-to-desktop-api/signing"
	"scenario-to-desktop-api/smoketest"
	"scenario-to-desktop-api/state"
	"scenario-to-desktop-api/system"
	"scenario-to-desktop-api/tasks"
	"scenario-to-desktop-api/telemetry"
	"scenario-to-desktop-api/toolexecution"
	"scenario-to-desktop-api/toolhandlers"
	"scenario-to-desktop-api/toolregistry"
)

// Global logger for middleware and initialization code
var globalLogger *slog.Logger

func init() {
	// Initialize global structured logger with JSON output
	globalLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// Server represents the API server
type Server struct {
	router      *mux.Router
	port        int
	templateDir string
	logger      *slog.Logger

	// Domain handlers (screaming architecture)
	buildHandler        *build.Handler
	telemetryHandler    *telemetry.Handler
	recordsHandler      *records.Handler
	scenarioHandler     *scenario.Handler
	systemHandler       *system.Handler
	pipelineHandler     *pipeline.Handler
	stateHandler        *state.Handler
	distributionHandler *distribution.Handler

	// Tool Discovery and Execution Protocol handlers
	toolsHandler         *toolhandlers.ToolsHandler
	toolExecutionHandler *toolexecution.Handler

	// Task orchestration service
	taskSvc *tasks.Service
}

// NewServer creates a new server instance
func NewServer(port int) *Server {
	vrooliRoot := detectVrooliRoot()
	scenarioRoot := filepath.Join(vrooliRoot, "scenarios")
	dataDir := filepath.Join(vrooliRoot, "scenarios", "scenario-to-desktop", "data")
	templateDir := "../templates" // Templates are in parent directory when running from api/

	// Initialize structured logger with JSON output
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// ===== Domain Services (Screaming Architecture) =====

	// Preflight domain (service used by pipeline stage)
	preflightService := preflightdomain.NewService()
	preflightService.StartJanitor()

	// Build domain
	buildStore := build.NewStore()
	buildService := build.NewService(build.WithStore(buildStore))
	buildHandler := build.NewHandler(buildService, buildStore,
		build.WithScenarioRoot(scenarioRoot),
		build.WithHandlerLogger(logger),
	)

	// Bundle domain (packager used by pipeline's BundleStage)
	bundlePackager := bundle.NewPackager()

	// Records domain (created before generation since generation uses recordDeleter)
	recordsStore, err := records.NewFileStore(filepath.Join(dataDir, "desktop_records_v2.json"))
	if err != nil {
		logger.Warn("domain records store unavailable, using nil", "error", err)
		recordsStore = nil
	}
	recordsHandler := records.NewHandler(recordsStore, nil, logger,
		records.WithScenarioRoot(scenarioRoot),
	)

	// Generation domain (service used by pipeline stage)
	generationBuildStore := &generationBuildStoreAdapter{store: buildStore}
	generationService := generation.NewService(
		generation.WithVrooliRoot(vrooliRoot),
		generation.WithTemplateDir(templateDir),
		generation.WithBuildStore(generationBuildStore),
		generation.WithLogger(logger),
	)

	// Smoke test domain
	smokeTestStore, err := smoketest.NewStore(filepath.Join(dataDir, "smoke_tests_v2.json"))
	if err != nil {
		logger.Warn("domain smoke test store unavailable, using in-memory", "error", err)
		smokeTestStore = smoketest.NewInMemoryStore()
	}
	cancelManager := smoketest.NewCancelManager()
	smokeTestService := smoketest.NewService(
		smoketest.WithStore(smokeTestStore),
		smoketest.WithCancelManager(cancelManager),
		smoketest.WithPort(port),
	)

	// Telemetry domain
	telemetryService := telemetry.NewService(vrooliRoot)
	telemetryHandler := telemetry.NewHandler(telemetryService)

	// Scenario domain
	scenarioHandler := scenario.NewHandler(vrooliRoot, nil, logger)

	// State domain (scenario state persistence)
	stateStore, err := state.NewStore(state.DefaultDataDir())
	if err != nil {
		logger.Warn("state store unavailable, using nil", "error", err)
		stateStore = nil
	}
	var stateHandler *state.Handler
	if stateStore != nil {
		stateService := state.NewService(stateStore, logger)
		stateHandler = state.NewHandler(stateService)
	}

	// System domain (wine service)
	wineService := system.NewWineService(logger)
	systemBuildStore := &systemBuildStoreAdapter{store: buildStore}
	systemHandler := system.NewHandler(wineService, systemBuildStore, templateDir)

	// Distribution domain (S3/R2 uploads)
	distributionRepo := distribution.NewGlobalRepository(
		distribution.WithVrooliRoot(vrooliRoot),
	)
	distributionStore := distribution.NewInMemoryStore()
	distributionService := distribution.NewService(
		distribution.WithRepository(distributionRepo),
		distribution.WithStore(distributionStore),
		distribution.WithLogger(logger),
	)
	distributionHandler := distribution.NewHandler(
		distribution.WithHandlerService(distributionService),
		distribution.WithHandlerRepository(distributionRepo),
		distribution.WithHandlerStore(distributionStore),
		distribution.WithHandlerLogger(logger),
	)

	// Pipeline orchestrator - wire up all stages with their dependencies
	// Create scenario analyzer for generation stage
	scenarioAnalyzer := generation.NewAnalyzer(vrooliRoot)

	// Create manifest generator for on-demand bundle manifest creation
	manifestGenerator := pipeline.NewDeploymentManagerGenerator(
		pipeline.WithGeneratorLogger(&pipeline.SlogLogger{Logger: logger}),
	)

	// Create pipeline stages with their service dependencies
	pipelineStages := []pipeline.Stage{
		pipeline.NewBundleStage(
			pipeline.WithScenarioRoot(scenarioRoot),
			pipeline.WithBundlePackager(bundlePackager),
			pipeline.WithManifestGenerator(manifestGenerator),
		),
		pipeline.NewPreflightStage(
			pipeline.WithPreflightService(preflightService),
		),
		pipeline.NewGenerateStage(
			pipeline.WithGenerateScenarioRoot(scenarioRoot),
			pipeline.WithGenerateService(generationService),
			pipeline.WithScenarioAnalyzer(scenarioAnalyzer),
		),
		pipeline.NewBuildStage(
			pipeline.WithBuildService(buildService),
			pipeline.WithBuildStore(buildStore),
		),
		pipeline.NewDistributionStage(
			pipeline.WithDistributionService(distributionService),
			pipeline.WithDistributionStore(distributionStore),
		),
		pipeline.NewSmokeTestStage(
			pipeline.WithSmokeTestService(smokeTestService),
			pipeline.WithSmokeTestStore(smokeTestStore),
		),
	}

	// Create file-backed pipeline store for persistence across restarts
	pipelineDataDir := filepath.Join(dataDir, "pipelines")
	pipelineStore, err := pipeline.NewFileStore(pipelineDataDir,
		pipeline.WithFileStoreLogger(&pipeline.SlogLogger{Logger: logger}),
	)
	if err != nil {
		logger.Warn("pipeline file store unavailable, using in-memory", "error", err)
		pipelineStore = nil
	}

	// Create orchestrator with optional file store
	orchestratorOpts := []pipeline.OrchestratorOption{
		pipeline.WithOrchestratorScenarioRoot(scenarioRoot),
		pipeline.WithLogger(&pipeline.SlogLogger{Logger: logger}),
		pipeline.WithStages(pipelineStages...),
	}
	if pipelineStore != nil {
		orchestratorOpts = append(orchestratorOpts, pipeline.WithStore(pipelineStore))
	}
	pipelineOrchestrator := pipeline.NewOrchestrator(orchestratorOpts...)
	pipelineHandler := pipeline.NewHandler(
		pipeline.WithOrchestrator(pipelineOrchestrator),
	)

	// ===== Tool Discovery and Execution Protocol =====

	// Initialize tool registry with scenario metadata
	toolReg := toolregistry.NewRegistry(toolregistry.RegistryConfig{
		ScenarioName:        "scenario-to-desktop",
		ScenarioVersion:     "1.0.0",
		ScenarioDescription: "Desktop application packaging, signing, and distribution",
	})

	// Register tool providers (pipeline tools plus signing, distribution, and inspection)
	toolReg.RegisterProvider(toolregistry.NewPipelineToolProvider())
	toolReg.RegisterProvider(toolregistry.NewSigningToolProvider())
	toolReg.RegisterProvider(toolregistry.NewDistributionToolProvider())
	toolReg.RegisterProvider(toolregistry.NewInspectionToolProvider())

	// Create tool discovery handler
	toolsHandler := toolhandlers.NewToolsHandler(toolReg)

	// Create build store adapter for tool execution
	toolBuildStore := &toolBuildStoreAdapter{store: buildStore}

	// Create tool executor with service dependencies
	toolExecutor := toolexecution.NewServerExecutor(toolexecution.ServerExecutorConfig{
		BuildStore: toolBuildStore,
		VrooliRoot: vrooliRoot,
		Logger:     logger,
		// Other services can be wired up as adapters are created
	})

	// Create tool execution handler
	toolExecutionHandler := toolexecution.NewHandler(toolExecutor)

	logger.Info("tool protocol initialized",
		"providers", toolReg.ProviderCount(),
		"tools", toolReg.ToolCount(context.Background()))

	// ===== Task Orchestration Service =====

	// Initialize investigation store for task persistence
	invStore := persistence.NewInvestigationStore(filepath.Join(dataDir, "investigations"))

	// Initialize agent manager service (optional - may not be available)
	agentSvcEnabled := os.Getenv("AGENT_MANAGER_ENABLED") != "false"
	var taskSvc *tasks.Service
	if agentSvcEnabled {
		agentSvc := agentmanager.NewAgentService(agentmanager.AgentServiceConfig{
			ProfileName: "scenario-to-desktop",
			ProfileKey:  "scenario-to-desktop",
			Timeout:     30 * time.Second,
			Enabled:     true,
		})

		// Initialize agent profile (best effort - don't fail if unavailable)
		initCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := agentSvc.Initialize(initCtx, agentmanager.DefaultProfileConfig()); err != nil {
			logger.Warn("failed to initialize agent-manager profile", "error", err)
		}
		cancel()

		// Create pipeline store adapter for task service
		// Uses the pipeline orchestrator as the single source of truth for pipeline status
		pipelineStore := &pipelineStoreAdapter{store: pipelineOrchestrator}

		// Create task service (nil progress hub for now - can add WebSocket later)
		taskSvc = tasks.NewService(invStore, pipelineStore, agentSvc, nil)
	}

	// ===== Create Server =====

	srv := &Server{
		router:      mux.NewRouter(),
		port:        port,
		templateDir: templateDir,
		logger:      logger,

		// Domain handlers
		buildHandler:        buildHandler,
		telemetryHandler:    telemetryHandler,
		recordsHandler:      recordsHandler,
		scenarioHandler:     scenarioHandler,
		systemHandler:       systemHandler,
		pipelineHandler:     pipelineHandler,
		stateHandler:        stateHandler,
		distributionHandler: distributionHandler,

		// Tool Protocol handlers
		toolsHandler:         toolsHandler,
		toolExecutionHandler: toolExecutionHandler,

		// Task orchestration
		taskSvc: taskSvc,
	}
	srv.setupRoutes()
	return srv
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health check - use api-core/health for standardized response
	healthHandler := health.New("scenario-to-desktop-api").
		Version("1.0.0").
		Check(health.CheckerFunc(func(ctx context.Context) health.CheckResult {
			return health.CheckResult{
				Name:      "database",
				Connected: true,
			}
		}), health.Optional).
		Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")

	// ===== Domain Handlers (Screaming Architecture) =====
	// Note: Preflight and Generation are now pipeline-only (no direct routes)

	// Build domain: /api/v1/desktop/download/*, /api/v1/desktop/webhook/*
	s.buildHandler.RegisterRoutes(s.router)

	// Telemetry domain: /api/v1/deployment/telemetry*
	s.telemetryHandler.RegisterRoutes(s.router)

	// Records domain: /api/v1/desktop/records*
	s.recordsHandler.RegisterRoutes(s.router)

	// Scenario domain: /api/v1/scenarios/desktop-status
	s.scenarioHandler.RegisterRoutes(s.router)

	// State domain: /api/v1/scenarios/{scenario}/state*
	if s.stateHandler != nil {
		s.stateHandler.RegisterRoutes(s.router)
	}

	// System domain: /api/v1/status, /api/v1/templates*, /api/v1/system/wine/*
	s.systemHandler.RegisterRoutes(s.router)

	// Signing domain: /api/v1/signing/*
	signingHandler := signing.NewHandler()
	signingHandler.RegisterRoutes(s.router)

	// Distribution domain: /api/v1/distribution/*
	s.distributionHandler.RegisterRoutes(s.router)

	// Pipeline orchestration - one-button deployment: /api/v1/pipeline/*
	s.pipelineHandler.RegisterRoutes(s.router)

	// Task orchestration - agent spawning for pipeline investigations
	s.registerTaskRoutes()

	// ===== Tool Discovery and Execution Protocol =====
	// GET /api/v1/tools - Returns complete tool manifest
	// GET /api/v1/tools/{name} - Returns specific tool definition
	// POST /api/v1/tools/execute - Execute a tool
	s.toolsHandler.RegisterRoutes(s.router)
	s.router.HandleFunc("/api/v1/tools/execute", s.toolExecutionHandler.Execute).Methods("POST", "OPTIONS")

	// ===== Legacy Routes (Not Yet Fully Migrated) =====
	// These handlers remain on Server struct until they're migrated to domain modules

	// Probe and proxy utilities
	s.router.HandleFunc("/api/v1/desktop/probe", s.probeEndpointsHandler).Methods("POST")
	s.router.HandleFunc("/api/v1/desktop/proxy-hints/{scenario_name}", s.proxyHintsHandler).Methods("GET")

	// Port resolution
	s.router.HandleFunc("/api/v1/ports/{scenario}/{port_name}", s.getScenarioPortHandler).Methods("GET")

	// Docs
	s.router.HandleFunc("/api/v1/docs/manifest", s.docsManifestHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/docs/content", s.docsContentHandler).Methods("GET")
	s.router.HandleFunc("/docs/{docPath:.*}", s.docsFileHandler).Methods("GET")

	// Icon preview
	s.router.HandleFunc("/api/v1/icons/preview", s.iconPreviewHandler).Methods("GET")

	// Setup middleware - CORS must be registered before logging to handle OPTIONS requests correctly
	s.router.Use(httputil.CORSMiddlewareFromEnv(s.logger))
	s.router.Use(httputil.LoggingMiddlewareStdout())
}

// Router returns the HTTP handler for use with server.Run
func (s *Server) Router() http.Handler {
	s.logger.Info("initializing server",
		"service", "scenario-to-desktop-api",
		"port", s.port,
		"endpoints", []string{"/api/v1/health", "/api/v1/status", "/api/v1/desktop/generate"})
	return handlers.RecoveryHandler()(s.router)
}

// Main function
func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "scenario-to-desktop",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// SECURITY: Validate port environment variables - prefer API_PORT, fallback to PORT
	port := 15200
	apiPortStr := os.Getenv("API_PORT")
	portStr := os.Getenv("PORT")

	if apiPortStr != "" {
		p, err := strconv.Atoi(apiPortStr)
		if err != nil {
			log.Fatalf("❌ Invalid API_PORT value '%s': must be a valid integer", apiPortStr)
		}
		if p < 1024 || p > 65535 {
			log.Fatalf("❌ Invalid API_PORT value %d: must be between 1024 and 65535", p)
		}
		port = p
	} else if portStr != "" {
		// Fallback to PORT for compatibility
		p, err := strconv.Atoi(portStr)
		if err != nil {
			log.Fatalf("❌ Invalid PORT value '%s': must be a valid integer", portStr)
		}
		if p < 1024 || p > 65535 {
			log.Fatalf("❌ Invalid PORT value %d: must be between 1024 and 65535", p)
		}
		port = p
	} else {
		globalLogger.Warn("no port configuration found",
			"message", "No API_PORT or PORT environment variable set",
			"action", "using default port",
			"default_port", port)
	}

	// Create server
	srv := NewServer(port)

	// Start with graceful shutdown via api-core
	if err := server.Run(server.Config{
		Handler: srv.Router(),
		Port:    strconv.Itoa(port),
	}); err != nil {
		globalLogger.Error("server failed", "error", err)
		log.Fatal(err)
	}
}
