package forest

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestEnsureMigrationsAddsSummaryVectorToExistingForest(t *testing.T) {
	db, err := sql.Open("sqlite", "file:forest-migration?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	_, err = db.Exec(`CREATE TABLE summaries (id TEXT PRIMARY KEY, body TEXT NOT NULL)`)
	require.NoError(t, err)

	require.NoError(t, EnsureMigrations(context.Background(), db))
	require.NoError(t, EnsureMigrations(context.Background(), db), "migration must be replay-safe")
	rows, err := db.Query(`PRAGMA table_info(summaries)`)
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
	require.True(t, columns["vector_json"])
}

func TestEnsureMigrationsIsNoopBeforeForestExists(t *testing.T) {
	db, err := sql.Open("sqlite", "file:forest-migration-empty?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, EnsureMigrations(context.Background(), db))
}
