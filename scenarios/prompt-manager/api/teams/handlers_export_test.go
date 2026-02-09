package teams

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"prompt-manager/interop"
	"prompt-manager/store"

	"github.com/gorilla/mux"
)

func TestExportClaudeCode_Success(t *testing.T) {
	handlers, teamStore, agentStore, relationStore := setupTestHandlers()

	teamStore.teams["team-1"] = &store.Team{
		ID:          "team-1",
		DisplayName: "Test Team",
		Mission:     "Test mission",
	}
	teamStore.roles["team-1"] = &store.TeamRoles{TeamID: "team-1", Roles: []store.Role{}}
	teamStore.orgChart["team-1"] = &store.OrgChart{TeamID: "team-1", Edges: []store.OrgEdge{}}

	agentStore.agents["agent-1"] = &store.Agent{
		ID:          "agent-1",
		DisplayName: "Lead Agent",
		Status:      store.AgentStatusActive,
	}
	agentStore.agents["agent-2"] = &store.Agent{
		ID:          "agent-2",
		DisplayName: "Worker Agent",
		Status:      store.AgentStatusActive,
	}

	_ = relationStore.SetTeamMember(context.TODO(), &store.TeamMemberRelation{TeamID: "team-1", AgentID: "agent-1", Status: store.MemberStatusActive})
	_ = relationStore.SetTeamMember(context.TODO(), &store.TeamMemberRelation{TeamID: "team-1", AgentID: "agent-2", Status: store.MemberStatusActive})

	req := httptest.NewRequest("GET", "/teams/team-1/export/claude-code", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.ExportClaudeCode(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var config interop.ToolTeamConfig
	if err := json.NewDecoder(w.Body).Decode(&config); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if config.TeamName != "team-1" {
		t.Errorf("expected team name 'team-1', got %q", config.TeamName)
	}

	if len(config.Members) != 2 {
		t.Errorf("expected 2 members, got %d", len(config.Members))
	}
}

func TestExportClaudeCode_TeamNotFound(t *testing.T) {
	handlers, _, _, _ := setupTestHandlers()

	req := httptest.NewRequest("GET", "/teams/nonexistent/export/claude-code", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	w := httptest.NewRecorder()

	handlers.ExportClaudeCode(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestExportClaudeCode_WithDocReader(t *testing.T) {
	handlers, teamStore, agentStore, relationStore := setupTestHandlers()

	teamStore.teams["team-docs"] = &store.Team{
		ID:          "team-docs",
		DisplayName: "Docs Team",
		Mission:     "Documentation",
	}
	teamStore.roles["team-docs"] = &store.TeamRoles{TeamID: "team-docs", Roles: []store.Role{}}
	teamStore.orgChart["team-docs"] = &store.OrgChart{TeamID: "team-docs", Edges: []store.OrgEdge{}}

	agentStore.agents["writer"] = &store.Agent{
		ID:          "writer",
		DisplayName: "Writer Agent",
	}

	_ = relationStore.SetTeamMember(context.TODO(), &store.TeamMemberRelation{TeamID: "team-docs", AgentID: "writer", Status: store.MemberStatusActive})

	// Set up doc reader data on MockTeamStore
	teamStore.responsibilities["team-docs"] = map[string]string{
		"writer": "Write documentation",
	}
	teamStore.heartbeatInstructions["team-docs"] = map[string]string{
		"writer": "Review pending docs",
	}

	req := httptest.NewRequest("GET", "/teams/team-docs/export/claude-code", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-docs"})
	w := httptest.NewRecorder()

	handlers.ExportClaudeCode(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// The export handler should have picked up doc reader and included data.
	// We can't directly check the snapshot contents from the response,
	// but the handler should succeed without error and produce valid output.
	var config interop.ToolTeamConfig
	if err := json.NewDecoder(w.Body).Decode(&config); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(config.Members) != 1 {
		t.Errorf("expected 1 member, got %d", len(config.Members))
	}
	if config.Members[0].Name != "writer" {
		t.Errorf("expected member name 'writer', got %q", config.Members[0].Name)
	}
}

func TestExportClaudeCode_NoMembers(t *testing.T) {
	handlers, teamStore, _, _ := setupTestHandlers()

	teamStore.teams["empty-team"] = &store.Team{
		ID:          "empty-team",
		DisplayName: "Empty Team",
	}
	teamStore.roles["empty-team"] = &store.TeamRoles{TeamID: "empty-team", Roles: []store.Role{}}
	teamStore.orgChart["empty-team"] = &store.OrgChart{TeamID: "empty-team", Edges: []store.OrgEdge{}}

	req := httptest.NewRequest("GET", "/teams/empty-team/export/claude-code", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "empty-team"})
	w := httptest.NewRecorder()

	handlers.ExportClaudeCode(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var config interop.ToolTeamConfig
	if err := json.NewDecoder(w.Body).Decode(&config); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(config.Members) != 0 {
		t.Errorf("expected 0 members, got %d", len(config.Members))
	}
}
