package resolver

import (
	"context"
	"testing"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
)

func TestSQLiteRepositoryStoresBackendAndUpstreams(t *testing.T) {
	// [REQ:NM-P0-002] Resolver config and upstreams are persisted in domain-owned tables.
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(d)

	_, err := repo.SaveBackend(context.Background(), BackendConfig{
		Backend:       AdGuardHomeBackend,
		BaseURL:       "http://adguard.local",
		Username:      "admin",
		CredentialRef: "vrooli/adguard-home",
	})
	require.NoError(t, err)
	require.NoError(t, repo.UpdateUpstreams(context.Background(), AdGuardHomeBackend, []string{"1.1.1.1", "9.9.9.9"}))

	cfg, err := repo.GetBackend(context.Background(), AdGuardHomeBackend)
	require.NoError(t, err)
	require.Equal(t, "vrooli/adguard-home", cfg.CredentialRef)
	require.NotZero(t, cfg.CreatedAt)
	require.NotZero(t, cfg.UpdatedAt)

	upstreams, err := repo.GetUpstreams(context.Background(), AdGuardHomeBackend)
	require.NoError(t, err)
	require.Equal(t, []string{"1.1.1.1", "9.9.9.9"}, upstreams)
}

func TestMigrateRenamesLegacyTokenReference(t *testing.T) {
	d := db.NewSQLite(t)
	ctx := context.Background()
	_, err := d.ExecContext(ctx, `
CREATE TABLE resolver_backends (
    backend TEXT PRIMARY KEY,
    base_url TEXT NOT NULL,
    username TEXT NOT NULL,
    token_ref TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
)`)
	require.NoError(t, err)
	_, err = d.ExecContext(ctx, `
INSERT INTO resolver_backends (backend, base_url, username, token_ref, created_at, updated_at)
VALUES ('adguard-home', 'http://adguard.local', 'admin', 'vrooli/adguard-home', 'now', 'now')`)
	require.NoError(t, err)

	require.NoError(t, Migrate(ctx, d))
	require.NoError(t, apidb.EnsureSchemas(ctx, d, apidb.SchemaProviderFunc(Schema)))

	var credentialRef string
	require.NoError(t, d.QueryRowContext(ctx, `SELECT credential_ref FROM resolver_backends WHERE backend = 'adguard-home'`).Scan(&credentialRef))
	require.Equal(t, "vrooli/adguard-home", credentialRef)
}
