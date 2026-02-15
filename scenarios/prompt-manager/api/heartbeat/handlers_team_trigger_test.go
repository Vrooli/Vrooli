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

func setupTeamTriggerTestHandlers(t *testing.T) (*Handlers, *store.FileTeamStore, *store.FileAgentStore, store.RelationStore) {
	t.Helper()
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()
	executor := NewExecutor(teamStore, agentStore, nil, "", nil)
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

	if err := teamStore.Create(context.Background(), &store.Team{
		ID:          "team-1",
		DisplayName: "Disabled Team",
		Enabled:     false,
	}); err != nil {
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

func TestTriggerTeam_SingleProcessNoLead(t *testing.T) {
	handlers, teamStore, _, _ := setupTeamTriggerTestHandlers(t)

	if err := teamStore.Create(context.Background(), &store.Team{
		ID:          "team-sp",
		DisplayName: "Single Process Team",
		Enabled:     true,
		SpawnMode:   "single-process",
	}); err != nil {
		t.Fatalf("create team: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/teams/team-sp/trigger", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-sp"})
	w := httptest.NewRecorder()

	handlers.TriggerTeam(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for no team lead, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTriggerTeam_MultiProcessNoConfigs(t *testing.T) {
	handlers, teamStore, agentStore, relationStore := setupTeamTriggerTestHandlers(t)

	if err := teamStore.Create(context.Background(), &store.Team{
		ID:          "team-mp",
		DisplayName: "Multi Process Team",
		Enabled:     true,
	}); err != nil {
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

	if resp.SpawnMode != "multi-process" {
		t.Errorf("expected spawnMode 'multi-process', got %q", resp.SpawnMode)
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

	if err := teamStore.Create(context.Background(), &store.Team{
		ID: "team-1", DisplayName: "Team", Enabled: true,
	}); err != nil {
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

	if err := teamStore.Create(ctx, &store.Team{
		ID:          "team-q",
		DisplayName: "Queue Team",
		Enabled:     true,
		SpawnMode:   "single-process",
	}); err != nil {
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
	if err := teamStore.SetOrgChart(ctx, "team-q", &store.OrgChart{
		TeamID: "team-q",
		Edges:  []store.OrgEdge{{ManagerAgentID: "lead", ReportAgentID: "dev-1"}},
	}); err != nil {
		t.Fatalf("set org chart: %v", err)
	}

	exec := &captureExecutor{}
	teamExecStore := NewTeamExecutionStore(exec, t.TempDir())
	executor := NewExecutor(teamStore, agentStore, nil, "", nil)
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

// ============== findTeamLead Tests ==============

func TestFindTeamLead_OrgChart(t *testing.T) {
	handlers, teamStore, agentStore, relationStore := setupTeamTriggerTestHandlers(t)
	ctx := context.Background()

	if err := teamStore.Create(ctx, &store.Team{ID: "team-org", DisplayName: "Org Team"}); err != nil {
		t.Fatalf("create team: %v", err)
	}
	for _, id := range []string{"lead", "dev-1", "dev-2"} {
		if err := agentStore.Create(ctx, &store.Agent{ID: id, DisplayName: id}); err != nil {
			t.Fatalf("create agent %s: %v", id, err)
		}
		if err := relationStore.SetTeamMember(ctx, &store.TeamMemberRelation{
			TeamID: "team-org", AgentID: id, Status: store.MemberStatusActive,
		}); err != nil {
			t.Fatalf("set member %s: %v", id, err)
		}
	}
	// lead -> dev-1, lead -> dev-2
	if err := teamStore.SetOrgChart(ctx, "team-org", &store.OrgChart{
		TeamID: "team-org",
		Edges: []store.OrgEdge{
			{ManagerAgentID: "lead", ReportAgentID: "dev-1"},
			{ManagerAgentID: "lead", ReportAgentID: "dev-2"},
		},
	}); err != nil {
		t.Fatalf("set org chart: %v", err)
	}

	result := handlers.findTeamLead(ctx, "team-org")
	if result != "lead" {
		t.Errorf("expected 'lead', got %q", result)
	}
}

func TestFindTeamLead_FallbackToFirstMember(t *testing.T) {
	handlers, teamStore, agentStore, relationStore := setupTeamTriggerTestHandlers(t)
	ctx := context.Background()

	if err := teamStore.Create(ctx, &store.Team{ID: "team-flat", DisplayName: "Flat Team"}); err != nil {
		t.Fatalf("create team: %v", err)
	}
	// No org chart, just members
	for _, id := range []string{"alice", "bob"} {
		if err := agentStore.Create(ctx, &store.Agent{ID: id, DisplayName: id}); err != nil {
			t.Fatalf("create agent %s: %v", id, err)
		}
		if err := relationStore.SetTeamMember(ctx, &store.TeamMemberRelation{
			TeamID: "team-flat", AgentID: id, Status: store.MemberStatusActive,
		}); err != nil {
			t.Fatalf("set member %s: %v", id, err)
		}
	}

	result := handlers.findTeamLead(ctx, "team-flat")
	// Should get the first member (order may vary, but should not be empty)
	if result == "" {
		t.Error("expected non-empty team lead, got empty string")
	}
}

func TestFindTeamLead_NoMembers(t *testing.T) {
	handlers, teamStore, _, _ := setupTeamTriggerTestHandlers(t)
	ctx := context.Background()

	if err := teamStore.Create(ctx, &store.Team{ID: "team-empty", DisplayName: "Empty"}); err != nil {
		t.Fatalf("create team: %v", err)
	}

	result := handlers.findTeamLead(ctx, "team-empty")
	if result != "" {
		t.Errorf("expected empty string for no members, got %q", result)
	}
}
