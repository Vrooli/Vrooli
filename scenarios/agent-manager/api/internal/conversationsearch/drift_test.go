package conversationsearch

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func driftDocuments(n int) []Document {
	documents := make([]Document, 0, n)
	for k := 0; k < n; k++ {
		document := testDocument()
		document.DocumentID = fmt.Sprintf("doc-%03d", k)
		document.SourceEventID = fmt.Sprintf("event-%03d", k)
		document.EventSequence = int64(k)
		document.ContentHash = fmt.Sprintf("hash-%03d", k)
		document.EvidenceRef = "agent-manager://runs/run-1/events/" + document.SourceEventID
		documents = append(documents, document)
	}
	return documents
}

// servingIndexer builds and activates one generation from source so each test
// starts from a catalog that matches it.
func servingIndexer(t *testing.T, source *mutableProjectionSource, options IndexerOptions) (*Indexer, *SQLiteRepository) {
	t.Helper()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	options.Source, options.Repository = source, repository
	indexer, err := NewIndexer(options)
	require.NoError(t, err)
	job, err := indexer.RunOnce(t.Context(), 0)
	require.NoError(t, err)
	require.Equal(t, ReindexComplete, job.State)
	return indexer, repository
}

func generationCount(t *testing.T, repository *SQLiteRepository) int {
	t.Helper()
	var count int
	require.NoError(t, repository.db.GetContext(t.Context(), &count, `SELECT COUNT(*) FROM conversation_search_generations`))
	return count
}

func TestSetDigestIgnoresTraversalOrder(t *testing.T) {
	t.Parallel()
	var forward, backward setDigest
	documents := driftDocuments(5)
	for k := range documents {
		forward.add(documents[k].DocumentID, documents[k].ContentHash)
		backward.add(documents[len(documents)-1-k].DocumentID, documents[len(documents)-1-k].ContentHash)
	}
	require.True(t, forward.equal(backward))
	var changed setDigest
	for k, document := range documents {
		if k == 2 {
			document.ContentHash += "-edited"
		}
		changed.add(document.DocumentID, document.ContentHash)
	}
	require.False(t, forward.equal(changed))
}

func TestDriftCheckLeavesAMatchingCatalogAlone(t *testing.T) {
	t.Parallel()
	source := &mutableProjectionSource{documents: driftDocuments(12)}
	indexer, repository := servingIndexer(t, source, IndexerOptions{PageSize: 5})
	before := generationCount(t, repository)

	indexer.checkDrift(t.Context())

	require.Len(t, indexer.SortedJobs(), 1, "a catalog that matches its source must not start another generation")
	require.Equal(t, before, generationCount(t, repository))
}

func TestDriftCheckRebuildsWhenSourceAndCatalogDisagree(t *testing.T) {
	t.Parallel()
	source := &mutableProjectionSource{documents: driftDocuments(12)}
	indexer, repository := servingIndexer(t, source, IndexerOptions{PageSize: 5})
	// The change bypassed the queue, so only the source comparison can see it.
	source.documents[3].Content, source.documents[3].ContentHash = "drifted content", "drifted-hash"

	indexer.checkDrift(t.Context())

	require.Eventually(t, func() bool {
		jobs := indexer.SortedJobs()
		return len(jobs) == 2 && jobs[1].State == ReindexComplete
	}, 5*time.Second, 10*time.Millisecond)
	served, err := repository.GetDocument(t.Context(), source.documents[3].DocumentID)
	require.NoError(t, err)
	require.Equal(t, "drifted-hash", served.ContentHash)
}

func TestDriftCheckRebuildsADegradedServingGeneration(t *testing.T) {
	t.Parallel()
	source := &mutableProjectionSource{documents: driftDocuments(3)}
	indexer, repository := servingIndexer(t, source, IndexerOptions{})
	_, err := repository.db.ExecContext(t.Context(), `UPDATE conversation_search_generations SET failed_documents = 1 WHERE state = 'active'`)
	require.NoError(t, err)

	indexer.checkDrift(t.Context())

	require.Eventually(t, func() bool { return len(indexer.SortedJobs()) == 2 }, 5*time.Second, 10*time.Millisecond)
}

func TestDriftCheckSkipsWhenAChangeLandsDuringTheWalk(t *testing.T) {
	t.Parallel()
	source := &mutableProjectionSource{documents: driftDocuments(4)}
	indexer, repository := servingIndexer(t, source, IndexerOptions{})
	source.documents[0].ContentHash = "drifted-hash"
	hooked := &changeDuringScanSource{mutableProjectionSource: source, enqueue: func() {
		// Recorded and already processed: nothing is pending afterwards, but
		// the two sides no longer describe one moment.
		require.NoError(t, repository.EnqueueChange(context.Background(), ChangeUpsertRun, "run-1", "", time.Now().UTC()))
		latest, err := repository.LatestChangeSequence(context.Background())
		require.NoError(t, err)
		require.NoError(t, repository.MarkChangesProcessed(context.Background(), latest, time.Now().UTC()))
	}}
	indexer.source = hooked

	indexer.checkDrift(t.Context())

	require.Len(t, indexer.SortedJobs(), 1, "a comparison spanning a change must wait for the next check")
}

type changeDuringScanSource struct {
	*mutableProjectionSource
	enqueue func()
	done    bool
}

func (s *changeDuringScanSource) LoadSourcePage(ctx context.Context, cursor *SourceCursor, limit int) (SourcePage, error) {
	if !s.done {
		s.done = true
		s.enqueue()
	}
	return s.mutableProjectionSource.LoadSourcePage(ctx, cursor, limit)
}

func TestStartupRebuildsOnlyAStaleRecipe(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		recipe  string
		rebuild bool
	}{{DefaultRecipeVersion, false}, {"conversation-search-v0", true}} {
		t.Run(test.recipe, func(t *testing.T) {
			t.Parallel()
			db := openProjectionTestDB(t)
			applyProjectionSchema(t, db)
			repository := NewSQLiteRepository(db)
			now := time.Now().UTC()
			require.NoError(t, repository.SaveGeneration(t.Context(), Generation{GenerationID: "serving", State: "active", RecipeVersion: test.recipe, CreatedAt: now, UpdatedAt: now}))
			indexer, err := NewIndexer(IndexerOptions{Source: &mutableProjectionSource{documents: driftDocuments(2)}, Repository: repository})
			require.NoError(t, err)

			require.NoError(t, indexer.launchInitial(t.Context()))

			if test.rebuild {
				require.Eventually(t, func() bool { return len(indexer.SortedJobs()) == 1 }, 5*time.Second, 10*time.Millisecond)
			} else {
				require.Empty(t, indexer.SortedJobs())
			}
		})
	}
}

// TestOwnerLoopTickNeverRebuildsAnIdleCorpus is the regression for the
// unconditional full rebuild on every repair tick.
func TestOwnerLoopTickNeverRebuildsAnIdleCorpus(t *testing.T) {
	t.Parallel()
	source := &mutableProjectionSource{documents: driftDocuments(3)}
	indexer, repository := servingIndexer(t, source, IndexerOptions{ChangeInterval: 5 * time.Millisecond, DriftCheckDelay: time.Hour})
	before := generationCount(t, repository)
	ctx, cancel := context.WithCancel(t.Context())
	finished := make(chan struct{})
	go func() { defer close(finished); indexer.loop(ctx) }()

	time.Sleep(150 * time.Millisecond)
	cancel()
	<-finished

	require.Len(t, indexer.SortedJobs(), 1)
	require.Equal(t, before, generationCount(t, repository), "idle ticks must not create generations")
}
