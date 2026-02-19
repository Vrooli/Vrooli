package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// [REQ:RECOVER-004] Exponential backoff on failed recovery
func TestRecoveryExponentialBackoff(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Server always fails — simulates persistent tunnel outage
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer ts.Close()

	checker := NewTunnelHealthChecker(
		WithCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("active\n"), nil
		}),
		WithMetricsURL(ts.URL),
	)

	// Use short backoff schedule for testing
	engine := NewRecoveryEngine(db, checker,
		WithConsecutiveFailures(1),
		WithMaxBackoffRetries(5),
		WithRecoveryCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			// Restart "succeeds" but tunnel stays unhealthy
			return nil, fmt.Errorf("restart failed")
		}),
	)
	engine.BackoffSchedule = []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		40 * time.Millisecond,
	}

	// First evaluation triggers recovery (fails)
	evt, err := engine.Evaluate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if evt == nil || evt.Outcome != "failure" {
		t.Fatalf("expected failed recovery, got %+v", evt)
	}

	state := engine.State()
	if state.BackoffLevel != 1 {
		t.Errorf("backoff_level = %d, want 1", state.BackoffLevel)
	}
	if state.NextRetryAfter.IsZero() {
		t.Error("expected next_retry_after to be set")
	}

	// Second evaluation should be suppressed by backoff
	evt, err = engine.Evaluate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if evt != nil {
		t.Error("expected no recovery during backoff period")
	}

	// Wait for backoff to expire, then evaluate again
	time.Sleep(15 * time.Millisecond)
	evt, err = engine.Evaluate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if evt == nil {
		t.Fatal("expected recovery after backoff expired")
	}
	if evt.Outcome != "failure" {
		t.Errorf("outcome = %q, want failure", evt.Outcome)
	}

	state = engine.State()
	if state.BackoffLevel != 2 {
		t.Errorf("backoff_level = %d, want 2", state.BackoffLevel)
	}
}

// [REQ:RECOVER-004] Backoff resets after successful recovery
func TestRecoveryBackoffResetsOnSuccess(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	recoveryAttempt := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Fails on first recovery, succeeds on second
		if recoveryAttempt >= 2 {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	defer ts.Close()

	checker := NewTunnelHealthChecker(
		WithCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("active\n"), nil
		}),
		WithMetricsURL(ts.URL),
	)

	engine := NewRecoveryEngine(db, checker,
		WithConsecutiveFailures(1),
		WithRecoveryCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			recoveryAttempt++
			if recoveryAttempt >= 2 {
				return nil, nil // successful restart
			}
			return nil, fmt.Errorf("restart failed")
		}),
	)
	engine.BackoffSchedule = []time.Duration{5 * time.Millisecond}

	// First: fails
	evt, _ := engine.Evaluate(context.Background())
	if evt == nil || evt.Outcome != "failure" {
		t.Fatal("expected first recovery to fail")
	}

	// Wait out backoff
	time.Sleep(10 * time.Millisecond)

	// Second: succeeds
	evt, _ = engine.Evaluate(context.Background())
	if evt == nil || evt.Outcome != "success" {
		t.Fatal("expected second recovery to succeed")
	}

	state := engine.State()
	if state.BackoffLevel != 0 {
		t.Errorf("backoff_level = %d, want 0 after success", state.BackoffLevel)
	}
	if state.FailedRecovery != 0 {
		t.Errorf("failed_recoveries = %d, want 0 after success", state.FailedRecovery)
	}
}
