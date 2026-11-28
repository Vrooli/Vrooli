package trials

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	trialsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/trials"
	trialsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/trials/trials_v1connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type trialsServiceStub struct {
	trialsconnect.UnimplementedTrialsServiceHandler

	err      error
	listReq  *trialsv1.ListTrialTasksRequest
	runReq   *trialsv1.RunTrialsRequest
	histReq  *trialsv1.GetTrialHistoryRequest
	showReq  *trialsv1.GetTrialRunRequest
	coverReq *trialsv1.GetGateCoverageRequest
}

type nilTrialsClient struct {
	trialsconnect.TrialsServiceClient
}

func (nilTrialsClient) ListTrialTasks(context.Context, *connect.Request[trialsv1.ListTrialTasksRequest]) (*connect.Response[trialsv1.ListTrialTasksResponse], error) {
	return nil, nil
}

func (nilTrialsClient) RunTrials(context.Context, *connect.Request[trialsv1.RunTrialsRequest]) (*connect.Response[trialsv1.RunTrialsResponse], error) {
	return nil, nil
}

func (nilTrialsClient) GetTrialHistory(context.Context, *connect.Request[trialsv1.GetTrialHistoryRequest]) (*connect.Response[trialsv1.GetTrialHistoryResponse], error) {
	return nil, nil
}

func (nilTrialsClient) GetTrialRun(context.Context, *connect.Request[trialsv1.GetTrialRunRequest]) (*connect.Response[trialsv1.GetTrialRunResponse], error) {
	return nil, nil
}

func (nilTrialsClient) GetGateCoverage(context.Context, *connect.Request[trialsv1.GetGateCoverageRequest]) (*connect.Response[trialsv1.GetGateCoverageResponse], error) {
	return nil, nil
}

func (s *trialsServiceStub) ListTrialTasks(_ context.Context, req *connect.Request[trialsv1.ListTrialTasksRequest]) (*connect.Response[trialsv1.ListTrialTasksResponse], error) {
	s.listReq = req.Msg
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&trialsv1.ListTrialTasksResponse{Tasks: []*trialsv1.TrialTask{{
		Id: "task-a", Suite: "baseline", Description: "check a contract", Negative: true,
	}}}), nil
}

func (s *trialsServiceStub) RunTrials(_ context.Context, req *connect.Request[trialsv1.RunTrialsRequest]) (*connect.Response[trialsv1.RunTrialsResponse], error) {
	s.runReq = req.Msg
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&trialsv1.RunTrialsResponse{Runs: []*trialsv1.TrialRun{{
		Id: "run-1", TaskId: "task-a", Suite: "baseline", Verdict: trialsv1.TrialVerdict_TRIAL_VERDICT_PASS,
		Tokens: 42, DurationMs: 900, At: timestamppb.New(time.Date(2026, time.January, 2, 3, 4, 0, 0, time.UTC)),
	}}}), nil
}

func (s *trialsServiceStub) GetTrialHistory(_ context.Context, req *connect.Request[trialsv1.GetTrialHistoryRequest]) (*connect.Response[trialsv1.GetTrialHistoryResponse], error) {
	s.histReq = req.Msg
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&trialsv1.GetTrialHistoryResponse{
		Points: []*trialsv1.TrialHistoryPoint{{
			At:          timestamppb.New(time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)),
			SuccessRate: .75, MedianTokens: 120, MedianDurationMs: 2500, RunCount: 4,
		}},
		RecentRuns: []*trialsv1.TrialRun{{Id: "run-1"}},
	}), nil
}

func (s *trialsServiceStub) GetTrialRun(_ context.Context, req *connect.Request[trialsv1.GetTrialRunRequest]) (*connect.Response[trialsv1.GetTrialRunResponse], error) {
	s.showReq = req.Msg
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&trialsv1.GetTrialRunResponse{Run: &trialsv1.TrialRun{
		Id: "run-1", Suite: "baseline", Verdict: trialsv1.TrialVerdict_TRIAL_VERDICT_FAIL,
		SandboxDiffRef: "", At: timestamppb.New(time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)),
	}}), nil
}

func (s *trialsServiceStub) GetGateCoverage(_ context.Context, req *connect.Request[trialsv1.GetGateCoverageRequest]) (*connect.Response[trialsv1.GetGateCoverageResponse], error) {
	s.coverReq = req.Msg
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&trialsv1.GetGateCoverageResponse{
		GuideTasksTotal: 8, GuideTasksWithGate: 6, GateCoverageRatio: .75,
	}), nil
}

func newTrialsTestHandlers(t *testing.T, svc *trialsServiceStub) (*handlers, *cliapp.ScenarioApp) {
	t.Helper()
	path, serviceHandler := trialsconnect.NewTrialsServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, serviceHandler)
	core := cliapptest.NewTestApp(t, mux)
	return newHandlers(core), core
}

func trialsRunContext(core *cliapp.ScenarioApp, opts cliapptest.TestRunContextOptions) (cliapp.RunContext, *strings.Builder) {
	var stdout strings.Builder
	opts.Stdout = &stdout
	opts.Core = core
	opts.Schema = cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "run-id"}},
		Flags: []cliapp.Flag{
			{Name: "suite"}, {Name: "task"}, {Name: "all", Bool: true},
		},
	}
	return cliapptest.NewTestRunContext(opts), &stdout
}

func requireProtoEqual(t *testing.T, want, got proto.Message) {
	t.Helper()
	require.Truef(t, proto.Equal(want, got), "protobuf mismatch: got %v, want %v", got, want)
}

func TestRunRequiresExplicitScopeBeforeCallingService(t *testing.T) {
	h := &handlers{}
	ctx, _ := trialsRunContext(nil, cliapptest.TestRunContextOptions{})

	err := h.run(ctx)
	require.EqualError(t, err, "trials run needs a scope: pass --task <id> for one task, --suite <family> for a family, or --all for the full suite")
}

func TestRunPassesExplicitTaskAndSuiteScopeAndRendersRun(t *testing.T) {
	svc := &trialsServiceStub{}
	h, core := newTrialsTestHandlers(t, svc)
	ctx, stdout := trialsRunContext(core, cliapptest.TestRunContextOptions{
		Flags:     map[string]string{"task": " task-a ", "suite": " baseline "},
		BoolFlags: map[string]bool{"all": true},
	})

	require.NoError(t, h.run(ctx))
	requireProtoEqual(t, &trialsv1.RunTrialsRequest{TaskId: "task-a", Suite: "baseline"}, svc.runReq)
	require.Contains(t, stdout.String(), "run-1 [pass] baseline tokens=42 duration_ms=900 @ 2026-01-02 03:04")
}

func TestRunAllExplicitlyDispatchesEmptyTaskAndSuite(t *testing.T) {
	svc := &trialsServiceStub{}
	h, core := newTrialsTestHandlers(t, svc)
	ctx, _ := trialsRunContext(core, cliapptest.TestRunContextOptions{BoolFlags: map[string]bool{"all": true}})

	require.NoError(t, h.run(ctx))
	requireProtoEqual(t, &trialsv1.RunTrialsRequest{}, svc.runReq)
}

func TestListTrimsSuiteAndRendersTaskDetails(t *testing.T) {
	svc := &trialsServiceStub{}
	h, core := newTrialsTestHandlers(t, svc)
	ctx, stdout := trialsRunContext(core, cliapptest.TestRunContextOptions{Flags: map[string]string{"suite": " baseline "}})

	require.NoError(t, h.list(ctx))
	requireProtoEqual(t, &trialsv1.ListTrialTasksRequest{Suite: "baseline"}, svc.listReq)
	require.Contains(t, stdout.String(), "1 trial task(s).")
	require.Contains(t, stdout.String(), "task-a [baseline] check a contract (negative)")
	require.Contains(t, stdout.String(), "`trials run --task <id>`")
}

func TestHistoryPassesFiltersAndRendersTrend(t *testing.T) {
	svc := &trialsServiceStub{}
	h, core := newTrialsTestHandlers(t, svc)
	ctx, stdout := trialsRunContext(core, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"task": " task-a ", "suite": " baseline "},
	})

	require.NoError(t, h.history(ctx))
	requireProtoEqual(t, &trialsv1.GetTrialHistoryRequest{TaskId: "task-a", Suite: "baseline"}, svc.histReq)
	require.Contains(t, stdout.String(), "1 trend point(s), 1 recent run(s).")
	require.Contains(t, stdout.String(), "2026-01-02 — success=75% median_tokens=120 median_ms=2500 runs=4")
}

func TestShowPassesRunIDAndRendersMissingDiffAsDash(t *testing.T) {
	svc := &trialsServiceStub{}
	h, core := newTrialsTestHandlers(t, svc)
	ctx, stdout := trialsRunContext(core, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"run-id": " run-1 "},
	})

	require.NoError(t, h.show(ctx))
	requireProtoEqual(t, &trialsv1.GetTrialRunRequest{Id: "run-1"}, svc.showReq)
	require.Contains(t, stdout.String(), "Run run-1.")
	require.Contains(t, stdout.String(), "sandbox diff: —")
	require.Contains(t, stdout.String(), "[fail]")
}

func TestCoverageRendersServerCountsAndRatio(t *testing.T) {
	svc := &trialsServiceStub{}
	h, core := newTrialsTestHandlers(t, svc)
	ctx, stdout := trialsRunContext(core, cliapptest.TestRunContextOptions{})

	require.NoError(t, h.coverage(ctx))
	requireProtoEqual(t, &trialsv1.GetGateCoverageRequest{}, svc.coverReq)
	require.Contains(t, stdout.String(), "Guide-gate coverage: 75% (6/8 Guide tasks have a live gate).")
}

func TestTrialCommandsSurfaceServiceErrors(t *testing.T) {
	svc := &trialsServiceStub{err: errors.New("service unavailable")}
	h, core := newTrialsTestHandlers(t, svc)
	cases := []struct {
		name string
		call func(cliapp.RunContext) error
		ctx  cliapptest.TestRunContextOptions
	}{
		{name: "list", call: h.list, ctx: cliapptest.TestRunContextOptions{}},
		{name: "run", call: h.run, ctx: cliapptest.TestRunContextOptions{Flags: map[string]string{"task": "task-a"}}},
		{name: "history", call: h.history, ctx: cliapptest.TestRunContextOptions{}},
		{name: "show", call: h.show, ctx: cliapptest.TestRunContextOptions{Positionals: map[string]string{"run-id": "run-1"}}},
		{name: "coverage", call: h.coverage, ctx: cliapptest.TestRunContextOptions{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := trialsRunContext(core, tc.ctx)
			require.ErrorContains(t, tc.call(ctx), "service unavailable")
		})
	}
}

func TestTrialCommandsRejectNilResponses(t *testing.T) {
	h := &handlers{client: nilTrialsClient{}}
	cases := []struct {
		name string
		call func(cliapp.RunContext) error
		ctx  cliapptest.TestRunContextOptions
		want string
	}{
		{name: "list", call: h.list, ctx: cliapptest.TestRunContextOptions{}, want: "server returned no tasks response"},
		{name: "run", call: h.run, ctx: cliapptest.TestRunContextOptions{Flags: map[string]string{"task": "task-a"}}, want: "server returned no runs response"},
		{name: "history", call: h.history, ctx: cliapptest.TestRunContextOptions{}, want: "server returned no history response"},
		{name: "show", call: h.show, ctx: cliapptest.TestRunContextOptions{Positionals: map[string]string{"run-id": "run-1"}}, want: "server returned no run"},
		{name: "coverage", call: h.coverage, ctx: cliapptest.TestRunContextOptions{}, want: "server returned no coverage response"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := trialsRunContext(nil, tc.ctx)
			require.EqualError(t, tc.call(ctx), tc.want)
		})
	}
}

func TestShowRejectsResponseWithoutRun(t *testing.T) {
	h := &handlers{client: nilRunTrialsClient{}}
	ctx, _ := trialsRunContext(nil, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"run-id": "run-1"},
	})

	require.EqualError(t, h.show(ctx), "server returned no run")
}

type nilRunTrialsClient struct {
	trialsconnect.TrialsServiceClient
}

func (nilRunTrialsClient) GetTrialRun(context.Context, *connect.Request[trialsv1.GetTrialRunRequest]) (*connect.Response[trialsv1.GetTrialRunResponse], error) {
	return connect.NewResponse(&trialsv1.GetTrialRunResponse{}), nil
}
