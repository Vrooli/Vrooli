package heartbeat

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"prompt-manager/internal/testutil/httpx"
)

func TestCreateInvestigationRunSetsErrorHopHeaderOnUpstreamFailure(t *testing.T) {
	mockClient := newMockAgentClient().WithCreateInvestigationRunError(errors.New("upstream failed"))
	handlers := NewHandlers(HandlersDeps{
		AgentClient: mockClient,
	})

	req := httpx.Request(t, http.MethodPost, "/runs/investigate", strings.NewReader(`{"run_ids":["run-1"],"depth":"standard"}`), nil)
	w := httpx.Recorder()

	handlers.CreateInvestigationRun(w, req)

	httpx.AssertStatus(t, w, http.StatusBadGateway)
	if got := w.Header().Get("X-Vrooli-Error-Hop"); got != "prompt-manager-api->agent-manager" {
		t.Fatalf("expected hop header prompt-manager-api->agent-manager, got %q", got)
	}
}

func TestCreateInvestigationApplyRunSetsErrorHopHeaderOnUpstreamFailure(t *testing.T) {
	mockClient := newMockAgentClient().WithCreateInvestigationApplyError(errors.New("upstream failed"))
	handlers := NewHandlers(HandlersDeps{
		AgentClient: mockClient,
	})

	req := httpx.Request(t, http.MethodPost, "/runs/investigation-apply", strings.NewReader(`{"investigation_run_id":"run-1"}`), nil)
	w := httpx.Recorder()

	handlers.CreateInvestigationApplyRun(w, req)

	httpx.AssertStatus(t, w, http.StatusBadGateway)
	if got := w.Header().Get("X-Vrooli-Error-Hop"); got != "prompt-manager-api->agent-manager" {
		t.Fatalf("expected hop header prompt-manager-api->agent-manager, got %q", got)
	}
}
