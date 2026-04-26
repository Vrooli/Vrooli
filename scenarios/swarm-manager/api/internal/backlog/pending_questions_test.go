package backlog

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"swarm-manager/internal/testutil"
)

func doPendingQuestions(t *testing.T, h *Handler, query string) *httptest.ResponseRecorder {
	t.Helper()
	path := "/api/v1/backlog/pending-questions"
	if query != "" {
		path += "?" + query
	}
	req := httptest.NewRequest("GET", path, nil)
	w := httptest.NewRecorder()
	h.PendingQuestions(w, req)
	return w
}

func TestPendingQuestions_EmptyBacklog(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := doPendingQuestions(t, h, "")
	testutil.AssertStatusOK(t, w)

	var resp PendingQuestionsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(resp.Items))
	}
}

func TestPendingQuestions_WorkshopDecisionsOnly(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "pending-ws", Title: "Pending Workshop", Status: StatusBacklog, Priority: 3, Tags: []string{},
	})

	// Write a workshop round with 2 unresolved and 1 resolved decision.
	selected := "A"
	round := WorkshopRound{
		RoundNum:  1,
		Readiness: map[string]int{"problem_clarity": 1, "scope_defined": 1, "approach_solid": 0, "testable": 0, "risk_awareness": 0},
		Items: []WorkshopItem{
			{ID: "d1", Type: "decision", Topic: "Architecture", Options: []WorkshopOption{{Key: "A", Label: "Monolith"}, {Key: "B", Label: "Microservices"}}, Selected: nil},
			{ID: "d2", Type: "decision", Topic: "Stack", Selected: nil},
			{ID: "d3", Type: "decision", Topic: "Resolved", Selected: &selected},
			{ID: "i1", Type: "info", Text: "Some info"},
		},
	}
	testutil.WriteJSONFile(t, filepath.Join(rootDir, "ideas", "pending-ws", "workshop", "round-001.json"), round)

	w := doPendingQuestions(t, h, "")
	testutil.AssertStatusOK(t, w)

	var resp PendingQuestionsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}

	pqi := resp.Items[0]
	if pqi.Kind != KindIdea || pqi.Name != "pending-ws" {
		t.Errorf("unexpected item: %s/%s", pqi.Kind, pqi.Name)
	}

	// Only 2 unresolved decisions — resolved d3 and info i1 should be excluded.
	if len(pqi.Questions) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(pqi.Questions))
	}
	for _, q := range pqi.Questions {
		if q.Source != "workshop" {
			t.Errorf("expected source=workshop, got %s", q.Source)
		}
		if q.RoundNumber != 1 {
			t.Errorf("expected round_number=1, got %d", q.RoundNumber)
		}
	}
	if pqi.Questions[0].ID != "d1" || pqi.Questions[0].Topic != "Architecture" {
		t.Errorf("first question mismatch: id=%s topic=%s", pqi.Questions[0].ID, pqi.Questions[0].Topic)
	}
	if len(pqi.Questions[0].Options) != 2 {
		t.Errorf("expected 2 options, got %d", len(pqi.Questions[0].Options))
	}
}

func TestPendingQuestions_ReviewItemsOnly(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "pending-review", Title: "Pending Review", Status: StatusBacklog, Priority: 2, Tags: []string{},
	})

	// Write PRD with targets using the modern format.
	itemDir := filepath.Join(rootDir, "ideas", "pending-review")
	archiveDir := filepath.Join(itemDir, "archive")
	prd := "# PRD\n\nThis is a test.\n\n## \U0001F3AF Operational Targets\n\n### \U0001F534 P0 \u2013 Must ship for viability\n- [ ] OT-P0-001 | Core Target | Must support X\n\n### \U0001F7E0 P1 \u2013 Should have post-launch\n- [ ] OT-P1-002 | Optional Target | Nice to have\n"
	testutil.WriteFile(t, filepath.Join(archiveDir, "PRD.md"), prd)

	// Mark one target as approved in review state.
	reviewState := map[string]ReviewState{
		"OT-P0-001": {ReviewStatus: "approved", ReviewedAt: "2026-01-01T00:00:00Z"},
	}
	testutil.WriteJSONFile(t, filepath.Join(itemDir, "archive", "review-state.json"), reviewState)

	w := doPendingQuestions(t, h, "source=review")
	testutil.AssertStatusOK(t, w)

	var resp PendingQuestionsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}

	pqi := resp.Items[0]
	// Only OT-P1-002 should be unreviewed.
	if len(pqi.Questions) != 1 {
		t.Fatalf("expected 1 question, got %d", len(pqi.Questions))
	}
	q := pqi.Questions[0]
	if q.Source != "review" || q.ReviewType != "target" {
		t.Errorf("expected review/target, got %s/%s", q.Source, q.ReviewType)
	}
	if q.ID != "OT-P1-002" {
		t.Errorf("expected OT-P1-002, got %s", q.ID)
	}
}

func TestPendingQuestions_MixedSources(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "mixed", Title: "Mixed Sources", Status: StatusBacklog, Priority: 1, Tags: []string{},
	})

	// Workshop round with 1 pending decision.
	round := WorkshopRound{
		RoundNum:  2,
		Readiness: map[string]int{"problem_clarity": 2},
		Items:     []WorkshopItem{{ID: "d1", Type: "decision", Topic: "Approach", Selected: nil}},
	}
	testutil.WriteJSONFile(t, filepath.Join(rootDir, "ideas", "mixed", "workshop", "round-002.json"), round)

	// PRD with 1 unreviewed target.
	prd := "# PRD\n\n## \U0001F3AF Operational Targets\n\n### \U0001F534 P0 \u2013 Must ship for viability\n- [ ] OT-P0-001 | Target | Desc\n"
	testutil.WriteFile(t, filepath.Join(rootDir, "ideas", "mixed", "archive", "PRD.md"), prd)

	w := doPendingQuestions(t, h, "source=all")
	testutil.AssertStatusOK(t, w)

	var resp PendingQuestionsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}

	pqi := resp.Items[0]
	// Should have workshop question first, then review question.
	if len(pqi.Questions) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(pqi.Questions))
	}
	if pqi.Questions[0].Source != "workshop" {
		t.Errorf("expected first question source=workshop, got %s", pqi.Questions[0].Source)
	}
	if pqi.Questions[1].Source != "review" {
		t.Errorf("expected second question source=review, got %s", pqi.Questions[1].Source)
	}
}

func TestPendingQuestions_AllResolved(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "all-done", Title: "All Done", Status: StatusReady, Priority: 2, Tags: []string{},
	})

	// All decisions resolved.
	selected := "B"
	round := WorkshopRound{
		RoundNum:  1,
		Readiness: map[string]int{"problem_clarity": 3},
		Items:     []WorkshopItem{{ID: "d1", Type: "decision", Selected: &selected}},
	}
	testutil.WriteJSONFile(t, filepath.Join(rootDir, "ideas", "all-done", "workshop", "round-001.json"), round)

	w := doPendingQuestions(t, h, "")
	testutil.AssertStatusOK(t, w)

	var resp PendingQuestionsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// No pending questions → empty items.
	if len(resp.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(resp.Items))
	}
}

func TestPendingQuestions_SortedByDependencyDepthAndPriority(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "base", Title: "Base", Status: StatusBacklog, Priority: 8,
		Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "peer", Title: "Peer", Status: StatusBacklog, Priority: 2,
		Created: "2026-01-02T00:00:00Z", Updated: "2026-01-02T00:00:00Z",
	})
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "child", Title: "Child", Status: StatusBacklog, Priority: 1, DependsOn: []string{"idea/base"},
		Created: "2026-01-03T00:00:00Z", Updated: "2026-01-03T00:00:00Z",
	})

	for _, name := range []string{"base", "peer", "child"} {
		testutil.WriteJSONFile(t, filepath.Join(rootDir, "ideas", name, "workshop", "round-001.json"), WorkshopRound{
			RoundNum: 1,
			Items:    []WorkshopItem{{ID: "d1", Type: "decision", Topic: name}},
		})
	}

	w := doPendingQuestions(t, h, "")
	testutil.AssertStatusOK(t, w)

	var resp PendingQuestionsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(resp.Items))
	}
	if resp.Items[0].Name != "peer" {
		t.Fatalf("first item = %s, want peer", resp.Items[0].Name)
	}
	if resp.Items[1].Name != "base" {
		t.Fatalf("second item = %s, want base", resp.Items[1].Name)
	}
	if resp.Items[2].Name != "child" {
		t.Fatalf("third item = %s, want child", resp.Items[2].Name)
	}
}

func TestPendingQuestions_UnblockingBoostBeatsRecency(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "hub", Title: "Hub", Status: StatusBacklog, Priority: 5,
		Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "fresh", Title: "Fresh", Status: StatusBacklog, Priority: 5,
		Created: "2026-01-03T00:00:00Z", Updated: "2026-01-03T00:00:00Z",
	})
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "downstream-a", Title: "Downstream A", Status: StatusBacklog, Priority: 5, DependsOn: []string{"idea/hub"},
		Created: "2026-01-02T00:00:00Z", Updated: "2026-01-02T00:00:00Z",
	})
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "downstream-b", Title: "Downstream B", Status: StatusBacklog, Priority: 5, DependsOn: []string{"idea/downstream-a"},
		Created: "2026-01-02T00:00:00Z", Updated: "2026-01-02T00:00:00Z",
	})

	for _, name := range []string{"hub", "fresh"} {
		testutil.WriteJSONFile(t, filepath.Join(rootDir, "ideas", name, "workshop", "round-001.json"), WorkshopRound{
			RoundNum: 1,
			Items:    []WorkshopItem{{ID: "d1", Type: "decision", Topic: name}},
		})
	}

	w := doPendingQuestions(t, h, "")
	testutil.AssertStatusOK(t, w)

	var resp PendingQuestionsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Items))
	}
	if resp.Items[0].Name != "hub" {
		t.Fatalf("first item = %s, want hub", resp.Items[0].Name)
	}
}

func TestPendingQuestions_LimitAndInitiativeFilter(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	for i, name := range []string{"a", "b", "c"} {
		initiative := "target"
		if i == 2 {
			initiative = "other"
		}
		createTestItem(t, rootDir, KindIdea, BacklogItem{
			Name: name, Title: name, Status: StatusBacklog, Priority: i + 1, Initiative: initiative,
			Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
		})
		testutil.WriteJSONFile(t, filepath.Join(rootDir, "ideas", name, "workshop", "round-001.json"), WorkshopRound{
			RoundNum: 1,
			Items:    []WorkshopItem{{ID: "d1", Type: "decision", Topic: name}},
		})
	}

	w := doPendingQuestions(t, h, "initiative=target&limit=1")
	testutil.AssertStatusOK(t, w)

	var resp PendingQuestionsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}
	if got := resp.Items[0].Name; got != "a" {
		t.Fatalf("first item = %s, want a", got)
	}
}

func TestPendingQuestions_InvalidQueryParams(t *testing.T) {
	h, _ := setupTestHandler(t)

	for _, query := range []string{"source=nope", "limit=-1", "limit=abc"} {
		w := doPendingQuestions(t, h, query)
		if w.Code != 400 {
			t.Fatalf("query %q: status = %d, want 400", query, w.Code)
		}
	}
}
