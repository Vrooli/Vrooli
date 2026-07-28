package heartbeat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"prompt-manager/internal/paths"
	"prompt-manager/store"
)

func TestGetAllPendingDecisions_Empty(t *testing.T) {
	handlers, _ := setupDecisionTestHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/decisions/pending", nil)
	w := httptest.NewRecorder()

	handlers.GetAllPendingDecisions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp AllPendingDecisionsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Teams) != 0 {
		t.Errorf("expected 0 teams, got %d", len(resp.Teams))
	}
	if resp.TotalCount != 0 {
		t.Errorf("expected totalCount 0, got %d", resp.TotalCount)
	}
}

func TestGetAllPendingDecisions_SingleTeam(t *testing.T) {
	handlers, teamStore := setupDecisionTestHandlers(t)
	ctx := context.Background()

	// Add 2 pending and 1 accepted decision
	for _, e := range []store.DecisionEntry{
		{ID: "dec-1", At: "2025-01-01T00:00:00Z", By: "agent-1", Decision: "D1", Rationale: "R1", Status: store.DecisionStatusPending},
		{ID: "dec-2", At: "2025-01-01T01:00:00Z", By: "agent-1", Decision: "D2", Rationale: "R2", Status: store.DecisionStatusAccepted},
		{ID: "dec-3", At: "2025-01-01T02:00:00Z", By: "agent-1", Decision: "D3", Rationale: "R3", Status: store.DecisionStatusPending},
	} {
		e := e
		if err := teamStore.AppendDecision(ctx, "team-1", &e); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/decisions/pending", nil)
	w := httptest.NewRecorder()

	handlers.GetAllPendingDecisions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp AllPendingDecisionsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Teams) != 1 {
		t.Fatalf("expected 1 team group, got %d", len(resp.Teams))
	}
	if resp.Teams[0].TeamID != "team-1" {
		t.Errorf("expected team-1, got %s", resp.Teams[0].TeamID)
	}
	if resp.Teams[0].TeamName != "Test Team" {
		t.Errorf("expected 'Test Team', got %s", resp.Teams[0].TeamName)
	}
	if len(resp.Teams[0].Entries) != 2 {
		t.Errorf("expected 2 pending entries, got %d", len(resp.Teams[0].Entries))
	}
	if resp.TotalCount != 2 {
		t.Errorf("expected totalCount 2, got %d", resp.TotalCount)
	}
}

func TestGetAllPendingDecisions_MultipleTeams(t *testing.T) {
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()
	executor := newTestExecutor(t, teamStore, agentStore, nil, "", nil, nil)
	handlers := NewHandlers(teamStore, agentStore, relationStore, nil, executor, nil, nil, nil)
	ctx := context.Background()

	// Create two teams
	if err := teamStore.Create(ctx, newIndependentTestTeam("team-a", "Team A")); err != nil {
		t.Fatalf("create team-a: %v", err)
	}
	if err := teamStore.Create(ctx, newIndependentTestTeam("team-b", "Team B")); err != nil {
		t.Fatalf("create team-b: %v", err)
	}

	// Team A: 1 pending
	entryA := store.DecisionEntry{ID: "dec-a1", At: "2025-01-01T00:00:00Z", By: "agent-1", Decision: "DA1", Rationale: "RA1", Status: store.DecisionStatusPending}
	if err := teamStore.AppendDecision(ctx, "team-a", &entryA); err != nil {
		t.Fatal(err)
	}

	// Team B: 2 pending, 1 completed
	for _, e := range []store.DecisionEntry{
		{ID: "dec-b1", At: "2025-01-01T00:00:00Z", By: "agent-2", Decision: "DB1", Rationale: "RB1", Status: store.DecisionStatusPending},
		{ID: "dec-b2", At: "2025-01-01T01:00:00Z", By: "agent-2", Decision: "DB2", Rationale: "RB2", Status: store.DecisionStatusPending},
		{ID: "dec-b3", At: "2025-01-01T02:00:00Z", By: "agent-2", Decision: "DB3", Rationale: "RB3", Status: store.DecisionStatusCompleted},
	} {
		e := e
		if err := teamStore.AppendDecision(ctx, "team-b", &e); err != nil {
			t.Fatal(err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/decisions/pending", nil)
	w := httptest.NewRecorder()

	handlers.GetAllPendingDecisions(w, req)

	var resp AllPendingDecisionsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Teams) != 2 {
		t.Fatalf("expected 2 team groups, got %d", len(resp.Teams))
	}
	if resp.TotalCount != 3 {
		t.Errorf("expected totalCount 3, got %d", resp.TotalCount)
	}
}

func TestGetAllPendingDecisions_NoPending(t *testing.T) {
	handlers, teamStore := setupDecisionTestHandlers(t)
	ctx := context.Background()

	// Add only accepted decisions
	entry := store.DecisionEntry{ID: "dec-1", At: "2025-01-01T00:00:00Z", By: "agent-1", Decision: "D1", Rationale: "R1", Status: store.DecisionStatusAccepted}
	if err := teamStore.AppendDecision(ctx, "team-1", &entry); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/decisions/pending", nil)
	w := httptest.NewRecorder()

	handlers.GetAllPendingDecisions(w, req)

	var resp AllPendingDecisionsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Teams) != 0 {
		t.Errorf("expected 0 teams, got %d", len(resp.Teams))
	}
	if resp.TotalCount != 0 {
		t.Errorf("expected totalCount 0, got %d", resp.TotalCount)
	}
}
