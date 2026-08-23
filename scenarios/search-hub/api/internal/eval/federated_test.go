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
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/shared"

	"search-hub/internal/eval"

	"github.com/vrooli/api-core/scheduletest"
)

type fakeFederatedQuery struct {
	responses map[string]*routingv1.QueryResponse
	errors    map[string]error
	snapshot  *evalv1.ConfigSnapshot
}

type fakeRoutability struct {
	status *routingv1.StatusResponse
}

func (f fakeRoutability) Status(context.Context) (*routingv1.StatusResponse, error) {
	return f.status, nil
}

func (f fakeFederatedQuery) Query(_ context.Context, req *routingv1.QueryRequest) (*routingv1.QueryResponse, error) {
	if err := f.errors[req.GetQuery()]; err != nil {
		return nil, err
	}
	return f.responses[req.GetQuery()], nil
}

func (f fakeFederatedQuery) Snapshot(context.Context) *evalv1.ConfigSnapshot {
	return f.snapshot
}

func federatedHit(provider, id string, score float64) *routingv1.SearchHit {
	return &routingv1.SearchHit{ProviderId: provider, Id: id, Score: score}
}

func newFederatedRunner(query eval.QueryClient) *eval.FederatedRunner {
	return eval.NewFederatedRunner(fakeResolver{desc: &registryv1.ProviderDescriptor{ProviderId: "owner.leaf"}}, query,
		scheduletest.New(time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)), func() string { return "federated-fixed" })
}

func federatedSuite(cases ...*evalv1.EvalCase) *evalv1.EvalSuite {
	return &evalv1.EvalSuite{SuiteId: "suite", ProviderId: "owner.leaf", Cases: cases}
}

func TestFederatedRunnerLabelsRoutingRankMarginAndDegradation(t *testing.T) {
	run, err := newFederatedRunner(fakeFederatedQuery{
		responses: map[string]*routingv1.QueryResponse{
			"met":       {CorporaSearched: []string{"owner.leaf"}, Ranked: []*routingv1.SearchHit{federatedHit("owner.leaf", "wanted", 0.8), federatedHit("owner.leaf", "other", 0.4)}, RoutingTrace: &sharedv1.RoutingTrace{StrategyName: "lexical-cross-encoder", IndexStatus: "not_used", SelectedProviderIds: []string{"owner.leaf"}, ReturnedEvidence: "hits"}},
			"misrouted": {CorporaSearched: []string{"sibling"}, Ranked: []*routingv1.SearchHit{federatedHit("sibling", "wanted", 0.9)}},
			"sibling":   {CorporaSearched: []string{"sibling"}, Ranked: []*routingv1.SearchHit{federatedHit("sibling", "answer", 0.9)}},
			"thin":      {CorporaSearched: []string{"owner.leaf"}, Ranked: []*routingv1.SearchHit{federatedHit("owner.leaf", "wanted", 0.8), federatedHit("owner.leaf", "other", 0.795)}},
			"empty":     {CorporaSearched: []string{"owner.leaf"}},
		},
		errors: map[string]error{"degraded": errors.New("routing timeout")},
	}).Run(context.Background(), federatedSuite(
		&evalv1.EvalCase{CaseId: "met", Query: "met", ExpectIds: []string{"wanted"}, ExpectWithinTopK: 1},
		&evalv1.EvalCase{CaseId: "misrouted", Query: "misrouted", ExpectIds: []string{"wanted"}},
		&evalv1.EvalCase{CaseId: "sibling", Query: "sibling", ExpectIds: []string{"wanted"}},
		&evalv1.EvalCase{CaseId: "thin", Query: "thin", ExpectIds: []string{"wanted"}, ExpectMinMargin: 0.02},
		&evalv1.EvalCase{CaseId: "empty", Query: "empty", ExpectIds: []string{"wanted"}},
		&evalv1.EvalCase{CaseId: "degraded", Query: "degraded", ExpectIds: []string{"wanted"}},
	), "baseline", 10)
	require.NoError(t, err)
	require.Equal(t, "federated", run.GetTier())
	got := map[string]*evalv1.CaseResult{}
	for _, result := range run.GetResults() {
		got[result.GetCaseId()] = result
	}
	require.Equal(t, "met", got["met"].GetOutcome())
	require.True(t, got["met"].GetProviderRouted())
	require.Equal(t, "lexical-cross-encoder", got["met"].GetRoutingTrace().GetStrategyName())
	require.Equal(t, []string{"owner.leaf"}, got["met"].GetRoutingTrace().GetSelectedProviderIds())
	require.InDelta(t, 0.5, got["met"].GetMargin(), 1e-9)
	require.Equal(t, "answered_by_sibling", got["misrouted"].GetOutcome())
	require.False(t, got["misrouted"].GetProviderRouted())
	require.Equal(t, "misrouted", got["sibling"].GetOutcome())
	require.False(t, got["sibling"].GetProviderRouted())
	require.Equal(t, "no_result", got["empty"].GetOutcome())
	require.Equal(t, "thin_margin", got["thin"].GetOutcome())
	require.Equal(t, "error", got["degraded"].GetOutcome())
	require.Equal(t, "routing timeout", got["degraded"].GetOutcomeReason())
	require.Contains(t, got["degraded"].GetRoutingTrace().GetUnavailableReason(), "query_error")
	require.True(t, run.GetDegraded())
}

func TestFederatedRunnerPersistsStratifiedRoutingEvidence(t *testing.T) {
	strata := []struct {
		name  string
		query string
	}{
		{name: "exact_identifier", query: "sqliteDemotionStore"},
		{name: "implementation_location", query: "where is provider demotion computed"},
		{name: "contract", query: "what is the search provider registration contract"},
		{name: "command", query: "how do I run the scenario test suite"},
		{name: "documentation", query: "where are the search hub retrieval guarantees documented"},
		{name: "workflow", query: "what happens when a provider becomes stale"},
		{name: "paraphrase", query: "find the code that decides whether a corpus can be routed"},
	}
	responses := make(map[string]*routingv1.QueryResponse, len(strata))
	cases := make([]*evalv1.EvalCase, 0, len(strata))
	for _, stratum := range strata {
		providerID := "owner.leaf"
		responses[stratum.query] = &routingv1.QueryResponse{
			CorporaSearched: []string{providerID},
			Ranked:          []*routingv1.SearchHit{federatedHit(providerID, stratum.name, 0.9), federatedHit(providerID, "background", 0.1)},
			RoutingTrace: &sharedv1.RoutingTrace{
				StrategyName:          "semantic-cross-encoder",
				IndexStatus:           "available",
				DenseTopProviderIds:   []string{providerID, "sibling.leaf"},
				LexicalTopProviderIds: []string{providerID, "sibling.leaf"},
				Candidates: []*sharedv1.ProviderRoutingEvidence{{
					ProviderId:        providerID,
					DenseRank:         1,
					DenseScore:        0.91,
					LexicalRank:       1,
					LexicalScore:      4.0,
					InEvidenceUnion:   true,
					CrossEncoderRank:  1,
					CrossEncoderScore: 0.95,
					Selected:          true,
				}},
				SelectedProviderIds: []string{providerID},
				SelectionReason:     "cross_encoder_guarded_lexical",
				ReturnedEvidence:    "hits",
			},
		}
		cases = append(cases, &evalv1.EvalCase{
			CaseId:             "diagnostic." + stratum.name,
			Query:              stratum.query,
			ExpectedProviderId: providerID,
			ExpectIds:          []string{stratum.name},
			ExpectWithinTopK:   1,
			Tags:               []string{"routing-diagnostic", stratum.name},
		})
	}

	run, err := newFederatedRunner(fakeFederatedQuery{responses: responses}).Run(
		context.Background(),
		&evalv1.EvalSuite{SuiteId: eval.RouterSuiteID, Cases: cases},
		"stratified-routing-diagnostics",
		10,
	)
	require.NoError(t, err)
	require.Len(t, run.GetResults(), len(strata))
	for _, result := range run.GetResults() {
		require.Equal(t, "met", result.GetOutcome(), result.GetCaseId())
		trace := result.GetRoutingTrace()
		require.NotNil(t, trace, result.GetCaseId())
		require.NotEmpty(t, trace.GetDenseTopProviderIds(), result.GetCaseId())
		require.NotEmpty(t, trace.GetLexicalTopProviderIds(), result.GetCaseId())
		require.NotEmpty(t, trace.GetCandidates(), result.GetCaseId())
		require.Equal(t, int32(1), trace.GetCandidates()[0].GetDenseRank(), result.GetCaseId())
		require.Equal(t, int32(1), trace.GetCandidates()[0].GetLexicalRank(), result.GetCaseId())
		require.True(t, trace.GetCandidates()[0].GetInEvidenceUnion(), result.GetCaseId())
		require.Equal(t, int32(1), trace.GetCandidates()[0].GetCrossEncoderRank(), result.GetCaseId())
		require.Equal(t, []string{"owner.leaf"}, trace.GetSelectedProviderIds(), result.GetCaseId())
		require.Equal(t, "hits", trace.GetReturnedEvidence(), result.GetCaseId())
		require.Equal(t, "owner.leaf", result.GetExpectedProviderId(), result.GetCaseId())
	}
}

func TestFederatedRunnerGradesNegativesAndCapturesConfig(t *testing.T) {
	query := fakeFederatedQuery{
		responses: map[string]*routingv1.QueryResponse{
			"negative": {
				CorporaSearched: []string{"owner.leaf"},
				Ranked:          []*routingv1.SearchHit{federatedHit("owner.leaf", "irrelevant", 0.2)},
				SelectorLeg:     "llm",
				RerankerLeg:     "cross-encoder:bge",
			},
			"positive": {
				CorporaSearched: []string{"owner.leaf"},
				Ranked:          []*routingv1.SearchHit{federatedHit("owner.leaf", "wanted", 0.9)},
				SelectorLeg:     "llm",
				RerankerLeg:     "cross-encoder:bge",
			},
		},
		snapshot: &evalv1.ConfigSnapshot{RerankerLeg: "cross-encoder:bge", EmbedModel: "nomic-embed-text", IndexedCount: 42},
	}
	run, err := newFederatedRunner(query).Run(context.Background(), federatedSuite(
		&evalv1.EvalCase{CaseId: "negative", Query: "negative", ExpectNoStrongHit: true, ExpectMaxScore: 0.3},
		&evalv1.EvalCase{CaseId: "positive", Query: "positive", ExpectIds: []string{"wanted"}, ExpectWithinTopK: 1},
	), "phase-3", 10)
	require.NoError(t, err)
	require.Equal(t, "met", run.GetResults()[0].GetOutcome())
	require.Equal(t, "llm", run.GetConfig().GetSelectorLeg())
	require.Equal(t, "cross-encoder:bge", run.GetConfig().GetRerankerLeg())
	require.Equal(t, "nomic-embed-text", run.GetConfig().GetEmbedModel())
	require.Equal(t, int32(42), run.GetConfig().GetIndexedCount())
	require.Equal(t, int32(2), run.GetAggregate().GetGradedCases())
}

func TestFederatedRunnerDoesNotDiscardGradeableDegradedResponses(t *testing.T) {
	run, err := newFederatedRunner(fakeFederatedQuery{responses: map[string]*routingv1.QueryResponse{
		"degraded-hit": {
			CorporaSearched: []string{"owner.leaf"},
			Ranked:          []*routingv1.SearchHit{federatedHit("owner.leaf", "wanted", 0.9), federatedHit("owner.leaf", "other", 0.1)},
			Degraded:        true,
			RoutingExplanation: []string{
				"reranker down; lexical leg served the response",
			},
		},
	}}).Run(context.Background(), federatedSuite(&evalv1.EvalCase{
		CaseId: "degraded-hit", Query: "degraded-hit", ExpectIds: []string{"wanted"}, ExpectWithinTopK: 1,
	}), "phase-3", 10)
	require.NoError(t, err)
	require.True(t, run.GetDegraded())
	require.Equal(t, "met", run.GetResults()[0].GetOutcome())
	require.Equal(t, int32(1), run.GetAggregate().GetGradedCases())
	require.Equal(t, float64(1), run.GetAggregate().GetPassRate())
}

func TestFederatedRunnerExcludesPolicyWithheldOwnersFromRoutingQuality(t *testing.T) {
	query := fakeFederatedQuery{responses: map[string]*routingv1.QueryResponse{
		"eligible": {
			CorporaSearched: []string{"eligible.leaf"},
			Ranked:          []*routingv1.SearchHit{federatedHit("eligible.leaf", "wanted", 0.9), federatedHit("eligible.leaf", "other", 0.1)},
		},
	}}
	runner := eval.NewFederatedRunnerWithRoutability(
		fakeResolver{},
		query,
		scheduletest.New(time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)),
		func() string { return "policy-aware-federated-fixed" },
		fakeRoutability{status: &routingv1.StatusResponse{Providers: []*routingv1.ProviderHealth{
			{ProviderId: "eligible.leaf", AutomaticEligible: true},
			{ProviderId: "withheld.leaf", AutomaticExclusionReason: "stale index: age=48h exceeds freshness budget=24h"},
		}}},
	)
	run, err := runner.Run(context.Background(), &evalv1.EvalSuite{
		SuiteId: eval.RouterSuiteID,
		Cases: []*evalv1.EvalCase{
			{CaseId: "eligible", Query: "eligible", ExpectedProviderId: "eligible.leaf", ExpectIds: []string{"wanted"}, ExpectWithinTopK: 1},
			{CaseId: "withheld", Query: "withheld", ExpectedProviderId: "withheld.leaf", ExpectIds: []string{"unavailable"}, ExpectWithinTopK: 1},
		},
	}, "policy-aware", 10)
	require.NoError(t, err)
	results := map[string]*evalv1.CaseResult{}
	for _, result := range run.GetResults() {
		results[result.GetCaseId()] = result
	}
	require.Equal(t, "met", results["eligible"].GetOutcome())
	require.Equal(t, "unavailable", results["withheld"].GetOutcome())
	require.Contains(t, results["withheld"].GetOutcomeReason(), "stale index")
	require.EqualValues(t, 1, run.GetAggregate().GetGradedCases())
	require.EqualValues(t, 1, run.GetAggregate().GetUnavailableCases())
	require.Equal(t, 1.0, run.GetAggregate().GetRoutingPrecision())
}

func TestFederatedRunnerUsesDefaultMarginAndRejectsUnknownProvider(t *testing.T) {
	run, err := newFederatedRunner(fakeFederatedQuery{responses: map[string]*routingv1.QueryResponse{
		"default": {CorporaSearched: []string{"owner.leaf"}, Ranked: []*routingv1.SearchHit{federatedHit("owner.leaf", "wanted", 0.8), federatedHit("owner.leaf", "other", 0.7)}},
	}}).Run(context.Background(), federatedSuite(&evalv1.EvalCase{CaseId: "default", Query: "default", ExpectIds: []string{"wanted"}}), "t", 10)
	require.NoError(t, err)
	require.Equal(t, "met", run.GetResults()[0].GetOutcome())
	require.Contains(t, run.GetResults()[0].GetOutcomeReason(), "default margin floor")
	unknown := eval.NewFederatedRunner(fakeResolver{err: errors.New("provider is not registered")}, fakeFederatedQuery{}, scheduletest.New(time.Now()), nil)
	_, err = unknown.Run(context.Background(), federatedSuite(), "t", 10)
	require.Error(t, err)
}
