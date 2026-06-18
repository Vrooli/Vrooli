package provision_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"vrooli-bridge/internal/provision"
	"vrooli-bridge/internal/testutil/db"
	"vrooli-bridge/internal/testutil/mocks"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "vrooli-bridge/internal/database"
)

func newSchemaDB(t *testing.T) (*sql.DB, *mocks.FakeClock) {
	t.Helper()
	clk := mocks.NewFakeClock(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC))
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(provision.Schema),
	))
	return d, clk
}

// [REQ:BRG-P0-006] A created op round-trips its durable identity and the
// repository assigns an id + created_at + the default QUEUED status.
func TestSQLiteRepository_CreateGetRoundTrip(t *testing.T) {
	d, clk := newSchemaDB(t)
	repo := provision.NewSQLiteRepository(d, clk)
	ctx := context.Background()

	created, err := repo.Create(ctx, provision.ProvisioningOp{
		NodeID:           "n1",
		TargetRevision:   "rev-B",
		RollbackRevision: "rev-A",
		TimeoutSeconds:   1800,
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)
	require.Equal(t, provision.StatusQueued, created.Status)

	got, err := repo.Get(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, "rev-B", got.TargetRevision)
	require.Equal(t, "rev-A", got.RollbackRevision)
	require.Equal(t, int64(1800), got.TimeoutSeconds)
}

// [REQ:BRG-P0-006] An unknown id is a typed not-found.
func TestSQLiteRepository_GetNotFound(t *testing.T) {
	d, clk := newSchemaDB(t)
	repo := provision.NewSQLiteRepository(d, clk)
	_, err := repo.Get(context.Background(), "ghost")
	var notFound provision.ErrOpNotFound
	require.ErrorAs(t, err, &notFound)
}

// [REQ:BRG-P0-006] Update persists the mutable lifecycle columns; the durable
// identity is unchanged.
func TestSQLiteRepository_UpdateLifecycle(t *testing.T) {
	d, clk := newSchemaDB(t)
	repo := provision.NewSQLiteRepository(d, clk)
	ctx := context.Background()
	created, err := repo.Create(ctx, provision.ProvisioningOp{NodeID: "n1", TargetRevision: "rev-B"})
	require.NoError(t, err)

	created.Status = provision.StatusCompleted
	created.ResultingRevision = "rev-B"
	created.ExitCode = 0
	created.StartedAt = clk.Now()
	created.FinishedAt = clk.Now()
	updated, err := repo.Update(ctx, created)
	require.NoError(t, err)
	require.Equal(t, provision.StatusCompleted, updated.Status)
	require.Equal(t, "rev-B", updated.ResultingRevision)

	got, _ := repo.Get(ctx, created.ID)
	require.Equal(t, provision.StatusCompleted, got.Status)
	require.False(t, got.FinishedAt.IsZero())
}

// [REQ:BRG-P0-006] Events are append-only and de-duplicated on (op_id, sequence)
// — a re-sent event at the same sequence is ignored, not double-stored.
func TestSQLiteRepository_EventsAppendOnlyDeduped(t *testing.T) {
	d, clk := newSchemaDB(t)
	repo := provision.NewSQLiteRepository(d, clk)
	ctx := context.Background()
	created, _ := repo.Create(ctx, provision.ProvisioningOp{NodeID: "n1", TargetRevision: "rev-B"})

	require.NoError(t, repo.AppendEvent(ctx, provision.ProvisionEvent{OpID: created.ID, Kind: provision.EventLog, Sequence: 1, LogChunk: "a"}))
	require.NoError(t, repo.AppendEvent(ctx, provision.ProvisionEvent{OpID: created.ID, Kind: provision.EventLog, Sequence: 1, LogChunk: "dup"}))
	require.NoError(t, repo.AppendEvent(ctx, provision.ProvisionEvent{OpID: created.ID, Kind: provision.EventExit, Sequence: 2, ExitCode: 0}))

	evs, err := repo.ListEvents(ctx, created.ID)
	require.NoError(t, err)
	require.Len(t, evs, 2, "the duplicate (op_id, sequence) was ignored")
	require.Equal(t, "a", evs[0].LogChunk, "the first write wins; the dup is ignored")
}

// [REQ:BRG-P0-006] Node versions upsert: the latest VERSION replaces the prior
// row for the node.
func TestSQLiteRepository_NodeVersionUpsert(t *testing.T) {
	d, clk := newSchemaDB(t)
	repo := provision.NewSQLiteRepository(d, clk)
	ctx := context.Background()

	_, err := repo.GetNodeVersion(ctx, "n1")
	var none provision.ErrNoNodeVersion
	require.ErrorAs(t, err, &none)

	require.NoError(t, repo.UpsertNodeVersion(ctx, provision.NodeVersion{NodeID: "n1", Revision: "rev-A", OpID: "op-1"}))
	require.NoError(t, repo.UpsertNodeVersion(ctx, provision.NodeVersion{NodeID: "n1", Revision: "rev-B", OpID: "op-2"}))

	got, err := repo.GetNodeVersion(ctx, "n1")
	require.NoError(t, err)
	require.Equal(t, "rev-B", got.Revision)
	require.Equal(t, "op-2", got.OpID)
}

// [REQ:BRG-P0-006] List returns ops newest-first and honours the node filter.
func TestSQLiteRepository_ListNewestFirstFiltered(t *testing.T) {
	d, clk := newSchemaDB(t)
	repo := provision.NewSQLiteRepository(d, clk)
	ctx := context.Background()

	for i, n := range []string{"n1", "n2", "n1"} {
		clk.Advance(time.Duration(i+1) * time.Minute)
		_, err := repo.Create(ctx, provision.ProvisioningOp{NodeID: n, TargetRevision: "rev"})
		require.NoError(t, err)
	}

	all, err := repo.List(ctx, provision.ListFilter{})
	require.NoError(t, err)
	require.Len(t, all, 3)

	n1, err := repo.List(ctx, provision.ListFilter{NodeID: "n1"})
	require.NoError(t, err)
	require.Len(t, n1, 2)
	require.True(t, n1[0].CreatedAt.After(n1[1].CreatedAt) || n1[0].CreatedAt.Equal(n1[1].CreatedAt))
}
