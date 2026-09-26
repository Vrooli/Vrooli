package eval

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"connectrpc.com/connect"

	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/control"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"

	internaleval "search-hub/internal/eval"
	internalregistry "search-hub/internal/registry"
)

// Runner is the execution seam the handler depends on: it runs a suite against
// its provider and returns an immutable EvalRun (which the handler then
// persists). internal/eval.Runner satisfies it; handler tests inject a fake.
type Runner interface {
	Run(ctx context.Context, suite *evalv1.EvalSuite, tag string, limit int32) (*evalv1.EvalRun, error)
}

// StrategyRunner is implemented by the federated runner because strategy
// selection is meaningful only for router-level evaluation arms. Keeping this
// optional preserves the provider-direct runner seam and its test doubles.
type StrategyRunner interface {
	RunWithStrategy(ctx context.Context, suite *evalv1.EvalSuite, tag string, limit int32, strategyName string) (*evalv1.EvalRun, error)
}

type Validator interface {
	ValidateCorpus(ctx context.Context, suite *evalv1.EvalSuite, deepK int32) (*evalv1.ValidateCorpusResponse, error)
}

// Sweeper is the optimization seam: it runs the two-tier, overfit-safe parameter
// sweep for a suite and (optionally) writes back a winner. internal/sweep.
// Orchestrator satisfies it; it persists one tagged run per arm itself (through
// its own store seam), so the handler only forwards the request/response.
type Sweeper interface {
	Run(ctx context.Context, req *evalv1.SweepRequest) (*evalv1.SweepResult, error)
}

// CorpusController writes a grown corpus back to a provider's search.json SSOT via
// the token-gated WriteCorpus control RPC. internal/control.Client satisfies it.
// `generate --apply` routes through this so the FILE is the mutation target, never
// the store (corpusMutationsGoThroughFile). A server without it can still preview.
type CorpusController interface {
	WriteCorpus(ctx context.Context, d *registryv1.ProviderDescriptor, controlToken string, corpus *evalv1.EvalSuite, dryRun bool) (*controlv1.WriteCorpusResponse, error)
}

// ControlTokenResolver resolves a provider's control token (minted at
// registration) to present on the WriteCorpus call. The registry store satisfies
// it (same Token method the sweep uses).
type ControlTokenResolver interface {
	Token(ctx context.Context, providerID string) (string, error)
}

// Deps wires the seams the Connect eval handler needs.
type Deps struct {
	Store    internaleval.Store
	Registry internalregistry.Store
	// Providers resolves a suite's provider descriptor (Generate samples the
	// provider's index; the corpus generator needs its endpoint + facets).
	Providers      internaleval.ProviderResolver
	Runner         Runner
	Federated      Runner
	Validator      Validator
	Sweeper        Sweeper
	ActiveStrategy string
	Routability    RoutabilityReader
	// Generator proposes machine-generated cases for a suite (Generate RPC).
	Generator CorpusGenerator
	// Control + Tokens drive the corpus write-back: `generate --apply` persists the
	// grown corpus into the provider's search.json (Control.WriteCorpus, authorized
	// with Tokens.Token), then re-registers the returned corpus into the store so
	// the store mirror re-syncs with the file. Both nil => apply is Unimplemented.
	Control CorpusController
	Tokens  ControlTokenResolver
	Logger  *log.Logger
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
		CompareStrategies(context.Context, *connect.Request[evalv1.CompareStrategiesRequest]) (*connect.Response[evalv1.CompareStrategiesResponse], error)
		ValidateCorpus(context.Context, *connect.Request[evalv1.ValidateCorpusRequest]) (*connect.Response[evalv1.ValidateCorpusResponse], error)
		ListRuns(context.Context, *connect.Request[evalv1.ListRunsRequest]) (*connect.Response[evalv1.ListRunsResponse], error)
		GetRun(context.Context, *connect.Request[evalv1.GetRunRequest]) (*connect.Response[evalv1.GetRunResponse], error)
		CompareRuns(context.Context, *connect.Request[evalv1.CompareRunsRequest]) (*connect.Response[evalv1.CompareRunsResponse], error)
		Sweep(context.Context, *connect.Request[evalv1.SweepRequest]) (*connect.Response[evalv1.SweepResponse], error)
		Generate(context.Context, *connect.Request[evalv1.GenerateRequest]) (*connect.Response[evalv1.GenerateResponse], error)
		PromoteCases(context.Context, *connect.Request[evalv1.PromoteCasesRequest]) (*connect.Response[evalv1.PromoteCasesResponse], error)
		ReapOrphanSuites(context.Context, *connect.Request[evalv1.ReapOrphanSuitesRequest]) (*connect.Response[evalv1.ReapOrphanSuitesResponse], error)
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

// ReapOrphanSuites is deliberately a dry-run unless confirm=true. It computes
// the orphan set from the live registry at call time, so a provider that came
// back between an audit and a confirmed run is never deleted accidentally.
func (h *connectHandler) ReapOrphanSuites(ctx context.Context, req *connect.Request[evalv1.ReapOrphanSuitesRequest]) (*connect.Response[evalv1.ReapOrphanSuitesResponse], error) {
	if h.deps.Registry == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("registry is not configured"))
	}
	suites, err := h.deps.Store.ListSuites(ctx, internaleval.ListSuitesFilter{})
	if err != nil {
		return nil, toConnectError(err)
	}
	providers, err := h.deps.Registry.List(ctx, internalregistry.ListFilter{})
	if err != nil {
		return nil, toConnectError(err)
	}
	registered := make(map[string]struct{}, len(providers))
	for _, provider := range providers {
		registered[provider.GetProviderId()] = struct{}{}
	}
	orphans := make([]*evalv1.EvalSuite, 0)
	for _, suite := range suites {
		if _, ok := registered[suite.GetProviderId()]; !ok {
			orphans = append(orphans, suite)
		}
	}
	resp := &evalv1.ReapOrphanSuitesResponse{OrphanSuites: orphans, Confirmed: req.Msg.GetConfirm()}
	if !req.Msg.GetConfirm() {
		return connect.NewResponse(resp), nil
	}
	for _, suite := range orphans {
		// Re-check the provider immediately before mutation to close the audit /
		// mutation race without introducing a broad delete primitive.
		if _, getErr := h.deps.Registry.Get(ctx, suite.GetProviderId()); getErr == nil {
			continue
		}
		if err := h.deps.Store.DeleteSuite(ctx, suite.GetSuiteId()); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.New("failed to reap orphan suite"))
		}
		resp.ReapedSuiteIds = append(resp.ReapedSuiteIds, suite.GetSuiteId())
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ListSuites(ctx context.Context, req *connect.Request[evalv1.ListSuitesRequest]) (*connect.Response[evalv1.ListSuitesResponse], error) {
	if req.Msg.GetProviderId() == internaleval.RouterSuiteID {
		suite, err := h.composedSuite(ctx)
		if err != nil {
			return nil, toConnectError(err)
		}
		return connect.NewResponse(&evalv1.ListSuitesResponse{Suites: []*evalv1.EvalSuite{suite}}), nil
	}
	suites, err := h.deps.Store.ListSuites(ctx, internaleval.ListSuitesFilter{ProviderID: req.Msg.GetProviderId()})
	if err != nil {
		h.deps.Logger.Printf("eval.ListSuites: %v", err)
		return nil, toConnectError(err)
	}
	if req.Msg.GetProviderId() == "" && h.deps.Registry != nil {
		composed, composeErr := h.composedSuite(ctx)
		if composeErr != nil {
			return nil, toConnectError(composeErr)
		}
		suites = append(suites, composed)
	}
	return connect.NewResponse(&evalv1.ListSuitesResponse{Suites: suites}), nil
}

func (h *connectHandler) GetSuite(ctx context.Context, req *connect.Request[evalv1.GetSuiteRequest]) (*connect.Response[evalv1.GetSuiteResponse], error) {
	suite, err := h.getSuite(ctx, req.Msg.GetSuiteId())
	if err != nil {
		return nil, h.logged("eval.GetSuite", req.Msg.GetSuiteId(), err)
	}
	return connect.NewResponse(&evalv1.GetSuiteResponse{
		Suite:    suite,
		Adequacy: internaleval.CheckAdequacy(suite, nil),
	}), nil
}

func (h *connectHandler) RunSuite(ctx context.Context, req *connect.Request[evalv1.RunSuiteRequest]) (*connect.Response[evalv1.RunSuiteResponse], error) {
	suiteID := req.Msg.GetSuiteId()
	suite, err := h.getSuite(ctx, suiteID)
	if err != nil {
		return nil, h.logged("eval.RunSuite.getSuite", suiteID, err)
	}
	runner := h.deps.Runner
	// The composed router suite is a synthetic corpus owned by Search Hub;
	// its only meaningful default execution is through the federated path.
	// Keep `evals run router.routing` useful without requiring callers to know
	// the implementation detail that this suite has no provider endpoint.
	if suiteID == internaleval.RouterSuiteID && strings.TrimSpace(req.Msg.GetTier()) == "" {
		if h.deps.Federated == nil {
			return nil, connect.NewError(connect.CodeUnimplemented, errors.New("federated eval tier is not configured"))
		}
		runner = h.deps.Federated
	} else if req.Msg.GetTier() == "federated" {
		if h.deps.Federated == nil {
			return nil, connect.NewError(connect.CodeUnimplemented, errors.New("federated eval tier is not configured"))
		}
		runner = h.deps.Federated
	}
	strategyName := strings.TrimSpace(req.Msg.GetStrategyName())
	if strategyName != "" {
		if suiteID != internaleval.RouterSuiteID || runner == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("strategy_name requires the router.routing federated suite"))
		}
		strategyRunner, ok := runner.(StrategyRunner)
		if !ok {
			return nil, connect.NewError(connect.CodeUnimplemented, errors.New("federated strategy evaluation is not configured"))
		}
		run, err := strategyRunner.RunWithStrategy(ctx, suite, req.Msg.GetTag(), req.Msg.GetLimit(), strategyName)
		if err != nil {
			h.deps.Logger.Printf("eval.RunSuite(%q, strategy=%q): %v", suiteID, strategyName, err)
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		if err := h.deps.Store.AppendRun(ctx, run); err != nil {
			h.deps.Logger.Printf("eval.RunSuite(%q, strategy=%q) persist: %v", suiteID, strategyName, err)
			return nil, connect.NewError(connect.CodeInternal, errors.New("internal error"))
		}
		return connect.NewResponse(&evalv1.RunSuiteResponse{Run: run, Adequacy: internaleval.CheckAdequacy(suite, nil)}), nil
	}
	run, err := runner.Run(ctx, suite, req.Msg.GetTag(), req.Msg.GetLimit())
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
	return connect.NewResponse(&evalv1.RunSuiteResponse{
		Run:      run,
		Adequacy: internaleval.CheckAdequacy(suite, nil),
	}), nil
}

func (h *connectHandler) getSuite(ctx context.Context, id string) (*evalv1.EvalSuite, error) {
	if id == internaleval.RouterSuiteID {
		return h.composedSuite(ctx)
	}
	return h.deps.Store.GetSuite(ctx, id)
}

func (h *connectHandler) composedSuite(ctx context.Context) (*evalv1.EvalSuite, error) {
	if h.deps.Registry == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("router suite requires registry"))
	}
	suites, err := h.deps.Store.ListSuites(ctx, internaleval.ListSuitesFilter{})
	if err != nil {
		return nil, err
	}
	providers, err := h.deps.Registry.List(ctx, internalregistry.ListFilter{})
	if err != nil {
		return nil, err
	}
	registered := make(map[string]struct{}, len(providers))
	for _, provider := range providers {
		registered[provider.GetProviderId()] = struct{}{}
	}
	return internaleval.ComposeRoutingSuite(suites, registered), nil
}

func (h *connectHandler) ValidateCorpus(ctx context.Context, req *connect.Request[evalv1.ValidateCorpusRequest]) (*connect.Response[evalv1.ValidateCorpusResponse], error) {
	if h.deps.Validator == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("corpus validation is not configured on this server"))
	}
	suiteID := req.Msg.GetSuiteId()
	suite, err := h.deps.Store.GetSuite(ctx, suiteID)
	if err != nil {
		return nil, h.logged("eval.ValidateCorpus.getSuite", suiteID, err)
	}
	resp, err := h.deps.Validator.ValidateCorpus(ctx, suite, req.Msg.GetDeepK())
	if err != nil {
		h.deps.Logger.Printf("eval.ValidateCorpus(%q): %v", suiteID, err)
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	// Manual validation is an operator diagnostic that must participate in the
	// same freshness gate as scheduled validation. Persist it when the backing
	// store exposes the durable seam; keeping the assertion optional preserves
	// small in-memory handler fakes used by unit tests.
	if appender, ok := h.deps.Store.(interface {
		AppendCorpusValidation(context.Context, string, *evalv1.ValidateCorpusResponse, time.Time) error
	}); ok {
		if err := appender.AppendCorpusValidation(ctx, suiteID, resp, time.Now().UTC()); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("persist corpus validation: %w", err))
		}
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ListRuns(ctx context.Context, req *connect.Request[evalv1.ListRunsRequest]) (*connect.Response[evalv1.ListRunsResponse], error) {
	runs, err := h.deps.Store.ListRuns(ctx, internaleval.ListRunsFilter{
		SuiteID: req.Msg.GetSuiteId(),
		Tag:     req.Msg.GetTag(),
		Tier:    req.Msg.GetTier(),
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

// Sweep runs the two-tier parameter sweep for a suite and returns the ranked
// result + promotion verdict. The orchestrator stores one tagged run per arm
// itself, so the handler only translates errors. A sweep error is almost always
// a caller-correctable precondition (no such suite, the suite's provider is
// unregistered/declares no control plane, or the suite has no positive cases) —
// not an internal fault.
func (h *connectHandler) Sweep(ctx context.Context, req *connect.Request[evalv1.SweepRequest]) (*connect.Response[evalv1.SweepResponse], error) {
	if h.deps.Sweeper == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("sweep is not configured on this server"))
	}
	res, err := h.deps.Sweeper.Run(ctx, req.Msg)
	if err != nil {
		h.deps.Logger.Printf("eval.Sweep(%q): %v", req.Msg.GetSuiteId(), err)
		var suiteNotFound internaleval.ErrSuiteNotFound
		if errors.As(err, &suiteNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, suiteNotFound)
		}
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&evalv1.SweepResponse{Result: res}), nil
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
