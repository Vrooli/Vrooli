package runs

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs"
	runsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs/runs_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "vrooli-bridge/cli/internal/testutil"
)

// fakeRuns is a minimal RunsService for the CLI round-trip. The waitStatus knob
// controls the verdict WaitRun returns.
type fakeRuns struct {
	waitStatus runsv1.RunStatus
	aborted    string
}

func (f *fakeRuns) GetRun(_ context.Context, req *connect.Request[runsv1.GetRunRequest]) (*connect.Response[runsv1.GetRunResponse], error) {
	return connect.NewResponse(&runsv1.GetRunResponse{
		Run:    &runsv1.Run{Id: req.Msg.Id, NodeId: "n1", Verb: "scenario test", Status: runsv1.RunStatus_RUN_STATUS_PASSED},
		Events: []*channelv1.RunEvent{{RunId: req.Msg.Id, Kind: channelv1.RunEventKind_RUN_EVENT_KIND_LOG, LogChunk: "PASS\n"}},
	}), nil
}

func (f *fakeRuns) ListRuns(context.Context, *connect.Request[runsv1.ListRunsRequest]) (*connect.Response[runsv1.ListRunsResponse], error) {
	return connect.NewResponse(&runsv1.ListRunsResponse{Runs: []*runsv1.Run{
		{Id: "run-1", NodeId: "n1", Verb: "scenario test", Status: runsv1.RunStatus_RUN_STATUS_PASSED},
	}}), nil
}

func (f *fakeRuns) WaitRun(_ context.Context, req *connect.Request[runsv1.WaitRunRequest]) (*connect.Response[runsv1.WaitRunResponse], error) {
	return connect.NewResponse(&runsv1.WaitRunResponse{
		Run: &runsv1.Run{Id: req.Msg.Id, NodeId: "n1", Verb: "scenario test", Status: f.waitStatus},
	}), nil
}

func (f *fakeRuns) AbortRun(_ context.Context, req *connect.Request[runsv1.AbortRunRequest]) (*connect.Response[runsv1.AbortRunResponse], error) {
	f.aborted = req.Msg.Id
	return connect.NewResponse(&runsv1.AbortRunResponse{
		Run: &runsv1.Run{Id: req.Msg.Id, Status: runsv1.RunStatus_RUN_STATUS_ABORTED},
	}), nil
}

func (f *fakeRuns) StreamRunEvents(_ context.Context, _ *connect.Request[runsv1.StreamRunEventsRequest], stream *connect.ServerStream[runsv1.RunEventMessage]) error {
	return stream.Send(&runsv1.RunEventMessage{Event: &channelv1.RunEvent{Kind: channelv1.RunEventKind_RUN_EVENT_KIND_LOG, LogChunk: "streamed\n"}})
}

func (f *fakeRuns) ReportRunEvent(context.Context, *connect.Request[runsv1.ReportRunEventRequest]) (*connect.Response[runsv1.ReportRunEventResponse], error) {
	return connect.NewResponse(&runsv1.ReportRunEventResponse{Accepted: true}), nil
}

func connectAPI(svc runsconnect.RunsServiceHandler) http.Handler {
	path, handler := runsconnect.NewRunsServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

// [REQ:BRG-P0-005] get / list round-trip through the generated client.
func TestRuns_GetAndList(t *testing.T) {
	core := clitest.NewTestApp(t, connectAPI(&fakeRuns{}))
	h := newHandlers(core)

	idSchema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id"}}}
	getCtx, getOut := cliapptest.NewCapturedRunContext(core, idSchema, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"id": "run-1"},
	})
	require.NoError(t, h.get(getCtx))
	require.Contains(t, getOut.String(), "run-1")

	listCtx, listOut := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "node"}, {Name: "limit"}}}, cliapptest.TestRunContextOptions{})
	require.NoError(t, h.list(listCtx))
	require.Contains(t, listOut.String(), "run-1")
}

// [REQ:BRG-P0-005] wait returns nil on a passing run and an error on a
// non-passing one (so the process exits non-zero).
func TestRuns_WaitExitsByVerdict(t *testing.T) {
	waitSchema := cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "id"}},
		Flags:       []cliapp.Flag{{Name: "timeout"}},
	}

	passCore := clitest.NewTestApp(t, connectAPI(&fakeRuns{waitStatus: runsv1.RunStatus_RUN_STATUS_PASSED}))
	hp := newHandlers(passCore)
	passCtx, _ := cliapptest.NewCapturedRunContext(passCore, waitSchema, cliapptest.TestRunContextOptions{Positionals: map[string]string{"id": "run-1"}})
	require.NoError(t, hp.wait(passCtx), "a passing run exits zero")

	failCore := clitest.NewTestApp(t, connectAPI(&fakeRuns{waitStatus: runsv1.RunStatus_RUN_STATUS_FAILED}))
	hf := newHandlers(failCore)
	failCtx, _ := cliapptest.NewCapturedRunContext(failCore, waitSchema, cliapptest.TestRunContextOptions{Positionals: map[string]string{"id": "run-1"}})
	require.Error(t, hf.wait(failCtx), "a failing run exits non-zero")
}

// [REQ:BRG-P0-005] abort round-trips and the server records the id.
func TestRuns_Abort(t *testing.T) {
	svc := &fakeRuns{}
	core := clitest.NewTestApp(t, connectAPI(svc))
	h := newHandlers(core)
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id"}}, Flags: []cliapp.Flag{{Name: "reason"}}}
	ctx, out := cliapptest.NewCapturedRunContext(core, schema, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"id": "run-1"}, Flags: map[string]string{"reason": "superseded"},
	})
	require.NoError(t, h.abort(ctx))
	require.Equal(t, "run-1", svc.aborted)
	require.Contains(t, out.String(), "Aborted")
}
