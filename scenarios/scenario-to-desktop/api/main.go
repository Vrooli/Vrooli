// DOC: docs/reference/api-architecture.md
// DOC: docs/internal/SEAMS.md
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"scenario-to-desktop-api/agentmanager"
	"scenario-to-desktop-api/build"
	"scenario-to-desktop-api/bundle"
	"scenario-to-desktop-api/captures"
	"scenario-to-desktop-api/deploy"
	"scenario-to-desktop-api/generation"
	"scenario-to-desktop-api/livedesktop"
	"scenario-to-desktop-api/persistence"
	"scenario-to-desktop-api/pipeline"
	"scenario-to-desktop-api/procmetrics"
	"scenario-to-desktop-api/records"
	"scenario-to-desktop-api/scenario"
	"scenario-to-desktop-api/screenrecording"
	"scenario-to-desktop-api/signing"
	"scenario-to-desktop-api/smoketest"
	"scenario-to-desktop-api/state"
	"scenario-to-desktop-api/storagemigrate"
	"scenario-to-desktop-api/storagepaths"
	"scenario-to-desktop-api/system"
	"scenario-to-desktop-api/tasks"
	"scenario-to-desktop-api/telemetry"
	"strconv"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/apihttp"
	coredb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
	_ "modernc.org/sqlite"

	preflightdomain "scenario-to-desktop-api/preflight"

	httputil "scenario-to-desktop-api/shared/http"
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
	router          *mux.Router
	port            int
	templateDir     string
	logger          *slog.Logger
	lifecycleCancel context.CancelFunc
	routedDB        *coredb.RoutedDB
	fileRoots       *filerouting.RoutedRoots

	// Domain handlers (screaming architecture)
	buildHandler     *build.Handler
	telemetryHandler *telemetry.Handler
	recordsHandler   *records.Handler
	scenarioHandler  *scenario.Handler
	systemHandler    *system.Handler
	pipelineHandler  *pipeline.Handler
	stateHandler     *state.Handler
	deployHandler    *deploy.Handler
	signingHandler   *signing.Handler
	preflightService preflightdomain.Service
	configAnalyzer   generation.ScenarioAnalyzer

	// Task orchestration service
	taskSvc *tasks.Service

	// Live desktop handler
	liveDesktopHandler *livedesktop.Handler
	liveDesktopService *livedesktop.Service

	// Captures handler
	capturesHandler *captures.Handler
	capturesService *captures.Service

	// Smoke test store for video serving
	smokeTestStore   smoketest.Store
	smokeTestService smoketest.Service
	smokeTestCancels smoketest.CancelManager
}

// NewServer creates a new server instance
func NewServer(port int) *Server {
	lifecycleCtx, lifecycleCancel := context.WithCancel(context.Background())
	initialized := false
	defer func() {
		if !initialized {
			lifecycleCancel()
		}
	}()
	vrooliRoot := detectVrooliRoot()
	scenarioRoot := filepath.Join(vrooliRoot, "scenarios")
	templateDir := "../templates" // Templates are in parent directory when running from api/

	// Initialize structured logger with JSON output
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	storePaths, ok := initStoragePaths(logger)
	if !ok {
		return nil
	}
	primaryPaths, err := storePaths.Paths()
	if err != nil {
		logger.Error("failed to resolve primary storage roots", "error", err)
		return nil
	}
	fileRoots := filerouting.New(primaryPaths)
	dbCtx, dbCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer dbCancel()
	routedDB, err := coredb.Open(dbCtx, coredb.Config{
		Driver:       coredb.DriverSQLite,
		DSN:          filepath.Join(primaryPaths.DataDir, "scenario-to-desktop.db"),
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		Logger:       logger.Info,
	})
	if err != nil {
		logger.Error("failed to open routed scenario database", "error", err)
		return nil
	}
	if err := coredb.EnsureSchemas(dbCtx, routedDB.Primary()); err != nil {
		_ = routedDB.Close()
		logger.Error("failed to initialize scenario database schema", "error", err)
		return nil
	}
	routedDB.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		return coredb.EnsureSchemas(ctx, pool)
	})

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
	recordsPath, err := storePaths.RecordsPath()
	if err != nil {
		logger.Warn("records path unavailable", "error", err)
	}
	recordsStore, err := records.NewFileStore(recordsPath)
	if err != nil {
		logger.Warn("domain records store unavailable, using nil", "error", err)
		recordsStore = nil
	}
	recordsHandler := records.NewHandler(recordsStore, &recordsBuildStoreAdapter{store: buildStore}, logger,
		records.WithScenarioRoot(scenarioRoot),
	)

	// Generation domain (service used by pipeline stage)
	generationBuildStore := &generationBuildStoreAdapter{store: buildStore}
	generationRecordStore := &generationRecordStoreAdapter{store: recordsStore}
	generationService := generation.NewService(
		generation.WithVrooliRoot(vrooliRoot),
		generation.WithTemplateDir(templateDir),
		generation.WithBuildStore(generationBuildStore),
		generation.WithLogger(logger),
		generation.WithRecordStore(generationRecordStore),
	)

	// Smoke test domain
	smokeTestsPath, pathErr := storePaths.SmokeTestsPath()
	if pathErr != nil {
		logger.Warn("smoke test path unavailable", "error", pathErr)
	}
	smokeTestStore, err := smoketest.NewStore(smokeTestsPath)
	if err != nil {
		logger.Warn("domain smoke test store unavailable, using in-memory", "error", err)
		smokeTestStore = smoketest.NewInMemoryStore()
	}
	cancelManager := smoketest.NewCancelManager()
	smokeTestLogger := smoketest.NewSlogAdapter(logger)
	smokeTestService := smoketest.NewDefaultSmokeTestService(
		smokeTestStore,
		cancelManager,
		nil, // telemetryIngestor - set later via handler
		port,
		smokeTestLogger,
	)

	// Wire screen recording into smoke test service
	smokeTestExecutor := smoketest.NewProcessExecutor(smokeTestLogger)
	recorder := screenrecording.NewRecorder(&screenrecordingExecutorAdapter{executor: smokeTestExecutor})
	displayMgr := screenrecording.NewDisplayManager()
	smokeTestService.WithRecording(recorder, displayMgr)

	// Process monitoring for app startup time and resource usage
	procReader := &procmetrics.LinuxProcReader{}
	shellFn := procmetrics.ShellFunc(func(ctx context.Context, env []string, name string, args ...string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, name, args...)
		if len(env) > 0 {
			cmd.Env = env
		}
		return cmd.Output()
	})
	windowDetector := procmetrics.NewXdotoolDetector(shellFn, logger)
	monitorFactory := procmetrics.NewDefaultMonitorFactory(procReader, windowDetector, logger)
	smokeTestService.WithMonitor(monitorFactory)

	// Wire smoke test store into records handler for video data enrichment
	recordsHandler.SetSmokeTestStore(&smokeTestRecordAdapter{store: smokeTestStore})

	// Captures domain (persistent screenshot/recording storage)
	capturesService, capturesHandler := initCapturesDomain(storePaths, logger)

	// Live desktop domain (interactive VNC sessions)
	linuxBackend := livedesktop.NewLinuxBackend(logger)
	liveDesktopStore := livedesktop.NewInMemoryStore()
	liveDesktopService := livedesktop.NewService(liveDesktopStore, linuxBackend, logger, vrooliRoot)
	liveDesktopDataDir, err := storePaths.EnsureLiveDesktopDir()
	if err != nil {
		logger.Warn("live desktop directory unavailable", "error", err)
	} else {
		liveDesktopService.WithDataDir(liveDesktopDataDir)
	}
	liveDesktopService.WithRecorder(recorder)
	if capturesService != nil {
		liveDesktopService.WithCaptures(capturesService)
	}
	liveDesktopHandler := livedesktop.NewHandler(liveDesktopService)
	// Start idle session janitor (30s check interval, 30m idle timeout)
	livedesktop.StartJanitor(lifecycleCtx, liveDesktopService, 30*time.Second, 30*time.Minute)

	// Telemetry domain
	telemetryDir, err := storePaths.EnsureTelemetryDir()
	if err != nil {
		logger.Warn("telemetry directory unavailable", "error", err)
	}
	telemetryService := telemetry.NewService(telemetryDir)
	telemetryHandler := telemetry.NewHandler(telemetryService)

	// Scenario domain
	scenarioRecordStore := &scenarioRecordStoreAdapter{store: recordsStore}
	scenarioHandler := scenario.NewHandler(vrooliRoot, scenarioRecordStore, logger)

	// State domain (scenario state persistence)
	stateStore, err := state.NewRoutedStore(fileRoots)
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
	configAnalyzer := generation.NewAnalyzer(vrooliRoot)

	pipelineDeps := pipelineInitDeps{
		scenarioRoot:         scenarioRoot,
		vrooliRoot:           vrooliRoot,
		logger:               logger,
		storePaths:           storePaths,
		preflightService:     preflightService,
		bundlePackager:       bundlePackager,
		generationService:    generationService,
		generationBuildStore: generationBuildStore,
		buildService:         buildService,
		buildStore:           buildStore,
		smokeTestService:     smokeTestService,
		smokeTestStore:       smokeTestStore,
	}
	pipelineOrchestrator, pipelineHandler, deployHandler := initPipelineStack(pipelineDeps)

	// ===== Task Orchestration Service =====
	dataRoot, err := storePaths.DataRoot()
	if err != nil {
		logger.Warn("data root unavailable", "error", err)
	}
	taskSvc := initTaskOrchestration(dataRoot, pipelineOrchestrator, logger, fmt.Sprintf("http://127.0.0.1:%d", port))
	signingHandler := signing.NewHandler()

	// ===== Create Server =====

	srv := &Server{
		router:          mux.NewRouter(),
		port:            port,
		templateDir:     templateDir,
		logger:          logger,
		lifecycleCancel: lifecycleCancel,
		routedDB:        routedDB,
		fileRoots:       fileRoots,

		// Domain handlers
		buildHandler:     buildHandler,
		telemetryHandler: telemetryHandler,
		recordsHandler:   recordsHandler,
		scenarioHandler:  scenarioHandler,
		systemHandler:    systemHandler,
		pipelineHandler:  pipelineHandler,
		stateHandler:     stateHandler,
		deployHandler:    deployHandler,
		signingHandler:   signingHandler,
		preflightService: preflightService,
		configAnalyzer:   configAnalyzer,
		// Live desktop handler
		liveDesktopHandler: liveDesktopHandler,
		liveDesktopService: liveDesktopService,
		// Captures handler
		capturesHandler: capturesHandler,
		capturesService: capturesService,
		// Task orchestration
		taskSvc: taskSvc,

		// Smoke test store for video serving
		smokeTestStore:   smokeTestStore,
		smokeTestService: smokeTestService,
		smokeTestCancels: cancelManager,
	}
	srv.registerDomainHandlers()
	initialized = true
	return srv
}

func initStoragePaths(logger *slog.Logger) (*storagepaths.Locator, bool) {
	paths, err := storagepaths.NewLocator()
	if err != nil {
		logger.Error("failed to initialize storage paths", "error", err)
		return nil, false
	}
	if _, err := paths.EnsureAll(); err != nil {
		logger.Error("failed to prepare storage roots", "error", err)
		return nil, false
	}
	return paths, true
}

// pipelineInitDeps bundles the services required to build the pipeline stack.
type pipelineInitDeps struct {
	scenarioRoot         string
	vrooliRoot           string
	logger               *slog.Logger
	storePaths           *storagepaths.Locator
	preflightService     preflightdomain.Service
	bundlePackager       bundle.Packager
	generationService    generation.Service
	generationBuildStore generation.BuildStore
	buildService         build.Service
	buildStore           build.Store
	smokeTestService     smoketest.Service
	smokeTestStore       smoketest.Store
}

// initPipelineStack wires up the pipeline orchestrator, manager, handler, and the
// deploy handler that shares the deploy-target repository.
func initPipelineStack(deps pipelineInitDeps) (*pipeline.DefaultOrchestrator, *pipeline.Handler, *deploy.Handler) {
	logger := deps.logger
	storePaths := deps.storePaths

	scenarioAnalyzer := generation.NewAnalyzer(deps.vrooliRoot)
	manifestGenerator := pipeline.NewDeploymentManagerGenerator(
		pipeline.WithGeneratorLogger(&pipeline.SlogLogger{Logger: logger}),
	)

	deployTargetsPath, err := storePaths.DeployTargetsPath()
	if err != nil {
		logger.Warn("deploy targets path unavailable", "error", err)
	}
	deployTargetRepo := deploy.NewTargetRepository(deployTargetsPath)

	stages := []pipeline.Stage{
		pipeline.NewResolveDeploymentStage(
			pipeline.WithResolveDeploymentScenarioRoot(deps.scenarioRoot),
		),
		pipeline.NewBundleStage(
			pipeline.WithScenarioRoot(deps.scenarioRoot),
			pipeline.WithBundlePackager(deps.bundlePackager),
			pipeline.WithManifestGenerator(manifestGenerator),
		),
		pipeline.NewPreflightStage(
			pipeline.WithPreflightService(deps.preflightService),
			pipeline.WithBundleabilityChecker(scenarioAnalyzer),
		),
		pipeline.NewGenerateStage(
			pipeline.WithGenerateScenarioRoot(deps.scenarioRoot),
			pipeline.WithGenerateService(deps.generationService),
			pipeline.WithScenarioAnalyzer(scenarioAnalyzer),
			pipeline.WithGenerateBuildStore(deps.generationBuildStore),
		),
		pipeline.NewBuildStage(
			pipeline.WithBuildService(deps.buildService),
			pipeline.WithBuildStore(deps.buildStore),
		),
		pipeline.NewSmokeTestStage(
			pipeline.WithSmokeTestService(deps.smokeTestService),
			pipeline.WithSmokeTestStore(deps.smokeTestStore),
		),
		pipeline.NewDeployStage(
			pipeline.WithDeployTargetRepo(deployTargetRepo),
		),
	}

	pipelineStore := newPipelineFileStore(storePaths, logger)
	indexStore := newPipelineIndexStore(storePaths, logger)

	orchestratorOpts := []pipeline.OrchestratorOption{
		pipeline.WithOrchestratorScenarioRoot(deps.scenarioRoot),
		pipeline.WithLogger(&pipeline.SlogLogger{Logger: logger}),
		pipeline.WithStages(stages...),
	}
	if pipelineStore != nil {
		orchestratorOpts = append(orchestratorOpts, pipeline.WithStore(pipelineStore))
	}
	orchestrator := pipeline.NewOrchestrator(orchestratorOpts...)

	manager := pipeline.NewManager(
		pipeline.WithManagerOrchestrator(orchestrator),
		pipeline.WithManagerIndexStore(indexStore),
		pipeline.WithManagerLogger(&pipeline.SlogLogger{Logger: logger}),
	)
	if n := manager.RecoverStalePipelines(); n > 0 {
		logger.Info("recovered stale pipelines at startup", "count", n)
	}

	handler := pipeline.NewHandler(
		pipeline.WithOrchestrator(orchestrator),
		pipeline.WithManager(manager),
	)
	return orchestrator, handler, deploy.NewHandler(deployTargetRepo)
}

func newPipelineFileStore(storePaths *storagepaths.Locator, logger *slog.Logger) *pipeline.FileStore {
	dataDir, err := storePaths.EnsurePipelineStateDir()
	if err != nil {
		logger.Warn("pipeline storage directory unavailable", "error", err)
	}
	store, err := pipeline.NewFileStore(dataDir,
		pipeline.WithFileStoreLogger(&pipeline.SlogLogger{Logger: logger}),
	)
	if err != nil {
		logger.Warn("pipeline file store unavailable, using in-memory", "error", err)
		return nil
	}
	return store
}

func newPipelineIndexStore(storePaths *storagepaths.Locator, logger *slog.Logger) *pipeline.ScenarioIndexStore {
	dataDir, err := storePaths.EnsurePipelineIndexDir()
	if err != nil {
		logger.Warn("pipeline index directory unavailable", "error", err)
	}
	store, err := pipeline.NewScenarioIndexStore(dataDir,
		pipeline.WithIndexStoreLogger(&pipeline.SlogLogger{Logger: logger}),
	)
	if err != nil {
		logger.Warn("scenario index store unavailable", "error", err)
		return nil
	}
	return store
}

// initCapturesDomain sets up the captures service and handler.
func initCapturesDomain(paths *storagepaths.Locator, logger *slog.Logger) (*captures.Service, *captures.Handler) {
	metaPath, err := paths.CapturesMetaPath()
	if err != nil {
		logger.Warn("captures meta path unavailable", "error", err)
		return nil, nil
	}
	capturesStore, err := captures.NewFileStore(metaPath)
	if err != nil {
		logger.Warn("captures store unavailable", "error", err)
		return nil, nil
	}
	filesDir, err := paths.EnsureCapturesDir()
	if err != nil {
		logger.Warn("captures files directory unavailable", "error", err)
		return nil, nil
	}
	svc := captures.NewService(paths.Resolver(), paths.Options(), filesDir, capturesStore)
	logger.Info("captures service initialized", "meta_path", metaPath)
	return svc, captures.NewHandler(svc)
}

// initTaskOrchestration sets up the task orchestration service with agent manager integration.
func initTaskOrchestration(dataDir string, pipelineOrchestrator *pipeline.DefaultOrchestrator, logger *slog.Logger, pipelineAPIURL string) *tasks.Service {
	if os.Getenv("AGENT_MANAGER_ENABLED") == "false" {
		return nil
	}
	invStore := persistence.NewInvestigationStore(filepath.Join(dataDir, "investigations"))
	agentSvc := agentmanager.NewAgentService(agentmanager.AgentServiceConfig{
		ProfileName: "scenario-to-desktop",
		ProfileKey:  "scenario-to-desktop/default",
		Timeout:     30 * time.Second,
		Enabled:     true,
	})

	initCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := agentSvc.Initialize(initCtx); err != nil {
		logger.Warn("failed to initialize agent-manager profile", "error", err)
	}
	cancel()

	pipelineStore := &pipelineStoreAdapter{store: pipelineOrchestrator}
	return tasks.NewService(invStore, pipelineStore, agentSvc, nil, pipelineAPIURL)
}

// registerDomainHandlers configures all API routes organized by domain.
// This follows the "screaming architecture" principle where the structure
// of the code screams about its purpose.
func (s *Server) registerDomainHandlers() {
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
	s.registerConnectHandlers()

	// ===== Domain Handlers (Screaming Architecture) =====
	// Note: Preflight and Generation are now pipeline-only (no direct routes)

	// Build domain: /api/v1/desktop/download/*, /api/v1/desktop/webhook/*
	s.buildHandler.RegisterRoutes(s.router)

	// Telemetry raw export remains HTTP; all domain operations are Connect RPCs.
	s.telemetryHandler.RegisterDownloadRoute(s.router)

	// Signing is served exclusively by the generated SigningService.

	// Pipeline orchestration is served exclusively by the generated PipelineService.

	// Live desktop: /api/v1/livedesktop/*
	if s.liveDesktopHandler != nil {
		s.liveDesktopHandler.RegisterRoutes(s.router)
	}

	// Captures: /api/v1/captures/*
	if s.capturesHandler != nil {
		s.capturesHandler.RegisterRoutes(s.router)
	}

	// Task orchestration - agent spawning for pipeline investigations

	// ===== Legacy Routes (Not Yet Fully Migrated) =====
	// These handlers remain on Server struct until they're migrated to domain modules

	// Docs
	s.router.HandleFunc("/api/v1/docs/manifest", s.docsManifestHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/docs/content", s.docsContentHandler).Methods("GET")
	s.router.HandleFunc("/docs/{docPath:.*}", s.docsFileHandler).Methods("GET")

	// Smoke test video serving
	s.router.HandleFunc("/api/v1/smoketest/{id}/video", s.smokeTestVideoHandler).Methods("GET")

	// Icon preview
	s.router.HandleFunc("/api/v1/icons/preview", s.iconPreviewHandler).Methods("GET")

	// Setup middleware - CORS must be registered before logging to handle OPTIONS requests correctly
	s.router.Use(httputil.SecurityHeaders())
	s.router.Use(httputil.CORSMiddlewareFromEnv(s.logger))
	s.router.Use(httputil.LoggingMiddlewareStdout())
}

// Router returns the HTTP handler for use with server.Run
func (s *Server) Router() http.Handler {
	s.logger.Info("initializing server",
		"service", "scenario-to-desktop-api",
		"port", s.port,
		"endpoints", []string{"/api/v1/health"})
	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, s.routedDB, s.fileRoots)
	rootMux.Handle("/", handlers.RecoveryHandler()(s.router))
	return apihttp.TestModeMiddleware(rootMux)
}

// Shutdown stops scenario-owned background work after the HTTP server has
// stopped accepting requests. It is safe to call more than once.
func (s *Server) Shutdown(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if s.lifecycleCancel != nil {
		s.lifecycleCancel()
	}
	var shutdownErr error
	if preflightJanitor, ok := s.preflightService.(interface {
		StopJanitor(context.Context) error
	}); ok {
		if err := preflightJanitor.StopJanitor(ctx); err != nil {
			shutdownErr = errors.Join(shutdownErr, fmt.Errorf("stop preflight janitor: %w", err))
		}
	}

	if s.liveDesktopService != nil {
		for _, session := range s.liveDesktopService.ListSessions() {
			if err := s.liveDesktopService.StopSession(session.ID); err != nil {
				s.logger.Warn("failed to stop live desktop session during shutdown", "session_id", session.ID, "error", err)
			}
		}
	}
	if s.taskSvc != nil {
		if err := s.taskSvc.Shutdown(ctx); err != nil {
			shutdownErr = errors.Join(shutdownErr, fmt.Errorf("shut down task service: %w", err))
		}
	}
	if s.routedDB != nil {
		shutdownErr = errors.Join(shutdownErr, s.routedDB.Close())
	}
	return shutdownErr
}

// Main function
func main() {
	// The stale-source guard must run before every command path so a rebuilt
	// binary re-execs before it can initialize any scenario-owned state.
	if preflight.Run(preflight.Config{
		ScenarioName: "scenario-to-desktop",
	}) {
		return // Process was re-exec'd after rebuild
	}

	if len(os.Args) > 1 && os.Args[1] == "storage-relocate" {
		if err := runStorageRelocate(os.Args[2:]); err != nil {
			globalLogger.Error("storage relocation failed", "error", err)
			log.Fatal(err)
		}
		return
	}

	// SECURITY: Validate port environment variables - prefer API_PORT, fallback to PORT
	port := 15200
	apiPortStr := os.Getenv("API_PORT")
	portStr := os.Getenv("PORT")

	switch {
	case apiPortStr != "":
		p, err := strconv.Atoi(apiPortStr)
		if err != nil {
			log.Fatalf("❌ Invalid API_PORT value '%s': must be a valid integer", apiPortStr)
		}
		if p < 1024 || p > 65535 {
			log.Fatalf("❌ Invalid API_PORT value %d: must be between 1024 and 65535", p)
		}
		port = p
	case portStr != "":
		// Fallback to PORT for compatibility
		p, err := strconv.Atoi(portStr)
		if err != nil {
			log.Fatalf("❌ Invalid PORT value '%s': must be a valid integer", portStr)
		}
		if p < 1024 || p > 65535 {
			log.Fatalf("❌ Invalid PORT value %d: must be between 1024 and 65535", p)
		}
		port = p
	default:
		globalLogger.Warn("no port configuration found",
			"message", "No API_PORT or PORT environment variable set",
			"action", "using default port",
			"default_port", port)
	}

	// Create server
	srv := NewServer(port)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			globalLogger.Error("scenario shutdown incomplete", "error", err)
		}
	}()

	// Start with graceful shutdown via api-core
	if err := server.Run(server.Config{
		Handler: srv.Router(),
		Port:    strconv.Itoa(port),
	}); err != nil {
		globalLogger.Error("server failed", "error", err)
		return
	}
}

func runStorageRelocate(args []string) error {
	fs := flag.NewFlagSet("storage-relocate", flag.ContinueOnError)
	repoRoot := fs.String("repo-root", "", "Override detected repo root")
	homeDir := fs.String("home-dir", "", "Override detected home directory")
	jsonOutput := fs.Bool("json", false, "Print machine-readable JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	result, err := storagemigrate.Run(storagemigrate.Options{
		RepoRoot: *repoRoot,
		HomeDir:  *homeDir,
	})
	if err != nil {
		return err
	}

	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	for _, moved := range result.Moved {
		_, _ = fmt.Fprintf(os.Stdout, "moved %-4s %s -> %s\n", moved.Kind, moved.Source, moved.Destination)
	}
	for _, skipped := range result.Skipped {
		_, _ = fmt.Fprintf(os.Stdout, "skipped %-4s %s\n", skipped.Kind, skipped.Source)
	}
	_, _ = fmt.Fprintf(os.Stdout, "storage relocation complete: %d moved, %d skipped\n", len(result.Moved), len(result.Skipped))
	return nil
}
