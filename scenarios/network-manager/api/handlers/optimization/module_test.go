package optimization

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	domainadapters "network-manager/internal/adapters"
	domainoptimization "network-manager/internal/optimization"
	domainsnapshot "network-manager/internal/snapshot"
	"network-manager/internal/testutil/db"
)

func TestModuleExposesEndpoints(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(domainadapters.Schema),
		apidb.SchemaProviderFunc(domainoptimization.Schema),
		apidb.SchemaProviderFunc(domainsnapshot.Schema),
	))
	m := Module(d)
	if m.Name != "optimization" {
		t.Fatalf("module name = %q, want optimization", m.Name)
	}
	if len(m.Endpoints) == 0 {
		t.Fatal("expected optimization endpoints")
	}
}
