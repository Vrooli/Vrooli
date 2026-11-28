package recall

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"

	"source-ledger/internal/classification"
	localdb "source-ledger/internal/database"
	"source-ledger/internal/facets"
	"source-ledger/internal/forest"
	"source-ledger/internal/journal"
	"source-ledger/internal/policy"
	vectorcodec "source-ledger/internal/vector"
)

func TestRecallVectorSnapshotRefreshesOnEnrichmentAndStaysScoped(t *testing.T) {
	ctx := t.Context()
	db, err := apidb.Open(ctx, apidb.Config{Driver: apidb.DriverSQLite, DSN: "file:recall-vector-version?mode=memory&cache=shared"})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, apidb.EnsureSchemas(ctx, db.Primary(), apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(journal.Schema), apidb.SchemaProviderFunc(facets.Schema),
		apidb.SchemaProviderFunc(forest.Schema), apidb.SchemaProviderFunc(vectorcodec.Schema)))
	repository := journal.NewSQLiteRepository(db.Primary())
	appendEntry := func(ctx context.Context, id string) {
		_, err := repository.Append(ctx, journal.Entry{
			ID: id, Scope: string(policy.ScopeFromContext(ctx)), Body: "Searchable evidence", FacetID: "episode",
		}, nil)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, `INSERT INTO facet_texts(id,entry_id,kind,text) VALUES(?,?,?,?)`, id+"-text", id, "topic", id)
		require.NoError(t, err)
	}
	appendEntry(ctx, "public")
	private := policy.WithScope(ctx, "team:private")
	appendEntry(private, "private")
	serving, err := vectorcodec.NewSQLiteGenerationServing(db.Reader())
	require.NoError(t, err)
	reader := NewSQLiteSource(db.Reader(), serving)
	initial, err := reader.NodeVectors(ctx)
	require.NoError(t, err)
	require.Empty(t, initial)
	_, err = db.ExecContext(ctx, `INSERT INTO embeddings(id,facet_text_id,vector_json,vector_blob,created_at)
VALUES('public-vector','public-text','',?,?), ('private-vector','private-text','',?,?)`,
		vectorcodec.Encode([]float64{1, 0}), "2026-11-27", vectorcodec.Encode([]float64{0, 1}), "2026-11-27")
	require.NoError(t, err)
	publicVectors, err := reader.NodeVectors(ctx)
	require.NoError(t, err)
	require.Equal(t, map[string][][]float64{"public": {{1, 0}}}, publicVectors)
	warm, err := reader.NodeVectors(ctx)
	require.NoError(t, err)
	require.Same(t, &publicVectors["public"][0][0], &warm["public"][0][0])
	privateVectors, err := reader.NodeVectors(private)
	require.NoError(t, err)
	require.Equal(t, map[string][][]float64{"private": {{0, 1}}}, privateVectors)
	_, err = db.ExecContext(private, `UPDATE embeddings SET vector_blob=? WHERE id='private-vector'`, vectorcodec.Encode([]float64{1, 1}))
	require.NoError(t, err)
	privateVectors, err = reader.NodeVectors(private)
	require.NoError(t, err)
	require.Equal(t, map[string][][]float64{"private": {{1, 1}}}, privateVectors)
}

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

func TestCurrentAccountRetainsScopedStateAndHistoricalEvidence(t *testing.T) {
	ctx := context.Background()
	db, err := apidb.Open(ctx, apidb.Config{Driver: apidb.DriverSQLite, DSN: "file:recall-account?mode=memory&cache=shared"})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, apidb.EnsureSchemas(ctx, db.Primary(), apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(journal.Schema), apidb.SchemaProviderFunc(facets.Schema), apidb.SchemaProviderFunc(forest.Schema)))
	repo := journal.NewSQLiteRepository(db.Primary())
	appendEntry := func(id, body, evidence, hint string) {
		t.Helper()
		entry := journal.Entry{ID: id, Body: body, FacetID: "episode", Evidence: evidence, RoutingHint: hint}
		if hint == "defect-account" {
			account, valid := decodeAccountEvidence(evidence)
			require.True(t, valid)
			entry.Kind = "observation"
			entry.ImportKey = fmt.Sprintf("request:remember/scenario-qa/%s/%d/0", account.CaseID, account.Revision)
		}
		_, err := repo.Append(ctx, entry, nil)
		require.NoError(t, err)
	}
	appendEntry("first", "Browser capture is broken.", "log://first", "")
	_, err = forest.NewSQLiteRepository(db.Primary()).CreateSummary(ctx,
		forest.Summary{ID: "prior-forest", Body: "Browser capture is broken.", FacetID: "episode"},
		[]forest.Edge{{ChildID: "first", ChildKind: "entry"}})
	require.NoError(t, err)
	appendEntry("local", "Verified browser capture fix for local sessions.", "log://local", "")
	appendEntry("remote", "Remote browser capture still times out.", "log://remote", "")
	appendEntry("setup", "Select the recorder configuration to set up browser capture.", "guide://setup", "")
	appendEntry("current", "Browser capture fixed locally. Remote browser capture still times out.",
		`{"case_id":"defect-capture","revision":2,"evidence_ids":["first","local","remote","setup"],"reconciled_ids":["first","local","remote"],"states":[{"scope":"local","status":"fixed"},{"scope":"remote","status":"confirmed"}]}`, "defect-account")
	appendEntry("delayed", "Browser capture was unverified everywhere.",
		`{"case_id":"defect-capture","revision":1,"evidence_ids":["first"],"states":[{"scope":"all","status":"suspected"}]}`, "defect-account")
	svc := NewService(NewSQLiteSource(db.Primary()), queryFixture(func(q string) ([]float64, error) {
		if q != "browser capture" && q != "select the recorder configuration to set up browser capture" {
			t.Errorf("query = %q, want the topic without category and status", q)
		}
		return nil, errors.New("offline")
	}), Config{})
	svc.SetClassificationIndex(categoryIndexFixture{
		categories: []classification.Category{{ID: "defect", Name: "Defects", Aliases: []string{"bugs"}}},
		matches:    map[string][]string{"first": {"defect"}},
	})
	result, err := svc.Recall(ctx, "open browser capture bugs", 5)
	require.NoError(t, err)
	require.Len(t, result.Hits, 1, "setup evidence has no case state")
	require.Equal(t, "account:current", result.Hits[0].Node.ID)
	require.Contains(t, result.Hits[0].Node.Text, "Remote browser capture still times out")
	broad, err := svc.Recall(ctx, "browser capture", 5)
	require.NoError(t, err)
	require.Len(t, broad.Hits, 2)
	broad, err = svc.Recall(ctx, "select the recorder configuration to set up browser capture", 5)
	require.NoError(t, err)
	require.Equal(t, "setup", broad.Hits[0].Node.ID)
	_, err = repo.Append(ctx, journal.Entry{
		ID: "legacy-report", Kind: "legacy-bug-report", FacetID: "episode", Body: "An unrelated migrated report",
		ImportKey: "request:legacy-scenario-qa-bugs/v1/snapshot",
	}, nil)
	require.NoError(t, err)
	ambient, err := svc.Wake(ctx, 0)
	require.NoError(t, err)
	require.Len(t, ambient.Hits, 2)
	require.Equal(t, "account:current", ambient.Hits[0].Node.ID)

	children, err := svc.Zoom(ctx, result.Hits[0].Node.ID)
	require.NoError(t, err)
	require.Len(t, children, 5)
	require.Equal(t, "log://first", children[0].Evidence)
	require.Equal(t, "log://remote", children[2].Evidence)
	otherScope, err := svc.Zoom(policy.WithScope(ctx, "team:other"), result.Hits[0].Node.ID)
	require.NoError(t, err)
	require.Empty(t, otherScope)

	_, err = repo.Append(ctx, journal.Entry{
		ID: "unpublished", Kind: "observation", FacetID: "episode", Body: "Unrelated hint",
		RoutingHint: "defect-account",
		Evidence:    `{"case_id":"defect-capture","revision":80,"evidence_ids":["first"],"states":[{"scope":"all","status":"fixed"}]}`,
		ImportKey:   "request:ordinary-observation",
	}, nil)
	require.NoError(t, err)
	_, err = repo.Append(ctx, journal.Entry{ID: "other-scope", Scope: "team:other", Body: "Private evidence", FacetID: "episode"}, nil)
	require.NoError(t, err)
	appendEntry("invalid-scope", "Unrelated reference", `{"case_id":"defect-capture","revision":100,"evidence_ids":["first","other-scope"],"states":[{"scope":"all","status":"fixed"}]}`, "defect-account")
	for _, invalidID := range []string{"unpublished", "invalid-scope"} {
		invalid, err := svc.Zoom(ctx, accountPrefix+invalidID)
		require.NoError(t, err)
		require.Empty(t, invalid)
	}

	appendEntry("fixed", "Remote browser capture fix verified.", "log://remote-fixed", "")
	appendEntry("closed", "Browser capture fixed and verified locally and remotely.",
		`{"case_id":"defect-capture","revision":3,"evidence_ids":["first","local","remote","fixed","setup"],"reconciled_ids":["first","local","remote","fixed"],"states":[{"scope":"local","status":"fixed"},{"scope":"remote","status":"fixed"}]}`, "defect-account")
	result, err = svc.Recall(ctx, "open browser capture bugs", 5)
	require.NoError(t, err)
	require.Empty(t, result.Hits, "closed cases and setup notes are outside the open view")
	result, err = svc.RecallCategory(ctx, "", 5, "bugs")
	require.NoError(t, err)
	require.Len(t, result.Hits, 1)
	require.Equal(t, "account:closed", result.Hits[0].Node.ID)
}
