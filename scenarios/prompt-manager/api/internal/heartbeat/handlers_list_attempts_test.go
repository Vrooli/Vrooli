package heartbeat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"prompt-manager/internal/store"
)

func TestListHeartbeatAttemptsExposesCompletenessMetadata(t *testing.T) {
	test := newTestHandlers(t)
	if err := test.TeamStore.AppendHeartbeatAttempt(context.Background(), "team-page", &store.HeartbeatAttempt{
		ID: "attempt-page", TeamID: "team-page", AgentID: "member-page", Status: store.HeartbeatStatusCompleted,
		Phase: "run_terminal", StartedAt: "2026-10-20T15:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/heartbeat-attempts?team_id=team-page&agent_id=member-page&limit=1", nil)
	response := httptest.NewRecorder()
	test.Handlers.ListHeartbeatAttempts(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var got struct {
		Attempts      []store.HeartbeatAttempt `json:"attempts"`
		Total         int                      `json:"total"`
		TotalExact    bool                     `json:"totalExact"`
		HasMore       bool                     `json:"hasMore"`
		ScannedEvents int                      `json:"scannedEvents"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Attempts) != 1 || got.Attempts[0].ID != "attempt-page" || got.Total != 1 || !got.TotalExact || got.HasMore || got.ScannedEvents != 0 {
		t.Fatalf("response completeness metadata = %+v", got)
	}
}

func TestListHeartbeatAttemptsRejectsUnsupportedEventCursor(t *testing.T) {
	test := newTestHandlers(t)
	request := httptest.NewRequest(http.MethodGet, "/heartbeat-attempts?team_id=team-page&agent_id=member-page&cursor=event-1", nil)
	response := httptest.NewRecorder()
	test.Handlers.ListHeartbeatAttempts(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s, want 400 for unsupported cursor", response.Code, response.Body.String())
	}
}
