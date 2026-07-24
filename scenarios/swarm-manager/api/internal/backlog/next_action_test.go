package backlog

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"swarm-manager/internal/execution"

	"github.com/gorilla/mux"
)

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
		{name: "ready item runs", item: BacklogItem{Name: "ready", Kind: KindIdea, Status: StatusReady, PlanRef: testPlanRef("ready")}, preflight: execution.ProcessPreflight{Ready: true}, want: NextActionRun},
		{name: "unaccepted plan is accepted", item: BacklogItem{Name: "unaccepted", Kind: KindIdea, Status: StatusReady, PlanRef: testPlanRef("unaccepted")}, preflight: execution.ProcessPreflight{BlockingReasons: []string{"canonical plan has not been explicitly accepted — accept the current plan revision before queueing"}}, want: NextActionAcceptPlan},
		{name: "invalid plan is repaired", item: BacklogItem{Name: "invalid", Kind: KindIdea, Status: StatusReady, PlanRef: testPlanRef("invalid")}, preflight: execution.ProcessPreflight{BlockingReasons: []string{"canonical plan is not valid: quality status is \"fail\""}}, want: NextActionRepairPlan},
		{name: "dependencies are resolved", item: BacklogItem{Name: "blocked", Kind: KindIdea, Status: StatusReady, PlanRef: testPlanRef("blocked"), DependsOn: []string{"idea/upstream"}}, preflight: execution.ProcessPreflight{Ready: true}, want: NextActionResolveDependencies},
		{name: "review is reviewed", item: BacklogItem{Name: "review", Kind: KindIdea, Status: StatusReviewPending}, want: NextActionReview},
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

func TestNextActionEndpointsSingleAndBoundedBatch(t *testing.T) {
	h, root := setupTestHandler(t)
	h.SetExecutionQueuer(&mockExecutionQueuer{preflightResult: execution.ProcessPreflight{Ready: true}})
	createReadyTestItem(t, root, KindIdea, BacklogItem{Name: "ready", Kind: KindIdea, Status: StatusReady})
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
