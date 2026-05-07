// Package sandbox.
//
// reconciler.go — the single dispatcher for every periodic reconciler.
//
// Each reconciler implements the same two-method interface (Name + Run);
// the Runner holds an ordered slice and ticks them all every interval.
// Per-reconciler metrics + an admin endpoint trigger (RunOne) fall out
// of this for free, replacing the previous hardcoded function body that
// inlined every reconciler call.
//
// Order is data, not code: pass the slice in the order you want it run.
// The wiring in main.go is the single place that decides the order.
package sandbox

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"workspace-sandbox/internal/clock"
	"workspace-sandbox/internal/types"
)

// Reconciler is the contract every periodic reconciler implements.
//
// Implementations should be idempotent and best-effort: a Run that
// errors mid-pass should still log + report, never panic. The Runner
// catches panics defensively but the post-hoc state is undefined.
type Reconciler interface {
	// Name is the stable identifier used by metrics and the admin
	// endpoint (POST /admin/reconcilers/{name}). Lowercase, no spaces.
	Name() string

	// Run executes one pass and returns a structured report. The Runner
	// timestamps the call and surfaces report fields as metrics.
	Run(ctx context.Context) ReconcileReport
}

// ReconcileReport is a generic per-pass summary. Each reconciler
// populates ItemsProcessed/ItemsFailed/LastError; structured details
// go in Details (per-reconciler shape).
type ReconcileReport struct {
	// ItemsProcessed is the total number of work units the reconciler
	// inspected (e.g., orphan dirs walked, daemons scanned).
	ItemsProcessed int

	// ItemsFailed is how many work units the reconciler tried and
	// failed to act on. A reconciler that does nothing reports 0.
	ItemsFailed int

	// Duration is how long the pass took.
	Duration time.Duration

	// LastError is the most recent error encountered. Empty on success.
	LastError string

	// Details carries reconciler-specific structured data (e.g., orphan
	// IDs, reaped PIDs) for the admin endpoint and debug logging.
	Details map[string]any
}

// Runner drives a slice of Reconcilers on a fixed interval. Replaces
// the old hardcoded LifecycleReconciler.runReconcilers function body.
type Runner struct {
	interval    time.Duration
	periodic    []Reconciler
	startupOnly []Reconciler
	stopCh      chan struct{}
	doneCh      chan struct{}
	clock       clock.Clock

	mu      sync.Mutex
	metrics map[string]ReconcilerMetrics
}

// ReconcilerMetrics is the per-reconciler observability surface
// exposed by Runner.Metrics. Useful both for the /metrics surface and
// for the admin endpoint's response shape.
type ReconcilerMetrics struct {
	LastRunAt      time.Time
	LastDuration   time.Duration
	ItemsProcessed int
	ItemsFailed    int
	LastError      string
	RunCount       int
}

// NewRunner constructs a Runner with the given interval.
//
// periodic reconcilers run on every tick (and once at startup).
// startupOnly reconcilers run exactly once before the first periodic
// pass — used for one-shot expirations like ManualReviewExpiry where
// re-running on every tick would be wasteful.
//
// clk is required: the dispatch ticker and per-reconciler LastRunAt
// metric both flow through it so tests can drive deterministic ticks
// via FakeClock.Advance.
func NewRunner(interval time.Duration, periodic, startupOnly []Reconciler, clk clock.Clock) *Runner {
	if clk == nil {
		panic("sandbox.NewRunner: clock is required")
	}
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	return &Runner{
		interval:    interval,
		periodic:    periodic,
		startupOnly: startupOnly,
		stopCh:      make(chan struct{}),
		doneCh:      make(chan struct{}),
		clock:       clk,
		metrics:     make(map[string]ReconcilerMetrics),
	}
}

// Start launches the reconciliation goroutine. The first pass runs
// synchronously (well, scheduled via go but the for-select enters
// before the ticker fires) so any pre-existing drift — including
// filesystem orphans accumulated while the API was down — is cleaned
// up early in boot.
func (r *Runner) Start() {
	if r == nil {
		return
	}
	go func() {
		ticker := r.clock.NewTicker(r.interval)
		defer ticker.Stop()
		defer close(r.doneCh)

		ctx := context.Background()
		r.runOnce(ctx, true)
		for {
			select {
			case <-ticker.C():
				r.runOnce(context.Background(), false)
			case <-r.stopCh:
				return
			}
		}
	}()
}

// Stop signals the goroutine to exit and blocks until it does.
// Idempotent: calling Stop twice is safe.
func (r *Runner) Stop() {
	if r == nil {
		return
	}
	select {
	case <-r.stopCh:
		return
	default:
		close(r.stopCh)
	}
	<-r.doneCh
}

// RunOne executes a single named reconciler synchronously. Returns the
// ReconcileReport on success, or an error when no reconciler matches.
// Used by the admin endpoint to fire a reconciler on demand.
func (r *Runner) RunOne(ctx context.Context, name string) (ReconcileReport, error) {
	if r == nil {
		return ReconcileReport{}, fmt.Errorf("runner is nil")
	}
	for _, rc := range r.periodic {
		if rc.Name() == name {
			return r.invoke(ctx, rc), nil
		}
	}
	for _, rc := range r.startupOnly {
		if rc.Name() == name {
			return r.invoke(ctx, rc), nil
		}
	}
	return ReconcileReport{}, fmt.Errorf("unknown reconciler %q", name)
}

// Names returns the names of every registered reconciler in order
// (periodic first, then startup-only). Used by the admin endpoint to
// surface available reconcilers.
func (r *Runner) Names() []string {
	if r == nil {
		return nil
	}
	out := make([]string, 0, len(r.periodic)+len(r.startupOnly))
	for _, rc := range r.periodic {
		out = append(out, rc.Name())
	}
	for _, rc := range r.startupOnly {
		out = append(out, rc.Name())
	}
	return out
}

// Metrics returns a snapshot of the per-reconciler metric map.
func (r *Runner) Metrics() map[string]ReconcilerMetrics {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(map[string]ReconcilerMetrics, len(r.metrics))
	for k, v := range r.metrics {
		out[k] = v
	}
	return out
}

// runOnce iterates the reconciler slice (plus startup-only on first
// pass) and updates per-reconciler metrics from each report.
func (r *Runner) runOnce(ctx context.Context, startup bool) {
	if startup {
		for _, rc := range r.startupOnly {
			r.invoke(ctx, rc)
		}
	}
	for _, rc := range r.periodic {
		r.invoke(ctx, rc)
	}
}

// invoke calls one reconciler and folds the result into r.metrics.
func (r *Runner) invoke(ctx context.Context, rc Reconciler) ReconcileReport {
	report := rc.Run(ctx)
	r.mu.Lock()
	m := r.metrics[rc.Name()]
	m.LastRunAt = r.clock.Now()
	m.LastDuration = report.Duration
	m.ItemsProcessed = report.ItemsProcessed
	m.ItemsFailed = report.ItemsFailed
	m.LastError = report.LastError
	m.RunCount++
	r.metrics[rc.Name()] = m
	r.mu.Unlock()
	return report
}

// =============================================================================
// Concrete Reconciler implementations
// =============================================================================

// LifecycleReconciler enforces idle/TTL/terminal lifecycle policies.
type LifecycleReconciler struct {
	svc *Service
}

// NewLifecycleReconciler wraps Service.ReconcileLifecycle as a Reconciler.
func NewLifecycleReconciler(svc *Service) *LifecycleReconciler {
	return &LifecycleReconciler{svc: svc}
}

func (r *LifecycleReconciler) Name() string { return "lifecycle" }

func (r *LifecycleReconciler) Run(ctx context.Context) ReconcileReport {
	if r == nil || r.svc == nil {
		return ReconcileReport{}
	}
	start := r.svc.clock.Now()
	r.svc.ReconcileLifecycle(ctx)
	return ReconcileReport{Duration: r.svc.clock.Since(start)}
}

// HealReconciler heals sandboxes whose mount has gone stale.
type HealReconciler struct {
	svc     *Service
	tracker *healTracker
	cfg     HealConfig
}

// NewHealReconciler wraps Service.ReconcileActiveMounts. The tracker
// is supplied so the same in-memory failure state survives across
// ticks (Phase 6 makes this durable).
func NewHealReconciler(svc *Service, tracker *healTracker, cfg HealConfig) *HealReconciler {
	return &HealReconciler{svc: svc, tracker: tracker, cfg: cfg}
}

func (r *HealReconciler) Name() string { return "heal" }

func (r *HealReconciler) Run(ctx context.Context) ReconcileReport {
	if r == nil || r.svc == nil || r.tracker == nil {
		return ReconcileReport{}
	}
	start := r.svc.clock.Now()
	r.svc.ReconcileActiveMounts(ctx, r.tracker, r.cfg)
	return ReconcileReport{Duration: r.svc.clock.Since(start)}
}

// OrphanReconciler walks the driver's BaseDir and releases dirs whose
// sandbox UUID is not registered in the repo (or is marked Deleted).
type OrphanReconciler struct {
	svc *Service
}

// NewOrphanReconciler wraps Service.ReconcileFilesystemOrphans.
func NewOrphanReconciler(svc *Service) *OrphanReconciler {
	return &OrphanReconciler{svc: svc}
}

func (r *OrphanReconciler) Name() string { return "orphan" }

func (r *OrphanReconciler) Run(ctx context.Context) ReconcileReport {
	if r == nil || r.svc == nil {
		return ReconcileReport{}
	}
	report := r.svc.ReconcileFilesystemOrphans(ctx)
	out := ReconcileReport{
		ItemsProcessed: report.FilesystemDirs,
		ItemsFailed:    report.OrphansFailed,
		Duration:       report.Duration,
		Details: map[string]any{
			"orphansCleaned": report.OrphansCleaned,
			"orphansFailed":  report.OrphansFailed,
			"failedIDs":      report.FailedIDs,
		},
	}
	if report.OrphansCleaned > 0 || report.OrphansFailed > 0 {
		log.Println(FormatOrphanReport(report))
	}
	return out
}

// DaemonReaperReconciler kills fuse-overlayfs daemons whose owning
// sandbox is gone.
type DaemonReaperReconciler struct {
	svc *Service
}

// NewDaemonReaperReconciler wraps Service.ReconcileStaleDaemons.
func NewDaemonReaperReconciler(svc *Service) *DaemonReaperReconciler {
	return &DaemonReaperReconciler{svc: svc}
}

func (r *DaemonReaperReconciler) Name() string { return "daemon-reaper" }

func (r *DaemonReaperReconciler) Run(ctx context.Context) ReconcileReport {
	if r == nil || r.svc == nil {
		return ReconcileReport{}
	}
	report := r.svc.ReconcileStaleDaemons(ctx)
	out := ReconcileReport{
		ItemsProcessed: report.Scanned,
		Duration:       report.Duration,
		Details: map[string]any{
			"reaped":       report.Reaped,
			"reapedPIDs":   report.ReapedPIDs,
			"skippedYoung": report.SkippedYoung,
			"skippedAlive": report.SkippedAlive,
		},
	}
	if report.Reaped > 0 {
		log.Println(FormatDaemonReapReport(report))
	}
	return out
}

// ManualReviewExpiryReconciler auto-denies abandoned manualReview=true
// sandboxes whose idle window exceeded the configured TTL. Startup-only:
// the TTL is a one-shot expiry, not a periodic check. Time is sourced
// from the wrapped service's clock, so tests drive expiry through the
// service's FakeClock rather than a per-reconciler `now` function.
type ManualReviewExpiryReconciler struct {
	svc *Service
	ttl time.Duration
}

// NewManualReviewExpiryReconciler wraps Service.ReconcileManualReviewExpiry.
func NewManualReviewExpiryReconciler(svc *Service, ttl time.Duration) *ManualReviewExpiryReconciler {
	return &ManualReviewExpiryReconciler{svc: svc, ttl: ttl}
}

func (r *ManualReviewExpiryReconciler) Name() string { return "manual-review-expiry" }

func (r *ManualReviewExpiryReconciler) Run(ctx context.Context) ReconcileReport {
	if r == nil || r.svc == nil || r.ttl <= 0 {
		return ReconcileReport{}
	}
	start := r.svc.clock.Now()
	r.svc.ReconcileManualReviewExpiry(ctx, r.ttl)
	return ReconcileReport{Duration: r.svc.clock.Since(start)}
}

// =============================================================================
// Default wiring
// =============================================================================

// DefaultRunner builds the production Runner with the canonical
// reconciler order. Used by main.go and the wiring tests so the order
// lives in exactly one place.
//
// Order matters:
//  1. Lifecycle: stops idle Active sandboxes and deletes expired ones.
//     This may transition repo records to Deleted, which the orphan
//     pass below cleans up on disk.
//  2. Heal: re-mounts Active sandboxes with stale mounts. Must run
//     AFTER lifecycle so we don't try to heal a sandbox lifecycle
//     just stopped.
//  3. Orphan: catches filesystem dirs the repo doesn't know about.
//     Runs after lifecycle so anything just transitioned to Deleted is
//     reaped this cycle, not next.
//  4. DaemonReaper: kills fuse-overlayfs daemons whose owning sandbox
//     is no longer in the repo. Independent of dir orphans (a daemon
//     can outlive its dir).
//  5. ArchiveRetention: evicts diff archives that exceed retention
//     levers (age, size, per-project cap). Independent of every other
//     reconciler — touches sandbox_diff_archives + the blobstore tree
//     only, never the live sandbox state.
//
// ManualReviewExpiry runs once at startup only.
//
// The heal tracker is wired with the durable repo (Phase 6) and its
// durable rows are loaded at boot before the first reconciler tick.
//
// retention may be nil — in which case the archive-retention
// reconciler is omitted. Production wiring always passes a non-nil
// provider that reads from the retention store.
func DefaultRunner(svc *Service, interval, manualReviewTTL time.Duration, healCfg HealConfig, retention RetentionPolicyProvider) *Runner {
	tracker := newHealTracker().withRepo(svc.repo)
	_ = tracker.loadFromRepo(context.Background())
	periodic := []Reconciler{
		NewLifecycleReconciler(svc),
		NewHealReconciler(svc, tracker, healCfg),
		NewOrphanReconciler(svc),
		NewDaemonReaperReconciler(svc),
	}
	if retention != nil {
		periodic = append(periodic, NewArchiveRetentionReconciler(svc, retention))
	}
	var startupOnly []Reconciler
	if manualReviewTTL > 0 {
		startupOnly = append(startupOnly, NewManualReviewExpiryReconciler(svc, manualReviewTTL))
	}
	return NewRunner(interval, periodic, startupOnly, svc.clock)
}

// =============================================================================
// Compile-time interface guards
// =============================================================================

var (
	_ Reconciler = (*LifecycleReconciler)(nil)
	_ Reconciler = (*HealReconciler)(nil)
	_ Reconciler = (*OrphanReconciler)(nil)
	_ Reconciler = (*DaemonReaperReconciler)(nil)
	_ Reconciler = (*ManualReviewExpiryReconciler)(nil)
	_ Reconciler = (*ArchiveRetentionReconciler)(nil)
)

// _ types is a doc-only alias to keep the types import live when the
// reconciler implementations don't otherwise touch types directly.
var _ = types.StatusActive
