package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tunnel-manager/domain"
)

// [REQ:RECOVER-006] Recovery handler API tests

func TestHandlerRecoveryState(t *testing.T) {
	mgr := &mockRecoveryManager{
		stateFn: func() domain.RecoveryState {
			return domain.RecoveryState{Status: "idle", ConsecFailures: 0}
		},
	}

	h := HandleRecoveryState(mgr)
	req := httptest.NewRequest("GET", "/api/v1/recovery/state", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var state domain.RecoveryState
	if err := json.Unmarshal(w.Body.Bytes(), &state); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if state.ConsecFailures != 0 {
		t.Errorf("expected 0 consecutive failures initially, got %d", state.ConsecFailures)
	}
}

func TestHandlerRecoveryEvents_EmptyList(t *testing.T) {
	mgr := &mockRecoveryManager{
		listEventsFn: func(limit int) ([]domain.RecoveryEvent, error) {
			return []domain.RecoveryEvent{}, nil
		},
	}

	h := HandleRecoveryEvents(mgr)
	req := httptest.NewRequest("GET", "/api/v1/recovery/events", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var events []domain.RecoveryEvent
	if err := json.Unmarshal(w.Body.Bytes(), &events); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected empty events list, got %d", len(events))
	}
}

func TestHandlerRecoveryEvents_WithEvents(t *testing.T) {
	mgr := &mockRecoveryManager{
		listEventsFn: func(limit int) ([]domain.RecoveryEvent, error) {
			return []domain.RecoveryEvent{
				{TriggerType: "ready_failure", Outcome: "success", Action: "systemctl_restart"},
			}, nil
		},
	}

	h := HandleRecoveryEvents(mgr)
	req := httptest.NewRequest("GET", "/api/v1/recovery/events", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var events []domain.RecoveryEvent
	if err := json.Unmarshal(w.Body.Bytes(), &events); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(events) == 0 {
		t.Error("expected at least one event")
	}
}

func TestHandlerCircuitReset(t *testing.T) {
	resetCalled := false
	mgr := &mockRecoveryManager{
		resetCircuitFn: func() { resetCalled = true },
	}

	h := HandleCircuitReset(mgr)
	req := httptest.NewRequest("POST", "/api/v1/recovery/circuit/reset", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !resetCalled {
		t.Error("expected ResetCircuit to be called")
	}
}

// [REQ:RECOVER-007] Recovery trigger handler tests

func TestHandlerRecoveryTrigger_NoForce(t *testing.T) {
	mgr := &mockRecoveryManager{
		triggerFn: func(ctx context.Context, force bool) (*domain.RecoveryEvent, error) {
			return &domain.RecoveryEvent{TriggerType: "manual", Outcome: "success", Action: "systemctl_restart"}, nil
		},
	}

	h := HandleRecoveryTrigger(mgr)
	req := httptest.NewRequest("POST", "/api/v1/recovery/trigger", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var evt domain.RecoveryEvent
	if err := json.Unmarshal(w.Body.Bytes(), &evt); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if evt.TriggerType != "manual" {
		t.Errorf("trigger_type = %q, want manual", evt.TriggerType)
	}
}

func TestHandlerRecoveryTrigger_WithForce(t *testing.T) {
	var gotForce bool
	mgr := &mockRecoveryManager{
		triggerFn: func(ctx context.Context, force bool) (*domain.RecoveryEvent, error) {
			gotForce = force
			return &domain.RecoveryEvent{TriggerType: "manual", Outcome: "success", Action: "systemctl_restart"}, nil
		},
	}

	h := HandleRecoveryTrigger(mgr)
	body := `{"force":true}`
	req := httptest.NewRequest("POST", "/api/v1/recovery/trigger", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var evt domain.RecoveryEvent
	if err := json.Unmarshal(w.Body.Bytes(), &evt); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if evt.TriggerType != "manual" {
		t.Errorf("trigger_type = %q, want manual", evt.TriggerType)
	}
	if evt.Outcome != "success" {
		t.Errorf("outcome = %q, want success", evt.Outcome)
	}
	if !gotForce {
		t.Error("expected force=true to be passed")
	}
}
