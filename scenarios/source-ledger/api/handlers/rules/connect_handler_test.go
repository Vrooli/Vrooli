package rules

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apiDatabase "github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"

	internaldatabase "source-ledger/internal/database"
	internalfacets "source-ledger/internal/facets"
	"source-ledger/internal/policy"

	rulesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/rules"
)

func TestListRulesRejectsUnregisteredScope(t *testing.T) {
	db, err := apiDatabase.Open(context.Background(), apiDatabase.Config{Driver: apiDatabase.DriverSQLite, DSN: "file:rules-handler?mode=memory&cache=shared"})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, apiDatabase.EnsureSchemas(context.Background(), db.Primary(), apiDatabase.SchemaProviderFunc(internaldatabase.SystemSchema), apiDatabase.SchemaProviderFunc(internalfacets.Schema)))

	registry := policy.NewRegistry(db.Primary())
	require.NoError(t, registry.Ensure(context.Background(), policy.BuiltInDefaults()))
	h := NewConnectHandler(internalfacets.NewService(internalfacets.NewSQLiteRepository(db.Primary())), nil, registry)

	_, err = h.ListRules(context.Background(), connect.NewRequest(&rulesv1.ListRulesRequest{Scope: "no-such-scope-xyz"}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}
