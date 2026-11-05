package capture_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	testdb "github.com/vrooli/api-core/databasetest"
	"web-search/internal/capture"
)

func newRepo(t *testing.T) (*capture.Repository, *sql.DB) {
	t.Helper()
	db := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(capture.Schema)))
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	return capture.NewRepository(db, func() time.Time { return now }), db
}

func TestEnqueueReplayAndPayloadConflict(t *testing.T) {
	r, _ := newRepo(t)
	first, existing, err := r.Enqueue(context.Background(), "attempt-1", "task-1", "delivery-1", []byte(`{"outcome":"failed"}`))
	require.NoError(t, err)
	require.False(t, existing)
	require.Equal(t, capture.StatePending, first.State)
	replay, existing, err := r.Enqueue(context.Background(), "attempt-1", "task-1", "delivery-1", []byte(`{"outcome":"failed"}`))
	require.NoError(t, err)
	require.True(t, existing)
	require.Equal(t, first.PayloadSHA256, replay.PayloadSHA256)
	_, _, err = r.Enqueue(context.Background(), "attempt-1", "task-1", "delivery-1", []byte(`{"outcome":"success"}`))
	require.Error(t, err)
}

func TestClaimLeaseAndDeliveryCounts(t *testing.T) {
	r, _ := newRepo(t)
	ctx := context.Background()
	_, _, err := r.Enqueue(ctx, "attempt-1", "task-1", "delivery-1", []byte("payload"))
	require.NoError(t, err)
	claimed, err := r.Claim(ctx, 10)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	require.Equal(t, 1, claimed[0].Attempts)
	require.NoError(t, r.MarkFailed(ctx, "attempt-1", "memory unavailable", time.Date(2026, 9, 5, 12, 1, 0, 0, time.UTC)))
	require.NoError(t, r.MarkDelivered(ctx, "attempt-1"))
	counts, err := r.Counts(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, counts.Delivered)
	require.Equal(t, 0, counts.Pending)
}

func TestDrainKeepsFailedDeliveryRetryable(t *testing.T) {
	r, _ := newRepo(t)
	ctx := context.Background()
	_, _, err := r.Enqueue(ctx, "attempt-1", "task-1", "delivery-1", []byte("payload"))
	require.NoError(t, err)
	delivered, err := r.Drain(ctx, 10, func(context.Context, capture.Attempt) error { return context.DeadlineExceeded })
	require.NoError(t, err)
	require.Zero(t, delivered)
	counts, err := r.Counts(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, counts.Failed)
	a, err := r.Get(ctx, "attempt-1")
	require.NoError(t, err)
	require.Equal(t, "context deadline exceeded", a.LastError, "delivery callback error should be retained")
}

func TestCountsInWindowExcludesAttemptsOutsideWindow(t *testing.T) {
	db := testdb.NewSQLite(t)
	clock := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(capture.Schema)))
	repo := capture.NewRepository(db, func() time.Time { return clock })
	_, _, err := repo.Enqueue(context.Background(), "attempt-1", "task-1", "delivery-1", []byte("one"))
	require.NoError(t, err)
	from := clock.Add(-time.Minute)
	to := clock.Add(time.Minute)
	counts, err := repo.CountsInWindow(context.Background(), from, to)
	require.NoError(t, err)
	require.Equal(t, 1, counts.Pending)

	counts, err = repo.CountsInWindow(context.Background(), clock.Add(time.Minute), clock.Add(2*time.Minute))
	require.NoError(t, err)
	require.Equal(t, 0, counts.Pending)
	require.Equal(t, 0, counts.Delivered)
}
