package backlog

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"swarm-manager/internal/testutil"
	"testing"
)

func doMaturitySummary(t *testing.T, h *Handler) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("GET", "/api/v1/backlog/maturity-summary", nil)
	w := httptest.NewRecorder()
	h.MaturitySummary(w, req)
	return w
}

func TestMaturitySummary_EmptyBacklog(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := doMaturitySummary(t, h)
	testutil.AssertStatusOK(t, w)

	var resp MaturitySummaryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(resp.Items))
	}
}

func TestMaturitySummary_NoWorkshopRounds(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "no-rounds", Title: "No Rounds", Status: StatusBacklog, Priority: 3, Tags: []string{},
	})

	w := doMaturitySummary(t, h)
	testutil.AssertStatusOK(t, w)

	var resp MaturitySummaryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}

	item := resp.Items[0]
	if item.RoundsCompleted != 0 {
		t.Errorf("expected 0 rounds, got %d", item.RoundsCompleted)
	}
	if item.Ready {
		t.Error("expected ready=false with no rounds")
	}
	if item.HasPlan {
		t.Error("expected has_plan=false with no plan.md")
	}
	if item.PendingSynthesis {
		t.Error("expected pending_synthesis=false with no rounds")
	}
}

func TestMaturitySummary_WithPlan(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "has-plan", Title: "Has Plan", Status: StatusReady, Priority: 2, Tags: []string{},
	})
	testutil.WriteFile(t, filepath.Join(rootDir, "ideas", "has-plan", "plan.md"), "# Plan\nTest plan.")

	w := doMaturitySummary(t, h)
	testutil.AssertStatusOK(t, w)

	var resp MaturitySummaryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}
	if !resp.Items[0].HasPlan {
		t.Error("expected has_plan=true when plan.md exists")
	}
}

func TestMaturitySummary_ResearchUsesConclusionDeliverable(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	createTestItem(t, rootDir, KindResearch, BacklogItem{
		Name: "has-conclusion", Title: "Has Conclusion", Status: StatusReady, Priority: 2, Tags: []string{},
	})
	testutil.WriteFile(t, filepath.Join(rootDir, "research", "has-conclusion", "conclusion.md"), "# Conclusion\nTest conclusion.")

	w := doMaturitySummary(t, h)
	testutil.AssertStatusOK(t, w)

	var resp MaturitySummaryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}
	if !resp.Items[0].HasPlan {
		t.Error("expected has_plan=true when conclusion.md exists for research items")
	}
}

func TestMaturitySummary_ReadyScores(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "fully-ready", Title: "Fully Ready", Status: StatusReady, Priority: 1, Tags: []string{},
	})

	// Write a round with all dimensions at max score.
	round := WorkshopRound{
		RoundNum: 1,
		Readiness: map[string]int{
			"problem_clarity": 3,
			"scope_defined":   3,
			"approach_solid":  3,
			"testable":        3,
			"risk_awareness":  3,
		},
		Items: []WorkshopItem{},
	}
	workshopDir := filepath.Join(rootDir, "ideas", "fully-ready", "workshop")
	testutil.WriteJSONFile(t, filepath.Join(workshopDir, "round-001.json"), round)

	w := doMaturitySummary(t, h)
	testutil.AssertStatusOK(t, w)

	var resp MaturitySummaryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}

	item := resp.Items[0]
	if !item.Ready {
		t.Error("expected ready=true with all scores at 3")
	}
	if item.RoundsCompleted != 1 {
		t.Errorf("expected 1 round completed, got %d", item.RoundsCompleted)
	}
}

func TestMaturitySummary_PendingSynthesis(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "pending-synthesis", Title: "Pending Synthesis", Status: StatusResearching, Priority: 1, Tags: []string{},
	})

	round := WorkshopRound{
		RoundNum:         1,
		PendingSynthesis: true,
		Readiness: map[string]int{
			"problem_clarity": 3,
			"scope_defined":   3,
			"approach_solid":  3,
			"testable":        3,
			"risk_awareness":  3,
		},
		Items: []WorkshopItem{},
	}
	workshopDir := filepath.Join(rootDir, "ideas", "pending-synthesis", "workshop")
	testutil.WriteJSONFile(t, filepath.Join(workshopDir, "round-001.json"), round)

	w := doMaturitySummary(t, h)
	testutil.AssertStatusOK(t, w)

	var resp MaturitySummaryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}
	if !resp.Items[0].PendingSynthesis {
		t.Error("expected pending_synthesis=true when latest round is awaiting synthesis")
	}
}

func TestMaturitySummary_LegacyAnsweredRoundInfersPendingSynthesis(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	createTestItem(t, rootDir, KindResearch, BacklogItem{
		Name: "legacy-finalize", Title: "Legacy Finalize", Status: StatusResearching, Priority: 1, Tags: []string{},
	})

	round := WorkshopRound{
		RoundNum: 1,
		Readiness: map[string]int{
			"problem_clarity": 3,
			"scope_defined":   3,
			"approach_solid":  3,
			"testable":        3,
			"risk_awareness":  3,
		},
		Items: []WorkshopItem{
			{ID: "d1", Type: "decision", Selected: strPtr("A")},
			{ID: "i1", Type: "info", Text: "legacy"},
		},
	}
	workshopDir := filepath.Join(rootDir, "research", "legacy-finalize", "workshop")
	testutil.WriteJSONFile(t, filepath.Join(workshopDir, "round-001.json"), round)

	w := doMaturitySummary(t, h)
	testutil.AssertStatusOK(t, w)

	var resp MaturitySummaryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}
	if !resp.Items[0].PendingSynthesis {
		t.Error("expected legacy answered round to infer pending_synthesis=true")
	}
}
