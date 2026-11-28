package optimization

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestSQLStoreReopensAndRejectsStaleRevision(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "optimize.db")
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	store, err := NewSQLStore(ctx, db)
	require.NoError(t, err)
	req := sampleRequest()
	reqCopy, err := cloneRequest(req)
	require.NoError(t, err)
	initial := Record{Request: reqCopy, Stage: StageAcquire, Evidence: []Evidence{}}
	_, err = store.Create(ctx, initial)
	require.NoError(t, err)
	first, err := store.Load(ctx, req.RunID)
	require.NoError(t, err)
	first.Stage = StageEvaluate
	first.Revision = 1
	_, err = store.CompareAndSwap(ctx, 0, first)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	db, err = sql.Open("sqlite", path)
	require.NoError(t, err)
	defer db.Close()
	reopened, err := NewSQLStore(ctx, db)
	require.NoError(t, err)
	persisted, err := reopened.Load(ctx, req.RunID)
	require.NoError(t, err)
	require.Equal(t, StageEvaluate, persisted.Stage)
	require.Equal(t, uint64(1), persisted.Revision)
	_, err = reopened.CompareAndSwap(ctx, 0, persisted)
	require.ErrorIs(t, err, ErrRevision)
}

func TestSQLStoreListsActiveRunsByChronologicalUpdateTime(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "optimize.db"))
	require.NoError(t, err)
	defer db.Close()
	store, err := NewSQLStore(ctx, db)
	require.NoError(t, err)

	base := time.Date(2026, time.November, 14, 21, 0, 0, 0, time.UTC)
	older := sampleRequest()
	older.RunID = "older-whole-second"
	newer := sampleRequest()
	newer.RunID = "newer-fractional-second"
	for _, record := range []Record{
		{Request: older, Stage: StageAcquire, UpdatedAt: base},
		{Request: newer, Stage: StageAcquire, UpdatedAt: base.Add(500 * time.Millisecond)},
	} {
		_, err := store.Create(ctx, record)
		require.NoError(t, err)
	}

	active, err := store.ListActive(ctx, 10)
	require.NoError(t, err)
	require.Len(t, active, 2)
	require.Equal(t, older.RunID, active[0].Request.RunID,
		"recovery should resume the oldest durable run first")
	require.Equal(t, newer.RunID, active[1].Request.RunID)
}
