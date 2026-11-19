package integrations

import (
	"context"
	"errors"
	"time"

	"github.com/vrooli/api-core/eventbus"
)

// RetrySubscriptions continuously reconciles subscriptions until the owner
// shuts down, including after Events loses or disables a subscription.
func RetrySubscriptions(ctx context.Context, interval time.Duration, reconcile func(context.Context, string) error, kinds ...string) {
	if reconcile == nil || len(kinds) == 0 {
		return
	}
	eventbus.RunSubscriptionReconciler(ctx, eventbus.SubscriptionReconcilerConfig{
		Interval: interval, AttemptTimeout: 15 * time.Second, Attempts: 1,
	}, func(ctx context.Context) error {
		var errs []error
		for _, kind := range kinds {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := reconcile(ctx, kind); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	}, nil)
}
