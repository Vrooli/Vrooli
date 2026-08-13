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

type fakeFederatedQuery struct {
	responses map[string]*routingv1.QueryResponse
	errors    map[string]error
}

func (f fakeFederatedQuery) Query(_ context.Context, req *routingv1.QueryRequest) (*routingv1.QueryResponse, error) {
	if err := f.errors[req.GetQuery()]; err != nil {
		return nil, err
	}
	return f.responses[req.GetQuery()], nil
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
			"met":       {CorporaSearched: []string{"owner.leaf"}, Ranked: []*routingv1.SearchHit{federatedHit("owner.leaf", "wanted", 0.8), federatedHit("owner.leaf", "other", 0.4)}},
			"misrouted": {CorporaSearched: []string{"sibling"}, Ranked: []*routingv1.SearchHit{federatedHit("sibling", "wanted", 0.9)}},
			"sibling":   {CorporaSearched: []string{"sibling"}, Ranked: []*routingv1.SearchHit{federatedHit("sibling", "answer", 0.9)}},
			"thin":      {CorporaSearched: []string{"owner.leaf"}, Ranked: []*routingv1.SearchHit{federatedHit("owner.leaf", "wanted", 0.8), federatedHit("owner.leaf", "other", 0.795)}},
		},
		errors: map[string]error{"degraded": errors.New("routing timeout")},
	}).Run(context.Background(), federatedSuite(
		&evalv1.EvalCase{CaseId: "met", Query: "met", ExpectIds: []string{"wanted"}, ExpectWithinTopK: 1},
		&evalv1.EvalCase{CaseId: "misrouted", Query: "misrouted", ExpectIds: []string{"wanted"}},
		&evalv1.EvalCase{CaseId: "sibling", Query: "sibling", ExpectIds: []string{"wanted"}},
		&evalv1.EvalCase{CaseId: "thin", Query: "thin", ExpectIds: []string{"wanted"}, ExpectMinMargin: 0.02},
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
	require.InDelta(t, 0.5, got["met"].GetMargin(), 1e-9)
	require.Equal(t, "misrouted", got["misrouted"].GetOutcome())
	require.Contains(t, got["misrouted"].GetOutcomeReason(), `expected provider "owner.leaf" absent from corpora_searched`)
	require.Equal(t, "answered_by_sibling", got["sibling"].GetOutcome())
	require.Contains(t, got["sibling"].GetOutcomeReason(), `expected provider "owner.leaf" absent from corpora_searched`)
	require.Equal(t, "thin_margin", got["thin"].GetOutcome())
	require.Equal(t, "error", got["degraded"].GetOutcome())
	require.Equal(t, "routing timeout", got["degraded"].GetOutcomeReason())
	require.True(t, run.GetDegraded())
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
