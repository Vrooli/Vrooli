package heartbeat

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"prompt-manager/store"

	"github.com/gorilla/mux"
)

func TestTriggerHeartbeatRequiresConfig(t *testing.T) {
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()

	if err := teamStore.Create(context.Background(), &store.Team{ID: "team-1", DisplayName: "Team"}); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := agentStore.Create(context.Background(), &store.Agent{ID: "agent-1", DisplayName: "Agent"}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := relationStore.SetTeamMember(context.Background(), &store.TeamMemberRelation{
		TeamID:  "team-1",
		AgentID: "agent-1",
		Status:  store.MemberStatusActive,
	}); err != nil {
		t.Fatalf("create membership: %v", err)
	}

	executor := NewExecutor(teamStore, agentStore, nil, "", nil)
	handlers := NewHandlers(teamStore, agentStore, relationStore, nil, executor, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/heartbeats/agent-1/trigger", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.TriggerHeartbeat(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTriggerHeartbeatRequiresMembership(t *testing.T) {
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()

	if err := teamStore.Create(context.Background(), &store.Team{ID: "team-1", DisplayName: "Team"}); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := agentStore.Create(context.Background(), &store.Agent{ID: "agent-1", DisplayName: "Agent"}); err != nil {
		t.Fatalf("create agent: %v", err)
	}

	executor := NewExecutor(teamStore, agentStore, nil, "", nil)
	handlers := NewHandlers(teamStore, agentStore, relationStore, nil, executor, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/heartbeats/agent-1/trigger", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.TriggerHeartbeat(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTriggerHeartbeat_MemberAlreadyQueued(t *testing.T) {
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()

	if err := teamStore.Create(context.Background(), &store.Team{
		ID:          "team-1",
		DisplayName: "Team",
		Enabled:     true,
	}); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := agentStore.Create(context.Background(), &store.Agent{ID: "agent-1", DisplayName: "Agent"}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := relationStore.SetTeamMember(context.Background(), &store.TeamMemberRelation{
		TeamID:  "team-1",
		AgentID: "agent-1",
		Status:  store.MemberStatusActive,
	}); err != nil {
		t.Fatalf("create membership: %v", err)
	}

	exec := &captureExecutor{}
	teamExecStore := NewTeamExecutionStore(exec, t.TempDir())
	executor := NewExecutor(teamStore, agentStore, nil, "", nil)
	handlers := NewHandlers(teamStore, agentStore, relationStore, nil, executor, nil, nil, teamExecStore)

	// First trigger should succeed (202 Accepted)
	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/heartbeats/agent-1/trigger", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w := httptest.NewRecorder()
	handlers.TriggerHeartbeat(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("first trigger: expected 202, got %d: %s", w.Code, w.Body.String())
	}

	// Second trigger for same agent should return 409 (already queued/running)
	req2 := httptest.NewRequest(http.MethodPost, "/teams/team-1/heartbeats/agent-1/trigger", nil)
	req2 = mux.SetURLVars(req2, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w2 := httptest.NewRecorder()
	handlers.TriggerHeartbeat(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Fatalf("second trigger: expected 409, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestTriggerHeartbeatBlockedWhenTeamDisabled(t *testing.T) {
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()

	if err := teamStore.Create(context.Background(), &store.Team{
		ID:          "team-1",
		DisplayName: "Team",
		Enabled:     false,
	}); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := agentStore.Create(context.Background(), &store.Agent{ID: "agent-1", DisplayName: "Agent"}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := relationStore.SetTeamMember(context.Background(), &store.TeamMemberRelation{
		TeamID:  "team-1",
		AgentID: "agent-1",
		Status:  store.MemberStatusActive,
	}); err != nil {
		t.Fatalf("create membership: %v", err)
	}

	if err := teamStore.SetHeartbeatConfig(context.Background(), "team-1", "agent-1", &store.HeartbeatConfig{
		TeamID:   "team-1",
		AgentID:  "agent-1",
		Schedule: "0 */6 * * *",
		Enabled:  true,
	}); err != nil {
		t.Fatalf("set heartbeat config: %v", err)
	}

	executor := NewExecutor(teamStore, agentStore, nil, "", nil)
	handlers := NewHandlers(teamStore, agentStore, relationStore, nil, executor, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/heartbeats/agent-1/trigger", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.TriggerHeartbeat(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d: %s", w.Code, w.Body.String())
	}
}
