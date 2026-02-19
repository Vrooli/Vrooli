package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// [REQ:RECOVER-005] Circuit breaker opens after consecutive failures
func TestCircuitBreakerOpens(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

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

	engine := NewRecoveryEngine(db, checker,
		WithConsecutiveFailures(1),
		WithMaxBackoffRetries(3), // open circuit after 3 failed recoveries
		WithRecoveryCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, fmt.Errorf("restart failed")
		}),
	)
	engine.BackoffSchedule = []time.Duration{1 * time.Millisecond}

	// Trigger 3 failed recoveries
	for i := 0; i < 3; i++ {
		time.Sleep(2 * time.Millisecond) // wait out backoff
		evt, err := engine.Evaluate(context.Background())
		if err != nil {
			t.Fatalf("evaluate %d: %v", i, err)
		}
		if evt == nil {
			t.Fatalf("expected recovery event on attempt %d", i+1)
		}
		if evt.Outcome != "failure" {
			t.Fatalf("expected failure on attempt %d, got %q", i+1, evt.Outcome)
		}
	}

	state := engine.State()
	if !state.CircuitOpen {
		t.Error("expected circuit breaker to be open")
	}
	if state.Status != "circuit_open" {
		t.Errorf("status = %q, want circuit_open", state.Status)
	}

	// Further evaluations should be suppressed
	evt, err := engine.Evaluate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if evt != nil {
		t.Error("expected no recovery when circuit is open")
	}
}

// [REQ:RECOVER-005] Circuit breaker resets after cooldown
func TestCircuitBreakerResetsAfterCooldown(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	readyOK := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if readyOK {
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
		WithMaxBackoffRetries(2),
		WithCircuitCooldown(10*time.Millisecond),
		WithRecoveryCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, fmt.Errorf("restart failed") // always fail to trip circuit
		}),
	)
	engine.BackoffSchedule = []time.Duration{1 * time.Millisecond}
	engine.ReadyPollTimeout = 5 * time.Millisecond
	engine.ReadyPollInterval = 1 * time.Millisecond

	// Trigger circuit open with 2 failed recoveries
	for i := 0; i < 2; i++ {
		time.Sleep(2 * time.Millisecond)
		_, _ = engine.Evaluate(context.Background())
	}

	state := engine.State()
	if !state.CircuitOpen {
		t.Fatal("expected circuit to be open")
	}

	// Wait for cooldown
	time.Sleep(15 * time.Millisecond)

	// Make tunnel healthy so evaluation finds no issue
	readyOK = true
	evt, err := engine.Evaluate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// After cooldown, circuit resets and health check succeeds — no recovery needed
	if evt != nil {
		t.Logf("got event: %+v", evt)
	}

	state = engine.State()
	if state.CircuitOpen {
		t.Error("expected circuit to be reset after cooldown")
	}
}

// [REQ:RECOVER-005] Manual circuit reset
func TestCircuitBreakerManualReset(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

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

	engine := NewRecoveryEngine(db, checker,
		WithConsecutiveFailures(1),
		WithMaxBackoffRetries(1),
		WithRecoveryCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, fmt.Errorf("failed")
		}),
	)
	engine.BackoffSchedule = []time.Duration{1 * time.Millisecond}

	// Trip circuit
	_, _ = engine.Evaluate(context.Background())

	if !engine.State().CircuitOpen {
		t.Fatal("expected circuit to be open")
	}

	engine.ResetCircuit()

	state := engine.State()
	if state.CircuitOpen {
		t.Error("circuit should be closed after manual reset")
	}
	if state.Status != "idle" {
		t.Errorf("status = %q, want idle after reset", state.Status)
	}
}
