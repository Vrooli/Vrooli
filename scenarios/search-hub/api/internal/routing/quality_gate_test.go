package routing_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	"search-hub/internal/routing"
)

type fakeEvalQuality struct {
	evidence routing.EvalQualityEvidence
}

func (f fakeEvalQuality) LatestProviderEval(context.Context, string) (routing.EvalQualityEvidence, error) {
	return f.evidence, nil
}

func TestAutomaticRoutingWithholdsFreshJunkLeakButExplicitReachesProvider(t *testing.T) {
	provider := &registryv1.ProviderDescriptor{
		ProviderId:    "synthetic.records",
		ProviderGroup: "synthetic",
		Type:          "record",
		Description:   "Synthetic records for quality-gate testing",
		State:         registryv1.ProviderState_PROVIDER_STATE_ACTIVE,
		Endpoint:      httpJSON("synthetic", "/search", `{"query":"{{query}}","limit":{{limit}}}`),
		ResultMapping: &registryv1.ResultMapping{ResultsPath: "results", IdField: "id", TitleField: "title", ScoreField: "score", ScoreScale: registryv1.ScoreScale_SCORE_SCALE_COSINE_0_1},
	}
	quality := fakeEvalQuality{evidence: routing.EvalQualityEvidence{
		RunID: "run-junk", Fresh: true, MeanStrongTop1: 0.227, MaxGibberishScore: 0.566, GibberishLeak: true,
	}}
	base := routing.Deps{
		Lister:   &fakeLister{providers: []*registryv1.ProviderDescriptor{provider}},
		Resolver: staticResolver{urls: map[string]string{"synthetic": "http://synthetic.test"}},
		Doer: routeDoer{byURL: map[string]cannedResponse{
			"http://synthetic.test/search": {status: 200, body: `{"results":[{"id":"one","title":"one","score":0.9}]}`},
		}},
		Classifier:    &fakeClassifier{result: routing.ClassifyResult{ProviderIDs: []string{provider.GetProviderId()}, Confidence: 0.99}},
		EvalQuality:   quality,
		QueryTimeout:  time.Second,
		RerankTimeout: time.Second,
	}

	automatic, err := routing.NewRouter(base).Query(context.Background(), &routingv1.QueryRequest{Query: "find records", Explain: true})
	require.NoError(t, err)
	require.Empty(t, automatic.GetCorporaSearched())
	require.Contains(t, strings.Join(automatic.GetRoutingExplanation(), "\n"), "withheld (junk leak)")
	require.Contains(t, strings.Join(automatic.GetRoutingExplanation(), "\n"), "run-junk")

	explicit, err := routing.NewRouter(base).Query(context.Background(), &routingv1.QueryRequest{Query: "find records", Group: "synthetic", Explain: true})
	require.NoError(t, err)
	require.Equal(t, []string{"synthetic.records"}, explicit.GetCorporaSearched())
}

func TestDegradedEvalEvidenceDoesNotWithholdProvider(t *testing.T) {
	provider := &registryv1.ProviderDescriptor{
		ProviderId:    "synthetic.records",
		ProviderGroup: "synthetic",
		Type:          "record",
		Description:   "Synthetic records for quality-gate testing",
		State:         registryv1.ProviderState_PROVIDER_STATE_ACTIVE,
		Endpoint:      httpJSON("synthetic", "/search", `{"query":"{{query}}","limit":{{limit}}}`),
		ResultMapping: &registryv1.ResultMapping{ResultsPath: "results", IdField: "id", TitleField: "title", ScoreField: "score", ScoreScale: registryv1.ScoreScale_SCORE_SCALE_COSINE_0_1},
	}
	r := routing.NewRouter(routing.Deps{
		Lister:   &fakeLister{providers: []*registryv1.ProviderDescriptor{provider}},
		Resolver: staticResolver{urls: map[string]string{"synthetic": "http://synthetic.test"}},
		Doer: routeDoer{byURL: map[string]cannedResponse{
			"http://synthetic.test/search": {status: 200, body: `{"results":[{"id":"one","title":"one","score":0.9}]}`},
		}},
		Classifier:   &fakeClassifier{result: routing.ClassifyResult{ProviderIDs: []string{provider.GetProviderId()}, Confidence: 0.99}},
		EvalQuality:  fakeEvalQuality{evidence: routing.EvalQualityEvidence{Fresh: true, Degraded: true, GibberishLeak: true}},
		QueryTimeout: time.Second,
	})
	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "find records"})
	require.NoError(t, err)
	require.Equal(t, []string{"synthetic.records"}, resp.GetCorporaSearched())
}

func TestQualityGateOptOutIsOwnerRecordedAndDoesNotWithhold(t *testing.T) {
	provider := &registryv1.ProviderDescriptor{
		ProviderId:           "synthetic.records",
		ProviderGroup:        "synthetic",
		Type:                 "record",
		Description:          "Synthetic records for quality-gate testing",
		State:                registryv1.ProviderState_PROVIDER_STATE_ACTIVE,
		JunkLeakOptOutReason: "external corpus has an intentionally broad query distribution",
		Endpoint:             httpJSON("synthetic", "/search", `{"query":"{{query}}","limit":{{limit}}}`),
		ResultMapping:        &registryv1.ResultMapping{ResultsPath: "results", IdField: "id", TitleField: "title", ScoreField: "score", ScoreScale: registryv1.ScoreScale_SCORE_SCALE_COSINE_0_1},
	}
	r := routing.NewRouter(routing.Deps{
		Lister:   &fakeLister{providers: []*registryv1.ProviderDescriptor{provider}},
		Resolver: staticResolver{urls: map[string]string{"synthetic": "http://synthetic.test"}},
		Doer: routeDoer{byURL: map[string]cannedResponse{
			"http://synthetic.test/search": {status: 200, body: `{"results":[{"id":"one","title":"one","score":0.9}]}`},
		}},
		Classifier:   &fakeClassifier{result: routing.ClassifyResult{ProviderIDs: []string{provider.GetProviderId()}, Confidence: 0.99}},
		EvalQuality:  fakeEvalQuality{evidence: routing.EvalQualityEvidence{Fresh: true, GibberishLeak: true, MeanStrongTop1: 0.227, MaxGibberishScore: 0.566}},
		QueryTimeout: time.Second,
	})
	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "find records", Explain: true})
	require.NoError(t, err)
	require.Equal(t, []string{"synthetic.records"}, resp.GetCorporaSearched())
	require.Contains(t, strings.Join(resp.GetRoutingExplanation(), "\n"), "quality gate opted out")
}

func TestAutomaticRoutingRequiresRecentReviewedSuiteEvidence(t *testing.T) {
	provider := &registryv1.ProviderDescriptor{
		ProviderId:    "synthetic.records",
		ProviderGroup: "synthetic",
		Type:          "record",
		Description:   "Synthetic records for lifecycle evidence testing",
		Lifecycle:     registryv1.Lifecycle_LIFECYCLE_PRODUCTION,
		State:         registryv1.ProviderState_PROVIDER_STATE_ACTIVE,
		Endpoint:      httpJSON("synthetic", "/search", `{"query":"{{query}}","limit":{{limit}}}`),
		ResultMapping: &registryv1.ResultMapping{ResultsPath: "results", IdField: "id", TitleField: "title", ScoreField: "score", ScoreScale: registryv1.ScoreScale_SCORE_SCALE_COSINE_0_1},
	}
	r := routing.NewRouter(routing.Deps{
		Lister:   &fakeLister{providers: []*registryv1.ProviderDescriptor{provider}},
		Resolver: staticResolver{urls: map[string]string{"synthetic": "http://synthetic.test"}},
		Doer: routeDoer{byURL: map[string]cannedResponse{
			"http://synthetic.test/search": {status: 200, body: `{"results":[{"id":"one","title":"one","score":0.9}]}`},
		}},
		Classifier:  &fakeClassifier{result: routing.ClassifyResult{ProviderIDs: []string{provider.GetProviderId()}, Confidence: 0.99}},
		EvalQuality: fakeEvalQuality{evidence: routing.EvalQualityEvidence{EvidenceAvailable: true}},
	})
	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "find records", Explain: true})
	require.NoError(t, err)
	require.Empty(t, resp.GetCorporaSearched())
	require.Contains(t, strings.Join(resp.GetRoutingExplanation(), "\n"), "no suite")
}
