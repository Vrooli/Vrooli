package sandbox

import (
	"context"
	"log"
	"time"

	"workspace-sandbox/internal/types"
)

// LifecycleReconciler enforces per-sandbox lifecycle policies on a schedule.
type LifecycleReconciler struct {
	service         *Service
	interval        time.Duration
	healTracker     *healTracker
	healCfg         HealConfig
	manualReviewTTL time.Duration
	stopCh          chan struct{}
	doneCh          chan struct{}
}

// NewLifecycleReconciler creates a reconciler with the given interval and heal config.
func NewLifecycleReconciler(service *Service, interval time.Duration, healCfg HealConfig) *LifecycleReconciler {
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	return &LifecycleReconciler{
		service:     service,
		interval:    interval,
		healTracker: newHealTracker(),
		healCfg:     healCfg,
		stopCh:      make(chan struct{}),
		doneCh:      make(chan struct{}),
	}
}

// WithManualReviewTTL configures the TTL for abandoned manualReview=true
// sandboxes. Per the auditability contract (Phase 4), sandboxes with
// ManualReview=true that remain idle past this TTL are auto-denied with
// Source=SourceWorkspaceSandboxGC. Zero disables expiry.
func (r *LifecycleReconciler) WithManualReviewTTL(ttl time.Duration) *LifecycleReconciler {
	if r != nil {
		r.manualReviewTTL = ttl
	}
	return r
}

// Start begins the reconciliation loop in a goroutine.
//
// On the very first tick (before the timer kicks in), all four
// reconcilers run synchronously so any pre-existing drift — including
// filesystem orphans accumulated while the API was down — is cleaned
// up before normal serving begins. Subsequent ticks run the same
// reconcilers except ReconcileManualReviewExpiry, which is
// startup-only because the manual-review TTL is a one-shot expiry,
// not a periodic check (see the comment in lifecycle_test.go for the
// design rationale).
//
// The orphan reconciler (ReconcileFilesystemOrphans) was added in
// 2026-04-28 after a mount-leak incident — see orphan_reconciler.go
// for the why.
func (r *LifecycleReconciler) Start() {
	if r == nil || r.service == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()
		defer close(r.doneCh)

		ctx := context.Background()
		r.runReconcilers(ctx, true)
		for {
			select {
			case <-ticker.C:
				r.runReconcilers(context.Background(), false)
			case <-r.stopCh:
				return
			}
		}
	}()
}

// runReconcilers fires the periodic reconcilers in dependency order.
// The startup pass additionally runs ReconcileManualReviewExpiry, which
// is a one-shot at boot rather than a periodic check.
//
// Order matters:
//  1. Lifecycle: stops idle Active sandboxes and deletes expired ones.
//     This may transition repo records to Deleted, which the orphan
//     pass below will then clean up on disk.
//  2. ManualReviewExpiry (startup only): auto-denies abandoned manual-
//     review sandboxes; same Deleted->orphan cascade as above.
//  3. ActiveMounts: re-mounts Active sandboxes with stale mounts. Must
//     run AFTER lifecycle so we don't try to heal a sandbox that
//     lifecycle just stopped.
//  4. FilesystemOrphans: catches dirs the repo doesn't know about
//     (drift, crash, agent-manager Delete that the API forgot, etc.).
//     Last so anything the earlier passes transitioned to Deleted is
//     reaped this cycle rather than next.
func (r *LifecycleReconciler) runReconcilers(ctx context.Context, startup bool) {
	r.service.ReconcileLifecycle(ctx)
	if startup {
		r.service.ReconcileManualReviewExpiry(ctx, r.manualReviewTTL, time.Now)
	}
	r.service.ReconcileActiveMounts(ctx, r.healTracker, r.healCfg)
	report := r.service.ReconcileFilesystemOrphans(ctx)
	// Only log when we found something or on startup, so steady-state
	// noise stays low. Operators can grep boot logs for "orphan-reconciler"
	// to verify the schedule is firing.
	if startup || report.OrphansCleaned > 0 || report.OrphansFailed > 0 {
		log.Println(FormatOrphanReport(report))
	}
}

// Stop stops the reconciliation loop.
func (r *LifecycleReconciler) Stop() {
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

// ReconcileLifecycle enforces lifecycle policies for all sandboxes.
func (s *Service) ReconcileLifecycle(ctx context.Context) {
	if s == nil || s.repo == nil {
		return
	}

	filter := &types.ListFilter{
		Status: []types.Status{
			types.StatusActive,
			types.StatusStopped,
			types.StatusApproved,
			types.StatusRejected,
			types.StatusError,
		},
		Limit: 10000,
	}
	result, err := s.repo.List(ctx, filter)
	if err != nil || result == nil {
		return
	}

	now := time.Now()
	for _, sandbox := range result.Sandboxes {
		if sandbox == nil {
			continue
		}
		if sandbox.Status == types.StatusDeleted {
			continue
		}
		behavior := normalizeBehavior(sandbox.Behavior)
		applyLifecycleIdle(ctx, s, sandbox, behavior, now)
		applyLifecycleTTL(ctx, s, sandbox, behavior, now)
		applyLifecycleTerminal(ctx, s, sandbox, behavior)
	}
}

func applyLifecycleIdle(ctx context.Context, s *Service, sandbox *types.Sandbox, behavior types.SandboxBehavior, now time.Time) {
	if behavior.Lifecycle.IdleTimeout <= 0 {
		return
	}
	if sandbox.Status != types.StatusActive {
		return
	}
	if now.Sub(sandbox.LastUsedAt) < behavior.Lifecycle.IdleTimeout {
		return
	}
	if _, err := s.Stop(ctx, sandbox.ID); err != nil {
		s.logAuditEvent(ctx, sandbox, "sandbox.warning", "system", "system", map[string]interface{}{
			"message": "failed to stop idle sandbox: " + err.Error(),
		})
	}
}

func applyLifecycleTTL(ctx context.Context, s *Service, sandbox *types.Sandbox, behavior types.SandboxBehavior, now time.Time) {
	if behavior.Lifecycle.TTL <= 0 {
		return
	}
	if sandbox.Status == types.StatusActive || sandbox.Status == types.StatusCreating {
		return
	}
	if now.Sub(sandbox.CreatedAt) < behavior.Lifecycle.TTL {
		return
	}
	if err := s.Delete(ctx, sandbox.ID); err != nil {
		s.logAuditEvent(ctx, sandbox, "sandbox.warning", "system", "system", map[string]interface{}{
			"message": "failed to delete sandbox after ttl: " + err.Error(),
		})
	}
}

func applyLifecycleTerminal(ctx context.Context, s *Service, sandbox *types.Sandbox, behavior types.SandboxBehavior) {
	if sandbox.Status != types.StatusApproved && sandbox.Status != types.StatusRejected {
		return
	}
	if !shouldDeleteOnStatus(behavior.Lifecycle, sandbox.Status) {
		return
	}
	if err := s.Delete(ctx, sandbox.ID); err != nil {
		s.logAuditEvent(ctx, sandbox, "sandbox.warning", "system", "system", map[string]interface{}{
			"message": "failed to delete sandbox on terminal status: " + err.Error(),
		})
	}
}

// ReconcileManualReviewExpiry auto-denies abandoned manualReview=true
// sandboxes whose idle window exceeds the configured ttl. The originating
// surface on the resulting state transition is recorded as
// SourceWorkspaceSandboxGC (system-only) so reviewers can tell GC-driven
// denials apart from operator denials.
//
// `now` is injected so tests can drive expiry deterministically; production
// callers pass time.Now.
//
// Phase 4 of agent-sandbox-audit-foundation. See
// scenarios/workspace-sandbox/docs/AUDITABILITY_CONTRACT.md.
func (s *Service) ReconcileManualReviewExpiry(ctx context.Context, ttl time.Duration, now func() time.Time) {
	if s == nil || s.repo == nil || ttl <= 0 {
		return
	}
	if now == nil {
		now = time.Now
	}

	filter := &types.ListFilter{
		Status: []types.Status{types.StatusActive, types.StatusStopped},
		Limit:  10000,
	}
	result, err := s.repo.List(ctx, filter)
	if err != nil || result == nil {
		return
	}

	cutoff := now()
	for _, sandbox := range result.Sandboxes {
		if sandbox == nil {
			continue
		}
		if !sandbox.Behavior.ManualReview {
			continue
		}
		// Idle window measured against LastUsedAt — for manualReview
		// sandboxes this is the run-end timestamp (no further activity
		// after the agent-manager run terminates).
		if cutoff.Sub(sandbox.LastUsedAt) < ttl {
			continue
		}

		sandbox.Status = types.StatusRejected
		if err := s.repo.Update(ctx, sandbox); err != nil {
			s.logAuditEvent(ctx, sandbox, "sandbox.warning", "system", "system", map[string]interface{}{
				"message": "manual-review TTL: failed to mark rejected: " + err.Error(),
			})
			continue
		}
		s.logAuditEventWithSource(ctx, sandbox, "rejected", "system", "system", types.SourceWorkspaceSandboxGC, map[string]interface{}{
			"reason":          "manualReview-ttl-expired",
			"manualReviewTtl": ttl.String(),
			"lastUsedAt":      sandbox.LastUsedAt.Format(time.RFC3339),
		})
		// Best-effort tear-down via existing terminal-status path.
		if err := s.Delete(ctx, sandbox.ID); err != nil {
			s.logAuditEvent(ctx, sandbox, "sandbox.warning", "system", "system", map[string]interface{}{
				"message": "manual-review TTL: failed to delete sandbox after auto-deny: " + err.Error(),
			})
		}
	}
}
