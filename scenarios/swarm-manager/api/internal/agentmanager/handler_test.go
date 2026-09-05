package agentmanager

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

type handlerStubService struct {
	enabled  bool
	state    RunState
	stateErr error
	stopErr  error
	stopped  string
}

func (s *handlerStubService) IsEnabled() bool { return s.enabled }

func (s *handlerStubService) IsAvailable(_ context.Context) bool { return true }

func (s *handlerStubService) ResolveURL(_ context.Context) (string, error) {
	return "http://agent-manager", nil
}

func (s *handlerStubService) GetProfileID() string { return "" }

func (s *handlerStubService) ApproveRun(_ context.Context, _ string, _ string, _ string) error {
	return nil
}

func (s *handlerStubService) ContinueRun(_ context.Context, _ string, _ string) error {
	return nil
}

func (s *handlerStubService) GetRunState(_ context.Context, runID string) (RunState, error) {
	if s.stateErr != nil {
		return RunState{}, s.stateErr
	}
	state := s.state
	if state.RunID == "" {
		state.RunID = runID
	}
	return state, nil
}

func (s *handlerStubService) GetRunDiff(_ context.Context, runID string) (RunDiff, error) {
	return RunDiff{RunID: runID}, nil
}

func (s *handlerStubService) StopRun(_ context.Context, runID string) error {
	s.stopped = runID
	return s.stopErr
}

func TestHandler_GetRun_OK(t *testing.T) {
	h := NewHandler(&handlerStubService{
		enabled: true,
		state: RunState{
			RunID:      "run-1",
			TaskID:     "task-1",
			Status:     "running",
			StartedAt:  "2026-02-19T00:00:00Z",
			FinishedAt: "",
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent-manager/runs/run-1", nil)
	req = mux.SetURLVars(req, map[string]string{"runID": "run-1"})
	w := httptest.NewRecorder()

	h.GetRun(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	body := w.Body.String()
	if body == "" || !strings.Contains(body, "\"run_id\":\"run-1\"") || !strings.Contains(body, "\"status\":\"running\"") {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestHandler_GetRun_NotFound(t *testing.T) {
	h := NewHandler(&handlerStubService{
		enabled:  true,
		stateErr: errors.New("agent-manager request failed: status 404"),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent-manager/runs/run-missing", nil)
	req = mux.SetURLVars(req, map[string]string{"runID": "run-missing"})
	w := httptest.NewRecorder()

	h.GetRun(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestHandler_StopRun_OK(t *testing.T) {
	stub := &handlerStubService{enabled: true}
	h := NewHandler(stub)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent-manager/runs/run-1/stop", nil)
	req = mux.SetURLVars(req, map[string]string{"runID": "run-1"})
	w := httptest.NewRecorder()

	h.StopRun(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if stub.stopped != "run-1" {
		t.Fatalf("expected stop to be requested for run-1, got %q", stub.stopped)
	}
}

func TestHandler_StopRun_NotEnabled(t *testing.T) {
	h := NewHandler(&handlerStubService{enabled: false})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent-manager/runs/run-1/stop", nil)
	req = mux.SetURLVars(req, map[string]string{"runID": "run-1"})
	w := httptest.NewRecorder()

	h.StopRun(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", w.Code)
	}
}
