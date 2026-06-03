package runs

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"

	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/runs"
	runsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/runs/runs_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	"data-backup-manager/cli/internal/testutil"
)

// stubRunsService is an in-test RunsServiceHandler that records the request it
// receives and returns canned responses.
type stubRunsService struct {
	gotTrigger *runsv1.TriggerRunRequest
}

func (s *stubRunsService) TriggerRun(_ context.Context, req *connect.Request[runsv1.TriggerRunRequest]) (*connect.Response[runsv1.TriggerRunResponse], error) {
	s.gotTrigger = req.Msg
	return connect.NewResponse(&runsv1.TriggerRunResponse{
		Run: &runsv1.Run{Id: "run-1", PlanId: req.Msg.PlanId, Trigger: runsv1.TriggerSource_TRIGGER_SOURCE_MANUAL},
	}), nil
}

func (s *stubRunsService) GetRun(_ context.Context, req *connect.Request[runsv1.GetRunRequest]) (*connect.Response[runsv1.GetRunResponse], error) {
	return connect.NewResponse(&runsv1.GetRunResponse{Run: &runsv1.Run{Id: req.Msg.Id}}), nil
}

func (s *stubRunsService) ListRuns(_ context.Context, _ *connect.Request[runsv1.ListRunsRequest]) (*connect.Response[runsv1.ListRunsResponse], error) {
	return connect.NewResponse(&runsv1.ListRunsResponse{}), nil
}

func (s *stubRunsService) ListTargetStatus(_ context.Context, _ *connect.Request[runsv1.ListTargetStatusRequest]) (*connect.Response[runsv1.ListTargetStatusResponse], error) {
	return connect.NewResponse(&runsv1.ListTargetStatusResponse{}), nil
}

func (s *stubRunsService) BrowseSnapshot(_ context.Context, _ *connect.Request[runsv1.BrowseSnapshotRequest]) (*connect.Response[runsv1.BrowseSnapshotResponse], error) {
	return connect.NewResponse(&runsv1.BrowseSnapshotResponse{}), nil
}

func (s *stubRunsService) GetRunStats(_ context.Context, _ *connect.Request[runsv1.GetRunStatsRequest]) (*connect.Response[runsv1.GetRunStatsResponse], error) {
	return connect.NewResponse(&runsv1.GetRunStatsResponse{Stats: &runsv1.RunStats{}}), nil
}

// TestTriggerRunCommand is the per-domain check: the trigger command parses its
// flags and calls the generated RunsService client against a real Connect server.
func TestTriggerRunCommand(t *testing.T) {
	stub := &stubRunsService{}
	mux := http.NewServeMux()
	path, h := runsconnect.NewRunsServiceHandler(stub)
	mux.Handle(path, h)
	app := testutil.NewTestApp(t, mux)

	hs := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "plan"},
	}}
	ctx, stdout := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{
			"plan": "plan-42",
		},
	})

	if err := hs.trigger(ctx); err != nil {
		t.Fatalf("trigger: %v", err)
	}
	out := stdout.String()

	if stub.gotTrigger == nil {
		t.Fatal("server did not receive TriggerRun")
	}
	if stub.gotTrigger.PlanId != "plan-42" {
		t.Fatalf("plan_id wrong: %q", stub.gotTrigger.PlanId)
	}
	if !strings.Contains(out, "run-1") {
		t.Fatalf("output missing run id: %q", out)
	}
}

// TestRegisterRunsLoadsFromManifest proves the manifest wiring produces the
// expected subcommands.
func TestRegisterRunsLoadsFromManifest(t *testing.T) {
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
	for _, want := range []string{"trigger", "get", "list", "status", "browse", "stats"} {
		if !got[want] {
			t.Errorf("missing subcommand %q", want)
		}
	}
}
