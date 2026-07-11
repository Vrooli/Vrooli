package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	capacitybroker "agent-manager/internal/adapters/capacity"
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/recommendation"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	runnercore "agent-manager/internal/adapters/runner/core"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/database"
	"agent-manager/internal/domain"
	"agent-manager/internal/eventlog"
	"agent-manager/internal/handlers"
	healthstore "agent-manager/internal/health"
	"agent-manager/internal/identity"
	"agent-manager/internal/metrics"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/orchestration/spawn"
	"agent-manager/internal/permissionpolicy"
	"agent-manager/internal/pricing"
	"agent-manager/internal/pricing/providers"
	"agent-manager/internal/promptmanager"
	"agent-manager/internal/repository"
	"agent-manager/internal/rolepolicy"
	"agent-manager/internal/stats"
	"agent-manager/internal/storage"

	agentconfig "agent-manager/internal/config"

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
	Port string
}

// Server wires the HTTP router, database, and orchestration service
type Server struct {
	config                *Config
	db                    *database.DB
	logger                *logrus.Logger
	router                *mux.Router
	orchestrator          orchestration.Service
	statsService          orchestration.StatsService
	statsRepo             repository.StatsRepository
	pricingService        pricing.Service
	wsHub                 *handlers.WebSocketHub
	reconciler            *orchestration.Reconciler
	awaitRegistry         *orchestration.AwaitRegistry
	recommendationWorker  *orchestration.RecommendationWorker
	modelHealthProbe      *healthstore.Probe
	rolePolicyState       *rolepolicy.State
	permissionPolicyState *permissionpolicy.State
	permissionPolicy      *permissionpolicy.Service
	storage               storage.Service
	statsEngine           *stats.Engine
	healthStore           *healthstore.Store
	eventRepo             eventlog.Repository
}

// NewServer initializes configuration, database, and routes
func NewServer() (*Server, error) {
	// Load levers first so the structured logger can be initialised
	// from the operator's chosen format/level before any orchestration
	// log site fires.
	levers, leversErr := agentconfig.LoadLevers()
	if levers == nil {
		defaults := agentconfig.DefaultLevers()
		levers = &defaults
	}
	obs.Init(levers.Observability.LogFormat, levers.Observability.LogLevel)
	if leversErr != nil {
		obs.Logger().Warn("config levers failed to load; using defaults", obs.KeyError, leversErr.Error())
	}

	// Initialize logrus for HTTP-ingress paths that still consume it
	// (database, pricing, websockets). The orchestration / runner /
	// phases layers use obs.Logger exclusively.
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	cfg := &Config{}

	// Database connection - SQLite is required, failure is fatal
	db, err := database.NewConnection(logger)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Create WebSocket hub for real-time event broadcasting (needed by orchestrator)
	wsHub := handlers.NewWebSocketHub()
	go wsHub.Run()

	// Create upload storage service
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "/tmp/agent-manager-uploads"
	}
	uploadStorage := storage.NewLocalService(uploadDir)

	// Create the orchestrator with appropriate repositories and broadcaster
	deps := createOrchestrator(db, wsHub, logger, uploadStorage, levers)

	// UseEncodedPath tells mux to match routes against the raw URL path (e.g., keeping %2F
	// as-is instead of decoding to /). This is required for model names containing slashes
	// like "aion-labs/aion-1.0" which are URL-encoded to "aion-labs%2Faion-1.0".
	srv := &Server{
		config:                cfg,
		db:                    db,
		logger:                logger,
		router:                mux.NewRouter().UseEncodedPath(),
		orchestrator:          deps.orchestrator,
		statsService:          deps.statsService,
		statsRepo:             deps.statsRepo,
		pricingService:        deps.pricingService,
		modelHealthProbe:      deps.modelHealthProbe,
		rolePolicyState:       deps.rolePolicyState,
		permissionPolicyState: deps.permissionPolicyState,
		permissionPolicy:      deps.permissionPolicy,
		wsHub:                 wsHub,
		reconciler:            deps.reconciler,
		awaitRegistry:         deps.awaitRegistry,
		recommendationWorker:  deps.recommendationWorker,
		storage:               uploadStorage,
		statsEngine:           deps.statsEngine,
		healthStore:           deps.healthStore,
		eventRepo:             deps.eventRepo,
	}

	// Start the reconciler for orphan detection and stale run recovery
	if srv.reconciler != nil {
		if err := srv.reconciler.RecoverInFlightRuns(context.Background()); err != nil {
			obs.Logger().Warn("initial run recovery failed", obs.KeyError, err.Error())
		}
		if err := srv.reconciler.Start(context.Background()); err != nil {
			obs.Logger().Warn("reconciler start failed", obs.KeyError, err.Error())
		}
	}

	// Re-spawn waiters for any runs that were parked when agent-manager last
	// stopped (durable park/resume restart recovery).
	if srv.awaitRegistry != nil {
		if n, err := srv.awaitRegistry.RecoverParkedRuns(context.Background()); err != nil {
			obs.Logger().Warn("parked-run waiter recovery failed", obs.KeyError, err.Error())
		} else if n > 0 {
			obs.Logger().Info("re-spawned waiters for parked runs", "count", n)
		}
	}

	// Start the recommendation worker for passive extraction from investigation runs
	if srv.recommendationWorker != nil {
		if err := srv.recommendationWorker.Start(context.Background()); err != nil {
			obs.Logger().Warn("recommendation worker start failed", obs.KeyError, err.Error())
		}
	}

	// Start the model health probe (startup sweep + periodic refresh).
	if srv.modelHealthProbe != nil {
		srv.modelHealthProbe.Start(context.Background())
	}

	// Boot the operational stats engine: replay (or resume from
	// checkpoint) so the engine reflects all historical events before
	// the first request lands. Failure is logged but not fatal — the
	// engine recovers on first Refresh.
	if srv.statsEngine != nil {
		if err := srv.statsEngine.Rebuild(context.Background()); err != nil {
			obs.Logger().Warn("stats engine rebuild failed", obs.KeyError, err.Error())
		}
	}

	srv.setupRoutes()
	return srv, nil
}

// orchestratorDeps holds the orchestrator and related services
type orchestratorDeps struct {
	orchestrator          orchestration.Service
	statsService          orchestration.StatsService
	statsRepo             repository.StatsRepository
	pricingService        pricing.Service
	reconciler            *orchestration.Reconciler
	awaitRegistry         *orchestration.AwaitRegistry
	recommendationWorker  *orchestration.RecommendationWorker
	modelHealthProbe      *healthstore.Probe
	rolePolicyState       *rolepolicy.State
	permissionPolicyState *permissionpolicy.State
	permissionPolicy      *permissionpolicy.Service
	statsEngine           *stats.Engine
	healthStore           *healthstore.Store
	eventRepo             eventlog.Repository
}

// createOrchestrator creates the orchestration service with all dependencies.
// levers is pre-loaded by NewServer so observability is initialised before the
// first log site fires.
func createOrchestrator(db *database.DB, wsHub *handlers.WebSocketHub, logger *logrus.Logger, uploadStorage storage.Service, levers *agentconfig.Levers) orchestratorDeps {
	bootLog := obs.Component("bootstrap")
	bootLog.Info("using SQLite persistence")
	storageLabel := "sqlite"
	eventStore := event.NewSQLiteStore(db.DB, logger)
	repos := database.NewRepositories(db, logger)
	profileRepo := repos.Profiles
	taskRepo := repos.Tasks
	runRepo := repos.Runs
	checkpointRepo := repos.Checkpoints
	idempotencyRepo := repos.Idempotency
	statsRepo := repos.Stats
	investigationSettingsRepo := repos.InvestigationSettings

	rolePolicyPath := rolepolicy.ResolvePath()
	rolePolicyState, err := rolepolicy.NewState(rolePolicyPath, rolepolicy.Requirement{
		Required: true,
		Reason:   "portable role selection must resolve through resource-owned coding-agent policies",
	})
	if err != nil {
		bootLog.Error("required role policy catalog is not ready", "path", rolePolicyPath, obs.KeyError, err.Error())
	} else {
		bootLog.Info("role policy catalog activated", "path", rolePolicyPath, "digest", rolePolicyState.Status().ActiveDigest)
	}

	permissionPolicyPath := permissionpolicy.ResolvePath()
	permissionPolicyState, err := permissionpolicy.NewState(permissionPolicyPath, permissionpolicy.Requirement{
		Required: true,
		Reason:   "global coding-agent permission intent must be validated before reconciliation",
	})
	if err != nil {
		bootLog.Error("required permission policy catalog is not ready", "path", permissionPolicyPath, obs.KeyError, err.Error())
	} else {
		bootLog.Info("permission policy catalog activated", "path", permissionPolicyPath, "digest", permissionPolicyState.Status().ActiveDigest)
	}
	permissionPolicy := permissionpolicy.NewService(
		permissionPolicyState,
		permissionpolicy.NewResourcePermissionProjector(nil),
		permissionpolicy.NewSQLiteAuditStore(db.DB),
	)

	// Create runner registry
	runnerRegistry := runner.NewRegistry()

	// Register Claude Code runner (codecs.Claude wired through core.Runner).
	hostLauncher := runner.NewHostLauncher()
	var claudeRunner *runnercore.Runner
	if claudeCodec, err := codecs.NewClaude(); err != nil {
		bootLog.Warn("Claude Code codec construction failed", obs.KeyRunnerType, string(domain.RunnerTypeClaudeCode), obs.KeyError, err.Error())
		if err := runnerRegistry.Register(runner.NewStubRunner(
			domain.RunnerTypeClaudeCode,
			fmt.Sprintf("claude-code runner failed to initialize: %v", err),
		)); err != nil {
			bootLog.Warn("stub Claude runner registration failed", obs.KeyRunnerType, string(domain.RunnerTypeClaudeCode), obs.KeyError, err.Error())
		}
	} else {
		claudeRunner = runnercore.NewRunner(claudeCodec, hostLauncher, nil)
		if err := runnerRegistry.Register(claudeRunner); err != nil {
			bootLog.Warn("Claude runner registration failed", obs.KeyRunnerType, string(domain.RunnerTypeClaudeCode), obs.KeyError, err.Error())
		}
		if avail, msg := claudeRunner.IsAvailable(context.Background()); avail {
			bootLog.Info("runner available", obs.KeyRunnerType, string(domain.RunnerTypeClaudeCode))
		} else {
			bootLog.Warn("runner unavailable", obs.KeyRunnerType, string(domain.RunnerTypeClaudeCode), obs.KeyMessage, msg)
		}
	}

	// Register Codex runner (codecs.Codex wired through core.Runner).
	var codexRunner *runnercore.Runner
	if codexCodec, err := codecs.NewCodex(); err != nil {
		bootLog.Warn("Codex codec construction failed", obs.KeyRunnerType, string(domain.RunnerTypeCodex), obs.KeyError, err.Error())
		if err := runnerRegistry.Register(runner.NewStubRunner(
			domain.RunnerTypeCodex,
			fmt.Sprintf("codex runner failed to initialize: %v", err),
		)); err != nil {
			bootLog.Warn("stub Codex runner registration failed", obs.KeyRunnerType, string(domain.RunnerTypeCodex), obs.KeyError, err.Error())
		}
	} else {
		codexRunner = runnercore.NewRunner(codexCodec, hostLauncher, nil)
		if err := runnerRegistry.Register(codexRunner); err != nil {
			bootLog.Warn("Codex runner registration failed", obs.KeyRunnerType, string(domain.RunnerTypeCodex), obs.KeyError, err.Error())
		}
		if avail, msg := codexRunner.IsAvailable(context.Background()); avail {
			bootLog.Info("runner available", obs.KeyRunnerType, string(domain.RunnerTypeCodex))
		} else {
			bootLog.Warn("runner unavailable", obs.KeyRunnerType, string(domain.RunnerTypeCodex), obs.KeyMessage, msg)
		}
	}

	// Register OpenCode runner (codecs.OpenCode wired through core.Runner).
	var openCodeRunner *runnercore.Runner
	if openCodeCodec, err := codecs.NewOpenCode(); err != nil {
		bootLog.Warn("OpenCode codec construction failed", obs.KeyRunnerType, string(domain.RunnerTypeOpenCode), obs.KeyError, err.Error())
		if err := runnerRegistry.Register(runner.NewStubRunner(
			domain.RunnerTypeOpenCode,
			fmt.Sprintf("opencode runner failed to initialize: %v", err),
		)); err != nil {
			bootLog.Warn("stub OpenCode runner registration failed", obs.KeyRunnerType, string(domain.RunnerTypeOpenCode), obs.KeyError, err.Error())
		}
	} else {
		openCodeRunner = runnercore.NewRunner(openCodeCodec, hostLauncher, nil)
		if err := runnerRegistry.Register(openCodeRunner); err != nil {
			bootLog.Warn("OpenCode runner registration failed", obs.KeyRunnerType, string(domain.RunnerTypeOpenCode), obs.KeyError, err.Error())
		}
		if avail, msg := openCodeRunner.IsAvailable(context.Background()); avail {
			bootLog.Info("runner available", obs.KeyRunnerType, string(domain.RunnerTypeOpenCode))
		} else {
			bootLog.Warn("runner unavailable", obs.KeyRunnerType, string(domain.RunnerTypeOpenCode), obs.KeyMessage, msg)
		}
	}

	// Register Grok runner (codecs.Grok wired through core.Runner).
	// Permission posture (D4): the codec maps SkipPermissionPrompt to grok's
	// --always-approve and does NOT write per-run permission rules; the grok
	// resource's operator-configured PreToolUse deny hook enforces (it holds
	// even under --always-approve). Per-run enforcement is a filed follow-up.
	var grokRunner *runnercore.Runner
	if grokCodec, err := codecs.NewGrok(); err != nil {
		bootLog.Warn("Grok codec construction failed", obs.KeyRunnerType, string(domain.RunnerTypeGrok), obs.KeyError, err.Error())
		if err := runnerRegistry.Register(runner.NewStubRunner(
			domain.RunnerTypeGrok,
			fmt.Sprintf("grok runner failed to initialize: %v", err),
		)); err != nil {
			bootLog.Warn("stub Grok runner registration failed", obs.KeyRunnerType, string(domain.RunnerTypeGrok), obs.KeyError, err.Error())
		}
	} else {
		grokRunner = runnercore.NewRunner(grokCodec, hostLauncher, nil)
		if err := runnerRegistry.Register(grokRunner); err != nil {
			bootLog.Warn("Grok runner registration failed", obs.KeyRunnerType, string(domain.RunnerTypeGrok), obs.KeyError, err.Error())
		}
		if avail, msg := grokRunner.IsAvailable(context.Background()); avail {
			bootLog.Info("runner available", obs.KeyRunnerType, string(domain.RunnerTypeGrok))
		} else {
			bootLog.Warn("runner unavailable", obs.KeyRunnerType, string(domain.RunnerTypeGrok), obs.KeyMessage, msg)
		}
	}

	// Create flag validator for runner-specific CLI flag validation
	flagValidator := runner.NewRegistryFlagValidator(runnerRegistry)

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

	// Wire the protected-mode SandboxLauncherFactory into every coding-
	// agent runner. The provider implements runner.SandboxLauncherFactory
	// directly. Runs whose ResolvedConfig.SandboxConfig.Mode == Protected
	// will route the agent process through workspace-sandbox /processes;
	// other runs continue to use the HostLauncher unchanged. See
	// execute/protected-sandbox-agent-launch.
	if claudeRunner != nil {
		claudeRunner.SetSandboxLauncherFactory(sandboxProvider)
	}
	if codexRunner != nil {
		codexRunner.SetSandboxLauncherFactory(sandboxProvider)
	}
	if openCodeRunner != nil {
		openCodeRunner.SetSandboxLauncherFactory(sandboxProvider)
	}

	// Load orchestration settings (file-backed, git-checked-in).
	orchSettingsPath := agentconfig.ResolveOrchestrationSettingsPath()
	orchSettingsStore, err := agentconfig.NewOrchestrationSettingsStore(orchSettingsPath)
	if err != nil {
		bootLog.Warn("orchestration settings load failed", "path", orchSettingsPath, obs.KeyError, err.Error())
	}

	// Build terminator config from orchestration settings (or defaults).
	terminatorCfg := orchestration.DefaultTerminatorConfig()
	if orchSettingsStore != nil {
		os := orchSettingsStore.Get()
		terminatorCfg = orchestration.TerminatorConfig{
			GracePeriod:      time.Duration(os.ProcessTermination.GracePeriodSeconds) * time.Second,
			MaxRetries:       os.ProcessTermination.TerminationMaxRetries,
			BaseBackoff:      500 * time.Millisecond,
			MaxBackoff:       5 * time.Second,
			VerifyTimeout:    2 * time.Second,
			KillProcessGroup: os.ProcessTermination.KillProcessGroup,
		}
	}

	// Create terminator for robust process termination (Phase 2)
	terminator := orchestration.NewTerminator(
		runRepo,
		runnerRegistry,
		terminatorCfg,
	)

	healthStore := healthstore.NewStore(db.DB)
	catalogRunners := make([]string, 0, len(domain.ValidRunnerTypes()))
	for _, runnerType := range domain.ValidRunnerTypes() {
		catalogRunners = append(catalogRunners, string(runnerType))
	}
	healthStore.RegisterRunners(catalogRunners)

	orchConfig := orchestration.DefaultConfig()
	baseConfig := agentconfig.Load()
	if baseConfig != nil {
		orchConfig.DefaultProjectRoot = strings.TrimSpace(baseConfig.Sandbox.ProjectRoot)
	}
	if levers != nil {
		orchConfig.DefaultTimeout = levers.Execution.DefaultTimeout
		orchConfig.MaxConcurrentRuns = levers.Concurrency.MaxConcurrentRuns
		orchConfig.RequireSandboxByDefault = levers.Safety.RequireSandboxByDefault
	}

	// Apply orchestration settings on top of levers (settings file is the primary source).
	if orchSettingsStore != nil {
		os := orchSettingsStore.Get()
		orchConfig.DefaultTimeout = time.Duration(os.RunExecution.RunTimeoutMinutes) * time.Minute
		orchConfig.MaxConcurrentRuns = os.RunExecution.MaxConcurrentRuns
		orchConfig.RequireSandboxByDefault = os.SafetyIsolation.RequireSandbox
	}

	// Create prompt-manager client for investigation prompt skills
	promptClient := promptmanager.NewHTTPClient()

	// Create recommendation extractor for investigation outputs. It holds an
	// advisory op-scoped capacity claim around each ollama generate (plan §7
	// Phase 7 ollama adopter) so the broker sees ollama as actively in use.
	recommendationExtractor := recommendation.NewOllamaExtractor().WithCapacity(&capacitybroker.CLIBroker{})

	// Load identity signing secret for agent identity tokens.
	identitySecret, err := identity.LoadOrCreateSecret(database.DataDir())
	if err != nil {
		log.Fatalf("Failed to initialize identity secret: %v", err)
	}

	// Build the spawn dispatcher up front so it can be installed on
	// the orchestrator. Defaults come from levers; QueueCapacity ==
	// 0 means "auto-derive from MaxConcurrentRuns".
	spawnCfg := spawn.Config{MaxStartingConcurrency: 1, QueueCapacity: orchConfig.MaxConcurrentRuns * 2}
	if levers != nil {
		spawnCfg.MaxStartingConcurrency = levers.Spawn.MaxStartingConcurrency
		spawnCfg.MinSpacing = levers.Spawn.MinSpacing
		if levers.Spawn.QueueCapacity > 0 {
			spawnCfg.QueueCapacity = levers.Spawn.QueueCapacity
		}
	}
	if spawnCfg.QueueCapacity < spawnCfg.MaxStartingConcurrency {
		spawnCfg.QueueCapacity = spawnCfg.MaxStartingConcurrency
	}
	spawnDispatcher := spawn.New(spawnCfg)
	workspaceSandboxEnsurer := orchestration.NewCommandWorkspaceSandboxEnsurer(sandboxProvider, levers.Sandbox)

	// Build orchestrator with all dependencies including WebSocket broadcaster and terminator
	orch := orchestration.New(
		profileRepo,
		taskRepo,
		runRepo,
		orchestration.WithConfig(orchConfig),
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(runnerRegistry),
		orchestration.WithSandbox(sandboxProvider),
		orchestration.WithWorkspaceSandboxEnsurer(workspaceSandboxEnsurer),
		orchestration.WithCheckpoints(checkpointRepo),
		orchestration.WithIdempotency(idempotencyRepo),
		orchestration.WithBroadcaster(wsHub),
		orchestration.WithTerminator(terminator),
		orchestration.WithStorageLabel(storageLabel),
		orchestration.WithRolePolicyState(rolePolicyState, rolepolicy.NewResourceRoleResolver(nil)),
		orchestration.WithHealthStore(healthStore),
		orchestration.WithRecommendationExtractor(recommendationExtractor),
		orchestration.WithInvestigationSettings(investigationSettingsRepo),
		orchestration.WithPromptClient(promptClient),
		orchestration.WithFlagValidator(flagValidator),
		orchestration.WithAttachmentStorage(uploadStorage),
		orchestration.WithOrchestrationSettings(orchSettingsStore),
		orchestration.WithIdentitySecret(identitySecret),
		orchestration.WithSpawnDispatcher(spawnDispatcher),
	)

	// Build reconciler config from orchestration settings (or defaults).
	reconcilerCfg := orchestration.DefaultReconcilerConfig()
	if orchSettingsStore != nil {
		os := orchSettingsStore.Get()
		reconcilerCfg = orchestration.ReconcilerConfig{
			Interval:          time.Duration(os.HealthDetection.ReconcilerIntervalSeconds) * time.Second,
			StaleThreshold:    time.Duration(os.HealthDetection.StaleThresholdSeconds) * time.Second,
			MaxRecoveryAge:    time.Duration(os.HealthDetection.MaxRecoveryAgeSeconds) * time.Second,
			OrphanGracePeriod: time.Duration(os.ProcessTermination.OrphanGracePeriodSeconds) * time.Second,
			MaxStaleRuns:      10,
			KillOrphans:       os.ProcessTermination.KillOrphans,
			AutoRecover:       true,
		}
	}

	// Create reconciler for orphan detection and stale run recovery (Phase 2)
	reconciler := orchestration.NewReconciler(
		runRepo,
		runnerRegistry,
		orchestration.WithReconcilerConfig(reconcilerCfg),
		orchestration.WithReconcilerEvents(eventStore),
		orchestration.WithReconcilerBroadcaster(wsHub),
		orchestration.WithReconcilerSandbox(sandboxProvider),
	)

	// Wire reconciler back to orchestrator for hot-reload propagation.
	orch.SetReconciler(reconciler)

	// Durable park/wait registry (Phase 3): one background waiter per parked
	// run's await-handle, driven by the per-producer Waiter seam. Built after the
	// orchestrator (it is the registry's waker) and wired back via setter,
	// mirroring the reconciler. Restart recovery (re-spawning waiters for
	// persisted parked runs) runs in startServer alongside reconciler recovery.
	awaitRegistry := orchestration.NewAwaitRegistry(
		orch,
		orchestration.NewTestGenieWaiter(nil),
		orchestration.NewGCTBaselineWaiter(nil),
		orchestration.NewLifecycleWaiter(nil),
	)
	orch.SetAwaitRegistry(awaitRegistry)

	// Create recommendation worker for passive extraction from investigation runs
	// Uses the investigation settings repository for tag allowlist filtering
	allowlistProvider := orchestration.NewSettingsAllowlistProvider(investigationSettingsRepo)
	recommendationWorker := orchestration.NewRecommendationWorker(
		runRepo,
		eventStore,
		recommendationExtractor,
		orchestration.WithRecommendationWorkerConfig(orchestration.RecommendationWorkerConfig{
			Interval:      30 * time.Second,
			MaxRetries:    3,
			RetryBackoff:  1 * time.Minute,
			MaxConcurrent: 1, // Serial processing to avoid overloading Ollama
		}),
		orchestration.WithRecommendationWorkerBroadcaster(wsHub),
		orchestration.WithRecommendationWorkerAllowlist(allowlistProvider),
	)

	// Create stats service for analytics
	statsSvc := orchestration.NewStatsOrchestrator(statsRepo)

	// Create pricing service for model pricing management
	pricingRepo := database.NewPricingRepository(db, logger)
	openRouterProvider := providers.NewOpenRouterProvider()
	pricingProviders := []pricing.Provider{openRouterProvider}
	pricingSvc := pricing.NewService(pricingRepo, pricingProviders, logger)

	// Model health probe: periodically re-checks each registered model. The probe is
	// intentionally cheap (no live inference) — the authoritative signal comes from
	// runtime classification in the executor. The probe ensures a fresh snapshot
	// after restart and surfaces hard binary-missing states.
	modelResolver := func(runnerType string) healthstore.ModelProber {
		if runnerRegistry == nil {
			return nil
		}
		r, err := runnerRegistry.Get(domain.RunnerType(runnerType))
		if err != nil {
			return nil
		}
		return r
	}
	probeCfg := healthstore.DefaultProbeConfig()
	if raw := strings.TrimSpace(envOrEmpty("AGENT_MANAGER_MODEL_HEALTH_INTERVAL")); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil {
			probeCfg.Interval = parsed
		} else {
			bootLog.Warn("invalid AGENT_MANAGER_MODEL_HEALTH_INTERVAL", "raw", raw, obs.KeyError, err.Error())
		}
	}
	// Models are resource-owned runtime facts. The health store continues to
	// record execution observations, but Agent Manager does not schedule model
	// probes from a static local inventory.
	modelHealthProbe := healthstore.NewProbe(healthStore, nil, modelResolver, nil, probeCfg)

	// Operational stats engine: incrementally aggregates typed-operational
	// events (fallbacks, health transitions, sandbox ops, heartbeat misses,
	// checkpoint failures, retries) into the metrics surface consumed by
	// /api/v1/stats/operational and the stats CLI commands. Distinct from
	// statsSvc above (which derives from run rows). Watermark persists in
	// stats_checkpoint so a crash resumes mid-replay instead of from zero.
	eventRepo := eventlog.NewSQLiteRepository(db.DB)
	checkpointStore := stats.NewSQLiteCheckpointStore(db.DB)
	statsEngine := stats.NewEngine(eventRepo, checkpointStore, "operational")

	bootLog.Info("orchestrator initialized", "storage", storageLabel, "sandbox", sandboxURL)
	return orchestratorDeps{
		orchestrator:          orch,
		statsService:          statsSvc,
		statsRepo:             statsRepo,
		pricingService:        pricingSvc,
		reconciler:            reconciler,
		awaitRegistry:         awaitRegistry,
		recommendationWorker:  recommendationWorker,
		modelHealthProbe:      modelHealthProbe,
		rolePolicyState:       rolePolicyState,
		permissionPolicyState: permissionPolicyState,
		permissionPolicy:      permissionPolicy,
		statsEngine:           statsEngine,
		healthStore:           healthStore,
		eventRepo:             eventRepo,
	}
}

func envOrEmpty(key string) string {
	v := os.Getenv(key)
	return v
}

func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)
	s.router.Use(corsMiddleware)

	// Health endpoint using api-core/health for standardized response format
	var rawDB *sql.DB
	if s.db != nil && s.db.DB != nil {
		rawDB = s.db.DB.DB // Access underlying *sql.DB from sqlx.DB (which is embedded in database.DB)
	}
	healthHandler := health.New().
		Version("1.0.0").
		Check(health.DB(rawDB), health.Critical).
		Check(rolePolicyHealthChecker(s.rolePolicyState), health.Critical).
		Check(permissionPolicyHealthChecker(s.permissionPolicyState), health.Critical).
		Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	// Detailed health for UI (includes sandbox + runner dependencies).
	// Keep /health minimal for infra probes.
	handler := handlers.New(
		s.orchestrator,
		handlers.WithStorage(s.storage),
		handlers.WithRolePolicyState(s.rolePolicyState),
		handlers.WithPermissionPolicy(s.permissionPolicyState, s.permissionPolicy),
	)
	handler.SetWebSocketHub(s.wsHub)
	s.router.HandleFunc("/api/v1/health", handler.Health).Methods("GET")

	// Register all API routes via the handlers package
	// WebSocket hub was created in NewServer and is shared with orchestrator
	handler.RegisterRoutes(s.router)

	routesLog := obs.Component("routes")

	// Register stats routes
	if s.statsService != nil {
		statsHandler := handlers.NewStatsHandler(s.statsService)
		statsHandler.RegisterRoutes(s.router)
		routesLog.Info("stats endpoints registered", "path", "/api/v1/stats/*")
	}

	// Register operational-event stats endpoints (typed event log).
	if s.statsEngine != nil {
		opStatsHandler := handlers.NewOperationalStatsHandler(s.statsEngine)
		opStatsHandler.RegisterRoutes(s.router)
		routesLog.Info("operational stats endpoints registered", "path", "/api/v1/stats/operational, /api/v1/stats/fallback")
	}

	// Register persisted health audit endpoints.
	if s.healthStore != nil {
		healthAuditHandler := handlers.NewHealthAuditHandler(s.healthStore)
		healthAuditHandler.RegisterRoutes(s.router)
		routesLog.Info("health audit endpoints registered", "path", "/api/v1/health/{models,runners,audit}")
	}

	// Register typed-event read endpoint.
	if s.eventRepo != nil {
		eventsHandler := handlers.NewEventsHandler(s.eventRepo)
		eventsHandler.RegisterRoutes(s.router)
		routesLog.Info("events endpoint registered", "path", "/api/v1/events")
	}

	// Register pricing routes
	if s.pricingService != nil && s.statsRepo != nil {
		pricingHandler := handlers.NewPricingHandler(s.pricingService, s.statsRepo)
		pricingHandler.RegisterRoutes(s.router)
		routesLog.Info("pricing endpoints registered", "path", "/api/v1/pricing/*")
	}

	// Prometheus metrics endpoint
	s.router.Handle("/metrics", metrics.Handler()).Methods("GET")

	routesLog.Info("websocket endpoint registered", "path", "/api/v1/ws")
	routesLog.Info("metrics endpoint registered", "path", "/metrics")
}

// rolePolicyHealthChecker keeps portable role authority a readiness
// dependency. A run can never safely resolve a role from an absent catalog.
func rolePolicyHealthChecker(state *rolepolicy.State) health.Checker {
	if state == nil {
		return nil
	}
	return health.CheckerFunc(func(context.Context) health.CheckResult {
		status := state.Status()
		if err := state.ReadinessError(); err != nil {
			detail := health.NewErrorDetail(
				rolepolicy.DiagnosticCodeCatalogInvalid,
				err.Error(),
				"configuration",
				true,
			)
			detail.Details = map[string]any{
				"path":               status.Path,
				"required":           status.Requirement.Required,
				"requirement_reason": status.Requirement.Reason,
				"active_digest":      status.ActiveDigest,
			}
			return health.CheckResult{Name: "role_policy_catalog", Connected: false, Error: detail}
		}
		return health.CheckResult{Name: "role_policy_catalog", Connected: true}
	})
}

// permissionPolicyHealthChecker separates invalid global desired state from
// resource availability. Resource projection is an explicit operator action;
// it is never performed as a side effect of an infrastructure health probe.
func permissionPolicyHealthChecker(state *permissionpolicy.State) health.Checker {
	if state == nil {
		return nil
	}
	return health.CheckerFunc(func(context.Context) health.CheckResult {
		status := state.Status()
		if err := state.ReadinessError(); err != nil {
			detail := health.NewErrorDetail(
				permissionpolicy.DiagnosticCodeCatalogInvalid,
				err.Error(),
				"configuration",
				true,
			)
			detail.Details = map[string]any{
				"path":               status.Path,
				"required":           status.Requirement.Required,
				"requirement_reason": status.Requirement.Reason,
				"active_digest":      status.ActiveDigest,
			}
			return health.CheckResult{Name: "permission_policy_catalog", Connected: false, Error: detail}
		}
		return health.CheckResult{Name: "permission_policy_catalog", Connected: true}
	})
}

// Router returns the HTTP handler for use with server.Run
func (s *Server) Router() http.Handler {
	return gorillaHandlers.RecoveryHandler()(s.router)
}

// Cleanup releases resources when the server shuts down
func (s *Server) Cleanup() error {
	shutdownLog := obs.Component("shutdown")

	// Stop the reconciler
	if s.reconciler != nil {
		if err := s.reconciler.Stop(); err != nil {
			shutdownLog.Warn("reconciler shutdown failed", obs.KeyError, err.Error())
		}
	}

	// Stop the durable park/wait waiters (parked runs stay durable; their
	// handles are persisted and re-spawned on the next boot).
	if s.awaitRegistry != nil {
		s.awaitRegistry.Stop()
	}

	// Stop the recommendation worker
	if s.recommendationWorker != nil {
		if err := s.recommendationWorker.Stop(); err != nil {
			shutdownLog.Warn("recommendation worker shutdown failed", obs.KeyError, err.Error())
		}
	}

	// Clean up database connection
	if s.db != nil {
		s.db.Close()
	}

	shutdownLog.Info("server stopped")
	return nil
}

// loggingMiddleware prints simple request logs
func loggingMiddleware(next http.Handler) http.Handler {
	httpLog := obs.Component("http")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		httpLog.Debug("request",
			"method", r.Method,
			"uri", r.RequestURI,
			obs.KeyDuration, time.Since(start).Milliseconds(),
		)
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
		// Extended timeouts for LLM-based operations (e.g., recommendation extraction)
		WriteTimeout: 3 * time.Minute,
		ReadTimeout:  1 * time.Minute,
		Cleanup: func(ctx context.Context) error {
			return srv.Cleanup()
		},
	}); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
