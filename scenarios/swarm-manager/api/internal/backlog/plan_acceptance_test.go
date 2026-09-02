package backlog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"swarm-manager/internal/planclient"

	"github.com/gorilla/mux"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

// acceptancePlanClient embeds the broad client interface but overrides only
// the read methods exercised by the acceptance boundary.
type acceptancePlanClient struct {
	planclient.Client
	plan *sharedv1.Plan
}

func (c acceptancePlanClient) GetPlan(context.Context, string) (*sharedv1.Plan, error) {
	return c.plan, nil
}

func (c acceptancePlanClient) RenderMarkdown(context.Context, string, bool) (planclient.RenderMarkdownResult, error) {
	return planclient.RenderMarkdownResult{Markdown: "# plan", Plan: c.plan, QualityStatus: "pass"}, nil
}

// [REQ:SWM-P0-004] acceptance gate: operator accept pins canonical plan
func TestAcceptPlanPinsCanonicalHashAndScopeVersion(t *testing.T) {
	h, root := setupTestHandler(t)
	item := BacklogItem{Name: "accept-me", Title: "Accept me", Description: "scope", Status: StatusBacklog, Priority: 5, PlanRef: &PlanRef{Provider: PlanRefProviderPlanManager, PlanID: "plan-1", Slug: "plan-1", Role: PlanRefRoleExecutionSpec}}
	createTestItem(t, root, KindExecute, item)
	// A finalized, never-started plan reports DRAFT because status is computed
	// from phase activity. First execution must still be accept-able.
	h.SetPlanClient(acceptancePlanClient{plan: &sharedv1.Plan{Id: "plan-1", Slug: "plan-1", ContentHash: "sha256:accepted", Status: sharedv1.PlanStatus_PLAN_STATUS_DRAFT}})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/execute/accept-me/plan-accept", strings.NewReader(`{}`))
	req = mux.SetURLVars(req, map[string]string{"kind": "execute", "name": "accept-me"})
	response := httptest.NewRecorder()
	h.AcceptPlan(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	saved, err := h.Store().LoadItem(KindExecute, "accept-me")
	if err != nil {
		t.Fatal(err)
	}
	if saved.PlanAcceptance == nil || saved.PlanAcceptance.Actor != "operator" || saved.PlanAcceptance.PlanContentHash != "sha256:accepted" || !PlanAcceptanceMatches(saved, "sha256:accepted") {
		t.Fatalf("saved acceptance = %#v", saved.PlanAcceptance)
	}
}

func TestAcceptPlanRejectsArchivedPlan(t *testing.T) {
	h, root := setupTestHandler(t)
	item := BacklogItem{Name: "archived", Title: "Archived", Status: StatusBacklog, Priority: 5, PlanRef: &PlanRef{Provider: PlanRefProviderPlanManager, PlanID: "plan-1", Slug: "plan-1", Role: PlanRefRoleExecutionSpec}}
	createTestItem(t, root, KindExecute, item)
	h.SetPlanClient(acceptancePlanClient{plan: &sharedv1.Plan{Id: "plan-1", Slug: "plan-1", ContentHash: "sha256:archived", Status: sharedv1.PlanStatus_PLAN_STATUS_ARCHIVED}})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/execute/archived/plan-accept", strings.NewReader(`{}`))
	req = mux.SetURLVars(req, map[string]string{"kind": "execute", "name": "archived"})
	response := httptest.NewRecorder()
	h.AcceptPlan(response, req)
	if response.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

// [REQ:SWM-P0-004] acceptance gate: acceptance cleared when plan/contract changes
func TestPlanAcceptanceIsClearedWhenWorkContractChanges(t *testing.T) {
	item := BacklogItem{
		Kind: KindFix, Name: "example", Title: "before",
		PlanRef:        &PlanRef{Provider: PlanRefProviderPlanManager, PlanID: "plan-1", Slug: "plan-1", Role: PlanRefRoleExecutionSpec},
		PlanAcceptance: &PlanAcceptance{Actor: "operator", AcceptedAt: "now", PlanContentHash: "sha256:one", SubjectVersion: "version"},
	}
	title := "after"
	ApplyItemPatch(&item, ItemPatch{Title: &title})
	if item.PlanAcceptance != nil {
		t.Fatalf("scope edit retained acceptance: %#v", item.PlanAcceptance)
	}
}
