package wiring

import (
	"context"
	"errors"
	"time"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
)

const (
	programRuntimeEventPattern = "vrooli.program_runtime.program_event.v1"
	// askResolvedEventPattern carries operator answers to effort decisions.
	askResolvedEventPattern = "notification_hub.ask.resolved.v1"
	// askContextRequestedEventPattern carries the operator's "I need more context"
	// questions about decisions a verified run raised.
	askContextRequestedEventPattern = operatordecision.AskContextRequestedEventType
)

// ReconcileProgramRuntimeSubscription creates or updates the program-runtime
// subscription owned by agent-manager. vrooli-events remains the owner of the
// durable row.
func ReconcileProgramRuntimeSubscription(ctx context.Context, target string) error {
	return reconcileEventSubscription(ctx, "agent-manager-program-events", programRuntimeEventPattern, target)
}

// ReconcileDecisionEventSubscription creates or updates the subscription that
// delivers notification-hub ask resolutions to the effort decision webhook.
func ReconcileDecisionEventSubscription(ctx context.Context, target string) error {
	return reconcileEventSubscription(ctx, "agent-manager-decision-answers", askResolvedEventPattern, target)
}

// ReconcileDecisionContextSubscription creates or updates the subscription that
// delivers the operator's questions about a decision to the same webhook, which
// hands each to the run that raised the decision.
func ReconcileDecisionContextSubscription(ctx context.Context, target string) error {
	return reconcileEventSubscription(ctx, "agent-manager-decision-context-requests", askContextRequestedEventPattern, target)
}

// reconcileEventSubscription keeps one enabled subscription per delivery
// target.
func reconcileEventSubscription(ctx context.Context, name, pattern, target string) error {
	base, err := discovery.ResolveScenarioURLDefault(ctx, "vrooli-events")
	if err != nil {
		return err
	}
	return reconcileEventSubscriptionAt(ctx, base, name, pattern, target)
}

func reconcileEventSubscriptionAt(ctx context.Context, base, name, pattern, target string) error {
	client, err := eventbus.NewSubscriptionClient(base, nil)
	if err != nil {
		return err
	}
	return client.EnsureNamedSubscription(ctx, eventbus.Subscription{
		Name: name, OwnerScenario: "agent-manager", EventPattern: pattern,
		DeliveryType: "webhook", DeliveryTarget: target, Enabled: true,
	})
}

// retryEventSubscriptions keeps subscriptions reconciled after startup as
// well as after an Events restart or a removed/disabled subscription.
func retryEventSubscriptions(ctx context.Context, interval time.Duration, checks ...func(context.Context) error) {
	if len(checks) == 0 {
		return
	}
	eventbus.RunSubscriptionReconciler(ctx, eventbus.SubscriptionReconcilerConfig{
		Interval: interval, AttemptTimeout: 15 * time.Second, Attempts: 1,
	}, func(ctx context.Context) error {
		var errs []error
		for _, check := range checks {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := check(ctx); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	}, nil)
}
