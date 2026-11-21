package harness

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	localdb "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/scheduletest"
)

func TestRunStoreUsesInjectedClockForRunAndCursorTimestamps(t *testing.T) {
	db := localdb.NewSQLite(t)
	_, err := db.Exec(Schema())
	require.NoError(t, err)

	instant := time.Date(2035, 2, 3, 4, 5, 6, 7, time.UTC)
	store := newRunStore(db, scheduletest.New(instant))
	ctx := context.Background()
	run, err := store.create(ctx, "swarm-manager-records", "/records", 4)
	require.NoError(t, err)
	require.Equal(t, instant, run.StartedAt)
	require.Equal(t, instant, run.UpdatedAt)

	require.NoError(t, store.running(ctx, run.ID))
	require.NoError(t, store.progress(ctx, run.ID, 2, 2, 0, 0, "record-2.json"))
	require.NoError(t, store.finish(ctx, run.ID, ImportRunCompleted, 4, 4, 0, 0, ""))
	got, err := store.get(ctx, run.ID)
	require.NoError(t, err)
	require.Equal(t, ImportRunCompleted, got.Status)
	require.Equal(t, instant, got.StartedAt)
	require.Equal(t, instant, got.CompletedAt)
	require.Equal(t, instant, got.UpdatedAt)

	require.NoError(t, store.setCursor(ctx, "swarm-manager-records", "/records", "record-4.json"))
	var cursorUpdatedAt string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT updated_at FROM harness_import_cursors WHERE runtime=?`, "swarm-manager-records").Scan(&cursorUpdatedAt))
	require.Equal(t, instant.Format(time.RFC3339Nano), cursorUpdatedAt)
}

func TestImportRunTransitionGraphRejectsUnknownAndTerminalEdges(t *testing.T) {
	for _, edge := range [][2]ImportRunStatus{
		{ImportRunQueued, ImportRunRunning},
		{ImportRunQueued, ImportRunFailed},
		{ImportRunRunning, ImportRunCompleted},
		{ImportRunRunning, ImportRunCompletedWithErrors},
		{ImportRunRunning, ImportRunFailed},
	} {
		require.NoError(t, requireRunTransition(edge[0], edge[1]))
	}
	for _, edge := range [][2]ImportRunStatus{
		{ImportRunQueued, ImportRunCompleted},
		{ImportRunRunning, ImportRunQueued},
		{ImportRunCompleted, ImportRunRunning},
		{ImportRunCompletedWithErrors, ImportRunFailed},
		{ImportRunFailed, ImportRunCompleted},
		{"unknown", ImportRunRunning},
		{ImportRunRunning, "unknown"},
	} {
		require.Error(t, requireRunTransition(edge[0], edge[1]), "%q -> %q", edge[0], edge[1])
	}
}

func TestRunStoreGuardsExpectedPriorStateAndTerminalWrites(t *testing.T) {
	db := localdb.NewSQLite(t)
	_, err := db.Exec(Schema())
	require.NoError(t, err)
	store := newRunStore(db, nil)
	ctx := context.Background()
	run, err := store.create(ctx, "swarm-manager-records", "/records", 1)
	require.NoError(t, err)
	require.Error(t, store.progress(ctx, run.ID, 1, 1, 0, 0, "record.json"), "queued runs cannot progress")
	require.NoError(t, store.running(ctx, run.ID))
	require.NoError(t, store.progress(ctx, run.ID, 1, 1, 0, 0, "record.json"))
	require.NoError(t, store.finish(ctx, run.ID, ImportRunCompleted, 1, 1, 0, 0, ""))
	require.Error(t, store.progress(ctx, run.ID, 2, 2, 0, 0, "late.json"), "terminal runs cannot progress")
	require.Error(t, store.finish(ctx, run.ID, ImportRunFailed, 1, 0, 0, 1, "late failure"), "terminal runs cannot finish again")
	require.Error(t, store.running(ctx, run.ID), "terminal runs cannot restart")
	got, err := store.get(ctx, run.ID)
	require.NoError(t, err)
	require.Equal(t, ImportRunCompleted, got.Status)
	require.Equal(t, 1, got.ProcessedSources)
	require.Equal(t, "record.json", got.CurrentPath)
}
