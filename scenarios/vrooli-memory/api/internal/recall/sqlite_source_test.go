package recall

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"

	localdb "vrooli-memory/internal/database"
	"vrooli-memory/internal/facets"
	"vrooli-memory/internal/forest"
	"vrooli-memory/internal/journal"
)

func TestSQLiteSourceReturnsLeavesAndDerivedSummaries(t *testing.T) {
	ctx := context.Background()
	db, err := apidb.Open(ctx, apidb.Config{Driver: apidb.DriverSQLite, DSN: "file:recall-source?mode=memory&cache=shared"})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, apidb.EnsureSchemas(ctx, db.Primary(), apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(journal.Schema), apidb.SchemaProviderFunc(facets.Schema), apidb.SchemaProviderFunc(forest.Schema)))

	leaf, err := journal.NewSQLiteRepository(db.Primary()).Append(ctx, journal.Entry{Body: "precise leaf", FacetID: "episode", FacetTexts: []journal.FacetText{{Kind: "topic", Text: "precise leaf", Vector: []float64{1, 0}}}}, nil)
	require.NoError(t, err)
	_, err = forest.NewSQLiteRepository(db.Primary()).CreateSummary(ctx, forest.Summary{ID: "summary", Body: "broader summary", FacetID: "episode", Vector: []float64{0, 1}, Depth: 1}, []forest.Edge{{ParentID: "summary", ChildID: leaf.ID, ChildKind: "entry"}})
	require.NoError(t, err)

	nodes, err := NewSQLiteSource(db.Primary()).Nodes(ctx)
	require.NoError(t, err)
	require.Len(t, nodes, 2)
	byID := map[string]Node{}
	for _, n := range nodes {
		byID[n.ID] = n
	}
	require.Equal(t, "summary", byID[leaf.ID].ParentID)
	require.False(t, byID[leaf.ID].Frontier)
	require.True(t, byID["summary"].Frontier)
	require.Equal(t, []float64{0, 1}, byID["summary"].Vector)
	require.True(t, byID["summary"].Summary)
	require.Equal(t, 1, byID["summary"].Span)
	require.False(t, byID[leaf.ID].Summary)
}
