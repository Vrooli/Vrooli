package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/sandbox"
	agentconfig "agent-manager/internal/config"
	"agent-manager/internal/database"
	"agent-manager/internal/domain"
	"agent-manager/internal/handlers"
	"agent-manager/internal/metrics"
	"agent-manager/internal/modelregistry"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/pricing"
	"agent-manager/internal/pricing/providers"
	"agent-manager/internal/repository"
	"agent-manager/internal/toolregistry"

	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
)

// Config holds runtime configuration
type Config struct {
	Port        string
	UseInMemory bool
}

// Server wires the HTTP router, database, and orchestration service
type Server struct {
	config                 *Config
	db                     *database.DB
	logger                 *logrus.Logger
	router                 *mux.Router
	orchestrator           orchestration.Service
	investigationService   orchestration.InvestigationService
	statsService           orchestration.StatsService
	statsRepo              repository.StatsRepository
	pricingService         pricing.Service
	wsHub                  *handlers.WebSocketHub
	reconciler             *orchestration.Reconciler
	toolRegistry           *toolregistry.Registry
}

// NewServer initializes configuration, database, and routes
func NewServer() (*Server, error) {
	// Initialize structured logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	cfg := &Config{
		UseInMemory: strings.ToLower(os.Getenv("USE_IN_MEMORY")) == "true",
	}

	var db *database.DB
	var err error

	// Database connection (optional for in-memory mode)
	if !cfg.UseInMemory {
		db, err = database.NewConnection(logger)
		if err != nil {
			log.Printf("Failed to connect to database, using in-memory storage: %v", err)
			cfg.UseInMemory = true
			db = nil
		}
	}

	// Create WebSocket hub for real-time event broadcasting (needed by orchestrator)
	wsHub := handlers.NewWebSocketHub()
	go wsHub.Run()

	// Create the orchestrator with appropriate repositories and broadcaster
	deps := createOrchestrator(db, cfg.UseInMemory, wsHub, logger)

	// Create tool registry for tool discovery protocol
	toolReg := toolregistry.NewRegistry(toolregistry.RegistryConfig{
		ScenarioName:        "agent-manager",
		ScenarioVersion:     "1.0.0",
		ScenarioDescription: "Manages coding agents for software engineering tasks. Supports Claude Code, Codex, and OpenCode runners with sandboxed execution and approval workflows.",
	})
	toolReg.RegisterProvider(toolregistry.NewAgentToolProvider())

	// UseEncodedPath tells mux to match routes against the raw URL path (e.g., keeping %2F
	// as-is instead of decoding to /). This is required for model names containing slashes
	// like "aion-labs/aion-1.0" which are URL-encoded to "aion-labs%2Faion-1.0".
	srv := &Server{
		config:               cfg,
		db:                   db,
		logger:               logger,
		router:               mux.NewRouter().UseEncodedPath(),
		orchestrator:         deps.orchestrator,
		investigationService: deps.investigationService,
		statsService:         deps.statsService,
		statsRepo:            deps.statsRepo,
		pricingService:       deps.pricingService,
		wsHub:                wsHub,
		reconciler:           deps.reconciler,
		toolRegistry:         toolReg,
	}

	// Start the reconciler for orphan detection and stale run recovery
	if srv.reconciler != nil {
		if err := srv.reconciler.Start(context.Background()); err != nil {
			log.Printf("Warning: Failed to start reconciler: %v", err)
		}
	}

	srv.setupRoutes()
	return srv, nil
}

// orchestratorDeps holds the orchestrator and related services
type orchestratorDeps struct {
	orchestrator         orchestration.Service
	investigationService orchestration.InvestigationService
	statsService         orchestration.StatsService
	statsRepo            repository.StatsRepository
	pricingService       pricing.Service
	reconciler           *orchestration.Reconciler
}

// createOrchestrator creates the orchestration service with all dependencies
func createOrchestrator(db *database.DB, useInMemory bool, wsHub *handlers.WebSocketHub, logger *logrus.Logger) orchestratorDeps {
	levers, err := agentconfig.LoadLevers()
	if err != nil {
		log.Printf("Warning: failed to load config levers: %v", err)
	}

	// Create repositories
	var (
		profileRepo       repository.ProfileRepository
		taskRepo          repository.TaskRepository
		runRepo           repository.RunRepository
		checkpointRepo    repository.CheckpointRepository
		idempotencyRepo   repository.IdempotencyRepository
		investigationRepo repository.InvestigationRepository
		statsRepo         repository.StatsRepository
	)

	// Create event store - use PostgreSQL when database is available
	var eventStore event.Store
	storageLabel := ""

	if useInMemory || db == nil {
		eventStore = event.NewMemoryStore()
		// In-memory fallback
		log.Printf("Using in-memory storage")
		storageLabel = "memory (fallback)"
		profileRepo = repository.NewMemoryProfileRepository()
		taskRepo = repository.NewMemoryTaskRepository()
		memRunRepo := repository.NewMemoryRunRepository()
		runRepo = memRunRepo
		checkpointRepo = repository.NewMemoryCheckpointRepository()
		idempotencyRepo = repository.NewMemoryIdempotencyRepository()
		investigationRepo = repository.NewMemoryInvestigationRepository()
		statsRepo = repository.NewMemoryStatsRepositoryWithRepos(memRunRepo, nil)
	} else {
		// PostgreSQL persistence
		log.Printf("Using PostgreSQL persistence")
		storageLabel = string(db.Dialect())
		eventStore = event.NewPostgresStore(db.DB, logger)
		repos := database.NewRepositories(db, logger)
		profileRepo = repos.Profiles
		taskRepo = repos.Tasks
		runRepo = repos.Runs
		checkpointRepo = repos.Checkpoints
		idempotencyRepo = repos.Idempotency
		investigationRepo = repos.Investigations
		statsRepo = repos.Stats
	}

	// Create runner registry
	runnerRegistry := runner.NewRegistry()

	// Register Claude Code runner (uses resource-claude-code)
	claudeRunner, err := runner.NewClaudeCodeRunner()
	if err != nil {
		log.Printf("Warning: Failed to create Claude Code runner: %v", err)
		if err := runnerRegistry.Register(runner.NewStubRunner(
			domain.RunnerTypeClaudeCode,
			fmt.Sprintf("claude-code runner failed to initialize: %v", err),
		)); err != nil {
			log.Printf("Warning: Failed to register stub Claude runner: %v", err)
		}
	} else {
		if err := runnerRegistry.Register(claudeRunner); err != nil {
			log.Printf("Warning: Failed to register Claude runner: %v", err)
		}
		if avail, msg := claudeRunner.IsAvailable(context.Background()); avail {
			log.Printf("Claude Code runner: available")
		} else {
			log.Printf("Claude Code runner: %s", msg)
		}
	}

	// Register Codex runner (uses resource-codex)
	codexRunner, err := runner.NewCodexRunner()
	if err != nil {
		log.Printf("Warning: Failed to create Codex runner: %v", err)
		if err := runnerRegistry.Register(runner.NewStubRunner(
			domain.RunnerTypeCodex,
			fmt.Sprintf("codex runner failed to initialize: %v", err),
		)); err != nil {
			log.Printf("Warning: Failed to register stub Codex runner: %v", err)
		}
	} else {
		if err := runnerRegistry.Register(codexRunner); err != nil {
			log.Printf("Warning: Failed to register Codex runner: %v", err)
		}
		if avail, msg := codexRunner.IsAvailable(context.Background()); avail {
			log.Printf("Codex runner: available")
		} else {
			log.Printf("Codex runner: %s", msg)
		}
	}

	// Register OpenCode runner (uses resource-opencode)
	openCodeRunner, err := runner.NewOpenCodeRunner()
	if err != nil {
		log.Printf("Warning: Failed to create OpenCode runner: %v", err)
		if err := runnerRegistry.Register(runner.NewStubRunner(
			domain.RunnerTypeOpenCode,
			fmt.Sprintf("opencode runner failed to initialize: %v", err),
		)); err != nil {
			log.Printf("Warning: Failed to register stub OpenCode runner: %v", err)
		}
	} else {
		if err := runnerRegistry.Register(openCodeRunner); err != nil {
			log.Printf("Warning: Failed to register OpenCode runner: %v", err)
		}
		if avail, msg := openCodeRunner.IsAvailable(context.Background()); avail {
			log.Printf("OpenCode runner: available")
		} else {
			log.Printf("OpenCode runner: %s", msg)
		}
	}

	// Create workspace-sandbox provider
	sandboxURL := os.Getenv("WORKSPACE_SANDBOX_URL")
	if sandboxURL == "" {
		if resolved, err := discovery.ResolveScenarioURLDefault(context.Background(), "workspace-sandbox"); err == nil {
			sandboxURL = resolved
		}
	}
	if sandboxURL == "" {
		// Try to get from port allocation
		port := os.Getenv("WORKSPACE_SANDBOX_API_PORT")
		if port == "" {
			port = "15427" // Default workspace-sandbox port
		}
		sandboxURL = fmt.Sprintf("http://localhost:%s", port)
	}
	sandboxProvider := sandbox.NewWorkspaceSandboxProvider(sandboxURL)

	// Create terminator for robust process termination (Phase 2)
	terminator := orchestration.NewTerminator(
		runRepo,
		runnerRegistry,
		orchestration.DefaultTerminatorConfig(),
	)

	modelRegistryPath := modelregistry.ResolvePath()
	modelRegistryStore, err := modelregistry.NewStore(modelRegistryPath)
	if err != nil {
		log.Printf("Warning: Failed to load model registry from %s: %v", modelRegistryPath, err)
	}

	orchConfig := orchestration.DefaultConfig()
	baseConfig := agentconfig.Load()
	if baseConfig != nil {
		orchConfig.DefaultProjectRoot = strings.TrimSpace(baseConfig.Sandbox.ProjectRoot)
	}
	if levers != nil {
		orchConfig.DefaultTimeout = levers.Execution.DefaultTimeout
		orchConfig.MaxConcurrentRuns = levers.Concurrency.MaxConcurrentRuns
		orchConfig.RequireSandboxByDefault = levers.Safety.RequireSandboxByDefault
		if len(levers.Runners.FallbackRunnerTypes) > 0 {
			seen := make(map[domain.RunnerType]struct{}, len(levers.Runners.FallbackRunnerTypes))
			for _, runnerType := range levers.Runners.FallbackRunnerTypes {
				rt := domain.RunnerType(runnerType)
				if !rt.IsValid() {
					log.Printf("Warning: skipping invalid fallback runner type: %s", runnerType)
					continue
				}
				if _, exists := seen[rt]; exists {
					continue
				}
				seen[rt] = struct{}{}
				orchConfig.RunnerFallbackTypes = append(orchConfig.RunnerFallbackTypes, rt)
			}
		}
	}

	// Build orchestrator with all dependencies including WebSocket broadcaster and terminator
	orch := orchestration.New(
		profileRepo,
		taskRepo,
		runRepo,
		orchestration.WithConfig(orchConfig),
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(runnerRegistry),
		orchestration.WithSandbox(sandboxProvider),
		orchestration.WithCheckpoints(checkpointRepo),
		orchestration.WithIdempotency(idempotencyRepo),
		orchestration.WithBroadcaster(wsHub),
		orchestration.WithTerminator(terminator),
		orchestration.WithStorageLabel(storageLabel),
		orchestration.WithModelRegistry(modelRegistryStore),
	)

	// Create reconciler for orphan detection and stale run recovery (Phase 2)
	reconciler := orchestration.NewReconciler(
		runRepo,
		runnerRegistry,
		orchestration.WithReconcilerConfig(orchestration.ReconcilerConfig{
			Interval:          30 * time.Second,
			StaleThreshold:    2 * time.Minute,
			OrphanGracePeriod: 5 * time.Minute,
			MaxStaleRuns:      10,
			KillOrphans:       true, // Always kill orphan processes
			AutoRecover:       true, // Auto-recover stale runs if process is alive
		}),
		orchestration.WithReconcilerBroadcaster(wsHub),
	)

	// Create investigation service for self-investigation capabilities
	investigationSvc := orchestration.NewInvestigationOrchestrator(orch, investigationRepo)

	// Create stats service for analytics
	statsSvc := orchestration.NewStatsOrchestrator(statsRepo)

	// Create pricing service for model pricing management
	var pricingRepo pricing.Repository
	if useInMemory || db == nil {
		pricingRepo = pricing.NewMemoryRepository()
	} else {
		pricingRepo = database.NewPricingRepository(db, logger)
	}
	openRouterProvider := providers.NewOpenRouterProvider()
	pricingProviders := []pricing.Provider{openRouterProvider}
	pricingSvc := pricing.NewService(pricingRepo, pricingProviders, logger)

	log.Printf("Orchestrator initialized (in-memory: %v, sandbox: %s)", useInMemory, sandboxURL)
	return orchestratorDeps{
		orchestrator:         orch,
		investigationService: investigationSvc,
		statsService:         statsSvc,
		statsRepo:            statsRepo,
		pricingService:       pricingSvc,
		reconciler:           reconciler,
	}
}

func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)
	s.router.Use(corsMiddleware)

	// Health endpoint using api-core/health for standardized response format
	// DB may be nil in in-memory mode - health.DB handles this gracefully
	var rawDB *sql.DB
	if s.db != nil && s.db.DB != nil {
		rawDB = s.db.DB.DB // Access underlying *sql.DB from sqlx.DB (which is embedded in database.DB)
	}
	healthHandler := health.New().
		Version("1.0.0").
		Check(health.DB(rawDB), health.Optional). // Optional since in-memory mode is valid
		Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	// Detailed health for UI (includes sandbox + runner dependencies).
	// Keep /health minimal for infra probes.
	handler := handlers.New(s.orchestrator)
	handler.SetWebSocketHub(s.wsHub)
	s.router.HandleFunc("/api/v1/health", handler.Health).Methods("GET")

	// Register all API routes via the handlers package
	// WebSocket hub was created in NewServer and is shared with orchestrator
	handler.RegisterRoutes(s.router)

	// Register investigation routes
	if s.investigationService != nil {
		investigationHandler := handlers.NewInvestigationHandler(s.investigationService)
		investigationHandler.RegisterRoutes(s.router)
		log.Printf("Investigation endpoints available at /api/v1/investigations")
	}

	// Register stats routes
	if s.statsService != nil {
		statsHandler := handlers.NewStatsHandler(s.statsService)
		statsHandler.RegisterRoutes(s.router)
		log.Printf("Stats endpoints available at /api/v1/stats/*")
	}

	// Register pricing routes
	if s.pricingService != nil && s.statsRepo != nil {
		pricingHandler := handlers.NewPricingHandler(s.pricingService, s.statsRepo)
		pricingHandler.RegisterRoutes(s.router)
		log.Printf("Pricing endpoints available at /api/v1/pricing/*")
	}

	// Register tool discovery routes
	toolsHandler := handlers.NewToolsHandler(s.toolRegistry)
	toolsHandler.RegisterRoutes(&muxRouteAdapter{s.router})

	// Prometheus metrics endpoint
	s.router.Handle("/metrics", metrics.Handler()).Methods("GET")

	log.Printf("Tool discovery endpoint available at /api/v1/tools")
	log.Printf("WebSocket endpoint available at /api/v1/ws")
	log.Printf("Prometheus metrics available at /metrics")
}

// muxRouteAdapter adapts mux.Router to the routeRegistrar interface.
// This enables the ToolsHandler to register routes without depending on mux directly.
type muxRouteAdapter struct {
	router *mux.Router
}

func (a *muxRouteAdapter) HandleFunc(path string, f func(http.ResponseWriter, *http.Request)) handlers.RouteMethoder {
	return &muxRouteMethoder{route: a.router.HandleFunc(path, f)}
}

type muxRouteMethoder struct {
	route *mux.Route
}

func (m *muxRouteMethoder) Methods(methods ...string) handlers.RouteMethoder {
	m.route.Methods(methods...)
	return m
}

// Router returns the HTTP handler for use with server.Run
func (s *Server) Router() http.Handler {
	return gorillaHandlers.RecoveryHandler()(s.router)
}

// Cleanup releases resources when the server shuts down
func (s *Server) Cleanup() error {
	// Stop the reconciler
	if s.reconciler != nil {
		if err := s.reconciler.Stop(); err != nil {
			s.log("reconciler shutdown failed", map[string]interface{}{"error": err.Error()})
		}
	}

	// Clean up database connection
	if s.db != nil {
		s.db.Close()
	}

	s.log("server stopped", nil)
	return nil
}

// loggingMiddleware prints simple request logs
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

// corsMiddleware adds CORS headers with configurable origins
// Set CORS_ALLOWED_ORIGINS env var to restrict (comma-separated list).
// Defaults to localhost-based origins for development safety.
func corsMiddleware(next http.Handler) http.Handler {
	allowedOrigins := getAllowedOrigins()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && isOriginAllowed(origin, allowedOrigins) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getAllowedOrigins returns the list of allowed CORS origins.
// Reads from CORS_ALLOWED_ORIGINS env var (comma-separated).
// Defaults to localhost patterns for development safety.
func getAllowedOrigins() []string {
	if origins := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS")); origins != "" {
		var result []string
		for _, o := range strings.Split(origins, ",") {
			if trimmed := strings.TrimSpace(o); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}
	// Default: allow localhost on common ports for development
	return []string{
		"http://localhost:*",
		"http://127.0.0.1:*",
	}
}

// isOriginAllowed checks if the origin matches any allowed pattern.
// Supports wildcard port matching with http://host:*
func isOriginAllowed(origin string, allowed []string) bool {
	for _, pattern := range allowed {
		if strings.HasSuffix(pattern, ":*") {
			// Wildcard port pattern: http://localhost:*
			prefix := strings.TrimSuffix(pattern, "*")
			if strings.HasPrefix(origin, prefix) {
				return true
			}
		} else if origin == pattern {
			return true
		}
	}
	return false
}

func (s *Server) log(msg string, fields map[string]interface{}) {
	if len(fields) == 0 {
		log.Println(msg)
		return
	}
	data, _ := json.Marshal(fields)
	log.Printf("%s | %s", msg, string(data))
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "agent-manager",
	}) {
		return // Process was re-exec'd after rebuild
	}

	srv, err := NewServer()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	if err := server.Run(server.Config{
		Handler: srv.Router(),
		Cleanup: func(ctx context.Context) error {
			return srv.Cleanup()
		},
	}); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
