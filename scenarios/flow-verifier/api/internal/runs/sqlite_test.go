package runs_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"flow-verifier/internal/runs"
	"flow-verifier/internal/testutil/db"
	"flow-verifier/internal/testutil/mocks"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "flow-verifier/internal/database"
)

func newRepo(t *testing.T) (runs.Repository, *mocks.FakeClock) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(runs.Schema),
	))
	clk := mocks.NewFakeClock(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	return runs.NewSQLiteRepository(d, clk), clk
}

func TestSQLiteRepository_InsertAndGetRoundTrip(t *testing.T) {
	repo, clk := newRepo(t)
	ctx := context.Background()

	in := runs.Run{
		FlowID:     "notes.attachment-upload.api",
		FlowPath:   "api/internal/notes/flow/flow.json",
		Root:       "/tmp/example",
		Mode:       runs.ModeCheck,
		Status:     runs.StatusPassed,
		Output:     "fresh notes.attachment-upload.api\n",
		StartedAt:  clk.Now(),
		FinishedAt: clk.Now().Add(2 * time.Second),
	}
	inserted, err := repo.Insert(ctx, in)
	require.NoError(t, err)
	require.NotEmpty(t, inserted.ID, "Insert must populate ID")
	require.Equal(t, int64(2000), inserted.DurationMs, "DurationMs derived from timestamps")

	got, err := repo.Get(ctx, inserted.ID)
	require.NoError(t, err)
	require.Equal(t, inserted.FlowID, got.FlowID)
	require.Equal(t, runs.StatusPassed, got.Status)
	require.Equal(t, runs.ModeCheck, got.Mode)
	require.Equal(t, in.Output, got.Output)
	require.True(t, got.StartedAt.Equal(in.StartedAt))
}

func TestSQLiteRepository_GetMissingReturnsNotFound(t *testing.T) {
	repo, _ := newRepo(t)
	_, err := repo.Get(context.Background(), "does-not-exist")
	var nf runs.ErrNotFound
	require.True(t, errors.As(err, &nf), "expected ErrNotFound, got %T", err)
}

func TestSQLiteRepository_ListOrderingAndFilter(t *testing.T) {
	repo, clk := newRepo(t)
	ctx := context.Background()

	now := clk.Now()
	mustInsert := func(flow string, finished time.Time, status runs.Status) {
		_, err := repo.Insert(ctx, runs.Run{
			FlowID: flow, FlowPath: flow + ".json", Root: "/r", Mode: runs.ModeCheck,
			Status: status, StartedAt: finished.Add(-time.Second), FinishedAt: finished,
		})
		require.NoError(t, err)
	}
	mustInsert("a", now.Add(1*time.Minute), runs.StatusPassed)
	mustInsert("b", now.Add(2*time.Minute), runs.StatusFailed)
	mustInsert("a", now.Add(3*time.Minute), runs.StatusPassed)

	all, err := repo.List(ctx, runs.ListQuery{})
	require.NoError(t, err)
	require.Len(t, all, 3)
	require.Equal(t, "a", all[0].FlowID, "newest first")
	require.Equal(t, "b", all[1].FlowID)

	aOnly, err := repo.List(ctx, runs.ListQuery{FlowID: "a"})
	require.NoError(t, err)
	require.Len(t, aOnly, 2)
	for _, r := range aOnly {
		require.Equal(t, "a", r.FlowID)
	}

	limited, err := repo.List(ctx, runs.ListQuery{Limit: 1})
	require.NoError(t, err)
	require.Len(t, limited, 1)
}

func TestSQLiteRepository_CounterexampleRoundTrip(t *testing.T) {
	repo, clk := newRepo(t)
	ctx := context.Background()

	in := runs.Run{
		FlowID: "f", FlowPath: "p", Root: "/r",
		Mode: runs.ModeCheck, Status: runs.StatusFailed,
		Counterexample: `{"violated":"safety"}`,
		ErrorMessage:   "quint produced counterexample",
		StartedAt:      clk.Now(), FinishedAt: clk.Now(),
	}
	inserted, err := repo.Insert(ctx, in)
	require.NoError(t, err)
	got, err := repo.Get(ctx, inserted.ID)
	require.NoError(t, err)
	require.Equal(t, in.Counterexample, got.Counterexample)
	require.Equal(t, in.ErrorMessage, got.ErrorMessage)
}
