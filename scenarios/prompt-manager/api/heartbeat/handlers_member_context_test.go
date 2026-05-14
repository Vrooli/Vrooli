package heartbeat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"prompt-manager/store"
	"testing"

	"github.com/gorilla/mux"
)

func TestGetMemberContext_Success(t *testing.T) {
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()

	ctx := context.Background()
	team := newIndependentTestTeam("team-1", "Team")
	if err := teamStore.Create(ctx, team); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := agentStore.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "Agent"}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := relationStore.SetTeamMember(ctx, &store.TeamMemberRelation{
		TeamID: "team-1", AgentID: "agent-1", Status: store.MemberStatusActive,
	}); err != nil {
		t.Fatalf("create membership: %v", err)
	}
	if err := teamStore.SetResponsibilities(ctx, "team-1", "agent-1", "Do important work"); err != nil {
		t.Fatalf("set responsibilities: %v", err)
	}

	executor := NewExecutor(teamStore, agentStore, nil, "", nil, nil)
	handlers := NewHandlers(teamStore, agentStore, relationStore, nil, executor, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/members/agent-1/context", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.GetMemberContext(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp MemberContextResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.TeamID != "team-1" {
		t.Errorf("expected teamId 'team-1', got %q", resp.TeamID)
	}
	if resp.AgentID != "agent-1" {
		t.Errorf("expected agentId 'agent-1', got %q", resp.AgentID)
	}
	if resp.Prompt == "" {
		t.Error("expected non-empty prompt")
	}
}

func TestGetMemberContext_TeamNotFound(t *testing.T) {
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()

	executor := NewExecutor(teamStore, agentStore, nil, "", nil, nil)
	handlers := NewHandlers(teamStore, agentStore, relationStore, nil, executor, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/teams/nonexistent/members/agent-1/context", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.GetMemberContext(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetMemberContext_MemberNotFound(t *testing.T) {
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()

	ctx := context.Background()
	team := newIndependentTestTeam("team-1", "Team")
	if err := teamStore.Create(ctx, team); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := agentStore.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "Agent"}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	// Note: not adding agent-1 as a member

	executor := NewExecutor(teamStore, agentStore, nil, "", nil, nil)
	handlers := NewHandlers(teamStore, agentStore, relationStore, nil, executor, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/members/agent-1/context", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.GetMemberContext(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetTeamExecutionStatus_Success(t *testing.T) {
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()

	ctx := context.Background()
	team := newIndependentTestTeam("team-1", "Team")
	if err := teamStore.Create(ctx, team); err != nil {
		t.Fatalf("create team: %v", err)
	}

	exec := &captureExecutor{}
	teamExecStore := NewTeamExecutionStore(teamStore, exec, storeDir, nil)
	handlers := NewHandlers(teamStore, agentStore, relationStore, nil, nil, nil, nil, teamExecStore)

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/execution-status", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.GetTeamExecutionStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp TeamExecutionStatus
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.State != "idle" {
		t.Errorf("expected state 'idle', got %q", resp.State)
	}
	if resp.TeamID != "team-1" {
		t.Errorf("expected teamId 'team-1', got %q", resp.TeamID)
	}
}

func TestGetTeamExecutionStatus_TeamNotFound(t *testing.T) {
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()

	handlers := NewHandlers(teamStore, agentStore, relationStore, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/teams/nonexistent/execution-status", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	w := httptest.NewRecorder()

	handlers.GetTeamExecutionStatus(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}
