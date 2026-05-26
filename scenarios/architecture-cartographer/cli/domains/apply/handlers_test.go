package apply

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	applyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/apply"
	applyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/apply/apply_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "architecture-cartographer/cli/internal/testutil"
)

type fakeService struct {
	applyconnect.UnimplementedApplyServiceHandler

	mu          sync.Mutex
	planReqs    []*applyv1.PlanApplyRequest
	planResp    *applyv1.PlanApplyResponse
	runErr      error
	historyResp *applyv1.ListApplyHistoryResponse
	baseResp    *applyv1.GetBuildBaselineResponse
}

func (s *fakeService) PlanApply(_ context.Context, req *connect.Request[applyv1.PlanApplyRequest]) (*connect.Response[applyv1.PlanApplyResponse], error) {
	s.mu.Lock()
	s.planReqs = append(s.planReqs, req.Msg)
	s.mu.Unlock()
	return connect.NewResponse(s.planResp), nil
}

func (s *fakeService) RunApply(_ context.Context, _ *connect.Request[applyv1.RunApplyRequest]) (*connect.Response[applyv1.RunApplyResponse], error) {
	if s.runErr != nil {
		return nil, s.runErr
	}
	return connect.NewResponse(&applyv1.RunApplyResponse{}), nil
}

func (s *fakeService) ListApplyHistory(_ context.Context, _ *connect.Request[applyv1.ListApplyHistoryRequest]) (*connect.Response[applyv1.ListApplyHistoryResponse], error) {
	return connect.NewResponse(s.historyResp), nil
}

func (s *fakeService) GetBuildBaseline(_ context.Context, _ *connect.Request[applyv1.GetBuildBaselineRequest]) (*connect.Response[applyv1.GetBuildBaselineResponse], error) {
	return connect.NewResponse(s.baseResp), nil
}

func connectAPI(t *testing.T, svc *fakeService) http.Handler {
	t.Helper()
	path, handler := applyconnect.NewApplyServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

func TestPlan_RendersOperations(t *testing.T) {
	svc := &fakeService{planResp: &applyv1.PlanApplyResponse{Plan: &applyv1.Plan{
		Id:       "p-1",
		Scenario: "demo",
		Domain:   "graph",
		Operations: []*applyv1.Operation{
			{Id: "op-1", Kind: applyv1.OperationKind_OPERATION_KIND_MOVE_FILE, FromPath: "a.go", ToPath: "b.go"},
		},
	}}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, planSchema(), cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"scenario": "demo", "domain": "graph"},
		Flags:       map[string]string{"conflict-id": "c-1,c-2"},
	})

	require.NoError(t, h.plan(ctx))
	require.Len(t, svc.planReqs, 1)
	require.Equal(t, []string{"c-1", "c-2"}, svc.planReqs[0].GetConflictIds())
	body := out.String()
	require.Contains(t, body, "1 operation(s)")
	require.Contains(t, body, "move_file a.go -> b.go")
}

// TestRun_SurfacesUnimplementedCleanly is the plan's required guard:
// CodeUnimplemented must render as a friendly capability-not-available
// message that names the unblocking plan — not a crash or stack trace.
func TestRun_SurfacesUnimplementedCleanly(t *testing.T) {
	svc := &fakeService{runErr: connect.NewError(connect.CodeUnimplemented, errors.New("apply execution unimplemented in v0.1"))}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, runSchema(), cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"plan_id": "p-1"},
	})

	err := h.run(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not implemented in v0.1")
	require.Contains(t, err.Error(), "apply-execution plan")
	require.NotContains(t, err.Error(), "unavailable: ") // not a raw connect wrap
}

func TestHistory_EmptyInV01(t *testing.T) {
	svc := &fakeService{historyResp: &applyv1.ListApplyHistoryResponse{}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, historySchema(), cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"scenario": "demo"},
	})

	require.NoError(t, h.history(ctx))
	require.Contains(t, out.String(), "0 apply run(s)")
}

func TestBaseline_EmptyInV01(t *testing.T) {
	svc := &fakeService{baseResp: &applyv1.GetBuildBaselineResponse{Baseline: &applyv1.BuildBaseline{Scenario: "demo"}}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "scenario", Required: true}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"scenario": "demo"},
	})

	require.NoError(t, h.baseline(ctx))
	require.Contains(t, out.String(), "No build baseline recorded")
}

func planSchema() cliapp.ArgSchema {
	return cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "scenario", Required: true}, {Name: "domain", Required: true}},
		Flags:       []cliapp.Flag{{Name: "conflict-id"}},
	}
}

func runSchema() cliapp.ArgSchema {
	return cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "plan_id", Required: true}},
		Flags:       []cliapp.Flag{{Name: "acknowledge", Bool: true}},
	}
}

func historySchema() cliapp.ArgSchema {
	return cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "scenario", Required: true}},
		Flags:       []cliapp.Flag{{Name: "domain"}, {Name: "page-size"}, {Name: "page-token"}},
	}
}
