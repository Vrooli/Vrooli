package planclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/plans"
	plansconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/plans/plans_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

type fakePlansService struct {
	plansconnect.UnimplementedPlansServiceHandler
	gotList         bool
	gotGetID        string
	gotRenderID     string
	gotCompact      bool
	gotImportSource string
	gotImportSlug   string
	gotAuditRunID   string
}

func (f *fakePlansService) ListPlans(_ context.Context, _ *connect.Request[plansv1.ListPlansRequest]) (*connect.Response[plansv1.ListPlansResponse], error) {
	f.gotList = true
	return connect.NewResponse(&plansv1.ListPlansResponse{Plans: []*sharedv1.Plan{
		{Id: "plan-1", Slug: "first-plan", Title: "First plan"},
	}}), nil
}

func (f *fakePlansService) GetPlan(_ context.Context, req *connect.Request[plansv1.GetPlanRequest]) (*connect.Response[plansv1.GetPlanResponse], error) {
	f.gotGetID = req.Msg.GetId()
	return connect.NewResponse(&plansv1.GetPlanResponse{Plan: &sharedv1.Plan{
		Id:   req.Msg.GetId(),
		Slug: "resolved-plan",
	}}), nil
}

func (f *fakePlansService) RenderMarkdown(_ context.Context, req *connect.Request[plansv1.RenderMarkdownRequest]) (*connect.Response[plansv1.RenderMarkdownResponse], error) {
	f.gotRenderID = req.Msg.GetId()
	f.gotCompact = req.Msg.GetCompact()
	return connect.NewResponse(&plansv1.RenderMarkdownResponse{
		Markdown:        "# Rendered",
		QualityStatus:   "clean",
		QualityFindings: []string{"none"},
		Plan:            &sharedv1.Plan{Id: req.Msg.GetId(), Slug: "rendered-plan"},
	}), nil
}

func (f *fakePlansService) ImportPlan(_ context.Context, req *connect.Request[plansv1.ImportPlanRequest]) (*connect.Response[plansv1.ImportPlanResponse], error) {
	f.gotImportSource = req.Msg.GetSourcePath()
	f.gotImportSlug = req.Msg.GetSlug()
	return connect.NewResponse(&plansv1.ImportPlanResponse{Plan: &sharedv1.Plan{
		Id:   "imported-id",
		Slug: req.Msg.GetSlug(),
	}}), nil
}

func (f *fakePlansService) ListAuditFacts(_ context.Context, req *connect.Request[plansv1.ListAuditFactsRequest]) (*connect.Response[plansv1.ListAuditFactsResponse], error) {
	f.gotAuditRunID = req.Msg.GetRunId()
	return connect.NewResponse(&plansv1.ListAuditFactsResponse{Facts: []*plansv1.PlanAuditFact{{
		EventId:       "plan-1:created:1",
		RunId:         req.Msg.GetRunId(),
		TaskId:        "task-1",
		Action:        "plan.created",
		PlanId:        "plan-1",
		ContentDigest: "sha256:plan-1",
		OccurredAt:    "2026-07-12T20:00:00Z",
	}}}), nil
}

// [REQ:REQ-P1-011-CANONICAL-LEDGER]
func TestConnectClient_PlansCallsResolvePerCall(t *testing.T) {
	service := &fakePlansService{}
	path, handler := plansconnect.NewPlansServiceHandler(service)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path[:len(path)] != path {
			t.Fatalf("unexpected request path %q, want prefix %q", r.URL.Path, path)
		}
		handler.ServeHTTP(w, r)
	}))
	defer server.Close()

	resolveCalls := 0
	client := NewConnectClient(server.Client(), func(context.Context) (string, error) {
		resolveCalls++
		return server.URL, nil
	})

	plans, err := client.ListPlans(context.Background())
	if err != nil {
		t.Fatalf("ListPlans: %v", err)
	}
	if len(plans) != 1 || plans[0].GetSlug() != "first-plan" || !service.gotList {
		t.Fatalf("ListPlans mismatch plans=%+v gotList=%v", plans, service.gotList)
	}

	plan, err := client.GetPlan(context.Background(), "plan-1")
	if err != nil {
		t.Fatalf("GetPlan: %v", err)
	}
	if plan.GetSlug() != "resolved-plan" || service.gotGetID != "plan-1" {
		t.Fatalf("GetPlan mismatch plan=%+v gotID=%q", plan, service.gotGetID)
	}

	rendered, err := client.RenderMarkdown(context.Background(), "plan-1", true)
	if err != nil {
		t.Fatalf("RenderMarkdown: %v", err)
	}
	if rendered.Markdown != "# Rendered" || rendered.QualityStatus != "clean" || service.gotRenderID != "plan-1" || !service.gotCompact {
		t.Fatalf("RenderMarkdown mismatch result=%+v service=%+v", rendered, service)
	}

	imported, err := client.ImportPlan(context.Background(), ImportPlanInput{SourcePath: "/tmp/external-plan.markdown", Slug: "new-plan"})
	if err != nil {
		t.Fatalf("ImportPlan: %v", err)
	}
	if imported.GetId() != "imported-id" || service.gotImportSource != "/tmp/external-plan.markdown" || service.gotImportSlug != "new-plan" {
		t.Fatalf("ImportPlan mismatch plan=%+v service=%+v", imported, service)
	}

	auditFacts, err := client.ListAuditFacts(context.Background(), "run-1")
	if err != nil {
		t.Fatalf("ListAuditFacts: %v", err)
	}
	if len(auditFacts) != 1 || service.gotAuditRunID != "run-1" || auditFacts[0].Action != "plan.created" || !auditFacts[0].OccurredAt.Equal(time.Date(2026, 7, 12, 20, 0, 0, 0, time.UTC)) {
		t.Fatalf("ListAuditFacts mismatch facts=%+v run_id=%q", auditFacts, service.gotAuditRunID)
	}
	if resolveCalls != 5 {
		t.Fatalf("resolver calls = %d, want 5", resolveCalls)
	}
}
