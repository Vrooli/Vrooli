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

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
	_ "modernc.org/sqlite"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/identity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/captures"
	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/graph"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/overview"
	"swarm-manager/internal/pathutil"
	"swarm-manager/internal/prompts"
	"swarm-manager/internal/queue"
	"swarm-manager/internal/review"
	"swarm-manager/internal/scenarios"
	"swarm-manager/internal/settings"
	"swarm-manager/internal/stats"
)

type Server struct {
	router            *mux.Router
	agentSvc          *agentmanager.AgentService
	agentActivitySvc  *agentactivity.Service
	settingsStore     *settings.Store
	backlogHandler    *backlog.Handler
	capturesHandler   *captures.Handler
	scenariosHandler  *scenarios.Handler
	initStore         *initiatives.Store
	initiativeService *initiatives.Service
	executionSvc      *execution.Service
	executionHandler  *execution.Handler
	reviewSvc         *review.Service
	reviewHandler     *review.Handler
	executionStopChan chan struct{}
	reviewStopChan    chan struct{}
	graphBroker       *graph.Broker
	queueHandler      *queue.Handler
	scenarioRoot      string
	eventDB           *sql.DB
	emitter           *eventlog.Emitter
	statsEngine       *stats.Engine
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
		ProfileKey:  getEnvDefault("AGENT_MANAGER_PROFILE_KEY", "swarm-manager"),
		Timeout:     30 * time.Second,
		Enabled:     agentEnabled,
	})

	srv := &Server{
		router:            mux.NewRouter(),
		agentSvc:          agentSvc,
		executionStopChan: make(chan struct{}),
		reviewStopChan:    make(chan struct{}),
		scenarioRoot:      scenarioRoot,
	}
	srv.setupRoutes()
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

	// --- Read-only surfaces ---
	overviewSvc := s.registerOverviewRoutes(backlogHandler, initService)
	if execSvc != nil {
		overviewSvc.SetGovernanceProvider(execSvc)
	}
	s.registerGraphRoutes(scenarioRoot)
	s.registerPromptRoutes(scenarioRoot)
	s.registerAgentManagerRoutes()
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
	if err := initStore.Migrate(); err != nil {
		slog.Warn("initiatives migration warning", "error", err)
	}
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
	settingsPath := filepath.Join(scenarioRoot, ".vrooli", "settings.json")
	settingsHandler := settings.NewHandler(settingsPath)
	settingsHandler.RegisterRoutes(s.router)
	s.settingsStore = settingsHandler.GetStore()

	// Wire settings into agent service for runtime profile config resolution.
	if s.agentSvc != nil {
		s.agentSvc.SetSettingsReader(settings.NewAgentAdapter(s.settingsStore))
	}
}

func (s *Server) registerAgentActivityRoutes(scenarioRoot string) {
	s.agentActivitySvc = agentactivity.NewService(agentactivity.ServiceConfig{
		StorePath:    filepath.Join(scenarioRoot, ".vrooli", "agent-activities.json"),
		AgentService: s.agentSvc,
	})
	agentActivityHandler := agentactivity.NewHandler(s.agentActivitySvc)
	agentActivityHandler.RegisterRoutes(s.router)
}

func (s *Server) registerAgentManagerRoutes() {
	agentManagerHandler := agentmanager.NewHandler(s.agentSvc)
	agentManagerHandler.RegisterRoutes(s.router)
}

func (s *Server) registerQueueRoutes(scenarioRoot string) {
	s.queueHandler = queue.NewHandler(filepath.Join(scenarioRoot, ".vrooli", "queue.json"))
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
	srv.initEventLog()
	srv.wireEventLoggers()

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

	if srv.agentSvc != nil && srv.agentSvc.IsEnabled() {
		initCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := srv.agentSvc.Initialize(initCtx, nil); err != nil {
			slog.Warn("failed to initialize agent-manager profile", "error", err)
		}
		cancel()
	}

	if err := server.Run(server.Config{
		Handler: srv.Handler(),
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
	close(srv.executionStopChan)
	close(srv.reviewStopChan)
}

func getEnvDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
