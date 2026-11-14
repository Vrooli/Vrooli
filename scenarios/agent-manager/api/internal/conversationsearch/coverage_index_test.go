package conversationsearch

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	coredb "github.com/vrooli/api-core/database"
)

func TestCoverageCountsUseMetadataIndexWithoutTemporaryGrouping(t *testing.T) {
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	var plans []struct {
		ID     int    `db:"id"`
		Parent int    `db:"parent"`
		Unused int    `db:"notused"`
		Detail string `db:"detail"`
	}
	require.NoError(t, db.Select(&plans, `EXPLAIN QUERY PLAN SELECT COUNT(*) FROM (
 SELECT source_run_id, source_message_id FROM conversation_search_catalog
 WHERE visible=1 GROUP BY source_run_id,source_message_id)`))
	var details []string
	for _, plan := range plans {
		details = append(details, plan.Detail)
	}
	plan := strings.Join(details, "\n")
	require.NotContains(t, plan, "TEMP B-TREE", "coverage grouping must stream ordered metadata: %s", plan)
	require.Contains(t, plan, "COVERING INDEX", "coverage must avoid wide content rows: %s", plan)
}

func TestCoverageMetadataIndexAdoptsPopulatedCatalogAndTracksVisibility(t *testing.T) {
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	// Reproduce the populated pre-index layout in this isolated fixture.
	_, err := db.Exec(`DROP INDEX IF EXISTS idx_conversation_search_visible_messages`)
	require.NoError(t, err)
	r := NewSQLiteRepository(db)
	ctx := context.Background()
	docs := make([]Document, 4)
	for n := range docs {
		docs[n] = testDocument()
		docs[n].DocumentID = []string{"chunk-a", "chunk-b", "other-run", "hidden"}[n]
		docs[n].SourceEventID = "same-event"
		docs[n].SourceMessageID = "same-message"
	}
	docs[0].ChunkTotal = 2
	docs[1].ChunkTotal = 2
	docs[1].ChunkIndex = 1
	docs[2].SourceRunID = "other-run"
	docs[3].SourceEventID = "hidden-event"
	docs[3].SourceMessageID = "hidden-message"
	docs[3].Visible = false
	for _, doc := range docs {
		require.NoError(t, r.UpsertDocument(ctx, doc))
	}
	assertCounts := func(messages, documents uint64) {
		t.Helper()
		m, d, l, err := r.queryCoverage(ctx)
		require.NoError(t, err)
		require.Equal(t, messages, m)
		require.Equal(t, documents, d)
		require.Equal(t, documents, l)
	}
	assertCounts(2, 3)
	for n := 0; n < 2; n++ {
		require.NoError(t, coredb.EnsureSchemas(ctx, db, coredb.SchemaProviderFunc(Schema)))
	}
	assertCounts(2, 3)
	for _, doc := range docs {
		loaded, err := r.GetDocument(ctx, doc.DocumentID)
		require.NoError(t, err)
		require.Equal(t, doc.Content, loaded.Content)
	}
	docs[0].Visible = false
	require.NoError(t, r.UpsertDocument(ctx, docs[0]))
	assertCounts(2, 2)
	require.NoError(t, r.DeleteDocument(ctx, docs[1].DocumentID))
	assertCounts(1, 1)
	docs[3].Visible = true
	require.NoError(t, r.UpsertDocument(ctx, docs[3]))
	assertCounts(2, 2)
}
