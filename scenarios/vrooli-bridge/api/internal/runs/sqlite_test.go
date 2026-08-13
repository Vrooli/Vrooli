package runs_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"vrooli-bridge/internal/runs"
	"vrooli-bridge/internal/testutil/db"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "vrooli-bridge/internal/database"
)

func newSchemaDB(t *testing.T) (*sql.DB, *scheduletest.FakeClock) {
	t.Helper()
	clk := scheduletest.New(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC))
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(runs.Schema),
	))
	return d, clk
}

// [REQ:BRG-P0-005] A created run round-trips its durable identity and the
// repository assigns an id + created_at.
func TestSQLiteRepository_CreateGetRoundTrip(t *testing.T) {
	d, clk := newSchemaDB(t)
	repo := runs.NewSQLiteRepository(d, clk)
	ctx := context.Background()

	created, err := repo.Create(ctx, runs.Run{
		NodeID:         "n1",
		Scenario:       "web-search",
		Verb:           "scenario test",
		Args:           []string{"web-search", "--json"},
		TimeoutSeconds: 600,
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)
	require.Equal(t, runs.StatusQueued, created.Status)

	got, err := repo.Get(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, "web-search", got.Scenario)
	require.Equal(t, []string{"web-search", "--json"}, got.Args)
	require.Equal(t, int64(600), got.TimeoutSeconds)
}

// [REQ:BRG-P0-005] An unknown id is a typed not-found.
func TestSQLiteRepository_GetNotFound(t *testing.T) {
	d, clk := newSchemaDB(t)
	repo := runs.NewSQLiteRepository(d, clk)
	_, err := repo.Get(context.Background(), "ghost")
	require.ErrorAs(t, err, &runs.ErrRunNotFound{})
}

// [REQ:BRG-P0-005] Lifecycle update persists status/exit/started/finished and
// list filters by node, newest-first.
func TestSQLiteRepository_UpdateAndListFilter(t *testing.T) {
	d, clk := newSchemaDB(t)
	repo := runs.NewSQLiteRepository(d, clk)
	ctx := context.Background()

	a, _ := repo.Create(ctx, runs.Run{NodeID: "n1", Verb: "scenario test"})
	clk.Advance(time.Second)
	_, _ = repo.Create(ctx, runs.Run{NodeID: "n2", Verb: "scenario test"})

	a.Status = runs.StatusPassed
	a.ExitCode = 0
	a.StartedAt = clk.Now()
	a.FinishedAt = clk.Now()
	updated, err := repo.Update(ctx, a)
	require.NoError(t, err)
	require.Equal(t, runs.StatusPassed, updated.Status)
	require.False(t, updated.FinishedAt.IsZero())

	n1Runs, err := repo.List(ctx, runs.ListFilter{NodeID: "n1"})
	require.NoError(t, err)
	require.Len(t, n1Runs, 1)
	require.Equal(t, "n1", n1Runs[0].NodeID)

	all, err := repo.List(ctx, runs.ListFilter{})
	require.NoError(t, err)
	require.Len(t, all, 2)
}

// [REQ:BRG-P0-005] The event history is append-only and de-duplicates a
// re-sent (run_id, sequence) — at-least-once delivery is safe.
func TestSQLiteRepository_AppendEventsDedup(t *testing.T) {
	d, clk := newSchemaDB(t)
	repo := runs.NewSQLiteRepository(d, clk)
	ctx := context.Background()

	run, _ := repo.Create(ctx, runs.Run{NodeID: "n1", Verb: "scenario test"})
	ev := runs.RunEvent{RunID: run.ID, Kind: runs.EventLog, Sequence: 1, LogChunk: "hello\n", EmittedAt: clk.Now()}
	require.NoError(t, repo.AppendEvent(ctx, ev))
	require.NoError(t, repo.AppendEvent(ctx, ev)) // re-send same sequence
	require.NoError(t, repo.AppendEvent(ctx, runs.RunEvent{RunID: run.ID, Kind: runs.EventExit, Sequence: 2, ExitCode: 0, EmittedAt: clk.Now()}))

	events, err := repo.ListEvents(ctx, run.ID)
	require.NoError(t, err)
	require.Len(t, events, 2, "the duplicate sequence is ignored, not double-stored")
	require.Equal(t, uint64(1), events[0].Sequence)
	require.Equal(t, runs.EventExit, events[1].Kind)
}

func TestSQLiteRepository_DeliveryAckIsDurableAndIdempotent(t *testing.T) {
	d, clk := newSchemaDB(t)
	repo := runs.NewSQLiteRepository(d, clk)
	ctx := context.Background()
	ack := runs.DeliveryAck{NodeID: "swarminator", FrameID: "frame-1", RunID: "run-1", ReceivedAt: clk.Now()}

	require.NoError(t, repo.RecordDeliveryAck(ctx, ack))
	require.NoError(t, repo.RecordDeliveryAck(ctx, ack))
	var count int
	require.NoError(t, d.QueryRowContext(ctx, "SELECT COUNT(*) FROM delivery_acks WHERE frame_id = ?", ack.FrameID).Scan(&count))
	require.Equal(t, 1, count)
}

func TestMigrate_AddsDeliveryColumnsWithoutReplacingRuns(t *testing.T) {
	d := db.NewSQLite(t)
	_, err := d.ExecContext(context.Background(), `CREATE TABLE runs (
		id TEXT PRIMARY KEY, node_id TEXT NOT NULL, scenario TEXT NOT NULL DEFAULT '',
		verb TEXT NOT NULL DEFAULT '', args TEXT NOT NULL DEFAULT '[]', status INTEGER NOT NULL DEFAULT 1,
		exit_code INTEGER NOT NULL DEFAULT 0, timeout_seconds INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL, started_at TEXT NOT NULL DEFAULT '', finished_at TEXT NOT NULL DEFAULT '',
		artifact_refs TEXT NOT NULL DEFAULT '[]')`)
	require.NoError(t, err)
	_, err = d.ExecContext(context.Background(), `INSERT INTO runs (id, node_id, created_at) VALUES ('legacy', 'n1', '2026-06-18T12:00:00Z')`)
	require.NoError(t, err)
	require.NoError(t, runs.Migrate(context.Background(), d))
	require.NoError(t, runs.Migrate(context.Background(), d), "migration is idempotent")
	var node string
	require.NoError(t, d.QueryRowContext(context.Background(), "SELECT node_id FROM runs WHERE id = 'legacy'").Scan(&node))
	require.Equal(t, "n1", node)
	var found int
	require.NoError(t, d.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM pragma_table_info('runs') WHERE name = 'delivery_lease_expires_at'").Scan(&found))
	require.Equal(t, 1, found)
}
