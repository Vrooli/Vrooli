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
