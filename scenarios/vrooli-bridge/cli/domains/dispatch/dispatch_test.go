package dispatch

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	dispatchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/dispatch"
	dispatchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/dispatch/dispatch_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "github.com/vrooli/cli-core/cliapptest"
)

// fakeDispatch is a stateful DispatchService so the CLI↔API round-trip exercises
// the real generated client + wire shapes.
type fakeDispatch struct {
	lastReq *dispatchv1.DispatchJobRequest
	dryRun  bool
}

func (f *fakeDispatch) DispatchJob(_ context.Context, req *connect.Request[dispatchv1.DispatchJobRequest]) (*connect.Response[dispatchv1.DispatchJobResponse], error) {
	f.lastReq = req.Msg
	f.dryRun = req.Header().Get("X-Dry-Run") == "true"
	runID := "run-1"
	if f.dryRun {
		runID = ""
	}
	return connect.NewResponse(&dispatchv1.DispatchJobResponse{
		RunId: runID, DryRun: f.dryRun,
		NodeId: req.Msg.NodeId, Scenario: req.Msg.Scenario, Verb: req.Msg.Verb, Args: req.Msg.Args,
	}), nil
}

func connectAPI(svc dispatchconnect.DispatchServiceHandler) http.Handler {
	path, handler := dispatchconnect.NewDispatchServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

// [REQ:BRG-P0-004] The dispatch CLI sends the typed job through the generated
// Connect client and reports the created run id.
func TestDispatch_JobRoundTrip(t *testing.T) {
	svc := &fakeDispatch{}
	core := clitest.NewTestApp(t, connectAPI(svc))
	h := newHandlers(core)

	schema := cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "node-id"}},
		Flags:       []cliapp.Flag{{Name: "verb"}, {Name: "scenario"}, {Name: "args"}, {Name: "timeout"}},
	}
	ctx, out := cliapptest.NewCapturedRunContext(core, schema, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"node-id": "n1"},
		Flags:       map[string]string{"verb": "scenario test", "scenario": "web-search", "args": "web-search,--json"},
	})
	require.NoError(t, h.job(ctx))

	require.Equal(t, "n1", svc.lastReq.NodeId)
	require.Equal(t, "scenario test", svc.lastReq.Verb)
	require.Equal(t, []string{"web-search", "--json"}, svc.lastReq.Args)
	require.Contains(t, out.String(), "run-1")
}
