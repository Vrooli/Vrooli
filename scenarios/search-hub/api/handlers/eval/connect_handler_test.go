package eval_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	evalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval/eval_v1connect"

	handler "search-hub/handlers/eval"
	internaleval "search-hub/internal/eval"
)

// fakeStore is a hand-written eval.Store fake. The real persistence path is
// covered by internal/eval/store_test.go; here we assert the handler's
// orchestration and error translation.
type fakeStore struct {
	suites    map[string]*evalv1.EvalSuite
	runs      map[string]*evalv1.EvalRun
	appended  []*evalv1.EvalRun
	upsertErr error
}

func newFakeStore() *fakeStore {
	return &fakeStore{suites: map[string]*evalv1.EvalSuite{}, runs: map[string]*evalv1.EvalRun{}}
}

func (f *fakeStore) UpsertSuite(_ context.Context, s *evalv1.EvalSuite) (bool, error) {
	if f.upsertErr != nil {
		return false, f.upsertErr
	}
	_, existed := f.suites[s.GetSuiteId()]
	f.suites[s.GetSuiteId()] = s
	return !existed, nil
}

func (f *fakeStore) ListSuites(_ context.Context, filter internaleval.ListSuitesFilter) ([]*evalv1.EvalSuite, error) {
	out := []*evalv1.EvalSuite{}
	for _, s := range f.suites {
		if filter.ProviderID == "" || s.GetProviderId() == filter.ProviderID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (f *fakeStore) GetSuite(_ context.Context, id string) (*evalv1.EvalSuite, error) {
	if s, ok := f.suites[id]; ok {
		return s, nil
	}
	return nil, internaleval.ErrSuiteNotFound{SuiteID: id}
}

func (f *fakeStore) AppendRun(_ context.Context, run *evalv1.EvalRun) error {
	f.appended = append(f.appended, run)
	f.runs[run.GetRunId()] = run
	return nil
}

func (f *fakeStore) ListRuns(_ context.Context, filter internaleval.ListRunsFilter) ([]*evalv1.EvalRun, error) {
	out := []*evalv1.EvalRun{}
	for _, r := range f.runs {
		if r.GetSuiteId() == filter.SuiteID && (filter.Tag == "" || r.GetTag() == filter.Tag) {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeStore) GetRun(_ context.Context, id string) (*evalv1.EvalRun, error) {
	if r, ok := f.runs[id]; ok {
		return r, nil
	}
	return nil, internaleval.ErrRunNotFound{RunID: id}
}

// fakeRunner returns a canned run (or error) and records its inputs.
type fakeRunner struct {
	out      *evalv1.EvalRun
	err      error
	lastTag  string
	lastID   string
	lastLim  int32
	gotSuite *evalv1.EvalSuite
}

func (f *fakeRunner) Run(_ context.Context, suite *evalv1.EvalSuite, tag string, limit int32) (*evalv1.EvalRun, error) {
	f.gotSuite = suite
	f.lastTag = tag
	f.lastID = suite.GetSuiteId()
	f.lastLim = limit
	if f.err != nil {
		return nil, f.err
	}
	return f.out, nil
}

func newClient(t *testing.T, store internaleval.Store, runner handler.Runner) evalconnect.EvalServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	path, h := evalconnect.NewEvalServiceHandler(handler.NewConnectHandler(handler.Deps{Store: store, Runner: runner, Logger: logger}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: h})
	return evalconnect.NewEvalServiceClient(server.Client(), server.URL)
}

func validSuite() *evalv1.EvalSuite {
	return &evalv1.EvalSuite{
		SuiteId:    "cli-health.commands.primary",
		ProviderId: "cli-health.commands",
		Name:       "primary",
		Cases:      []*evalv1.EvalCase{{CaseId: "c1", Query: "q", ExpectWithinTopK: 3}},
	}
}

func TestRegisterSuiteCreated(t *testing.T) {
	store := newFakeStore()
	client := newClient(t, store, &fakeRunner{})

	resp, err := client.RegisterSuite(context.Background(), connect.NewRequest(&evalv1.RegisterSuiteRequest{Suite: validSuite()}))
	require.NoError(t, err)
	require.True(t, resp.Msg.GetCreated())
	require.Equal(t, "cli-health.commands.primary", resp.Msg.GetSuite().GetSuiteId())
}

func TestRegisterSuiteInvalidArgument(t *testing.T) {
	store := newFakeStore()
	store.upsertErr = internaleval.ErrInvalidSuite{Field: "cases", Reason: "required"}
	client := newClient(t, store, &fakeRunner{})

	_, err := client.RegisterSuite(context.Background(), connect.NewRequest(&evalv1.RegisterSuiteRequest{Suite: validSuite()}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestGetSuiteNotFound(t *testing.T) {
	client := newClient(t, newFakeStore(), &fakeRunner{})
	_, err := client.GetSuite(context.Background(), connect.NewRequest(&evalv1.GetSuiteRequest{SuiteId: "nope"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestRunSuiteExecutesAndPersists(t *testing.T) {
	store := newFakeStore()
	_, _ = store.UpsertSuite(context.Background(), validSuite())
	runner := &fakeRunner{out: &evalv1.EvalRun{RunId: "run-xyz", SuiteId: "cli-health.commands.primary", Tag: "cross-encoder"}}
	client := newClient(t, store, runner)

	resp, err := client.RunSuite(context.Background(), connect.NewRequest(&evalv1.RunSuiteRequest{
		SuiteId: "cli-health.commands.primary", Tag: "cross-encoder", Limit: 7,
	}))
	require.NoError(t, err)
	require.Equal(t, "run-xyz", resp.Msg.GetRun().GetRunId())
	require.Equal(t, "cross-encoder", runner.lastTag)
	require.EqualValues(t, 7, runner.lastLim)
	require.Equal(t, "cli-health.commands.primary", runner.gotSuite.GetSuiteId())
	require.Len(t, store.appended, 1, "the run must be persisted")
	require.Equal(t, "run-xyz", store.appended[0].GetRunId())
}

func TestRunSuiteUnknownSuiteNotFound(t *testing.T) {
	client := newClient(t, newFakeStore(), &fakeRunner{})
	_, err := client.RunSuite(context.Background(), connect.NewRequest(&evalv1.RunSuiteRequest{SuiteId: "ghost"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestCompareRunsBuildsDeltas(t *testing.T) {
	store := newFakeStore()
	store.runs["a"] = &evalv1.EvalRun{
		RunId: "a", SuiteId: "s", Tag: "rerank-off",
		Results: []*evalv1.CaseResult{
			{CaseId: "c1", Outcome: "below_expectation", ObservedTopScore: 0.55},
			{CaseId: "gib", Outcome: "unexpected_hit", ObservedTopScore: 0.52},
		},
	}
	store.runs["b"] = &evalv1.EvalRun{
		RunId: "b", SuiteId: "s", Tag: "cross-encoder",
		Results: []*evalv1.CaseResult{
			{CaseId: "c1", Outcome: "met", ObservedTopScore: 0.82},
			{CaseId: "gib", Outcome: "met", ObservedTopScore: 0.02},
		},
	}
	client := newClient(t, store, &fakeRunner{})

	resp, err := client.CompareRuns(context.Background(), connect.NewRequest(&evalv1.CompareRunsRequest{RunA: "a", RunB: "b"}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetDeltas(), 2)
	byCase := map[string]*evalv1.CaseDelta{}
	for _, d := range resp.Msg.GetDeltas() {
		byCase[d.GetCaseId()] = d
	}
	require.Equal(t, "below_expectation", byCase["c1"].GetOutcomeA())
	require.Equal(t, "met", byCase["c1"].GetOutcomeB())
	require.InDelta(t, 0.52, byCase["gib"].GetTopScoreA(), 1e-9)
	require.InDelta(t, 0.02, byCase["gib"].GetTopScoreB(), 1e-9)
}

func TestCompareRunsMissingRunNotFound(t *testing.T) {
	store := newFakeStore()
	store.runs["a"] = &evalv1.EvalRun{RunId: "a"}
	client := newClient(t, store, &fakeRunner{})
	_, err := client.CompareRuns(context.Background(), connect.NewRequest(&evalv1.CompareRunsRequest{RunA: "a", RunB: "missing"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}
