package resolver

import (
	"context"
	"testing"

	"network-manager/internal/testutil/db"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
)

func TestSQLiteRepositoryStoresBackendAndUpstreams(t *testing.T) {
	// [REQ:NM-P0-002] Resolver config and upstreams are persisted in domain-owned tables.
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(d)

	_, err := repo.SaveBackend(context.Background(), BackendConfig{
		Backend:  AdGuardHomeBackend,
		BaseURL:  "http://adguard.local",
		Username: "admin",
		TokenRef: "secret://adguard/token",
	})
	require.NoError(t, err)
	require.NoError(t, repo.UpdateUpstreams(context.Background(), AdGuardHomeBackend, []string{"1.1.1.1", "9.9.9.9"}))

	cfg, err := repo.GetBackend(context.Background(), AdGuardHomeBackend)
	require.NoError(t, err)
	require.Equal(t, "secret://adguard/token", cfg.TokenRef)
	require.NotZero(t, cfg.CreatedAt)
	require.NotZero(t, cfg.UpdatedAt)

	upstreams, err := repo.GetUpstreams(context.Background(), AdGuardHomeBackend)
	require.NoError(t, err)
	require.Equal(t, []string{"1.1.1.1", "9.9.9.9"}, upstreams)
}
