package backlog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"swarm-manager/internal/execution"

	"github.com/gorilla/mux"
)

type fixedDecisionCount struct{ counts map[string]int }

func (p fixedDecisionCount) PendingDecisionCounts(context.Context) (map[string]int, error) {
	return p.counts, nil
}

// [REQ:SWM-P0-010] item next-action projection driven by execution preflight reasons
func TestResolveNextActionDecisionTable(t *testing.T) {
	tests := []struct {
		name      string
		item      BacklogItem
		preflight execution.ProcessPreflight
		want      NextActionID
	}{
		{name: "no canonical plan authors plan", item: BacklogItem{Name: "no-plan", Kind: KindIdea, Status: StatusBacklog}, want: NextActionAuthorPlan},
		{name: "suggestion is accepted before planning", item: BacklogItem{Name: "suggested", Kind: KindIdea, Status: StatusSuggested}, want: NextActionAcceptSuggestion},
		{name: "plan-ready item without criteria defines them", item: BacklogItem{Name: "criteria", Kind: KindIdea, Status: StatusReady, PlanRef: testPlanRef("criteria")}, want: NextActionDefineCriteria},
		{name: "ready item runs", item: BacklogItem{Name: "ready", Kind: KindIdea, Status: StatusReady, PlanRef: testPlanRef("ready"), AcceptanceCriteria: testAcceptanceCriteria()}, preflight: execution.ProcessPreflight{Ready: true}, want: NextActionRun},
		{name: "unaccepted plan is accepted", item: BacklogItem{Name: "unaccepted", Kind: KindIdea, Status: StatusReady, PlanRef: testPlanRef("unaccepted"), AcceptanceCriteria: testAcceptanceCriteria()}, preflight: execution.ProcessPreflight{BlockingReasons: []string{"canonical plan has not been explicitly accepted — accept the current plan revision before queueing"}, BlockingDetails: []execution.ProcessBlockingReason{{Code: "plan_not_accepted", Message: "canonical plan has not been explicitly accepted — accept the current plan revision before queueing"}}}, want: NextActionAcceptPlan},
		{name: "invalid plan is repaired", item: BacklogItem{Name: "invalid", Kind: KindIdea, Status: StatusReady, PlanRef: testPlanRef("invalid"), AcceptanceCriteria: testAcceptanceCriteria()}, preflight: execution.ProcessPreflight{BlockingReasons: []string{"canonical plan is not valid: quality status is \"fail\""}, BlockingDetails: []execution.ProcessBlockingReason{{Code: "plan_invalid", Message: "canonical plan is not valid: quality status is \"fail\""}}}, want: NextActionRepairPlan},
		{name: "dependencies are resolved", item: BacklogItem{Name: "blocked", Kind: KindIdea, Status: StatusReady, PlanRef: testPlanRef("blocked"), AcceptanceCriteria: testAcceptanceCriteria(), DependsOn: []string{"idea/upstream"}}, preflight: execution.ProcessPreflight{Ready: true}, want: NextActionResolveDependencies},
		{name: "review is reviewed", item: BacklogItem{Name: "review", Kind: KindIdea, Status: StatusReviewPending}, want: NextActionReview},
		{name: "active review views execution", item: BacklogItem{Name: "in-review", Kind: KindIdea, Status: StatusInReview}, want: NextActionViewExecution},
		{name: "active item views execution", item: BacklogItem{Name: "active", Kind: KindIdea, Status: StatusInProgress}, want: NextActionViewExecution},
		{name: "failed item retries", item: BacklogItem{Name: "failed", Kind: KindIdea, Status: StatusFailed}, want: NextActionRetry},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, root := setupTestHandler(t)
			h.SetExecutionQueuer(&mockExecutionQueuer{preflightResult: tt.preflight})
			if tt.item.Name == "blocked" {
				createTestItem(t, root, KindIdea, BacklogItem{Name: "upstream", Kind: KindIdea, Status: StatusBacklog})
			}
			createTestItem(t, root, tt.item.Kind, tt.item)
			got, err := h.ResolveNextAction(t.Context(), tt.item)
			if err != nil {
				t.Fatalf("ResolveNextAction() error = %v", err)
			}
			if got.ID != tt.want {
				t.Fatalf("action = %q, want %q (%+v)", got.ID, tt.want, got)
			}
			if got.ID == NextActionRun && !got.Enabled {
				t.Fatal("run action must be enabled only when preflight is ready")
			}
		})
	}
}

// Every canonical lifecycle state has a deterministic projection. Open states
// must remain operator-actionable; active wait states deliberately resolve to
// view_execution so the ranked inbox can exclude them without inventing a
// disabled primary CTA.
func TestResolveNextActionLifecycleMatrix(t *testing.T) {
	tests := []struct {
		status BacklogStatus
		want   NextActionID
	}{
		{StatusSuggested, NextActionAcceptSuggestion},
		{StatusBacklog, NextActionAuthorPlan},
		{StatusResearching, NextActionAuthorPlan},
		{StatusReady, NextActionAuthorPlan},
		{StatusQueued, NextActionViewExecution},
		{StatusInProgress, NextActionViewExecution},
		{StatusInReview, NextActionViewExecution},
		{StatusReviewPending, NextActionReview},
		{StatusCompleted, NextActionArchive},
		{StatusFailed, NextActionRetry},
		{StatusNeedsFollowup, NextActionAuthorFollowup},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			h, _ := setupTestHandler(t)
			h.SetExecutionQueuer(&mockExecutionQueuer{preflightResult: execution.ProcessPreflight{Ready: true}})
			item := BacklogItem{Name: string(tt.status), Kind: KindIdea, Status: tt.status}
			action, err := h.ResolveNextAction(t.Context(), item)
			if err != nil {
				t.Fatal(err)
			}
			if action.ID != tt.want || !action.Enabled {
				t.Fatalf("status %q resolved %#v, want enabled %q", tt.status, action, tt.want)
			}
		})
	}
}

func TestNextActionEndpointsSingleAndBoundedBatch(t *testing.T) {
	h, root := setupTestHandler(t)
	h.SetExecutionQueuer(&mockExecutionQueuer{preflightResult: execution.ProcessPreflight{Ready: true}})
	createReadyTestItem(t, root, KindIdea, BacklogItem{Name: "ready", Kind: KindIdea, Status: StatusReady, AcceptanceCriteria: testAcceptanceCriteria()})
	router := mux.NewRouter()
	h.RegisterRoutes(router)

	single := httptest.NewRequest(http.MethodGet, "/api/v1/backlog/idea/ready/next-action", nil)
	singleW := httptest.NewRecorder()
	router.ServeHTTP(singleW, single)
	if singleW.Code != http.StatusOK {
		t.Fatalf("single status = %d: %s", singleW.Code, singleW.Body.String())
	}
	var singleBody struct {
		Action NextActionProjection `json:"action"`
	}
	if err := json.NewDecoder(singleW.Body).Decode(&singleBody); err != nil {
		t.Fatal(err)
	}
	if singleBody.Action.ID != NextActionRun {
		t.Fatalf("single action = %q", singleBody.Action.ID)
	}

	tooMany := strings.Repeat("idea/ready,", maxNextActionBatch+1)
	items := strings.Split(strings.TrimSuffix(tooMany, ","), ",")
	body, _ := json.Marshal(map[string]any{"items": items})
	batch := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/next-actions", strings.NewReader(string(body)))
	batchW := httptest.NewRecorder()
	router.ServeHTTP(batchW, batch)
	if batchW.Code != http.StatusBadRequest {
		t.Fatalf("batch bound status = %d: %s", batchW.Code, batchW.Body.String())
	}

	body, _ = json.Marshal(map[string]any{"items": []string{"idea/ready", "idea/missing", "bad"}})
	batch = httptest.NewRequest(http.MethodPost, "/api/v1/backlog/next-actions", strings.NewReader(string(body)))
	batchW = httptest.NewRecorder()
	router.ServeHTTP(batchW, batch)
	if batchW.Code != http.StatusOK {
		t.Fatalf("batch status = %d: %s", batchW.Code, batchW.Body.String())
	}
	var batchBody struct {
		Results []nextActionBatchResult `json:"results"`
	}
	if err := json.NewDecoder(batchW.Body).Decode(&batchBody); err != nil {
		t.Fatal(err)
	}
	if len(batchBody.Results) != 3 || batchBody.Results[0].Action == nil || batchBody.Results[0].Action.ID != NextActionRun {
		t.Fatalf("unexpected batch results: %+v", batchBody.Results)
	}
	if batchBody.Results[1].Error == "" || batchBody.Results[2].Error == "" {
		t.Fatalf("mixed batch must preserve per-item errors: %+v", batchBody.Results)
	}
}

func TestResolveNextActionRejectsUnmappedBlockerCode(t *testing.T) {
	h, root := setupTestHandler(t)
	h.SetExecutionQueuer(&mockExecutionQueuer{preflightResult: execution.ProcessPreflight{
		BlockingReasons: []string{"a new blocker"},
		BlockingDetails: []execution.ProcessBlockingReason{{Code: "unknown", Message: "a new blocker"}},
	}})
	item := BacklogItem{Name: "unknown-blocker", Kind: KindIdea, Status: StatusReady, PlanRef: testPlanRef("unknown-blocker"), AcceptanceCriteria: testAcceptanceCriteria()}
	createTestItem(t, root, item.Kind, item)
	if _, err := h.ResolveNextAction(t.Context(), item); err == nil {
		t.Fatal("ResolveNextAction accepted an unmapped blocker code")
	}
}

// [REQ:SWM-P0-010] Every execution blocker that can reach the shared
// preflight response must resolve to an explicit, enabled operator action.
// This is intentionally a closed table: adding a new code requires choosing
// its action here and in ResolveNextAction rather than silently falling back
// to an ambiguous generic state.
func TestResolveNextActionCoversCanonicalBlockerCodes(t *testing.T) {
	tests := []struct {
		code string
		want NextActionID
	}{
		{code: "plan_changed", want: NextActionAcceptPlan},
		{code: "plan_not_accepted", want: NextActionAcceptPlan},
		{code: "plan_invalid", want: NextActionRepairPlan},
		{code: "unmet_dependencies", want: NextActionResolveDependencies},
		{code: "queue_cap", want: NextActionRun},
		{code: "cost_cap", want: NextActionRun},
		{code: "circuit_open", want: NextActionRun},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			h, root := setupTestHandler(t)
			h.SetExecutionQueuer(&mockExecutionQueuer{preflightResult: execution.ProcessPreflight{
				BlockingReasons: []string{tt.code},
				BlockingDetails: []execution.ProcessBlockingReason{{Code: tt.code, Message: tt.code}},
			}})
			item := BacklogItem{Name: tt.code, Kind: KindIdea, Status: StatusReady, PlanRef: testPlanRef(tt.code), AcceptanceCriteria: testAcceptanceCriteria()}
			if tt.code == "unmet_dependencies" {
				item.DependsOn = []string{"idea/upstream"}
				createTestItem(t, root, KindIdea, BacklogItem{Name: "upstream", Kind: KindIdea, Status: StatusBacklog})
			}
			createTestItem(t, root, item.Kind, item)
			action, err := h.ResolveNextAction(t.Context(), item)
			if err != nil {
				t.Fatal(err)
			}
			if action.ID != tt.want || !action.Enabled {
				t.Fatalf("action = %#v, want enabled %q", action, tt.want)
			}
		})
	}
}

func testAcceptanceCriteria() []Criterion {
	return []Criterion{{ID: "criterion-1", Gherkin: "Given a valid plan When it executes Then it has an independently reviewable outcome."}}
}

func TestResolveNextActionDispatchesPersistedFollowUp(t *testing.T) {
	h, _ := setupTestHandler(t)
	item := BacklogItem{Name: "follow-up", Kind: KindExecute, Status: StatusNeedsFollowup, PendingFollowUp: &FollowUp{Steering: "Repair the regression.", Disposition: FollowUpReplan}}
	action, err := h.ResolveNextAction(t.Context(), item)
	if err != nil {
		t.Fatal(err)
	}
	if action.ID != NextActionDispatchFollowup || action.FollowUp == nil || action.FollowUp.Disposition != FollowUpReplan {
		t.Fatalf("action = %#v", action)
	}
}

func TestResolveNextActionPrioritizesOpenDecisions(t *testing.T) {
	h, root := setupTestHandler(t)
	h.SetExecutionQueuer(&mockExecutionQueuer{preflightResult: execution.ProcessPreflight{Ready: true}})
	h.SetDecisionCountProvider(fixedDecisionCount{counts: map[string]int{"idea/proposal": 2}})
	item := BacklogItem{Name: "proposal", Kind: KindIdea, Status: StatusReady, PlanRef: testPlanRef("proposal")}
	createTestItem(t, root, item.Kind, item)
	action, err := h.ResolveNextAction(t.Context(), item)
	if err != nil {
		t.Fatal(err)
	}
	if action.ID != NextActionDecide || !action.Enabled || !strings.Contains(action.Reason, "2") {
		t.Fatalf("action = %#v", action)
	}
}

func TestDeclaredNextActionTransitionKeys(t *testing.T) {
	for action, want := range map[NextActionID]string{NextActionAuthorPlan: "plan.author", NextActionRepairPlan: "plan.repair", NextActionDispatchFollowup: "follow_up.dispatch"} {
		if got := TransitionKeyForNextAction(action); got != want {
			t.Fatalf("%s transition = %q, want %q", action, got, want)
		}
	}
}
