package inventory

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	testdb "github.com/vrooli/api-core/databasetest"
	localdb "network-manager/internal/database"
)

func newInventoryDB(t *testing.T) *sqliteRepository {
	t.Helper()
	db := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(Schema),
	))
	return NewSQLiteRepository(db).(*sqliteRepository)
}

func TestSQLiteRepositorySaveListGetAndGroup(t *testing.T) {
	repo := newInventoryDB(t)
	now := time.Date(2026, 6, 23, 19, 0, 0, 0, time.UTC)

	saved, err := repo.SaveDevice(context.Background(), Device{
		ID:                 "dev-sql",
		Hostname:           "nas",
		IPAddress:          "192.168.1.50",
		MACAddress:         "00:11:22:33:44:66",
		Group:              "storage",
		IdentityConfidence: "high",
		Notes:              []string{"matched existing device"},
		LastSeen:           now,
		CreatedAt:          now,
		UpdatedAt:          now,
	})
	require.NoError(t, err)
	require.Equal(t, "dev-sql", saved.ID)

	got, err := repo.GetDevice(context.Background(), "dev-sql")
	require.NoError(t, err)
	require.Equal(t, "nas", got.Hostname)
	require.Equal(t, []string{"matched existing device"}, got.Notes)

	list, err := repo.ListDevices(context.Background(), "storage")
	require.NoError(t, err)
	require.Len(t, list, 1)

	updated, err := repo.UpdateGroup(context.Background(), "dev-sql", "trusted")
	require.NoError(t, err)
	require.Equal(t, "trusted", updated.Group)

	empty, err := repo.ListDevices(context.Background(), "storage")
	require.NoError(t, err)
	require.Empty(t, empty)
}

func TestSQLiteRepositorySchemaIsIdempotent(t *testing.T) {
	db := testdb.NewSQLite(t)
	for range 2 {
		require.NoError(t, apidb.EnsureSchemas(context.Background(), db,
			apidb.SchemaProviderFunc(localdb.SystemSchema),
			apidb.SchemaProviderFunc(Schema),
		))
	}
}
