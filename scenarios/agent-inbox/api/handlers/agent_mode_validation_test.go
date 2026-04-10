package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// =============================================================================
// Validation Tests (no DB required)
// =============================================================================

func TestStartAgentMode_InvalidUUID(t *testing.T) {
	h := &Handlers{}
	req := httptest.NewRequest("POST", "/api/v1/chats/not-a-uuid/agent-mode/start", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "not-a-uuid"})
	w := httptest.NewRecorder()

	h.StartAgentMode(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSendAgentMessage_InvalidUUID(t *testing.T) {
	h := &Handlers{}
	req := httptest.NewRequest("POST", "/api/v1/chats/bad/agent-mode/message", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "bad"})
	w := httptest.NewRecorder()

	h.SendAgentMessage(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetAgentEvents_InvalidUUID(t *testing.T) {
	h := &Handlers{}
	req := httptest.NewRequest("GET", "/api/v1/chats/xyz/agent-mode/events", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "xyz"})
	w := httptest.NewRecorder()

	h.GetAgentEvents(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestStopAgentMode_InvalidUUID(t *testing.T) {
	h := &Handlers{}
	req := httptest.NewRequest("POST", "/api/v1/chats/xyz/agent-mode/stop", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "xyz"})
	w := httptest.NewRecorder()

	h.StopAgentMode(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetAgentStatus_InvalidUUID(t *testing.T) {
	h := &Handlers{}
	req := httptest.NewRequest("GET", "/api/v1/chats/xyz/agent-mode/status", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "xyz"})
	w := httptest.NewRecorder()

	h.GetAgentStatus(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestClearAgentMode_InvalidUUID(t *testing.T) {
	h := &Handlers{}
	req := httptest.NewRequest("POST", "/api/v1/chats/xyz/agent-mode/clear", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "xyz"})
	w := httptest.NewRecorder()

	h.ClearAgentMode(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
