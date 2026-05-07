package aisearch

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

// SyncLoop periodically drives Reconciler.RunOnce on a ticker. Mirrors
// internal/feedback/sweeper.go in shape: RunOnce is exported separately for
// synchronous boot-time use and tests; Start blocks on a ticker until the
// caller's context is canceled.
//
// Decision boundary: "should we tick now?" — Disabled and Interval are the
// only knobs operators see; both are surfaced via env vars (read by
// NewSyncLoop) so a kill switch is one named decision rather than a flag
// scattered across runtime if-checks.
type SyncLoop struct {
	Reconciler *Reconciler
	Interval   time.Duration
	Disabled   bool
	Clock      func() time.Time
}

// NewSyncLoop constructs a SyncLoop, reading Interval and Disabled from the
// process environment via the env-var resolvers. Operators tune behavior
// through env, never via code edits.
func NewSyncLoop(r *Reconciler) *SyncLoop {
	return &SyncLoop{
		Reconciler: r,
		Interval:   ResolveSyncInterval(),
		Disabled:   ResolveSyncDisabled(),
		Clock:      time.Now,
	}
}

// RunOnce drives one Reconciler.RunOnce. ErrReconcileBusy is swallowed: a
// previous RunOnce already in flight is doing the work this tick would have.
// Other errors are logged at warn level — they never panic the loop.
//
// Returns the plan/result for tests; production callers ignore the values.
func (s *SyncLoop) RunOnce(ctx context.Context) (*DriftReport, *ApplyResult, error) {
	if s == nil || s.Reconciler == nil {
		return nil, nil, nil
	}
	plan, result, err := s.Reconciler.RunOnce(ctx)
	switch {
	case errors.Is(err, ErrReconcileBusy):
		// Concurrent tick raced; not an error condition.
		return plan, result, nil
	case err != nil:
		slog.Warn("[aisearch] sync_loop: reconcile failed", "err", err)
		return plan, result, err
	}
	if plan != nil && plan.HasWork() {
		slog.Info("[aisearch] sync_loop: reconcile applied",
			"upserts", plan.UpsertCount(),
			"deletes", plan.DeleteCount(),
			"unchanged", plan.UnchangedBacklog+plan.UnchangedInitiative,
			"legacy", plan.LegacyBacklog+plan.LegacyInitiative,
		)
	}
	return plan, result, nil
}

// Start blocks until ctx is canceled, calling RunOnce on every Interval tick.
// No-op when Disabled or Interval <= 0 — the boot-time RunOnce in main.go
// remains the convergence guarantee in those cases.
//
// Panics inside RunOnce are recovered so a transient disk error or store
// inconsistency doesn't kill the safety net.
func (s *SyncLoop) Start(ctx context.Context) {
	if s == nil || s.Reconciler == nil {
		return
	}
	if s.Disabled {
		slog.Info("[aisearch] sync_loop disabled via AI_SEARCH_SYNC_DISABLED; periodic reconcile will not run")
		return
	}
	if s.Interval <= 0 {
		slog.Info("[aisearch] sync_loop interval ≤ 0; periodic reconcile will not run", "interval", s.Interval)
		return
	}
	slog.Info("[aisearch] sync_loop started", "interval", s.Interval)
	t := time.NewTicker(s.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("[aisearch] sync_loop stopped", "reason", ctx.Err())
			return
		case <-t.C:
			s.runWithRecover(ctx)
		}
	}
}

func (s *SyncLoop) runWithRecover(ctx context.Context) {
	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("[aisearch] sync_loop: panic recovered", "panic", rec)
		}
	}()
	_, _, _ = s.RunOnce(ctx)
}
