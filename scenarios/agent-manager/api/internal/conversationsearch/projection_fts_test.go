package conversationsearch

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func lexicalDocumentIDs(t *testing.T, repository *SQLiteRepository, query string) []string {
	t.Helper()
	candidates, err := repository.LexicalCandidates(context.Background(), CandidateQuery{
		Query: query, Limit: 10, ContentClasses: []ContentClass{testDocument().ContentClass},
	})
	require.NoError(t, err)
	ids := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		ids = append(ids, candidate.Document.DocumentID)
	}
	return ids
}

func TestCatalogFTSStaysConsistentThroughUpdatesDeletesAndVisibility(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	ctx := context.Background()
	document := testDocument()

	require.NoError(t, repository.UpsertDocument(ctx, document))
	require.Equal(t, []string{document.DocumentID}, lexicalDocumentIDs(t, repository, "corrected"))
	requireLexicalIntegrity(t, repository)

	document.Content, document.ContentHash = "replacement words about lighthouses", "content-v2"
	require.NoError(t, repository.UpsertDocument(ctx, document))
	require.Empty(t, lexicalDocumentIDs(t, repository, "corrected"), "the index must drop replaced text")
	require.Equal(t, []string{document.DocumentID}, lexicalDocumentIDs(t, repository, "lighthouses"))
	requireLexicalIntegrity(t, repository)

	_, err := db.Exec(`UPDATE conversation_search_catalog SET visible = 0 WHERE document_id = ?`, document.DocumentID)
	require.NoError(t, err)
	require.Empty(t, lexicalDocumentIDs(t, repository, "lighthouses"), "hidden documents are never served")
	requireLexicalIntegrity(t, repository)

	_, err = db.Exec(`UPDATE conversation_search_catalog SET visible = 1 WHERE document_id = ?`, document.DocumentID)
	require.NoError(t, err)
	require.Equal(t, []string{document.DocumentID}, lexicalDocumentIDs(t, repository, "lighthouses"))

	require.NoError(t, repository.DeleteDocument(ctx, document.DocumentID))
	require.Empty(t, lexicalDocumentIDs(t, repository, "lighthouses"))
	requireLexicalIntegrity(t, repository)

	var contentCopies int
	require.NoError(t, db.Get(&contentCopies, `SELECT COUNT(*) FROM sqlite_master WHERE name = 'conversation_search_catalog_fts_content'`))
	require.Zero(t, contentCopies, "the lexical index must read text from the catalog, not store its own copy")
}

// TestCatalogFTSSurvivesVacuum guards the explicit id: VACUUM may renumber an
// implicit rowid, which would point every lexical hit at the wrong document.
func TestCatalogFTSSurvivesVacuum(t *testing.T) {
	t.Parallel()
	db, err := sqlx.Connect("sqlite", "file:"+filepath.Join(t.TempDir(), "vacuum.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	db.SetMaxOpenConns(1)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	ctx := context.Background()
	words := map[string]string{"doc-a": "anchor", "doc-b": "bellows", "doc-c": "compass"}
	for _, id := range []string{"doc-a", "doc-b", "doc-c"} {
		require.NoError(t, repository.UpsertDocument(ctx, legacyDocument(id, words[id])))
	}
	// Removing the lowest row leaves the gap a renumbering VACUUM would close.
	require.NoError(t, repository.DeleteDocument(ctx, "doc-a"))
	var before int64
	require.NoError(t, db.Get(&before, `SELECT id FROM conversation_search_catalog WHERE document_id = 'doc-c'`))

	_, err = db.Exec(`VACUUM`)
	require.NoError(t, err)

	var after int64
	require.NoError(t, db.Get(&after, `SELECT id FROM conversation_search_catalog WHERE document_id = 'doc-c'`))
	require.Equal(t, before, after)
	require.Equal(t, []string{"doc-c"}, lexicalDocumentIDs(t, repository, "compass"))
	require.Equal(t, []string{"doc-b"}, lexicalDocumentIDs(t, repository, "bellows"))
	requireLexicalIntegrity(t, repository)
}
