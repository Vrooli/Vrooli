package eventbus

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// SubscriptionReconcilerConfig controls lifecycle scheduling for a durable
// named-subscription reconciliation. Reconcile should resolve the current
// Events endpoint and reconcile all desired subscriptions on every call.
type SubscriptionReconcilerConfig struct {
	Interval       time.Duration
	AttemptTimeout time.Duration
	CycleTimeout   time.Duration
	Attempts       int
	RetryDelay     time.Duration
	// RetryDelayForAttempt overrides RetryDelay with a per-retry policy. The
	// argument is the zero-based retry number (first retry is 0).
	RetryDelayForAttempt func(retry int) time.Duration
	// Wait may be supplied by deterministic tests. Production callers should
	// leave it nil to use a context-aware timer.
	Wait func(context.Context, time.Duration) bool
}

// RunSubscriptionReconciler performs an immediate reconciliation and repeats
// it until ctx is canceled. Each periodic cycle uses a bounded number of
// bounded attempts, then waits Interval before starting another cycle. The
// callback runs again for each attempt, allowing endpoint re-resolution after
// Events restarts or port changes. Errors are reported once per failed cycle.
func RunSubscriptionReconciler(ctx context.Context, cfg SubscriptionReconcilerConfig, reconcile func(context.Context) error, report func(error)) {
	if ctx == nil || reconcile == nil || cfg.Interval <= 0 || cfg.AttemptTimeout <= 0 || cfg.Attempts <= 0 || cfg.RetryDelay < 0 {
		return
	}
	wait := subscriptionReconcilerWait(cfg)
	for ctx.Err() == nil {
		err := ReconcileSubscription(ctx, cfg, reconcile)
		if ctx.Err() != nil {
			return
		}
		if err != nil && ctx.Err() == nil && report != nil {
			report(err)
		}
		if !wait(ctx, cfg.Interval) {
			return
		}
	}
}

// ReconcileSubscription performs one bounded retry cycle. Use it directly for
// startup windows; RunSubscriptionReconciler adds periodic recovery around the
// same policy for long-lived owners.
func ReconcileSubscription(ctx context.Context, cfg SubscriptionReconcilerConfig, reconcile func(context.Context) error) error {
	if ctx == nil || reconcile == nil || cfg.AttemptTimeout <= 0 || cfg.Attempts <= 0 || cfg.RetryDelay < 0 {
		return errors.New("invalid subscription reconciliation configuration")
	}
	cycleCtx := ctx
	cycleCancel := func() {}
	if cfg.CycleTimeout > 0 {
		cycleCtx, cycleCancel = context.WithTimeout(ctx, cfg.CycleTimeout)
	}
	defer cycleCancel()
	return reconcileSubscriptionWithRetry(cycleCtx, cfg, reconcile, subscriptionReconcilerWait(cfg))
}

func subscriptionReconcilerWait(cfg SubscriptionReconcilerConfig) func(context.Context, time.Duration) bool {
	if cfg.Wait != nil {
		return cfg.Wait
	}
	return waitSubscriptionReconciler
}

func reconcileSubscriptionWithRetry(ctx context.Context, cfg SubscriptionReconcilerConfig, reconcile func(context.Context) error, wait func(context.Context, time.Duration) bool) error {
	var lastErr error
	for attempt := 0; attempt < cfg.Attempts; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		attemptCtx, cancel := context.WithTimeout(ctx, cfg.AttemptTimeout)
		lastErr = reconcile(attemptCtx)
		cancel()
		if lastErr == nil {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if attempt+1 == cfg.Attempts {
			break
		}
		delay := cfg.RetryDelay
		if cfg.RetryDelayForAttempt != nil {
			delay = cfg.RetryDelayForAttempt(attempt)
		}
		if delay < 0 {
			delay = 0
		}
		if !wait(ctx, delay) {
			if err := ctx.Err(); err != nil {
				return err
			}
			return lastErr
		}
	}
	return fmt.Errorf("subscription reconciliation failed after %d bounded attempts: %w", cfg.Attempts, lastErr)
}

func waitSubscriptionReconciler(ctx context.Context, duration time.Duration) bool {
	if duration <= 0 {
		return ctx.Err() == nil
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return ctx.Err() == nil
	}
}
