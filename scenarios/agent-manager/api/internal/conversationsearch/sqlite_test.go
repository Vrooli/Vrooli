package conversationsearch

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	coredb "github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"
)

func TestSchemaIsIdempotentOnPopulatedDatabaseWithoutProjection(t *testing.T) {
	t.Parallel()

	db := openProjectionTestDB(t)
	ctx := context.Background()
	require.NoError(t, coredb.EnsureSchemas(ctx, db, coredb.SchemaProviderFunc(func() string {
		return `CREATE TABLE IF NOT EXISTS run_events (id TEXT PRIMARY KEY, data TEXT NOT NULL);
INSERT INTO run_events(id, data) VALUES ('canonical-1', '{"content":"must survive"}');`
	})))

	require.NoError(t, coredb.EnsureSchemas(ctx, db, coredb.SchemaProviderFunc(Schema)))
	require.NoError(t, coredb.EnsureSchemas(ctx, db, coredb.SchemaProviderFunc(Schema)))

	var canonical string
	require.NoError(t, db.Get(&canonical, `SELECT data FROM run_events WHERE id = 'canonical-1'`))
	require.Equal(t, `{"content":"must survive"}`, canonical)
	for _, object := range []string{"conversation_search_documents", "conversation_search_fts", "conversation_search_checkpoints", "conversation_search_generations"} {
		var count int
		require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM sqlite_master WHERE name = ?`, object))
		require.Equalf(t, 1, count, "missing schema object %s", object)
	}
}

func TestSQLiteProjectionSynchronizesFTSAndPreservesStableIdentity(t *testing.T) {
	t.Parallel()

	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	ctx := context.Background()
	document := testDocument()

	require.NoError(t, repository.UpsertDocument(ctx, document))
	require.Equal(t, []string{"doc-stable"}, ftsIDs(t, db, "corrected"))

	var initialRowID int64
	require.NoError(t, db.Get(&initialRowID, `SELECT rowid FROM conversation_search_documents WHERE document_id = ?`, document.DocumentID))
	document.Content = "adaptive admission uses the shared capacity ledger"
	document.ContentHash = "content-v2"
	require.NoError(t, repository.UpsertDocument(ctx, document))
	require.Empty(t, ftsIDs(t, db, "corrected"))
	require.Equal(t, []string{"doc-stable"}, ftsIDs(t, db, "capacity"))

	loaded, err := repository.GetDocument(ctx, document.DocumentID)
	require.NoError(t, err)
	require.Equal(t, document.DocumentID, loaded.DocumentID)
	require.Equal(t, document.Tags, loaded.Tags)

	filler := testDocument()
	filler.DocumentID = "doc-filler"
	filler.SourceEventID = "event-filler"
	filler.SourceMessageID = "message-filler"
	filler.Content = "unrelated sentinel"
	filler.ContentHash = "filler-content"
	require.NoError(t, repository.UpsertDocument(ctx, filler))

	require.NoError(t, repository.DeleteDocument(ctx, document.DocumentID))
	require.Empty(t, ftsIDs(t, db, "capacity"))
	require.ErrorIs(t, repository.DeleteDocument(ctx, document.DocumentID), ErrNotFound)

	require.NoError(t, repository.UpsertDocument(ctx, document))
	var replacementRowID int64
	require.NoError(t, db.Get(&replacementRowID, `SELECT rowid FROM conversation_search_documents WHERE document_id = ?`, document.DocumentID))
	require.NotEqual(t, initialRowID, replacementRowID)
	require.Equal(t, document.DocumentID, ftsIDs(t, db, "capacity")[0])
}

func TestSQLiteProjectionRejectsInvalidWriteWithoutFTSResidue(t *testing.T) {
	t.Parallel()

	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	document := testDocument()
	document.Tags = nil
	document.ChunkIndex = 2
	document.ChunkTotal = 1

	err := repository.UpsertDocument(context.Background(), document)
	require.ErrorContains(t, err, "chunk index")
	var catalog, lexical int
	require.NoError(t, db.Get(&catalog, `SELECT COUNT(*) FROM conversation_search_documents`))
	require.NoError(t, db.Get(&lexical, `SELECT COUNT(*) FROM conversation_search_fts`))
	require.Zero(t, catalog)
	require.Zero(t, lexical)
}

func TestIndexerPromotesValidatedShadowAndKeepsPreviousOnFailure(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	old := testDocument()
	old.Content = "previous active generation"
	old.ContentHash = "old-hash"
	require.NoError(t, repository.UpsertDocument(context.Background(), old))

	source := &mutableProjectionSource{documents: []Document{func() Document {
		document := testDocument()
		document.DocumentID = "next-document"
		document.SourceEventID = "next-event"
		document.SourceMessageID = "next-message"
		document.Content = "validated next generation"
		document.ContentHash = "next-hash"
		return document
	}()}}
	indexer, err := NewIndexer(IndexerOptions{Source: source, Repository: repository, PageSize: 1})
	require.NoError(t, err)
	job, err := indexer.RunOnce(context.Background(), 10)
	require.NoError(t, err)
	require.Equal(t, ReindexComplete, job.State)
	require.Empty(t, ftsIDs(t, db, "previous"))
	require.Equal(t, []string{"next-document"}, ftsIDs(t, db, "validated"))

	// A source mutation between staging and validation fails the candidate. The
	// serving projection remains byte-for-byte on the previous active version.
	source.mutateAfterFirstScan = true
	source.loads = 0
	failed, err := indexer.RunOnce(context.Background(), 10)
	require.Error(t, err)
	require.Equal(t, ReindexFailed, failed.State)
	require.Equal(t, []string{"next-document"}, ftsIDs(t, db, "validated"))
	var staged int
	require.NoError(t, db.Get(&staged, `SELECT COUNT(*) FROM conversation_search_generation_documents WHERE generation_id = ?`, failed.ID))
	require.Zero(t, staged)
}

func TestIndexerDeleteNotificationRemovesFTSBeforeAsyncRepair(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	document := testDocument()
	require.NoError(t, repository.UpsertDocument(context.Background(), document))
	indexer, err := NewIndexer(IndexerOptions{Source: &mutableProjectionSource{}, Repository: repository})
	require.NoError(t, err)
	require.NoError(t, indexer.Notify(context.Background(), ChangeDeleteEvent, document.SourceRunID, document.SourceEventID))
	require.Empty(t, ftsIDs(t, db, "corrected"))
	tombstones, err := repository.TombstonedDocumentIDs(context.Background(), document.SourceRunID, document.SourceEventID)
	require.NoError(t, err)
	require.Equal(t, []string{document.DocumentID}, tombstones)
	changes, err := repository.PendingChanges(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, changes, 1)
	require.Equal(t, ChangeDeleteEvent, changes[0].Operation)
}

func TestApplyStagedChangesReplacesOnlyNamedRun(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	ctx := context.Background()

	changed := testDocument()
	changed.Content, changed.ContentHash = "old changed run", "old-changed-hash"
	unrelated := testDocument()
	unrelated.DocumentID, unrelated.SourceRunID = "unrelated-document", "unrelated-run"
	unrelated.SourceEventID, unrelated.SourceMessageID = "unrelated-event", "unrelated-message"
	unrelated.Content, unrelated.ContentHash = "unrelated serving sentinel", "unrelated-hash"
	require.NoError(t, repository.UpsertDocument(ctx, changed))
	require.NoError(t, repository.UpsertDocument(ctx, unrelated))

	const generationID = "bounded-change"
	require.NoError(t, repository.BeginStagedGeneration(ctx, generationID))
	replacement := changed
	replacement.Content, replacement.ContentHash = "new changed run", "new-changed-hash"
	require.NoError(t, repository.StageDocument(ctx, generationID, replacement))
	require.NoError(t, repository.ApplyStagedChanges(ctx, generationID, []ProjectionChange{{
		Sequence: 1, Operation: ChangeUpsertRun, SourceRunID: changed.SourceRunID,
	}}))

	require.Empty(t, ftsIDs(t, db, "old"))
	require.Equal(t, []string{changed.DocumentID}, ftsIDs(t, db, "new"))
	require.Equal(t, []string{unrelated.DocumentID}, ftsIDs(t, db, "unrelated"))
}

func TestInitialGenerationRepublishKeepsFTSInLockstep(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	ctx := context.Background()
	const generationID = "resumable-initial"
	now := time.Now().UTC()
	require.NoError(t, repository.SaveGeneration(ctx, Generation{
		GenerationID: generationID, State: "building", RecipeVersion: DefaultRecipeVersion,
		PlannedDocuments: 1, ProcessedDocuments: 1, CreatedAt: now, UpdatedAt: now,
	}))
	require.NoError(t, repository.BeginStagedGeneration(ctx, generationID))
	require.NoError(t, repository.StageDocument(ctx, generationID, testDocument()))

	require.NoError(t, repository.PublishStagedGeneration(ctx, generationID, 1))
	var firstRowID int64
	require.NoError(t, db.Get(&firstRowID, `SELECT rowid FROM conversation_search_documents WHERE document_id=?`, testDocument().DocumentID))
	require.NoError(t, repository.PublishStagedGeneration(ctx, generationID, 1))
	var resumedRowID int64
	require.NoError(t, db.Get(&resumedRowID, `SELECT rowid FROM conversation_search_documents WHERE document_id=?`, testDocument().DocumentID))
	require.Equal(t, firstRowID, resumedRowID, "an unchanged resumed snapshot must not churn serving or FTS identity")

	changed := testDocument()
	changed.Content, changed.ContentHash = "changed staged content", "changed-content-hash"
	require.NoError(t, repository.StageDocument(ctx, generationID, changed))
	require.NoError(t, repository.PublishStagedGeneration(ctx, generationID, 1))

	var catalog, lexical int
	require.NoError(t, db.Get(&catalog, `SELECT COUNT(*) FROM conversation_search_documents`))
	require.NoError(t, db.Get(&lexical, `SELECT COUNT(*) FROM conversation_search_fts`))
	require.Equal(t, 1, catalog)
	require.Equal(t, catalog, lexical)
	require.Equal(t, []string{changed.DocumentID}, ftsIDs(t, db, "changed"))
}

func TestIndexerSemanticFailureStillPublishesLexicalSnapshot(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	old := testDocument()
	old.Content = "serving generation remains readable"
	old.ContentHash = "serving-hash"
	require.NoError(t, repository.UpsertDocument(context.Background(), old))
	next := testDocument()
	next.DocumentID, next.SourceEventID, next.SourceMessageID = "candidate", "candidate-event", "candidate-message"
	next.Content, next.ContentHash = "candidate remains lexically searchable", "candidate-hash"
	indexer, err := NewIndexer(IndexerOptions{Source: &mutableProjectionSource{documents: []Document{next}}, Repository: repository, Semantic: failingSemanticRebuilder{}})
	require.NoError(t, err)
	job, err := indexer.RunOnce(context.Background(), 10)
	require.Error(t, err)
	require.Equal(t, ReindexFailed, job.State)
	require.Empty(t, ftsIDs(t, db, "serving"))
	require.Equal(t, []string{"candidate"}, ftsIDs(t, db, "candidate"))
}

func TestIndexerPromotesValidatedSnapshotWhenNewChangesArriveDuringSemanticBuild(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	next := testDocument()
	next.Content, next.ContentHash = "validated snapshot content", "snapshot-hash"
	rebuilder := semanticRebuilderFunc(func(ctx context.Context, _ string) error {
		return repository.EnqueueChange(ctx, ChangeUpsertRun, next.SourceRunID, "", time.Now().UTC())
	})
	indexer, err := NewIndexer(IndexerOptions{Source: &mutableProjectionSource{documents: []Document{next}}, Repository: repository, Semantic: rebuilder})
	require.NoError(t, err)

	job, err := indexer.RunOnce(context.Background(), 10)
	require.NoError(t, err)
	require.Equal(t, ReindexComplete, job.State)
	require.Equal(t, []string{next.DocumentID}, ftsIDs(t, db, "snapshot"))
	changes, err := repository.PendingChanges(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, changes, 1, "post-snapshot change must remain queued for incremental catch-up")
}

func TestIndexerRejectsPromotionWhenDeletionArrivesDuringSemanticBuild(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	old := testDocument()
	old.Content, old.ContentHash = "serving content", "serving-hash"
	require.NoError(t, repository.UpsertDocument(context.Background(), old))
	next := old
	next.Content, next.ContentHash = "candidate content", "candidate-hash"
	rebuilder := semanticRebuilderFunc(func(ctx context.Context, _ string) error {
		return repository.EnqueueChange(ctx, ChangeDeleteEvent, next.SourceRunID, next.SourceEventID, time.Now().UTC())
	})
	indexer, err := NewIndexer(IndexerOptions{Source: &mutableProjectionSource{documents: []Document{next}}, Repository: repository, Semantic: rebuilder})
	require.NoError(t, err)

	job, err := indexer.RunOnce(context.Background(), 10)
	require.ErrorContains(t, err, "canonical deletion arrived")
	require.Equal(t, ReindexFailed, job.State)
	require.Equal(t, []string{next.DocumentID}, ftsIDs(t, db, "candidate"))
}

func TestIndexerAppliesQueuedRunChangeIncrementally(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	old := testDocument()
	old.Content = "old incremental content"
	old.ContentHash = "old-incremental"
	require.NoError(t, repository.UpsertDocument(context.Background(), old))
	next := old
	next.Content = "new incremental content"
	next.ContentHash = "new-incremental"
	next.IndexedAt = time.Now().UTC()
	source := &mutableProjectionSource{documents: []Document{next}}
	indexer, err := NewIndexer(IndexerOptions{Source: source, Repository: repository})
	require.NoError(t, err)
	require.NoError(t, repository.EnqueueChange(context.Background(), ChangeUpsertRun, old.SourceRunID, "", time.Now().UTC()))
	job, err := indexer.reindex(context.Background(), 0, "incremental-test", false, true)
	require.NoError(t, err)
	require.Eventually(t, func() bool { current, ok := indexer.Status(job.ID); return ok && current.State == ReindexComplete }, time.Second, time.Millisecond)
	require.Empty(t, ftsIDs(t, db, "old"))
	require.Equal(t, []string{old.DocumentID}, ftsIDs(t, db, "new"))
	changes, err := repository.PendingChanges(context.Background(), 10)
	require.NoError(t, err)
	require.Empty(t, changes)
}

func TestIndexerCancellationLeavesServingProjectionReadable(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	old := testDocument()
	old.Content = "readable during cancellation"
	old.ContentHash = "cancel-old"
	require.NoError(t, repository.UpsertDocument(context.Background(), old))
	source := &blockingProjectionSource{started: make(chan struct{})}
	indexer, err := NewIndexer(IndexerOptions{Source: source, Repository: repository})
	require.NoError(t, err)
	job, err := indexer.reindex(context.Background(), 0, "cancel-test", false, false)
	require.NoError(t, err)
	<-source.started
	require.True(t, indexer.Cancel(job.ID))
	require.Eventually(t, func() bool { current, ok := indexer.Status(job.ID); return ok && current.State == ReindexCancelled }, time.Second, time.Millisecond)
	require.Equal(t, []string{old.DocumentID}, ftsIDs(t, db, "readable"))
}

func TestIndexerRestartRollsBackInterruptedShadowBeforeRepair(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	now := time.Now().UTC()
	generation := Generation{GenerationID: "interrupted", State: "building", RecipeVersion: DefaultRecipeVersion, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, repository.SaveGeneration(context.Background(), generation))
	require.NoError(t, repository.BeginStagedGeneration(context.Background(), generation.GenerationID))
	require.NoError(t, repository.StageDocument(context.Background(), generation.GenerationID, testDocument()))
	indexer, err := NewIndexer(IndexerOptions{Source: &mutableProjectionSource{}, Repository: repository})
	require.NoError(t, err)
	indexer.recoverInterrupted(context.Background())
	recovered, err := repository.LoadGeneration(context.Background(), generation.GenerationID)
	require.NoError(t, err)
	require.Equal(t, "failed", recovered.State)
	count, err := repository.CountStagedGeneration(context.Background(), generation.GenerationID)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestIndexerRestartResumesCompleteStagedSnapshot(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	now := time.Now().UTC()
	document := testDocument()
	generation := Generation{
		GenerationID: "interrupted-complete", State: "building", RecipeVersion: DefaultRecipeVersion,
		PlannedDocuments: 1, ProcessedDocuments: 1, SourceCheckpoint: checkpointFor(document), CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, repository.SaveGeneration(context.Background(), generation))
	require.NoError(t, repository.BeginStagedGeneration(context.Background(), generation.GenerationID))
	require.NoError(t, repository.StageDocument(context.Background(), generation.GenerationID, document))
	var rebuilds atomic.Int64
	indexer, err := NewIndexer(IndexerOptions{
		Source: &mutableProjectionSource{}, Repository: repository,
		Semantic: semanticRebuilderFunc(func(context.Context, string) error { rebuilds.Add(1); return nil }),
	})
	require.NoError(t, err)
	indexer.recoverInterrupted(context.Background())
	require.Eventually(t, func() bool {
		job, ok := indexer.Status(generation.GenerationID)
		return ok && job.State == ReindexComplete
	}, time.Second, time.Millisecond)
	require.EqualValues(t, 1, rebuilds.Load())
	require.Equal(t, []string{document.DocumentID}, ftsIDs(t, db, "corrected"))
	recovered, err := repository.LoadGeneration(context.Background(), generation.GenerationID)
	require.NoError(t, err)
	require.Equal(t, "active", recovered.State)
}

func TestIndexerStartupDoesNotReplaceHealthyActiveGeneration(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	now := time.Now().UTC()
	require.NoError(t, repository.SaveGeneration(context.Background(), Generation{
		GenerationID: "already-active", State: "active", RecipeVersion: DefaultRecipeVersion,
		CreatedAt: now, UpdatedAt: now,
	}))
	source := &statusBlockingProjectionSource{called: make(chan struct{})}
	indexer, err := NewIndexer(IndexerOptions{Source: source, Repository: repository})
	require.NoError(t, err)
	indexer.launchInitial(context.Background())
	select {
	case <-source.called:
		t.Fatal("startup launched a full canonical repair despite an active generation")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestIndexerStatusUsesReconciledSnapshotWithoutWalkingCanonicalCorpus(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	source := &statusBlockingProjectionSource{called: make(chan struct{})}
	indexer, err := NewIndexer(IndexerOptions{Source: source, Repository: repository})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	status, err := indexer.StatusSnapshot(ctx)
	require.NoError(t, err)
	require.Zero(t, status.CanonicalMessages)
	select {
	case <-source.called:
		t.Fatal("status must not synchronously walk the canonical corpus")
	default:
	}
}

type blockingProjectionSource struct {
	started chan struct{}
	once    sync.Once
}

type statusBlockingProjectionSource struct{ called chan struct{} }

func (s *statusBlockingProjectionSource) LoadSourcePage(ctx context.Context, _ *SourceCursor, _ int) (SourcePage, error) {
	close(s.called)
	<-ctx.Done()
	return SourcePage{}, ctx.Err()
}

func (s *blockingProjectionSource) LoadSourcePage(ctx context.Context, _ *SourceCursor, _ int) (SourcePage, error) {
	s.once.Do(func() { close(s.started) })
	<-ctx.Done()
	return SourcePage{}, ctx.Err()
}

type failingSemanticRebuilder struct{}

func (failingSemanticRebuilder) Rebuild(context.Context, string) error {
	return errors.New("embedding batch failed")
}

type semanticRebuilderFunc func(context.Context, string) error

func (f semanticRebuilderFunc) Rebuild(ctx context.Context, generation string) error {
	return f(ctx, generation)
}

type mutableProjectionSource struct {
	documents            []Document
	loads                int
	mutateAfterFirstScan bool
}

func (s *mutableProjectionSource) LoadSourcePage(_ context.Context, cursor *SourceCursor, limit int) (SourcePage, error) {
	if cursor != nil {
		return SourcePage{}, nil
	}
	s.loads++
	documents := append([]Document(nil), s.documents...)
	if s.mutateAfterFirstScan && s.loads > 1 && len(documents) > 0 {
		documents[0].ContentHash += "-mutated"
	}
	return SourcePage{Documents: documents}, nil
}

func TestSQLiteProjectionConstraintFailureRollsBackFTSUpdate(t *testing.T) {
	t.Parallel()

	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	require.NoError(t, repository.UpsertDocument(context.Background(), testDocument()))

	_, err := db.ExecContext(context.Background(), `UPDATE conversation_search_documents
        SET content = 'poison replacement', tags_json = '{'
        WHERE document_id = 'doc-stable'`)
	require.Error(t, err)
	require.Equal(t, []string{"doc-stable"}, ftsIDs(t, db, "corrected"))
	require.Empty(t, ftsIDs(t, db, "poison"))
}

func TestSQLiteProjectionVisibilityDeletesAndStatus(t *testing.T) {
	t.Parallel()

	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	ctx := context.Background()
	visible := testDocument()
	hidden := testDocument()
	hidden.DocumentID = "doc-hidden"
	hidden.SourceEventID = "event-hidden"
	hidden.SourceMessageID = "message-hidden"
	hidden.Visible = false

	require.NoError(t, repository.UpsertDocument(ctx, visible))
	require.NoError(t, repository.UpsertDocument(ctx, hidden))
	visibleCount, catalogCount, lexicalCount, err := repository.CountCoverage(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), visibleCount)
	require.Equal(t, uint64(1), catalogCount)
	require.Equal(t, uint64(1), lexicalCount)
	isVisible, err := repository.VisibleDocument(ctx, hidden.DocumentID)
	require.NoError(t, err)
	require.False(t, isVisible)

	checkpoint := Checkpoint{SourceName: "events", SourceCursor: "42", SourceFingerprint: "source-v1", UpdatedAt: time.Now().UTC()}
	require.NoError(t, repository.SaveCheckpoint(ctx, checkpoint))
	loadedCheckpoint, err := repository.LoadCheckpoint(ctx, checkpoint.SourceName)
	require.NoError(t, err)
	require.Equal(t, checkpoint.SourceCursor, loadedCheckpoint.SourceCursor)

	generation := Generation{GenerationID: "generation-1", State: "active", RecipeVersion: "recipe-v1", PlannedDocuments: 2, ProcessedDocuments: 2, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	require.NoError(t, repository.SaveGeneration(ctx, generation))
	loadedGeneration, err := repository.LoadGeneration(ctx, generation.GenerationID)
	require.NoError(t, err)
	require.Equal(t, generation.State, loadedGeneration.State)

	deleted, err := repository.DeleteRun(ctx, visible.SourceRunID)
	require.NoError(t, err)
	require.Equal(t, int64(2), deleted)
	visibleCount, catalogCount, lexicalCount, err = repository.CountCoverage(ctx)
	require.NoError(t, err)
	require.Zero(t, visibleCount)
	require.Zero(t, catalogCount)
	require.Zero(t, lexicalCount)
}

func TestSQLiteProjectionCoverageSeparatesMessagesFromChunks(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	first := testDocument()
	first.ChunkTotal = 2
	second := first
	second.DocumentID = "doc-stable:1"
	second.ChunkIndex = 1
	second.Content = "second chunk"
	second.ContentHash = digest(second.Content)
	require.NoError(t, repository.UpsertDocument(context.Background(), first))
	require.NoError(t, repository.UpsertDocument(context.Background(), second))

	visibleMessages, catalogDocuments, lexicalDocuments, err := repository.CountCoverage(context.Background())
	require.NoError(t, err)
	require.Equal(t, uint64(1), visibleMessages)
	require.Equal(t, uint64(2), catalogDocuments)
	require.Equal(t, uint64(2), lexicalDocuments)
}

func TestProjectionFilterQueryUsesDeclaredIndex(t *testing.T) {
	t.Parallel()

	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	rows, err := db.Queryx(`EXPLAIN QUERY PLAN SELECT document_id FROM conversation_search_documents
        WHERE visible = 1 AND harness = ? AND occurred_at >= ?
        ORDER BY occurred_at, document_id`, "claude-code", "2026-01-01T00:00:00Z")
	require.NoError(t, err)
	defer rows.Close()
	var details []string
	for rows.Next() {
		var id, parent, notUsed int
		var detail string
		require.NoError(t, rows.Scan(&id, &parent, &notUsed, &detail))
		details = append(details, detail)
	}
	require.NoError(t, rows.Err())
	require.Contains(t, strings.Join(details, "\n"), "idx_conversation_search_harness_time")
}

func TestProjectionTestsUseOnlyMemoryDatabase(t *testing.T) {
	// This guard makes the destructive fixture boundary executable: the helper
	// below is the only database constructor in this package's tests.
	require.Contains(t, projectionTestDSNTemplate, "mode=memory")
	require.NotContains(t, projectionTestDSNTemplate, "agent-manager.db")
}

const projectionTestDSNTemplate = "file:conversation-search-%s?mode=memory&cache=shared"

func openProjectionTestDB(t testing.TB) *sqlx.DB {
	t.Helper()
	dsn := fmt.Sprintf(projectionTestDSNTemplate, strings.ReplaceAll(t.Name(), "/", "-"))
	db, err := sqlx.Connect("sqlite", dsn)
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db
}

func applyProjectionSchema(t testing.TB, db *sqlx.DB) {
	t.Helper()
	require.NoError(t, coredb.EnsureSchemas(context.Background(), db, coredb.SchemaProviderFunc(Schema)))
}

func testDocument() Document {
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	return Document{
		DocumentID: "doc-stable", SourceRunID: "run-1", SourceEventID: "event-1", SourceMessageID: "message-1",
		ChunkIndex: 0, ChunkTotal: 1, EventSequence: 7, Role: "assistant", OccurredAt: now,
		Content: "the corrected reasoning found adaptive phase scheduling", ContentClass: 1,
		SourceHash: "source-v1", ContentHash: "content-v1", RecipeVersion: "recipe-v1",
		Harness: "claude-code", SourceSessionID: "session-public", ProviderOrigin: "agent-manager.runs",
		ProjectScope: "vrooli", CWDScope: "workspace", Runner: "claude", Model: "model", Profile: "profile",
		RunStatus: "complete", RunLabel: "fixture", Tags: []string{"testing"}, Workloads: []string{"implementation"},
		EvidenceRef: "agent-manager://runs/run-1/events/event-1", Visible: true, IndexedAt: now,
	}
}

func ftsIDs(t *testing.T, db *sqlx.DB, query string) []string {
	t.Helper()
	var ids []string
	require.NoError(t, db.Select(&ids, `SELECT document_id FROM conversation_search_fts WHERE conversation_search_fts MATCH ? ORDER BY document_id`, query))
	return ids
}

func TestRepositoryNotFoundContract(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	_, err := NewSQLiteRepository(db).GetDocument(context.Background(), "absent")
	require.True(t, errors.Is(err, ErrNotFound))
}
