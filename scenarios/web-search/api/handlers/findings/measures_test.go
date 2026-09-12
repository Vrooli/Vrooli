package findings

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/scheduletest"
	measures "github.com/vrooli/measures-go"
	internallivesearch "web-search/internal/livesearch"
	internalresearch "web-search/internal/research"
)

func TestLiveAndResearchMeasuresUseObservedFixedWindowDenominators(t *testing.T) {
	clk := scheduletest.New(time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC))
	live := internallivesearch.NewMetrics(clk.Now)
	eventAt := clk.Now().Add(-time.Minute)
	live.Record(internallivesearch.MetricCacheHit, eventAt)
	live.Record(internallivesearch.MetricCacheMiss, eventAt)
	live.Record(internallivesearch.MetricGovernorRefusal, eventAt)
	outcomes := internalresearch.NewOutcomeMetrics()
	outcomes.Record(internalresearch.OutcomeMetric{At: eventAt, SupportedClaims: 1, AssessedClaims: 2, SupportedQuestions: 1, RequiredQuestions: 2, FetchSuccesses: 1, FetchAttempts: 2, Calls: 3, Duration: 40 * time.Millisecond})

	reg := measures.NewRegistry(measures.WithClock(clk.Now))
	require.NoError(t, registerLiveMeasures(reg, live, clk))
	require.NoError(t, registerOutcomeMeasures(reg, outcomes, clk))

	result, err := reg.Execute(context.Background(), measures.MeasureRequest{Measure: MeasureCacheHitRate, Params: map[string]string{"window": string(measures.TokenThisWeek)}})
	require.NoError(t, err)
	require.Equal(t, "0.5000", result.Value)
	result, err = reg.Execute(context.Background(), measures.MeasureRequest{Measure: MeasureGovernorRefusals, Params: map[string]string{"window": string(measures.TokenThisWeek)}})
	require.NoError(t, err)
	require.Equal(t, "1", result.Value)
	result, err = reg.Execute(context.Background(), measures.MeasureRequest{Measure: MeasureSupportedClaimRate, Params: map[string]string{"window": string(measures.TokenThisWeek)}})
	require.NoError(t, err)
	require.Equal(t, "0.5000", result.Value)
	result, err = reg.Execute(context.Background(), measures.MeasureRequest{Measure: MeasureFetchSuccessRate, Params: map[string]string{"window": string(measures.TokenThisWeek)}})
	require.NoError(t, err)
	require.Equal(t, "0.5000", result.Value)
}
