package plans

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/demand"
	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/plans"
	plansconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/plans/plans_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

type planDemandStub struct {
	acquired []demand.AcquireRequest
	released []string
}

func (s *planDemandStub) Acquire(_ context.Context, req demand.AcquireRequest) (demand.Lease, error) {
	s.acquired = append(s.acquired, req)
	return demand.Lease{LeaseID: req.LeaseID}, nil
}

func (s *planDemandStub) Renew(context.Context, string, time.Duration) (demand.Lease, error) {
	return demand.Lease{}, nil
}

func (s *planDemandStub) Release(_ context.Context, leaseID, _ string) (demand.Lease, error) {
	s.released = append(s.released, leaseID)
	return demand.Lease{LeaseID: leaseID}, nil
}

type planManagerStub struct {
	plansconnect.UnimplementedPlansServiceHandler
	listReq   *plansv1.ListPlansRequest
	getReq    *plansv1.GetPlanRequest
	renderReq *plansv1.RenderMarkdownRequest
	err       error
}

func (s *planManagerStub) ListPlans(_ context.Context, req *connect.Request[plansv1.ListPlansRequest]) (*connect.Response[plansv1.ListPlansResponse], error) {
	s.listReq = req.Msg
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&plansv1.ListPlansResponse{Plans: []*sharedv1.Plan{{Id: "p1", Title: "Plan", Slug: "plan", Mirror: &sharedv1.RenderedPlanMirror{Path: "/plans/plan.md"}}}}), nil
}

func (s *planManagerStub) GetPlan(_ context.Context, req *connect.Request[plansv1.GetPlanRequest]) (*connect.Response[plansv1.GetPlanResponse], error) {
	s.getReq = req.Msg
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&plansv1.GetPlanResponse{Plan: &sharedv1.Plan{Id: "p2", Slug: "second"}}), nil
}

func (s *planManagerStub) RenderMarkdown(_ context.Context, req *connect.Request[plansv1.RenderMarkdownRequest]) (*connect.Response[plansv1.RenderMarkdownResponse], error) {
	s.renderReq = req.Msg
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&plansv1.RenderMarkdownResponse{Markdown: "# Rendered", Mirror: &sharedv1.RenderedPlanMirror{Path: "/plans/rendered.md"}}), nil
}

func servePlanManager(t *testing.T, service plansconnect.PlansServiceHandler) *httptest.Server {
	t.Helper()
	path, handler := plansconnect.NewPlansServiceHandler(service)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func TestPlanManagerConnectClientUsesGeneratedMethodsAndPreservesWorkspaceScope(t *testing.T) {
	service := &planManagerStub{}
	server := servePlanManager(t, service)
	client := ConnectPlanManagerClient{Client: server.Client(), BaseURL: server.URL}
	ctx := context.Background()
	workspace := WorkspaceScope{ID: "workspace-1", Root: "/repo"}

	listed, err := client.ListPlans(ctx, workspace, true)
	if err != nil {
		t.Fatalf("ListPlans: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != "p1" || listed[0].Path != "/plans/plan.md" {
		t.Fatalf("ListPlans projection = %#v", listed)
	}
	if service.listReq == nil || !service.listReq.GetIncludeArchived() || service.listReq.GetWorkspace().GetId() != "workspace-1" || service.listReq.GetWorkspace().GetRoot() != "/repo" {
		t.Fatalf("ListPlans request = %#v", service.listReq)
	}

	got, err := client.GetPlan(ctx, workspace, "second")
	if err != nil {
		t.Fatalf("GetPlan: %v", err)
	}
	if got.ID != "p2" || service.getReq.GetId() != "second" || service.getReq.GetWorkspace().GetRoot() != "/repo" {
		t.Fatalf("GetPlan projection/request = %#v / %#v", got, service.getReq)
	}

	rendered, err := client.RenderMarkdown(ctx, workspace, "preview")
	if err != nil {
		t.Fatalf("RenderMarkdown: %v", err)
	}
	if rendered.Content != "# Rendered" || rendered.Plan.ID != "preview" || rendered.Plan.Path != "/plans/rendered.md" {
		t.Fatalf("RenderMarkdown projection = %#v", rendered)
	}
	if service.renderReq.GetId() != "preview" || service.renderReq.GetWorkspace().GetId() != "workspace-1" {
		t.Fatalf("RenderMarkdown request = %#v", service.renderReq)
	}
}

func TestDefaultPlanManagerClientResolvesCurrentEndpointForEveryRPC(t *testing.T) {
	firstOwner := &planManagerStub{}
	firstServer := servePlanManager(t, firstOwner)
	secondOwner := &planManagerStub{}
	secondServer := servePlanManager(t, secondOwner)

	currentURL := firstServer.URL
	resolutions := 0
	client, err := newDiscoveredPlanManagerClient(context.Background(), func(_ context.Context, scenario string) (string, error) {
		if scenario != planManagerScenario {
			t.Fatalf("resolved scenario = %q, want %q", scenario, planManagerScenario)
		}
		resolutions++
		return currentURL, nil
	})
	if err != nil {
		t.Fatalf("construct default client: %v", err)
	}

	if _, err := client.GetPlan(context.Background(), WorkspaceScope{}, "first"); err != nil {
		t.Fatalf("first GetPlan: %v", err)
	}
	if firstOwner.getReq == nil {
		t.Fatal("initial endpoint did not receive request")
	}

	currentURL = secondServer.URL
	if _, err := client.GetPlan(context.Background(), WorkspaceScope{}, "after-restart"); err != nil {
		t.Fatalf("GetPlan after endpoint change: %v", err)
	}
	if secondOwner.getReq == nil || secondOwner.getReq.GetId() != "after-restart" {
		t.Fatalf("updated endpoint request = %#v", secondOwner.getReq)
	}
	if resolutions != 3 { // one constructor validation and one resolution per RPC
		t.Fatalf("resolution count = %d, want 3", resolutions)
	}
}

func TestDefaultPlanManagerClientClassifiesDiscoveryErrorsAsUnavailable(t *testing.T) {
	want := errors.New("discovery unavailable")
	_, err := newDiscoveredPlanManagerClient(context.Background(), func(context.Context, string) (string, error) {
		return "", want
	})
	if !errors.Is(err, ErrPlanManagerUnavailable) || !errors.Is(err, want) {
		t.Fatalf("constructor discovery error = %v, want unavailable and original cause", err)
	}

	client := ConnectPlanManagerClient{
		resolveURL: func(context.Context) (string, error) { return "", want },
	}
	if _, err := client.GetPlan(context.Background(), WorkspaceScope{}, "plan"); !errors.Is(err, ErrPlanManagerUnavailable) || !errors.Is(err, want) {
		t.Fatalf("per-RPC discovery error = %v, want unavailable and original cause", err)
	}
}

func TestPlanManagerConnectClientOwnsAndReleasesDemandLease(t *testing.T) {
	server := servePlanManager(t, &planManagerStub{})
	stub := &planDemandStub{}
	client := ConnectPlanManagerClient{
		Client:           server.Client(),
		BaseURL:          server.URL,
		Demand:           stub,
		DemandConsumerID: "test:plan-client",
	}
	if _, err := client.ListPlans(context.Background(), WorkspaceScope{}, false); err != nil {
		t.Fatalf("ListPlans: %v", err)
	}
	if len(stub.acquired) != 1 || len(stub.released) != 1 {
		t.Fatalf("demand lifecycle = acquired %d released %d", len(stub.acquired), len(stub.released))
	}
	req := stub.acquired[0]
	if req.Kind != demand.KindDependency || req.ConsumerID != "test:plan-client" {
		t.Fatalf("unexpected demand request: %+v", req)
	}
	if stub.released[0] != req.LeaseID {
		t.Fatalf("released lease %q, acquired %q", stub.released[0], req.LeaseID)
	}
}

func TestPlanManagerConnectErrorsKeepTypedFallbackClassification(t *testing.T) {
	service := &planManagerStub{err: connect.NewError(connect.CodeNotFound, context.Canceled)}
	server := servePlanManager(t, service)
	client := ConnectPlanManagerClient{Client: server.Client(), BaseURL: server.URL}
	_, err := client.GetPlan(context.Background(), WorkspaceScope{}, "missing")
	if err == nil || !errors.Is(err, ErrPlanManagerNotFound) {
		t.Fatalf("GetPlan error = %v, want not-found classification", err)
	}

	for _, test := range []struct {
		code connect.Code
		want error
	}{
		{code: connect.CodeNotFound, want: ErrPlanManagerNotFound},
		{code: connect.CodeInvalidArgument, want: ErrPlanManagerInvalid},
		{code: connect.CodeAlreadyExists, want: ErrPlanManagerConflict},
		{code: connect.CodeUnavailable, want: ErrPlanManagerUnavailable},
		{code: connect.CodeInternal, want: ErrPlanManagerServer},
	} {
		if got := classifyPlanManagerConnectError(connect.NewError(test.code, context.Canceled)); !errors.Is(got, test.want) {
			t.Errorf("classify %s = %v, want errors.Is(_, %v)", test.code, got, test.want)
		}
	}
}
