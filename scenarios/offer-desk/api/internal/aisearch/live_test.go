package aisearch

import (
	"log"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	pkg "github.com/vrooli/ai-go/search"
	"github.com/vrooli/ai-go/search/searchtest"
	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
)

func TestStartDegradesToLexicalWhenProviderMissing(t *testing.T) {
	store, source, ctx := testSource(t)
	_, err := store.CreateNode(ctx, offerspb.NodeKind_OFFER, "Aquila launch offer", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)

	live := Start(ctx, source, filepath.Join(t.TempDir(), "missing.json"), log.Default())
	require.False(t, live.EngineAvailable(), "a missing provider must not fail boot")

	resp, err := live.Search(ctx, "Aquila", 10)
	require.NoError(t, err)
	require.NotEmpty(t, resp.Hits)
}

func TestLiveSearchServesLexicalWithoutEngine(t *testing.T) {
	store, source, ctx := testSource(t)
	_, err := store.CreateNode(ctx, offerspb.NodeKind_OFFER, "Aquila launch offer", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)

	live := NewLexicalSearch(NewService(source))
	require.False(t, live.EngineAvailable())

	resp, err := live.Search(ctx, "Aquila", 10)
	require.NoError(t, err)
	require.NotEmpty(t, resp.Hits)
	require.NotEmpty(t, resp.Generation)
}

func TestLiveSearchProjectsEngineHitsWithCatalogGeneration(t *testing.T) {
	store, source, ctx := testSource(t)
	_, err := store.CreateNode(ctx, offerspb.NodeKind_OFFER, "Aquila launch offer", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)

	vecStore := searchtest.NewVectorStore()
	vecStore.QueryResults = []pkg.SearchResult{{
		ID:    "point-1",
		Score: 0.9,
		Payload: map[string]any{
			"id": "node:abc", "kind": "node-offer", "title": "Ranked Offer",
			"snippet": "s", "follow_up": "catalog/node/abc",
			"freshness": FreshnessFresh, "historical": false,
		},
	}}
	eng := newTestEngine(vecStore, &searchtest.Embedder{Vector: []float64{1, 0, 0}, AvailableValue: true}, &fakeSource{})

	live := NewLiveSearch(eng, NewService(source))
	require.True(t, live.EngineAvailable())

	resp, err := live.Search(ctx, "aquila", 10)
	require.NoError(t, err)
	require.Len(t, resp.Hits, 1)
	require.Equal(t, "node:abc", resp.Hits[0].ID)
	require.Equal(t, "Ranked Offer", resp.Hits[0].Title)
	require.NotEmpty(t, resp.Generation, "engine hits still carry the authoritative catalog generation")
	require.NotEmpty(t, resp.MaterializedAt)
}

func TestApplyIndexTimeUsesReconcileOverMaterialization(t *testing.T) {
	status := StatusReport{LastIndexedAt: time.Date(2026, time.September, 9, 1, 50, 0, 0, time.UTC)}

	got := applyIndexTime(status, "2026-09-12T23:40:00Z")
	require.Equal(t, time.Date(2026, time.September, 12, 23, 40, 0, 0, time.UTC), got.LastIndexedAt,
		"the registered index timestamp is the last successful reconcile, not source age")
}

func TestApplyIndexTimeFallsBackWhenReconcileUnavailable(t *testing.T) {
	materialized := time.Date(2026, time.September, 9, 1, 50, 0, 0, time.UTC)
	status := StatusReport{LastIndexedAt: materialized}

	require.Equal(t, materialized, applyIndexTime(status, "").LastIndexedAt,
		"a boot without a completed reconcile must not fabricate an index time")
	require.Equal(t, materialized, applyIndexTime(status, "not-a-time").LastIndexedAt,
		"an unparsable reconcile time must not fabricate an index time")
}

func TestEngineTextFallbackServesLexicalRecords(t *testing.T) {
	store, source, ctx := testSource(t)
	_, err := store.CreateNode(ctx, offerspb.NodeKind_OFFER, "Aquila launch offer", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)

	lexical := NewService(source)
	vecStore := searchtest.NewVectorStore()
	vecStore.AvailableValue = false
	eng := NewEngineFromComponents(
		EngineComponents{
			Embedder:    &searchtest.Embedder{Vector: []float64{1, 0, 0}, AvailableValue: false},
			VectorStore: vecStore,
			Spec: pkg.CollectionSpec{
				Name: Collection, DenseSize: 3, DenseDistance: pkg.DefaultDenseDistance, Model: "test-model",
			},
		},
		NewCatalogSource(source),
		pkg.TuningConfig{Engine: pkg.EngineDense}.WithDefaults(),
		textFallbackOf(lexical),
		2,
		0,
	)

	live := NewLiveSearch(eng, lexical)
	resp, err := live.Search(ctx, "Aquila", 10)
	require.NoError(t, err)
	require.NotEmpty(t, resp.Hits, "a down vector backend degrades to the lexical leg, not an empty page")

	var found bool
	for _, hit := range resp.Hits {
		if hit.ID != "" && hit.FollowUp != "" && hit.Title == "Aquila launch offer" {
			found = true
		}
	}
	require.True(t, found, "the lexical fallback hit survives the engine's typed projection")
}
