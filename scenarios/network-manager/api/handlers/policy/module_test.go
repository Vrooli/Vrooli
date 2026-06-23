package policy

import (
	"context"
	"testing"

	"network-manager/internal/testutil/db"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
)

func TestModuleExposesEndpoints(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema)))
	m := Module(d)
	if m.Name != "policy" {
		t.Fatalf("module name = %q, want policy", m.Name)
	}
	if len(m.Endpoints) == 0 {
		t.Fatal("expected policy endpoints")
	}
}
