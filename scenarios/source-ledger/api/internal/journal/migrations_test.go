package journal

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestEnsureMigrationsAddsImportProvenanceToExistingEntries(t *testing.T) {
	db, err := sql.Open("sqlite", "file:journal-migration?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	_, err = db.Exec(`CREATE TABLE entries (id TEXT PRIMARY KEY, body TEXT NOT NULL)`)
	require.NoError(t, err)
	require.NoError(t, EnsureMigrations(context.Background(), db))
	require.NoError(t, EnsureMigrations(context.Background(), db), "migration must be replay-safe")
	rows, err := db.Query(`PRAGMA table_info(entries)`)
	require.NoError(t, err)
	defer rows.Close()
	columns := map[string]bool{}
	for rows.Next() {
		var cid, notNull, pk int
		var name, typ string
		var defaultValue any
		require.NoError(t, rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk))
		columns[name] = true
	}
	require.True(t, columns["source_harness"])
	require.True(t, columns["source_path"])
	require.True(t, columns["imported_at"])
}

func TestEnsureMigrationsIsNoopBeforeEntriesExists(t *testing.T) {
	db, err := sql.Open("sqlite", "file:journal-migration-empty?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, EnsureMigrations(context.Background(), db))
}

func TestEnsureMigrationsGuardsJournalUpdatesAndDeletes(t *testing.T) { // [REQ:SL-P0-001]
	db, err := sql.Open("sqlite", "file:journal-append-only?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	_, err = db.Exec(Schema())
	require.NoError(t, err)
	require.NoError(t, EnsureMigrations(context.Background(), db))
	_, err = db.Exec(`INSERT INTO entries(id,body,facet_id,kind,created_at) VALUES('immutable','body','unclassified','test','2026-08-06T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE entries SET body='changed' WHERE id='immutable'`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "append-only")
	_, err = db.Exec(`DELETE FROM entries WHERE id='immutable'`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "append-only")
}

func TestEnsureMigrationsRejectsLostJournalRows(t *testing.T) {
	db, err := sql.Open("sqlite", "file:journal-high-water?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	_, err = db.Exec(`CREATE TABLE entries (id TEXT PRIMARY KEY, body TEXT NOT NULL)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO entries(id,body) VALUES('remaining','body')`)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE journal_high_water_mark (id INTEGER PRIMARY KEY CHECK (id=1), max_rowid INTEGER NOT NULL, recorded_at TEXT NOT NULL)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO journal_high_water_mark(id,max_rowid,recorded_at) VALUES(1,2,'2026-08-06T00:00:00Z')`)
	require.NoError(t, err)
	err = EnsureMigrations(context.Background(), db)
	require.Error(t, err)
	require.Contains(t, err.Error(), "high-water mark")
}

func TestSQLiteAppendAdvancesJournalHighWaterMark(t *testing.T) {
	db, err := sql.Open("sqlite", "file:journal-hwm-append?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	_, err = db.Exec(Schema())
	require.NoError(t, err)
	repo := NewSQLiteRepository(db)
	created, err := repo.Append(context.Background(), Entry{Body: "durable", FacetID: UnclassifiedFacet, Kind: "test"}, nil)
	require.NoError(t, err)
	var rowID, marked int64
	require.NoError(t, db.QueryRow(`SELECT rowid FROM entries WHERE id=?`, created.ID).Scan(&rowID))
	require.NoError(t, db.QueryRow(`SELECT max_rowid FROM journal_high_water_mark WHERE id=1`).Scan(&marked))
	require.Equal(t, rowID, marked)
}
