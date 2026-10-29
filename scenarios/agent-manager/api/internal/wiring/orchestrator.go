package wiring

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-manager/internal/adapters/artifact"
	"agent-manager/internal/adapters/database"
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/adapters/webconsole"
	agentconfig "agent-manager/internal/config"
	"agent-manager/internal/domain"
	durabilitysource "agent-manager/internal/durability"
	"agent-manager/internal/eventlog"
	"agent-manager/internal/handlers"
	healthstore "agent-manager/internal/health"
	"agent-manager/internal/identity"
	"agent-manager/internal/invocationreadmodel"
	investigationlearning "agent-manager/internal/learning"
	"agent-manager/internal/modelpolicydrift"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/orchestration/spawn"
	"agent-manager/internal/permissionpolicy"
	"agent-manager/internal/piimask"
	"agent-manager/internal/pricing"
	"agent-manager/internal/pricing/providers"
	"agent-manager/internal/promptmanager"
	"agent-manager/internal/repository"
	"agent-manager/internal/rolepolicy"
	"agent-manager/internal/runstate"
	"agent-manager/internal/stats"
	"agent-manager/internal/storage"
	"agent-manager/internal/structuredresult"
	"agent-manager/internal/supervision"
	"agent-manager/internal/usageforecast"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/operatordecision"
)

// OrchestratorDependencies is the runtime service graph assembled by the
// composition root. It keeps main free of construction details while making
// every dependency consumed by server startup explicit.
type OrchestratorDependencies struct {
	OwnerIdentity         authn.TokenVerifier
	Orchestrator          *orchestration.Orchestrator
	StatsRepository       repository.StatsRepository
	PricingService        pricing.Service
	PricingRepository     pricing.Repository
	Reconciler            *orchestration.Reconciler
	AwaitRegistry         *orchestration.AwaitRegistry
	WorkflowNudger        *orchestration.WorkflowNudger
	TranscriptImporter    *orchestration.TranscriptIndex
	FrictionPublisher     *orchestration.FrictionPublishScheduler
	ClaudeQuotaFeed       *pricing.ClaudeStatuslineReader // statusline samples -> quota_observations
	UsageAlerts           *usageforecast.Alerter          // will_exhaust -> notification-hub
	ModelHealthProbe      *healthstore.Probe
	ModelPolicyDrift      *modelpolicydrift.Scheduler
	RolePolicyState       *rolepolicy.State
	PermissionPolicyState *permissionpolicy.State
	PermissionPolicy      *permissionpolicy.Service
	StatsEngine           *stats.Engine
	HealthStore           *healthstore.Store
	EventRepository       eventlog.Repository
	SupervisionService    *supervision.Service
	SupervisionScheduler  *supervision.Scheduler
	WatchActionAuthorizer handlers.WatchActionAuthorizer
	InvocationReadModel   invocationreadmodel.Store
	// DecisionPublisher raises operator decisions through Vrooli Events; nil
	// when Vrooli Events is unresolved.
	DecisionPublisher operatordecision.Publisher
	WorkspaceSandbox  interface {
		IsAvailable(context.Context) (bool, string)
	}
}

// NewOrchestrator builds the production orchestration graph. No background
// workers start here; Server startup owns lifecycle ordering separately.
func NewOrchestrator(db *database.DB, hub *handlers.WebSocketHub, logger *logrus.Logger, uploads storage.Service, levers *agentconfig.Levers, fileRoots ...*filerouting.RoutedRoots) (OrchestratorDependencies, error) {
	if db == nil || db.Routed == nil {
		return OrchestratorDependencies{}, fmt.Errorf("database is required")
	}
	if logger == nil {
		return OrchestratorDependencies{}, fmt.Errorf("logger is required")
	}
	if levers == nil {
		defaults := agentconfig.DefaultLevers()
		levers = &defaults
	}
	if len(fileRoots) == 0 || fileRoots[0] == nil {
		return OrchestratorDependencies{}, fmt.Errorf("durable file roots are required for Agent Manager startup")
	}
	bootLog := obs.Component("bootstrap")
	var runStateResolver runstate.RootResolver
	runStateResolver = runstate.RoutedRoot{Roots: fileRoots[0]}
	if validator, ok := runStateResolver.(interface{ Validate(context.Context) error }); ok {
		if err := validator.Validate(context.Background()); err != nil {
			return OrchestratorDependencies{}, fmt.Errorf("validate durable run state root: %w", err)
		}
	}
	bootLog.Info("using SQLite persistence")
	repos := database.NewRepositories(db, logger)
	eventStore := event.NewSQLiteStore(db, logger)

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
	permissionPolicy := permissionpolicy.NewService(permissionState, permissionpolicy.NewResourcePermissionProjector(nil), permissionpolicy.NewSQLiteAuditStore(db))

	pricingRepository := database.NewPricingRepository(db, logger)
	pricingService := pricing.NewServiceWithModelResolver(pricingRepository, []pricing.Provider{providers.NewOpenRouterProvider(), providers.NewOpenCodeGoProvider()}, logger, pricing.NewCLIModelResolver())
	quotaObservationStore, _ := pricingRepository.(pricing.QuotaObservationRepository)
	startPricingLifecycle(pricingService, bootLog)
	runners := NewRunners(pricingCodecAdapter{service: pricingService})
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
	if runners.Antigravity != nil {
		runners.Antigravity.SetSandboxLauncherFactory(sandboxProvider)
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

	healthStore := healthstore.NewStore(db)
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
		// Present Agent Manager's operator-issued service principal so Web
		// Console can pin a verified stable-create owner.
		serviceTokens := webconsole.NewServiceTokenSource(webconsole.AuthorityServiceCredential, webconsole.AuthenticatorServiceExchanger(nil), nil)
		interactiveSessions = webconsole.NewClient(baseURL, webconsole.NewServicePrincipalHTTPClient(serviceTokens))
		log.Printf("interactive-runner: web-console session controller wired (base=%s)", baseURL)
	} else {
		log.Printf("interactive-runner: web-console API base unresolved; interactive runs disabled")
	}
	roleResolver := rolepolicy.NewResourceRoleResolver(nil)
	var artifactCollector *artifact.SQLiteCollector
	if len(fileRoots) > 0 && fileRoots[0] != nil {
		artifactCollector = artifact.NewSQLiteCollector(db, fileRoots[0])
	}
	extractor := structuredresult.NewGatewayExtractor(nil)
	receiptsBaseURL, _ := os.LookupEnv("VROOLI_EVENTS_API_BASE")
	if receiptsBaseURL == "" {
		receiptsBaseURL, _ = discovery.ResolveScenarioURLDefault(context.Background(), "vrooli-events")
	}
	receiptsClient := eventbus.Client{BaseURL: receiptsBaseURL}
	receiptTargets, receiptTargetsErr := declaredReceiptTargets(receiptCaptureDeclarationPath())
	if receiptTargetsErr != nil {
		bootLog.Warn("receipt capture declaration unavailable", obs.KeyError, receiptTargetsErr.Error())
	}
	receiptReader := newReceiptSummaryReader(receiptsClient, receiptTargets, productionReceiptRuntimeReader)
	ownerIdentity := authn.NewScenarioAuthenticatorProvider(authn.JWTConfig{
		Audience: strings.TrimSpace(os.Getenv("VROOLI_AUTH_SCENARIO_AUDIENCE")),
	})
	runtimeOwnerIdentity := "agent-manager:" + uuid.NewString()
	actionAuthorizer := watchActionAuthorizer{orchestrator: nil, owners: ownerIdentity}
	opts := []orchestration.Option{
		orchestration.WithConfig(orchConfig), orchestration.WithEvents(eventStore), orchestration.WithQuotaObservationStore(quotaObservationStore), orchestration.WithRunners(registry), orchestration.WithSandbox(sandboxProvider),
		orchestration.WithWorkspaceSandboxEnsurer(workspaceEnsurer), orchestration.WithCheckpoints(repos.Checkpoints), orchestration.WithIdempotency(repos.Idempotency),
		orchestration.WithWorkflowRepository(repos.Workflows), orchestration.WithWorkflowExecutionRepository(repos.WorkflowExecutions), orchestration.WithBroadcaster(hub),
		orchestration.WithTerminator(terminator), orchestration.WithStorageLabel("sqlite"), orchestration.WithRolePolicyState(roleState, roleResolver),
		orchestration.WithStructuredExtractor(extractor), orchestration.WithLabelGenerator(extractor), orchestration.WithHealthStore(healthStore), orchestration.WithInvestigationSettings(repos.InvestigationSettings),
		orchestration.WithPromptClient(promptmanager.NewHTTPClient()), orchestration.WithFlagValidator(flagValidator), orchestration.WithAttachmentStorage(uploads),
		orchestration.WithOrchestrationSettings(settingsStore), orchestration.WithIdentitySecret(identitySecret), orchestration.WithSpawnDispatcher(spawnDispatcher),
		orchestration.WithRunStateRootResolver(runStateResolver), orchestration.WithArtifacts(artifactCollector), orchestration.WithReceiptSummaryReader(receiptReader), orchestration.WithFindings(repos.Findings), orchestration.WithInvestigationLifecycleRepository(repos.Investigations), orchestration.WithInvestigationLearningRecorder(investigationlearning.NewClient()), orchestration.WithReceiptEvidenceStore(repos.ReceiptEvidence), orchestration.WithInvestigationLedgerStore(repos.InvestigationLedger), orchestration.WithInvocationReadModel(repos.InvocationReadModel), orchestration.WithDurabilityBoundary(repos.DurabilityBoundary),
		orchestration.WithOwnerIdentity(ownerIdentity),
		orchestration.WithAuthorizations(database.NewAuthorizationRepository(db)),
		orchestration.WithSessionBindingProofReader(orchestration.NewCLISessionBindingProofReader()),
		orchestration.WithStandingGrants(database.NewStandingGrantRepository(db)),
		orchestration.WithPendingAuthorizations(database.NewPendingAuthorizationRepository(db)),
		orchestration.WithRuntimeOwnerIdentity(runtimeOwnerIdentity),
		// Weighted-token caps (DL-8) stay off unless AGENT_MANAGER_TOKEN_CAP_MODE
		// selects report or enforce.
		orchestration.WithTokenCapPolicy(orchestration.TokenCapPolicyFromEnv()),
	}
	if interactiveSessions != nil {
		opts = append(opts, orchestration.WithInteractiveSessions(interactiveSessions))
	}
	if levers != nil {
		importPolicy, policyErr := levers.ConversationPolicy.Policy()
		if policyErr != nil {
			return OrchestratorDependencies{}, fmt.Errorf("compile conversation import policy: %w", policyErr)
		}
		if importPolicy.Enabled() {
			opts = append(opts, orchestration.WithConversationPolicy(importPolicy))
			// Wire the PII masker alongside the policy so a `redact` rule can
			// actually scrub PII (via the ai-gateway DETECT_PII route) before a
			// matched run is indexed. The detector is resolved at call time and
			// invoked only for a redact-matched import, so an operator with only
			// exclude/index rules pays nothing. Fail-closed is owned by the masker.
			opts = append(opts, orchestration.WithPIIMasker(piimask.New(piimask.NewGatewayDetector())))
		}
	}
	// Pin the web-console UI base only when an operator fixed one via the
	// environment. Otherwise leave it unset so the orchestrator resolves a
	// request-correct, tunnel-aware URL per link (a localhost port here would
	// strand the review/outcome notifications on a phone).
	if uiBaseOverride := webconsole.ResolveUIBaseOverride(); uiBaseOverride != "" {
		opts = append(opts, orchestration.WithWebConsoleUIBase(uiBaseOverride))
	}
	var decisionPublisher operatordecision.Publisher
	if receiptsClient.Enabled() {
		decisionPublisher = receiptsClient
		// Run approvals (needs_review) publish *.decision_requested.v1 here so
		// notification-hub alerts the operator with an Approve/Reject ask.
		opts = append(opts, orchestration.WithApprovalEventPublisher(receiptsClient))
	}
	orch := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs, opts...)
	actionAuthorizer.orchestrator = orch
	orch.SetDurabilityEvidenceReader(durabilitysource.NewSwarmEvidenceClient())

	reconcilerCfg := orchestration.DefaultReconcilerConfig()
	if settingsStore != nil {
		reconcilerCfg = orchestration.ReconcilerConfigWithSettings(reconcilerCfg, settingsStore.Get())
		reconcilerCfg.OwnerIdentity = runtimeOwnerIdentity
	}
	reconcilerOpts := []orchestration.ReconcilerOption{orchestration.WithReconcilerConfig(reconcilerCfg), orchestration.WithReconcilerEvents(eventStore), orchestration.WithReconcilerEventRetention(eventStore), orchestration.WithReconcilerArtifactRetention(artifactCollector), orchestration.WithReconcilerBroadcaster(hub), orchestration.WithReconcilerSandbox(sandboxProvider), orchestration.WithReconcilerWorkflowRecovery(orch), orchestration.WithReconcilerTerminalAccounting(orch), orchestration.WithReconcilerWorkflowWaitingLiveness(orch), orchestration.WithReconcilerPendingRunRecovery(orch), orchestration.WithReconcilerRunStateRootResolver(runStateResolver)}
	if interactiveSessions != nil {
		reconcilerOpts = append(reconcilerOpts, orchestration.WithReconcilerInteractive(interactiveSessions))
	}
	reconciler := orchestration.NewReconciler(repos.Runs, registry, reconcilerOpts...)
	orch.SetReconciler(reconciler)
	awaitRegistry := orchestration.NewAwaitRegistry(orch, orchestration.NewTestGenieWaiter(nil), orchestration.NewGCTBaselineWaiter(nil), orchestration.NewLifecycleWaiter(nil), orchestration.NewChildrenWaiter(orch, 0), orchestration.NewOperatorDecisionWaiter())
	orch.SetAwaitRegistry(awaitRegistry)
	workflowNudger := orchestration.NewWorkflowNudger(orch.NudgeDrive, levers.Workflow.NudgeWorkers, levers.Workflow.NudgeDriveTimeout, func(ctx context.Context, id uuid.UUID, failure obs.PanicFailure) {
		if _, err := orch.FailWorkflowExecution(ctx, id, "nudger_panic", failure.Error()); err != nil {
			obs.Logger().Error("failed to persist recovered workflow nudger panic", "executionId", id.String(), obs.KeyError, err.Error())
		}
	})
	orch.SetWorkflowNudger(workflowNudger)

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
	raw, _ := os.LookupEnv("AGENT_MANAGER_MODEL_HEALTH_INTERVAL")
	if raw = strings.TrimSpace(raw); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil {
			probeCfg.Interval = parsed
		} else {
			bootLog.Warn("invalid AGENT_MANAGER_MODEL_HEALTH_INTERVAL", "raw", raw, obs.KeyError, err.Error())
		}
	}
	eventRepo := eventlog.NewSQLiteRepository(db)
	supervisionRepo := supervision.NewRepository(db)
	supervisionPolicies := supervision.NewPolicyStore(db, supervision.NewSourceLedgerOutcomeSink())
	if _, err := supervisionPolicies.EnsureInitialActive(context.Background(), supervision.DefaultSupervisionPolicy(), "agent-manager-bootstrap"); err != nil {
		return OrchestratorDependencies{}, fmt.Errorf("initialize supervision policy: %w", err)
	}
	supervisionService := supervision.NewService(supervisionRepo, eventRepo)
	supervisionService.SetPolicyStore(supervisionPolicies)
	orch.SetFamilySupervision(supervisionService)
	supervisionActions := supervision.NewActionService(supervisionRepo, supervisionRunController{orchestrator: orch})
	supervisionService.SetActionService(supervisionActions)
	programEvaluator := supervision.NewProgramRuntimeEvaluator(supervisionPolicies)
	supervisionService.SetPolicyArtifactResolver(programEvaluator.CurrentArtifactDigest)
	supervisionPolicies.SetReplayEvaluator(programEvaluator)
	supervisionEvaluator := supervision.PolicyControlledEvaluator{Store: supervisionPolicies, Delegate: programEvaluator}
	frictionSource, _ := repos.InvocationReadModel.(supervision.FrictionSource)
	subjectResolver := supervision.NewRunSubjectResolver(repos.Runs, frictionSource)
	supervisionService.SetSubjectResolver(subjectResolver)
	supervisionProcessor := supervision.NewProcessor(supervisionService, subjectResolver, supervisionEvaluator)
	supervisionScheduler := supervision.NewScheduler(supervisionRepo, supervisionProcessor, func(err error) {
		obs.Component("cohort-supervision").Warn("watch processing failed", obs.KeyError, err.Error())
	})
	supervisionScheduler.SetPolicyStore(supervisionPolicies)
	effortConfig := effortDiscoveryConfig()
	supervisionService.Efforts = supervision.NewEffortService(supervisionRepo, supervisionRunController{orchestrator: orch}, effortConfig)
	supervisionService.Efforts.SetRunRegistry(repos.Runs)
	effortSignalCursors, ok := repos.Runs.(supervision.EffortSignalCursorStore)
	if !ok {
		return OrchestratorDependencies{}, fmt.Errorf("run repository does not support retained effort signal cursors")
	}
	supervisionService.Efforts.SetEffortSignalCursorStore(effortSignalCursors)
	supervisionService.Efforts.SetQuotaObservationStore(quotaObservationStore)
	supervisionService.Efforts.SetSpendReader(orch)
	orch.SetGoalHomeParkSource(supervisionService.Efforts)
	orch.SetDeliveryEfforts(supervisionService.Efforts)
	if receiptsClient.Enabled() {
		// Operator decisions (DL-5) publish decision_requested events here.
		supervisionService.Efforts.SetDecisionPublisher(receiptsClient)
	}
	supervisionScheduler.SetEffortService(supervisionService.Efforts)
	supervisionService.SetSchedulerKick(supervisionScheduler.Kick)
	awaitRegistry.RegisterWaiter(orchestration.NewSupervisionWaiter(supervisionService))
	statsEngine := stats.NewEngine(eventRepo, stats.NewSQLiteCheckpointStore(db), "operational")
	modelPolicyRoot := strings.TrimSpace(os.Getenv("PROJECT_ROOT"))
	if modelPolicyRoot == "" {
		modelPolicyRoot, _ = filepath.Abs(filepath.Join("..", "..", ".."))
	}
	modelPolicyDrift := modelpolicydrift.New(modelPolicyRoot, "", modelPolicyDriftInterval(), modelPolicyReporter{client: promptmanager.NewHTTPClient()})
	usageAlerts := &usageforecast.Alerter{Store: usageforecast.NewSQLiteAlertStore(db), UsageURL: usagePageURL, OnError: func(err error) {
		obs.Component("usage-alerts").Warn("will-exhaust evaluation failed", obs.KeyError, err.Error())
	}}
	usageAlerts.Readings, _ = pricingRepository.(pricing.QuotaReadingSource)
	if receiptsClient.Enabled() {
		usageAlerts.Publisher = receiptsClient
	}
	bootLog.Info("orchestrator initialized", "storage", "sqlite", "sandbox", sandboxURL)
	return OrchestratorDependencies{OwnerIdentity: ownerIdentity, Orchestrator: orch, StatsRepository: repos.Stats, PricingService: pricingService, PricingRepository: pricingRepository, Reconciler: reconciler, AwaitRegistry: awaitRegistry, WorkflowNudger: workflowNudger, TranscriptImporter: newTranscriptIndex(orch, repos.TranscriptIndex, runStateResolver, levers), FrictionPublisher: orchestration.NewFrictionPublishScheduler(orch, frictionPublishInterval()), ClaudeQuotaFeed: &pricing.ClaudeStatuslineReader{Path: pricing.DefaultClaudeStatuslineSamplesPath(), Store: quotaObservationStore}, UsageAlerts: usageAlerts, ModelHealthProbe: NewModelHealthProbe(healthStore, nil, modelResolver, probeCfg), ModelPolicyDrift: modelPolicyDrift, DecisionPublisher: decisionPublisher, RolePolicyState: roleState, PermissionPolicyState: permissionState, PermissionPolicy: permissionPolicy, StatsEngine: statsEngine, HealthStore: healthStore, EventRepository: eventRepo, SupervisionService: supervisionService, SupervisionScheduler: supervisionScheduler, WatchActionAuthorizer: actionAuthorizer, InvocationReadModel: repos.InvocationReadModel, WorkspaceSandbox: sandboxProvider}, nil
}

// usagePageURL links a will-exhaust notification to the Agent Manager Usage
// page. Out-of-band links may reach a phone, so the tunnel-aware notice base
// is preferred; an unresolved base omits the link.
func usagePageURL(ctx context.Context) string {
	base, err := discovery.ResolveExternalURLForNotice(ctx, "agent-manager")
	if err != nil || strings.TrimSpace(base) == "" {
		return ""
	}
	return strings.TrimRight(base, "/") + "/usage"
}

func frictionPublishInterval() time.Duration {
	raw := strings.TrimSpace(os.Getenv("AGENT_MANAGER_FRICTION_PUBLISH_INTERVAL"))
	if raw == "" {
		return 6 * time.Hour
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed < time.Minute {
		return time.Minute
	}
	if parsed > 24*time.Hour {
		return 24 * time.Hour
	}
	return parsed
}

func modelPolicyDriftInterval() time.Duration {
	value := strings.TrimSpace(os.Getenv("AGENT_MANAGER_MODEL_POLICY_DRIFT_INTERVAL"))
	if value == "" {
		return 24 * time.Hour
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 || parsed > 14*24*time.Hour {
		return 24 * time.Hour
	}
	return parsed
}

func startPricingLifecycle(service pricing.Service, log *slog.Logger) {
	if service == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if err := service.RefreshPricing(ctx); err != nil {
			log.Warn("pricing warm refresh failed", obs.KeyError, err.Error())
		}
	}()
	go func() {
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			if err := service.RefreshPricing(ctx); err != nil {
				log.Warn("scheduled pricing refresh failed", obs.KeyError, err.Error())
			}
			cancel()
		}
	}()
}

func resolveWorkspaceSandboxURL() string {
	configured, _ := os.LookupEnv("WORKSPACE_SANDBOX_URL")
	if configured = strings.TrimSpace(configured); configured != "" {
		return configured
	}
	if resolved, err := discovery.ResolveScenarioURLDefault(context.Background(), "workspace-sandbox"); err == nil && resolved != "" {
		return resolved
	}
	port, _ := os.LookupEnv("WORKSPACE_SANDBOX_API_PORT")
	if strings.TrimSpace(port) == "" {
		// An unavailable sandbox is represented explicitly. Sandboxed launches
		// then fail with the provider's actionable connectivity error instead of
		// being silently routed to an arbitrary legacy port.
		return ""
	}
	return fmt.Sprintf("http://localhost:%s", port)
}
