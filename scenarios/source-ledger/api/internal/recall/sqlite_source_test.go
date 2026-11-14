package recall

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"

	localdb "source-ledger/internal/database"
	"source-ledger/internal/facets"
	"source-ledger/internal/forest"
	"source-ledger/internal/journal"
	"source-ledger/internal/policy"
)

func TestRecallSourceIsolatedByScopeAndDefaultsToAgentMemory(t *testing.T) { // [REQ:SL-P0-002]
	ctx := context.Background()
	db, err := apidb.Open(ctx, apidb.Config{Driver: apidb.DriverSQLite, DSN: "file:recall-scope?mode=memory&cache=shared"})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, apidb.EnsureSchemas(ctx, db.Primary(), apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(journal.Schema), apidb.SchemaProviderFunc(facets.Schema), apidb.SchemaProviderFunc(forest.Schema)))
	repo := journal.NewSQLiteRepository(db.Primary())
	agent, err := repo.Append(ctx, journal.Entry{ID: "agent-entry", Scope: "agent-memory", Body: "agent", FacetID: "episode"}, nil)
	require.NoError(t, err)
	team, err := repo.Append(policy.WithScope(ctx, "team:marketing-crew"), journal.Entry{ID: "team-entry", Scope: "team:marketing-crew", Body: "team", FacetID: "episode"}, nil)
	require.NoError(t, err)

	defaultNodes, err := NewSQLiteSource(db.Primary()).Nodes(ctx)
	require.NoError(t, err)
	require.Len(t, defaultNodes, 1)
	require.Equal(t, agent.ID, defaultNodes[0].ID)

	teamNodes, err := NewSQLiteSource(db.Primary()).Nodes(policy.WithScope(ctx, "team:marketing-crew"))
	require.NoError(t, err)
	require.Len(t, teamNodes, 1)
	require.Equal(t, team.ID, teamNodes[0].ID)
}

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
	require.Equal(t, []float64{0, 1}, byID["summary"].Vectors[0])
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

// entries.facet_id records the write-time guess and is never revised. The
// authority is the latest assignment, which is how compaction already reads it.
// Reading the stale column made corrected memories invisible to wake.
func TestNodesReadFacetFromLatestAssignmentNotTheStaleEntryColumn(t *testing.T) { // [REQ:VMEM-P0-005] [REQ:VMEM-P0-008]
	ctx := context.Background()
	db, err := apidb.Open(ctx, apidb.Config{Driver: apidb.DriverSQLite, DSN: "file:recall-latest-facet?mode=memory&cache=shared"})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, apidb.EnsureSchemas(ctx, db.Primary(), apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(journal.Schema), apidb.SchemaProviderFunc(facets.Schema), apidb.SchemaProviderFunc(forest.Schema)))

	// Written while inference was unavailable, then classified afterwards.
	leaf, err := journal.NewSQLiteRepository(db.Primary()).Append(ctx, journal.Entry{Body: "a trap worth remembering", FacetID: journal.UnclassifiedFacet, FacetTexts: []journal.FacetText{{Kind: "topic", Text: "a trap", Vector: []float64{1, 0}}}}, nil)
	require.NoError(t, err)
	_, err = db.Primary().ExecContext(ctx, `INSERT INTO facet_assignments(id,entry_id,facet_id,assigned_at,actor_id) VALUES('late','`+leaf.ID+`','gotcha','2099-01-01T00:00:00Z','classifier')`)
	require.NoError(t, err)

	nodes, err := NewSQLiteSource(db.Primary()).Nodes(ctx)
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	require.Equal(t, "gotcha", nodes[0].FacetID, "the corrected facet must reach recall, not the write-time guess")
}

// Wake scores no similarity, so it must not pay for the embedding corpus. This
// also pins the frontier contract: absorbed leaves are excluded unless pinned.
func TestAmbientNodesSkipsVectorsAndAbsorbedLeaves(t *testing.T) { // [REQ:VMEM-P0-008]
	ctx := context.Background()
	db, err := apidb.Open(ctx, apidb.Config{Driver: apidb.DriverSQLite, DSN: "file:recall-ambient?mode=memory&cache=shared"})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, apidb.EnsureSchemas(ctx, db.Primary(), apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(journal.Schema), apidb.SchemaProviderFunc(facets.Schema), apidb.SchemaProviderFunc(forest.Schema)))

	repo := journal.NewSQLiteRepository(db.Primary())
	absorbed, err := repo.Append(ctx, journal.Entry{Body: "absorbed leaf", FacetID: "episode", FacetTexts: []journal.FacetText{{Kind: "topic", Text: "absorbed", Vector: []float64{1, 0}}}}, nil)
	require.NoError(t, err)
	free, err := repo.Append(ctx, journal.Entry{Body: "free leaf", FacetID: "gotcha", FacetTexts: []journal.FacetText{{Kind: "topic", Text: "free", Vector: []float64{0, 1}}}}, nil)
	require.NoError(t, err)
	_, err = forest.NewSQLiteRepository(db.Primary()).CreateSummary(ctx, forest.Summary{ID: "summary", Body: "summary", FacetID: "episode", Vector: []float64{0, 1}, Depth: 1}, []forest.Edge{{ParentID: "summary", ChildID: absorbed.ID, ChildKind: "entry"}})
	require.NoError(t, err)

	nodes, err := NewSQLiteSource(db.Primary()).AmbientNodes(ctx, 8)
	require.NoError(t, err)

	ids := map[string]Node{}
	for _, n := range nodes {
		ids[n.ID] = n
		require.Empty(t, n.Vectors, "ambient nodes must carry no embeddings")
	}
	require.NotContains(t, ids, absorbed.ID, "an absorbed leaf is represented by its summary, not itself")
	require.Contains(t, ids, free.ID)
	require.Contains(t, ids, "summary")
	require.True(t, ids[free.ID].Frontier)
}

func TestSupersededSourcesAndCompactedAncestorsAreExcludedWithoutLosingCleanFrontier(t *testing.T) {
	ctx := context.Background()
	db, err := apidb.Open(ctx, apidb.Config{Driver: apidb.DriverSQLite, DSN: "file:recall-superseded-frontier?mode=memory&cache=shared"})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, apidb.EnsureSchemas(ctx, db.Primary(), apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(journal.Schema), apidb.SchemaProviderFunc(facets.Schema), apidb.SchemaProviderFunc(forest.Schema)))

	journalRepo := journal.NewSQLiteRepository(db.Primary())
	forestRepo := forest.NewSQLiteRepository(db.Primary())
	facetRepo := facets.NewSQLiteRepository(db.Primary())
	appendEntry := func(id, body string) journal.Entry {
		t.Helper()
		entry, appendErr := journalRepo.Append(ctx, journal.Entry{ID: id, Body: body, FacetID: "episode"}, nil)
		require.NoError(t, appendErr)
		return entry
	}
	oldFrontier := appendEntry("superseded-frontier", "old frontier")
	replacement := appendEntry("replacement-frontier", "replacement frontier")
	pinnedOld := appendEntry("superseded-pinned", "old pinned memory")
	oldNested := appendEntry("superseded-nested", "old nested memory")
	cleanNestedSibling := appendEntry("clean-nested-sibling", "clean nested sibling")
	cleanDirectChild := appendEntry("clean-direct-child", "clean direct child")
	cleanPinned := appendEntry("clean-pinned", "clean pinned memory")
	unrelatedLeaf := appendEntry("unrelated-leaf", "unrelated summary leaf")
	otherScope := policy.WithScope(ctx, "team:other")
	foreignMarked := appendEntryInScope(t, journalRepo, otherScope, "foreign-superseded", "foreign marked memory")
	foreignReplacement := appendEntryInScope(t, journalRepo, otherScope, "foreign-replacement", "foreign replacement")

	_, err = forestRepo.CreateSummary(ctx, forest.Summary{ID: "dirty-child", Body: "summary with superseded source", FacetID: "episode", Depth: 1}, []forest.Edge{
		{ChildID: oldNested.ID, ChildKind: "entry"},
		{ChildID: cleanNestedSibling.ID, ChildKind: "entry"},
	})
	require.NoError(t, err)
	_, err = forestRepo.CreateSummary(ctx, forest.Summary{ID: "clean-subsummary", Body: "clean nested summary", FacetID: "episode", Depth: 1}, []forest.Edge{{ChildID: cleanDirectChild.ID, ChildKind: "entry"}})
	require.NoError(t, err)
	_, err = forestRepo.CreateSummary(ctx, forest.Summary{ID: "unrelated-summary", Body: "unrelated clean summary", FacetID: "episode", Depth: 1}, []forest.Edge{{ChildID: unrelatedLeaf.ID, ChildKind: "entry"}})
	require.NoError(t, err)
	_, err = forestRepo.CreateSummary(ctx, forest.Summary{ID: "dirty-root", Body: "ancestor of superseded source", FacetID: "episode", Depth: 2}, []forest.Edge{
		{ChildID: "dirty-child", ChildKind: "summary"},
		{ChildID: "clean-subsummary", ChildKind: "summary"},
	})
	require.NoError(t, err)
	_, err = forestRepo.CreateSummary(otherScope, forest.Summary{ID: "foreign-summary", Body: "foreign summary", FacetID: "episode", Depth: 1}, []forest.Edge{{ChildID: foreignMarked.ID, ChildKind: "entry"}})
	require.NoError(t, err)
	require.NoError(t, facetRepo.MarkSuperseded(ctx, oldFrontier.ID, replacement.ID))
	require.NoError(t, facetRepo.MarkSuperseded(ctx, pinnedOld.ID, replacement.ID))
	require.NoError(t, facetRepo.MarkSuperseded(ctx, oldNested.ID, replacement.ID))
	require.NoError(t, facetRepo.MarkSuperseded(otherScope, foreignMarked.ID, foreignReplacement.ID))
	for _, id := range []string{pinnedOld.ID, cleanPinned.ID} {
		_, err = db.Primary().ExecContext(ctx, `INSERT INTO pins(entry_id,review_at,pinned_at) VALUES(?,NULL,?)`, id, time.Now().UTC().Format(time.RFC3339Nano))
		require.NoError(t, err)
	}

	nodes, err := NewSQLiteSource(db.Primary()).Nodes(ctx)
	require.NoError(t, err)
	semantic := make(map[string]Node, len(nodes))
	for _, node := range nodes {
		semantic[node.ID] = node
	}
	for _, id := range []string{oldFrontier.ID, pinnedOld.ID, oldNested.ID, "dirty-child", "dirty-root"} {
		require.NotContains(t, semantic, id, "superseded content and summaries that contain it are not recall candidates")
	}
	for _, id := range []string{replacement.ID, cleanNestedSibling.ID, cleanDirectChild.ID, "clean-subsummary", cleanPinned.ID, "unrelated-summary"} {
		require.Contains(t, semantic, id, "clean records and records outside this scope remain available")
	}
	require.Empty(t, semantic[cleanNestedSibling.ID].ParentID)
	require.True(t, semantic[cleanNestedSibling.ID].Frontier)
	require.Empty(t, semantic["clean-subsummary"].ParentID)
	require.True(t, semantic["clean-subsummary"].Frontier)
	require.Equal(t, "clean-subsummary", semantic[cleanDirectChild.ID].ParentID)
	require.NotContains(t, semantic, foreignMarked.ID)
	require.NotContains(t, semantic, "foreign-summary")

	ambientNodes, err := NewSQLiteSource(db.Primary()).AmbientNodes(ctx, 50)
	require.NoError(t, err)
	ambient := make(map[string]Node, len(ambientNodes))
	for _, node := range ambientNodes {
		ambient[node.ID] = node
		require.Empty(t, node.Vectors)
	}
	for _, id := range []string{oldFrontier.ID, pinnedOld.ID, oldNested.ID, "dirty-child", "dirty-root"} {
		require.NotContains(t, ambient, id, "a pin must not resurrect superseded content")
	}
	for _, id := range []string{replacement.ID, cleanNestedSibling.ID, "clean-subsummary", cleanPinned.ID} {
		require.Contains(t, ambient, id, "clean children are promoted to the frontier when a superseded ancestor summary is hidden")
	}
	require.True(t, ambient[cleanNestedSibling.ID].Frontier)
	require.True(t, ambient["clean-subsummary"].Frontier)
	require.True(t, ambient[cleanPinned.ID].Pinned)
	require.Contains(t, ambient, "unrelated-summary", "a clean summary remains a frontier node")
	require.NotContains(t, ambient, unrelatedLeaf.ID, "a clean leaf still absorbed by a clean summary is not promoted")
	foreignAmbient, err := NewSQLiteSource(db.Primary()).AmbientNodes(otherScope, 50)
	require.NoError(t, err)
	foreignIDs := make(map[string]struct{}, len(foreignAmbient))
	for _, node := range foreignAmbient {
		foreignIDs[node.ID] = struct{}{}
	}
	require.NotContains(t, foreignIDs, foreignMarked.ID)
	require.NotContains(t, foreignIDs, "foreign-summary", "supersession also taints summaries in its own scope")
	require.Contains(t, foreignIDs, foreignReplacement.ID)

	historical, err := journalRepo.Get(ctx, oldNested.ID)
	require.NoError(t, err)
	require.Equal(t, "old nested memory", historical.Body, "superseding changes recall projection, not journal history")
}

func appendEntryInScope(t *testing.T, repo *journal.SQLiteRepository, ctx context.Context, id, body string) journal.Entry {
	t.Helper()
	entry, err := repo.Append(ctx, journal.Entry{ID: id, Scope: string(policy.ScopeFromContext(ctx)), Body: body, FacetID: "episode"}, nil)
	require.NoError(t, err)
	return entry
}

func TestLivenessCountsUnabsorbedLeavesAndReportsSummaryTimes(t *testing.T) { // [REQ:SL-P0-003]
	ctx := context.Background()
	db, err := apidb.Open(ctx, apidb.Config{Driver: apidb.DriverSQLite, DSN: "file:recall-liveness?mode=memory&cache=shared"})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(journal.Schema), apidb.SchemaProviderFunc(facets.Schema), apidb.SchemaProviderFunc(forest.Schema)))
	repo := journal.NewSQLiteRepository(db.Primary())
	older, err := repo.Append(ctx, journal.Entry{ID: "older-leaf", Body: "older", FacetID: "episode", CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}, nil)
	require.NoError(t, err)
	_, err = repo.Append(ctx, journal.Entry{ID: "newer-leaf", Body: "newer", FacetID: "episode", CreatedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)}, nil)
	require.NoError(t, err)
	_, err = forest.NewSQLiteRepository(db.Primary()).CreateSummary(ctx, forest.Summary{ID: "summary", Scope: "agent-memory", Body: "summary", FacetID: "episode", CreatedAt: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)}, []forest.Edge{{ParentID: "summary", ChildID: older.ID, ChildKind: "entry"}})
	require.NoError(t, err)

	liveness, err := NewSQLiteSource(db.Primary()).Liveness(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, liveness.UnsummarizedLeafCount)
	require.Equal(t, "2026-01-02T00:00:00Z", liveness.OldestUnsummarizedLeafAt)
	require.Equal(t, "2026-01-03T00:00:00Z", liveness.LastSummaryAt)
}
