package homeintegration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	"network-manager/internal/testutil/db"
)

func TestModuleExposesEndpoints(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema)))
	m := Module(d)
	if m.Name != "home_integration" {
		t.Fatalf("module name = %q, want home_integration", m.Name)
	}
	if len(m.Endpoints) == 0 {
		t.Fatal("expected home integration endpoints")
	}
}
