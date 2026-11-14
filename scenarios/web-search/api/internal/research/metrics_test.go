package research_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"web-search/internal/research"
)

func TestOutcomeMetricsKeepFixedWindowAndExposeTruncation(t *testing.T) {
	metrics := research.NewOutcomeMetrics()
	now := time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)
	metrics.Record(research.OutcomeMetric{At: now, SupportedClaims: 1, AssessedClaims: 2, FetchSuccesses: 1, FetchAttempts: 2, Calls: 2, Duration: 10 * time.Millisecond})
	metrics.Record(research.OutcomeMetric{At: now.Add(time.Hour), SupportedClaims: 1, AssessedClaims: 1, FetchSuccesses: 1, FetchAttempts: 1, Calls: 1, Duration: 20 * time.Millisecond})
	totals, attempts, complete := metrics.Snapshot(now, now.Add(30*time.Minute))
	require.True(t, complete)
	require.Equal(t, 1, attempts)
	require.Equal(t, 1, totals.SupportedClaims)
	require.Equal(t, 2, totals.AssessedClaims)

	for i := 0; i < 4097; i++ {
		metrics.Record(research.OutcomeMetric{At: now})
	}
	_, _, complete = metrics.Snapshot(now, now.Add(time.Minute))
	require.False(t, complete, "discarded history must make window readings unavailable")
}
