package sandbox

import (
	"context"
	"time"

	"workspace-sandbox/internal/types"
)

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

	now := s.clock.Now()
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
// Time is sourced from s.clock so tests can drive expiry deterministically
// via FakeClock.SetNow / FakeClock.Advance.
//
// Phase 4 of agent-sandbox-audit-foundation. See
// scenarios/workspace-sandbox/docs/AUDITABILITY_CONTRACT.md.
func (s *Service) ReconcileManualReviewExpiry(ctx context.Context, ttl time.Duration) {
	if s == nil || s.repo == nil || ttl <= 0 {
		return
	}

	filter := &types.ListFilter{
		Status: []types.Status{types.StatusActive, types.StatusStopped},
		Limit:  10000,
	}
	result, err := s.repo.List(ctx, filter)
	if err != nil || result == nil {
		return
	}

	cutoff := s.clock.Now()
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
