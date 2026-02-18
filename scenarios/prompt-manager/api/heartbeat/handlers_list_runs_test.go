package heartbeat

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListRuns_ResolvesProfileKeyToAgentProfileID(t *testing.T) {
	mockClient := newMockAgentClient().
		WithEnsureProfileResponse(&EnsureProfileResponse{
			Profile: &AgentProfile{
				ID:         "6a89415d-9f95-4ee0-a534-00619ce6fd5a",
				ProfileKey: "prompt-manager-heartbeat",
			},
		}).
		WithListRunsResponse(&ListRunsResponse{
			Runs: []*Run{{ID: "run-1", Status: "RUN_STATUS_COMPLETE"}},
			Total: 1,
		})

	handlers := NewHandlers(nil, nil, nil, nil, nil, nil, mockClient, nil)

	req := httptest.NewRequest(http.MethodGet, "/runs?profile_key=prompt-manager-heartbeat&limit=5", nil)
	w := httptest.NewRecorder()
	handlers.ListRuns(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	mockClient.mu.Lock()
	defer mockClient.mu.Unlock()

	if len(mockClient.ensureProfileCalls) != 1 {
		t.Fatalf("expected 1 EnsureProfile call, got %d", len(mockClient.ensureProfileCalls))
	}
	if got := mockClient.ensureProfileCalls[0]; got == nil || got.ProfileKey != "prompt-manager-heartbeat" {
		t.Fatalf("expected EnsureProfile(profile_key=prompt-manager-heartbeat), got %+v", got)
	}
	if len(mockClient.listRunsCalls) != 1 {
		t.Fatalf("expected 1 ListRuns call, got %d", len(mockClient.listRunsCalls))
	}
	if mockClient.listRunsCalls[0].AgentProfileID != "6a89415d-9f95-4ee0-a534-00619ce6fd5a" {
		t.Fatalf("expected AgentProfileID to be resolved UUID, got %q", mockClient.listRunsCalls[0].AgentProfileID)
	}
}

func TestListRuns_ProfileKeyResolutionFailureReturnsBadGateway(t *testing.T) {
	mockClient := newMockAgentClient().WithEnsureProfileResponse(&EnsureProfileResponse{})
	handlers := NewHandlers(nil, nil, nil, nil, nil, nil, mockClient, nil)

	req := httptest.NewRequest(http.MethodGet, "/runs?profile_key=prompt-manager-heartbeat", nil)
	w := httptest.NewRecorder()
	handlers.ListRuns(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", w.Code, w.Body.String())
	}
}
