package backlog

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"swarm-manager/internal/testutil"
)

func TestBacklogSummary_LegacyAnsweredRoundInfersPendingSynthesis(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	createTestItem(t, rootDir, KindResearch, BacklogItem{
		Name: "legacy-summary", Title: "Legacy Summary", Status: StatusResearching, Priority: 1, Tags: []string{},
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
	workshopDir := filepath.Join(rootDir, "research", "legacy-summary", "workshop")
	testutil.WriteJSONFile(t, filepath.Join(workshopDir, "round-001.json"), round)

	req := httptest.NewRequest("GET", "/api/v1/backlog/summary", nil)
	w := httptest.NewRecorder()
	h.BacklogSummary(w, req)
	testutil.AssertStatusOK(t, w)

	var resp BacklogSummaryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Maturity.Items) != 1 {
		t.Fatalf("expected 1 maturity item, got %d", len(resp.Maturity.Items))
	}
	if !resp.Maturity.Items[0].PendingSynthesis {
		t.Fatal("expected legacy answered round to infer pending_synthesis=true in backlog summary")
	}
}
