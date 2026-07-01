package routing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"

	"search-hub/internal/routing"
)

// recordingRecorder captures the samples the router emits so telemetry wiring
// is testable without the metrics store.
type recordingRecorder struct {
	samples []routing.TelemetrySample
}

func (r *recordingRecorder) Record(_ context.Context, s routing.TelemetrySample) {
	r.samples = append(r.samples, s)
}

func TestQueryRecordsTelemetry(t *testing.T) {
	rec := &recordingRecorder{}
	r := routing.NewRouter(routing.Deps{
		Lister:   threeProviderLister(),
		Resolver: threeProviderResolver(),
		Doer:     threeProviderDoer(),
		Recorder: rec,
	})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "restart a scenario", All: true})
	require.NoError(t, err)
	require.Len(t, rec.samples, 1, "one query records exactly one telemetry sample")

	s := rec.samples[0]
	require.NotEmpty(t, s.QueryHash)
	require.NotContains(t, s.QueryHash, "restart", "query text is hashed, never stored raw")
	require.ElementsMatch(t, []string{"command", "component", "record"}, s.RoutedTypes)
	// cli-health(1) + ui-health(1) + swarm-manager(1) = 3 hits across the fan-out.
	require.Equal(t, 3, s.ResultCount)
	require.Equal(t, 1, s.ProviderResults["cli-health.commands"].HitCount)
	require.GreaterOrEqual(t, s.ProviderResults["cli-health.commands"].LatencyMs, int64(0))
	require.Equal(t, resp.GetLatencyMs(), s.LatencyMs)
	require.False(t, s.Reranked)
}

func TestQueryRecordsZeroResultAndDegraded(t *testing.T) {
	rec := &recordingRecorder{}
	// One provider, resolvable but the HTTP call errors ⇒ degraded, zero hits.
	r := routing.NewRouter(routing.Deps{
		Lister:   &fakeLister{providers: []*registryv1.ProviderDescriptor{cliHealthCommands()}},
		Resolver: threeProviderResolver(),
		Doer:     routeDoer{byURL: map[string]cannedResponse{}}, // no canned response ⇒ transport error
		Recorder: rec,
	})

	_, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "x", All: true})
	require.NoError(t, err)
	require.Len(t, rec.samples, 1)
	require.Equal(t, 0, rec.samples[0].ResultCount)
	require.True(t, rec.samples[0].Degraded)
	provider := rec.samples[0].ProviderResults["cli-health.commands"]
	require.True(t, provider.Degraded)
	require.Equal(t, "other", provider.DegradeReason)
}

func TestQueryNoRecorderIsNoop(t *testing.T) {
	// No Recorder wired ⇒ Query still succeeds (telemetry is optional).
	r := routing.NewRouter(routing.Deps{
		Lister:   threeProviderLister(),
		Resolver: threeProviderResolver(),
		Doer:     threeProviderDoer(),
	})
	_, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "x", All: true})
	require.NoError(t, err)
}

func TestStatusReportsProviderHealthAndModels(t *testing.T) {
	// swarm-manager is unreachable; cli-health + ui-health resolve.
	resolver := staticResolver{
		urls: map[string]string{"cli-health": "http://cli-health.test", "ui-health": "http://ui-health.test"},
		errs: map[string]error{"swarm-manager": errors.New("not running")},
	}
	clf := &fakeClassifier{result: routing.ClassifyResult{}}     // Available ⇒ true (err nil)
	rrk := &fakeReranker{err: errors.New("reranker model down")} // Available ⇒ false
	r := routing.NewRouter(routing.Deps{
		Lister:     threeProviderLister(),
		Resolver:   resolver,
		Doer:       threeProviderDoer(),
		Classifier: clf,
		Reranker:   rrk,
	})

	st, err := r.Status(context.Background())
	require.NoError(t, err)
	require.Len(t, st.GetProviders(), 3)

	byID := map[string]*routingv1.ProviderHealth{}
	for _, h := range st.GetProviders() {
		byID[h.GetProviderId()] = h
	}
	require.True(t, byID["cli-health.commands"].GetReachable())
	require.False(t, byID["cli-health.commands"].GetDegraded())
	require.False(t, byID["swarm-manager.records"].GetReachable())
	require.True(t, byID["swarm-manager.records"].GetDegraded())
	require.Contains(t, byID["swarm-manager.records"].GetFreshness(), "unreachable")

	require.True(t, st.GetClassifierAvailable())
	require.False(t, st.GetRerankerAvailable())
}

func TestStatusNoModelsWiredReportsUnavailable(t *testing.T) {
	r := routing.NewRouter(routing.Deps{
		Lister:   &fakeLister{providers: []*registryv1.ProviderDescriptor{cliHealthCommands()}},
		Resolver: threeProviderResolver(),
		Doer:     threeProviderDoer(),
	})
	st, err := r.Status(context.Background())
	require.NoError(t, err)
	require.False(t, st.GetClassifierAvailable(), "no classifier wired ⇒ unavailable")
	require.False(t, st.GetRerankerAvailable(), "no reranker wired ⇒ unavailable")
	require.Len(t, st.GetProviders(), 1)
	require.True(t, st.GetProviders()[0].GetReachable())
}

func TestStatusListerErrorPropagates(t *testing.T) {
	r := routing.NewRouter(routing.Deps{
		Lister:   &fakeLister{err: errors.New("db down")},
		Resolver: threeProviderResolver(),
		Doer:     threeProviderDoer(),
	})
	_, err := r.Status(context.Background())
	require.Error(t, err, "a registry read failure is a real error (unlike per-provider failures)")
}
