package eventbus

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRunSubscriptionReconcilerRetriesAndContinuesPeriodicRecovery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	attempts, waits, reported := 0, 0, 0
	var endpoints []string
	resolvedEndpoint := "http://events-old"
	cfg := SubscriptionReconcilerConfig{
		Interval: time.Minute, AttemptTimeout: time.Second, Attempts: 2, RetryDelay: time.Second,
		Wait: func(context.Context, time.Duration) bool {
			waits++
			if waits == 1 {
				resolvedEndpoint = "http://events-new"
			}
			// Wait 1 is startup backoff, wait 2 is the cycle interval. Let a
			// second cycle run, then cancel at its backoff.
			return waits <= 2
		},
	}
	RunSubscriptionReconciler(ctx, cfg, func(context.Context) error {
		attempts++
		endpoints = append(endpoints, resolvedEndpoint)
		if attempts == 1 {
			return errors.New("Events restarting")
		}
		if attempts == 2 {
			return nil
		}
		return errors.New("Events unavailable again")
	}, func(error) { reported++ })
	if attempts != 3 {
		t.Fatalf("reconcile attempts = %d, want 3", attempts)
	}
	if reported != 1 {
		t.Fatalf("reported failed cycles = %d, want 1", reported)
	}
	if got, want := endpoints, []string{"http://events-old", "http://events-new", "http://events-new"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("resolved endpoints = %v, want %v", got, want)
	}
}

func TestRunSubscriptionReconcilerBoundsAttemptsAndWaits(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	attempts, waits := 0, 0
	var reported error
	RunSubscriptionReconciler(ctx, SubscriptionReconcilerConfig{
		Interval: time.Minute, AttemptTimeout: time.Second, Attempts: 3, RetryDelay: time.Second,
		Wait: func(context.Context, time.Duration) bool {
			waits++
			return waits < 3
		},
	}, func(context.Context) error {
		attempts++
		return errors.New("still unavailable")
	}, func(err error) { reported = err })
	if attempts != 3 {
		t.Fatalf("attempts = %d, want bounded maximum 3", attempts)
	}
	if reported == nil || !strings.Contains(reported.Error(), "after 3 bounded attempts") {
		t.Fatalf("reported error = %v, want bounded attempt error", reported)
	}
}

func TestReconcileSubscriptionCancelsPromptly(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0
	err := ReconcileSubscription(ctx, SubscriptionReconcilerConfig{
		Attempts: 5, AttemptTimeout: time.Second, RetryDelay: time.Hour,
	}, func(context.Context) error {
		attempts++
		cancel()
		return errors.New("unavailable")
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context cancellation", err)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want cancellation to stop retries", attempts)
	}
}

func TestRunSubscriptionReconcilerStopsOnOwnerShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	attempts, waits := 0, 0
	RunSubscriptionReconciler(ctx, SubscriptionReconcilerConfig{
		Interval: time.Hour, AttemptTimeout: time.Second, Attempts: 5, RetryDelay: time.Hour,
		Wait: func(ctx context.Context, _ time.Duration) bool {
			waits++
			return waitSubscriptionReconciler(ctx, time.Hour)
		},
	}, func(context.Context) error {
		attempts++
		cancel()
		return errors.New("Events unavailable during shutdown")
	}, nil)
	if attempts != 1 || waits != 0 {
		t.Fatalf("attempts=%d waits=%d, want one attempt and no retry after shutdown", attempts, waits)
	}
}

func TestRunSubscriptionReconcilerBoundsAttemptContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var gotDeadline time.Time
	RunSubscriptionReconciler(ctx, SubscriptionReconcilerConfig{
		Interval: time.Second, AttemptTimeout: time.Second, Attempts: 1,
		Wait: func(context.Context, time.Duration) bool { return false },
	}, func(attempt context.Context) error {
		var ok bool
		gotDeadline, ok = attempt.Deadline()
		if !ok {
			t.Fatal("attempt context has no deadline")
		}
		return nil
	}, nil)
	remaining := time.Until(gotDeadline)
	if gotDeadline.IsZero() || remaining > time.Second || remaining <= 0 {
		t.Fatalf("attempt deadline = %v (%v remaining), want a live deadline within one second", gotDeadline, remaining)
	}
}
