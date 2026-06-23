package inventory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	localdb "network-manager/internal/database"
	domaininventory "network-manager/internal/inventory"
	testdb "network-manager/internal/testutil/db"
)

func TestModuleExposesEndpoints(t *testing.T) {
	db := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(domaininventory.Schema),
	))
	m := Module(db)
	if m.Name != "inventory" {
		t.Fatalf("module name = %q, want inventory", m.Name)
	}
	if len(m.Endpoints) == 0 {
		t.Fatal("expected inventory endpoints")
	}
}

func TestSchemaIsDomainOwned(t *testing.T) {
	require.Contains(t, Schema(), "CREATE TABLE IF NOT EXISTS devices")
	require.Contains(t, Schema(), "CREATE TABLE IF NOT EXISTS device_groups")
}
