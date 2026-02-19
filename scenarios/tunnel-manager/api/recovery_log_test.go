package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// [REQ:RECOVER-006] Recovery event persistence
func TestRecoveryEventPersistence(t *testing.T) {
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
		WithRecoveryCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			readyOK = true
			return nil, nil
		}),
	)

	// Trigger recovery
	evt, err := engine.Evaluate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if evt == nil {
		t.Fatal("expected recovery event")
	}

	// Check event was persisted
	events, err := engine.ListEvents(10)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected at least one persisted event")
	}
	found := events[0]
	if found.TriggerType != "ready_failure" {
		t.Errorf("trigger_type = %q, want ready_failure", found.TriggerType)
	}
	if found.Action != "systemctl_restart" {
		t.Errorf("action = %q, want systemctl_restart", found.Action)
	}
	if found.Outcome != "success" {
		t.Errorf("outcome = %q, want success", found.Outcome)
	}
}

// [REQ:RECOVER-007] Manual recovery via API
func TestManualRecovery(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	checker := NewTunnelHealthChecker(
		WithCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("active\n"), nil
		}),
		WithMetricsURL(ts.URL),
	)

	var restartCalled bool
	engine := NewRecoveryEngine(db, checker,
		WithRecoveryCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			restartCalled = true
			return nil, nil
		}),
	)

	evt, err := engine.TriggerManual(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if !restartCalled {
		t.Error("expected restart to be called")
	}
	if evt.Outcome != "success" {
		t.Errorf("outcome = %q, want success", evt.Outcome)
	}
	if evt.TriggerType != "manual" {
		t.Errorf("trigger = %q, want manual", evt.TriggerType)
	}
}

// [REQ:RECOVER-007] Manual recovery respects circuit breaker
func TestManualRecoveryRespectsCircuit(t *testing.T) {
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
			return nil, nil
		}),
	)
	engine.BackoffSchedule = []time.Duration{1 * time.Millisecond}
	engine.ReadyPollTimeout = 10 * time.Millisecond
	engine.ReadyPollInterval = 1 * time.Millisecond

	// Trip circuit
	_, _ = engine.Evaluate(context.Background())

	// Manual without force — should be skipped
	evt, err := engine.TriggerManual(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if evt.Outcome != "skipped" {
		t.Errorf("outcome = %q, want skipped", evt.Outcome)
	}

	// Manual with force — should proceed
	evt, err = engine.TriggerManual(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	if evt.Outcome == "skipped" {
		t.Error("expected force to bypass circuit breaker")
	}
}
