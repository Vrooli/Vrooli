package wiring

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/adapters/webconsole"
	agentconfig "agent-manager/internal/config"
	"agent-manager/internal/database"
	"agent-manager/internal/domain"
	"agent-manager/internal/eventlog"
	"agent-manager/internal/handlers"
	healthstore "agent-manager/internal/health"
	"agent-manager/internal/identity"
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
	"agent-manager/internal/structuredresult"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/discovery"
)

// OrchestratorDependencies is the runtime service graph assembled by the
// composition root. It keeps main free of construction details while making
// every dependency consumed by server startup explicit.
type OrchestratorDependencies struct {
	Orchestrator          *orchestration.Orchestrator
	StatsService          orchestration.StatsService
	StatsRepository       repository.StatsRepository
	PricingService        pricing.Service
	Reconciler            *orchestration.Reconciler
	AwaitRegistry         *orchestration.AwaitRegistry
	WorkflowNudger        *orchestration.WorkflowNudger
	ModelHealthProbe      *healthstore.Probe
	RolePolicyState       *rolepolicy.State
	PermissionPolicyState *permissionpolicy.State
	PermissionPolicy      *permissionpolicy.Service
	StatsEngine           *stats.Engine
	HealthStore           *healthstore.Store
	EventRepository       eventlog.Repository
}

// NewOrchestrator builds the production orchestration graph. No background
// workers start here; Server startup owns lifecycle ordering separately.
func NewOrchestrator(db *database.DB, hub *handlers.WebSocketHub, logger *logrus.Logger, uploads storage.Service, levers *agentconfig.Levers) (OrchestratorDependencies, error) {
	if db == nil || db.DB == nil {
		return OrchestratorDependencies{}, fmt.Errorf("database is required")
	}
	if logger == nil {
		return OrchestratorDependencies{}, fmt.Errorf("logger is required")
	}
	if levers == nil {
		defaults := agentconfig.DefaultLevers()
		levers = &defaults
	}
	bootLog := obs.Component("bootstrap")
	bootLog.Info("using SQLite persistence")
	repos := database.NewRepositories(db, logger)
	eventStore := event.NewSQLiteStore(db.DB, logger)

	rolePolicyPath := rolepolicy.ResolvePath()
	roleState, err := rolepolicy.NewState(rolePolicyPath, rolepolicy.Requirement{Required: true, Reason: "portable role selection must resolve through resource-owned coding-agent policies"})
	if err != nil {
		bootLog.Error("required role policy catalog is not ready", "path", rolePolicyPath, obs.KeyError, err.Error())
	} else {
		bootLog.Info("role policy catalog activated", "path", rolePolicyPath, "digest", roleState.Status().ActiveDigest)
	}
	permissionPolicyPath := permissionpolicy.ResolvePath()
	permissionState, err := permissionpolicy.NewState(permissionPolicyPath, permissionpolicy.Requirement{Required: true, Reason: "global coding-agent permission intent must be validated before reconciliation"})
	if err != nil {
		bootLog.Error("required permission policy catalog is not ready", "path", permissionPolicyPath, obs.KeyError, err.Error())
	} else {
		bootLog.Info("permission policy catalog activated", "path", permissionPolicyPath, "digest", permissionState.Status().ActiveDigest)
	}
	permissionPolicy := permissionpolicy.NewService(permissionState, permissionpolicy.NewResourcePermissionProjector(nil), permissionpolicy.NewSQLiteAuditStore(db.DB))

	runners := NewRunners()
	registry := runners.Registry
	flagValidator := runner.NewRegistryFlagValidator(registry)
	sandboxURL := resolveWorkspaceSandboxURL()
	sandboxProvider := sandbox.NewWorkspaceSandboxProvider(sandboxURL)
	if runners.Claude != nil {
		runners.Claude.SetSandboxLauncherFactory(sandboxProvider)
	}
	if runners.Codex != nil {
		runners.Codex.SetSandboxLauncherFactory(sandboxProvider)
	}
	if runners.OpenCode != nil {
		runners.OpenCode.SetSandboxLauncherFactory(sandboxProvider)
	}

	settingsStore, err := agentconfig.NewOrchestrationSettingsStore(agentconfig.ResolveOrchestrationSettingsPath())
	if err != nil {
		bootLog.Warn("orchestration settings load failed", obs.KeyError, err.Error())
	}
	terminatorCfg := orchestration.DefaultTerminatorConfig()
	if settingsStore != nil {
		settings := settingsStore.Get()
		terminatorCfg = orchestration.TerminatorConfig{GracePeriod: time.Duration(settings.ProcessTermination.GracePeriodSeconds) * time.Second, MaxRetries: settings.ProcessTermination.TerminationMaxRetries, BaseBackoff: 500 * time.Millisecond, MaxBackoff: 5 * time.Second, VerifyTimeout: 2 * time.Second, KillProcessGroup: settings.ProcessTermination.KillProcessGroup}
	}
	terminator := orchestration.NewTerminator(repos.Runs, registry, terminatorCfg)

	healthStore := healthstore.NewStore(db.DB)
	runnerNames := make([]string, 0, len(domain.ValidRunnerTypes()))
	for _, runnerType := range domain.ValidRunnerTypes() {
		runnerNames = append(runnerNames, string(runnerType))
	}
	healthStore.RegisterRunners(runnerNames)

	orchConfig := orchestration.DefaultConfig()
	if base := agentconfig.Load(); base != nil {
		orchConfig.DefaultProjectRoot = strings.TrimSpace(base.Sandbox.ProjectRoot)
	}
	if levers != nil {
		orchConfig.DefaultTimeout = levers.Execution.DefaultTimeout
		orchConfig.MaxConcurrentRuns = levers.Concurrency.MaxConcurrentRuns
		orchConfig.RequireSandboxByDefault = levers.Safety.RequireSandboxByDefault
	}
	if settingsStore != nil {
		settings := settingsStore.Get()
		orchConfig.DefaultTimeout = time.Duration(settings.RunExecution.RunTimeoutMinutes) * time.Minute
		orchConfig.MaxConcurrentRuns = settings.RunExecution.MaxConcurrentRuns
		orchConfig.RequireSandboxByDefault = settings.SafetyIsolation.RequireSandbox
	}
	identitySecret, err := identity.LoadOrCreateSecret(database.DataDir())
	if err != nil {
		return OrchestratorDependencies{}, fmt.Errorf("initialize identity secret: %w", err)
	}
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
	workspaceEnsurer := orchestration.NewCommandWorkspaceSandboxEnsurer(sandboxProvider, levers.Sandbox)

	var interactiveSessions webconsole.SessionController
	if baseURL := webconsole.ResolveBaseURL(); baseURL != "" {
		interactiveSessions = webconsole.NewClient(baseURL, nil)
		log.Printf("interactive-runner: web-console session controller wired (base=%s)", baseURL)
	} else {
		log.Printf("interactive-runner: web-console API base unresolved; interactive runs disabled")
	}
	roleResolver := rolepolicy.NewResourceRoleResolver(nil)
	extractor := &structuredresult.RunnerExtractor{Roles: roleState, Resolver: roleResolver, Runners: registry, WorkingDir: database.DataDir(), Timeout: 2 * time.Minute}
	opts := []orchestration.Option{
		orchestration.WithConfig(orchConfig), orchestration.WithEvents(eventStore), orchestration.WithRunners(registry), orchestration.WithSandbox(sandboxProvider),
		orchestration.WithWorkspaceSandboxEnsurer(workspaceEnsurer), orchestration.WithCheckpoints(repos.Checkpoints), orchestration.WithIdempotency(repos.Idempotency),
		orchestration.WithWorkflowRepository(repos.Workflows), orchestration.WithWorkflowExecutionRepository(repos.WorkflowExecutions), orchestration.WithBroadcaster(hub),
		orchestration.WithTerminator(terminator), orchestration.WithStorageLabel("sqlite"), orchestration.WithRolePolicyState(roleState, roleResolver),
		orchestration.WithStructuredExtractor(extractor), orchestration.WithHealthStore(healthStore), orchestration.WithInvestigationSettings(repos.InvestigationSettings),
		orchestration.WithPromptClient(promptmanager.NewHTTPClient()), orchestration.WithFlagValidator(flagValidator), orchestration.WithAttachmentStorage(uploads),
		orchestration.WithOrchestrationSettings(settingsStore), orchestration.WithIdentitySecret(identitySecret), orchestration.WithSpawnDispatcher(spawnDispatcher),
	}
	if interactiveSessions != nil {
		opts = append(opts, orchestration.WithInteractiveSessions(interactiveSessions), orchestration.WithWebConsoleUIBase(webconsole.ResolveUIBaseURL()))
	}
	orch := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs, opts...)

	reconcilerCfg := orchestration.DefaultReconcilerConfig()
	if settingsStore != nil {
		settings := settingsStore.Get()
		reconcilerCfg = orchestration.ReconcilerConfig{Interval: time.Duration(settings.HealthDetection.ReconcilerIntervalSeconds) * time.Second, StaleThreshold: time.Duration(settings.HealthDetection.StaleThresholdSeconds) * time.Second, MaxRecoveryAge: time.Duration(settings.HealthDetection.MaxRecoveryAgeSeconds) * time.Second, OrphanGracePeriod: time.Duration(settings.ProcessTermination.OrphanGracePeriodSeconds) * time.Second, MaxStaleRuns: 10, PendingThreshold: 5 * time.Minute, KillOrphans: settings.ProcessTermination.KillOrphans, AutoRecover: true}
	}
	reconcilerOpts := []orchestration.ReconcilerOption{orchestration.WithReconcilerConfig(reconcilerCfg), orchestration.WithReconcilerEvents(eventStore), orchestration.WithReconcilerBroadcaster(hub), orchestration.WithReconcilerSandbox(sandboxProvider), orchestration.WithReconcilerWorkflowRecovery(orch), orchestration.WithReconcilerWorkflowWaitingLiveness(orch), orchestration.WithReconcilerPendingRunRecovery(orch)}
	if interactiveSessions != nil {
		reconcilerOpts = append(reconcilerOpts, orchestration.WithReconcilerInteractive(interactiveSessions))
	}
	reconciler := orchestration.NewReconciler(repos.Runs, registry, reconcilerOpts...)
	orch.SetReconciler(reconciler)
	awaitRegistry := orchestration.NewAwaitRegistry(orch, orchestration.NewTestGenieWaiter(nil), orchestration.NewGCTBaselineWaiter(nil), orchestration.NewLifecycleWaiter(nil))
	orch.SetAwaitRegistry(awaitRegistry)
	workflowNudger := orchestration.NewWorkflowNudger(orch.NudgeDrive, levers.Workflow.NudgeWorkers, levers.Workflow.NudgeDriveTimeout, func(ctx context.Context, id uuid.UUID, failure obs.PanicFailure) {
		if _, err := orch.FailWorkflowExecution(ctx, id, "nudger_panic", failure.Error()); err != nil {
			obs.Logger().Error("failed to persist recovered workflow nudger panic", "executionId", id.String(), obs.KeyError, err.Error())
		}
	})
	orch.SetWorkflowNudger(workflowNudger)

	pricingService := pricing.NewService(database.NewPricingRepository(db, logger), []pricing.Provider{providers.NewOpenRouterProvider()}, logger)
	modelResolver := func(runnerType string) healthstore.ModelProber {
		if registry == nil {
			return nil
		}
		prober, err := registry.Get(domain.RunnerType(runnerType))
		if err != nil {
			return nil
		}
		return prober
	}
	probeCfg := healthstore.DefaultProbeConfig()
	if raw := strings.TrimSpace(os.Getenv("AGENT_MANAGER_MODEL_HEALTH_INTERVAL")); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil {
			probeCfg.Interval = parsed
		} else {
			bootLog.Warn("invalid AGENT_MANAGER_MODEL_HEALTH_INTERVAL", "raw", raw, obs.KeyError, err.Error())
		}
	}
	eventRepo := eventlog.NewSQLiteRepository(db.DB)
	statsEngine := stats.NewEngine(eventRepo, stats.NewSQLiteCheckpointStore(db.DB), "operational")
	bootLog.Info("orchestrator initialized", "storage", "sqlite", "sandbox", sandboxURL)
	return OrchestratorDependencies{Orchestrator: orch, StatsService: orchestration.NewStatsOrchestrator(repos.Stats), StatsRepository: repos.Stats, PricingService: pricingService, Reconciler: reconciler, AwaitRegistry: awaitRegistry, WorkflowNudger: workflowNudger, ModelHealthProbe: healthstore.NewProbe(healthStore, nil, modelResolver, nil, probeCfg), RolePolicyState: roleState, PermissionPolicyState: permissionState, PermissionPolicy: permissionPolicy, StatsEngine: statsEngine, HealthStore: healthStore, EventRepository: eventRepo}, nil
}

func resolveWorkspaceSandboxURL() string {
	if configured := strings.TrimSpace(os.Getenv("WORKSPACE_SANDBOX_URL")); configured != "" {
		return configured
	}
	if resolved, err := discovery.ResolveScenarioURLDefault(context.Background(), "workspace-sandbox"); err == nil && resolved != "" {
		return resolved
	}
	port := os.Getenv("WORKSPACE_SANDBOX_API_PORT")
	if port == "" {
		port = "15427"
	}
	return fmt.Sprintf("http://localhost:%s", port)
}
