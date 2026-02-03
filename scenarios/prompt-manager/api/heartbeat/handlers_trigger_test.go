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

	executor := NewExecutor(teamStore, agentStore, nil, "")
	handlers := NewHandlers(teamStore, relationStore, nil, executor)

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

	executor := NewExecutor(teamStore, agentStore, nil, "")
	handlers := NewHandlers(teamStore, relationStore, nil, executor)

	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/heartbeats/agent-1/trigger", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.TriggerHeartbeat(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
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

	executor := NewExecutor(teamStore, agentStore, nil, "")
	handlers := NewHandlers(teamStore, relationStore, nil, executor)

	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/heartbeats/agent-1/trigger", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.TriggerHeartbeat(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d: %s", w.Code, w.Body.String())
	}
}
