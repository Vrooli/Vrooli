package audit_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	db "github.com/vrooli/api-core/databasetest"
	"vrooli-bridge/internal/audit"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "vrooli-bridge/internal/database"
)

func newSchemaStore(t *testing.T) (audit.Store, *scheduletest.FakeClock, *sql.DB) {
	t.Helper()
	clk := scheduletest.New(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC))
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(audit.Schema),
	))
	return audit.NewSQLiteStore(d, clk), clk, d
}

// [REQ:BRG-P0-008] Each dispatch writes an immutable audit record with actor,
// node, verb/args, and outcome; List returns them newest-first.
func TestAudit_AppendAndListNewestFirst(t *testing.T) {
	store, clk, _ := newSchemaStore(t)
	ctx := context.Background()

	first, err := store.Append(ctx, audit.Record{
		Action: audit.ActionDispatch, Actor: "owner-1", NodeID: "n1",
		Scenario: "web-search", Verb: "scenario test", Args: []string{"web-search"},
		Outcome: audit.OutcomeAccepted, RunID: "run-1",
	})
	require.NoError(t, err)
	require.NotEmpty(t, first.ID)
	require.False(t, first.RecordedAt.IsZero())

	clk.Advance(time.Second)
	_, err = store.Append(ctx, audit.Record{
		Action: audit.ActionDispatch, Actor: "owner-1", NodeID: "n1",
		Verb: "scenario deploy", Outcome: audit.OutcomeRejected, Detail: "verb not allowlisted",
	})
	require.NoError(t, err)

	records, err := store.List(ctx, audit.ListFilter{})
	require.NoError(t, err)
	require.Len(t, records, 2)
	require.Equal(t, audit.OutcomeRejected, records[0].Outcome, "newest first")
	require.Equal(t, "scenario test", records[1].Verb)
	require.Equal(t, []string{"web-search"}, records[1].Args)
}

// [REQ:BRG-P0-008] Records require an actor and node — a record cannot be
// written anonymously.
func TestAudit_RequiresActorAndNode(t *testing.T) {
	store, _, _ := newSchemaStore(t)
	ctx := context.Background()
	_, err := store.Append(ctx, audit.Record{NodeID: "n1", Verb: "x"})
	require.ErrorAs(t, err, &audit.ErrInvalidRecord{})
	_, err = store.Append(ctx, audit.Record{Actor: "o", Verb: "x"})
	require.ErrorAs(t, err, &audit.ErrInvalidRecord{})
}

// [REQ:BRG-P0-008] Filters narrow by node and by run.
func TestAudit_Filters(t *testing.T) {
	store, _, _ := newSchemaStore(t)
	ctx := context.Background()
	_, _ = store.Append(ctx, audit.Record{Actor: "o", NodeID: "n1", RunID: "run-1", Outcome: audit.OutcomeAccepted})
	_, _ = store.Append(ctx, audit.Record{Actor: "o", NodeID: "n2", RunID: "run-2", Outcome: audit.OutcomeAccepted})

	byNode, err := store.List(ctx, audit.ListFilter{NodeID: "n1"})
	require.NoError(t, err)
	require.Len(t, byNode, 1)
	require.Equal(t, "n1", byNode[0].NodeID)

	byRun, err := store.List(ctx, audit.ListFilter{RunID: "run-2"})
	require.NoError(t, err)
	require.Len(t, byRun, 1)
	require.Equal(t, "run-2", byRun[0].RunID)
}

// [REQ:BRG-P0-008] The trail is append-only: the store exposes only Append +
// List (no update/delete), so a written record is immutable. Two appends are
// two distinct rows — there is no overwrite path. (The compile-time guarantee
// that the Store interface carries no mutation verb is the structural proof;
// this exercises that re-appending never clobbers an existing record.)
func TestAudit_AppendOnlyImmutable(t *testing.T) {
	store, _, _ := newSchemaStore(t)
	ctx := context.Background()
	a, _ := store.Append(ctx, audit.Record{Actor: "o", NodeID: "n1", Verb: "scenario test", Outcome: audit.OutcomeAccepted})
	b, _ := store.Append(ctx, audit.Record{Actor: "o", NodeID: "n1", Verb: "scenario test", Outcome: audit.OutcomeAccepted})
	require.NotEqual(t, a.ID, b.ID, "each append is a new immutable record, never an overwrite")

	all, _ := store.List(ctx, audit.ListFilter{NodeID: "n1"})
	require.Len(t, all, 2)
}
