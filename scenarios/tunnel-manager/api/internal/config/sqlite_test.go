package config_test

import (
	"context"
	"testing"

	"tunnel-manager/internal/config"
	"tunnel-manager/internal/testutil/db"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "tunnel-manager/internal/database"
)

func newRepo(t *testing.T) config.ConfigRepository {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(config.Schema),
	))
	return config.NewSQLiteRepository(d)
}

func TestSQLite_GetUnsetReturnsDefaults(t *testing.T) {
	repo := newRepo(t)
	got, err := repo.Get(context.Background())
	require.NoError(t, err)
	require.Equal(t, config.DefaultMode, got.Mode)
	require.Equal(t, config.ModeLocal, got.Mode)
	require.Equal(t, config.DefaultPromEndpoint, got.PromEndpoint)
}

func TestSQLite_UpsertAndGetRoundTrip(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	want := config.TunnelConfig{
		Mode:         config.ModeLocal,
		TunnelID:     "tid-123",
		AccountID:    "acct-456",
		CredRef:      "vrooli/cloudflare:api-token",
		PromEndpoint: "127.0.0.1:9999",
	}
	stored, err := repo.Upsert(ctx, want)
	require.NoError(t, err)
	require.Equal(t, want, stored)

	got, err := repo.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestSQLite_UpsertIsSingletonOverwrite(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	_, err := repo.Upsert(ctx, config.TunnelConfig{Mode: config.ModeRemote, TunnelID: "first"})
	require.NoError(t, err)
	_, err = repo.Upsert(ctx, config.TunnelConfig{Mode: config.ModeLocal, TunnelID: "second"})
	require.NoError(t, err)

	got, err := repo.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, config.ModeLocal, got.Mode)
	require.Equal(t, "second", got.TunnelID, "one logical row, overwritten in place")
}

func TestSQLite_UpsertDefaultsMode(t *testing.T) {
	repo := newRepo(t)
	stored, err := repo.Upsert(context.Background(), config.TunnelConfig{TunnelID: "x"})
	require.NoError(t, err)
	require.Equal(t, config.DefaultMode, stored.Mode)
	require.Equal(t, config.DefaultPromEndpoint, stored.PromEndpoint)
}
