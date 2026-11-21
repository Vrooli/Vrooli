package control

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/databasetest"
)

func TestEnsureIdentitySourceUpgradesLegacyControllerTable(t *testing.T) {
	db := databasetest.NewSQLite(t)
	_, err := db.ExecContext(context.Background(), `CREATE TABLE lighting_controllers (
		controller_id TEXT PRIMARY KEY,
		backend_id TEXT NOT NULL,
		name TEXT NOT NULL DEFAULT '',
		vendor TEXT NOT NULL DEFAULT '',
		description TEXT NOT NULL DEFAULT '',
		enrolled INTEGER NOT NULL DEFAULT 0,
		last_seen_at TEXT NOT NULL
	)`)
	require.NoError(t, err)
	require.NoError(t, EnsureIdentitySource(context.Background(), db))
	require.NoError(t, EnsureIdentitySource(context.Background(), db), "migration is safe to repeat")

	rows, err := db.QueryContext(context.Background(), "PRAGMA table_info(lighting_controllers)")
	require.NoError(t, err)
	defer rows.Close()
	var found bool
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, primaryKey int
		var defaultValue any
		require.NoError(t, rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primaryKey))
		found = found || name == "identity_source"
	}
	require.NoError(t, rows.Err())
	require.True(t, found)
}
