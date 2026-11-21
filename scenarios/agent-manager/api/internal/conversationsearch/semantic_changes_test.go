package conversationsearch

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type recordingChangeRebuilder struct {
	calls    int
	deletes  []string
	deadline bool
}

func (r *recordingChangeRebuilder) Rebuild(context.Context, string) error { return nil }

func (r *recordingChangeRebuilder) RebuildChanges(ctx context.Context, _ string, _ []Document, deleted []string, _ func(context.Context) error) error {
	r.calls++
	r.deletes = append([]string(nil), deleted...)
	_, r.deadline = ctx.Deadline()
	return nil
}

func changeTestIndexer(t *testing.T, semantic SemanticRebuilder) *Indexer {
	t.Helper()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	indexer, err := NewIndexer(IndexerOptions{Source: &mutableProjectionSource{}, Repository: NewSQLiteRepository(db), Semantic: semantic})
	require.NoError(t, err)
	return indexer
}

func classDocument(id string, class ContentClass) Document {
	document := testDocument()
	document.DocumentID, document.ContentClass = id, class
	return document
}

// TestSemanticChangesSkipNeverEmbeddedToolDocuments is the regression for
// compacted tool payloads copying the serving vector collection per pass.
func TestSemanticChangesSkipNeverEmbeddedToolDocuments(t *testing.T) {
	t.Parallel()
	rebuilder := &recordingChangeRebuilder{}
	indexer := changeTestIndexer(t, rebuilder)
	documents := []Document{classDocument("tool-call", ContentClassToolCall), classDocument("tool-result", ContentClassToolResult)}

	require.NoError(t, indexer.rebuildSemanticChanges(t.Context(), "generation", 1, documents, []string{"tool-call", "tool-result"}))

	require.Zero(t, rebuilder.calls, "tool documents were never embedded, so their change must not start a semantic generation")
}

func TestSemanticChangesKeepEmbeddedAndTombstonedDeletes(t *testing.T) {
	t.Parallel()
	rebuilder := &recordingChangeRebuilder{}
	indexer := changeTestIndexer(t, rebuilder)
	documents := []Document{classDocument("prose", ContentClassProse), classDocument("tool", ContentClassToolResult)}

	require.NoError(t, indexer.rebuildSemanticChanges(t.Context(), "generation", 1, documents, []string{"prose", "tool", "tombstoned"}))

	require.Equal(t, 1, rebuilder.calls)
	require.ElementsMatch(t, []string{"prose", "tombstoned"}, rebuilder.deletes)
	require.True(t, rebuilder.deadline, "the incremental semantic leg must be time-bounded")
}

// chunkedEvent returns chunks 0..total-1 of one event with the given class.
func chunkedEvent(class ContentClass, total int, hash string) []Document {
	documents := make([]Document, 0, total)
	for index := 0; index < total; index++ {
		document := testDocument()
		document.DocumentID = fmt.Sprintf("event-1-chunk-%d", index)
		document.ChunkIndex, document.ChunkTotal = index, total
		document.ContentClass, document.ContentHash = class, fmt.Sprintf("%s-%d", hash, index)
		documents = append(documents, document)
	}
	return documents
}

// runShrinkingEventChange serves a total-chunk event, then applies one queued
// event change whose source now yields a single chunk, as payload compaction
// does, and returns the recorded semantic calls.
func runShrinkingEventChange(t *testing.T, class ContentClass) *recordingChangeRebuilder {
	t.Helper()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	for _, document := range chunkedEvent(class, 3, "before") {
		require.NoError(t, repository.UpsertDocument(t.Context(), document))
	}
	rebuilder := &recordingChangeRebuilder{}
	indexer, err := NewIndexer(IndexerOptions{
		Source:     &mutableProjectionSource{documents: chunkedEvent(class, 1, "after")},
		Repository: repository,
		Semantic:   rebuilder,
	})
	require.NoError(t, err)
	require.NoError(t, repository.EnqueueChange(t.Context(), ChangeUpsertRun, "run-1", "event-1", time.Now().UTC()))

	job, err := indexer.reindex(t.Context(), 0, "shrinking-event", false, true)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		current, ok := indexer.Status(job.ID)
		return ok && current.State == ReindexComplete
	}, 5*time.Second, time.Millisecond)
	ids, err := repository.RunDocumentIDs(t.Context(), "run-1")
	require.NoError(t, err)
	require.Equal(t, []string{"event-1-chunk-0"}, ids, "the shrunken event keeps only its remaining chunk")
	return rebuilder
}

// TestIncrementalToolPayloadShrinkNeverTouchesTheVectorStore is the regression
// for compacted tool payloads: their vanished chunks were never embedded.
func TestIncrementalToolPayloadShrinkNeverTouchesTheVectorStore(t *testing.T) {
	t.Parallel()
	rebuilder := runShrinkingEventChange(t, ContentClassToolResult)
	require.Zero(t, rebuilder.calls, "a shrinking tool payload must not start a semantic generation")
}

func TestIncrementalProseShrinkStillDeletesItsVanishedChunks(t *testing.T) {
	t.Parallel()
	rebuilder := runShrinkingEventChange(t, ContentClassProse)
	require.Equal(t, 1, rebuilder.calls)
	require.Subset(t, rebuilder.deletes, []string{"event-1-chunk-1", "event-1-chunk-2"})
}

func TestSemanticChangesPassPrivacyDeletionsWithoutDocuments(t *testing.T) {
	t.Parallel()
	rebuilder := &recordingChangeRebuilder{}
	indexer := changeTestIndexer(t, rebuilder)

	require.NoError(t, indexer.rebuildSemanticChanges(t.Context(), "generation", 1, nil, []string{"deleted-prose"}))

	require.Equal(t, 1, rebuilder.calls)
	require.Equal(t, []string{"deleted-prose"}, rebuilder.deletes)
}
