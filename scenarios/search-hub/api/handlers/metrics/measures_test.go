package metrics_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	gomeasures "github.com/vrooli/measures-go"

	handler "search-hub/handlers/metrics"
	internalmetrics "search-hub/internal/metrics"
)

type fakeRangeInsights struct {
	out  *internalmetrics.Insights
	from time.Time
	to   time.Time
}

func (f *fakeRangeInsights) InsightsRange(_ context.Context, from, to time.Time) (*internalmetrics.Insights, error) {
	f.from, f.to = from, to
	return f.out, nil
}

func TestMeasureRegistryDeclarations(t *testing.T) {
	reg, err := handler.NewMeasureRegistry(&fakeRangeInsights{}, fixedMeasureNow)
	require.NoError(t, err)

	names := []string{}
	for _, decl := range reg.Declarations() {
		names = append(names, decl.Name)
	}
	require.ElementsMatch(t, []string{
		handler.MeasureFederatedLatency,
		handler.MeasureDegradedQueryRate,
		handler.MeasureProviderDegradationRate,
	}, names)
}

func TestMeasureRegistryExecutesFederatedLatency(t *testing.T) {
	reader := &fakeRangeInsights{out: &internalmetrics.Insights{
		LatencyP50Ms: 120,
		LatencyP95Ms: 450,
	}}
	reg, err := handler.NewMeasureRegistry(reader, fixedMeasureNow)
	require.NoError(t, err)

	got, err := reg.Execute(context.Background(), gomeasures.MeasureRequest{
		Measure: handler.MeasureFederatedLatency,
		Params:  map[string]string{"window": string(gomeasures.TokenLast7d)},
	})
	require.NoError(t, err)
	require.Equal(t, "450", got.Value)
	require.Equal(t, "120", got.Fields[0]["p50_ms"])
	require.Contains(t, got.Provenance.ExecutedQuery, "query_telemetry latency percentiles")
	require.False(t, got.Provenance.ComputedAt.IsZero())
	require.Equal(t, fixedMeasureNow().AddDate(0, 0, -7), reader.from)
	require.Equal(t, fixedMeasureNow(), reader.to)
}

func TestMeasureRegistryScopesProviderDegradationRate(t *testing.T) {
	reader := &fakeRangeInsights{out: &internalmetrics.Insights{
		ProviderUsage: []internalmetrics.ProviderUsage{
			{ProviderID: "slow.provider", TimesRouted: 4, DegradedCount: 3},
			{ProviderID: "healthy.provider", TimesRouted: 10, DegradedCount: 0},
		},
	}}
	reg, err := handler.NewMeasureRegistry(reader, fixedMeasureNow)
	require.NoError(t, err)

	got, err := reg.Execute(context.Background(), gomeasures.MeasureRequest{
		Measure: handler.MeasureProviderDegradationRate,
		Params: map[string]string{
			"window":      string(gomeasures.TokenThisWeek),
			"provider_id": "slow.provider",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "0.7500", got.Value)
	require.Contains(t, got.Provenance.ExecutedQuery, `provider_id="slow.provider"`)
}

func fixedMeasureNow() time.Time {
	return time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
}
