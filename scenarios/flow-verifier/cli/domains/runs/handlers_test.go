package runs

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "github.com/vrooli/cli-core/cliapptest"

	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/runs"
	runsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/runs/runs_v1connect"
)

type fakeService struct {
	listResp *runsv1.ListRunsResponse
	getResp  *runsv1.GetRunResponse
	getErr   error
}

func (f *fakeService) ListRuns(_ context.Context, _ *connect.Request[runsv1.ListRunsRequest]) (*connect.Response[runsv1.ListRunsResponse], error) {
	if f.listResp == nil {
		f.listResp = &runsv1.ListRunsResponse{}
	}
	return connect.NewResponse(f.listResp), nil
}

func (f *fakeService) GetRun(_ context.Context, _ *connect.Request[runsv1.GetRunRequest]) (*connect.Response[runsv1.GetRunResponse], error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return connect.NewResponse(f.getResp), nil
}

func connectAPI(t *testing.T, svc *fakeService) http.Handler {
	t.Helper()
	path, handler := runsconnect.NewRunsServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

func TestRunsList_RendersResults(t *testing.T) {
	svc := &fakeService{listResp: &runsv1.ListRunsResponse{Runs: []*runsv1.Run{
		{Id: "r1", FlowId: "f1", Status: runsv1.RunStatus_RUN_STATUS_PASSED},
	}}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "flow"}, {Name: "limit"}}}
	ctx, out := cliapptest.NewCapturedRunContext(core, schema, cliapptest.TestRunContextOptions{})

	require.NoError(t, h.list(ctx))
	require.Contains(t, out.String(), "Found 1 run(s)")
	require.Contains(t, out.String(), "r1")
}

func TestRunsShow_RendersDetail(t *testing.T) {
	svc := &fakeService{getResp: &runsv1.GetRunResponse{Run: &runsv1.Run{Id: "r1", FlowId: "f1", Output: "stdout body"}}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "run-id", Required: true}}}
	ctx, out := cliapptest.NewCapturedRunContext(core, schema, cliapptest.TestRunContextOptions{Positionals: map[string]string{"run-id": "r1"}})

	require.NoError(t, h.show(ctx))
	require.Contains(t, out.String(), "r1")
	require.Contains(t, out.String(), "stdout body")
}
