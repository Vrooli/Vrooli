package heartbeat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"prompt-manager/store"

	"github.com/gorilla/mux"
)

func TestRetryRun_RetriesHeartbeatRunByTag(t *testing.T) {
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()

	ctx := context.Background()
	if err := teamStore.Create(ctx, &store.Team{ID: "team-1", DisplayName: "Team", Enabled: true}); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := agentStore.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "Agent"}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := relationStore.SetTeamMember(ctx, &store.TeamMemberRelation{TeamID: "team-1", AgentID: "agent-1", Status: store.MemberStatusActive}); err != nil {
		t.Fatalf("create membership: %v", err)
	}
	if err := teamStore.SetHeartbeatConfig(ctx, "team-1", "agent-1", &store.HeartbeatConfig{TeamID: "team-1", AgentID: "agent-1", Enabled: true, Schedule: "0 * * * *"}); err != nil {
		t.Fatalf("set heartbeat config: %v", err)
	}

	mockClient := newMockAgentClient().
		WithGetRunResponse("run-failed", &Run{ID: "run-failed", Tag: "heartbeat-team-1-agent-1-2026-02-18T10-00-00Z", Status: "RUN_STATUS_FAILED"}).
		WithCreateTaskResponse(&Task{ID: "task-1", Title: "Heartbeat: team-1/agent-1"}).
		WithCreateRunResponse(&Run{ID: "run-retry-1", Status: "RUN_STATUS_RUNNING"})

	executor := NewExecutor(teamStore, agentStore, mockClient, t.TempDir(), nil, nil)
	handlers := NewHandlers(teamStore, agentStore, relationStore, nil, executor, nil, mockClient, nil)

	req := httptest.NewRequest(http.MethodPost, "/runs/run-failed/retry", nil)
	req = mux.SetURLVars(req, map[string]string{"runId": "run-failed"})
	w := httptest.NewRecorder()

	handlers.RetryRun(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}

	var resp TriggerHeartbeatResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.TeamID != "team-1" || resp.AgentID != "agent-1" {
		t.Fatalf("unexpected target in response: %+v", resp)
	}
	if resp.RunID != "run-retry-1" {
		t.Fatalf("expected retry run id run-retry-1, got %q", resp.RunID)
	}
}

func TestRetryRun_RejectsNonHeartbeatRun(t *testing.T) {
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()

	mockClient := newMockAgentClient().
		WithGetRunResponse("run-1", &Run{ID: "run-1", Tag: "manual-tag", Status: "RUN_STATUS_FAILED"})

	executor := NewExecutor(teamStore, agentStore, mockClient, t.TempDir(), nil, nil)
	handlers := NewHandlers(teamStore, agentStore, relationStore, nil, executor, nil, mockClient, nil)

	req := httptest.NewRequest(http.MethodPost, "/runs/run-1/retry", nil)
	req = mux.SetURLVars(req, map[string]string{"runId": "run-1"})
	w := httptest.NewRecorder()

	handlers.RetryRun(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
