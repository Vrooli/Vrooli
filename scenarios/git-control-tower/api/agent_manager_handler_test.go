package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleAgentRunList_Empty(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	resp.OK(&AgentRunListResponse{
		Runs:  []AgentRun{},
		Total: 0,
	})

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %s", ct)
	}
}

func TestHandleAgentProfiles_Unavailable(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	resp.ServiceUnavailable("agent-manager is not available")

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rr.Code)
	}
}

func TestHandleAgentRunCreate_Response(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	resp.OK(AgentRunCreateResponse{
		RunID:  "run-001",
		TaskID: "task-001",
	})

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %s", ct)
	}
}

func TestHandleAgentRunDiff_Response(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	resp.OK(AgentRunDiffResponse{
		RunID: "run-001",
		Files: []AgentRunDiffFile{
			{Path: "main.go", ChangeType: "modified", Additions: 10, Deletions: 3},
		},
	})

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestHandleAgentRunEvents_Response(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	resp.OK(AgentRunEventsResponse{
		Events: []AgentRunEvent{
			{ID: "evt-1", RunID: "run-001", Sequence: 1, EventType: "message"},
		},
	})

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}
