package forest

import (
	"context"
	"testing"
	"time"

	"source-ledger/internal/facets"
	"source-ledger/internal/journal"
	vectorcodec "source-ledger/internal/vector"

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

func TestNodesUsesLatestFacetAssignmentAndLoadsEveryEmbeddingSpace(t *testing.T) {
	db, err := apidb.Open(context.Background(), apidb.Config{Driver: apidb.DriverSQLite, DSN: "file:forest-nodes?mode=memory&cache=shared"})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db.Primary(), apidb.SchemaProviderFunc(journal.Schema), apidb.SchemaProviderFunc(facets.Schema), apidb.SchemaProviderFunc(Schema)))
	created := time.Now().UTC().Add(-48 * time.Hour).Format(time.RFC3339Nano)
	_, err = db.Primary().Exec(`INSERT INTO facet_definitions(id,scope,label,created_at) VALUES('unclassified','agent-memory','Unclassified',?),('gotcha','agent-memory','Gotcha',?)`, created, created)
	require.NoError(t, err)
	_, err = db.Primary().Exec(`INSERT INTO facet_policies(facet_id,scope,retention_policy,compaction_eligible) VALUES('unclassified','agent-memory','retain',0),('gotcha','agent-memory','episode',1)`)
	require.NoError(t, err)
	_, err = db.Primary().Exec(`INSERT INTO entries(id,scope,body,facet_id,kind,created_at) VALUES('entry','agent-memory','body','unclassified','test',?)`, created)
	require.NoError(t, err)
	_, err = db.Primary().Exec(`INSERT INTO facet_assignments(id,entry_id,facet_id,assigned_at,actor_id) VALUES('old','entry','unclassified',?,'classifier'),('new','entry','gotcha',?,'operator')`, created, time.Now().UTC().Format(time.RFC3339Nano))
	require.NoError(t, err)
	for index, vector := range [][]float64{{1, 0}, {0, 1}, {1, 1}} {
		textID := "text-" + string(rune('a'+index))
		_, err = db.Primary().Exec(`INSERT INTO facet_texts(id,entry_id,kind,text) VALUES(?,?,?,?)`, textID, "entry", "space", textID)
		require.NoError(t, err)
		_, err = db.Primary().Exec(`INSERT INTO embeddings(id,facet_text_id,vector_json,vector_blob,created_at) VALUES(?,?,?,?,?)`, "embedding-"+textID, textID, "", vectorcodec.Encode(vector), created)
		require.NoError(t, err)
	}
	nodes, err := NewSQLiteRepository(db.Primary()).Nodes(context.Background(), 0)
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	require.Equal(t, "gotcha", nodes[0].FacetID)
	require.Len(t, nodes[0].Vectors, 3)
}
