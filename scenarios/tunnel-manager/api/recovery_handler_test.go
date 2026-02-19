package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// [REQ:RECOVER-006] Recovery handler API tests

func TestHandlerRecoveryState(t *testing.T) {
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
	engine := NewRecoveryEngine(db, checker)

	handler := handleRecoveryState(engine)
	req := httptest.NewRequest("GET", "/api/v1/recovery/state", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var state RecoveryState
	if err := json.Unmarshal(w.Body.Bytes(), &state); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if state.ConsecFailures != 0 {
		t.Errorf("expected 0 consecutive failures initially, got %d", state.ConsecFailures)
	}
}

func TestHandlerRecoveryEvents_EmptyList(t *testing.T) {
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
	engine := NewRecoveryEngine(db, checker)

	handler := handleRecoveryEvents(engine)
	req := httptest.NewRequest("GET", "/api/v1/recovery/events", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var events []RecoveryEvent
	if err := json.Unmarshal(w.Body.Bytes(), &events); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected empty events list, got %d", len(events))
	}
}

func TestHandlerRecoveryEvents_WithEvents(t *testing.T) {
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

	// Trigger a recovery event
	_, err := engine.Evaluate(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	handler := handleRecoveryEvents(engine)
	req := httptest.NewRequest("GET", "/api/v1/recovery/events", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var events []RecoveryEvent
	if err := json.Unmarshal(w.Body.Bytes(), &events); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(events) == 0 {
		t.Error("expected at least one event")
	}
}

func TestHandlerCircuitReset(t *testing.T) {
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

	// Trip the circuit
	_, _ = engine.Evaluate(context.Background())

	state := engine.State()
	if !state.CircuitOpen {
		t.Fatal("circuit should be open after exhausting retries")
	}

	// Reset via handler
	handler := handleCircuitReset(engine)
	req := httptest.NewRequest("POST", "/api/v1/recovery/circuit/reset", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	state = engine.State()
	if state.CircuitOpen {
		t.Error("circuit should be closed after reset")
	}
}

// [REQ:RECOVER-007] Recovery trigger handler tests

func TestHandlerRecoveryTrigger_NoForce(t *testing.T) {
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
	engine := NewRecoveryEngine(db, checker,
		WithRecoveryCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, nil
		}),
	)

	handler := handleRecoveryTrigger(engine)
	req := httptest.NewRequest("POST", "/api/v1/recovery/trigger", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var evt RecoveryEvent
	if err := json.Unmarshal(w.Body.Bytes(), &evt); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if evt.TriggerType != "manual" {
		t.Errorf("trigger_type = %q, want manual", evt.TriggerType)
	}
}

func TestHandlerRecoveryTrigger_WithForce(t *testing.T) {
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
	engine := NewRecoveryEngine(db, checker,
		WithRecoveryCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, nil
		}),
	)

	handler := handleRecoveryTrigger(engine)
	body := `{"force":true}`
	req := httptest.NewRequest("POST", "/api/v1/recovery/trigger", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var evt RecoveryEvent
	if err := json.Unmarshal(w.Body.Bytes(), &evt); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if evt.TriggerType != "manual" {
		t.Errorf("trigger_type = %q, want manual", evt.TriggerType)
	}
	if evt.Outcome != "success" {
		t.Errorf("outcome = %q, want success", evt.Outcome)
	}
}
