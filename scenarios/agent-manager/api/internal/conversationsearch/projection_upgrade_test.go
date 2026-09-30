package conversationsearch

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func legacyDocument(id, word string) Document {
	document := testDocument()
	document.DocumentID = id
	document.SourceEventID, document.SourceMessageID = "event-"+id, "message-"+id
	document.Content = "legacy transcript mentions " + word
	document.ContentHash = "hash-" + id
	return document
}

// seedLegacyProjection recreates the pre-2026-09-14 shape beside the current
// schema: a catalog keyed by its implicit rowid feeding a content-storing FTS
// table, with its legacy indexes.
func seedLegacyProjection(t *testing.T, repository *SQLiteRepository, documents ...Document) {
	t.Helper()
	ctx := context.Background()
	for _, statement := range []string{
		`CREATE TABLE conversation_search_documents AS SELECT ` + projectionDocumentColumns + ` FROM conversation_search_catalog WHERE 0`,
		`CREATE INDEX idx_conversation_search_content_hash ON conversation_search_documents(content_hash, document_id) WHERE visible = 1`,
		`CREATE INDEX idx_conversation_search_role_time ON conversation_search_documents(role, occurred_at, document_id) WHERE visible = 1`,
		`CREATE VIRTUAL TABLE conversation_search_fts USING fts5(document_id UNINDEXED, content, tokenize = 'unicode61 remove_diacritics 2')`,
		`CREATE TRIGGER conversation_search_documents_ai AFTER INSERT ON conversation_search_documents WHEN new.visible = 1
BEGIN INSERT INTO conversation_search_fts(rowid, document_id, content) VALUES (new.rowid, new.document_id, new.content); END`,
	} {
		_, err := repository.db.ExecContext(ctx, statement)
		require.NoError(t, err)
	}
	for _, document := range documents {
		require.NoError(t, repository.UpsertDocument(ctx, document))
	}
	_, err := repository.db.ExecContext(ctx, `INSERT INTO conversation_search_documents SELECT `+projectionDocumentColumns+` FROM conversation_search_catalog`)
	require.NoError(t, err)
	_, err = repository.db.ExecContext(ctx, `DELETE FROM conversation_search_catalog`)
	require.NoError(t, err)
}

func requireLexicalIntegrity(t *testing.T, repository *SQLiteRepository) {
	t.Helper()
	_, err := repository.db.ExecContext(context.Background(), `INSERT INTO conversation_search_catalog_fts(conversation_search_catalog_fts) VALUES ('integrity-check')`)
	require.NoError(t, err)
}

func TestUpgradeLegacyProjectionCopiesCatalogAndDropsLegacyObjects(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	ctx := context.Background()
	kept, deleted, other := legacyDocument("doc-kept", "lighthouse"), legacyDocument("doc-deleted", "volcano"), legacyDocument("doc-other", "glacier")
	seedLegacyProjection(t, repository, kept, deleted, other)
	// A canonical deletion queued before the upgrade must not be revived.
	require.NoError(t, repository.EnqueueChange(ctx, ChangeDeleteEvent, deleted.SourceRunID, deleted.SourceEventID, time.Now().UTC()))

	upgraded, err := repository.UpgradeLegacyProjection(ctx)

	require.NoError(t, err)
	require.True(t, upgraded)
	ids, err := repository.ProjectionDocumentIDs(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{"doc-kept", "doc-other"}, ids)
	require.Equal(t, []string{"doc-kept"}, ftsIDs(t, db, "lighthouse"))
	require.Empty(t, ftsIDs(t, db, "volcano"))
	var legacyObjects []string
	require.NoError(t, db.Select(&legacyObjects, `SELECT name FROM sqlite_master
WHERE name IN ('conversation_search_documents', 'conversation_search_fts', 'conversation_search_fts_content',
  'conversation_search_documents_ai', 'idx_conversation_search_content_hash', 'idx_conversation_search_role_time')`))
	require.Empty(t, legacyObjects)
	requireLexicalIntegrity(t, repository)

	again, err := repository.UpgradeLegacyProjection(ctx)
	require.NoError(t, err)
	require.False(t, again, "an upgraded projection has nothing left to move")
}

func TestUpgradeLegacyProjectionResumesAfterAPartialCopy(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	ctx := context.Background()
	first, second := legacyDocument("doc-first", "harbor"), legacyDocument("doc-second", "meadow")
	seedLegacyProjection(t, repository, first, second)
	// An interrupted earlier attempt already copied the first document.
	require.NoError(t, repository.UpsertDocument(ctx, first))

	upgraded, err := repository.UpgradeLegacyProjection(ctx)

	require.NoError(t, err)
	require.True(t, upgraded)
	ids, err := repository.ProjectionDocumentIDs(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{"doc-first", "doc-second"}, ids)
	require.Equal(t, []string{"doc-first"}, ftsIDs(t, db, "harbor"))
	requireLexicalIntegrity(t, repository)
}

func TestLaunchInitialUpgradesALegacyProjectionWithoutRebuilding(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	ctx := context.Background()
	kept := legacyDocument("doc-kept", "lighthouse")
	seedLegacyProjection(t, repository, kept)
	now := time.Now().UTC()
	require.NoError(t, repository.SaveGeneration(ctx, Generation{GenerationID: "serving", State: "active", RecipeVersion: DefaultRecipeVersion, CreatedAt: now, UpdatedAt: now}))
	indexer, err := NewIndexer(IndexerOptions{Source: &mutableProjectionSource{documents: []Document{kept}}, Repository: repository})
	require.NoError(t, err)

	require.NoError(t, indexer.launchInitial(ctx))

	require.Empty(t, indexer.SortedJobs(), "the copied catalog keeps serving; no full rebuild")
	require.Equal(t, []string{"doc-kept"}, ftsIDs(t, db, "lighthouse"))
}
