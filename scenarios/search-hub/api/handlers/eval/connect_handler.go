package eval

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"

	internaleval "search-hub/internal/eval"
)

// Runner is the execution seam the handler depends on: it runs a suite against
// its provider and returns an immutable EvalRun (which the handler then
// persists). internal/eval.Runner satisfies it; handler tests inject a fake.
type Runner interface {
	Run(ctx context.Context, suite *evalv1.EvalSuite, tag string, limit int32) (*evalv1.EvalRun, error)
}

// Deps wires the seams the Connect eval handler needs.
type Deps struct {
	Store  internaleval.Store
	Runner Runner
	Logger *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the EvalService handler. Logger defaults to
// log.Default() when nil.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

// Compile-time guarantee the handler satisfies the generated interface: an RPC
// added to eval.proto the handler hasn't implemented breaks the build here.
var _ = func() any {
	type evalServiceHandler interface {
		RegisterSuite(context.Context, *connect.Request[evalv1.RegisterSuiteRequest]) (*connect.Response[evalv1.RegisterSuiteResponse], error)
		ListSuites(context.Context, *connect.Request[evalv1.ListSuitesRequest]) (*connect.Response[evalv1.ListSuitesResponse], error)
		GetSuite(context.Context, *connect.Request[evalv1.GetSuiteRequest]) (*connect.Response[evalv1.GetSuiteResponse], error)
		RunSuite(context.Context, *connect.Request[evalv1.RunSuiteRequest]) (*connect.Response[evalv1.RunSuiteResponse], error)
		ListRuns(context.Context, *connect.Request[evalv1.ListRunsRequest]) (*connect.Response[evalv1.ListRunsResponse], error)
		GetRun(context.Context, *connect.Request[evalv1.GetRunRequest]) (*connect.Response[evalv1.GetRunResponse], error)
		CompareRuns(context.Context, *connect.Request[evalv1.CompareRunsRequest]) (*connect.Response[evalv1.CompareRunsResponse], error)
	}
	var _ evalServiceHandler = (*connectHandler)(nil)
	return nil
}()

func (h *connectHandler) RegisterSuite(ctx context.Context, req *connect.Request[evalv1.RegisterSuiteRequest]) (*connect.Response[evalv1.RegisterSuiteResponse], error) {
	suite := req.Msg.GetSuite()
	created, err := h.deps.Store.UpsertSuite(ctx, suite)
	if err != nil {
		connectErr := toConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("eval.RegisterSuite(%q): %v", suite.GetSuiteId(), err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&evalv1.RegisterSuiteResponse{Suite: suite, Created: created}), nil
}

func (h *connectHandler) ListSuites(ctx context.Context, req *connect.Request[evalv1.ListSuitesRequest]) (*connect.Response[evalv1.ListSuitesResponse], error) {
	suites, err := h.deps.Store.ListSuites(ctx, internaleval.ListSuitesFilter{ProviderID: req.Msg.GetProviderId()})
	if err != nil {
		h.deps.Logger.Printf("eval.ListSuites: %v", err)
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&evalv1.ListSuitesResponse{Suites: suites}), nil
}

func (h *connectHandler) GetSuite(ctx context.Context, req *connect.Request[evalv1.GetSuiteRequest]) (*connect.Response[evalv1.GetSuiteResponse], error) {
	suite, err := h.deps.Store.GetSuite(ctx, req.Msg.GetSuiteId())
	if err != nil {
		return nil, h.logged("eval.GetSuite", req.Msg.GetSuiteId(), err)
	}
	return connect.NewResponse(&evalv1.GetSuiteResponse{Suite: suite}), nil
}

func (h *connectHandler) RunSuite(ctx context.Context, req *connect.Request[evalv1.RunSuiteRequest]) (*connect.Response[evalv1.RunSuiteResponse], error) {
	suiteID := req.Msg.GetSuiteId()
	suite, err := h.deps.Store.GetSuite(ctx, suiteID)
	if err != nil {
		return nil, h.logged("eval.RunSuite.getSuite", suiteID, err)
	}
	run, err := h.deps.Runner.Run(ctx, suite, req.Msg.GetTag(), req.Msg.GetLimit())
	if err != nil {
		h.deps.Logger.Printf("eval.RunSuite(%q): %v", suiteID, err)
		// A failed run is most often an unregistered/unreachable provider — a
		// caller-correctable precondition, not an internal fault.
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if err := h.deps.Store.AppendRun(ctx, run); err != nil {
		h.deps.Logger.Printf("eval.RunSuite(%q) persist: %v", suiteID, err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("internal error"))
	}
	return connect.NewResponse(&evalv1.RunSuiteResponse{Run: run}), nil
}

func (h *connectHandler) ListRuns(ctx context.Context, req *connect.Request[evalv1.ListRunsRequest]) (*connect.Response[evalv1.ListRunsResponse], error) {
	runs, err := h.deps.Store.ListRuns(ctx, internaleval.ListRunsFilter{
		SuiteID: req.Msg.GetSuiteId(),
		Tag:     req.Msg.GetTag(),
		Limit:   int(req.Msg.GetLimit()),
	})
	if err != nil {
		h.deps.Logger.Printf("eval.ListRuns: %v", err)
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&evalv1.ListRunsResponse{Runs: runs}), nil
}

func (h *connectHandler) GetRun(ctx context.Context, req *connect.Request[evalv1.GetRunRequest]) (*connect.Response[evalv1.GetRunResponse], error) {
	run, err := h.deps.Store.GetRun(ctx, req.Msg.GetRunId())
	if err != nil {
		return nil, h.logged("eval.GetRun", req.Msg.GetRunId(), err)
	}
	return connect.NewResponse(&evalv1.GetRunResponse{Run: run}), nil
}

func (h *connectHandler) CompareRuns(ctx context.Context, req *connect.Request[evalv1.CompareRunsRequest]) (*connect.Response[evalv1.CompareRunsResponse], error) {
	runA, err := h.deps.Store.GetRun(ctx, req.Msg.GetRunA())
	if err != nil {
		return nil, h.logged("eval.CompareRuns.a", req.Msg.GetRunA(), err)
	}
	runB, err := h.deps.Store.GetRun(ctx, req.Msg.GetRunB())
	if err != nil {
		return nil, h.logged("eval.CompareRuns.b", req.Msg.GetRunB(), err)
	}
	return connect.NewResponse(&evalv1.CompareRunsResponse{
		RunA:   runA,
		RunB:   runB,
		Deltas: buildDeltas(runA, runB),
	}), nil
}

// buildDeltas pairs the two runs' per-case results by case_id. Cases present in
// only one run carry empty outcome/score on the missing side (suites drift over
// time). Order: run_a's cases first (run order), then run_b-only cases.
func buildDeltas(a, b *evalv1.EvalRun) []*evalv1.CaseDelta {
	byCaseB := make(map[string]*evalv1.CaseResult, len(b.GetResults()))
	for _, cr := range b.GetResults() {
		byCaseB[cr.GetCaseId()] = cr
	}
	seen := make(map[string]struct{}, len(a.GetResults()))
	out := make([]*evalv1.CaseDelta, 0, len(a.GetResults()))
	for _, ca := range a.GetResults() {
		seen[ca.GetCaseId()] = struct{}{}
		out = append(out, delta(ca, byCaseB[ca.GetCaseId()]))
	}
	for _, cb := range b.GetResults() {
		if _, ok := seen[cb.GetCaseId()]; ok {
			continue
		}
		out = append(out, delta(nil, cb))
	}
	return out
}

func delta(a, b *evalv1.CaseResult) *evalv1.CaseDelta {
	d := &evalv1.CaseDelta{}
	if a != nil {
		d.CaseId = a.GetCaseId()
		d.OutcomeA = a.GetOutcome()
		d.TopScoreA = a.GetObservedTopScore()
		d.ExpectedRankA = a.GetExpectedRank()
	}
	if b != nil {
		d.CaseId = b.GetCaseId()
		d.OutcomeB = b.GetOutcome()
		d.TopScoreB = b.GetObservedTopScore()
		d.ExpectedRankB = b.GetExpectedRank()
	}
	return d
}

// logged translates err and logs only genuine internal faults (not NotFound /
// InvalidArgument, which are caller-facing).
func (h *connectHandler) logged(op, id string, err error) error {
	connectErr := toConnectError(err)
	if connect.CodeOf(connectErr) == connect.CodeInternal {
		h.deps.Logger.Printf("%s(%q): %v", op, id, err)
	}
	return connectErr
}

// toConnectError translates eval domain sentinels into Connect codes at the
// transport edge (the domain layer never imports connect). Validation failures
// become InvalidArgument; a missing suite/run becomes NotFound; everything else
// is an opaque Internal.
func toConnectError(err error) error {
	if err == nil {
		return nil
	}
	var invalid internaleval.ErrInvalidSuite
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	}
	var suiteNotFound internaleval.ErrSuiteNotFound
	if errors.As(err, &suiteNotFound) {
		return connect.NewError(connect.CodeNotFound, suiteNotFound)
	}
	var runNotFound internaleval.ErrRunNotFound
	if errors.As(err, &runNotFound) {
		return connect.NewError(connect.CodeNotFound, runNotFound)
	}
	return connect.NewError(connect.CodeInternal, errors.New("internal error"))
}
