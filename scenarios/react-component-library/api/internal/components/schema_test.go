package components_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	dbtest "github.com/vrooli/api-core/databasetest"

	"react-component-library/internal/components"
)

func TestEnsureMigrationsIsIdempotent(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	_, err := db.ExecContext(ctx, `CREATE TABLE component_versions (component_id TEXT NOT NULL)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `CREATE TABLE component_test_reports (id TEXT NOT NULL)`)
	require.NoError(t, err)

	require.NoError(t, components.EnsureMigrations(ctx, db))
	require.NoError(t, components.EnsureMigrations(ctx, db))

	var presenceDefault string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT dflt_value FROM pragma_table_info('component_versions') WHERE name='presence'`).Scan(&presenceDefault))
	require.Equal(t, "'materialized'", presenceDefault)

	var revisionDefault string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT dflt_value FROM pragma_table_info('component_test_reports') WHERE name='source_revision'`).Scan(&revisionDefault))
	require.Equal(t, "''", revisionDefault)
	var indexes int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_component_versions_component_presence'`).Scan(&indexes))
	require.Equal(t, 1, indexes)
}
