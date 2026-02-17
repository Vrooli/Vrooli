package heartbeat

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateInvestigationRunSetsErrorHopHeaderOnUpstreamFailure(t *testing.T) {
	mockClient := newMockAgentClient().WithCreateInvestigationRunError(errors.New("upstream failed"))
	handlers := NewHandlers(nil, nil, nil, nil, nil, nil, mockClient, nil)

	req := httptest.NewRequest(http.MethodPost, "/runs/investigate", strings.NewReader(`{"run_ids":["run-1"],"depth":"standard"}`))
	w := httptest.NewRecorder()

	handlers.CreateInvestigationRun(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("X-Vrooli-Error-Hop"); got != "prompt-manager-api->agent-manager" {
		t.Fatalf("expected hop header prompt-manager-api->agent-manager, got %q", got)
	}
}

func TestCreateInvestigationApplyRunSetsErrorHopHeaderOnUpstreamFailure(t *testing.T) {
	mockClient := newMockAgentClient().WithCreateInvestigationApplyError(errors.New("upstream failed"))
	handlers := NewHandlers(nil, nil, nil, nil, nil, nil, mockClient, nil)

	req := httptest.NewRequest(http.MethodPost, "/runs/investigation-apply", strings.NewReader(`{"investigation_run_id":"run-1"}`))
	w := httptest.NewRecorder()

	handlers.CreateInvestigationApplyRun(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("X-Vrooli-Error-Hop"); got != "prompt-manager-api->agent-manager" {
		t.Fatalf("expected hop header prompt-manager-api->agent-manager, got %q", got)
	}
}
