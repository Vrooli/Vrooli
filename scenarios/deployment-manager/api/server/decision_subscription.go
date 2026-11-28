package server

import (
	"context"
	"fmt"
	"time"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
)

const (
	decisionSubscriptionOwner  = "deployment-manager"
	decisionSubscriptionName   = "deployment-manager-decision-answers"
	decisionSubscriptionCycle  = 30 * time.Second
	decisionSubscriptionWindow = 15 * time.Second
)

// reconcileDecisionEventSubscription keeps exactly one enabled vrooli-events
// subscription that delivers notification-hub ask resolutions to the
// deployment-manager release-decision webhook. Without it an answered release
// decision is never delivered back, so the ask-don't-block round-trip is
// dead-on-arrival. vrooli-events owns the durable row; this only ensures it
// exists and targets the live webhook. Ported from swarm-manager's
// reconcileDecisionEventSubscription.
func reconcileDecisionEventSubscriptionWithResolver(ctx context.Context, target string, resolve func(context.Context) (string, error)) error {
	base, err := resolve(ctx)
	if err != nil {
		return err
	}
	client, err := eventbus.NewSubscriptionClient(base, nil)
	if err != nil {
		return err
	}
	const pattern = operatordecision.AskResolvedEventType
	payload := eventbus.Subscription{
		Name: decisionSubscriptionName, OwnerScenario: decisionSubscriptionOwner,
		EventPattern: pattern, DeliveryType: "webhook", DeliveryTarget: target, Enabled: true,
	}
	// Deployment Manager has one AskResolved webhook route. Older releases
	// stored that same route under several names, so this owner explicitly
	// adopts the full exact tuple; target-only or partial matches remain foreign.
	return client.EnsureNamedSubscriptionWithExactRouteAdoption(ctx, payload)
}

// reconcileDecisionEventSubscriptionWithRetry bounds startup recovery when
// Events is unavailable or restarting. It never delays the server itself; the
// caller runs it in a background goroutine with a total deadline.
func reconcileDecisionEventSubscriptionWithRetry(ctx context.Context, target string, resolve func(context.Context) (string, error), delay time.Duration) error {
	err := eventbus.ReconcileSubscription(ctx, eventbus.SubscriptionReconcilerConfig{
		Attempts: 5, AttemptTimeout: decisionSubscriptionWindow, RetryDelay: delay,
	}, func(attempt context.Context) error {
		return reconcileDecisionEventSubscriptionWithResolver(attempt, target, resolve)
	})
	if err != nil {
		return fmt.Errorf("reconcile %s decision subscription: %w", decisionSubscriptionOwner, err)
	}
	return nil
}

// runDecisionEventSubscriptionReconciler owns periodic recovery for this
// optional integration. Every reconciliation cycle has a deadline, cycles do
// not overlap, and the caller's server lifecycle context stops retries during
// shutdown.
func runDecisionEventSubscriptionReconciler(ctx context.Context, target string, resolve func(context.Context) (string, error), cycle, window, delay time.Duration, wait func(context.Context, time.Duration) bool, report func(error)) {
	var waitCycle func(context.Context, time.Duration) bool
	if wait != nil {
		waitCycle = wait
	}
	eventbus.RunSubscriptionReconciler(ctx, eventbus.SubscriptionReconcilerConfig{
		Interval: cycle, AttemptTimeout: window, CycleTimeout: window, Attempts: 5,
		RetryDelayForAttempt: func(attempt int) time.Duration { return delay * time.Duration(1<<attempt) },
		Wait:                 waitCycle,
	}, func(ctx context.Context) error {
		return reconcileDecisionEventSubscriptionWithResolver(ctx, target, resolve)
	}, report)
}

func waitDecisionSubscriptionCycle(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return ctx.Err() == nil
	}
}
