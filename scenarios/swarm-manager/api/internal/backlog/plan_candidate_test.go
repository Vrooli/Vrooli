package backlog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"swarm-manager/internal/planclient"

	"github.com/gorilla/mux"
	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/plans"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

type candidateApplyClient struct {
	acceptancePlanClient
	gotID, gotHash string
	acknowledged   bool
}

func (*candidateApplyClient) CreateCandidateRevision(context.Context, planclient.CandidateRevisionInput) (*plansv1.CandidateRevision, error) {
	return &plansv1.CandidateRevision{Id: "candidate-1"}, nil
}

func (*candidateApplyClient) PreviewCandidateRevision(context.Context, string) (*plansv1.CandidateRevisionPreview, error) {
	return &plansv1.CandidateRevisionPreview{}, nil
}

func (c *candidateApplyClient) ApplyCandidateRevision(_ context.Context, id, hash string, acknowledged bool) (*plansv1.ApplyCandidateRevisionResponse, error) {
	c.gotID, c.gotHash, c.acknowledged = id, hash, acknowledged
	return &plansv1.ApplyCandidateRevisionResponse{Candidate: &plansv1.CandidateRevision{Id: id}, Plan: c.plan}, nil
}

// [REQ:SWM-P0-002] one evolving plan per item revised in place via candidate apply
func TestApplyPlanCandidateClearsAcceptanceAfterPlanManagerApply(t *testing.T) {
	h, root := setupTestHandler(t)
	item := BacklogItem{Name: "candidate", Title: "Candidate", Status: StatusBacklog, Priority: 5, PlanRef: &PlanRef{Provider: PlanRefProviderPlanManager, PlanID: "plan-1", Role: PlanRefRoleExecutionSpec}, PlanAcceptance: &PlanAcceptance{Actor: "operator", PlanContentHash: "sha256:base", SubjectVersion: "subject"}}
	createTestItem(t, root, KindExecute, item)
	client := &candidateApplyClient{acceptancePlanClient: acceptancePlanClient{plan: &sharedv1.Plan{Id: "plan-1", ContentHash: "sha256:base"}}}
	h.SetPlanClient(client)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/execute/candidate/plan-candidates/candidate-1/apply", strings.NewReader(`{"acknowledge_quality_impact":true}`))
	req = mux.SetURLVars(req, map[string]string{"kind": "execute", "name": "candidate", "candidateID": "candidate-1"})
	response := httptest.NewRecorder()
	h.ApplyPlanCandidate(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if client.gotID != "candidate-1" || client.gotHash != "sha256:base" || !client.acknowledged {
		t.Fatalf("apply request = id=%q hash=%q acknowledged=%t", client.gotID, client.gotHash, client.acknowledged)
	}
	saved, err := h.Store().LoadItem(KindExecute, item.Name)
	if err != nil {
		t.Fatal(err)
	}
	if saved.PlanAcceptance != nil {
		t.Fatalf("candidate application retained acceptance: %#v", saved.PlanAcceptance)
	}
}

var _ planclient.CandidateClient = (*candidateApplyClient)(nil)
