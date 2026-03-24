package backlog

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"swarm-manager/internal/testutil"
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
