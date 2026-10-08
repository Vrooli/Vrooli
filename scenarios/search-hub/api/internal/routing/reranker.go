package routing

import (
	"context"
	"sort"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

// Reranker fuses the per-provider candidate shortlist into one comparable,
// cross-provider ranked list. It is the merge step the plan (§4) calls out as
// the answer to "raw provider scores aren't comparable": rather than falsely
// interleaving heterogeneous per-provider scores, the reranker re-scores every
// candidate against the query on one scale so they can be ordered together.
//
// Like the Classifier it is a swappable seam (production: local Ollama
// qwen3:4b as an LLM-as-reranker via OllamaReranker; tests: a deterministic
// fake) so the router stays unit-testable without a model, and so the impl can
// be swapped for a true cross-encoder (bge-reranker-v2-m3) when that resource
// lands — with no change to the router contract (plan Appendix A.3).
type Reranker interface {
	// Rerank scores each candidate against query and returns them as one
	// unified list ordered most-relevant first, with each hit's RerankScore
	// set. An error means the reranker could not produce an ordering (model
	// down, unparseable output); the router treats that as graceful degradation
	// — it keeps the honest by-provider grouping and flags the response degraded
	// rather than failing the query (plan §Phase 6: mandatory degradation).
	Rerank(ctx context.Context, query string, candidates []*routingv1.SearchHit) ([]*routingv1.SearchHit, error)
	// Available reports whether the backing model is reachable. The hot query
	// path does not call this (it relies on Rerank's error for degradation); it
	// exists for the Phase 7 Status surface.
	Available(ctx context.Context) bool
}

// fuseGroups flattens the by-provider grouping into one candidate shortlist for
// the reranker. Degraded groups carry no hits, so they contribute nothing; the
// order here is irrelevant because the reranker reorders by relevance.
func fuseGroups(groups []*routingv1.ProviderResultGroup) []*routingv1.SearchHit {
	out := make([]*routingv1.SearchHit, 0, len(groups))
	for _, g := range groups {
		out = append(out, g.GetHits()...)
	}
	return out
}

// applyRerank assigns the model's per-candidate scores (indexed positionally,
// already normalized to [0,1]) onto a copy of the candidate slice and returns
// it sorted most-relevant first. The underlying SearchHit pointers are shared
// with the groups, so a hit's RerankScore is visible in both the unified list
// and its provenance group — one fact, two views.
//
// Sort key: rerank_score desc, then the original per-provider normalized Score
// desc as a tie-breaker. The secondary key matters because a small LLM-as-
// reranker often hedges, scoring many homogeneous candidates identically (e.g.
// every "restart"-ish command at 5/10); without it those ties would resolve to
// arbitrary fan-out order, throwing away the providers' own retrieval signal.
// Breaking ties by the original score keeps the order sensible even when the
// reranker is uninformative. The final stable sort makes output deterministic.
func applyRerank(candidates []*routingv1.SearchHit, scores []float64) []*routingv1.SearchHit {
	ranked := make([]*routingv1.SearchHit, len(candidates))
	copy(ranked, candidates)
	for i, h := range ranked {
		if i < len(scores) {
			h.RerankScore = scores[i]
		}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		ri, rj := ranked[i].GetRerankScore(), ranked[j].GetRerankScore()
		if ri != rj {
			return ri > rj
		}
		return ranked[i].GetScore() > ranked[j].GetScore()
	})
	return ranked
}

// normalizeRerankScore clamps a raw 0–10 pointwise relevance score and maps it
// to [0,1], so RerankScore sits on the same band as the pre-rerank, per-provider
// normalized Score (the descriptor's ScoreScale already normalizes that to
// [0,1]). Keeping both on one scale makes the unified list honestly comparable.
func normalizeRerankScore(s float64) float64 {
	switch {
	case s < 0:
		s = 0
	case s > 10:
		s = 10
	}
	return s / 10.0
}
