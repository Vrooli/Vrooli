package snapshot

import (
	"context"
	"testing"

	domainsnapshot "network-manager/internal/snapshot"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
)

func TestModuleExposesEndpoints(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(domainsnapshot.Schema)))
	m := Module(d)
	if m.Name != "snapshot" {
		t.Fatalf("module name = %q, want snapshot", m.Name)
	}
	if len(m.Endpoints) == 0 {
		t.Fatal("expected snapshot endpoints")
	}
}
