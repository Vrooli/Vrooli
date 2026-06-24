package monitoring

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	localdb "network-manager/internal/database"
	"network-manager/internal/testutil/db"
)

func TestSQLiteRepositoryPersistsMonitoringRecords(t *testing.T) {
	// [REQ:NM-P1-007] Monitoring schedules, runs, and alerts are durable
	// domain-owned records rather than in-memory UI state.
	ctx := context.Background()
	handle := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, handle,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(Schema),
		apidb.SchemaProviderFunc(Schema),
	))
	repo := NewSQLiteRepository(handle)
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)

	schedule, err := repo.UpsertSchedule(ctx, Schedule{
		ID:                   "schedule-1",
		Name:                 "Home baseline watch",
		Profile:              "home",
		BaselineSnapshotID:   "baseline-1",
		IntervalMinutes:      60,
		Enabled:              true,
		LatencyThresholdMS:   100,
		UnavailableThreshold: 1,
		Effects:              []string{"stored"},
		CreatedAt:            now,
		UpdatedAt:            now,
	})
	require.NoError(t, err)
	require.Equal(t, "schedule-1", schedule.ID)

	run, err := repo.SaveRun(ctx, Run{
		ID:                 "run-1",
		ScheduleID:         "schedule-1",
		SnapshotID:         "snapshot-2",
		Status:             "regression_detected",
		Summary:            "1 regression alert detected.",
		RegressionDetected: true,
		Effects:            []string{"compared"},
		CreatedAt:          now,
	})
	require.NoError(t, err)
	require.True(t, run.RegressionDetected)

	_, err = repo.SaveAlert(ctx, Alert{
		ID:         "alert-1",
		ScheduleID: "schedule-1",
		SnapshotID: "snapshot-2",
		Severity:   "warning",
		Status:     "open",
		Summary:    "DNS lookup latency regressed.",
		Evidence:   []string{"delta 160ms"},
		CreatedAt:  now,
	})
	require.NoError(t, err)

	schedules, err := repo.ListSchedules(ctx, false)
	require.NoError(t, err)
	require.Len(t, schedules, 1)
	require.Equal(t, []string{"stored"}, schedules[0].Effects)

	alerts, err := repo.ListAlerts(ctx, "schedule-1", true)
	require.NoError(t, err)
	require.Len(t, alerts, 1)
	require.Equal(t, []string{"delta 160ms"}, alerts[0].Evidence)
}
