package integrations

import (
	"context"
	"time"
)

// RetrySubscriptions continuously reconciles subscriptions until the owner
// shuts down, including after Events loses or disables a subscription.
func RetrySubscriptions(ctx context.Context, interval time.Duration, reconcile func(context.Context, string) error, kinds ...string) {
	for len(kinds) > 0 {
		for _, kind := range kinds {
			if ctx.Err() != nil {
				return
			}
			attempt, cancel := context.WithTimeout(ctx, 15*time.Second)
			_ = reconcile(attempt, kind)
			cancel()
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
