package journal

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"source-ledger/internal/policy"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/retention/sqlitemaintenance"
	"github.com/vrooli/api-core/storage"
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
	var migrated bool
	require.NoError(t, db.QueryRow(`SELECT EXISTS (SELECT 1 FROM sqlite_schema WHERE name='journal_entry_positions')`).Scan(&migrated))
	require.False(t, migrated, "migration must roll back the partial position table")
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
	require.NoError(t, db.QueryRow(`SELECT position FROM journal_entry_positions WHERE entry_id=?`, created.ID).Scan(&rowID))
	require.NoError(t, db.QueryRow(`SELECT max_rowid FROM journal_high_water_mark WHERE id=1`).Scan(&marked))
	require.Equal(t, rowID, marked)
}

func TestJournalPositionsSurviveMigrationAndVacuum(t *testing.T) {
	ctx := t.Context()
	dsn, err := storage.SQLiteDSNAt(filepath.Join(t.TempDir(), "legacy-journal.db"), storage.SQLiteTuning{})
	require.NoError(t, err)
	pool, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: dsn, MaxOpenConns: 1})
	require.NoError(t, err)
	defer pool.Close()
	db := pool.Primary()
	_, err = db.ExecContext(ctx, schema)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
INSERT INTO entries(rowid,id,scope,body,facet_id,kind,import_key,source_path,created_at)
VALUES(2,'z-first','agent-memory','first body','test','note','first-key','a.md','2026-11-26T00:00:00Z'),
      (43,'a-latest','agent-memory','second body','test','note','second-key','b.md','2026-11-26T00:00:00Z'),
      (90,'other','team:other','third body','test','note',NULL,'','2026-11-26T00:00:00Z');
CREATE TABLE journal_high_water_mark (id INTEGER PRIMARY KEY CHECK(id=1), max_rowid INTEGER NOT NULL, recorded_at TEXT NOT NULL);
INSERT INTO journal_high_water_mark VALUES (1,90,'original mark');
INSERT INTO facet_texts(id,entry_id,kind,text) VALUES('view','a-latest','topic','second entry view');
INSERT INTO embeddings(id,facet_text_id,vector_json,vector_blob,created_at)
VALUES('vector','view','',X'0000803f','2026-11-26T00:00:00Z');
INSERT INTO journal_retry_queue(id,entry_id,reason,created_at)
VALUES('retry','a-latest','embed','2026-11-26T00:00:00Z');
CREATE TRIGGER entries_append_only_update BEFORE UPDATE ON entries BEGIN
  SELECT RAISE(ABORT, 'journal entries are append-only');
END;
CREATE TRIGGER entries_append_only_delete BEFORE DELETE ON entries BEGIN
  SELECT RAISE(ABORT, 'journal entries are append-only');
END;
`)
	require.NoError(t, err)
	require.NoError(t, ValidateHighWaterMark(ctx, db))
	require.NoError(t, EnsureMigrations(ctx, db))
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(Schema)))
	require.NoError(t, EnsureMigrations(ctx, db))

	// SQLite can preserve hidden rowids for indexed tables. Reassign them in
	// the fixture so any remaining dependence on them fails on every build.
	_, err = db.ExecContext(ctx, `DROP TRIGGER entries_append_only_update;
UPDATE entries SET rowid=CASE id WHEN 'z-first' THEN 900 WHEN 'a-latest' THEN 14 ELSE 300 END`)
	require.NoError(t, err)
	require.NoError(t, EnsureMigrations(ctx, db))

	result, err := sqlitemaintenance.Maintain(ctx, pool, sqlitemaintenance.Options{AllowConversion: true, ForceConversion: true})
	require.NoError(t, err)
	require.True(t, result.Converted)
	require.Empty(t, result.Deferred)
	require.Equal(t, "incremental", result.After.AutoVacuum)
	require.NoError(t, EnsureMigrations(ctx, db), "boot after VACUUM")

	for id, want := range map[string]int{"z-first": 2, "a-latest": 43, "other": 90} {
		var position int
		require.NoError(t, db.QueryRowContext(ctx, `SELECT position FROM journal_entry_positions WHERE entry_id=?`, id).Scan(&position))
		require.Equal(t, want, position)
	}
	var mark int
	var recordedAt string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT max_rowid,recorded_at FROM journal_high_water_mark`).Scan(&mark, &recordedAt))
	require.Equal(t, 90, mark)
	require.Equal(t, "original mark", recordedAt)

	repository := NewSQLiteRepository(db)
	after, err := repository.ListAfter(ctx, "z-first", 10)
	require.NoError(t, err)
	require.Len(t, after, 1)
	require.Equal(t, "a-latest", after[0].ID)
	require.Equal(t, "second body", after[0].Body)
	require.Equal(t, "b.md", after[0].Import.Path)
	require.Len(t, after[0].FacetTexts, 1)
	require.Equal(t, "view", after[0].FacetTexts[0].ID)
	afterCrossScope, err := repository.ListAfter(ctx, "other", 10)
	require.NoError(t, err)
	require.Empty(t, afterCrossScope)
	recent, err := repository.ListRecent(ctx, "note", 10)
	require.NoError(t, err)
	require.Len(t, recent, 2)
	require.Equal(t, "a-latest", recent[0].ID)

	var vector, retry string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT hex(em.vector_blob), q.id
FROM entries e JOIN facet_texts f ON f.entry_id=e.id JOIN embeddings em ON em.facet_text_id=f.id
JOIN journal_retry_queue q ON q.entry_id=e.id WHERE e.id='a-latest'`).Scan(&vector, &retry))
	require.Equal(t, "0000803F", vector)
	require.Equal(t, "retry", retry)
	var violations int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_foreign_key_check`).Scan(&violations))
	require.Zero(t, violations)
	var integrity string
	require.NoError(t, db.QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&integrity))
	require.Equal(t, "ok", integrity)

	for _, mutation := range []string{
		`UPDATE entries SET body='changed' WHERE id='a-latest'`,
		`DELETE FROM entries WHERE id='a-latest'`,
		`UPDATE journal_entry_positions SET position=100 WHERE entry_id='a-latest'`,
		`DELETE FROM journal_entry_positions WHERE entry_id='a-latest'`,
		`INSERT OR REPLACE INTO journal_entry_positions(position,entry_id) VALUES(43,'forged')`,
		`INSERT OR REPLACE INTO entries(id,body,facet_id,kind,created_at)
VALUES ('a-latest','replaced','test','note','2026-11-26T00:00:00Z')`,
	} {
		_, err := db.ExecContext(ctx, mutation)
		require.ErrorContains(t, err, "append-only")
	}
	latest := "a-latest"
	appendInput := Entry{ID: "new", Body: "new body", FacetID: "test", Kind: "note", CreatedAt: time.Date(2026, 11, 26, 0, 0, 0, 0, time.UTC), ExpectedLatest: &latest}
	invalidAppend := appendInput
	invalidAppend.FacetTexts = []FacetText{{ID: "view", Text: "duplicate dependent record"}}
	_, err = repository.Append(ctx, invalidAppend, nil)
	require.Error(t, err)
	require.NoError(t, db.QueryRowContext(ctx, `SELECT max_rowid FROM journal_high_water_mark`).Scan(&mark))
	require.Equal(t, 90, mark, "a failed append must roll its position and watermark back")

	_, err = repository.Append(ctx, appendInput, nil)
	require.NoError(t, err)
	require.NoError(t, db.QueryRowContext(ctx, `SELECT position FROM journal_entry_positions WHERE entry_id='new'`).Scan(&mark))
	require.Equal(t, 91, mark)
	require.NoError(t, db.QueryRowContext(ctx, `SELECT max_rowid FROM journal_high_water_mark`).Scan(&mark))
	require.Equal(t, 91, mark)
	require.NoError(t, ValidateHighWaterMark(ctx, db))
	after, err = repository.ListAfter(ctx, "a-latest", 10)
	require.NoError(t, err)
	require.Len(t, after, 1)
	require.Equal(t, "new", after[0].ID)

	_, err = db.ExecContext(ctx, `DROP TRIGGER journal_entry_positions_append_only_delete;
DELETE FROM journal_entry_positions WHERE entry_id='z-first'`)
	require.NoError(t, err)
	require.ErrorContains(t, EnsureMigrations(ctx, db), "inconsistent")
}

// Latest checkpoint reads must filter before limiting and isolate team scopes.
func TestListRecentFiltersKindScopeAndUsesAppendOrderForTies(t *testing.T) { // [REQ:SL-P0-004]
	db, err := sql.Open("sqlite", "file:journal-latest?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(Schema())
	require.NoError(t, err)
	for _, row := range []struct{ id, scope, kind, body string }{
		{"z-active", "team:director-swarm", "walk-checkpoint", "active"},
		{"a-complete", "team:director-swarm", "walk-checkpoint", "completed"},
		{"noise", "team:director-swarm", "observation", "unrelated"},
		{"other", "team:other", "walk-checkpoint", "wrong scope"},
	} {
		_, err = db.Exec(`INSERT INTO entries(id,scope,body,facet_id,kind,created_at) VALUES(?,?,?,'unclassified',?,'2026-09-04T00:00:00Z')`, row.id, row.scope, row.body, row.kind)
		require.NoError(t, err)
	}
	repo := NewSQLiteRepository(db)
	rows, err := repo.ListRecent(policy.WithScope(context.Background(), "team:director-swarm"), "walk-checkpoint", 1)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "a-complete", rows[0].ID)
}
