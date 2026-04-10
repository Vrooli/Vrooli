package heartbeat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"prompt-manager/store"
	"prompt-manager/teamconfig"

	"github.com/gorilla/mux"
)

func setupTeamTriggerTestHandlers(t *testing.T) (*Handlers, *store.FileTeamStore, *store.FileAgentStore, store.RelationStore) {
	t.Helper()
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()
	executor := NewExecutor(teamStore, agentStore, nil, "", nil, nil)
	handlers := NewHandlers(teamStore, agentStore, relationStore, nil, executor, nil, nil, nil)
	return handlers, teamStore, agentStore, relationStore
}

func TestTriggerTeam_TeamNotFound(t *testing.T) {
	handlers, _, _, _ := setupTeamTriggerTestHandlers(t)

	req := httptest.NewRequest(http.MethodPost, "/teams/nonexistent/trigger", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	w := httptest.NewRecorder()

	handlers.TriggerTeam(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTriggerTeam_TeamDisabled(t *testing.T) {
	handlers, teamStore, _, _ := setupTeamTriggerTestHandlers(t)

	team := newIndependentTestTeam("team-1", "Disabled Team")
	team.Enabled = false
	if err := teamStore.Create(context.Background(), team); err != nil {
		t.Fatalf("create team: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/trigger", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.TriggerTeam(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTriggerTeam_LeaderLedTargetsExplicitLead(t *testing.T) {
	handlers, teamStore, _, relationStore := setupTeamTriggerTestHandlers(t)
	handlers.teamExecStore = NewTeamExecutionStore(teamStore, &captureExecutor{}, t.TempDir())
	ctx := context.Background()

	if err := teamStore.Create(ctx, newLeaderLedSingleProcessTestTeam("team-sp", "Single Process Team", "lead")); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := relationStore.SetTeamMember(ctx, &store.TeamMemberRelation{
		TeamID: "team-sp", AgentID: "lead", Status: store.MemberStatusActive,
	}); err != nil {
		t.Fatalf("set member: %v", err)
	}
	if err := teamStore.SetHeartbeatConfig(ctx, "team-sp", "lead", &store.HeartbeatConfig{
		TeamID:   "team-sp",
		AgentID:  "lead",
		Schedule: "0 * * * *",
		Enabled:  true,
	}); err != nil {
		t.Fatalf("set config: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/teams/team-sp/trigger", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-sp"})
	w := httptest.NewRecorder()

	handlers.TriggerTeam(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}

	var resp TriggerTeamResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.RuntimeMode != teamconfig.RuntimeModeSingleProcess {
		t.Fatalf("expected runtimeMode %q, got %q", teamconfig.RuntimeModeSingleProcess, resp.RuntimeMode)
	}
	if resp.CoordinationPattern != teamconfig.CoordinationPatternLeaderLed {
		t.Fatalf("expected coordinationPattern %q, got %q", teamconfig.CoordinationPatternLeaderLed, resp.CoordinationPattern)
	}
	if len(resp.Triggers) != 1 || resp.Triggers[0].AgentID != "lead" {
		t.Fatalf("expected one lead trigger, got %+v", resp.Triggers)
	}
}

func TestTriggerTeam_LeaderLedRequiresActiveLeadMembership(t *testing.T) {
	handlers, teamStore, _, relationStore := setupTeamTriggerTestHandlers(t)
	handlers.teamExecStore = NewTeamExecutionStore(teamStore, &captureExecutor{}, t.TempDir())
	ctx := context.Background()

	if err := teamStore.Create(ctx, newLeaderLedSingleProcessTestTeam("team-sp", "Single Process Team", "lead")); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := relationStore.SetTeamMember(ctx, &store.TeamMemberRelation{
		TeamID: "team-sp", AgentID: "lead", Status: store.MemberStatusInactive,
	}); err != nil {
		t.Fatalf("set member: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/teams/team-sp/trigger", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-sp"})
	w := httptest.NewRecorder()

	handlers.TriggerTeam(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTriggerTeam_LeaderLedRequiresLeadHeartbeatConfig(t *testing.T) {
	handlers, teamStore, _, relationStore := setupTeamTriggerTestHandlers(t)
	handlers.teamExecStore = NewTeamExecutionStore(teamStore, &captureExecutor{}, t.TempDir())
	ctx := context.Background()

	if err := teamStore.Create(ctx, newLeaderLedSingleProcessTestTeam("team-sp", "Single Process Team", "lead")); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := relationStore.SetTeamMember(ctx, &store.TeamMemberRelation{
		TeamID: "team-sp", AgentID: "lead", Status: store.MemberStatusActive,
	}); err != nil {
		t.Fatalf("set member: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/teams/team-sp/trigger", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-sp"})
	w := httptest.NewRecorder()

	handlers.TriggerTeam(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTriggerTeam_MultiProcessNoConfigs(t *testing.T) {
	handlers, teamStore, agentStore, relationStore := setupTeamTriggerTestHandlers(t)

	if err := teamStore.Create(context.Background(), newIndependentTestTeam("team-mp", "Multi Process Team")); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := agentStore.Create(context.Background(), &store.Agent{ID: "agent-1", DisplayName: "Agent"}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := relationStore.SetTeamMember(context.Background(), &store.TeamMemberRelation{
		TeamID: "team-mp", AgentID: "agent-1", Status: store.MemberStatusActive,
	}); err != nil {
		t.Fatalf("set member: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/teams/team-mp/trigger", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-mp"})
	w := httptest.NewRecorder()

	handlers.TriggerTeam(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}

	var resp TriggerTeamResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.RuntimeMode != teamconfig.RuntimeModeMultiProcess {
		t.Errorf("expected runtimeMode %q, got %q", teamconfig.RuntimeModeMultiProcess, resp.RuntimeMode)
	}
	if resp.CoordinationPattern != teamconfig.CoordinationPatternIndependent {
		t.Errorf("expected coordinationPattern %q, got %q", teamconfig.CoordinationPatternIndependent, resp.CoordinationPattern)
	}

	// No heartbeat configs means no triggers
	if len(resp.Triggers) != 0 {
		t.Errorf("expected 0 triggers (no configs), got %d", len(resp.Triggers))
	}
}

func TestTriggerTeam_ExecutorNotConfigured(t *testing.T) {
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()
	// No executor
	handlers := NewHandlers(teamStore, agentStore, relationStore, nil, nil, nil, nil, nil)

	if err := teamStore.Create(context.Background(), newIndependentTestTeam("team-1", "Team")); err != nil {
		t.Fatalf("create team: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/trigger", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.TriggerTeam(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTriggerTeam_MemberAlreadyQueued(t *testing.T) {
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()
	ctx := context.Background()

	if err := teamStore.Create(ctx, newLeaderLedSingleProcessTestTeam("team-q", "Queue Team", "lead")); err != nil {
		t.Fatalf("create team: %v", err)
	}
	for _, id := range []string{"lead", "dev-1"} {
		if err := agentStore.Create(ctx, &store.Agent{ID: id, DisplayName: id}); err != nil {
			t.Fatalf("create agent %s: %v", id, err)
		}
		if err := relationStore.SetTeamMember(ctx, &store.TeamMemberRelation{
			TeamID: "team-q", AgentID: id, Status: store.MemberStatusActive,
		}); err != nil {
			t.Fatalf("set member %s: %v", id, err)
		}
	}
	if err := teamStore.SetHeartbeatConfig(ctx, "team-q", "lead", &store.HeartbeatConfig{
		TeamID:   "team-q",
		AgentID:  "lead",
		Schedule: "0 * * * *",
		Enabled:  true,
	}); err != nil {
		t.Fatalf("set lead config: %v", err)
	}
	exec := &captureExecutor{}
	teamExecStore := NewTeamExecutionStore(teamStore, exec, t.TempDir())
	executor := NewExecutor(teamStore, agentStore, nil, "", nil, nil)
	handlers := NewHandlers(teamStore, agentStore, relationStore, nil, executor, nil, nil, teamExecStore)

	// First team trigger should succeed
	req := httptest.NewRequest(http.MethodPost, "/teams/team-q/trigger", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-q"})
	w := httptest.NewRecorder()
	handlers.TriggerTeam(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("first trigger: expected 202, got %d: %s", w.Code, w.Body.String())
	}

	// Second trigger should return 409 (lead already queued/running)
	req2 := httptest.NewRequest(http.MethodPost, "/teams/team-q/trigger", nil)
	req2 = mux.SetURLVars(req2, map[string]string{"id": "team-q"})
	w2 := httptest.NewRecorder()
	handlers.TriggerTeam(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Fatalf("second trigger: expected 409, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestTriggerTeam_IndependentTriggersConfiguredMembers(t *testing.T) {
	handlers, teamStore, agentStore, relationStore := setupTeamTriggerTestHandlers(t)
	ctx := context.Background()
	handlers.teamExecStore = NewTeamExecutionStore(teamStore, &captureExecutor{}, t.TempDir())

	if err := teamStore.Create(ctx, newIndependentTestTeam("team-independent", "Independent Team")); err != nil {
		t.Fatalf("create team: %v", err)
	}
	for _, id := range []string{"agent-1", "agent-2"} {
		if err := agentStore.Create(ctx, &store.Agent{ID: id, DisplayName: id}); err != nil {
			t.Fatalf("create agent %s: %v", id, err)
		}
		if err := relationStore.SetTeamMember(ctx, &store.TeamMemberRelation{
			TeamID: "team-independent", AgentID: id, Status: store.MemberStatusActive,
		}); err != nil {
			t.Fatalf("set member %s: %v", id, err)
		}
	}
	if err := teamStore.SetHeartbeatConfig(ctx, "team-independent", "agent-1", &store.HeartbeatConfig{
		TeamID:   "team-independent",
		AgentID:  "agent-1",
		Schedule: "0 * * * *",
		Enabled:  true,
	}); err != nil {
		t.Fatalf("set config: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/teams/team-independent/trigger", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-independent"})
	w := httptest.NewRecorder()

	handlers.TriggerTeam(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}

	var resp TriggerTeamResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Triggers) != 1 || resp.Triggers[0].AgentID != "agent-1" {
		t.Fatalf("expected only configured agent to trigger, got %+v", resp.Triggers)
	}
}
