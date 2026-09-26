package heartbeat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"prompt-manager/internal/store"

	"github.com/gorilla/mux"
)

func setupHandoffTestHandlers(t *testing.T) (*Handlers, *store.FileTeamStore) {
	t.Helper()
	h := newTestHandlers(t)
	handlers, teamStore := h.Handlers, h.TeamStore

	ctx := context.Background()
	if err := teamStore.Create(ctx, newIndependentTestTeam("team-1", "Test Team")); err != nil {
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

func TestExecutorStoresResolvedRunOutput(t *testing.T) {
	client := newMockAgentClient().WithGetRunResponse("run-resolved", &Run{
		ID: "run-resolved",
		Result: &RunResult{
			FinalOutput: "resolved handoff from the selected candidate",
			Selection:   RunResultSelection{Status: "selected", SelectedCandidateID: "candidate-2"},
		},
	})
	h := newTestHandlers(t, func(cfg *testHandlersConfig) { cfg.agentClient = client })
	ctx := context.Background()
	if err := h.TeamStore.Create(ctx, newIndependentTestTeam("team-resolved", "Resolved")); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := h.TeamStore.EnsureMemberDir(ctx, "team-resolved", "agent-1"); err != nil {
		t.Fatalf("ensure member dir: %v", err)
	}

	h.Executor.extractAndStoreHandoff(ctx, "team-resolved", "agent-1", "run-resolved", time.Now().UTC())
	content, err := h.TeamStore.GetLastHandoff(ctx, "team-resolved", "agent-1")
	if err != nil {
		t.Fatalf("get handoff: %v", err)
	}
	if content != "resolved handoff from the selected candidate" {
		t.Fatalf("handoff=%q, want resolver output", content)
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
	if resp.Entries[0].Content != "Second" {
		t.Errorf("expected newest entry first, got: %s", resp.Entries[0].Content)
	}
	if resp.Entries[1].Content != "First" {
		t.Errorf("expected oldest entry last, got: %s", resp.Entries[1].Content)
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

func TestClearHandoffHistory_All(t *testing.T) {
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

	// DELETE all
	req := httptest.NewRequest(http.MethodDelete, "/teams/team-1/handoff-history", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()
	handlers.ClearHandoffHistory(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	// Verify empty
	req2 := httptest.NewRequest(http.MethodGet, "/teams/team-1/handoff-history", nil)
	req2 = mux.SetURLVars(req2, map[string]string{"id": "team-1"})
	w2 := httptest.NewRecorder()
	handlers.GetHandoffHistory(w2, req2)

	var resp HandoffHistoryResponse
	if err := json.NewDecoder(w2.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Entries) != 0 {
		t.Errorf("expected 0 entries after clear, got %d", len(resp.Entries))
	}
}

func TestClearHandoffHistory_ByAgent(t *testing.T) {
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

	// DELETE only agent-1
	req := httptest.NewRequest(http.MethodDelete, "/teams/team-1/handoff-history?agent=agent-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()
	handlers.ClearHandoffHistory(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	// Verify only agent-2 remains
	req2 := httptest.NewRequest(http.MethodGet, "/teams/team-1/handoff-history", nil)
	req2 = mux.SetURLVars(req2, map[string]string{"id": "team-1"})
	w2 := httptest.NewRecorder()
	handlers.GetHandoffHistory(w2, req2)

	var resp HandoffHistoryResponse
	if err := json.NewDecoder(w2.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Entries) != 1 {
		t.Fatalf("expected 1 entry after agent-filtered clear, got %d", len(resp.Entries))
	}
	if resp.Entries[0].AgentID != "agent-2" {
		t.Errorf("expected remaining entry from agent-2, got: %s", resp.Entries[0].AgentID)
	}
}

func TestClearLastHandoff(t *testing.T) {
	handlers, teamStore := setupHandoffTestHandlers(t)
	ctx := context.Background()

	if err := teamStore.SetLastHandoff(ctx, "team-1", "agent-1", "Some handoff content"); err != nil {
		t.Fatalf("set handoff: %v", err)
	}

	// DELETE
	req := httptest.NewRequest(http.MethodDelete, "/teams/team-1/members/agent-1/handoff", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w := httptest.NewRecorder()
	handlers.ClearLastHandoff(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	// Verify 404
	req2 := httptest.NewRequest(http.MethodGet, "/teams/team-1/members/agent-1/handoff", nil)
	req2 = mux.SetURLVars(req2, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w2 := httptest.NewRecorder()
	handlers.GetLastHandoff(w2, req2)

	if w2.Code != http.StatusNotFound {
		t.Errorf("expected 404 after clearing last handoff, got %d", w2.Code)
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
