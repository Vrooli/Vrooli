package journal_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"

	"token-economy/internal/journal"
	"token-economy/internal/mints"
)

type journalStore interface {
	journal.Repository
	journal.Projector
	journal.HolderEventReader
}

func newJournalRepository(t *testing.T) (journalStore, *sql.DB) {
	t.Helper()
	db := dbtest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), db,
		database.SchemaProviderFunc(mints.Schema),
		database.SchemaProviderFunc(journal.Schema),
	))
	return journal.NewSQLiteRepository(db), db
}

func seedTokenType(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	repo := mints.NewSQLiteRepository(db)
	_, err := repo.Create(context.Background(), mints.TokenType{
		ID: id, Name: id, Symbol: "T", Color: "#2563eb",
		SupplyPolicy: mints.SupplyPolicyUnbounded,
		Authority:    mints.MinterAuthority{TokenTypeID: id, Subject: "parent:alex"},
		CreatedAt:    time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
}

func event(id, tokenTypeID, holderID string, amount int64, kind journal.EventKind, cause string, at time.Time) journal.Event {
	return journal.Event{
		ID: id, TokenTypeID: tokenTypeID, HolderID: holderID, Amount: amount,
		Kind: kind, CauseReference: cause, Reason: cause,
		ActorIdentity: "operator", ActorKind: journal.ActorKindOperator,
		ActorVerificationStatus: journal.VerificationAbsent, CreatedAt: at,
	}
}

// [REQ:TKE-P0-010] Appending preserves immutable event order and the complete cause record.
func TestRepositoryAppendAndRead(t *testing.T) {
	repo, db := newJournalRepository(t)
	seedTokenType(t, db, "chores")
	ctx := context.Background()
	createdAt := time.Date(2026, 8, 19, 13, 0, 0, 0, time.UTC)
	want := event("earned-1", "chores", "child:sam", 7, journal.EventKindCredit, "grant:weekly", createdAt)

	got, err := repo.Append(ctx, want)
	require.NoError(t, err)
	require.Equal(t, want, got)
	events, err := repo.Read(ctx, "child:sam", "chores")
	require.NoError(t, err)
	require.Equal(t, []journal.Event{want}, events)
}

func TestRepositoryRefusesCrossHolderReversal(t *testing.T) {
	repo, db := newJournalRepository(t)
	seedTokenType(t, db, "chores")
	ctx := context.Background()
	base := time.Date(2026, 8, 19, 13, 0, 0, 0, time.UTC)
	_, err := repo.Append(ctx, event("earned-1", "chores", "child:sam", 7, journal.EventKindCredit, "grant:weekly", base))
	require.NoError(t, err)

	_, err = repo.Append(ctx, event("reverse-1", "chores", "child:lee", 7, journal.EventKindReversal, "earned-1", base.Add(time.Second)))
	require.ErrorIs(t, err, journal.ErrInvalidJournalEvent)
	leeEvents, readErr := repo.Read(ctx, "child:lee", "chores")
	require.NoError(t, readErr)
	require.Empty(t, leeEvents)
}

func TestEnsureSchemaMigratesExistingJournalAdditively(t *testing.T) {
	db := dbtest.NewSQLite(t)
	_, err := db.ExecContext(context.Background(), `CREATE TABLE journal_events (
		id TEXT PRIMARY KEY,
		token_type_id TEXT NOT NULL,
		holder_id TEXT NOT NULL,
		amount INTEGER NOT NULL,
		kind TEXT NOT NULL,
		cause_reference TEXT NOT NULL,
		actor_identity TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL
	)`)
	require.NoError(t, err)
	require.NoError(t, journal.EnsureSchema(context.Background(), db))
	require.NoError(t, journal.EnsureSchema(context.Background(), db), "migration must be idempotent")

	rows, err := db.QueryContext(context.Background(), `PRAGMA table_info(journal_events)`)
	require.NoError(t, err)
	defer rows.Close()
	columns := map[string]bool{}
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue any
		require.NoError(t, rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey))
		columns[name] = true
	}
	require.NoError(t, rows.Err())
	for _, name := range []string{"reason", "actor_kind", "actor_verification_status", "actor_run_id"} {
		require.True(t, columns[name], "missing migrated journal_events.%s", name)
	}
}
