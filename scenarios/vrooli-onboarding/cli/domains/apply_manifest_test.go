package domains

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	applyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/apply"
	applyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/apply/applyv1connect"
)

func TestManifestApplyCommandsUseTypedRequests(t *testing.T) {
	manifest, err := os.ReadFile("../manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	service := &applyManifestService{}
	path, handler := applyconnect.NewApplyServiceHandler(service)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	core := cliapptest.NewTestApp(t, mux)
	_, groups, err := LoadManifestGroups(core, manifest)
	if err != nil {
		t.Fatal(err)
	}
	var group *cliapp.SubcommandGroup
	for i := range groups {
		if groups[i].Name == "apply" {
			group = &groups[i]
			break
		}
	}
	if group == nil {
		t.Fatal("manifest apply group is missing")
	}
	commands := make(map[string]cliapp.Command, len(group.Subcommands))
	for _, command := range group.Subcommands {
		commands[command.Name] = command
	}
	for _, name := range []string{"start", "status", "plan"} {
		if command, ok := commands[name]; !ok || command.RunCtx == nil {
			t.Fatalf("manifest apply %s command is not executable", name)
		}
	}
	cliapptest.AssertPrimitiveEvidence(t, commands["start"], cliapp.PrimitiveProtoMutation)
	cliapptest.AssertPrimitiveEvidence(t, commands["status"], cliapp.PrimitiveProtoList)
	cliapptest.AssertPrimitiveEvidence(t, commands["plan"], cliapp.PrimitiveProtoList)

	runCommand := func(name string, args ...string) {
		t.Helper()
		var stdout, stderr bytes.Buffer
		ctx, err := cliapp.NewTestRunContextFromArgs(commands[name].Args, args, core, &stdout, &stderr)
		if err != nil {
			t.Fatalf("build %s context: %v", name, err)
		}
		if err := commands[name].RunCtx(ctx); err != nil {
			t.Fatalf("run manifest apply %s: %v (stderr %q)", name, err, stderr.String())
		}
	}
	runCommand("start", "--target", "node-a", "--json")
	runCommand("status", "--target", "node-a", "--run-id", "run-123", "--json")
	runCommand("plan", "--target", "node-a", "--json")

	if service.startTarget != "node-a" {
		t.Fatalf("StartApply target = %q, want node-a", service.startTarget)
	}
	if service.runTarget != "node-a" || service.runID != "run-123" {
		t.Fatalf("GetApplyRun request = target %q, run %q", service.runTarget, service.runID)
	}
	if service.planTarget != "node-a" {
		t.Fatalf("GetApplyPlan target = %q, want node-a", service.planTarget)
	}
}

type applyManifestService struct {
	startTarget string
	runTarget   string
	runID       string
	planTarget  string
}

func (s *applyManifestService) StartApply(_ context.Context, req *connect.Request[applyv1.StartApplyRequest]) (*connect.Response[applyv1.StartApplyResponse], error) {
	s.startTarget = req.Msg.GetTarget()
	return connect.NewResponse(&applyv1.StartApplyResponse{Run: &applyv1.GetApplyRunResponse{RunId: "run-started", Status: applyv1.ApplyRunState_APPLY_RUN_STATE_APPLYING}}), nil
}

func (s *applyManifestService) GetApplyRun(_ context.Context, req *connect.Request[applyv1.GetApplyRunRequest]) (*connect.Response[applyv1.GetApplyRunResponse], error) {
	s.runTarget, s.runID = req.Msg.GetTarget(), req.Msg.GetRunId()
	return connect.NewResponse(&applyv1.GetApplyRunResponse{RunId: s.runID, Status: applyv1.ApplyRunState_APPLY_RUN_STATE_APPLIED}), nil
}

func (s *applyManifestService) GetApplyPlan(_ context.Context, req *connect.Request[applyv1.GetApplyPlanRequest]) (*connect.Response[applyv1.GetApplyPlanResponse], error) {
	s.planTarget = req.Msg.GetTarget()
	return connect.NewResponse(&applyv1.GetApplyPlanResponse{Target: s.planTarget, Items: []*applyv1.ApplyItem{{Kind: "scenario", Name: "demo", ObservedState: "missing"}}}), nil
}

func (*applyManifestService) ReviewApply(context.Context, *connect.Request[applyv1.ReviewApplyRequest]) (*connect.Response[applyv1.ReviewApplyResponse], error) {
	return connect.NewResponse(&applyv1.ReviewApplyResponse{}), nil
}

func (*applyManifestService) CancelApply(context.Context, *connect.Request[applyv1.CancelApplyRequest]) (*connect.Response[applyv1.CancelApplyResponse], error) {
	return connect.NewResponse(&applyv1.CancelApplyResponse{}), nil
}
