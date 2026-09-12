package heartbeat

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"prompt-manager/internal/testutil/httpx"

	"github.com/vrooli/api-core/apihttptest"
)

func TestCreateInvestigationRunRejectsRetiredLegacyMode(t *testing.T) {
	mockClient := newMockAgentClient()
	handlers := NewHandlers(HandlersDeps{
		AgentClient: mockClient,
	})

	req := httpx.Request(t, http.MethodPost, "/runs/investigate", strings.NewReader(`{"run_ids":["run-1"],"depth":"standard"}`), nil)
	w := httpx.Recorder()

	handlers.CreateInvestigationRun(w, req)

	apihttptest.AssertStatus(t, w.Result(), http.StatusGone)
}

func TestCreateInvestigationRunTypedModeUsesFiniteLifecycle(t *testing.T) {
	mockClient := newMockAgentClient()
	mockClient.createTypedInvestigationResp = []byte(`{"investigation":{"investigationId":"typed-1","operationStatus":"queued"},"reused":false}`)
	handlers := NewHandlers(HandlersDeps{AgentClient: mockClient})

	req := httpx.Request(t, http.MethodPost, "/runs/investigate", strings.NewReader(`{"run_ids":["run-b","run-a","run-a"],"depth":"deep","custom_context":"diagnose","typed":true,"request_key":"ui/request-1"}`), nil)
	w := httpx.Recorder()
	handlers.CreateInvestigationRun(w, req)

	apihttptest.AssertStatus(t, w.Result(), http.StatusOK)
	if got := w.Body.String(); got != string(mockClient.createTypedInvestigationResp) {
		t.Fatalf("typed response = %s, want %s", got, mockClient.createTypedInvestigationResp)
	}
}

func TestListRunsTypedInvestigationsReturnsOwnerRecords(t *testing.T) {
	mockClient := newMockAgentClient()
	mockClient.listTypedInvestigationsResp = []byte(`{"investigations":[{"investigationId":"typed-1","operationStatus":"completed"}]}`)
	handlers := NewHandlers(HandlersDeps{AgentClient: mockClient})

	req := httpx.Request(t, http.MethodGet, "/runs?typed_investigations=true&limit=20", nil, nil)
	w := httpx.Recorder()
	handlers.ListRuns(w, req)

	apihttptest.AssertStatus(t, w.Result(), http.StatusOK)
	if got := w.Body.String(); got != string(mockClient.listTypedInvestigationsResp) {
		t.Fatalf("typed list response = %s, want %s", got, mockClient.listTypedInvestigationsResp)
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

	apihttptest.AssertStatus(t, w.Result(), http.StatusBadGateway)
	if got := w.Header().Get("X-Vrooli-Error-Hop"); got != "prompt-manager-api->agent-manager" {
		t.Fatalf("expected hop header prompt-manager-api->agent-manager, got %q", got)
	}
}
