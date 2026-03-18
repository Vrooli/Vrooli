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

func setupHandoffTestHandlers(t *testing.T) (*Handlers, *store.FileTeamStore) {
	t.Helper()
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()
	executor := NewExecutor(teamStore, agentStore, nil, "", nil, nil)
	handlers := NewHandlers(teamStore, agentStore, relationStore, nil, executor, nil, nil, nil)

	ctx := context.Background()
	if err := teamStore.Create(ctx, &store.Team{
		ID: "team-1", DisplayName: "Test Team", Enabled: true,
	}); err != nil {
		t.Fatalf("create team: %v", err)
	}
	// Ensure member directory exists for handoff storage
	if err := teamStore.EnsureMemberDir(ctx, "team-1", "agent-1"); err != nil {
		t.Fatalf("ensure member dir: %v", err)
	}
	return handlers, teamStore
}

func TestGetLastHandoff_Found(t *testing.T) {
	handlers, teamStore := setupHandoffTestHandlers(t)
	ctx := context.Background()

	if err := teamStore.SetLastHandoff(ctx, "team-1", "agent-1", "**Status**: Done\n\n**Completed**: Fixed auth"); err != nil {
		t.Fatalf("set handoff: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/members/agent-1/handoff", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.GetLastHandoff(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp HandoffResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Content == "" {
		t.Error("expected non-empty content")
	}
	if resp.TeamID != "team-1" {
		t.Errorf("expected teamId 'team-1', got: %s", resp.TeamID)
	}
	if resp.AgentID != "agent-1" {
		t.Errorf("expected agentId 'agent-1', got: %s", resp.AgentID)
	}
}

func TestGetLastHandoff_NotFound(t *testing.T) {
	handlers, _ := setupHandoffTestHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/members/agent-1/handoff", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.GetLastHandoff(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetHandoffHistory_All(t *testing.T) {
	handlers, teamStore := setupHandoffTestHandlers(t)
	ctx := context.Background()

	entries := []store.HandoffEntry{
		{AgentID: "agent-1", RunID: "run-1", Timestamp: "2025-01-01T00:00:00Z", Content: "First"},
		{AgentID: "agent-2", RunID: "run-2", Timestamp: "2025-01-01T01:00:00Z", Content: "Second"},
	}
	for i := range entries {
		if err := teamStore.AppendHandoffHistory(ctx, "team-1", &entries[i]); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/handoff-history", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.GetHandoffHistory(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp HandoffHistoryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(resp.Entries))
	}
}

func TestGetHandoffHistory_FilterAgent(t *testing.T) {
	handlers, teamStore := setupHandoffTestHandlers(t)
	ctx := context.Background()

	entries := []store.HandoffEntry{
		{AgentID: "agent-1", RunID: "run-1", Timestamp: "2025-01-01T00:00:00Z", Content: "First"},
		{AgentID: "agent-2", RunID: "run-2", Timestamp: "2025-01-01T01:00:00Z", Content: "Second"},
		{AgentID: "agent-1", RunID: "run-3", Timestamp: "2025-01-01T02:00:00Z", Content: "Third"},
	}
	for i := range entries {
		if err := teamStore.AppendHandoffHistory(ctx, "team-1", &entries[i]); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/handoff-history?agent=agent-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.GetHandoffHistory(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp HandoffHistoryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Entries) != 2 {
		t.Fatalf("expected 2 entries for agent-1, got %d", len(resp.Entries))
	}
}

func TestGetHandoffHistory_Limit(t *testing.T) {
	handlers, teamStore := setupHandoffTestHandlers(t)
	ctx := context.Background()

	entries := []store.HandoffEntry{
		{AgentID: "agent-1", RunID: "run-1", Timestamp: "2025-01-01T00:00:00Z", Content: "First"},
		{AgentID: "agent-1", RunID: "run-2", Timestamp: "2025-01-01T01:00:00Z", Content: "Second"},
		{AgentID: "agent-1", RunID: "run-3", Timestamp: "2025-01-01T02:00:00Z", Content: "Third"},
	}
	for i := range entries {
		if err := teamStore.AppendHandoffHistory(ctx, "team-1", &entries[i]); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/handoff-history?last=2", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.GetHandoffHistory(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp HandoffHistoryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Entries) != 2 {
		t.Fatalf("expected 2 entries with last=2, got %d", len(resp.Entries))
	}
}
