package backlog

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"swarm-manager/internal/testutil"
)

func doFeedbackSummary(t *testing.T, h *Handler) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("GET", "/api/v1/backlog/feedback-summary", nil)
	w := httptest.NewRecorder()
	h.FeedbackSummary(w, req)
	return w
}

func TestFeedbackSummary_EmptyBacklog(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := doFeedbackSummary(t, h)
	testutil.AssertStatusOK(t, w)

	var resp FeedbackSummaryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.TotalPending != 0 {
		t.Errorf("expected total_pending 0, got %d", resp.TotalPending)
	}
	if len(resp.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(resp.Items))
	}
}

func TestFeedbackSummary_NoPendingDecisions(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "resolved", Title: "Resolved Item", Status: StatusReady, Priority: 3, Tags: []string{},
	})

	// Write a workshop round where all decisions are resolved.
	selected := "A"
	round := WorkshopRound{
		RoundNum:  1,
		Readiness: map[string]int{"problem_clarity": 2, "scope_defined": 2, "approach_solid": 2, "testable": 2, "risk_awareness": 2},
		Items: []WorkshopItem{
			{ID: "d1", Type: "decision", Selected: &selected},
		},
	}
	workshopDir := filepath.Join(rootDir, "ideas", "resolved", "workshop")
	testutil.WriteJSONFile(t, filepath.Join(workshopDir, "round-001.json"), round)

	w := doFeedbackSummary(t, h)
	testutil.AssertStatusOK(t, w)

	var resp FeedbackSummaryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// No pending decisions → item should not appear.
	if len(resp.Items) != 0 {
		t.Errorf("expected 0 items (no pending decisions), got %d", len(resp.Items))
	}
}

func TestFeedbackSummary_WithPendingDecisions(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "pending", Title: "Pending Item", Status: StatusBacklog, Priority: 3, Tags: []string{},
	})

	// Write a workshop round with 2 unresolved decisions.
	round := WorkshopRound{
		RoundNum:  1,
		Readiness: map[string]int{"problem_clarity": 1, "scope_defined": 1, "approach_solid": 0, "testable": 0, "risk_awareness": 0},
		Items: []WorkshopItem{
			{ID: "d1", Type: "decision", Selected: nil},
			{ID: "d2", Type: "decision", Selected: nil},
			{ID: "i1", Type: "info"},
		},
	}
	workshopDir := filepath.Join(rootDir, "ideas", "pending", "workshop")
	testutil.WriteJSONFile(t, filepath.Join(workshopDir, "round-001.json"), round)

	w := doFeedbackSummary(t, h)
	testutil.AssertStatusOK(t, w)

	var resp FeedbackSummaryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}
	if resp.Items[0].PendingDecisions != 2 {
		t.Errorf("expected 2 pending decisions, got %d", resp.Items[0].PendingDecisions)
	}
	if resp.TotalPending != 2 {
		t.Errorf("expected total_pending 2, got %d", resp.TotalPending)
	}
}

func TestFeedbackSummary_MultipleItems(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	// Item with 1 pending decision.
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "one-pending", Title: "One Pending", Status: StatusBacklog, Priority: 3, Tags: []string{},
	})
	round1 := WorkshopRound{
		RoundNum:  1,
		Readiness: map[string]int{"problem_clarity": 2},
		Items:     []WorkshopItem{{ID: "d1", Type: "decision", Selected: nil}},
	}
	testutil.WriteJSONFile(t, filepath.Join(rootDir, "ideas", "one-pending", "workshop", "round-001.json"), round1)

	// Item with no workshop rounds (should not appear).
	createTestItem(t, rootDir, KindFix, BacklogItem{
		Name: "no-rounds", Title: "No Rounds", Status: StatusBacklog, Priority: 3, Tags: []string{},
	})

	// Item with all decisions resolved (should not appear).
	selected := "B"
	createTestItem(t, rootDir, KindExecute, BacklogItem{
		Name: "all-resolved", Title: "All Resolved", Status: StatusReady, Priority: 2, Tags: []string{},
	})
	round2 := WorkshopRound{
		RoundNum:  1,
		Readiness: map[string]int{"problem_clarity": 3},
		Items:     []WorkshopItem{{ID: "d1", Type: "decision", Selected: &selected}},
	}
	testutil.WriteJSONFile(t, filepath.Join(rootDir, "execute", "all-resolved", "workshop", "round-001.json"), round2)

	w := doFeedbackSummary(t, h)
	testutil.AssertStatusOK(t, w)

	var resp FeedbackSummaryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.TotalItemsAffected != 1 {
		t.Errorf("expected 1 affected item, got %d", resp.TotalItemsAffected)
	}
	if resp.TotalPending != 1 {
		t.Errorf("expected total_pending 1, got %d", resp.TotalPending)
	}
}
