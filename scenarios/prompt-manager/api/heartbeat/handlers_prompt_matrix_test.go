package heartbeat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"prompt-manager/internal/paths"
	"prompt-manager/store"

	"github.com/gorilla/mux"
)

func TestPreviewPromptMatrixHandler(t *testing.T) {
	ctx := context.Background()
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	relationStore := fileStore.Relations()

	// Create a team
	team := newIndependentTestTeam("team-1", "Test Team")
	if err := teamStore.Create(ctx, team); err != nil {
		t.Fatalf("create team: %v", err)
	}

	// Create two agents
	agent1 := &store.Agent{ID: "agent-1", DisplayName: "Agent One", Status: store.AgentStatusActive, FileOrder: []string{"SOUL.md"}}
	agent2 := &store.Agent{ID: "agent-2", DisplayName: "Agent Two", Status: store.AgentStatusActive, FileOrder: []string{"SOUL.md"}}
	if err := agentStore.Create(ctx, agent1); err != nil {
		t.Fatalf("create agent1: %v", err)
	}
	if err := agentStore.Create(ctx, agent2); err != nil {
		t.Fatalf("create agent2: %v", err)
	}

	// Add agents as team members
	if err := relationStore.SetTeamMember(ctx, &store.TeamMemberRelation{
		TeamID: team.ID, AgentID: agent1.ID, Status: "active",
	}); err != nil {
		t.Fatalf("add member 1: %v", err)
	}
	if err := relationStore.SetTeamMember(ctx, &store.TeamMemberRelation{
		TeamID: team.ID, AgentID: agent2.ID, Status: "active",
	}); err != nil {
		t.Fatalf("add member 2: %v", err)
	}

	executor := newTestExecutor(t, teamStore, agentStore, nil, "", nil, nil)
	handlers := NewHandlers(HandlersDeps{
		TeamStore:     teamStore,
		AgentStore:    agentStore,
		RelationStore: relationStore,
		Executor:      executor,
	})

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/prompt-matrix", nil)
	req = mux.SetURLVars(req, map[string]string{"id": team.ID})
	w := httptest.NewRecorder()

	handlers.PreviewPromptMatrix(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp TeamPromptMatrixResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.TeamID != team.ID {
		t.Fatalf("expected teamId %q, got %q", team.ID, resp.TeamID)
	}
	if len(resp.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(resp.Entries))
	}

	// Verify entries have display names and sections (sections may be empty but no error)
	for _, entry := range resp.Entries {
		if entry.DisplayName == "" {
			t.Fatalf("expected non-empty displayName for agent %s", entry.AgentID)
		}
		if entry.Error != "" {
			t.Fatalf("unexpected error for agent %s: %s", entry.AgentID, entry.Error)
		}
		if !hasPromptSectionKind(entry.Sections, promptSectionKindMemberPolicy) {
			t.Fatalf("expected member policy section for agent %s", entry.AgentID)
		}
	}
}

func hasPromptSectionKind(sections []PromptSection, kind string) bool {
	for _, section := range sections {
		if section.Kind == kind {
			return true
		}
	}
	return false
}

func TestPreviewPromptMatrixNotFound(t *testing.T) {
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	relationStore := fileStore.Relations()

	executor := newTestExecutor(t, teamStore, agentStore, nil, "", nil, nil)
	handlers := NewHandlers(HandlersDeps{
		TeamStore:     teamStore,
		AgentStore:    agentStore,
		RelationStore: relationStore,
		Executor:      executor,
	})

	req := httptest.NewRequest(http.MethodGet, "/teams/nonexistent/prompt-matrix", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	w := httptest.NewRecorder()

	handlers.PreviewPromptMatrix(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPreviewPromptMatrixEmptyTeam(t *testing.T) {
	ctx := context.Background()
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	relationStore := fileStore.Relations()

	// Create a team with no members
	team := newIndependentTestTeam("team-empty", "Empty Team")
	if err := teamStore.Create(ctx, team); err != nil {
		t.Fatalf("create team: %v", err)
	}

	executor := newTestExecutor(t, teamStore, agentStore, nil, "", nil, nil)
	handlers := NewHandlers(HandlersDeps{
		TeamStore:     teamStore,
		AgentStore:    agentStore,
		RelationStore: relationStore,
		Executor:      executor,
	})

	req := httptest.NewRequest(http.MethodGet, "/teams/team-empty/prompt-matrix", nil)
	req = mux.SetURLVars(req, map[string]string{"id": team.ID})
	w := httptest.NewRecorder()

	handlers.PreviewPromptMatrix(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp TeamPromptMatrixResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.TeamID != team.ID {
		t.Fatalf("expected teamId %q, got %q", team.ID, resp.TeamID)
	}
	if len(resp.Entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(resp.Entries))
	}
}
