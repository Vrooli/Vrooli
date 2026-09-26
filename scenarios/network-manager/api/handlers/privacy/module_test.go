package privacy

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
	if m.Name != "privacy" {
		t.Fatalf("module name = %q, want privacy", m.Name)
	}
	if len(m.Endpoints) == 0 {
		t.Fatal("expected privacy endpoints")
	}
}
