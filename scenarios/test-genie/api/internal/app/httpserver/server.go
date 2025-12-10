package httpserver

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"test-genie/internal/agents"
	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/queue"
	"test-genie/internal/scenarios"
)

// Config controls the HTTP transport settings.
type Config struct {
	Port        string
	ServiceName string
}

// Logger captures the logging surface the HTTP layer relies on.
type Logger interface {
	Print(v ...interface{})
	Printf(format string, v ...interface{})
}

// Dependencies encapsulates the services the HTTP layer needs to operate.
type Dependencies struct {
	DB           *sql.DB
	SuiteQueue   suiteRequestQueue
	Executions   execution.ExecutionHistory
	ExecutionSvc suiteExecutor
	Scenarios    scenarioDirectory
	PhaseCatalog phaseCatalog
	AgentService *agents.AgentService
	Logger       Logger
}

type suiteRequestQueue interface {
	Queue(ctx context.Context, payload queue.QueueSuiteRequestInput) (*queue.SuiteRequest, error)
	List(ctx context.Context, limit int) ([]queue.SuiteRequest, error)
	Get(ctx context.Context, id uuid.UUID) (*queue.SuiteRequest, error)
	StatusSnapshot(ctx context.Context) (queue.SuiteRequestSnapshot, error)
}

type suiteExecutor interface {
	Execute(ctx context.Context, input execution.SuiteExecutionInput) (*orchestrator.SuiteExecutionResult, error)
	ExecuteWithEvents(ctx context.Context, input execution.SuiteExecutionInput, emit orchestrator.ExecutionEventCallback) (*orchestrator.SuiteExecutionResult, error)
}

type scenarioDirectory interface {
	ListSummaries(ctx context.Context) ([]scenarios.ScenarioSummary, error)
	GetSummary(ctx context.Context, name string) (*scenarios.ScenarioSummary, error)
	RunScenarioTests(ctx context.Context, name string, preferred string, extraArgs []string) (*scenarios.TestingCommand, *scenarios.TestingRunnerResult, error)
	RunUISmoke(ctx context.Context, name string, uiURL string, browserlessURL string, timeoutMs int64) (*scenarios.UISmokeResult, error)
	RunUISmokeWithOpts(ctx context.Context, name string, opts scenarios.UISmokeOptions) (*scenarios.UISmokeResult, error)
	ListFiles(ctx context.Context, name string, opts scenarios.FileListOptions) ([]scenarios.FileNode, error)
	ListFilesWithMeta(ctx context.Context, name string, opts scenarios.FileListOptions) (scenarios.FileListResult, error)
}

type phaseCatalog interface {
	DescribePhases() []phases.Descriptor
	GlobalPhaseToggles() (orchestrator.PhaseToggleConfig, error)
	SaveGlobalPhaseToggles(orchestrator.PhaseToggleConfig) (orchestrator.PhaseToggleConfig, error)
}

// Server wires the HTTP router, configuration, and service dependencies behind intentional seams.
type Server struct {
	config           Config
	db               *sql.DB
	router           *mux.Router
	suiteRequests    suiteRequestQueue
	executionHistory execution.ExecutionHistory
	executionSvc     suiteExecutor
	scenarios        scenarioDirectory
	phaseCatalog     phaseCatalog
	logger           Logger
	agentService     *agents.AgentService
	wsManager        *WebSocketManager
}

// New creates a configured HTTP server instance.
func New(config Config, deps Dependencies) (*Server, error) {
	if strings.TrimSpace(config.Port) == "" {
		return nil, fmt.Errorf("api port is required")
	}
	if deps.DB == nil {
		return nil, fmt.Errorf("database dependency is required")
	}
	if deps.SuiteQueue == nil {
		return nil, fmt.Errorf("suite request service is required")
	}
	if deps.Executions == nil {
		return nil, fmt.Errorf("execution history service is required")
	}
	if deps.ExecutionSvc == nil {
		return nil, fmt.Errorf("execution service is required")
	}
	if deps.Scenarios == nil {
		return nil, fmt.Errorf("scenario directory service is required")
	}
	if deps.PhaseCatalog == nil {
		return nil, fmt.Errorf("phase catalog dependency is required")
	}
	if deps.AgentService == nil {
		return nil, fmt.Errorf("agent service dependency is required")
	}

	logger := deps.Logger
	if logger == nil {
		logger = log.Default()
	}

	wsManager := NewWebSocketManager()

	srv := &Server{
		config:           config,
		db:               deps.DB,
		router:           mux.NewRouter(),
		suiteRequests:    deps.SuiteQueue,
		executionHistory: deps.Executions,
		executionSvc:     deps.ExecutionSvc,
		scenarios:        deps.Scenarios,
		phaseCatalog:     deps.PhaseCatalog,
		logger:           logger,
		agentService:     deps.AgentService,
		wsManager:        wsManager,
	}

	srv.setupRoutes()
	return srv, nil
}

func (s *Server) setupRoutes() {
	s.router.Use(s.loggingMiddleware)
	// Health endpoint at root for infrastructure agents
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
	// WebSocket endpoint at root level for real-time updates
	s.router.HandleFunc("/ws", s.wsManager.HandleWebSocket)

	apiRouter := s.router.PathPrefix("/api/v1").Subrouter()
	apiRouter.HandleFunc("/health", s.handleHealth).Methods("GET")
	apiRouter.HandleFunc("/config", s.handleGetConfig).Methods("GET")
	apiRouter.HandleFunc("/suite-requests", s.handleCreateSuiteRequest).Methods("POST")
	apiRouter.HandleFunc("/suite-requests", s.handleListSuiteRequests).Methods("GET")
	apiRouter.HandleFunc("/suite-requests/{id}", s.handleGetSuiteRequest).Methods("GET")
	apiRouter.HandleFunc("/phases", s.handleListPhases).Methods("GET")
	apiRouter.HandleFunc("/phases/settings", s.handleGetPhaseSettings).Methods("GET")
	apiRouter.HandleFunc("/phases/settings", s.handleUpdatePhaseSettings).Methods("PUT")
	apiRouter.HandleFunc("/executions", s.handleExecuteSuite).Methods("POST")
	apiRouter.HandleFunc("/executions/stream", s.handleExecuteSuiteStream).Methods("POST")
	apiRouter.HandleFunc("/executions", s.handleListExecutions).Methods("GET")
	apiRouter.HandleFunc("/executions/{id}", s.handleGetExecution).Methods("GET")
	apiRouter.HandleFunc("/scenarios", s.handleListScenarios).Methods("GET")
	apiRouter.HandleFunc("/scenarios/{name}", s.handleGetScenario).Methods("GET")
	apiRouter.HandleFunc("/scenarios/{name}/run-tests", s.handleRunScenarioTests).Methods("POST")
	apiRouter.HandleFunc("/scenarios/{name}/ui-smoke", s.handleUISmoke).Methods("POST")
	apiRouter.HandleFunc("/scenarios/{name}/files", s.handleListScenarioFiles).Methods("GET")
	apiRouter.HandleFunc("/agents/models", s.handleListAgentModels).Methods("GET")
	apiRouter.HandleFunc("/agents/spawn", s.handleSpawnAgents).Methods("POST")
	apiRouter.HandleFunc("/agents/active", s.handleListActiveAgents).Methods("GET")
	apiRouter.HandleFunc("/agents/check-conflicts", s.handleCheckScopeConflicts).Methods("POST")
	apiRouter.HandleFunc("/agents/validate-paths", s.handleValidatePromptPaths).Methods("POST")
	apiRouter.HandleFunc("/agents/blocked-commands", s.handleGetBlockedCommands).Methods("GET")
	apiRouter.HandleFunc("/agents/cleanup", s.handleCleanupAgents).Methods("POST")
	apiRouter.HandleFunc("/agents/containment-status", s.handleContainmentStatus).Methods("GET")
	apiRouter.HandleFunc("/agents/{id}", s.handleGetAgent).Methods("GET")
	apiRouter.HandleFunc("/agents/{id}/stop", s.handleStopAgent).Methods("POST")
	apiRouter.HandleFunc("/agents/{id}/heartbeat", s.handleAgentHeartbeat).Methods("POST")
	apiRouter.HandleFunc("/agents/stop-all", s.handleStopAllAgents).Methods("POST")
	// Server-side spawn session tracking (prevents duplicate spawns across browser tabs)
	apiRouter.HandleFunc("/agents/spawn-sessions", s.handleGetSpawnSessions).Methods("GET")
	apiRouter.HandleFunc("/agents/spawn-sessions/check-conflicts", s.handleCheckSpawnSessionConflicts).Methods("POST")
	apiRouter.HandleFunc("/agents/spawn-sessions/clear", s.handleClearSpawnSessions).Methods("POST")

	// Docs endpoints for in-app documentation browser
	apiRouter.HandleFunc("/docs/manifest", s.handleGetDocsManifest).Methods("GET")
	apiRouter.HandleFunc("/docs/content", s.handleGetDocContent).Methods("GET")

	// Requirements endpoints for requirements sync UI
	apiRouter.HandleFunc("/scenarios/{name}/requirements", s.handleGetScenarioRequirements).Methods("GET")
	apiRouter.HandleFunc("/scenarios/{name}/requirements/sync", s.handleSyncScenarioRequirements).Methods("POST")
}

// Start launches the HTTP server with graceful shutdown.
func (s *Server) Start() error {
	s.log("starting server", map[string]interface{}{
		"service": s.serviceName(),
		"port":    s.config.Port,
	})

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", s.config.Port),
		Handler: handlers.RecoveryHandler()(s.router),
		// Extended timeouts to support long-running SSE streams for test execution
		// Test suites can run for up to 15 minutes
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 20 * time.Minute, // Extended for SSE streaming
		IdleTimeout:  120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
	case err := <-errCh:
		return fmt.Errorf("server startup failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	s.log("server stopped", nil)
	return nil
}

// loggingMiddleware prints simple request logs.
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Printf("[%s] %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

func (s *Server) log(msg string, fields map[string]interface{}) {
	if s.logger == nil {
		return
	}
	if len(fields) == 0 {
		s.logger.Print(msg)
		return
	}
	s.logger.Printf("%s | %v", msg, fields)
}

func (s *Server) serviceName() string {
	name := strings.TrimSpace(s.config.ServiceName)
	if name == "" {
		return "Test Genie API"
	}
	return name
}
