package plans

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"

	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/plans"
	plansconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/plans/plans_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	"data-backup-manager/cli/internal/testutil"
)

// stubPlansService is an in-test PlansServiceHandler that records the request
// it receives and returns a canned plan.
type stubPlansService struct {
	gotCreate *plansv1.CreatePlanRequest
}

func (s *stubPlansService) CreatePlan(_ context.Context, req *connect.Request[plansv1.CreatePlanRequest]) (*connect.Response[plansv1.CreatePlanResponse], error) {
	s.gotCreate = req.Msg
	return connect.NewResponse(&plansv1.CreatePlanResponse{
		Plan: &plansv1.Plan{
			Id:             "plan-1",
			Name:           req.Msg.Name,
			TargetIds:      req.Msg.TargetIds,
			DestinationIds: req.Msg.DestinationIds,
		},
	}), nil
}

func (s *stubPlansService) GetPlan(_ context.Context, req *connect.Request[plansv1.GetPlanRequest]) (*connect.Response[plansv1.GetPlanResponse], error) {
	return connect.NewResponse(&plansv1.GetPlanResponse{Plan: &plansv1.Plan{Id: req.Msg.Id}}), nil
}

func (s *stubPlansService) ListPlans(_ context.Context, _ *connect.Request[plansv1.ListPlansRequest]) (*connect.Response[plansv1.ListPlansResponse], error) {
	return connect.NewResponse(&plansv1.ListPlansResponse{}), nil
}

func (s *stubPlansService) UpdatePlan(_ context.Context, req *connect.Request[plansv1.UpdatePlanRequest]) (*connect.Response[plansv1.UpdatePlanResponse], error) {
	return connect.NewResponse(&plansv1.UpdatePlanResponse{Plan: &plansv1.Plan{Id: req.Msg.Id}}), nil
}

func (s *stubPlansService) DeletePlan(_ context.Context, _ *connect.Request[plansv1.DeletePlanRequest]) (*connect.Response[plansv1.DeletePlanResponse], error) {
	return connect.NewResponse(&plansv1.DeletePlanResponse{Removed: true}), nil
}

// TestCreatePlanCommand is the per-domain check: the create command parses its
// flags (including comma-separated target/destination ids) and calls the generated
// PlansService client against a real Connect server.
func TestCreatePlanCommand(t *testing.T) {
	stub := &stubPlansService{}
	mux := http.NewServeMux()
	path, h := plansconnect.NewPlansServiceHandler(stub)
	mux.Handle(path, h)
	app := testutil.NewTestApp(t, mux)

	hs := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "name"},
		{Name: "targets"},
		{Name: "destinations"},
		{Name: "schedule"},
		{Name: "keep-latest"},
		{Name: "enabled"},
		{Name: "protection-tier"},
		{Name: "allow-incomplete-coverage", Bool: true},
	}}
	ctx, stdout := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{
			"name":         "daily-backup",
			"targets":      "tgt-1,tgt-2",
			"destinations": "dst-1",
		},
	})

	if err := hs.create(ctx); err != nil {
		t.Fatalf("create: %v", err)
	}
	out := stdout.String()

	if stub.gotCreate == nil {
		t.Fatal("server did not receive CreatePlan")
	}
	if stub.gotCreate.Name != "daily-backup" {
		t.Fatalf("name wrong: %q", stub.gotCreate.Name)
	}
	if len(stub.gotCreate.TargetIds) != 2 {
		t.Fatalf("expected 2 target ids, got %v", stub.gotCreate.TargetIds)
	}
	if len(stub.gotCreate.DestinationIds) != 1 || stub.gotCreate.DestinationIds[0] != "dst-1" {
		t.Fatalf("destination ids wrong: %v", stub.gotCreate.DestinationIds)
	}
	if !strings.Contains(out, "plan-1") {
		t.Fatalf("output missing plan id: %q", out)
	}
}

// TestRegisterPlansLoadsFromManifest proves the manifest wiring produces the
// expected subcommands.
func TestRegisterPlansLoadsFromManifest(t *testing.T) {
	manifest := readManifest(t)
	app := &cliapp.ScenarioApp{}
	group, err := Register(app, manifest)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	got := map[string]bool{}
	for _, c := range group.Subcommands {
		got[c.Name] = true
	}
	for _, want := range []string{"create", "get", "list", "update", "delete"} {
		if !got[want] {
			t.Errorf("missing subcommand %q", want)
		}
	}
}
