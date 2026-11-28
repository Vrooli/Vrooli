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
	budget   time.Duration
	// validate is the pre-promotion check the indexer handed over (nil: none),
	// and during runs inside the pass, before it returns.
	validate func(context.Context) error
	during   func(context.Context)
}

func (r *recordingChangeRebuilder) Rebuild(context.Context, string) error { return nil }

func (r *recordingChangeRebuilder) RebuildChanges(ctx context.Context, _ string, _ []Document, deleted []string, validate func(context.Context) error) error {
	r.calls++
	r.validate = validate
	if r.during != nil {
		r.during(ctx)
	}
	r.deletes = append([]string(nil), deleted...)
	var deadline time.Time
	deadline, r.deadline = ctx.Deadline()
	r.budget = time.Until(deadline)
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
	require.Greater(t, rebuilder.budget, semanticIncrementalOverhead-time.Minute,
		"the budget must cover the collection copy, validation and promotion every pass pays, not only its two documents")
}

func TestSemanticChangesBudgetIsOverheadPlusEmbeddingAndStaysBounded(t *testing.T) {
	t.Parallel()
	require.Equal(t, semanticIncrementalOverhead, semanticChangesBudget(0))
	require.Equal(t, semanticIncrementalOverhead+4*time.Duration(semanticRebuildPerDocument), semanticChangesBudget(4))
	require.Equal(t, semanticRebuildMaximumBudget, semanticChangesBudget(1<<40), "a runaway change set cannot unbound the pass")
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

	job, err := indexer.reindex(t.Context(), 0, "shrinking-event", false, true, "test")
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

// A kick or tick can find a queue another drain already emptied. That must
// finish quietly: a failed generation row for "nothing to do" is what filled
// the catalog with junk and read as a failing index.
func TestIncrementalDrainOfAnEmptyQueueLeavesNoGenerationRow(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	indexer, err := NewIndexer(IndexerOptions{Source: &mutableProjectionSource{}, Repository: repository})
	require.NoError(t, err)

	job, err := indexer.reindex(t.Context(), 0, "empty-drain", false, true, "test")
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		current, ok := indexer.Status(job.ID)
		return ok && current.State == ReindexComplete
	}, time.Second, time.Millisecond)
	require.Eventually(t, func() bool {
		indexer.mu.Lock()
		defer indexer.mu.Unlock()
		return !indexer.running
	}, time.Second, time.Millisecond)

	var rows int
	require.NoError(t, db.GetContext(t.Context(), &rows, `SELECT COUNT(*) FROM conversation_search_generations`))
	require.Zero(t, rows, "an empty drain must not mint a generation")
}

// A deletion queued while an incremental semantic pass runs must not fail the
// pass: the candidate is a copy of the serving collection, the deletion stays
// pending and the next pass removes it from both legs, so promoting adds no
// exposure the already-applied lexical leg lacks. Gating on it failed half of
// all passes and left their vectors stale.
func TestIncrementalSemanticPassPromotesWhenADeletionArrivesDuringIt(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	old := testDocument()
	require.NoError(t, repository.UpsertDocument(t.Context(), old))
	next := old
	next.Content = "revised content that must be re-embedded"
	next.ContentHash = "revised"
	next.IndexedAt = time.Now().UTC()
	rebuilder := &recordingChangeRebuilder{}
	rebuilder.during = func(ctx context.Context) {
		require.NoError(t, repository.EnqueueChange(ctx, ChangeDeleteRun, "another-run", "", time.Now().UTC()))
	}
	indexer, err := NewIndexer(IndexerOptions{Source: &mutableProjectionSource{documents: []Document{next}}, Repository: repository, Semantic: rebuilder})
	require.NoError(t, err)
	require.NoError(t, repository.EnqueueChange(t.Context(), ChangeUpsertRun, old.SourceRunID, "", time.Now().UTC()))

	job, err := indexer.reindex(t.Context(), 0, "deletion-during-pass", false, true, "test")
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		current, ok := indexer.Status(job.ID)
		return ok && (current.State == ReindexComplete || current.State == ReindexFailed)
	}, 2*time.Second, time.Millisecond)

	current, _ := indexer.Status(job.ID)
	require.Equal(t, ReindexComplete, current.State, "a later deletion must not fail the pass: %s", current.ErrorCode)
	require.Equal(t, 1, rebuilder.calls)
	require.Nil(t, rebuilder.validate, "the incremental pass must not hand the vector store a deletion gate")
	pending, err := repository.PendingChanges(t.Context(), 10)
	require.NoError(t, err)
	require.Len(t, pending, 1, "the later deletion stays queued for the next pass")
	require.Equal(t, ChangeDeleteRun, pending[0].Operation)
}
