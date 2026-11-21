package templates

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	db "github.com/vrooli/api-core/databasetest"
)

// studio_templates is a new table, so the migration story is simply "apply the
// declarative schema on every boot". This test proves the second boot is a
// no-op that neither fails nor duplicates the table or its index.
func TestMigrateIsIdempotent(t *testing.T) {
	sqlDB := db.NewSQLite(t)
	ctx := context.Background()

	require.NoError(t, Migrate(ctx, sqlDB))

	var tables int
	require.NoError(t, sqlDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'studio_templates'`).Scan(&tables))
	require.Equal(t, 1, tables, "migration must create studio_templates")

	require.NoError(t, Migrate(ctx, sqlDB))
	require.NoError(t, Migrate(ctx, sqlDB))

	require.NoError(t, sqlDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'studio_templates'`).Scan(&tables))
	require.Equal(t, 1, tables, "re-running Migrate must not duplicate the table")

	var indexes int
	require.NoError(t, sqlDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'studio_templates_updated_at'`).Scan(&indexes))
	require.Equal(t, 1, indexes, "re-running Migrate must not duplicate the index")

	// The table the schema declares is usable immediately.
	_, err := sqlDB.ExecContext(ctx,
		`INSERT INTO studio_templates (id, name, created_at, updated_at)
		 VALUES ('migrated', 'Migrated', '2026-11-19T10:00:00.000Z', '2026-11-19T10:00:00.000Z')`)
	require.NoError(t, err)
}
