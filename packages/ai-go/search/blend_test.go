package aisearch

import (
	"context"
	"testing"
)

// TestApplyRerankRRFDoesNotBuryStrongRetrieval is the core property the blend
// exists for: a result the reranker scores LOW but retrieval ranked HIGH must
// not be buried to the bottom (the pure-reorder failure mode). Retrieval order
// is [A,B,C] (A best); the reranker prefers B>C>A.
func TestApplyRerankRRFDoesNotBuryStrongRetrieval(t *testing.T) {
	hits := []SearchResult{
		{ID: "A", Score: 0.9},
		{ID: "B", Score: 0.5},
		{ID: "C", Score: 0.4},
	}
	scores := []RerankScore{{ID: "A", Score: 0.10}, {ID: "B", Score: 0.99}, {ID: "C", Score: 0.50}}

	pure := ApplyRerank(cloneResults(hits), scores)
	if pure[len(pure)-1].ID != "A" {
		t.Fatalf("precondition: pure rerank should bury A last, got %v", blendIDs(pure))
	}

	blended := ApplyRerankRRF(cloneResults(hits), scores, DefaultRRFK)
	if blended[0].ID != "B" {
		t.Errorf("blend should keep the reranker's top (B) first, got %v", blendIDs(blended))
	}
	// A (retrieval rank 0, rerank rank 2) must outrank C (retrieval 2, rerank 1).
	posA, posC := indexOf(blended, "A"), indexOf(blended, "C")
	if posA >= posC {
		t.Errorf("blend should rescue strongly-retrieved A above C, got %v", blendIDs(blended))
	}
}

func TestApplyRerankRRFEmptyScoresKeepsOrder(t *testing.T) {
	hits := []SearchResult{{ID: "A"}, {ID: "B"}}
	out := ApplyRerankRRF(hits, nil, DefaultRRFK)
	if out[0].ID != "A" || out[1].ID != "B" {
		t.Errorf("no scores => retrieval order preserved, got %v", blendIDs(out))
	}
}

func TestBuildChunkPayloadRecipeAffectsHash(t *testing.T) {
	chunk := Chunk{SourceID: "s", Index: 0, Body: "vrooli scenario list", Meta: map[string]any{"k": "v"}}

	base := buildChunkPayload(chunk, "vrooli scenario list", "src", 1, "")
	again := buildChunkPayload(chunk, "vrooli scenario list", "src", 1, "")
	withRecipe := buildChunkPayload(chunk, "vrooli scenario list", "src", 1, "model=nomic|q=search_query: |d=search_document: ")

	if base[payloadHashKey] != again[payloadHashKey] {
		t.Fatal("empty-recipe hash must be stable across calls")
	}
	if base[payloadHashKey] == withRecipe[payloadHashKey] {
		t.Error("a non-empty recipe must change the drift hash (forces re-embed on prefix change)")
	}
	// Back-compat: empty recipe == the legacy composePayloadHash over the text.
	if base[payloadHashKey] != composePayloadHash("vrooli scenario list", base) {
		t.Error("empty recipe must reproduce the legacy hash exactly")
	}
}

func TestEmbedRecipeEmptyWhenSymmetric(t *testing.T) {
	sym := &cliEmbedder{model: fixtureEmbeddingModel} // no prefixes
	if got := embedderRecipe(sym); got != "" {
		t.Errorf("symmetric embedder recipe must be empty, got %q", got)
	}
	if r := embedderRecipe(NewEmbedder(fixtureEmbeddingModel)); r != "" {
		t.Errorf("default NewEmbedder is symmetric => empty recipe, got %q", r)
	}
	prefixed := NewEmbedderForConfig(Config{EmbedModel: fixtureNomicEmbeddingModel, EmbedTaskPrefix: true}).(RecipeEmbedder)
	if prefixed.EmbedRecipe() == "" {
		t.Error("prefixed embedder must report a non-empty recipe")
	}
}

// helpers
func cloneResults(in []SearchResult) []SearchResult {
	out := make([]SearchResult, len(in))
	copy(out, in)
	return out
}

func blendIDs(rs []SearchResult) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.ID
	}
	return out
}

func indexOf(rs []SearchResult, id string) int {
	for i, r := range rs {
		if r.ID == id {
			return i
		}
	}
	return -1
}

// TestServiceBlendLabelsWeakFromRawRerankScores pins the blend-path weak
// label: a blended hit's SCORE is a rank-fusion magnitude (~2/(K+1), far
// below every absolute weak band — on a tiny corpus junk blends to within
// epsilon of the top hit), so weakness must be judged from the reranker's
// RAW calibrated scores in the leg's own regime. The regression this guards:
// every blended hit (including near-exact matches) rendered "(weak)" because
// the blended magnitude was compared against an absolute threshold.
func TestServiceBlendLabelsWeakFromRawRerankScores(t *testing.T) {
	store := &queryStore{available: true, results: []SearchResult{
		docResult("go-version", 0.80, "current stable Go version is 1.26"),
		docResult("eiffel", 0.55, "the Eiffel Tower is in Paris"),
	}}
	// Cross-encoder raw scores: the on-topic hit is high, the junk hit ~0.
	rr := NewRerankerChain(&stubReranker{
		name:      "cross-encoder:test",
		available: true,
		scores:    []RerankScore{{ID: "go-version", Score: 0.92}, {ID: "eiffel", Score: 0.02}},
	})
	svc := NewService(ServiceOptions{
		Embedder:      &countingEmbedder{},
		VectorStore:   store,
		Reranker:      rr,
		RerankEnabled: true,
		RerankBlend:   true,
		Project:       docProjector,
		Shortlist:     50,
	})

	resp, err := svc.Search(context.Background(), SearchQuery{Query: "latest Go version", Mode: ModeDense, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Reranker != "blend:cross-encoder:test" {
		t.Fatalf("reranker leg = %q, want blend:cross-encoder:test", resp.Reranker)
	}
	// The engine must resolve the blended path to the fusion regime (its scores
	// are RRF rank-fusion, not cross-encoder sigmoids) and expose it on the
	// response, so adopters never re-derive from the "blend:" leg name.
	if resp.Regime != "fused" {
		t.Fatalf("blend regime = %q, want fused", resp.Regime)
	}
	byID := map[string]SearchResult{}
	for _, h := range resp.Results {
		byID[h.ID] = h
	}
	top, ok := byID["go-version"]
	if !ok {
		t.Fatalf("go-version missing from results: %+v", resp.Results)
	}
	if top.Weak {
		t.Fatalf("near-exact match labeled weak (raw 0.92 vs cross-encoder band): %+v", top)
	}
	junk, ok := byID["eiffel"]
	if ok && !junk.Weak {
		t.Fatalf("junk hit not labeled weak (raw 0.02): %+v", junk)
	}
	// The blended SCORE still owns the order and keeps its rank-fusion
	// magnitude (~2/(K+1)) — only the label semantics changed.
	if top.Score > 1.0/float64(DefaultRRFK)*2+0.001 {
		t.Fatalf("blend score should stay a rank-fusion magnitude, got %f", top.Score)
	}
}

// TestServiceBlendUnscoredHitIsWeak: a shortlist member the reranker declined
// to score gets no vouching — it must be labeled weak rather than inheriting
// strength from its retrieval rank.
func TestServiceBlendUnscoredHitIsWeak(t *testing.T) {
	store := &queryStore{available: true, results: []SearchResult{
		docResult("scored", 0.80, "scored doc"),
		docResult("unscored", 0.75, "unscored doc"),
	}}
	rr := NewRerankerChain(&stubReranker{
		name:      "cross-encoder:test",
		available: true,
		scores:    []RerankScore{{ID: "scored", Score: 0.9}},
	})
	svc := NewService(ServiceOptions{
		Embedder:      &countingEmbedder{},
		VectorStore:   store,
		Reranker:      rr,
		RerankEnabled: true,
		RerankBlend:   true,
		Project:       docProjector,
		Shortlist:     50,
	})

	resp, err := svc.Search(context.Background(), SearchQuery{Query: "doc", Mode: ModeDense, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range resp.Results {
		switch h.ID {
		case "scored":
			if h.Weak {
				t.Fatalf("scored strong hit labeled weak: %+v", h)
			}
		case "unscored":
			if !h.Weak {
				t.Fatalf("unscored hit must be weak: %+v", h)
			}
		}
	}
}
