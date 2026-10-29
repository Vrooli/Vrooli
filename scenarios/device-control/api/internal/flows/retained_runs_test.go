package flows

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"device-control/internal/execution"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestRetainedRunExpiryBoundaryRestartAndLegacy(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "runs.sqlite"))
	require.NoError(t, err)
	defer db.Close()
	now := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	repo, err := NewSQLiteRetainedRuns(ctx, db)
	require.NoError(t, err)
	repo.now = func() time.Time { return now }
	expiry := now.Add(time.Hour)
	run := RetainedRun{DeviceID: "tv", Result: execution.RunResult{RunID: "current"}, ExpiresAt: &expiry}
	require.NoError(t, repo.Put(ctx, run))
	require.NoError(t, repo.Put(ctx, run))
	second := RetainedRun{DeviceID: "tv", Result: execution.RunResult{RunID: "second"}, ExpiresAt: &expiry}
	require.NoError(t, repo.Put(ctx, second))
	extended := expiry.Add(time.Hour)
	run.ExpiresAt = &extended
	require.ErrorContains(t, repo.Put(ctx, run), "conflict")
	legacy := RetainedRun{DeviceID: "tv", Result: execution.RunResult{RunID: "legacy"}}
	payload, err := json.Marshal(legacy)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO device_control_retained_runs(run_id,device_id,payload) VALUES(?,?,?)`, "legacy", "tv", string(payload))
	require.NoError(t, err)
	restarted, err := NewSQLiteRetainedRuns(ctx, db)
	require.NoError(t, err)
	restarted.now = func() time.Time { return now }
	listed, err := restarted.List(ctx)
	require.NoError(t, err)
	require.Len(t, listed, 3)
	require.Equal(t, expiry, *listed[0].ExpiresAt)
	require.Nil(t, listed[1].ExpiresAt, "legacy record must not acquire a silent deadline")
	got, err := restarted.Get(ctx, "current")
	require.NoError(t, err)
	require.Equal(t, expiry, *got.ExpiresAt)
	now = expiry
	_, err = restarted.Get(ctx, "current")
	require.ErrorIs(t, err, ErrInvalidRequest)
	var n int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM device_control_retained_runs WHERE run_id='current'`).Scan(&n))
	require.Zero(t, n)
	listed, err = restarted.List(ctx)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, "legacy", listed[0].RunID)
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM device_control_retained_runs WHERE run_id='second'`).Scan(&n))
	require.Zero(t, n, "listing removes expired rows at the deadline")
	_, err = restarted.Get(ctx, "legacy")
	require.NoError(t, err)
}

func TestRetentionRejectsExpiredOrMissingDeadline(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "runs.sqlite"))
	require.NoError(t, err)
	defer db.Close()
	repo, err := NewSQLiteRetainedRuns(context.Background(), db)
	require.NoError(t, err)
	now := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	repo.now = func() time.Time { return now }
	run := RetainedRun{DeviceID: "tv", Result: execution.RunResult{RunID: "run"}}
	require.ErrorIs(t, repo.Put(context.Background(), run), ErrInvalidRequest)
	run.ExpiresAt = &now
	require.ErrorIs(t, repo.Put(context.Background(), run), ErrInvalidRequest)
}
