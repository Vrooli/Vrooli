package aisearch

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	pkg "github.com/vrooli/ai-go/search"
	"github.com/vrooli/ai-go/search/searchtest"

	internalartifacts "content-desk/internal/artifacts"
	internalledger "content-desk/internal/ledger"
)

func TestRecordToSourceDocCarriesTypedProjection(t *testing.T) {
	record := Record{
		ID:         "publish:pub-1",
		Kind:       KindPublishRecord,
		Revision:   "sha256:abcd",
		Visibility: VisibilityOperator,
		FollowUp:   "publish/pub-1",
		Title:      "draft-1",
		Snippet:    "published https://example.test/a; channel linkedin",
		Body:       "draft draft-1. channel linkedin",
		Freshness:  "stale",
		Historical: true,
		Metadata:   map[string]any{"channel": "linkedin"},
	}

	doc := recordToSourceDoc(record)
	require.Equal(t, record.ID, doc.ID)
	require.Equal(t, editorialKind, doc.Kind)
	require.Equal(t, record.Revision, doc.ContentHash, "content hash gates source-level drift")
	require.Contains(t, doc.Body, "draft-1")
	require.Contains(t, doc.Body, "freshness: stale")
	require.Contains(t, doc.Body, "historical record")
	require.Equal(t, record.FollowUp, doc.Meta["follow_up"])
	require.Equal(t, record.ID, doc.Meta["id"])
	require.Equal(t, record.Kind, doc.Meta["kind"])
	require.Equal(t, record.Historical, doc.Meta["historical"])
	require.Equal(t, record.Visibility, doc.Meta["visibility"])
	require.Contains(t, doc.Meta, "record")
}

func TestEditorialSourceProjectsEveryLoadedRecord(t *testing.T) {
	source := NewStoreSource(
		sourceDraftsStub{
			drafts:    []internalartifacts.Draft{{ID: "draft-1", Body: "body text"}},
			revisions: map[string]internalartifacts.CurrentRevision{"draft-1": {DraftID: "draft-1"}},
		},
		sourceLedgerStub{},
	)

	docs, err := NewEditorialSource(source).LoadAll(context.Background())
	require.NoError(t, err)
	require.Len(t, docs, 1)
	require.Equal(t, "draft:draft-1", docs[0].ID)
	require.Equal(t, editorialKind, docs[0].Kind)
	require.NotEmpty(t, docs[0].ContentHash)
	require.Equal(t, "draft/draft-1", docs[0].Meta["follow_up"])
}

func TestEngineSearchProjectsTypedHits(t *testing.T) {
	ctx := context.Background()
	store := searchtest.NewVectorStore()
	store.QueryResults = []pkg.SearchResult{
		{ID: "point-1", Score: 0.91, Payload: map[string]any{
			"id":         "draft:draft-1",
			"kind":       KindDraft,
			"title":      "draft-1",
			"snippet":    "draft seed",
			"follow_up":  "draft/draft-1",
			"freshness":  "",
			"historical": false,
		}},
	}
	embedder := &searchtest.Embedder{Vector: []float64{1, 0, 0}, AvailableValue: true}
	eng := newTestEngine(store, embedder, &fakeSource{})

	hits, method, err := eng.Search(ctx, "seed", 10)
	require.NoError(t, err)
	require.NotEmpty(t, method)
	require.Len(t, hits, 1)
	require.Equal(t, "draft:draft-1", hits[0].ID)
	require.Equal(t, "draft-1", hits[0].Title)
	require.Equal(t, "draft/draft-1", hits[0].FollowUp)
	require.False(t, hits[0].Historical)
	require.Equal(t, 0.91, hits[0].Score)
}

type fakeSource struct {
	docs []pkg.SourceDoc
	err  error
}

func (f *fakeSource) LoadAll(context.Context) ([]pkg.SourceDoc, error) {
	return f.docs, f.err
}

func newTestEngine(store pkg.VectorStore, embedder pkg.Embedder, source pkg.Source) *Engine {
	return NewEngineFromComponents(
		EngineComponents{
			Embedder:    embedder,
			VectorStore: store,
			Spec: pkg.CollectionSpec{
				Name:          Collection,
				DenseSize:     3,
				DenseDistance: pkg.DefaultDenseDistance,
				Model:         "test-model",
			},
		},
		source,
		pkg.TuningConfig{Engine: pkg.EngineDense}.WithDefaults(),
		nil,
		2,
		0,
	)
}

// reconcileAll runs one full plan/apply cycle and returns the indexed point count.
func reconcileAll(t *testing.T, eng *Engine, store *searchtest.VectorStore) int {
	t.Helper()
	plan, err := eng.Reconciler().Plan(context.Background())
	require.NoError(t, err)
	_, err = eng.Reconciler().Apply(context.Background(), plan)
	require.NoError(t, err)
	count, err := store.CountPoints(context.Background())
	require.NoError(t, err)
	return count
}

// pointForSource finds the single stored point whose payload source id matches.
// It also asserts the point is not duplicated, which is the "no duplicate stale
// hit" invariant this regression exists to protect.
func pointForSource(t *testing.T, store *searchtest.VectorStore, sourceID string) (string, map[string]any, bool) {
	t.Helper()
	var foundID string
	var found map[string]any
	matches := 0
	for id, point := range store.Points {
		if got, _ := point.Payload["source_id"].(string); got == sourceID {
			matches++
			foundID = id
			found = point.Payload
		}
	}
	require.LessOrEqual(t, matches, 1, "source %s must map to at most one live point", sourceID)
	return foundID, found, matches == 1
}

// TestReconcileTracksOwnerDeletionRevocationAndApprovalChanges proves Phase 3
// ordered step 10: when the editorial owner deletes a draft, revokes a publish
// record, or withdraws an approval for a newer revision, the shared reconciler
// removes the obsolete points and refreshes a changed record in place. A stale
// or superseded record must never accumulate a second live hit.
func TestReconcileTracksOwnerDeletionRevocationAndApprovalChanges(t *testing.T) {
	store := searchtest.NewVectorStore()
	embedder := &searchtest.Embedder{Vector: []float64{1, 0, 0}, AvailableValue: true}

	approved := "operator"
	drafts := &sourceDraftsStub{
		drafts: []internalartifacts.Draft{
			{ID: "draft-keep", Body: "keep body", Status: internalartifacts.DraftApproved},
			{ID: "draft-delete", Body: "delete body", Status: internalartifacts.DraftApproved},
			{ID: "draft-archive", Body: "archive body", Status: internalartifacts.DraftApproved},
		},
		revisions: map[string]internalartifacts.CurrentRevision{
			"draft-keep":    {DraftID: "draft-keep", RevisionID: "rev-keep", ApprovalActorKind: approved},
			"draft-delete":  {DraftID: "draft-delete", RevisionID: "rev-delete", ApprovalActorKind: approved},
			"draft-archive": {DraftID: "draft-archive", RevisionID: "rev-archive", ApprovalActorKind: approved},
		},
	}
	day := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	ledger := &sourceLedgerStub{records: []internalledger.PublishRecord{
		{ID: "pub-keep", DraftID: "draft-keep", Channel: "linkedin", PublishedAt: day},
		{ID: "pub-revoke", DraftID: "draft-keep", Channel: "linkedin", PublishedAt: day.Add(time.Hour)},
	}}
	source := NewStoreSource(drafts, ledger)
	eng := newTestEngine(store, embedder, NewEditorialSource(source))

	require.Equal(t, 5, reconcileAll(t, eng, store), "three drafts plus two publish records are indexed")

	archiveIDBefore, archivePayloadBefore, ok := pointForSource(t, store, "draft:draft-archive")
	require.True(t, ok)
	require.Equal(t, true, archivePayloadBefore["record"].(map[string]any)["approved"])
	_, _, revokedBefore := pointForSource(t, store, "publish:pub-revoke")
	require.True(t, revokedBefore)

	// Owner-side mutation: the draft is deleted (no longer listed), the publish
	// record is revoked (dropped from history), and the archive draft's
	// approval is withdrawn by a newer unapproved revision.
	drafts.drafts = []internalartifacts.Draft{
		{ID: "draft-keep", Body: "keep body", Status: internalartifacts.DraftApproved},
		{ID: "draft-archive", Body: "archive body", Status: internalartifacts.DraftBlocked},
	}
	drafts.revisions["draft-archive"] = internalartifacts.CurrentRevision{DraftID: "draft-archive", RevisionID: "rev-archive-2"}
	source.ledger = &sourceLedgerStub{records: []internalledger.PublishRecord{
		{ID: "pub-keep", DraftID: "draft-keep", Channel: "linkedin", PublishedAt: day},
	}}

	require.Equal(t, 3, reconcileAll(t, eng, store), "deleted and revoked points are removed, not left as ghosts")

	_, _, keepCount := pointForSource(t, store, "draft:draft-keep")
	require.True(t, keepCount)
	_, _, deleteCount := pointForSource(t, store, "draft:draft-delete")
	require.False(t, deleteCount, "a deleted draft must not be served as a current hit")
	_, _, revokedCount := pointForSource(t, store, "publish:pub-revoke")
	require.False(t, revokedCount, "a revoked publish record must not be served as a current hit")

	archiveIDAfter, archivePayloadAfter, archiveExists := pointForSource(t, store, "draft:draft-archive")
	require.True(t, archiveExists)
	require.Equal(t, archiveIDBefore, archiveIDAfter, "a changed record must be refreshed in place, never duplicated")
	archiveRecord, ok := archivePayloadAfter["record"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, false, archiveRecord["approved"], "the newer unapproved revision must not present the withdrawn approval")
	require.Equal(t, "rev-archive-2", archiveRecord["revision_id"])
	require.NotEqual(t, archivePayloadBefore["payload_hash"], archivePayloadAfter["payload_hash"])
}
