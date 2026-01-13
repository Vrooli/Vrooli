package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"

	"scenario-to-desktop-api/agentmanager"
	"scenario-to-desktop-api/build"
	"scenario-to-desktop-api/bundle"
	"scenario-to-desktop-api/generation"
	"scenario-to-desktop-api/persistence"
	"scenario-to-desktop-api/pipeline"
	preflightdomain "scenario-to-desktop-api/preflight"
	"scenario-to-desktop-api/records"
	"scenario-to-desktop-api/scenario"
	httputil "scenario-to-desktop-api/shared/http"
	"scenario-to-desktop-api/signing"
	"scenario-to-desktop-api/smoketest"
	"scenario-to-desktop-api/system"
	"scenario-to-desktop-api/tasks"
	"scenario-to-desktop-api/telemetry"
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
	preflightHandler  *preflightdomain.Handler
	buildHandler      *build.Handler
	generationHandler *generation.Handler
	smokeTestHandler  *smoketest.Handler
	telemetryHandler  *telemetry.Handler
	recordsHandler    *records.Handler
	scenarioHandler   *scenario.Handler
	systemHandler     *system.Handler
	pipelineHandler   *pipeline.Handler
	bundleHandler     *bundle.Handler

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

	// Preflight domain
	preflightService := preflightdomain.NewService()
	preflightService.StartJanitor()
	preflightHandler := preflightdomain.NewHandler(preflightService)

	// Build domain
	buildStore := build.NewStore()
	buildService := build.NewService(build.WithStore(buildStore))
	buildHandler := build.NewHandler(buildService, buildStore,
		build.WithScenarioRoot(scenarioRoot),
		build.WithHandlerLogger(logger),
	)

	// Bundle domain
	bundlePackager := bundle.NewPackager()
	bundleHandler := bundle.NewHandler(bundlePackager)

	// Records domain (created before generation since generation uses recordDeleter)
	recordsStore, err := records.NewFileStore(filepath.Join(dataDir, "desktop_records_v2.json"))
	if err != nil {
		logger.Warn("domain records store unavailable, using nil", "error", err)
		recordsStore = nil
	}
	recordsHandler := records.NewHandler(recordsStore, nil, logger)

	// Generation domain
	generationBuildStore := &generationBuildStoreAdapter{store: buildStore}
	generationService := generation.NewService(
		generation.WithVrooliRoot(vrooliRoot),
		generation.WithTemplateDir(templateDir),
		generation.WithBuildStore(generationBuildStore),
		generation.WithLogger(logger),
	)
	// Create recordDeleter adapter if store is available
	var recordDeleter generation.RecordDeleter
	if recordsStore != nil {
		recordDeleter = &recordDeleterAdapter{store: recordsStore}
	}
	generationHandler := generation.NewHandler(generationService,
		generation.WithRecordDeleter(recordDeleter),
		generation.WithHandlerLogger(logger),
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
	smokeTestHandler := smoketest.NewHandler(smokeTestService, smokeTestStore, cancelManager,
		smoketest.WithOutputPathFunc(func(scenarioName string) string {
			return filepath.Join(scenarioRoot, scenarioName, "platforms", "electron")
		}),
		smoketest.WithPackageFinder(&defaultSmokeTestPackageFinder{}),
	)

	// Telemetry domain
	telemetryService := telemetry.NewService(vrooliRoot)
	telemetryHandler := telemetry.NewHandler(telemetryService)

	// Scenario domain
	scenarioHandler := scenario.NewHandler(vrooliRoot, nil, logger)

	// System domain (wine service)
	wineService := system.NewWineService(logger)
	systemBuildStore := &systemBuildStoreAdapter{store: buildStore}
	systemHandler := system.NewHandler(wineService, systemBuildStore, templateDir)

	// Pipeline orchestrator - wire up all stages with their dependencies
	// Create scenario analyzer for generation stage
	scenarioAnalyzer := generation.NewAnalyzer(vrooliRoot)

	// Create pipeline stages with their service dependencies
	pipelineStages := []pipeline.Stage{
		pipeline.NewBundleStage(
			pipeline.WithScenarioRoot(scenarioRoot),
			pipeline.WithBundlePackager(bundlePackager),
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
		pipeline.NewSmokeTestStage(
			pipeline.WithSmokeTestService(smokeTestService),
			pipeline.WithSmokeTestStore(smokeTestStore),
		),
	}

	pipelineOrchestrator := pipeline.NewOrchestrator(
		pipeline.WithOrchestratorScenarioRoot(scenarioRoot),
		pipeline.WithLogger(&pipeline.SlogLogger{Logger: logger}),
		pipeline.WithStages(pipelineStages...),
	)
	pipelineHandler := pipeline.NewHandler(
		pipeline.WithOrchestrator(pipelineOrchestrator),
	)

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
		pipelineStoreAdapter := &pipelineStoreAdapter{store: pipelineOrchestrator}

		// Create task service (nil progress hub for now - can add WebSocket later)
		taskSvc = tasks.NewService(invStore, pipelineStoreAdapter, agentSvc, nil)
	}

	// ===== Create Server =====

	srv := &Server{
		router:      mux.NewRouter(),
		port:        port,
		templateDir: templateDir,
		logger:      logger,

		// Domain handlers
		preflightHandler:  preflightHandler,
		buildHandler:      buildHandler,
		generationHandler: generationHandler,
		smokeTestHandler:  smokeTestHandler,
		telemetryHandler:  telemetryHandler,
		recordsHandler:    recordsHandler,
		scenarioHandler:   scenarioHandler,
		systemHandler:     systemHandler,
		pipelineHandler:   pipelineHandler,
		bundleHandler:     bundleHandler,

		// Task orchestration
		taskSvc: taskSvc,
	}
	srv.setupRoutes()
	return srv
}

// recordDeleterAdapter adapts records.FileStore to generation.RecordDeleter interface
type recordDeleterAdapter struct {
	store *records.FileStore
}

func (a *recordDeleterAdapter) DeleteByScenario(scenarioName string) int {
	return a.store.DeleteByScenario(scenarioName)
}

// systemBuildStoreAdapter adapts build.Store to system.BuildStore interface
type systemBuildStoreAdapter struct {
	store *build.InMemoryStore
}

func (a *systemBuildStoreAdapter) Snapshot() map[string]*system.BuildStatus {
	snapshot := a.store.Snapshot()
	result := make(map[string]*system.BuildStatus, len(snapshot))
	for id, status := range snapshot {
		result[id] = &system.BuildStatus{
			Status: status.Status,
		}
	}
	return result
}

// pipelineStoreAdapter adapts pipeline.Orchestrator to tasks.PipelineStore interface
type pipelineStoreAdapter struct {
	store pipeline.Orchestrator
}

func (a *pipelineStoreAdapter) Get(pipelineID string) (*pipeline.Status, bool) {
	return a.store.GetStatus(pipelineID)
}

// generationBuildStoreAdapter adapts build.InMemoryStore to generation.BuildStore interface
type generationBuildStoreAdapter struct {
	store *build.InMemoryStore
}

func (a *generationBuildStoreAdapter) Create(buildID string) *generation.BuildStatus {
	now := time.Now()
	status := &generation.BuildStatus{
		BuildID:   buildID,
		Status:    "building",
		StartedAt: now,
		BuildLog:  []string{},
		ErrorLog:  []string{},
		Artifacts: map[string]string{},
		Metadata:  map[string]interface{}{},
	}
	// Save to underlying store
	a.store.Save(&build.Status{
		BuildID:   buildID,
		Status:    "building",
		CreatedAt: now,
		BuildLog:  []string{},
		ErrorLog:  []string{},
		Artifacts: map[string]string{},
		Metadata:  map[string]interface{}{},
	})
	return status
}

func (a *generationBuildStoreAdapter) Get(buildID string) (*generation.BuildStatus, bool) {
	status, ok := a.store.Get(buildID)
	if !ok {
		return nil, false
	}
	return &generation.BuildStatus{
		BuildID:     status.BuildID,
		Status:      status.Status,
		OutputPath:  status.OutputPath,
		StartedAt:   status.CreatedAt,
		CompletedAt: status.CompletedAt,
		BuildLog:    status.BuildLog,
		ErrorLog:    status.ErrorLog,
		Artifacts:   status.Artifacts,
		Metadata:    status.Metadata,
	}, true
}

func (a *generationBuildStoreAdapter) Update(buildID string, fn func(status *generation.BuildStatus)) {
	a.store.Update(buildID, func(status *build.Status) {
		// Convert to generation.BuildStatus, apply fn, convert back
		genStatus := &generation.BuildStatus{
			BuildID:     status.BuildID,
			Status:      status.Status,
			OutputPath:  status.OutputPath,
			StartedAt:   status.CreatedAt,
			CompletedAt: status.CompletedAt,
			BuildLog:    status.BuildLog,
			ErrorLog:    status.ErrorLog,
			Artifacts:   status.Artifacts,
			Metadata:    status.Metadata,
		}
		fn(genStatus)
		// Copy back relevant fields
		status.Status = genStatus.Status
		status.OutputPath = genStatus.OutputPath
		status.CompletedAt = genStatus.CompletedAt
		status.BuildLog = genStatus.BuildLog
		status.ErrorLog = genStatus.ErrorLog
		status.Artifacts = genStatus.Artifacts
		status.Metadata = genStatus.Metadata
	})
}

// defaultSmokeTestPackageFinder is the default package finder for smoke tests
type defaultSmokeTestPackageFinder struct{}

func (f *defaultSmokeTestPackageFinder) FindBuiltPackage(distPath, platform string) (string, error) {
	return findBuiltPackageStandalone(distPath, platform)
}

// findBuiltPackageStandalone finds the built package file for a specific platform.
// This is a standalone version of the function for use by domain adapters.
func findBuiltPackageStandalone(distPath, platform string) (string, error) {
	// Check if dist-electron directory exists
	if _, err := os.Stat(distPath); os.IsNotExist(err) {
		return "", fmt.Errorf("dist-electron directory not found at %s", distPath)
	}

	// Platform-specific file patterns
	var patterns []string
	switch platform {
	case "win":
		patterns = []string{"*.msi", "*Setup.exe", "*.exe"}
	case "mac":
		patterns = []string{"*.pkg", "*.dmg", "*.zip"}
	case "linux":
		patterns = []string{"*.AppImage", "*.deb"}
	default:
		return "", fmt.Errorf("unknown platform: %s", platform)
	}

	// Search for matching files
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(distPath, pattern))
		if err != nil {
			continue
		}
		if len(matches) > 0 {
			// Return the first match with platform-specific preferences
			if platform == "win" && len(matches) > 1 {
				for _, match := range matches {
					if strings.HasSuffix(strings.ToLower(match), ".msi") {
						return match, nil
					}
				}
				for _, match := range matches {
					if strings.Contains(strings.ToLower(match), "setup") {
						return match, nil
					}
				}
			}
			if platform == "mac" && len(matches) > 1 {
				for _, match := range matches {
					if strings.HasSuffix(strings.ToLower(match), ".pkg") {
						return match, nil
					}
				}
				for _, match := range matches {
					lowerMatch := strings.ToLower(match)
					if !strings.Contains(lowerMatch, "arm64") && !strings.Contains(lowerMatch, "blockmap") {
						return match, nil
					}
				}
				for _, match := range matches {
					if !strings.Contains(strings.ToLower(match), "blockmap") {
						return match, nil
					}
				}
			}
			if platform == "mac" {
				for _, match := range matches {
					if !strings.Contains(strings.ToLower(match), "blockmap") {
						return match, nil
					}
				}
			}
			return matches[0], nil
		}
	}

	return "", fmt.Errorf("no built package found for platform %s in %s", platform, distPath)
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

	// Preflight domain: /api/v1/desktop/preflight*, /api/v1/desktop/bundle-manifest
	s.preflightHandler.RegisterRoutes(s.router)

	// Build domain: /api/v1/desktop/build*, /api/v1/desktop/status/*
	s.buildHandler.RegisterRoutes(s.router)

	// Generation domain: /api/v1/desktop/generate, /api/v1/desktop/generate/quick
	s.generationHandler.RegisterRoutes(s.router)

	// Smoke test domain: /api/v1/desktop/smoke-test/*
	s.smokeTestHandler.RegisterRoutes(s.router)

	// Telemetry domain: /api/v1/deployment/telemetry*
	s.telemetryHandler.RegisterRoutes(s.router)

	// Records domain: /api/v1/desktop/records*
	s.recordsHandler.RegisterRoutes(s.router)

	// Scenario domain: /api/v1/scenarios/desktop-status
	s.scenarioHandler.RegisterRoutes(s.router)

	// System domain: /api/v1/status, /api/v1/templates*, /api/v1/system/wine/*
	s.systemHandler.RegisterRoutes(s.router)

	// Signing domain: /api/v1/signing/*
	signingHandler := signing.NewHandler()
	signingHandler.RegisterRoutes(s.router)

	// Pipeline orchestration - one-button deployment: /api/v1/pipeline/*
	s.pipelineHandler.RegisterRoutes(s.router)

	// Task orchestration - agent spawning for pipeline investigations
	s.registerTaskRoutes()

	// Bundle domain: /api/v1/bundle/package, /api/v1/desktop/package
	s.bundleHandler.RegisterRoutes(s.router)

	// ===== Legacy Routes (Not Yet Fully Migrated) =====
	// These handlers remain on Server struct until they're migrated to domain modules

	// Probe and proxy utilities
	s.router.HandleFunc("/api/v1/desktop/probe", s.probeEndpointsHandler).Methods("POST")
	s.router.HandleFunc("/api/v1/desktop/proxy-hints/{scenario_name}", s.proxyHintsHandler).Methods("GET")

	// Desktop test (different from smoke test - runs validation tests)
	s.router.HandleFunc("/api/v1/desktop/test", s.testDesktopHandler).Methods("POST")

	// Port resolution
	s.router.HandleFunc("/api/v1/ports/{scenario}/{port_name}", s.getScenarioPortHandler).Methods("GET")

	// Deployment-manager integration
	s.router.HandleFunc("/api/v1/deployment-manager/bundles/export", s.exportBundleHandler).Methods("POST")
	s.router.HandleFunc("/api/v1/deployment-manager/build/auto", s.deploymentManagerAutoBuildHandler).Methods("POST")
	s.router.HandleFunc("/api/v1/deployment-manager/build/auto/{build_id}", s.deploymentManagerAutoBuildStatusHandler).Methods("GET")

	// Test artifacts
	s.router.HandleFunc("/api/v1/desktop/test-artifacts", s.listTestArtifactsHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/desktop/test-artifacts/cleanup", s.cleanupTestArtifactsHandler).Methods("POST")

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
