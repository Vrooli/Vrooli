package sandbox

import (
	"context"
	"time"

	"workspace-sandbox/internal/types"
)

// LifecycleReconciler enforces per-sandbox lifecycle policies on a schedule.
type LifecycleReconciler struct {
	service  *Service
	interval time.Duration
	stopCh   chan struct{}
	doneCh   chan struct{}
}

// NewLifecycleReconciler creates a reconciler with the given interval.
func NewLifecycleReconciler(service *Service, interval time.Duration) *LifecycleReconciler {
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	return &LifecycleReconciler{
		service:  service,
		interval: interval,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

// Start begins the reconciliation loop in a goroutine.
func (r *LifecycleReconciler) Start() {
	if r == nil || r.service == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()
		defer close(r.doneCh)

		r.service.ReconcileLifecycle(context.Background())
		for {
			select {
			case <-ticker.C:
				r.service.ReconcileLifecycle(context.Background())
			case <-r.stopCh:
				return
			}
		}
	}()
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
