package harness

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	harnessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/harness"
)

func TestImportProgressEstimateUsesPersistedRunTiming(t *testing.T) {
	now := time.Date(2026, 7, 29, 20, 0, 0, 0, time.UTC)
	rate, eta, ok := importProgressEstimate(&harnessv1.ImportRun{
		Status:           "running",
		ProcessedSources: 30,
		TotalSources:     90,
		StartedAt:        now.Add(-time.Minute).Format(time.RFC3339Nano),
	}, now)
	require.True(t, ok)
	require.InDelta(t, 30, rate, 0.01)
	require.Equal(t, 2*time.Minute, eta)
}

func TestImportProgressEstimateSuppressesTerminalAndInvalidRuns(t *testing.T) {
	now := time.Now()
	_, _, ok := importProgressEstimate(&harnessv1.ImportRun{Status: "completed", ProcessedSources: 1, TotalSources: 2, StartedAt: now.Add(-time.Minute).Format(time.RFC3339Nano)}, now)
	require.False(t, ok)
	_, _, ok = importProgressEstimate(&harnessv1.ImportRun{Status: "running", ProcessedSources: 1, TotalSources: 2, StartedAt: "not-a-time"}, now)
	require.False(t, ok)
}
