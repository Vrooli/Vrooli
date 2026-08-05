package forest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"
)

func TestSummaryAndEdgesCommitAtomicallyAndRebuild(t *testing.T) { // [REQ:VMEM-P0-007]
	db, err := apidb.Open(context.Background(), apidb.Config{Driver: apidb.DriverSQLite, DSN: "file:forest?mode=memory&cache=shared"})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db.Primary(), apidb.SchemaProviderFunc(Schema)))
	r := NewSQLiteRepository(db.Primary())
	_, err = r.CreateSummary(context.Background(), Summary{ID: "summary", Body: "summary", FacetID: "episode", Depth: 1}, []Edge{{ChildID: "entry", ChildKind: "invalid"}, {ChildID: "entry-2", ChildKind: "entry"}})
	require.Error(t, err)
	frontier, err := r.Frontier(context.Background())
	require.NoError(t, err)
	require.Empty(t, frontier)
	_, err = r.CreateSummary(context.Background(), Summary{ID: "summary", Body: "summary", FacetID: "episode", Depth: 1}, []Edge{{ChildID: "entry", ChildKind: "entry"}, {ChildID: "entry-2", ChildKind: "entry"}})
	require.NoError(t, err)
	frontier, err = r.Frontier(context.Background())
	require.NoError(t, err)
	require.Len(t, frontier, 1)
	require.NoError(t, r.Rebuild(context.Background()))
	frontier, err = r.Frontier(context.Background())
	require.NoError(t, err)
	require.Empty(t, frontier)
}
