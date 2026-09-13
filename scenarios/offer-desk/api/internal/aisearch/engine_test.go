package aisearch

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	pkg "github.com/vrooli/ai-go/search"
	"github.com/vrooli/ai-go/search/searchtest"
	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
)

func TestRecordToSourceDocCarriesTypedProjection(t *testing.T) {
	record := Record{
		ID:         "fact:activation_rate",
		Kind:       KindFact,
		Revision:   "sha256:abcd",
		Visibility: VisibilityOperator,
		FollowUp:   "catalog/fact/activation_rate",
		Title:      "activation_rate",
		Snippet:    "fact activation_rate = 0.75; freshness stale",
		Body:       "observed fact activation_rate = 0.75",
		Freshness:  FreshnessStale,
		Historical: true,
		Metadata:   map[string]any{"dimension": "activation"},
	}

	doc := recordToSourceDoc(record)
	require.Equal(t, record.ID, doc.ID)
	require.Equal(t, catalogKind, doc.Kind)
	require.Equal(t, record.Revision, doc.ContentHash, "content hash gates source-level drift")
	require.Contains(t, doc.Body, "activation_rate")
	require.Contains(t, doc.Body, "freshness: stale")
	require.Contains(t, doc.Body, "historical record")
	require.Equal(t, record.FollowUp, doc.Meta["follow_up"])
	require.Equal(t, record.ID, doc.Meta["id"])
	require.Equal(t, record.Kind, doc.Meta["kind"])
	require.Equal(t, record.Freshness, doc.Meta["freshness"])
	require.Equal(t, record.Historical, doc.Meta["historical"])
	require.Equal(t, record.Visibility, doc.Meta["visibility"])
	require.Contains(t, doc.Meta, "record")
}

func TestCatalogSourceProjectsEveryLoadedRecord(t *testing.T) {
	store, source, ctx := testSource(t)
	_, err := store.CreateNode(ctx, offerspb.NodeKind_OFFER, "seed offer", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)

	snapshot, err := source.Load(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, snapshot.Records)

	docs, err := NewCatalogSource(source).LoadAll(ctx)
	require.NoError(t, err)
	require.Len(t, docs, len(snapshot.Records))

	revisions := make(map[string]string, len(snapshot.Records))
	for _, r := range snapshot.Records {
		revisions[r.ID] = r.Revision
	}
	for _, doc := range docs {
		require.Equal(t, revisions[doc.ID], doc.ContentHash)
		require.Equal(t, catalogKind, doc.Kind)
		require.NotEmpty(t, doc.Body)
	}
}

func TestEngineReconcilesAndDeletesGhosts(t *testing.T) {
	ctx := context.Background()
	src := &fakeSource{docs: []pkg.SourceDoc{
		{ID: "node:offer-1", Kind: catalogKind, ContentHash: "sha256:1", Body: "offer one", Meta: map[string]any{"id": "node:offer-1", "kind": "node-offer", "title": "offer one", "follow_up": "catalog/node/offer-1"}},
		{ID: "fact:activation_rate", Kind: catalogKind, ContentHash: "sha256:2", Body: "activation rate", Meta: map[string]any{"id": "fact:activation_rate", "kind": KindFact, "title": "activation_rate", "follow_up": "catalog/fact/activation_rate"}},
	}}
	store := searchtest.NewVectorStore()
	embedder := &searchtest.Embedder{Vector: []float64{1, 0, 0}, AvailableValue: true}
	eng := newTestEngine(store, embedder, src)

	_, apply, err := eng.Reconciler().RunOnce(ctx)
	require.NoError(t, err)
	require.Empty(t, apply.Errors)
	require.Len(t, apply.Collections, 1)
	require.Equal(t, 2, apply.Collections[0].Upserted)

	count, err := store.CountPoints(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	// A warm reconcile over unchanged content plans no work.
	plan, _, err := eng.Reconciler().RunOnce(ctx)
	require.NoError(t, err)
	require.False(t, plan.HasWork(), "unchanged corpus must not re-embed")

	// Removing a source deletes its ghost point.
	src.docs = src.docs[:1]
	_, apply, err = eng.Reconciler().RunOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, apply.Collections[0].Deleted)
	count, err = store.CountPoints(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestEngineSearchProjectsTypedHits(t *testing.T) {
	ctx := context.Background()
	store := searchtest.NewVectorStore()
	store.QueryResults = []pkg.SearchResult{
		{ID: "point-1", Score: 0.91, Payload: map[string]any{
			"id":         "node:offer-1",
			"kind":       "node-offer",
			"title":      "Seed Offer",
			"snippet":    "seed offer — status idea",
			"follow_up":  "catalog/node/offer-1",
			"freshness":  FreshnessFresh,
			"historical": false,
		}},
	}
	embedder := &searchtest.Embedder{Vector: []float64{1, 0, 0}, AvailableValue: true}
	eng := newTestEngine(store, embedder, &fakeSource{})

	hits, method, err := eng.Search(ctx, "seed offer", 10)
	require.NoError(t, err)
	require.NotEmpty(t, method)
	require.Len(t, hits, 1)
	require.Equal(t, "node:offer-1", hits[0].ID)
	require.Equal(t, "Seed Offer", hits[0].Title)
	require.Equal(t, "catalog/node/offer-1", hits[0].FollowUp)
	require.Equal(t, FreshnessFresh, hits[0].Freshness)
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

func TestEngineEnsureCollectionUsesResolvedSpec(t *testing.T) {
	store := searchtest.NewVectorStore()
	eng := newTestEngine(store, &searchtest.Embedder{Vector: []float64{1, 0, 0}}, &fakeSource{})
	require.NoError(t, eng.EnsureCollection(context.Background()))
	require.Equal(t, Collection, eng.Spec().Name)
}
