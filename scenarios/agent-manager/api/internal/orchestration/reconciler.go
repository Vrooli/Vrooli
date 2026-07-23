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
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/adapters/webconsole"
	cfgpkg "agent-manager/internal/config"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/orchestration/phases"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
)

// ReconcilerConfig holds configuration for the reconciliation service.
type ReconcilerConfig struct {
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
}

// DefaultReconcilerConfig returns sensible defaults.
// StaleThreshold is 5 minutes to match executor config and allow for slow operations.
// MaxRecoveryAge is 10 minutes — if the executor heartbeat has been absent that long
// while the process is still alive, the executor is gone (e.g., agent-manager restarted)
// and the process should be killed rather than perpetually recovered.
// OrphanGracePeriod is 10 minutes to avoid killing newly started processes.
func DefaultReconcilerConfig() ReconcilerConfig {
	return ReconcilerConfig{
		Interval:          30 * time.Second,
		StaleThreshold:    5 * time.Minute,  // More forgiving - allows for slow DB updates
		MaxRecoveryAge:    10 * time.Minute, // Kill process if stale beyond this
		OrphanGracePeriod: 10 * time.Minute, // Longer grace period for safety
		MaxStaleRuns:      10,
		PendingThreshold:  5 * time.Minute,
		KillOrphans:       true, // Always kill orphan processes
		AutoRecover:       true, // Auto-recover stale runs if process is alive
	}
}

// Reconciler manages orphan detection and stale run recovery.
type Reconciler struct {
	runs              repository.RunRepository
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

	config ReconcilerConfig
	clock  func() time.Time

	// levers exposes internal threshold knobs (e.g. recovery tail tick).
	// Defaulted to config.DefaultLevers(); callers can override via
	// WithReconcilerLevers when wiring the orchestrator.
	levers cfgpkg.Levers

	// State
	mu           sync.Mutex
	running      bool
	stopCh       chan struct{}
	doneCh       chan struct{}
	lastRunTime  time.Time
	lastRunStats ReconcileStats

	// Broadcaster for real-time updates
	broadcaster EventBroadcaster

	recoveryMu         sync.Mutex
	tailers            map[uuid.UUID]context.CancelFunc
	workflowRecovery   WorkflowExecutionRecoverer
	workflowLiveness   WorkflowWaitingLivenessRecoverer
	pendingRunRecovery PendingRunRecoverer
}

// ReconcileStats contains statistics from a reconciliation cycle.
type ReconcileStats struct {
	Timestamp            time.Time
	Duration             time.Duration
	RunsChecked          int
	StaleRuns            int
	OrphansFound         int
	RunsRecovered        int
	OrphansKilled        int
	ReviewChecked        int
	ReviewSynced         int
	WorkflowRecoveryRuns int
	Errors               []string
}

type WorkflowExecutionRecoverer interface{ RecoverWorkflowExecutions(context.Context) error }

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
		runs:    runs,
		events:  nil,
		runners: runners,
		config:  DefaultReconcilerConfig(),
		clock:   time.Now,
		levers:  cfgpkg.DefaultLevers(),
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
		tailers: make(map[uuid.UUID]context.CancelFunc),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// ReconcilerOption configures the reconciler.
type ReconcilerOption func(*Reconciler)

// WithReconcilerConfig sets custom configuration.
func WithReconcilerConfig(cfg ReconcilerConfig) ReconcilerOption {
	return func(r *Reconciler) {
		r.config = cfg
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
	stats := r.reconcile(ctx)
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
			stats := r.reconcile(ctx)
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
	if stats.StaleRuns > 0 || stats.OrphansFound > 0 {
		r.log().Info("cycle complete",
			"checked", stats.RunsChecked,
			"stale", stats.StaleRuns,
			"orphans", stats.OrphansFound,
			"recovered", stats.RunsRecovered,
			"killed", stats.OrphansKilled,
			"errors", len(stats.Errors),
		)
	}
}

// reconcile performs the actual reconciliation work.
func (r *Reconciler) reconcile(ctx context.Context) ReconcileStats {
	start := r.now()
	stats := ReconcileStats{Timestamp: start}
	if r.workflowRecovery != nil {
		if err := r.workflowRecovery.RecoverWorkflowExecutions(ctx); err != nil {
			stats.Errors = append(stats.Errors, "workflow recovery: "+err.Error())
		} else {
			stats.WorkflowRecoveryRuns++
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

	// Step 7: Garbage-collect old terminal run state directories.
	r.cleanupRunStateDirs(ctx)

	stats.Duration = time.Since(start)
	return stats
}

func (r *Reconciler) reapPendingRun(ctx context.Context, run *domain.Run) {
	age := time.Since(run.CreatedAt).Round(time.Second)
	reason := fmt.Sprintf("run remained pending for %v without dispatcher execution", age)
	r.log().Warn("reaping aged pending run", obs.KeyRunID, run.ID.String(), "pendingAge", age.String())
	r.markRunFailed(ctx, run, reason)
	if r.events == nil {
		return
	}
	message := fmt.Sprintf("reconciler failed stranded pending run after %v: %s", age, reason)
	if err := r.events.Append(ctx, run.ID, domain.NewLogEvent(run.ID, "error", message)); err != nil {
		r.log().Warn("pending reap event append failed", obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
	}
}

func (r *Reconciler) syncReviewRuns(ctx context.Context, stats *ReconcileStats) {
	if r.sandbox == nil {
		return
	}

	needsReview := domain.RunStatusNeedsReview
	reviewRuns, err := r.runs.List(ctx, repository.RunListFilter{
		Status: &needsReview,
	})
	if err != nil {
		stats.Errors = append(stats.Errors, "failed to list needs_review runs: "+err.Error())
		return
	}
	stats.ReviewChecked = len(reviewRuns)

	for _, run := range reviewRuns {
		// List deliberately omits sandbox_id and other heavy fields. Review
		// synchronization needs that durable association, so reload the full row
		// instead of broadening the read-side list contract for every caller.
		full, getErr := r.runs.Get(ctx, run.ID)
		if getErr != nil || full == nil {
			if getErr != nil {
				stats.Errors = append(stats.Errors, "failed to reload needs_review run "+run.ID.String()+": "+getErr.Error())
			}
			continue
		}
		run = full
		if run.SandboxID == nil {
			continue
		}
		sb, err := r.sandbox.Get(ctx, *run.SandboxID)
		if err != nil {
			continue
		}

		switch sb.Status {
		case sandbox.SandboxStatusApproved:
			if run.Status == domain.RunStatusComplete && run.ApprovalState == domain.ApprovalStateApproved {
				continue
			}
			r.markRunApprovedFromSandbox(ctx, run, "workspace-sandbox-sync")
			stats.ReviewSynced++
		case sandbox.SandboxStatusRejected:
			if run.Status == domain.RunStatusFailed && run.ApprovalState == domain.ApprovalStateRejected {
				continue
			}
			r.markRunRejectedFromSandbox(ctx, run, "workspace-sandbox-sync")
			stats.ReviewSynced++
		}
	}
}

func (r *Reconciler) markRunApprovedFromSandbox(ctx context.Context, run *domain.Run, actor string) {
	now := r.now()
	run.ApprovalState = domain.ApprovalStateApproved
	run.ApprovedBy = actor
	run.ApprovedAt = &now
	run.Status = domain.RunStatusComplete
	run.Phase = domain.RunPhaseCompleted
	run.EndedAt = &now
	run.UpdatedAt = now

	if err := r.runs.Update(ctx, run); err != nil {
		r.log().Warn("approved run sync failed", obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
		return
	}
	if r.broadcaster != nil {
		r.broadcaster.BroadcastRunStatus(run)
	}
}

func (r *Reconciler) markRunRejectedFromSandbox(ctx context.Context, run *domain.Run, actor string) {
	now := r.now()
	run.ApprovalState = domain.ApprovalStateRejected
	run.ApprovedBy = actor
	run.ApprovedAt = &now
	run.Status = domain.RunStatusFailed
	run.Phase = domain.RunPhaseCompleted
	run.EndedAt = &now
	run.UpdatedAt = now

	if err := r.runs.Update(ctx, run); err != nil {
		r.log().Warn("rejected run sync failed", obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
		return
	}
	if r.broadcaster != nil {
		r.broadcaster.BroadcastRunStatus(run)
	}
}

// handleStaleRun handles a run that appears to have stalled.
func (r *Reconciler) handleStaleRun(ctx context.Context, run *domain.Run, stats *ReconcileStats) {
	tag := run.GetTag()
	var heartbeatAge time.Duration
	if run.LastHeartbeat != nil {
		heartbeatAge = time.Since(*run.LastHeartbeat)
	} else {
		heartbeatAge = time.Since(run.CreatedAt)
	}

	r.log().Debug("checking stale run",
		obs.KeyRunID, run.ID.String(),
		"tag", tag,
		"status", string(run.Status),
		"heartbeatAge", heartbeatAge.Round(time.Second).String(),
		"staleThreshold", r.config.StaleThreshold.String(),
	)

	// reconcile() supplies pruned-column runs (no ResolvedConfig) — re-fetch
	// with Get so recoverRun has everything recoveryParser needs. See the
	// matching note in RecoverInFlightRuns for the production bug this guards.
	if full, err := r.runs.Get(ctx, run.ID); err == nil && full != nil {
		run = full
	}

	if result, err := r.recoverRun(ctx, run, true); err == nil && result != nil {
		if result.Recovered {
			stats.RunsRecovered++
		}
		if run.Status == domain.RunStatusComplete || run.Status == domain.RunStatusFailed || run.Status == domain.RunStatusCancelled {
			return
		}
	}

	// Interactive runs live in a web-console tmux pane, not a local tagged child.
	// recoverRun already verified the session (GetSession) and reattached the
	// tailer or finalized the run, so the pgid scan / MaxRecoveryAge kill below
	// must not run — it would falsely fail a healthy interactive run.
	if run.ExecutionMode.Normalized() == domain.ExecutionModeInteractive {
		return
	}

	// First, check if the process is actually still running
	processAlive := r.isProcessAlive(ctx, run)

	if !processAlive {
		// Process died but DB wasn't updated - mark as failed
		r.log().Warn("run process not found, marking failed",
			obs.KeyRunID, run.ID.String(),
			"tag", tag,
			"heartbeatAge", heartbeatAge.Round(time.Second).String(),
		)
		r.markRunFailed(ctx, run, fmt.Sprintf("process terminated unexpectedly (detected by reconciler after %v without heartbeat, tag=%s)",
			heartbeatAge.Round(time.Second), tag))
		return
	}

	// Process is alive but heartbeat is stale - could be legitimate slow work,
	// or the executor is gone (e.g., agent-manager restarted) and nobody is
	// managing this process anymore.
	r.log().Info("run stale but process alive",
		obs.KeyRunID, run.ID.String(),
		"lastHeartbeat", formatTimePtr(run.LastHeartbeat),
	)

	// If the heartbeat has been absent beyond MaxRecoveryAge, the executor
	// is gone. Kill the process and mark the run as failed rather than
	// perpetually recovering it.
	if r.config.MaxRecoveryAge > 0 && heartbeatAge > r.config.MaxRecoveryAge {
		r.log().Warn("run exceeded max recovery age, killing process",
			obs.KeyRunID, run.ID.String(),
			"tag", tag,
			"heartbeatAge", heartbeatAge.Round(time.Second).String(),
			"maxRecoveryAge", r.config.MaxRecoveryAge.String(),
		)
		r.killRunProcesses(ctx, run)
		r.markRunFailed(ctx, run, fmt.Sprintf(
			"executor heartbeat absent for %v (max recovery age %v exceeded) — process killed by reconciler (tag=%s)",
			heartbeatAge.Round(time.Second), r.config.MaxRecoveryAge, tag))
		return
	}

	if r.config.AutoRecover {
		// The process is alive but the executor heartbeat loop isn't updating.
		// Don't reset LastHeartbeat here — we need heartbeat age to keep growing
		// so MaxRecoveryAge can eventually trigger. Just count this as a recovery
		// (i.e., "we chose not to kill it yet").
		stats.RunsRecovered++
	}
}

// markRunFailed marks a run as failed due to unexpected termination.
func (r *Reconciler) markRunFailed(ctx context.Context, run *domain.Run, reason string) {
	now := r.now()
	run.Status = domain.RunStatusFailed
	run.ErrorMsg = reason
	run.EndedAt = &now
	run.UpdatedAt = now

	if err := r.runs.Update(ctx, run); err != nil {
		r.log().Warn("run status update failed", obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
	}

	// Broadcast status change
	if r.broadcaster != nil {
		r.broadcaster.BroadcastRunStatus(run)
	}
}

// isProcessAlive checks if the process for a run is still running.
func (r *Reconciler) isProcessAlive(ctx context.Context, run *domain.Run) bool {
	tag := run.GetTag()
	runnerType := "unknown"
	if run.ResolvedConfig != nil {
		runnerType = string(run.ResolvedConfig.RunnerType)
	}

	r.log().Debug("isProcessAlive check",
		obs.KeyRunID, run.ID.String(),
		"tag", tag,
		obs.KeyRunnerType, runnerType,
	)

	// Method 1: Check via runner if available
	if r.runners != nil && run.ResolvedConfig != nil {
		if runner, err := r.runners.Get(run.ResolvedConfig.RunnerType); err == nil {
			// Try to detect via runner's internal tracking
			// This requires the runner to implement a status check method
			// For now, fall through to process scanning
			_ = runner
		}
	}

	// Method 2: Scan /proc for the process
	alive := r.scanForProcess(tag)
	r.log().Debug("isProcessAlive result",
		obs.KeyRunID, run.ID.String(),
		"tag", tag,
		"alive", alive,
	)
	return alive
}

// scanForProcess checks if the runner process for a run is still alive.
//
// We intentionally avoid "pgrep -f <tag>" because it matches ANY process whose
// command line contains the tag string — including child processes (shells, tee,
// cleanup handlers) that inherited the tag via environment variables. These
// lingering children cause false positives that prevent the reconciler from
// detecting dead runs.
//
// Instead, we scan only for known runner executables (claude, codex, opencode)
// and verify they carry the tag via either:
//   - --tag <tag> in their command line arguments, OR
//   - *_AGENT_TAG=<tag> in their /proc/<pid>/environ
func (r *Reconciler) scanForProcess(tag string) bool {
	found := r.scanRunnerProcessesByTag(tag)
	if found {
		r.log().Debug("runner process found", "tag", tag)
	} else {
		r.log().Debug("no runner process found", "tag", tag)
	}
	return found
}

// scanRunnerProcessesByTag checks if any known runner process (claude, codex, opencode)
// is alive with the given tag. It checks both command-line --tag arguments and
// environment variables for precise matching.
func (r *Reconciler) scanRunnerProcessesByTag(tag string) bool {
	for _, runnerName := range []string{"claude", "codex", "opencode"} {
		if r.scanRunnerProcessByTag(runnerName, tag) {
			return true
		}
	}
	return false
}

// scanRunnerProcessByTag checks if a specific runner type has a process with the given tag.
func (r *Reconciler) scanRunnerProcessByTag(runnerName, tag string) bool {
	cmd := exec.Command("pgrep", "-af", runnerName)
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}

		pid, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		command := parts[1]

		// First check: does the command line have --tag with our specific tag?
		if cmdTag := extractTagFromCommand(command); cmdTag == tag {
			r.log().Debug("pid matched tag via --tag", "pid", pid, "tag", tag)
			return true
		}

		// Second check: does the process environment have the tag?
		if extractTagFromEnv(pid) == tag {
			r.log().Debug("pid matched tag via env", "pid", pid, "tag", tag)
			return true
		}
	}

	return false
}

// OrphanProcess represents a process that's running but not tracked in the database.
type OrphanProcess struct {
	PID       int
	Tag       string
	Command   string
	StartTime time.Time
}

// detectOrphanProcesses scans for agent processes not tracked in the database.
func (r *Reconciler) detectOrphanProcesses(ctx context.Context, knownTags map[string]*domain.Run) []OrphanProcess {
	var orphans []OrphanProcess

	// Scan for claude-code processes
	orphans = append(orphans, r.scanRunnerProcesses("claude", knownTags)...)

	// Scan for codex processes
	orphans = append(orphans, r.scanRunnerProcesses("codex", knownTags)...)

	// Scan for opencode processes
	orphans = append(orphans, r.scanRunnerProcesses("opencode", knownTags)...)

	return orphans
}

// scanRunnerProcesses scans for processes of a specific runner type.
func (r *Reconciler) scanRunnerProcesses(runnerName string, knownTags map[string]*domain.Run) []OrphanProcess {
	var orphans []OrphanProcess

	// Look for processes with agent-manager tags
	// Tags are typically UUIDs or "scenario-taskid" format
	cmd := exec.Command("pgrep", "-af", runnerName)
	output, err := cmd.Output()
	if err != nil {
		return orphans
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		// Parse PID and command
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}

		pid, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		command := parts[1]

		// Extract tag from command line (look for --tag argument)
		tag := extractTagFromCommand(command)
		if tag == "" {
			continue
		}

		// Check if this tag is known
		if _, known := knownTags[tag]; known {
			continue // Not an orphan
		}

		// Check if it looks like an agent-manager managed process
		// (UUIDs or known prefixes like "ecosystem-", "test-genie-")
		if !looksLikeAgentManagerTag(tag) {
			continue // Not our process
		}

		// Get process start time
		startTime := r.getProcessStartTime(pid)

		// Only consider it an orphan if it's been running longer than grace period
		if time.Since(startTime) < r.config.OrphanGracePeriod {
			continue // Too new, might be a race condition
		}

		orphans = append(orphans, OrphanProcess{
			PID:       pid,
			Tag:       tag,
			Command:   command,
			StartTime: startTime,
		})
	}

	return orphans
}

// extractTagFromCommand extracts the --tag value from a command line.
func extractTagFromCommand(command string) string {
	parts := strings.Fields(command)
	for i, part := range parts {
		if strings.HasPrefix(part, "CLAUDE_CODE_AGENT_TAG=") {
			return strings.TrimPrefix(part, "CLAUDE_CODE_AGENT_TAG=")
		}
		if strings.HasPrefix(part, "CODEX_AGENT_TAG=") {
			return strings.TrimPrefix(part, "CODEX_AGENT_TAG=")
		}
		if strings.HasPrefix(part, "OPENCODE_AGENT_TAG=") {
			return strings.TrimPrefix(part, "OPENCODE_AGENT_TAG=")
		}
		if strings.HasPrefix(part, "AGENT_TAG=") {
			return strings.TrimPrefix(part, "AGENT_TAG=")
		}
		if part == "--tag" && i+1 < len(parts) {
			return parts[i+1]
		}
		if strings.HasPrefix(part, "--tag=") {
			return strings.TrimPrefix(part, "--tag=")
		}
	}
	return ""
}

func extractTagFromEnv(pid int) string {
	envPath := fmt.Sprintf("/proc/%d/environ", pid)
	data, err := os.ReadFile(envPath)
	if err != nil || len(data) == 0 {
		return ""
	}

	for _, entry := range strings.Split(string(data), "\x00") {
		if strings.HasPrefix(entry, "CLAUDE_CODE_AGENT_TAG=") {
			return strings.TrimPrefix(entry, "CLAUDE_CODE_AGENT_TAG=")
		}
		if strings.HasPrefix(entry, "CODEX_AGENT_TAG=") {
			return strings.TrimPrefix(entry, "CODEX_AGENT_TAG=")
		}
		if strings.HasPrefix(entry, "OPENCODE_AGENT_TAG=") {
			return strings.TrimPrefix(entry, "OPENCODE_AGENT_TAG=")
		}
		if strings.HasPrefix(entry, "AGENT_TAG=") {
			return strings.TrimPrefix(entry, "AGENT_TAG=")
		}
	}

	return ""
}

// looksLikeAgentManagerTag checks if a tag looks like it was created by agent-manager.
func looksLikeAgentManagerTag(tag string) bool {
	// Check if it's a UUID
	if _, err := uuid.Parse(tag); err == nil {
		return true
	}

	// Check for known prefixes
	knownPrefixes := []string{
		"ecosystem-",
		"heartbeat-",
		"test-genie-",
		"agent-manager-",
		"run-",
	}
	for _, prefix := range knownPrefixes {
		if strings.HasPrefix(tag, prefix) {
			return true
		}
	}

	return false
}

// getProcessStartTime gets the start time of a process.
func (r *Reconciler) getProcessStartTime(pid int) time.Time {
	// Read process start time from /proc/[pid]/stat
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	data, err := os.ReadFile(statPath)
	if err != nil {
		return time.Time{}
	}

	// The start time is field 22 (0-indexed: 21)
	// It's in clock ticks since boot
	fields := strings.Fields(string(data))
	if len(fields) < 22 {
		return time.Time{}
	}

	startTicks, err := strconv.ParseInt(fields[21], 10, 64)
	if err != nil {
		return time.Time{}
	}

	// Get system boot time
	uptimeData, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return time.Time{}
	}
	uptimeStr := strings.Fields(string(uptimeData))[0]
	uptime, err := strconv.ParseFloat(uptimeStr, 64)
	if err != nil {
		return time.Time{}
	}

	// Get clock ticks per second (usually 100)
	clkTck := int64(100) // Default, could read from sysconf

	// Calculate process start time
	processUptimeSeconds := float64(startTicks) / float64(clkTck)
	bootTime := r.now().Add(-time.Duration(uptime * float64(time.Second)))
	startTime := bootTime.Add(time.Duration(processUptimeSeconds * float64(time.Second)))

	return startTime
}

// handleOrphan handles an orphan process.
func (r *Reconciler) handleOrphan(ctx context.Context, orphan OrphanProcess, stats *ReconcileStats) {
	r.log().Info("orphan process detected",
		"pid", orphan.PID,
		"tag", orphan.Tag,
		"runningSince", orphan.StartTime.Format(time.RFC3339),
	)

	if !r.config.KillOrphans {
		// Just log it, don't kill
		return
	}

	// Kill the orphan process
	if err := r.killProcess(orphan.PID); err != nil {
		stats.Errors = append(stats.Errors, fmt.Sprintf("failed to kill orphan %d: %v", orphan.PID, err))
	} else {
		stats.OrphansKilled++
		r.log().Info("orphan process killed", "pid", orphan.PID, "tag", orphan.Tag)

		// Clean up resource registries to remove stale entries
		r.cleanupResourceRegistries(ctx)
	}
}

// cleanupResourceRegistries runs cleanup on all agent resource registries.
// This removes stale entries from the file-based registries that track running agents.
func (r *Reconciler) cleanupResourceRegistries(ctx context.Context) {
	// List of resource CLI commands that maintain agent registries
	resourceCommands := []string{
		"resource-codex",
		"resource-opencode",
	}

	for _, cmd := range resourceCommands {
		// Run agents cleanup to remove stale entries
		cleanupCmd := exec.CommandContext(ctx, cmd, "agents", "cleanup")
		if err := cleanupCmd.Run(); err != nil {
			// Log but don't fail - cleanup is best-effort
			// The resource might not be installed or the command might not exist
			if !strings.Contains(err.Error(), "executable file not found") {
				r.log().Warn("resource agents cleanup failed", "command", cmd, obs.KeyError, err.Error())
			}
		}
	}
}

// killRunProcesses finds and kills all processes associated with a run's tag.
func (r *Reconciler) killRunProcesses(ctx context.Context, run *domain.Run) {
	tag := run.GetTag()

	// Find PIDs matching the tag via runner process scan
	for _, runnerName := range []string{"claude", "codex", "opencode"} {
		cmd := exec.Command("pgrep", "-af", runnerName)
		output, err := cmd.Output()
		if err != nil {
			continue
		}

		for _, line := range strings.Split(string(output), "\n") {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, " ", 2)
			if len(parts) < 2 {
				continue
			}
			pid, err := strconv.Atoi(parts[0])
			if err != nil {
				continue
			}

			command := parts[1]
			if extractTagFromCommand(command) == tag || extractTagFromEnv(pid) == tag {
				r.log().Info("killing run process", "pid", pid, "tag", tag)
				if err := r.killProcess(pid); err != nil {
					r.log().Warn("kill PID failed", "pid", pid, obs.KeyError, err.Error())
				}
			}
		}
	}

	r.cleanupResourceRegistries(ctx)
}

// killProcess kills a process with retry and escalation.
func (r *Reconciler) killProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	// Try SIGTERM first
	if err := process.Signal(os.Interrupt); err != nil {
		// Process might already be dead
		return nil
	}

	// Wait a short time for graceful shutdown
	time.Sleep(500 * time.Millisecond)

	// Check if still running
	if err := process.Signal(nil); err != nil {
		// Process is dead
		return nil
	}

	// Force kill with SIGKILL
	return process.Kill()
}
