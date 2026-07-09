package planclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

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
	if resolveCalls != 4 {
		t.Fatalf("resolver calls = %d, want 4", resolveCalls)
	}
}
