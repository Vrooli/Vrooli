package recovery_test

import (
	"context"
	"testing"
	"time"

	"tunnel-manager/internal/recovery"
	"tunnel-manager/internal/testutil/db"
	"tunnel-manager/internal/testutil/mocks"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "tunnel-manager/internal/database"
)

func newRepo(t *testing.T) (recovery.Repository, *mocks.FakeClock) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(recovery.Schema),
	))
	clk := mocks.NewFakeClock(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	return recovery.NewSQLiteRepository(d, clk), clk
}

func TestSQLite_PersistAssignsIDAndTimestamp(t *testing.T) {
	repo, clk := newRepo(t)
	got, err := repo.PersistEvent(context.Background(), recovery.RecoveryEvent{
		Trigger: recovery.TriggerManual, Outcome: recovery.OutcomeSuccess, Details: "ok", Attempt: 1,
	})
	require.NoError(t, err)
	require.NotEmpty(t, got.ID)
	require.Equal(t, recovery.ActionRestart, got.Action, "default action applied")
	require.Equal(t, clk.Now().UTC(), got.CreatedAt)
}

func TestSQLite_ListEventsNewestFirstAndLimit(t *testing.T) {
	repo, clk := newRepo(t)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		_, err := repo.PersistEvent(ctx, recovery.RecoveryEvent{
			Trigger: recovery.TriggerManual, Outcome: recovery.OutcomeSuccess, Attempt: i,
		})
		require.NoError(t, err)
		clk.Advance(time.Second)
	}
	all, err := repo.ListEvents(ctx, 0)
	require.NoError(t, err)
	require.Len(t, all, 3)
	require.Equal(t, 2, all[0].Attempt, "newest first")

	limited, err := repo.ListEvents(ctx, 1)
	require.NoError(t, err)
	require.Len(t, limited, 1)
}

func TestSQLite_PersistPrunesExpiredEvents(t *testing.T) {
	repo, clk := newRepo(t)
	ctx := context.Background()

	_, err := repo.PersistEvent(ctx, recovery.RecoveryEvent{
		Trigger: recovery.TriggerManual, Outcome: recovery.OutcomeFailure, Attempt: 1,
	})
	require.NoError(t, err)

	clk.Advance(recovery.EventRetentionWindow - time.Second)
	_, err = repo.PersistEvent(ctx, recovery.RecoveryEvent{
		Trigger: recovery.TriggerManual, Outcome: recovery.OutcomeSkipped, Attempt: 2,
	})
	require.NoError(t, err)
	all, err := repo.ListEvents(ctx, 0)
	require.NoError(t, err)
	require.Len(t, all, 2)

	clk.Advance(2 * time.Second)
	_, err = repo.PersistEvent(ctx, recovery.RecoveryEvent{
		Trigger: recovery.TriggerManual, Outcome: recovery.OutcomeSuccess, Attempt: 3,
	})
	require.NoError(t, err)
	all, err = repo.ListEvents(ctx, 0)
	require.NoError(t, err)
	require.Len(t, all, 2, "oldest recovery event was outside retention and pruned")
	require.Equal(t, 3, all[0].Attempt)
	require.Equal(t, 2, all[1].Attempt)
}
