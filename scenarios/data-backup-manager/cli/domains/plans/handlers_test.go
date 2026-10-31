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

	testutil "github.com/vrooli/cli-core/cliapptest"
)

// stubPlansService is an in-test PlansServiceHandler that records the request
// it receives and returns a canned plan.
type stubPlansService struct {
	gotCreate *plansv1.CreatePlanRequest
	gotUpdate *plansv1.UpdatePlanRequest
	current   *plansv1.Plan
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
	if s.current != nil {
		return connect.NewResponse(&plansv1.GetPlanResponse{Plan: s.current}), nil
	}
	return connect.NewResponse(&plansv1.GetPlanResponse{Plan: &plansv1.Plan{Id: req.Msg.Id}}), nil
}

func (s *stubPlansService) ListPlans(_ context.Context, _ *connect.Request[plansv1.ListPlansRequest]) (*connect.Response[plansv1.ListPlansResponse], error) {
	return connect.NewResponse(&plansv1.ListPlansResponse{}), nil
}

func (s *stubPlansService) UpdatePlan(_ context.Context, req *connect.Request[plansv1.UpdatePlanRequest]) (*connect.Response[plansv1.UpdatePlanResponse], error) {
	s.gotUpdate = req.Msg
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

func updateSchema() cliapp.ArgSchema {
	return cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "id"},
		{Name: "name"},
		{Name: "targets"},
		{Name: "destinations"},
		{Name: "schedule"},
		{Name: "keep-latest"},
		{Name: "enabled"},
		{Name: "protection-tier"},
		{Name: "recovery-drill-schedule"},
		{Name: "allow-incomplete-coverage", Bool: true},
	}}
}

func existingPlan() *plansv1.Plan {
	return &plansv1.Plan{
		Id:                    "plan-1",
		Name:                  "nightly",
		TargetIds:             []string{"tgt-1", "tgt-2"},
		DestinationIds:        []string{"dst-1"},
		Schedule:              "0 4 * * *",
		Retention:             &plansv1.RetentionPolicy{KeepLatest: 3},
		Enabled:               true,
		ProtectionTier:        plansv1.ProtectionTier_PROTECTION_TIER_CRITICAL_PRIMARY,
		RecoveryDrillSchedule: "168h",
	}
}

func runUpdate(t *testing.T, flags map[string]string) *plansv1.UpdatePlanRequest {
	t.Helper()
	stub := &stubPlansService{current: existingPlan()}
	mux := http.NewServeMux()
	path, h := plansconnect.NewPlansServiceHandler(stub)
	mux.Handle(path, h)
	app := testutil.NewTestApp(t, mux)
	ctx, _ := cliapptest.NewCapturedRunContext(app, updateSchema(), cliapptest.TestRunContextOptions{Flags: flags})
	if err := newHandlers(app).update(ctx); err != nil {
		t.Fatalf("update: %v", err)
	}
	if stub.gotUpdate == nil {
		t.Fatal("server did not receive UpdatePlan")
	}
	return stub.gotUpdate
}

// TestUpdatePlanKeepsOmittedFields proves `plans update` is a partial update:
// changing only keep-latest must not disable the plan, clear its drill
// schedule, reset its tier, or drop its targets.
func TestUpdatePlanKeepsOmittedFields(t *testing.T) {
	got := runUpdate(t, map[string]string{"id": "plan-1", "keep-latest": "7"})
	want := existingPlan()
	if got.Retention.GetKeepLatest() != 7 {
		t.Fatalf("keep-latest = %d, want 7", got.Retention.GetKeepLatest())
	}
	if got.Name != want.Name || got.Schedule != want.Schedule || !got.Enabled ||
		got.ProtectionTier != want.ProtectionTier || got.RecoveryDrillSchedule != want.RecoveryDrillSchedule ||
		strings.Join(got.TargetIds, ",") != "tgt-1,tgt-2" || strings.Join(got.DestinationIds, ",") != "dst-1" {
		t.Fatalf("omitted fields changed: %+v", got)
	}
}

// TestUpdatePlanAppliesSuppliedFields proves supplied flags still replace,
// including an explicit empty drill schedule and an explicit disable.
func TestUpdatePlanAppliesSuppliedFields(t *testing.T) {
	got := runUpdate(t, map[string]string{
		"id": "plan-1", "enabled": "false", "recovery-drill-schedule": "", "targets": "tgt-9",
	})
	if got.Enabled {
		t.Fatal("explicit --enabled false was not applied")
	}
	if got.RecoveryDrillSchedule != "" {
		t.Fatalf("explicit empty drill schedule not applied: %q", got.RecoveryDrillSchedule)
	}
	if strings.Join(got.TargetIds, ",") != "tgt-9" {
		t.Fatalf("targets = %v, want [tgt-9]", got.TargetIds)
	}
	if got.Retention.GetKeepLatest() != 3 {
		t.Fatalf("omitted keep-latest changed to %d", got.Retention.GetKeepLatest())
	}
}
