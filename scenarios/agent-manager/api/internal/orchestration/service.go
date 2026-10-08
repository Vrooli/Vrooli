// This file defines the orchestration service composition and shared dependencies.
package orchestration

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/adapters/artifact"
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/adapters/webconsole"
	agentconfig "agent-manager/internal/config"
	"agent-manager/internal/domain"
	"agent-manager/internal/durability"
	"agent-manager/internal/findings"
	"agent-manager/internal/health"
	"agent-manager/internal/investigation"
	"agent-manager/internal/invocationreadmodel"
	investigationlearning "agent-manager/internal/learning"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/orchestration/phases"
	"agent-manager/internal/orchestration/spawn"
	"agent-manager/internal/policy"
	"agent-manager/internal/pricing"
	"agent-manager/internal/promptmanager"
	"agent-manager/internal/repository"
	"agent-manager/internal/rolepolicy"
	"agent-manager/internal/runreport"
	"agent-manager/internal/runstate"
	"agent-manager/internal/storage"
	"agent-manager/internal/structuredresult"
	"agent-manager/internal/supervision"
	"agent-manager/internal/workflowruntime"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/effortauthority"
	isolation "github.com/vrooli/vrooli/packages/nativeisolation"
)

// Orchestrator coordinates agent execution using injected dependencies.
type Orchestrator struct {
	authorizations  domain.AuthorizationRepository
	maintenanceGate interface {
		Admit(context.Context) (func(), error)
	}
	familySupervision *supervision.Service
	familyOwners      *familyOwnerClients
	// wakeMu serializes the parked→running claim. The durable run repository is
	// intentionally a simple whole-row update, so two waiter notifications that
	// arrive concurrently must not both observe parked and start continuations.
	wakeMu                            sync.Mutex
	serialScanMu                      sync.Mutex
	serialScanFrom, serialScanThrough time.Time
	serialScanOffset                  int
	// parkTurnEnds holds, per recently parked run, a channel closed once the
	// parked turn's agent process has been stopped. WakeRun waits on it so a
	// continuation never starts beside the still-running parked turn.
	parkTurnEnds sync.Map // uuid.UUID -> chan struct{}
	// parkCompactions holds, per parked run whose session is being compacted,
	// a channel closed when the compaction ends. WakeRun waits on it.
	parkCompactions sync.Map // uuid.UUID -> chan struct{}
	parkCompaction  *parkCompactionSettings
	// tokenCaps enforces weighted-token caps from persisted usage; nil is off.
	tokenCaps *tokenCapEnforcer

	// terminalAccounting remembers which ended standalone runs have settled
	// terminal usage, so the reconcile sweep revisits only runs that still owe it.
	terminalAccounting terminalAccountingSweep

	// Repositories (persistence)
	profiles              repository.ProfileRepository
	workflows             repository.WorkflowRepository
	workflowExecutions    repository.WorkflowExecutionRepository
	tasks                 repository.TaskRepository
	runs                  repository.RunRepository
	checkpoints           repository.CheckpointRepository            // For resumption support
	idempotency           repository.IdempotencyRepository           // For replay safety
	investigationSettings repository.InvestigationSettingsRepository // For investigation config

	// Adapters (external integrations)
	runners           runner.Registry
	sandbox           sandbox.Provider
	workspaceSandbox  phases.WorkspaceSandboxEnsurer
	events            event.Store
	quotaObservations pricing.QuotaObservationRepository
	artifacts         artifact.Collector
	runStateRoot      string
	runStateResolver  runstate.RootResolver

	// Policy evaluation
	policy policy.Evaluator

	// Real-time event broadcasting (WebSocket)
	broadcaster EventBroadcaster

	// Robust termination (Phase 2)
	terminator *Terminator

	// credentialUseReleaser is a best-effort post-commit hook owned by the
	// credential authority integration. Terminal run transitions revoke every
	// run-bound credential capability before the run is exposed as finished.
	credentialUseReleaser CredentialUseReleaser

	// Flag validation
	flagValidator runner.FlagValidator

	// Configuration
	config OrchestratorConfig
	// ownerIdentity names this Agent Manager owner lifetime for durable run
	// stream and finalization fencing.
	runtimeOwnerIdentity string

	// clock is the wall-clock seam for orchestration state transitions. It is
	// injected by deterministic tests and defaults to time.Now in production.
	clock func() time.Time

	// Storage label for health reporting (e.g., sqlite).
	storageLabel string

	rolePolicy   *rolepolicy.State
	roleResolver rolepolicy.Resolver

	// Model + runner health audit (SQLite-persisted, populated by runtime
	// classification + the periodic probe). Snapshots derive from
	// MAX(timestamp) over the audit tables; nil store means health
	// observability is disabled (writes are silent no-ops, snapshots
	// return empty).
	healthStore *health.Store

	// Prompt-manager client for reading investigation prompts from skills.
	promptClient promptmanager.Client

	// File storage for uploaded attachments.
	storage storage.Service

	// receipts is an optional, read-only Vrooli Events seam. It must never
	// influence run execution or terminal state; reports surface its status.
	receipts            ReceiptSummaryReader
	findings            findings.Repository
	typedInvestigations investigation.Repository
	receiptEvidence     runreport.ReceiptJoinStore
	investigationLedger runreport.LedgerStore
	invocationReadModel invocationreadmodel.Store
	durabilityEvidence  DurabilityEvidenceReader
	durabilityBoundary  durability.BoundaryStore
	learningRecorder    investigationlearning.Recorder

	// Orchestration settings store (file-backed, hot-reloadable).
	orchestrationSettings *agentconfig.OrchestrationSettingsStore

	// Reconciler reference for hot-reload propagation.
	reconciler *Reconciler

	// Identity signing secret for agent identity tokens.
	identitySecret []byte
	// ownerIdentity verifies an owner JWT before its subject and scope ceiling
	// are admitted into a run. A missing provider is permitted for internal
	// callers that do not present an owner token; a presented token never
	// falls back to an unverified identity.
	ownerIdentity   authn.TokenVerifier
	effortAuthority effortauthority.Engine
	// Admission-only seam is private; production installs a concrete ready
	// factory. Launch and terminal ownership never use this interface.
	finiteNativeFactory interface {
		Enabled() bool
		CheckBinding(string, isolation.Binding) error
	}
	finiteNativeTerminal interface {
		Terminal(context.Context, string) error
	}

	// dispatcher serializes runner startups and exposes queue depth.
	// All run-spawn paths (CreateRun, ResumeRun) MUST go through it —
	// see contract decision 2 in scenarios/agent-manager/docs/internal/SEAMS.md.
	dispatcher *spawn.Dispatcher

	// awaitRegistry drives durable park/wait (Phase 3): it owns a background
	// waiter per parked run's await-handle, wakes the run on resolve/deadline,
	// and re-spawns waiters for persisted parked runs after a restart. Nil
	// disables the auto-waiter (park/wake transitions still work; nothing drives
	// them) — wired post-construction via SetAwaitRegistry to break the
	// registry↔orchestrator construction cycle (mirrors SetReconciler).
	awaitRegistry *AwaitRegistry

	// interactiveSessions is the web-console session controller that drives
	// interactive runs (ExecutionMode=interactive): agent-manager launches the
	// real interactive CLI in a web-console tmux session and tails its
	// agent-owned transcript. Nil when interactive mode is not wired — an
	// interactive run then fails cleanly at execute time rather than hanging.
	interactiveSessions webconsole.SessionController

	// webConsoleUIBase is the resolved web-console UI base URL (browser-facing
	// origin), captured once at wiring time. Used to build the run-detail deep
	// link (Run.WebConsoleSessionURL) for interactive runs. Empty when the UI
	// base could not be resolved — the deep link is then omitted and clients
	// fall back to the session id.
	webConsoleUIBase string

	// interactiveDrivers tracks the live interactive coordinators agent-manager
	// owns (the initial Execute turn and any Continue-driven follow-up turn) so
	// StopRun can cancel the coordinator deterministically and wait for it to
	// exit before finalizing — see stopInteractiveRun. Always non-nil (set in
	// New).
	interactiveDrivers *interactiveDriverRegistry

	// structuredResults owns the deterministic-first typed-output projection.
	// It is always present; its optional extractor is selected by portable role.
	structuredResults phases.StructuredResultResolver
	labelGenerator    structuredresult.Extractor
	workflowEngine    *workflowruntime.Engine

	// workflowNudger drives a parent execution forward when one of its runs (or
	// child workflows) reaches terminal, so no consumer polls Advance. Nil
	// disables completion-driven advance — the reconciler recovery sweep still
	// re-drives non-terminal executions. Wired post-construction via
	// SetWorkflowNudger to break the nudger↔orchestrator construction cycle
	// (mirrors SetAwaitRegistry).
	workflowNudger *WorkflowNudger

	// workflowWaiters backs the blocking WaitWorkflowExecution RPC: an
	// event-driven notifier the drive path fires when an execution settles
	// terminal. Always non-nil (set in New) so waits work even before the
	// nudger is wired.
	workflowWaiters *workflowWaitRegistry

	// conversationSearchNotify is a post-commit accelerator for the derived
	// conversation indexes. Notification failure never rolls back or changes the
	// outcome of a canonical write; periodic source comparison repairs misses.
	conversationSearchNotify func(context.Context, string, string, string) error
	// conversationSearchRevive clears an external publication tombstone when
	// an operator explicitly imports or republishes the same provenance.
	conversationSearchRevive func(context.Context, string, string) error
}

// CredentialUseReleaser revokes capabilities bound to a terminal agent run.
// Implementations must be idempotent and must never return credential values.
type CredentialUseReleaser interface {
	RevokeRunCredentialUse(context.Context, uuid.UUID, string) error
}

// WithCredentialUseReleaser installs the terminal credential cleanup seam.
func WithCredentialUseReleaser(releaser CredentialUseReleaser) Option {
	return func(o *Orchestrator) { o.credentialUseReleaser = releaser }
}

// SetCredentialUseReleaser installs the terminal credential cleanup seam after
// construction, which keeps scenario discovery and wiring order flexible.
func (o *Orchestrator) SetCredentialUseReleaser(releaser CredentialUseReleaser) {
	if o != nil {
		o.credentialUseReleaser = releaser
	}
}

// SetConversationSearchNotifier installs the derived-index post-commit hook.
func (o *Orchestrator) SetConversationSearchNotifier(notify func(context.Context, string, string, string) error) {
	if o != nil {
		o.conversationSearchNotify = notify
	}
}

// SetConversationSearchReviver installs the post-import tombstone revival
// hook. Import is an explicit write intent, so it may restore a previously
// tombstoned derived projection without changing the owning run lifecycle.
func (o *Orchestrator) SetConversationSearchReviver(revive func(context.Context, string, string) error) {
	if o != nil {
		o.conversationSearchRevive = revive
	}
}

func (o *Orchestrator) reviveConversationSearch(ctx context.Context, harness, sessionID string) error {
	if o == nil || o.conversationSearchRevive == nil || strings.TrimSpace(harness) == "" || strings.TrimSpace(sessionID) == "" {
		return nil
	}
	return o.conversationSearchRevive(ctx, harness, sessionID)
}

func (o *Orchestrator) notifyConversationSearch(ctx context.Context, operation, runID, eventID string) {
	if o == nil || o.conversationSearchNotify == nil {
		return
	}
	if err := o.conversationSearchNotify(ctx, operation, runID, eventID); err != nil {
		obs.Component("conversation-search").Warn("post-commit index notification failed", "operation", operation, "run_id", runID, obs.KeyError, err.Error())
	}
}

// SetDurabilityEvidenceReader installs the swarm-owned read seam. It is a
// setter because scenario discovery is resolved after the orchestrator is
// constructed. A missing reader is represented as unlinked evidence.
func (o *Orchestrator) SetDurabilityEvidenceReader(reader DurabilityEvidenceReader) {
	o.durabilityEvidence = reader
}

// WithDurabilityBoundary installs the durable analysis-epoch store. Without it
// Durability refuses to grade rather than inventing an epoch.
func WithDurabilityBoundary(store durability.BoundaryStore) Option {
	return func(o *Orchestrator) {
		o.durabilityBoundary = store
	}
}

// Durability projects a run's durable friction and optional swarm evidence.
// It deliberately never reads Run.Result, Run.Status, or any other
// self-authored completion field.
func (o *Orchestrator) Durability(ctx context.Context, id uuid.UUID) (durability.Verdict, error) {
	run, err := o.GetRun(ctx, id)
	if err != nil {
		return durability.Verdict{}, err
	}
	started := time.Time{}
	if run.StartedAt != nil {
		started = run.StartedAt.UTC()
	} else if run.ImportedAt != nil {
		started = run.ImportedAt.UTC()
	}
	lane := durability.LaneUnlinked
	if run.IdentityTokenHash != "" {
		lane = durability.LaneVerified
	} else if run.ImportSourceSessionID != "" {
		lane = durability.LaneObserved
	}
	evidence := make([]durability.Evidence, 0)
	episodes, err := o.Episodes(ctx, id)
	if err != nil {
		return durability.Verdict{}, err
	}
	for _, episode := range episodes {
		evidence = append(evidence, durability.Evidence{Kind: "friction", Reference: "agent-manager://runs/" + id.String() + "/episodes/" + episode.EpisodeID, At: started, Lane: lane})
	}
	if o.durabilityEvidence == nil {
		// An unconfigured source was never consulted. Staying silent here would
		// let a run with no local friction report durable while the entire
		// pushback and rework lane went unread.
		evidence = append(evidence, durability.Evidence{Kind: "swarm-evidence-not-configured", Reference: "swarm-manager://durability/evidence", At: started, Lane: durability.LaneUnlinked, Degraded: true})
	} else if observed, readErr := o.durabilityEvidence.ReadDurabilityEvidence(ctx, run); readErr != nil {
		evidence = append(evidence, durability.Evidence{Kind: "swarm-evidence-unavailable", Reference: "swarm-manager://durability/evidence", At: started, Lane: durability.LaneUnlinked, Degraded: true})
	} else {
		evidence = append(evidence, observed...)
	}
	boundary, err := durability.ResolveBoundary(ctx, o.durabilityBoundary, systemNow())
	if err != nil {
		return durability.Verdict{}, err
	}
	return durability.Project(boundary, durability.Work{ID: id.String(), Subject: append([]string(nil), run.Subject...), StartedAt: started, Lane: lane}, evidence), nil
}

// systemNow is the production clock behind injected orchestration clocks.
var systemNow = time.Now

// OrchestratorConfig holds service configuration.
type OrchestratorConfig struct {
	DefaultTimeout          time.Duration
	MaxConcurrentRuns       int
	DefaultProjectRoot      string
	RequireSandboxByDefault bool
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() OrchestratorConfig {
	return OrchestratorConfig{
		DefaultTimeout:          60 * time.Minute,
		MaxConcurrentRuns:       10,
		RequireSandboxByDefault: true,
	}
}

// Option configures the Orchestrator.
type Option func(*Orchestrator)

// ReceiptSummaryReader provides only the bounded receipt discriminator needed
// by RunReport. Receipt payloads remain on the dedicated inspection endpoint.
type ReceiptSummaryReader interface {
	ReadReceiptSummary(context.Context, uuid.UUID) (runreport.ReceiptSummary, error)
}

type ReceiptSummaryReaderFunc func(context.Context, uuid.UUID) (runreport.ReceiptSummary, error)

func (f ReceiptSummaryReaderFunc) ReadReceiptSummary(ctx context.Context, id uuid.UUID) (runreport.ReceiptSummary, error) {
	return f(ctx, id)
}

// WithConfig sets the configuration.
func WithConfig(cfg OrchestratorConfig) Option {
	return func(o *Orchestrator) {
		o.config = cfg
	}
}

// WithOwnerIdentity installs the verifier for presented scenario-authenticator
// owner tokens. Validation remains fail-closed whenever a token is supplied.
func WithOwnerIdentity(provider authn.TokenVerifier) Option {
	return func(o *Orchestrator) {
		o.ownerIdentity = provider
	}
}

// WithClock injects the wall clock used for durable orchestration timestamps.
// A nil clock retains the production default.
func WithClock(clock func() time.Time) Option {
	return func(o *Orchestrator) {
		if clock != nil {
			o.clock = clock
		}
	}
}

// WithRunners sets the runner registry.
func WithRunners(r runner.Registry) Option {
	return func(o *Orchestrator) {
		o.runners = r
	}
}

// WithSandbox sets the sandbox provider.
func WithSandbox(s sandbox.Provider) Option {
	return func(o *Orchestrator) {
		o.sandbox = s
	}
}

// WithWorkspaceSandboxEnsurer wires the run-time availability seam used by
// sandboxed run setup/finalization when workspace-sandbox is unavailable.
func WithWorkspaceSandboxEnsurer(e phases.WorkspaceSandboxEnsurer) Option {
	return func(o *Orchestrator) {
		o.workspaceSandbox = e
	}
}

// WithPolicy sets the policy evaluator.
func WithPolicy(p policy.Evaluator) Option {
	return func(o *Orchestrator) {
		o.policy = p
	}
}

// WithEvents sets the event store.
func WithEvents(e event.Store) Option {
	return func(o *Orchestrator) {
		o.events = e
	}
}

// WithQuotaObservationStore installs the pricing-owned durable sink for
// provider quota frames. Observation failures remain diagnostic and never
// change a run's owner outcome.
func WithQuotaObservationStore(store pricing.QuotaObservationRepository) Option {
	return func(o *Orchestrator) { o.quotaObservations = store }
}

func WithReceiptSummaryReader(reader ReceiptSummaryReader) Option {
	return func(o *Orchestrator) { o.receipts = reader }
}

func WithFindings(repo findings.Repository) Option {
	return func(o *Orchestrator) { o.findings = repo }
}

// WithInvestigationLifecycleRepository wires the caller-neutral typed
// investigation lifecycle. The orchestration package only reconciles its own
// terminal workflow outcome into that repository; admission remains owned by
// the API handler.
func WithInvestigationLifecycleRepository(repo investigation.Repository) Option {
	return func(o *Orchestrator) { o.typedInvestigations = repo }
}

// WithInvestigationLearningRecorder installs the single owner of outcome-
// linked memory capture. A nil recorder leaves a durable pending state rather
// than weakening investigation completion.
func WithInvestigationLearningRecorder(recorder investigationlearning.Recorder) Option {
	return func(o *Orchestrator) { o.learningRecorder = recorder }
}

func WithReceiptEvidenceStore(store runreport.ReceiptJoinStore) Option {
	return func(o *Orchestrator) { o.receiptEvidence = store }
}

func WithInvestigationLedgerStore(store runreport.LedgerStore) Option {
	return func(o *Orchestrator) { o.investigationLedger = store }
}

// WithInvocationReadModel wires the independent durable analytics projection.
// It is deliberately distinct from the legacy report cache.
func WithInvocationReadModel(store invocationreadmodel.Store) Option {
	return func(o *Orchestrator) { o.invocationReadModel = store }
}

// WithArtifacts sets the artifact collector.
func WithArtifacts(a artifact.Collector) Option {
	return func(o *Orchestrator) {
		o.artifacts = a
	}
}

// WithCheckpoints sets the checkpoint repository for resumption support.
func WithCheckpoints(c repository.CheckpointRepository) Option {
	return func(o *Orchestrator) {
		o.checkpoints = c
	}
}

// WithIdempotency sets the idempotency repository for replay safety.
func WithIdempotency(i repository.IdempotencyRepository) Option {
	return func(o *Orchestrator) {
		o.idempotency = i
	}
}

func WithWorkflowRepository(repo repository.WorkflowRepository) Option {
	return func(o *Orchestrator) { o.workflows = repo }
}

func WithWorkflowExecutionRepository(repo repository.WorkflowExecutionRepository) Option {
	return func(o *Orchestrator) { o.workflowExecutions = repo }
}

// WithBroadcaster sets the event broadcaster for real-time WebSocket updates.
func WithBroadcaster(b EventBroadcaster) Option {
	return func(o *Orchestrator) {
		o.broadcaster = b
	}
}

// WithTerminator sets the terminator for robust process termination.
func WithTerminator(t *Terminator) Option {
	return func(o *Orchestrator) {
		o.terminator = t
	}
}

// WithStorageLabel sets the storage label reported by health checks.
func WithStorageLabel(label string) Option {
	return func(o *Orchestrator) {
		o.storageLabel = strings.TrimSpace(label)
	}
}

// WithRolePolicyState wires portable role resolution. The resolver is an
// explicit seam so tests never need installed resource CLIs.
func WithRolePolicyState(state *rolepolicy.State, resolver rolepolicy.Resolver) Option {
	return func(o *Orchestrator) {
		o.rolePolicy = state
		o.roleResolver = resolver
	}
}

// WithStructuredExtractor wires the optional constrained extraction backend.
// The backend receives the portable role from ResultSpec and cannot bypass
// local schema validation.
func WithStructuredExtractor(extractor structuredresult.Extractor) Option {
	return func(o *Orchestrator) {
		o.structuredResults = structuredresult.Resolver{Extractor: extractor}
	}
}

// WithLabelGenerator wires the constrained ai-gateway seam used only when an
// imported transcript has neither a harness title nor a user prompt. It is
// deliberately separate from structured result projection so title generation
// cannot affect run outcomes.
func WithLabelGenerator(generator structuredresult.Extractor) Option {
	return func(o *Orchestrator) {
		o.labelGenerator = generator
	}
}

// WithHealthStore wires the persisted health audit store. The executor
// records every model-availability classification (ok or failed) here
// via the ModelHealthReporter seam, and GetModelHealthSnapshot reads
// the current snapshot from the same store. Pass nil to disable health
// observability (writes become no-ops).
func WithHealthStore(store *health.Store) Option {
	return func(o *Orchestrator) {
		o.healthStore = store
	}
}

// WithInvestigationSettings sets the investigation settings repository.
func WithInvestigationSettings(repo repository.InvestigationSettingsRepository) Option {
	return func(o *Orchestrator) {
		o.investigationSettings = repo
	}
}

// WithFlagValidator sets the flag validator for runner-specific flag validation.
func WithFlagValidator(v runner.FlagValidator) Option {
	return func(o *Orchestrator) {
		o.flagValidator = v
	}
}

// WithPromptClient sets the prompt-manager client for reading investigation prompts from skills.
func WithPromptClient(client promptmanager.Client) Option {
	return func(o *Orchestrator) {
		o.promptClient = client
	}
}

// WithAttachmentStorage sets the file storage service for resolving attachment IDs.
func WithAttachmentStorage(s storage.Service) Option {
	return func(o *Orchestrator) {
		o.storage = s
	}
}

// WithRunStateRoot supplies the explicit root for durable per-run artifacts.
func WithRunStateRoot(root string) Option {
	return func(o *Orchestrator) { o.runStateRoot = root }
}

// WithRunStateRootResolver selects a state root from the operation context.
// Production wiring uses this so HTTP test-mode leases survive dispatch.
func WithRunStateRootResolver(resolver runstate.RootResolver) Option {
	return func(o *Orchestrator) { o.runStateResolver = resolver }
}

func (o *Orchestrator) resolveRunStateRoot(ctx context.Context) (string, error) {
	if o.runStateResolver != nil {
		return o.runStateResolver.Resolve(ctx)
	}
	if o.runStateRoot == "" {
		return "", fmt.Errorf("run state root is required")
	}
	return o.runStateRoot, nil
}

func (o *Orchestrator) recordRunStateWrite(ctx context.Context) {
	if o.runStateResolver != nil {
		o.runStateResolver.RecordWrite(ctx)
	}
}

// WithOrchestrationSettings sets the orchestration settings store.
func WithOrchestrationSettings(store *agentconfig.OrchestrationSettingsStore) Option {
	return func(o *Orchestrator) {
		o.orchestrationSettings = store
	}
}

// WithIdentitySecret sets the HMAC secret for signing agent identity tokens.
func WithIdentitySecret(secret []byte) Option {
	return func(o *Orchestrator) {
		o.identitySecret = secret
	}
}

// WithSpawnDispatcher installs the runner-startup dispatcher. The
// orchestrator routes every spawn (CreateRun, ResumeRun) through it
// to enforce startup serialization and surface queue depth in
// CreateRunResponse. Supplying nil leaves the dispatcher unset, which
// is only valid in tests that mock CreateRun's spawn path.
func WithSpawnDispatcher(d *spawn.Dispatcher) Option {
	return func(o *Orchestrator) {
		o.dispatcher = d
	}
}

// WithInteractiveSessions wires the web-console session controller that drives
// interactive-mode runs. Without it, a run created with
// ExecutionMode=interactive fails cleanly at execute time.
func WithInteractiveSessions(sessions webconsole.SessionController) Option {
	return func(o *Orchestrator) {
		o.interactiveSessions = sessions
	}
}

// WithWebConsoleUIBase sets the resolved web-console UI base URL used to build
// the run-detail deep link for interactive runs. Empty disables the deep link
// (clients fall back to the session id).
func WithWebConsoleUIBase(base string) Option {
	return func(o *Orchestrator) {
		o.webConsoleUIBase = base
	}
}

// SetReconciler sets the reconciler reference for hot-reload propagation.
// This is called after construction because the reconciler depends on the orchestrator.
func (o *Orchestrator) SetReconciler(r *Reconciler) {
	o.reconciler = r
	if r != nil {
		r.finiteSerialRecovery = o
		r.structuredResults = o.structuredResults
		r.interactiveRecoveryMu = &o.wakeMu
		r.interactiveLiveDrivers = o.interactiveDrivers
	}
}

// SetAwaitRegistry wires the durable park/wait registry after construction. The
// registry takes the orchestrator as its waker (WakeRun/ListParkedRuns), so it
// must be built once the orchestrator exists — mirroring SetReconciler.
func (o *Orchestrator) SetAwaitRegistry(r *AwaitRegistry) {
	o.awaitRegistry = r
}

// SetWorkflowNudger wires the completion-nudge queue. The nudger's drive
// function is o.driveWorkflowExecution, so it must be built once the
// orchestrator exists — mirroring SetAwaitRegistry. Nil leaves completion-
// driven advance disabled (the reconciler recovery sweep still re-drives).
func (o *Orchestrator) SetWorkflowNudger(n *WorkflowNudger) {
	o.workflowNudger = n
}

// SpawnStats returns the current spawn-dispatcher state. Safe to call
// from the HTTP response path. Returns the zero Stats when the
// dispatcher is unset (test-only).
func (o *Orchestrator) SpawnStats() spawn.Stats {
	if o.dispatcher == nil {
		return spawn.Stats{}
	}
	return o.dispatcher.Stats()
}

// New creates a new Orchestrator with the given dependencies.
//
// If no spawn dispatcher is supplied via [WithSpawnDispatcher], a
// default one is constructed with single-slot serialization and queue
// capacity = MaxConcurrentRuns * 2. This keeps tests that don't care
// about the dispatcher working without per-test boilerplate, while
// production wiring still passes an explicitly-tuned dispatcher.
func New(
	profiles repository.ProfileRepository,
	tasks repository.TaskRepository,
	runs repository.RunRepository,
	opts ...Option,
) *Orchestrator {
	o := &Orchestrator{
		profiles:             profiles,
		tasks:                tasks,
		runs:                 runs,
		config:               DefaultConfig(),
		clock:                time.Now,
		interactiveDrivers:   newInteractiveDriverRegistry(),
		structuredResults:    structuredresult.Resolver{},
		workflowWaiters:      newWorkflowWaitRegistry(),
		runtimeOwnerIdentity: "agent-manager:" + uuid.NewString(),
	}

	for _, opt := range opts {
		opt(o)
	}
	if o.workflowExecutions != nil && o.workflows != nil {
		expressions, _ := workflowruntime.NewExpressionEvaluator()
		o.workflowEngine = &workflowruntime.Engine{Store: o.workflowExecutions, Catalog: o.workflows, Children: workflowChildLauncher{o: o}, Subworkflows: workflowSubworkflowLauncher{o: o}, Expressions: expressions, Now: o.now}
		if source, ok := o.promptClient.(promptmanager.AssignmentClient); ok && source != nil {
			o.workflowEngine.PromptResolver = workflowPromptResolver{source: source}
		}
	}

	if o.dispatcher == nil {
		queueCap := o.config.MaxConcurrentRuns * 2
		if queueCap < 1 {
			queueCap = 8
		}
		o.dispatcher = spawn.New(spawn.Config{
			MaxStartingConcurrency: 1,
			QueueCapacity:          queueCap,
		})
	}

	return o
}

// WithRuntimeOwnerIdentity installs the process-lifetime identity used for
// durable run ownership. Tests and production wiring can share one identity
// between orchestration and recovery components.
func WithRuntimeOwnerIdentity(identity string) Option {
	return func(o *Orchestrator) {
		if strings.TrimSpace(identity) != "" {
			o.runtimeOwnerIdentity = strings.TrimSpace(identity)
		}
	}
}

func (o *Orchestrator) now() time.Time {
	if o != nil && o.clock != nil {
		return o.clock()
	}
	return systemNow()
}

// -----------------------------------------------------------------------------
// AgentProfile Operations
// -----------------------------------------------------------------------------

// WithEffortAuthority installs an explicitly configured authority. No default
// provider, client enrollment, credential exchange or grant is inferred.
func WithEffortAuthority(a effortauthority.Engine) Option {
	return func(o *Orchestrator) { o.effortAuthority = a }
}

func (o *Orchestrator) finiteNativeEnabled() bool {
	return o.finiteNativeFactory != nil && o.finiteNativeFactory.Enabled()
}
