package eval_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"

	"search-hub/internal/eval"

	"github.com/vrooli/api-core/scheduletest"
)

// fakeResolver returns a canned descriptor (or an error) for the runner's
// registry read seam.
type fakeResolver struct {
	desc *registryv1.ProviderDescriptor
	err  error
}

func (f fakeResolver) Get(_ context.Context, _ string) (*registryv1.ProviderDescriptor, error) {
	return f.desc, f.err
}

// fakeClient returns canned hits per query and a canned snapshot.
type fakeClient struct {
	byQuery  map[string][]*routingv1.SearchHit
	errQuery map[string]error
	snap     *evalv1.ConfigSnapshot
}

func (f fakeClient) Search(_ context.Context, _ *registryv1.ProviderDescriptor, query string, _ int32, _ eval.SearchCallOptions) ([]*routingv1.SearchHit, error) {
	if f.errQuery != nil {
		if err, ok := f.errQuery[query]; ok {
			return nil, err
		}
	}
	return f.byQuery[query], nil
}

func (f fakeClient) Snapshot(_ context.Context, _ *registryv1.ProviderDescriptor) *evalv1.ConfigSnapshot {
	return f.snap
}

func hit(id string, score float64) *routingv1.SearchHit {
	return &routingv1.SearchHit{Id: id, Title: id, Score: score}
}

func newRunner(client eval.ProviderClient) *eval.Runner {
	clk := scheduletest.New(time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC))
	resolver := fakeResolver{desc: &registryv1.ProviderDescriptor{ProviderId: "p"}}
	var n int
	return eval.NewRunner(resolver, client, clk, func() string { n++; return "run-fixed" })
}

func suiteWith(cases ...*evalv1.EvalCase) *evalv1.EvalSuite {
	return &evalv1.EvalSuite{SuiteId: "s", ProviderId: "p", Cases: cases}
}

func TestRunner_OutcomeLabels(t *testing.T) {
	cases := []*evalv1.EvalCase{
		// met: expected id within k, top score in band.
		{CaseId: "strong-met", Query: "q-met", Tags: []string{"strong"}, ExpectIds: []string{"a"}, ExpectWithinTopK: 3, ExpectMinScore: 0.6},
		// below: expected id absent.
		{CaseId: "missing", Query: "q-missing", ExpectIds: []string{"zzz"}, ExpectWithinTopK: 3},
		// below: top score under min.
		{CaseId: "low-score", Query: "q-low", ExpectIds: []string{"a"}, ExpectMinScore: 0.9},
		// n/a: no expectations.
		{CaseId: "nada", Query: "q-nada", Tags: []string{"weak-real"}},
		// gibberish met: nothing exceeds the ceiling.
		{CaseId: "gib-ok", Query: "q-gib-ok", Tags: []string{"gibberish"}, ExpectNoStrongHit: true, ExpectMaxScore: 0.4},
		// gibberish unexpected_hit: a hit exceeds the ceiling.
		{CaseId: "gib-bad", Query: "q-gib-bad", Tags: []string{"gibberish"}, ExpectNoStrongHit: true, ExpectMaxScore: 0.4},
	}
	client := fakeClient{
		snap: &evalv1.ConfigSnapshot{RerankerLeg: "cross-encoder:bge", RerankEnabled: true},
		byQuery: map[string][]*routingv1.SearchHit{
			"q-met":     {hit("a", 0.82), hit("b", 0.5)},
			"q-missing": {hit("x", 0.7), hit("y", 0.6)},
			"q-low":     {hit("a", 0.42)},
			"q-nada":    {hit("a", 0.3)},
			"q-gib-ok":  {hit("a", 0.31), hit("b", 0.2)},
			"q-gib-bad": {hit("a", 0.77)},
		},
	}
	run, err := newRunner(client).Run(context.Background(), suiteWith(cases...), "cross-encoder", 0)
	require.NoError(t, err)
	require.Equal(t, "run-fixed", run.GetRunId())
	require.Equal(t, "cross-encoder", run.GetTag())
	require.Equal(t, "cross-encoder:bge", run.GetConfig().GetRerankerLeg())

	got := map[string]*evalv1.CaseResult{}
	for _, cr := range run.GetResults() {
		got[cr.GetCaseId()] = cr
	}
	require.Equal(t, "met", got["strong-met"].GetOutcome())
	require.EqualValues(t, 1, got["strong-met"].GetExpectedRank())
	require.Equal(t, "below_expectation", got["missing"].GetOutcome())
	require.EqualValues(t, 0, got["missing"].GetExpectedRank())
	require.Equal(t, "below_expectation", got["low-score"].GetOutcome())
	require.Equal(t, "n/a", got["nada"].GetOutcome())
	require.Equal(t, "met", got["gib-ok"].GetOutcome())
	require.Equal(t, "unexpected_hit", got["gib-bad"].GetOutcome())

	// Aggregate: 3 met (strong-met, gib-ok, gib-bad? no — gib-bad is unexpected_hit).
	agg := run.GetAggregate()
	require.EqualValues(t, 6, agg.GetCases())
	require.EqualValues(t, 2, agg.GetMet()) // strong-met + gib-ok
	require.EqualValues(t, 2, agg.GetBelow())
	require.InDelta(t, 0.82, agg.GetMeanStrongTop1(), 1e-9)
	require.InDelta(t, 0.77, agg.GetMaxGibberishScore(), 1e-9) // worst junk leakage
}

func TestRunner_SearchErrorDegradesCaseNotRun(t *testing.T) {
	client := fakeClient{
		errQuery: map[string]error{"boom": errors.New("provider down")},
		byQuery:  map[string][]*routingv1.SearchHit{"ok": {hit("a", 0.8)}},
	}
	suite := suiteWith(
		&evalv1.EvalCase{CaseId: "bad", Query: "boom", ExpectMinScore: 0.5},
		&evalv1.EvalCase{CaseId: "good", Query: "ok", ExpectIds: []string{"a"}, ExpectMinScore: 0.5},
	)
	run, err := newRunner(client).Run(context.Background(), suite, "t", 0)
	require.NoError(t, err, "a single failing case must not prevent the run from being persisted")
	got := map[string]string{}
	for _, cr := range run.GetResults() {
		got[cr.GetCaseId()] = cr.GetOutcome()
	}
	require.Equal(t, "error", got["bad"])
	require.Equal(t, "met", got["good"])
	require.Equal(t, "provider down", run.GetResults()[0].GetOutcomeReason())
	require.True(t, run.GetDegraded())
	require.Equal(t, "provider down", run.GetDegradedReason())
	require.EqualValues(t, 1, run.GetAggregate().GetGradedCases())
}

func TestRunner_ExcludesCandidateCasesFromProviderAndAggregate(t *testing.T) {
	client := fakeClient{
		errQuery: map[string]error{"candidate": errors.New("candidate must not be searched")},
		byQuery:  map[string][]*routingv1.SearchHit{"reviewed": {hit("a", 0.9)}},
	}
	run, err := newRunner(client).Run(context.Background(), suiteWith(
		&evalv1.EvalCase{CaseId: "candidate", Query: "candidate", Status: "candidate", Tags: []string{"candidate"}, ExpectIds: []string{"a"}},
		&evalv1.EvalCase{CaseId: "reviewed", Query: "reviewed", ExpectIds: []string{"a"}},
	), "candidate-exclusion", 0)
	require.NoError(t, err)
	results := map[string]*evalv1.CaseResult{}
	for _, result := range run.GetResults() {
		results[result.GetCaseId()] = result
	}
	require.Equal(t, "n/a", results["candidate"].GetOutcome())
	require.Equal(t, "candidate excluded from acceptance denominator", results["candidate"].GetOutcomeReason())
	require.Equal(t, "met", results["reviewed"].GetOutcome())
	require.False(t, run.GetDegraded(), "candidate must not invoke the provider")
	require.EqualValues(t, 2, run.GetAggregate().GetCases())
	require.EqualValues(t, 1, run.GetAggregate().GetGradedCases())
	require.EqualValues(t, 1, run.GetAggregate().GetMet())
	require.InDelta(t, 1.0, run.GetAggregate().GetPassRate(), 1e-9)
}

func TestRunner_InfrastructureFailureIsUnavailableAndExcluded(t *testing.T) {
	client := fakeClient{
		errQuery: map[string]error{"down": errors.New("request: provider returned HTTP 503")},
		byQuery:  map[string][]*routingv1.SearchHit{"ok": {hit("a", 0.8)}},
	}
	run, err := newRunner(client).Run(context.Background(), suiteWith(
		&evalv1.EvalCase{CaseId: "down", Query: "down", ExpectIds: []string{"a"}},
		&evalv1.EvalCase{CaseId: "ok", Query: "ok", ExpectIds: []string{"a"}},
	), "t", 0)
	require.NoError(t, err)
	require.Equal(t, "unavailable", run.GetResults()[0].GetOutcome())
	require.Len(t, run.GetUnavailableCases(), 1)
	require.Contains(t, run.GetUnavailableCases()[0].GetReason(), "503")
	require.EqualValues(t, 1, run.GetAggregate().GetGradedCases())
	require.EqualValues(t, 1, run.GetAggregate().GetUnavailableCases())
	require.InDelta(t, 1.0, run.GetAggregate().GetPassRate(), 1e-9)
}

func TestRunner_AllInfrastructureFailuresHaveNoQualityDenominator(t *testing.T) {
	run, err := newRunner(fakeClient{errQuery: map[string]error{
		"one": context.DeadlineExceeded,
		"two": errors.New("qdrant: connect: connection refused"),
	}}).Run(context.Background(), suiteWith(
		&evalv1.EvalCase{CaseId: "one", Query: "one", ExpectIds: []string{"a"}},
		&evalv1.EvalCase{CaseId: "two", Query: "two", ExpectIds: []string{"a"}},
	), "t", 0)
	require.NoError(t, err)
	require.EqualValues(t, 0, run.GetAggregate().GetGradedCases())
	require.EqualValues(t, 2, run.GetAggregate().GetUnavailableCases())
	require.Empty(t, run.GetAggregate().GetPassRate())
	require.NotEmpty(t, run.GetUnavailableReason())
}

func TestRunner_ZeroGradedCasesDegradesRun(t *testing.T) {
	run, err := newRunner(fakeClient{byQuery: map[string][]*routingv1.SearchHit{
		"ungraded": {hit("a", 0.2)},
	}}).Run(context.Background(), suiteWith(&evalv1.EvalCase{
		CaseId: "ungraded", Query: "ungraded", Tags: []string{"weak-real"},
	}), "t", 0)
	require.NoError(t, err)
	require.Equal(t, "n/a", run.GetResults()[0].GetOutcome())
	require.EqualValues(t, 0, run.GetAggregate().GetGradedCases())
	require.True(t, run.GetDegraded())
	require.Equal(t, "run produced zero graded cases", run.GetDegradedReason())
}

func TestRunner_UnresolvedProviderErrors(t *testing.T) {
	r := eval.NewRunner(
		fakeResolver{err: eval.ErrSuiteNotFound{SuiteID: "p"}},
		fakeClient{},
		scheduletest.New(time.Now().UTC()),
		nil,
	)
	_, err := r.Run(context.Background(), suiteWith(&evalv1.EvalCase{CaseId: "c", Query: "q"}), "t", 0)
	require.Error(t, err, "an unregistered provider is a real error (the suite can't run)")
}
