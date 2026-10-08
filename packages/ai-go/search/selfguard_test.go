package aisearch

import (
	"context"
	"testing"
)

// selfguard_test.go is the OFFLINE retrieval-logic self-guard. Every recall /
// accuracy gate the adopters run is live-gated (needs Ollama + Qdrant) and so
// never runs in `make test`/CI — a regression in the RRF rerank-blend, the
// regime floor, or the regime-aware weak-labeling would pass CI silently. This
// file closes that hole: it drives the production-shaped read-path (Service.Search)
// against a deterministic in-memory vector store + a stub reranker over a tiny
// LABELLED corpus with a known-correct order, and asserts the exact golden ranks.
// It runs under plain `go test ./...` with NO env gating. It complements (does
// not replace) the adopters' live gates: the live gates measure real recall on
// real embeddings; this guard freezes the deterministic pipeline composition so a
// fusion/floor/blend/weak regression fails fast and locally.
//
// To prove it bites: perturb the pipeline (e.g. make ApplyRerankRRF a pure
// reorder, or drop the fusion-regime floor branch) and this test goes red.

// guardDoc builds a labelled corpus point: id is the canonical command, score is
// the retrieval (post-qdrant-fusion) score, body is the embedding/rerank text.
func guardDoc(id string, score float64, body string) SearchResult {
	return SearchResult{ID: id, Score: score, Payload: map[string]any{"body": body, "relative_path": id, "source_id": id}}
}

// TestOfflineSelfGuardCommandCorpus freezes the cli-health command-corpus shape:
// dense retrieval + rerank BLEND on + floor on. The labelled scenario is the
// documented failure mode: the canonical command ("scenario restart") is strongly
// RETRIEVED but a literal-token lookalike ("scenario list") wins the cross-encoder
// on surface tokens, while pure gibberish is present to be rejected.
//
// Golden properties asserted (any pipeline regression breaks at least one):
//   - BLEND rescues the strongly-retrieved canonical above the literal lookalike
//     (a pure reorder would bury it — that is the −0.20-recall failure mode).
//   - The gibberish point (reranker drove it to 0) ranks LAST — junk rejection by
//     rank without an absolute floor (the blend's documented win).
//
// (Weak-labeling is asserted separately in TestOfflineSelfGuardWeakLabelCosine:
// the RRF-fused blend score is ~0.03 scale, below every threshold, so the weak
// flag is uninformative on this path by construction — see WeakThresholdFusion.)
func TestOfflineSelfGuardCommandCorpus(t *testing.T) {
	store := &queryStore{available: true, results: []SearchResult{
		guardDoc("scenario restart", 0.95, "vrooli scenario restart restart a scenario"),
		guardDoc("scenario list", 0.80, "vrooli scenario list list scenarios"),
		guardDoc("scenario logs", 0.55, "vrooli scenario logs tail scenario logs"),
		guardDoc("zzz gibberish", 0.50, "qwerty asdf gibberish nonsense"),
	}}
	// Cross-encoder prefers the literal "list" lookalike on tokens, ranks the
	// canonical "restart" lower, and collapses gibberish to 0 — the exact shape the
	// blend exists to correct.
	rr := NewRerankerChain(&stubReranker{
		name:      "cross-encoder:test",
		available: true,
		scores: []RerankScore{
			{ID: "scenario list", Score: 0.92},
			{ID: "scenario restart", Score: 0.70},
			{ID: "scenario logs", Score: 0.40},
			{ID: "zzz gibberish", Score: 0.0},
		},
	})
	svc := NewService(ServiceOptions{
		Embedder:      &countingEmbedder{},
		VectorStore:   store,
		Reranker:      rr,
		RerankEnabled: true,
		RerankBlend:   true,
		ApplyFloor:    true,
		Project:       docProjector,
		Shortlist:     50,
	})

	resp, err := svc.Search(context.Background(), SearchQuery{Query: "restart a scenario", Mode: ModeDense, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}

	ids := blendIDs(resp.Results)
	// Blend rescues the strongly-retrieved canonical above the literal lookalike.
	posRestart, posList := indexOf(resp.Results, "scenario restart"), indexOf(resp.Results, "scenario list")
	if posRestart == -1 || posList == -1 {
		t.Fatalf("both real commands must survive, got %v", ids)
	}
	if posRestart >= posList {
		t.Fatalf("blend must lift strongly-retrieved 'scenario restart' above literal lookalike 'scenario list', got %v", ids)
	}
	// Junk rejection by rank: the gibberish point (reranker drove it to 0) ranks
	// LAST even though it was retrieved mid-pack.
	posGib := indexOf(resp.Results, "zzz gibberish")
	if posGib != len(resp.Results)-1 {
		t.Fatalf("gibberish must rank last, got %v", ids)
	}
}

// TestOfflineSelfGuardWeakLabelCosine freezes the regime-aware weak-labeling on
// the dense/cosine path, where scores are real 0..1 cosine and the threshold
// cleanly separates strong from weak. A regression that drops the regime split
// (e.g. judging cosine scores on the cross-encoder band, or hard-coding a single
// threshold) flips one of these labels.
func TestOfflineSelfGuardWeakLabelCosine(t *testing.T) {
	store := &queryStore{available: true, results: []SearchResult{
		guardDoc("strong", 0.80, "clear strong match"),
		guardDoc("weak", 0.30, "barely related"), // below the cosine weak threshold
	}}
	svc := NewService(ServiceOptions{
		Embedder:    &countingEmbedder{},
		VectorStore: store,
		ApplyFloor:  false, // isolate the weak label from the floor
		Project:     docProjector,
	})
	resp, err := svc.Search(context.Background(), SearchQuery{Query: "strong", Mode: ModeDense, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("floor off keeps both, got %d", len(resp.Results))
	}
	byID := map[string]bool{}
	for _, r := range resp.Results {
		byID[r.ID] = r.Weak
	}
	if byID["strong"] {
		t.Errorf("0.80 cosine hit must be strong, not weak")
	}
	if !byID["weak"] {
		t.Errorf("0.30 cosine hit must be weak-labeled")
	}
}

// TestOfflineSelfGuardDocFusionRegime freezes the KO doc-corpus shape: hybrid
// retrieval + rerank OFF + floor ON. The fusion-regime floor must keep low RRF
// scores (a rank signal, not 0..1 cosine) that the absolute cosine HardFloor
// would wrongly annihilate, while still cutting the relative tail. This is the
// regression sentinel for the regime-aware floor branch.
func TestOfflineSelfGuardDocFusionRegime(t *testing.T) {
	store := &queryStore{available: true, results: []SearchResult{
		{ID: "top.md", Score: 0.56, Payload: map[string]any{"body": "deployment guide"}},
		{ID: "mid.md", Score: 0.40, Payload: map[string]any{"body": "config reference"}},
		{ID: "low.md", Score: 0.30, Payload: map[string]any{"body": "changelog"}}, // < cosine 0.35 HardFloor
	}}
	svc := NewService(ServiceOptions{
		Embedder:      &countingEmbedder{},
		SparseEncoder: NewBM25SparseEncoder(),
		VectorStore:   store,
		ApplyFloor:    true,
		Project:       docProjector,
	})
	resp, err := svc.Search(context.Background(), SearchQuery{Query: "deploy", Mode: ModeHybrid, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Method != "hybrid" {
		t.Fatalf("method = %q, want hybrid", resp.Method)
	}
	// All three survive: the fusion regime disables the absolute cosine HardFloor
	// (which would have annihilated low.md at 0.30) and the relative gap from 0.56
	// keeps the tail. A regression that floors on the cosine band would drop low.md.
	if len(resp.Results) != 3 {
		t.Fatalf("fusion-regime floor must keep all 3 RRF hits, got %d: %v", len(resp.Results), blendIDs(resp.Results))
	}
	if resp.Results[0].ID != "top.md" {
		t.Fatalf("retrieval order must be preserved (rerank off), got %v", blendIDs(resp.Results))
	}
}
