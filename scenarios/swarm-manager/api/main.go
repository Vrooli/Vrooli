// Package main provides the Swarm Manager API server.
//
// DOC: docs/concepts/ARCHITECTURE.md#logical-architecture
// DOC: docs/internal/INTENT.md#api-components
//
// The API serves as the backend for the Swarm Manager UI, providing endpoints
// for backlog management, scenario lifecycle, execution control, settings,
// agent coordination, and real-time graph streaming.
//
// Related PRD targets: OT-P0-002, OT-P0-005, OT-P0-006
package main

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/aisearch"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/captures"
	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/graph"
	"swarm-manager/internal/identity"
	"swarm-manager/internal/initiativereview"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/overview"
	"swarm-manager/internal/pathutil"
	"swarm-manager/internal/prompts"
	"swarm-manager/internal/queue"
	"swarm-manager/internal/review"
	"swarm-manager/internal/runtimepaths"
	"swarm-manager/internal/scenarios"
	"swarm-manager/internal/settings"
	"swarm-manager/internal/stats"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
	_ "modernc.org/sqlite"
)

type Server struct {
	router              *mux.Router
	agentSvc            *agentmanager.AgentService
	agentActivitySvc    *agentactivity.Service
	settingsStore       *settings.Store
	backlogHandler      *backlog.Handler
	capturesHandler     *captures.Handler
	scenariosHandler    *scenarios.Handler
	initStore           *initiatives.Store
	initiativeService   *initiatives.Service
	executionSvc        *execution.Service
	executionHandler    *execution.Handler
	reviewSvc           *review.Service
	reviewHandler       *review.Handler
	initiativeReviewSvc *initiativereview.Service
	executionStopChan   chan struct{}
	reviewStopChan      chan struct{}
	initReviewStopChan  chan struct{}
	graphBroker         *graph.Broker
	graphDispatch       *graph.Dispatch
	queueHandler        *queue.Handler
	scenarioRoot        string
	eventDB             *sql.DB
	emitter             *eventlog.Emitter
	statsEngine         *stats.Engine
	aiSearchSvc         *aisearch.Service
	aiSearchStopChan    chan struct{}
	feedbackSweeperStop chan struct{}
}

// NewServer initializes routes using the default scenario root resolved from
// the environment.
func NewServer() *Server {
	return NewServerWithRoot(pathutil.ResolveScenarioRoot("swarm-manager"))
}

// NewServerWithRoot initializes routes using the given scenario root directory.
// Tests should use this with t.TempDir() to avoid touching production data.
func NewServerWithRoot(scenarioRoot string) *Server {
	agentEnabled := strings.ToLower(strings.TrimSpace(os.Getenv("AGENT_MANAGER_ENABLED"))) != "false"
	agentSvc := agentmanager.NewAgentService(agentmanager.AgentServiceConfig{
		ProfileName: getEnvDefault("AGENT_MANAGER_PROFILE_NAME", "swarm-manager"),
		ProfileKey:  getEnvDefault("AGENT_MANAGER_PROFILE_KEY", "swarm-manager/default"),
		Timeout:     30 * time.Second,
		Enabled:     agentEnabled,
	})

	srv := &Server{
		router:              mux.NewRouter(),
		agentSvc:            agentSvc,
		executionStopChan:   make(chan struct{}),
		reviewStopChan:      make(chan struct{}),
		initReviewStopChan:  make(chan struct{}),
		aiSearchStopChan:    make(chan struct{}),
		feedbackSweeperStop: make(chan struct{}),
		scenarioRoot:        scenarioRoot,
	}
	// initEventLog must run before setupRoutes so that route registration
	// captures a non-nil s.emitter. Constructors like registerFeedbackRoutes
	// build internal services (backlog.Service for proposal apply) that take
	// the emitter at construction time and have no SetEventLogger backstop;
	// if s.emitter were still nil here, those services would hold a typed-nil
	// *eventlog.Emitter behind a non-nil CreationEventEmitter interface and
	// panic on first emit.
	srv.initEventLog()
	srv.setupRoutes()
	srv.wireEventLoggers()
	return srv
}

func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)
	s.router.Use(identity.Middleware(identity.CLIUtilVerifier{}))
	scenarioRoot := s.scenarioRoot
	scenariosDir := filepath.Dir(scenarioRoot)

	// --- Infrastructure ---
	s.registerHealthRoutes()
	s.registerSettingsRoutes(scenarioRoot)      // Must be before backlog/execution (they depend on settings store)
	s.registerAgentActivityRoutes(scenarioRoot) // Must be before backlog/execution (they depend on agent activity)
	s.registerScenarioRoutes(scenariosDir)

	// --- Core domain ---
	backlogHandler := s.registerBacklogRoutes(scenarioRoot)
	initService := s.registerInitiativeRoutes(scenarioRoot, backlogHandler)
	s.registerCapturesRoutes(scenarioRoot, backlogHandler)

	// --- Execution & review ---
	execSvc := s.registerExecutionRoutes(scenarioRoot)
	s.registerReviewRoutes(scenarioRoot, execSvc)
	s.registerQueueRoutes(scenarioRoot)

	// --- Cross-domain wiring ---
	s.scenariosHandler.SetBacklogLister(backlogHandler.Store())
	s.scenariosHandler.SetInitiativesLister(initService)
	if execSvc != nil {
		s.scenariosHandler.SetExecutionLister(execSvc)
	}

	// --- Read-only surfaces ---
	overviewSvc := s.registerOverviewRoutes(backlogHandler, initService)
	if execSvc != nil {
		overviewSvc.SetGovernanceProvider(execSvc)
	}
	materializer := s.registerGraphRoutes(scenarioRoot)
	s.registerFeedbackRoutes(materializer)
	s.registerInitiativeReviewRoutes(materializer)
	s.registerPromptRoutes(scenarioRoot)
	s.registerAgentManagerRoutes()

	// --- AI search (must come last so readers see fully-wired stores) ---
	s.registerAISearchRoutes(backlogHandler, initService)
}

// registerAISearchRoutes constructs the aisearch service from environment
// configuration, wires index-on-write hooks into the backlog and initiative
// stores, and registers HTTP routes under /api/v1/search/ai. If required
// resources (Ollama, Qdrant) are not configured, the service is still created
// so /status can explain why AI search is unavailable; write hooks are still
// attached so index operations queue correctly once resources come online.
func (s *Server) registerAISearchRoutes(backlogHandler *backlog.Handler, initService *initiatives.Service) {
	cfg := aisearch.LoadConfigFromEnv()

	embedder := aisearch.NewEmbedder(cfg.OllamaURL, cfg.EmbeddingModel)
	backlogVS := aisearch.NewVectorStore(cfg.QdrantURL, cfg.QdrantAPIKey, cfg.BacklogCollection, cfg.VectorDimensions)
	initVS := aisearch.NewVectorStore(cfg.QdrantURL, cfg.QdrantAPIKey, cfg.InitiativeCollection, cfg.VectorDimensions)

	backlogReader := aisearch.NewBacklogStoreAdapter(backlogHandler.Store())
	initReader := aisearch.NewInitiativeStoreAdapter(s.initStore)

	svc := aisearch.NewService(embedder, backlogVS, initVS, backlogReader, initReader, cfg.Threshold)

	// Only attach indexer hooks if both subsystems are configured; otherwise
	// write-path goroutines would bang on unreachable URLs on every mutation.
	if cfg.OllamaURL != "" && cfg.QdrantURL != "" {
		backlogHandler.SetAIIndexer(svc)
		initService.SetAIIndexer(svc)
	}

	handler := aisearch.NewHandler(svc)
	handler.RegisterRoutes(s.router)
	s.aiSearchSvc = svc
}

func (s *Server) registerHealthRoutes() {
	healthHandler := health.New().Version("1.0.0").
		Check(health.CheckerFunc(func(_ context.Context) health.CheckResult {
			return health.CheckResult{Name: "database", Connected: true}
		}), health.Optional).
		Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")
}

func (s *Server) registerBacklogRoutes(scenarioRoot string) *backlog.Handler {
	backlogHandler := backlog.NewHandlerWithClients(scenarioRoot, s.requireTrackedAgentService(), nil)
	backlogHandler.SetPolicyProvider(settings.NewPolicyAdapter(s.settingsStore))
	backlogHandler.SetGovernanceProvider(settings.NewGovernanceAdapter(s.settingsStore))
	backlogHandler.RegisterRoutes(s.router)
	backlogHandler.StartWorkshopTicker()
	s.backlogHandler = backlogHandler
	return backlogHandler
}

func (s *Server) registerInitiativeRoutes(scenarioRoot string, backlogHandler *backlog.Handler) *initiatives.Service {
	initStore := initiatives.NewStore(scenarioRoot)
	s.initStore = initStore
	initService := initiatives.NewService(initStore, backlogHandler.Store())
	initHandler := initiatives.NewHandler(initService)
	initHandler.RegisterRoutes(s.router)
	s.initiativeService = initService

	// Wire initiative assigner into backlog handler for batch operations.
	backlogHandler.SetInitiativeAssigner(initiatives.NewBacklogAssignerAdapter(initService))
	return initService
}

func (s *Server) registerOverviewRoutes(backlogHandler *backlog.Handler, initService *initiatives.Service) *overview.Service {
	overviewSvc := overview.NewService(backlogHandler.Store(), initService)
	overviewHandler := overview.NewHandler(overviewSvc)
	overviewHandler.RegisterRoutes(s.router)
	return overviewSvc
}

func (s *Server) registerCapturesRoutes(scenarioRoot string, backlogHandler *backlog.Handler) {
	capturesHandler := captures.NewHandler(scenarioRoot, s.requireTrackedAgentService(), nil)
	capturesHandler.SetBacklogCreator(captures.NewBacklogItemCreatorAdapter(backlogHandler.Store()))
	capturesHandler.RegisterRoutes(s.router)
	s.capturesHandler = capturesHandler
}

func (s *Server) registerScenarioRoutes(scenariosDir string) {
	s.scenariosHandler = scenarios.NewHandler(scenariosDir)
	s.scenariosHandler.RegisterRoutes(s.router)
}

func (s *Server) registerSettingsRoutes(scenarioRoot string) {
	settingsPath := filepath.Join(scenarioRoot, "config", "settings.json")
	settingsHandler := settings.NewHandler(settingsPath)
	settingsHandler.RegisterRoutes(s.router)
	s.settingsStore = settingsHandler.GetStore()

	// Wire settings into agent service for runtime profile config resolution.
	if s.agentSvc != nil {
		s.agentSvc.SetSettingsReader(settings.NewAgentAdapter(s.settingsStore))
	}
}

func (s *Server) registerAgentActivityRoutes(_ string) {
	storePath, err := runtimepaths.StatePath("agent-activities.json")
	if err != nil {
		panic(err)
	}
	s.agentActivitySvc = agentactivity.NewService(agentactivity.ServiceConfig{
		StorePath:    storePath,
		AgentService: s.agentSvc,
	})
	agentActivityHandler := agentactivity.NewHandler(s.agentActivitySvc)
	agentActivityHandler.RegisterRoutes(s.router)
}

func (s *Server) registerAgentManagerRoutes() {
	agentManagerHandler := agentmanager.NewHandler(s.agentSvc)
	agentManagerHandler.RegisterRoutes(s.router)
}

func (s *Server) registerQueueRoutes(_ string) {
	storePath, err := runtimepaths.StatePath("queue.json")
	if err != nil {
		panic(err)
	}
	s.queueHandler = queue.NewHandler(storePath)
	s.queueHandler.RegisterRoutes(s.router)
}

func (s *Server) requireTrackedAgentService() *agentactivity.Service {
	if s.agentActivitySvc == nil {
		panic("agent activity service must be initialized before agent-dependent routes")
	}
	return s.agentActivitySvc
}

func (s *Server) registerPromptRoutes(scenarioRoot string) {
	promptHandler := prompts.NewHandler(scenarioRoot, nil)
	promptHandler.RegisterRoutes(s.router)
}

// Handler returns the HTTP handler with recovery middleware.
func (s *Server) Handler() http.Handler {
	return handlers.RecoveryHandler()(s.router)
}

// loggingMiddleware prints simple request logs.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("request", "method", r.Method, "uri", r.RequestURI, "duration", time.Since(start))
	})
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "swarm-manager",
	}) {
		return // Process was re-exec'd after rebuild
	}

	slog.Info("running in filesystem-only mode")

	srv := NewServer()
	srv.runMigrationsOnce()

	// Register stats endpoint (requires event log).
	if srv.statsEngine != nil {
		statsHandler := stats.NewHandler(srv.statsEngine)
		statsHandler.RegisterRoutes(srv.router)
	}

	if srv.executionHandler != nil {
		go srv.executionHandler.StartBackgroundWorker(srv.executionStopChan)
	}

	if srv.reviewSvc != nil {
		srv.reviewSvc.RecoverActiveRounds()
		go srv.reviewSvc.StartBackgroundWorker(srv.reviewStopChan)
	}

	if srv.initiativeReviewSvc != nil {
		srv.initiativeReviewSvc.RecoverActiveRounds()
		go srv.initiativeReviewSvc.StartBackgroundWorker(srv.initReviewStopChan)
	}

	if srv.agentSvc != nil && srv.agentSvc.IsEnabled() {
		initCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := srv.agentSvc.Initialize(initCtx, nil); err != nil {
			slog.Warn("failed to initialize agent-manager profile", "error", err)
		}
		cancel()
	}

	srv.startAISearchBackground()

	if err := server.Run(server.Config{
		Handler: srv.Handler(),
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
	close(srv.executionStopChan)
	close(srv.reviewStopChan)
	close(srv.initReviewStopChan)
	close(srv.aiSearchStopChan)
	close(srv.feedbackSweeperStop)
}

// startAISearchBackground kicks off two background tasks for aisearch:
// a one-shot startup backfill (if index and on-disk counts diverge) and a
// periodic drift-reconciliation loop (every 5 minutes). Both are no-ops when
// Ollama or Qdrant is unreachable — the goroutines log and move on.
func (s *Server) startAISearchBackground() {
	if s.aiSearchSvc == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		needs, indexed, disk, err := s.aiSearchSvc.NeedsReindex(ctx)
		if err != nil {
			slog.Info("aisearch startup backfill skipped", "reason", err.Error())
			return
		}
		if needs {
			slog.Info("aisearch startup backfill: index drift detected, reindexing",
				"indexed", indexed, "on_disk", disk)
			s.aiSearchSvc.StartReindex()
		}
	}()

	syncCtx, cancel := context.WithCancel(context.Background())
	s.aiSearchSvc.StartPeriodicSync(syncCtx, 5*time.Minute)
	go func() {
		<-s.aiSearchStopChan
		cancel()
	}()
}

func getEnvDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
