package scenarios

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/testutil"
)

// --- Stubs ---

type stubBacklogLister struct {
	items []backlog.BacklogItem
}

func (s stubBacklogLister) LoadAll(_ []backlog.BacklogKind) ([]backlog.BacklogItem, error) {
	return s.items, nil
}

type stubExecutionLister struct {
	records []execution.Record
}

func (s stubExecutionLister) List(_ context.Context, _ execution.ListFilters) ([]execution.Record, error) {
	return s.records, nil
}

// --- Helpers ---

var baseTime = time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)

func makeItem(name string, kind backlog.BacklogKind, status backlog.BacklogStatus, scenarios []string, tags []string) backlog.BacklogItem {
	globs := make([]string, 0, len(scenarios))
	for _, sc := range scenarios {
		globs = append(globs, "scenarios/"+sc+"/**")
	}
	return backlog.BacklogItem{
		Name:            name,
		Kind:            kind,
		Status:          status,
		AcceptanceAllow: globs,
		Tags:            tags,
	}
}

func makeExecWithReview(scenario, classification string, reviewedAt time.Time) execution.Record {
	ts := reviewedAt.Format(time.RFC3339)
	return execution.Record{
		ExecutionID: "exec-" + scenario,
		CreatedAt:   reviewedAt.Format(time.RFC3339),
		Finalization: &execution.Finalization{
			Scenarios: []execution.ScenarioFinalization{
				{
					ScenarioName: scenario,
					Review: execution.ScenarioReviewStep{
						Status: execution.FinalizationStatusCompleted,
						Result: &execution.ReviewResult{
							Classification: classification,
							ReviewedAt:     ts,
						},
					},
				},
			},
		},
	}
}

// --- Tests ---

func TestReviewQueue_Empty(t *testing.T) {
	results, total, excluded := computeReviewQueue(nil, nil, "", 5, baseTime, nil)
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
	if total != 0 {
		t.Fatalf("expected 0 total, got %d", total)
	}
	if excluded != 0 {
		t.Fatalf("expected 0 excluded, got %d", excluded)
	}
}

func TestReviewQueue_WorkloadRanking(t *testing.T) {
	items := []backlog.BacklogItem{
		makeItem("item-1", backlog.KindExecute, backlog.StatusReady, []string{"busy-scenario"}, nil),
		makeItem("item-2", backlog.KindExecute, backlog.StatusReady, []string{"busy-scenario"}, nil),
		makeItem("item-3", backlog.KindExecute, backlog.StatusQueued, []string{"busy-scenario"}, nil),
		makeItem("item-4", backlog.KindIdea, backlog.StatusBacklog, []string{"busy-scenario"}, nil),
		makeItem("item-5", backlog.KindFix, backlog.StatusBacklog, []string{"busy-scenario"}, nil),
		makeItem("item-6", backlog.KindExecute, backlog.StatusReady, []string{"quiet-scenario"}, nil),
	}

	results, _, _ := computeReviewQueue(items, nil, "", 10, baseTime, nil)
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}
	if results[0].scenarioName != "busy-scenario" {
		t.Errorf("expected busy-scenario first, got %q", results[0].scenarioName)
	}
	if results[0].pendingCount != 5 {
		t.Errorf("expected 5 pending for busy-scenario, got %d", results[0].pendingCount)
	}
	if results[1].scenarioName != "quiet-scenario" {
		t.Errorf("expected quiet-scenario second, got %q", results[1].scenarioName)
	}
}

func TestReviewQueue_ExcludesQATaggedFixes(t *testing.T) {
	items := []backlog.BacklogItem{
		makeItem("item-1", backlog.KindExecute, backlog.StatusReady, []string{"healthy-scenario"}, nil),
		makeItem("item-2", backlog.KindExecute, backlog.StatusReady, []string{"broken-scenario"}, nil),
		// This fix item should cause broken-scenario to be excluded.
		makeItem("qa-fix", backlog.KindFix, backlog.StatusReady, []string{"broken-scenario"}, []string{"preemptive-qa"}),
	}

	results, total, excluded := computeReviewQueue(items, nil, "", 10, baseTime, nil)
	if excluded != 1 {
		t.Fatalf("expected 1 excluded, got %d", excluded)
	}
	if total != 2 {
		t.Fatalf("expected 2 total scenarios, got %d", total)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].scenarioName != "healthy-scenario" {
		t.Errorf("expected healthy-scenario, got %q", results[0].scenarioName)
	}
}

func TestReviewQueue_ExcludesChoreTaggedItems(t *testing.T) {
	items := []backlog.BacklogItem{
		makeItem("item-1", backlog.KindExecute, backlog.StatusReady, []string{"scenario-a"}, nil),
		makeItem("qa-chore", backlog.KindChore, backlog.StatusBacklog, []string{"scenario-a"}, []string{"preemptive-qa"}),
	}

	_, _, excluded := computeReviewQueue(items, nil, "", 10, baseTime, nil)
	if excluded != 1 {
		t.Fatalf("expected 1 excluded, got %d", excluded)
	}
}

func TestReviewQueue_CompletedFixDoesNotExclude(t *testing.T) {
	items := []backlog.BacklogItem{
		makeItem("item-1", backlog.KindExecute, backlog.StatusReady, []string{"scenario-a"}, nil),
		// Completed fix should not cause exclusion.
		makeItem("qa-fix-done", backlog.KindFix, backlog.StatusCompleted, []string{"scenario-a"}, []string{"preemptive-qa"}),
	}

	results, _, excluded := computeReviewQueue(items, nil, "", 10, baseTime, nil)
	if excluded != 0 {
		t.Fatalf("expected 0 excluded, got %d", excluded)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestReviewQueue_StalenessBoost(t *testing.T) {
	items := []backlog.BacklogItem{
		makeItem("item-1", backlog.KindExecute, backlog.StatusReady, []string{"stale-scenario"}, nil),
		makeItem("item-2", backlog.KindExecute, backlog.StatusReady, []string{"fresh-scenario"}, nil),
	}

	records := []execution.Record{
		makeExecWithReview("stale-scenario", "needs_work", baseTime.Add(-29*24*time.Hour)),
		makeExecWithReview("fresh-scenario", "ready", baseTime.Add(-1*time.Hour)),
	}

	results, _, _ := computeReviewQueue(items, records, "", 10, baseTime, nil)
	if len(results) < 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// Both have 1 pending item (same workload). Stale scenario should rank higher
	// due to staleness component.
	if results[0].scenarioName != "stale-scenario" {
		t.Errorf("expected stale-scenario first, got %q", results[0].scenarioName)
	}
}

func TestReviewQueue_RecentActivityBoost(t *testing.T) {
	items := []backlog.BacklogItem{
		makeItem("item-1", backlog.KindExecute, backlog.StatusReady, []string{"active-scenario"}, nil),
		makeItem("item-2", backlog.KindExecute, backlog.StatusReady, []string{"idle-scenario"}, nil),
	}

	// active-scenario has many recent executions.
	var records []execution.Record
	for i := 0; i < 5; i++ {
		records = append(records, execution.Record{
			ExecutionID: "exec-active-" + time.Now().String(),
			CreatedAt:   baseTime.Add(-time.Duration(i) * 24 * time.Hour).Format(time.RFC3339),
			Finalization: &execution.Finalization{
				Scenarios: []execution.ScenarioFinalization{
					{ScenarioName: "active-scenario"},
				},
			},
		})
	}

	results, _, _ := computeReviewQueue(items, records, "", 10, baseTime, nil)
	if len(results) < 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].scenarioName != "active-scenario" {
		t.Errorf("expected active-scenario first, got %q", results[0].scenarioName)
	}
	if results[0].recentExecutionCount != 5 {
		t.Errorf("expected 5 recent executions, got %d", results[0].recentExecutionCount)
	}
}

func TestReviewQueue_CooldownSet(t *testing.T) {
	items := []backlog.BacklogItem{
		makeItem("item-1", backlog.KindExecute, backlog.StatusReady, []string{"just-reviewed"}, nil),
	}

	records := []execution.Record{
		makeExecWithReview("just-reviewed", "ready", baseTime.Add(-6*time.Hour)),
	}

	results, _, _ := computeReviewQueue(items, records, "", 10, baseTime, nil)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].cooldownUntil.IsZero() {
		t.Error("expected cooldown to be set for recently-reviewed scenario")
	}
	expectedCooldown := baseTime.Add(-6*time.Hour + defaultCooldownHours*time.Hour)
	if !results[0].cooldownUntil.Equal(expectedCooldown) {
		t.Errorf("expected cooldown at %v, got %v", expectedCooldown, results[0].cooldownUntil)
	}
}

func TestReviewQueue_NoCooldownForOldReview(t *testing.T) {
	items := []backlog.BacklogItem{
		makeItem("item-1", backlog.KindExecute, backlog.StatusReady, []string{"old-review"}, nil),
	}

	records := []execution.Record{
		makeExecWithReview("old-review", "needs_work", baseTime.Add(-48*time.Hour)),
	}

	results, _, _ := computeReviewQueue(items, records, "", 10, baseTime, nil)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].cooldownUntil.IsZero() {
		t.Error("expected no cooldown for old review")
	}
}

func TestReviewQueue_LimitCaps(t *testing.T) {
	var items []backlog.BacklogItem
	for i := 0; i < 10; i++ {
		name := "scenario-" + strconv.Itoa(i)
		items = append(items, makeItem("item-"+name, backlog.KindExecute, backlog.StatusReady, []string{name}, nil))
	}

	results, total, _ := computeReviewQueue(items, nil, "", 3, baseTime, nil)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if total != 10 {
		t.Fatalf("expected 10 total, got %d", total)
	}
}

func TestReviewQueue_NoAcceptanceAllow(t *testing.T) {
	items := []backlog.BacklogItem{
		{
			Name:   "no-globs",
			Kind:   backlog.KindExecute,
			Status: backlog.StatusReady,
			// No AcceptanceAllow set.
		},
	}

	results, total, _ := computeReviewQueue(items, nil, "", 10, baseTime, nil)
	if len(results) != 0 {
		t.Fatalf("expected 0 results for items without acceptance_allow, got %d", len(results))
	}
	if total != 0 {
		t.Fatalf("expected 0 total, got %d", total)
	}
}

func TestReviewQueue_CompletedItemsIgnored(t *testing.T) {
	items := []backlog.BacklogItem{
		makeItem("completed-item", backlog.KindExecute, backlog.StatusCompleted, []string{"done-scenario"}, nil),
		makeItem("failed-item", backlog.KindFix, backlog.StatusFailed, []string{"done-scenario"}, nil),
		makeItem("active-item", backlog.KindExecute, backlog.StatusReady, []string{"active-scenario"}, nil),
	}

	results, _, _ := computeReviewQueue(items, nil, "", 10, baseTime, nil)
	// Only active-scenario should appear (done-scenario has only terminal items).
	found := false
	for _, r := range results {
		if r.scenarioName == "done-scenario" {
			t.Error("done-scenario should not appear (all items are terminal)")
		}
		if r.scenarioName == "active-scenario" {
			found = true
		}
	}
	if !found {
		t.Error("expected active-scenario in results")
	}
}

func TestReviewQueue_PrimarySignal(t *testing.T) {
	// Scenario with high workload should have "workload" as primary signal.
	items := []backlog.BacklogItem{
		makeItem("item-1", backlog.KindExecute, backlog.StatusReady, []string{"high-workload"}, nil),
		makeItem("item-2", backlog.KindExecute, backlog.StatusReady, []string{"high-workload"}, nil),
		makeItem("item-3", backlog.KindExecute, backlog.StatusReady, []string{"high-workload"}, nil),
	}

	results, _, _ := computeReviewQueue(items, nil, "", 10, baseTime, nil)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].primarySignal != "workload" {
		t.Errorf("expected primary signal 'workload', got %q", results[0].primarySignal)
	}
}

func TestReviewQueue_CustomExcludeTag(t *testing.T) {
	items := []backlog.BacklogItem{
		makeItem("item-1", backlog.KindExecute, backlog.StatusReady, []string{"scenario-a"}, nil),
		makeItem("custom-fix", backlog.KindFix, backlog.StatusReady, []string{"scenario-a"}, []string{"custom-tag"}),
	}

	// Default tag should not exclude.
	_, _, excluded := computeReviewQueue(items, nil, "", 10, baseTime, nil)
	if excluded != 0 {
		t.Fatalf("expected 0 excluded with default tag, got %d", excluded)
	}

	// Custom tag should exclude.
	_, _, excluded = computeReviewQueue(items, nil, "custom-tag", 10, baseTime, nil)
	if excluded != 1 {
		t.Fatalf("expected 1 excluded with custom tag, got %d", excluded)
	}
}

// --- HTTP Handler Test ---

func TestReviewQueue_HTTPHandler(t *testing.T) {
	root, sources := setupTestScenarios(t)
	handler := newTestHandler(root, sources)
	handler.SetBacklogLister(stubBacklogLister{
		items: []backlog.BacklogItem{
			makeItem("item-1", backlog.KindExecute, backlog.StatusReady, []string{"test-scenario-1"}, nil),
			makeItem("item-2", backlog.KindExecute, backlog.StatusReady, []string{"test-scenario-1"}, nil),
		},
	})
	handler.SetExecutionLister(stubExecutionLister{})

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/review-queue?limit=5", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	testutil.AssertStatusOK(t, rec)

	var resp struct {
		Items          []json.RawMessage `json:"items"`
		TotalScenarios int               `json:"total_scenarios"`
		ExcludedCount  int               `json:"excluded_count"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}
	if resp.TotalScenarios != 1 {
		t.Errorf("expected 1 total scenario, got %d", resp.TotalScenarios)
	}
}

func TestReviewQueue_HTTPHandler_InvalidLimit(t *testing.T) {
	root, sources := setupTestScenarios(t)
	handler := newTestHandler(root, sources)
	handler.SetBacklogLister(stubBacklogLister{})
	handler.SetExecutionLister(stubExecutionLister{})

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/review-queue?limit=999", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid limit, got %d", rec.Code)
	}
}

// --- Existence filtering tests (Rec #1) ---

func TestReviewQueue_FiltersNonExistentScenarios(t *testing.T) {
	items := []backlog.BacklogItem{
		makeItem("item-1", backlog.KindExecute, backlog.StatusReady, []string{"real-scenario"}, nil),
		makeItem("item-2", backlog.KindExecute, backlog.StatusReady, []string{"phantom-scenario"}, nil),
	}

	existing := map[string]bool{"real-scenario": true}
	results, total, excluded := computeReviewQueue(items, nil, "", 10, baseTime, existing)

	if total != 2 {
		t.Fatalf("expected 2 total, got %d", total)
	}
	if excluded != 1 {
		t.Fatalf("expected 1 excluded (phantom), got %d", excluded)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].scenarioName != "real-scenario" {
		t.Errorf("expected real-scenario, got %q", results[0].scenarioName)
	}
}

func TestReviewQueue_NilExistingScenariosSkipsFilter(t *testing.T) {
	// With nil existingScenarios and a maintenance item, all scenarios pass through.
	items := []backlog.BacklogItem{
		makeItem("item-1", backlog.KindFix, backlog.StatusReady, []string{"any-scenario"}, nil),
	}

	results, _, excluded := computeReviewQueue(items, nil, "", 10, baseTime, nil)
	if excluded != 0 {
		t.Fatalf("expected 0 excluded with nil existingScenarios, got %d", excluded)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestReviewQueue_EmptyExistingScenariosExcludesAll(t *testing.T) {
	items := []backlog.BacklogItem{
		makeItem("item-1", backlog.KindExecute, backlog.StatusReady, []string{"scenario-a"}, nil),
	}

	existing := map[string]bool{} // empty = no scenarios exist
	results, _, excluded := computeReviewQueue(items, nil, "", 10, baseTime, existing)
	if excluded != 1 {
		t.Fatalf("expected 1 excluded, got %d", excluded)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

// --- Greenfield fallback heuristic tests (Rec #2) ---

func TestGreenfieldFallback_ExcludesCreationOnlyScenarios(t *testing.T) {
	items := []backlog.BacklogItem{
		makeItem("plan-it", backlog.KindExecute, backlog.StatusReady, []string{"planned-scenario"}, nil),
		makeItem("research-it", backlog.KindResearch, backlog.StatusBacklog, []string{"planned-scenario"}, nil),
	}

	// Simulate: computeReviewQueue was called with nil existingScenarios,
	// so the scenario made it through. Now apply the fallback.
	input := []reviewQueueResult{
		{scenarioName: "planned-scenario", pendingCount: 2},
	}
	results, excluded := applyGreenfieldFallback(items, input, 0)
	if excluded != 1 {
		t.Fatalf("expected 1 excluded, got %d", excluded)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestGreenfieldFallback_KeepsScenarioWithMaintenanceItems(t *testing.T) {
	items := []backlog.BacklogItem{
		makeItem("build-it", backlog.KindExecute, backlog.StatusReady, []string{"mixed-scenario"}, nil),
		makeItem("fix-it", backlog.KindFix, backlog.StatusReady, []string{"mixed-scenario"}, nil),
	}

	input := []reviewQueueResult{
		{scenarioName: "mixed-scenario", pendingCount: 2},
	}
	results, excluded := applyGreenfieldFallback(items, input, 0)
	if excluded != 0 {
		t.Fatalf("expected 0 excluded, got %d", excluded)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestGreenfieldFallback_KeepsCreationOnlyWithReviewHistory(t *testing.T) {
	items := []backlog.BacklogItem{
		makeItem("build-it", backlog.KindExecute, backlog.StatusReady, []string{"reviewed-scenario"}, nil),
	}

	// Scenario has been reviewed before — even though all items are creation kinds,
	// it should not be excluded (it existed at some point).
	input := []reviewQueueResult{
		{scenarioName: "reviewed-scenario", pendingCount: 1, lastReviewAt: baseTime.Add(-24 * time.Hour)},
	}
	results, excluded := applyGreenfieldFallback(items, input, 0)
	if excluded != 0 {
		t.Fatalf("expected 0 excluded, got %d", excluded)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestReviewQueue_HTTPHandler_NoDeps(t *testing.T) {
	root, sources := setupTestScenarios(t)
	handler := newTestHandler(root, sources)
	// Don't set backlog/execution listers.

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/review-queue", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when deps not configured, got %d", rec.Code)
	}
}
