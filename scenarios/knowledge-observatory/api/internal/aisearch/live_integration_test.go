package aisearch

import (
	"context"
	"os"
	"testing"
	"time"

	pkg "github.com/vrooli/aisearch-go"
)

// TestLiveReindexAndHybridQuery indexes a bounded slice of the real repo into a
// live Qdrant via a live Ollama embedder, then runs a hybrid (dense+sparse RRF)
// query and asserts it returns real documentation with non-empty content — the
// exact failure mode the baseline (§1) exhibited (junk hits, content:"").
//
// It is gated on KO_AISEARCH_LIVE so it never runs in CI without infra:
//
//	KO_AISEARCH_LIVE=1 \
//	KO_LIVE_SCENARIOS_ROOT=/abs/path/to/Vrooli/scenarios \
//	go test ./internal/aisearch/ -run TestLiveReindexAndHybridQuery -v
//
// Optional: QDRANT_URL (default http://127.0.0.1:6333),
// KO_LIVE_COLLECTION (default vrooli-docs), KO_LIVE_BUDGET (default 80 embeds).
func TestLiveReindexAndHybridQuery(t *testing.T) {
	if os.Getenv("KO_AISEARCH_LIVE") == "" {
		t.Skip("set KO_AISEARCH_LIVE=1 (and KO_LIVE_SCENARIOS_ROOT) to run the live reindex proof")
	}
	scenariosRoot := os.Getenv("KO_LIVE_SCENARIOS_ROOT")
	if scenariosRoot == "" {
		t.Fatal("KO_LIVE_SCENARIOS_ROOT must point at the repo's scenarios/ directory")
	}
	qdrantURL := os.Getenv("QDRANT_URL")
	if qdrantURL == "" {
		qdrantURL = pkg.DefaultQdrantURL
	}
	collection := os.Getenv("KO_LIVE_COLLECTION")
	if collection == "" {
		collection = DefaultCollection
	}
	budget := 80

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	embedder := pkg.NewEmbedder(pkg.DefaultEmbedModel)
	if !embedder.Available(ctx) {
		t.Fatal("ollama embedder unavailable; start the ollama resource first")
	}
	store := pkg.NewVectorStore(qdrantURL, os.Getenv("QDRANT_API_KEY"), collection)
	if !store.Available(ctx) {
		t.Fatal("qdrant unavailable; start the qdrant resource first")
	}

	idx, err := NewIndexer(Options{
		Embedder:         embedder,
		VectorStore:      store,
		ScenariosRoot:    scenariosRoot,
		MaxEmbedsPerTick: budget, // bounded so the proof is fast (resumable in prod)
	})
	if err != nil {
		t.Fatalf("NewIndexer: %v", err)
	}
	if err := idx.EnsureCollection(ctx); err != nil {
		t.Fatalf("EnsureCollection: %v", err)
	}

	res, err := idx.Reindex(ctx, false)
	if err != nil {
		t.Fatalf("Reindex: %v", err)
	}
	t.Logf("reindex: planned=%d upserted=%d deferred=%d errors=%d", res.Planned, res.Upserted, res.Deferred, len(res.Errors))
	for _, e := range res.Errors {
		t.Logf("  reindex error: %s", e)
	}
	if res.Upserted == 0 {
		t.Fatal("reindex upserted nothing")
	}

	count, err := store.CountPoints(ctx)
	if err != nil || count == 0 {
		t.Fatalf("expected indexed points, got count=%d err=%v", count, err)
	}
	t.Logf("collection %q now holds %d points", collection, count)

	// Hybrid query: dense (semantic) + sparse (BM25) fused server-side via RRF.
	query := "overall platform architecture and operating model"
	dense, err := embedder.Embed(ctx, query)
	if err != nil {
		t.Fatalf("embed query: %v", err)
	}
	sparse := pkg.NewBM25SparseEncoder().Encode(query)
	hits, err := store.Query(ctx, pkg.HybridQuery{
		Dense:         dense,
		Sparse:        &sparse,
		Fusion:        "rrf",
		Limit:         5,
		PrefetchLimit: 50,
	})
	if err != nil {
		t.Fatalf("hybrid query: %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("hybrid query returned no hits")
	}
	for i, h := range hits {
		body, _ := h.Payload["body"].(string)
		rel, _ := h.Payload[MetaRelativePath].(string)
		hp, _ := h.Payload[MetaHeadingPath].(string)
		t.Logf("  #%d score=%.4f %s [%s] body=%dB", i+1, h.Score, rel, hp, len(body))
		if body == "" {
			t.Errorf("hit %d has empty body payload (the legacy content:\"\" defect)", i+1)
		}
		if rel == "" {
			t.Errorf("hit %d has empty relative_path (federation contract field)", i+1)
		}
	}
}
