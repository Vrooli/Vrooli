package adapters

import (
	"context"
	"testing"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
)

func TestModuleExposesEndpoints(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema)))
	m := Module(d)
	if m.Name != "adapters" {
		t.Fatalf("module name = %q, want adapters", m.Name)
	}
	if len(m.Endpoints) == 0 {
		t.Fatal("expected adapters endpoints")
	}
}
