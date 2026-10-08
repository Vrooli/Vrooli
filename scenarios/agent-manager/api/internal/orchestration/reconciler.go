// Package orchestration provides the core orchestration service for agent-manager.
//
// This file contains the Reconciler which handles orphan detection and stale run
// recovery. It runs as a background service that periodically:
// - Detects runs that appear stuck (no heartbeat for too long)
// - Cleans up orphaned processes that are running without corresponding database records
// - Recovers from agent-manager crashes by reconciling actual state with DB state
//
// RECONCILIATION LOOP:
//   1. List all "running" runs from database
//   2. Check each run's heartbeat - mark as stale if too old
//   3. Scan for orphan processes that aren't tracked in DB
//   4. Handle stale runs (mark failed or attempt recovery)
//   5. Handle orphans (kill or adopt)

package orchestration

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/adapters/artifact"
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/adapters/webconsole"
	cfgpkg "agent-manager/internal/config"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/orchestration/phases"
	"agent-manager/internal/repository"
	"agent-manager/internal/runstate"

	"github.com/google/uuid"
)

// ReconcilerConfig holds configuration for the reconciliation service.
type ReconcilerConfig struct {
	// OwnerIdentity is the process identity used when claiming recovered runs.
	// It must be unique per Agent Manager owner lifetime.
	OwnerIdentity string

	// Interval is how often to run reconciliation
	Interval time.Duration

	// StaleThreshold is how long without heartbeat before marking a run as stale
	StaleThreshold time.Duration

	// MaxRecoveryAge is the maximum time a run can be stale before the reconciler
	// stops auto-recovering it and kills the process instead. This prevents
	// orphaned processes (e.g., after agent-manager restart) from being kept
	// alive indefinitely by auto-recovery.
	MaxRecoveryAge time.Duration

	// OrphanGracePeriod is how long to wait before killing orphan processes
	OrphanGracePeriod time.Duration

	// MaxStaleRuns is the maximum number of stale runs to process per cycle
	MaxStaleRuns int

	// PendingThreshold is the maximum time a run may remain queued without a
	// dispatcher entry before it is failed. A pending run has neither a process
	// nor a heartbeat, so CreatedAt is its only durable liveness signal.
	PendingThreshold time.Duration

	// KillOrphans determines whether to automatically kill orphan processes
	KillOrphans bool

	// AutoRecover determines whether to automatically recover stale runs
	AutoRecover bool

	// InteractiveSessionRetention is how long an ended interactive run keeps its
	// web-console session (and idle agent CLI) for `run continue` before the
	// reconciler archives it. It also ages out agent-manager sessions no run
	// references. Zero disables the release sweep.
	InteractiveSessionRetention time.Duration
}

const (
	// orphanSandboxOperationTimeout prevents one wedged provider request from
	// stalling the process-wide reconciliation loop. A diff is only evidence
	// for deletion when the provider answers before this deadline.
	orphanSandboxOperationTimeout = 15 * time.Second
	// orphanSandboxSweepLimit keeps reconciliation bounded even if a provider
	// returns an unexpectedly large inventory.
	orphanSandboxSweepLimit = 10
	// interactiveSessionReleaseLimit bounds web-console archives per cycle so a
	// large retained backlog drains over several cycles.
	interactiveSessionReleaseLimit = 20
)

// DefaultReconcilerConfig returns sensible defaults.
// StaleThreshold is 5 minutes to match executor config and allow for slow operations.
// MaxRecoveryAge is 10 minutes — it is a diagnostic threshold for a live
// executor whose owner has not reported. It never authorizes killing a healthy
// detached process; explicit cancellation owns termination.
// OrphanGracePeriod is 10 minutes to avoid killing newly started processes.
func DefaultReconcilerConfig() ReconcilerConfig {
	return ReconcilerConfig{
		OwnerIdentity:     "agent-manager:" + uuid.NewString(),
		Interval:          30 * time.Second,
		StaleThreshold:    5 * time.Minute,  // More forgiving - allows for slow DB updates
		MaxRecoveryAge:    10 * time.Minute, // Diagnostic threshold; never a kill authorization
		OrphanGracePeriod: 10 * time.Minute, // Longer grace period for safety
		MaxStaleRuns:      10,
		PendingThreshold:  5 * time.Minute,
		KillOrphans:       true, // Always kill orphan processes
		AutoRecover:       true, // Auto-recover stale runs if process is alive
		// Window for `run continue` into an ended interactive run's live session.
		InteractiveSessionRetention: 2 * time.Hour,
	}
}

// Reconciler manages orphan detection and stale run recovery.
type Reconciler struct {
	runs              repository.RunRepository
	ownerIdentity     string
	events            event.Store
	runners           runner.Registry
	sandbox           sandbox.Provider
	structuredResults phases.StructuredResultResolver

	// sessions is the web-console session controller used to verify interactive
	// runs' liveness (GetSession) during recovery — interactive CLIs live in
	// web-console tmux, not a local tagged child, so the pgid scan does not apply
	// to them. Nil when interactive recovery is not wired (recovery then no-ops
	// for interactive runs rather than falsely completing or failing them).
	sessions webconsole.SessionController

	// interactiveDebounce overrides the interactive coordinator's turn-boundary
	// idle window during reattach (0 uses the coordinator default). Kept as a
	// field so tests can shrink it without a live clock.
	interactiveDebounce time.Duration

	// interactiveSessionPoll overrides the reattached tailer's mid-tail
	// session-liveness cadence (0 uses the coordinator default). Field so tests
	// can detect a vanished session quickly.
	interactiveSessionPoll time.Duration

	// interactiveSessionReattachWindow overrides how long a reattached tailer
	// tolerates a missing session before failing the run (0 uses the coordinator
	// default 3 minutes). Field so tests can fail fast.
	interactiveSessionReattachWindow time.Duration

	config ReconcilerConfig
	clock  func() time.Time

	// levers exposes internal threshold knobs (e.g. recovery tail tick).
	// Defaulted to config.DefaultLevers(); callers can override via
	// WithReconcilerLevers when wiring the orchestrator.
	levers            cfgpkg.Levers
	runStateRoot      string
	runStateResolver  runstate.RootResolver
	eventRetention    event.RetentionStore
	artifactRetention artifact.RetentionCollector

	// State
	mu           sync.Mutex
	running      bool
	stopCh       chan struct{}
	doneCh       chan struct{}
	lastRunTime  time.Time
	lastRunStats ReconcileStats

	// Broadcaster for real-time updates
	broadcaster EventBroadcaster

	// Shares the continuation owner lock after SetReconciler. Recovery must not
	// read a half-admitted interactive invocation.
	interactiveRecoveryMu  *sync.Mutex
	interactiveLiveDrivers *interactiveDriverRegistry
	interactiveTailDone    map[uuid.UUID]chan struct{}
	recoveryMu             sync.Mutex
	tailers                map[uuid.UUID]context.CancelFunc
	tailerOwners           map[uuid.UUID]context.Context
	workflowRecovery       WorkflowExecutionRecoverer
	workflowLiveness       WorkflowWaitingLivenessRecoverer
	pendingRunRecovery     PendingRunRecoverer
	terminalAccounting     TerminalAccountingRecoverer
	finiteSerialRecovery   interface{ ReconcileFiniteSerialEpisodes(context.Context) error }
	storageMaintainer      StorageMaintainer
}

// StorageMaintainer keeps the database file inside its declared budget. Each
// call does bounded work, so it can run on every reconcile cycle.
type StorageMaintainer interface {
	Maintain(context.Context) error
}

// SetStorageMaintainer installs the storage owner after construction; the
// maintainer depends on the maintenance fence, which is built after the
// reconciler.
func (r *Reconciler) SetStorageMaintainer(m StorageMaintainer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.storageMaintainer = m
}

func (r *Reconciler) currentStorageMaintainer() StorageMaintainer {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.storageMaintainer
}

// ReconcileStats contains statistics from a reconciliation cycle.
type ReconcileStats struct {
	Timestamp               time.Time
	Duration                time.Duration
	RunsChecked             int
	StaleRuns               int
	OrphansFound            int
	RunsRecovered           int
	OrphansKilled           int
	ReviewChecked           int
	ReviewSynced            int
	SandboxOrphansChecked   int
	SandboxOrphansReclaimed int
	SandboxOrphansPreserved int
	// InteractiveSessionsReleased counts web-console sessions archived after
	// their interactive run's retention window (or as unreferenced orphans).
	InteractiveSessionsReleased int
	WorkflowRecoveryRuns        int
	EventsPruned                int
	ArtifactsPruned             int
	Errors                      []string
}

type WorkflowExecutionRecoverer interface{ RecoverWorkflowExecutions(context.Context) error }

// TerminalAccountingRecoverer settles terminal usage for ended standalone runs
// from the harness's retained transcript. The orchestrator owns it.
type TerminalAccountingRecoverer interface {
	RecoverStandaloneTerminalAccounting(context.Context) error
}

type WorkflowWaitingLivenessRecoverer interface {
	ReconcileUnarmedWorkflowWaits(context.Context, time.Duration, time.Duration) error
}

// PendingRunRecoverer re-enqueues a persisted pending run after a process
// restart. The orchestrator owns this operation because it alone has the task,
// profile, checkpoint, and spawn-dispatcher dependencies needed to resume it.
type PendingRunRecoverer interface {
	ResumeRun(context.Context, uuid.UUID) (*domain.Run, error)
}

// NewReconciler creates a new reconciler with the given dependencies.
func NewReconciler(
	runs repository.RunRepository,
	runners runner.Registry,
	opts ...ReconcilerOption,
) *Reconciler {
	r := &Reconciler{
		runs:                  runs,
		ownerIdentity:         "agent-manager:" + uuid.NewString(),
		events:                nil,
		runners:               runners,
		config:                DefaultReconcilerConfig(),
		clock:                 time.Now,
		levers:                cfgpkg.DefaultLevers(),
		stopCh:                make(chan struct{}),
		doneCh:                make(chan struct{}),
		tailers:               make(map[uuid.UUID]context.CancelFunc),
		interactiveRecoveryMu: &sync.Mutex{},
		interactiveTailDone:   make(map[uuid.UUID]chan struct{}),
		tailerOwners:          make(map[uuid.UUID]context.Context),
	}

	for _, opt := range opts {
		opt(r)
	}
	if strings.TrimSpace(r.ownerIdentity) == "" {
		r.ownerIdentity = "agent-manager:" + uuid.NewString()
	}

	return r
}

// ReconcilerOption configures the reconciler.
type ReconcilerOption func(*Reconciler)

// WithReconcilerConfig sets custom configuration.
func WithReconcilerConfig(cfg ReconcilerConfig) ReconcilerOption {
	return func(r *Reconciler) {
		r.config = cfg
		if strings.TrimSpace(cfg.OwnerIdentity) != "" {
			r.ownerIdentity = strings.TrimSpace(cfg.OwnerIdentity)
		}
	}
}

// WithReconcilerClock injects the wall-clock used for reconciliation state
// timestamps and process-age calculations. Nil retains the production clock.
func WithReconcilerClock(clock func() time.Time) ReconcilerOption {
	return func(r *Reconciler) {
		if clock != nil {
			r.clock = clock
		}
	}
}

func (r *Reconciler) now() time.Time {
	if r != nil && r.clock != nil {
		return r.clock()
	}
	return systemNow()
}

// WithReconcilerBroadcaster sets the event broadcaster.
func WithReconcilerBroadcaster(b EventBroadcaster) ReconcilerOption {
	return func(r *Reconciler) {
		r.broadcaster = b
	}
}

func WithReconcilerEvents(store event.Store) ReconcilerOption {
	return func(r *Reconciler) {
		r.events = store
	}
}

// WithReconcilerSandbox sets the sandbox provider for approval sync.
func WithReconcilerSandbox(s sandbox.Provider) ReconcilerOption {
	return func(r *Reconciler) {
		r.sandbox = s
	}
}

// WithReconcilerLevers overrides the lever set used for internal cadence
// (recovery tail tick, etc.). Defaults to cfgpkg.DefaultLevers().
func WithReconcilerLevers(l cfgpkg.Levers) ReconcilerOption {
	return func(r *Reconciler) {
		r.levers = l
	}
}

func WithReconcilerRunStateRoot(root string) ReconcilerOption {
	return func(r *Reconciler) { r.runStateRoot = root }
}

func WithReconcilerRunStateRootResolver(resolver runstate.RootResolver) ReconcilerOption {
	return func(r *Reconciler) { r.runStateResolver = resolver }
}

// WithReconcilerEventRetention wires bounded event-history reclamation.
func WithReconcilerEventRetention(store event.RetentionStore) ReconcilerOption {
	return func(r *Reconciler) { r.eventRetention = store }
}

func WithReconcilerArtifactRetention(collector artifact.RetentionCollector) ReconcilerOption {
	return func(r *Reconciler) { r.artifactRetention = collector }
}

func (r *Reconciler) resolveRunStateRoot(ctx context.Context) (string, error) {
	if r.runStateResolver != nil {
		return r.runStateResolver.Resolve(ctx)
	}
	if r.runStateRoot == "" {
		return "", fmt.Errorf("run state root is required")
	}
	return r.runStateRoot, nil
}

// WithReconcilerInteractive wires the web-console session controller the
// reconciler uses to recover interactive runs (ExecutionMode=interactive): it
// verifies the session with GetSession and reattaches the transcript tailer.
// Without it, interactive runs are left untouched by recovery.
func WithReconcilerInteractive(sessions webconsole.SessionController) ReconcilerOption {
	return func(r *Reconciler) {
		r.sessions = sessions
	}
}

func WithReconcilerWorkflowRecovery(recoverer WorkflowExecutionRecoverer) ReconcilerOption {
	return func(r *Reconciler) { r.workflowRecovery = recoverer }
}

func WithReconcilerTerminalAccounting(recoverer TerminalAccountingRecoverer) ReconcilerOption {
	return func(r *Reconciler) { r.terminalAccounting = recoverer }
}

func WithReconcilerWorkflowWaitingLiveness(recoverer WorkflowWaitingLivenessRecoverer) ReconcilerOption {
	return func(r *Reconciler) { r.workflowLiveness = recoverer }
}

// WithReconcilerPendingRunRecovery wires the orchestration-owned resumption
// path used for pending rows discovered during startup recovery.
func WithReconcilerPendingRunRecovery(recoverer PendingRunRecoverer) ReconcilerOption {
	return func(r *Reconciler) { r.pendingRunRecovery = recoverer }
}

// Start begins the reconciliation loop.
func (r *Reconciler) Start(ctx context.Context) error {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return domain.NewStateError("Reconciler", "running", "start", "already running")
	}
	r.running = true
	r.stopCh = make(chan struct{})
	r.doneCh = make(chan struct{})
	r.mu.Unlock()

	go r.loop(ctx)
	r.log().Info("reconciler started",
		"interval", r.config.Interval.String(),
		"staleThreshold", r.config.StaleThreshold.String(),
	)
	return nil
}

// log returns the reconciler's component-tagged structured logger.
// Centralised so every call site uses the same component name.
func (r *Reconciler) log() *slog.Logger { return obs.Component("reconciler") }

// formatTimePtr renders an optional timestamp as RFC3339 or "<nil>" so
// it serialises cleanly into structured log fields without printing the
// type name.
func formatTimePtr(t *time.Time) string {
	if t == nil {
		return "<nil>"
	}
	return t.Format(time.RFC3339)
}

// Stop gracefully stops the reconciliation loop.
func (r *Reconciler) Stop() error {
	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		return nil
	}
	r.mu.Unlock()

	close(r.stopCh)
	<-r.doneCh

	r.mu.Lock()
	r.running = false
	r.mu.Unlock()

	r.log().Info("reconciler stopped")
	return nil
}

// IsRunning returns whether the reconciler is active.
func (r *Reconciler) IsRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running
}

// LastStats returns the statistics from the last reconciliation cycle.
func (r *Reconciler) LastStats() ReconcileStats {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lastRunStats
}

// RunOnce performs a single reconciliation cycle.
// This is useful for testing or manual triggering.
func (r *Reconciler) RunOnce(ctx context.Context) ReconcileStats {
	return r.reconcile(ctx)
}

// Config returns the reconciler's current configuration.
func (r *Reconciler) Config() ReconcilerConfig {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.config
}

// ReconcilerConfigWithSettings returns base with only the fields operator
// orchestration settings own replaced. Every settings-driven rebuild goes
// through here so fields settings do not own (PendingThreshold,
// InteractiveSessionRetention, OwnerIdentity, ...) keep their base value
// instead of silently dropping to zero.
func ReconcilerConfigWithSettings(base ReconcilerConfig, s cfgpkg.OrchestrationSettings) ReconcilerConfig {
	base.Interval = time.Duration(s.HealthDetection.ReconcilerIntervalSeconds) * time.Second
	base.StaleThreshold = time.Duration(s.HealthDetection.StaleThresholdSeconds) * time.Second
	base.MaxRecoveryAge = time.Duration(s.HealthDetection.MaxRecoveryAgeSeconds) * time.Second
	base.OrphanGracePeriod = time.Duration(s.ProcessTermination.OrphanGracePeriodSeconds) * time.Second
	base.KillOrphans = s.ProcessTermination.KillOrphans
	return base
}

// UpdateConfig applies new configuration to the reconciler at runtime.
// The new interval takes effect after the current cycle completes.
func (r *Reconciler) UpdateConfig(cfg ReconcilerConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.config = cfg
}

// loop runs the reconciliation loop using a timer for hot-reloadable intervals.
func (r *Reconciler) loop(ctx context.Context) {
	defer close(r.doneCh)

	// Run once immediately on startup.
	stats := r.reconcileGuarded(ctx)
	r.updateStats(stats)

	r.mu.Lock()
	interval := r.config.Interval
	r.mu.Unlock()
	timer := time.NewTimer(interval)
	defer timer.Stop()

	for {
		select {
		case <-r.stopCh:
			return
		case <-ctx.Done():
			return
		case <-timer.C:
			stats := r.reconcileGuarded(ctx)
			r.updateStats(stats)
			// Re-read interval (may have changed via UpdateConfig).
			r.mu.Lock()
			interval = r.config.Interval
			r.mu.Unlock()
			timer.Reset(interval)
		}
	}
}

// updateStats updates the last run statistics.
func (r *Reconciler) updateStats(stats ReconcileStats) {
	r.mu.Lock()
	r.lastRunTime = stats.Timestamp
	r.lastRunStats = stats
	r.mu.Unlock()

	// Log summary
	if stats.StaleRuns > 0 || stats.OrphansFound > 0 || stats.SandboxOrphansChecked > 0 || stats.InteractiveSessionsReleased > 0 {
		r.log().Info("cycle complete",
			"checked", stats.RunsChecked,
			"stale", stats.StaleRuns,
			"orphans", stats.OrphansFound,
			"recovered", stats.RunsRecovered,
			"killed", stats.OrphansKilled,
			"sandboxOrphansChecked", stats.SandboxOrphansChecked,
			"sandboxOrphansReclaimed", stats.SandboxOrphansReclaimed,
			"sandboxOrphansPreserved", stats.SandboxOrphansPreserved,
			"interactiveSessionsReleased", stats.InteractiveSessionsReleased,
			"errors", len(stats.Errors),
		)
	}
}

// reconcile performs the actual reconciliation work.
// reconcileGuarded contains a panic from any sweep inside one cycle so the
// reconciliation loop — the process-wide recovery safety net — keeps ticking
// on the next interval instead of taking down the API.
func (r *Reconciler) reconcileGuarded(ctx context.Context) (stats ReconcileStats) {
	defer obs.RecoverToFailure("reconciler cycle", nil)
	return r.reconcile(ctx)
}

func (r *Reconciler) reconcile(ctx context.Context) ReconcileStats {
	start := r.now()
	stats := ReconcileStats{Timestamp: start}
	if r.finiteSerialRecovery != nil {
		if err := r.finiteSerialRecovery.ReconcileFiniteSerialEpisodes(ctx); err != nil {
			stats.Errors = append(stats.Errors, "finite serial recovery: "+err.Error())
		}
	}
	if r.workflowRecovery != nil {
		if err := r.workflowRecovery.RecoverWorkflowExecutions(ctx); err != nil {
			stats.Errors = append(stats.Errors, "workflow recovery: "+err.Error())
		} else {
			stats.WorkflowRecoveryRuns++
		}
	}
	if r.terminalAccounting != nil {
		if err := r.terminalAccounting.RecoverStandaloneTerminalAccounting(ctx); err != nil {
			stats.Errors = append(stats.Errors, "terminal accounting recovery: "+err.Error())
		}
	}
	if r.workflowLiveness != nil {
		if err := r.workflowLiveness.ReconcileUnarmedWorkflowWaits(ctx, r.levers.Workflow.UnarmedWaitWarningThreshold, r.levers.Workflow.UnarmedWaitFailureThreshold); err != nil {
			stats.Errors = append(stats.Errors, "workflow waiting liveness: "+err.Error())
		}
	}

	// Step 1: List every run whose LivenessPolicy marks it for scanning. The
	// per-status policy table (domain.LivenessPolicy) is the single source of
	// truth for which statuses the reconciler inspects — replacing the old
	// hard-coded running|starting list. New statuses (e.g. parked) opt in by
	// declaring Scanned in the table rather than by ad-hoc exemption here.
	var dbRuns []*domain.Run
	for _, status := range domain.LivenessScannedStatuses() {
		statusFilter := status
		runs, err := r.runs.List(ctx, repository.RunListFilter{
			Status: &statusFilter,
		})
		if err != nil {
			// An infra error listing one status should not abort the whole
			// cycle (orphan/review sweeps below are still useful); record it
			// and continue.
			stats.Errors = append(stats.Errors, "failed to list "+string(status)+" runs: "+err.Error())
			continue
		}
		dbRuns = append(dbRuns, runs...)
	}
	stats.RunsChecked = len(dbRuns)

	// Build a map of known run tags for orphan detection. Only statuses whose
	// policy expects a live process protect a matching process from being
	// reaped as an orphan.
	knownTags := make(map[string]*domain.Run)
	for _, run := range dbRuns {
		if run.Status.LivenessPolicy().ExpectsProcess {
			knownTags[run.GetTag()] = run
		}
	}

	// Step 2: Pending runs intentionally have no heartbeat or process. Reap a
	// queue entry that has exceeded its bounded lifetime so a lost dispatcher
	// handoff cannot leave durable state invisible forever.
	for _, run := range dbRuns {
		if run.Status != domain.RunStatusPending || r.config.PendingThreshold <= 0 || time.Since(run.CreatedAt) <= r.config.PendingThreshold {
			continue
		}
		stats.StaleRuns++
		r.reapPendingRun(ctx, run)
	}

	// Step 3: Check each active run for staleness, dispatching on its liveness policy.
	// Only statuses that expect a heartbeat are stale-checked; only those with
	// a non-none stale action get recover-or-kill handling.
	for _, run := range dbRuns {
		policy := run.Status.LivenessPolicy()
		if run.ExecutionMode.Normalized() == domain.ExecutionModeAttached {
			// The launcher attaches before exec'ing the harness. Give that
			// handoff a bounded grace period, then require the token-bearing
			// process (or an explicitly supplied PID) to still exist.
			if r.now().Sub(run.CreatedAt) < attachedRunLivenessGracePeriod {
				continue
			}
			full, err := r.runs.Get(ctx, run.ID)
			if err != nil || full == nil {
				if err != nil {
					stats.Errors = append(stats.Errors, "failed to reload attached run "+run.ID.String()+": "+err.Error())
				}
				continue
			}
			if !r.attachedRunProcessAlive(full) {
				stats.StaleRuns++
				r.markRunFailed(ctx, full, "attached harness process is no longer present")
				r.appendAttachedLifecycleEvent(ctx, full.ID, "liveness_failed", "process no longer present")
			}
			continue
		}
		if !policy.ExpectsHeartbeat || policy.StaleAction == domain.StaleRunActionNone {
			continue
		}
		if run.IsStale(r.config.StaleThreshold) {
			stats.StaleRuns++
			r.handleStaleRun(ctx, run, &stats)
		}
	}

	// Step 4: Scan for orphan processes
	orphans := r.detectOrphanProcesses(ctx, knownTags)
	stats.OrphansFound = len(orphans)

	// Step 5: Handle orphans
	for _, orphan := range orphans {
		r.handleOrphan(ctx, orphan, &stats)
	}

	// Step 6: Sync needs_review runs with sandbox status
	r.syncReviewRuns(ctx, &stats)

	// Step 6b: Reconcile run-owned sandboxes whose durable run owner has
	// disappeared. This is intentionally owner-backed and conservative: only
	// old active sandboxes with an explicit run metadata binding are examined,
	// and deletion is allowed only after the provider proves an empty diff.
	r.reconcileOrphanSandboxes(ctx, &stats)

	// Step 6c: Release web-console sessions retained for continuation once
	// their interactive run has been ended longer than the retention window.
	r.releaseRetainedInteractiveSessions(ctx, &stats)

	// Step 7: Garbage-collect old terminal run state directories.
	r.cleanupRunStateDirs(ctx)

	// Step 8: Bound event history without holding a long SQLite write lock.
	if deleted, err := r.cleanupExpiredEvents(ctx); err != nil {
		stats.Errors = append(stats.Errors, "event retention: "+err.Error())
	} else {
		stats.EventsPruned = deleted
	}
	if deleted, err := r.cleanupExpiredArtifacts(ctx); err != nil {
		stats.Errors = append(stats.Errors, "artifact retention: "+err.Error())
	} else {
		stats.ArtifactsPruned = deleted
	}
	r.recordImportedEventCompaction(ctx, &stats)

	// Step 9: Return pages freed by the retention steps above to the
	// filesystem, within the declared storage budget.
	if maintainer := r.currentStorageMaintainer(); maintainer != nil {
		if err := maintainer.Maintain(ctx); err != nil {
			stats.Errors = append(stats.Errors, "storage maintenance: "+err.Error())
		}
	}

	stats.Duration = time.Since(start)
	return stats
}
