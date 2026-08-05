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

func TestSQLiteSourceReturnsDepthTwoNodes(t *testing.T) { // [REQ:VMEM-P0-003]
	ctx := context.Background()
	db, err := apidb.Open(ctx, apidb.Config{Driver: apidb.DriverSQLite, DSN: "file:recall-source-depth-two?mode=memory&cache=shared"})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, apidb.EnsureSchemas(ctx, db.Primary(), apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(journal.Schema), apidb.SchemaProviderFunc(facets.Schema), apidb.SchemaProviderFunc(forest.Schema)))

	leaf, err := journal.NewSQLiteRepository(db.Primary()).Append(ctx, journal.Entry{Body: "depth two leaf", FacetID: "episode", FacetTexts: []journal.FacetText{{Kind: "topic", Text: "depth two leaf", Vector: []float64{1, 0}}}}, nil)
	require.NoError(t, err)
	forestRepo := forest.NewSQLiteRepository(db.Primary())
	_, err = forestRepo.CreateSummary(ctx, forest.Summary{ID: "summary-one", Body: "first summary", FacetID: "episode", Vector: []float64{1, 0}, Depth: 1}, []forest.Edge{{ChildID: leaf.ID, ChildKind: "entry"}})
	require.NoError(t, err)
	_, err = forestRepo.CreateSummary(ctx, forest.Summary{ID: "summary-two", Body: "second summary", FacetID: "episode", Vector: []float64{1, 0}, Depth: 2}, []forest.Edge{{ChildID: "summary-one", ChildKind: "summary"}})
	require.NoError(t, err)

	nodes, err := NewSQLiteSource(db.Primary()).Nodes(ctx)
	require.NoError(t, err)
	byID := map[string]Node{}
	for _, node := range nodes {
		byID[node.ID] = node
	}
	require.Equal(t, 2, byID["summary-two"].Depth)
	require.Equal(t, "summary-two", byID["summary-one"].ParentID)
	require.Equal(t, "summary-one", byID[leaf.ID].ParentID)
}

func TestResolvedThreadIsAbsentFromRecallSource(t *testing.T) { // [REQ:VMEM-P0-005]
	ctx := context.Background()
	db, err := apidb.Open(ctx, apidb.Config{Driver: apidb.DriverSQLite, DSN: "file:recall-source-resolved?mode=memory&cache=shared"})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, apidb.EnsureSchemas(ctx, db.Primary(), apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(journal.Schema), apidb.SchemaProviderFunc(facets.Schema), apidb.SchemaProviderFunc(forest.Schema)))
	facetRepo := facets.NewSQLiteRepository(db.Primary())
	require.NoError(t, facetRepo.Seed(ctx))
	entry, err := journal.NewSQLiteRepository(db.Primary()).Append(ctx, journal.Entry{Body: "resolved thread", FacetID: "thread", FacetTexts: []journal.FacetText{{Kind: "topic", Text: "resolved thread", Vector: []float64{1, 0}}}}, nil)
	require.NoError(t, err)
	_, err = facetRepo.Assign(ctx, facets.Assignment{EntryID: entry.ID, FacetID: "thread", ActorID: "test"})
	require.NoError(t, err)
	require.NoError(t, facetRepo.ResolveThread(ctx, entry.ID))

	nodes, err := NewSQLiteSource(db.Primary()).Nodes(ctx)
	require.NoError(t, err)
	for _, node := range nodes {
		require.NotEqual(t, entry.ID, node.ID)
	}
}
