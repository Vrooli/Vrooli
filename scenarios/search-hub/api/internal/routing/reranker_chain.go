package routing

import (
	"context"
	"fmt"
	"sort"
	"strings"

	aisearch "github.com/vrooli/ai-go/search"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

// SharedReranker adapts the shared ai-go/search reranker chain to Search Hub's
// local routing.Reranker seam. Search Hub owns the fusion semantics for
// provider hits; the shared package owns TEI and Ollama client logic.
type SharedReranker struct {
	chain aisearch.Reranker
}

var _ Reranker = (*SharedReranker)(nil)

// NewDefaultRerankerChain builds Search Hub's production reranker:
// TEI cross-encoder first, Ollama LLM fallback second.
func NewDefaultRerankerChain() *SharedReranker {
	return NewSharedReranker(aisearch.NewRerankerChain(
		aisearch.NewCrossEncoderReranker("", ""),
		aisearch.NewLLMReranker(""),
	))
}

// NewSharedReranker adapts a shared reranker implementation. Tests inject a
// fake aisearch.Reranker; production passes *aisearch.RerankerChain.
func NewSharedReranker(chain aisearch.Reranker) *SharedReranker {
	return &SharedReranker{chain: chain}
}

// Rerank converts Search Hub hits to shared rerank candidates, delegates to the
// active shared reranker leg, and applies returned scores back onto the hits.
func (r *SharedReranker) Rerank(ctx context.Context, query string, candidates []*routingv1.SearchHit) ([]*routingv1.SearchHit, error) {
	if r == nil || r.chain == nil {
		return nil, fmt.Errorf("reranker chain unavailable")
	}
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("empty query")
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no candidates to rerank")
	}
	shared := make([]aisearch.RerankCandidate, len(candidates))
	for i, h := range candidates {
		shared[i] = aisearch.RerankCandidate{
			ID:   rerankCandidateKey(i, h.GetId()),
			Text: rerankCandidateText(h),
		}
	}
	scores, err := r.chain.Rerank(ctx, query, shared)
	if err != nil {
		scores, err = r.retryAfterActiveFailure(ctx, query, shared, err)
		if err != nil {
			return nil, err
		}
	}
	if len(scores) == 0 {
		return nil, fmt.Errorf("reranker chain unavailable")
	}
	return applySharedRerank(candidates, scores), nil
}

// Available reports whether any shared chain leg is reachable.
func (r *SharedReranker) Available(ctx context.Context) bool {
	return r != nil && r.chain != nil && r.chain.Available(ctx)
}

// ActiveName returns the shared chain's active leg name when exposed.
func (r *SharedReranker) ActiveName(ctx context.Context) string {
	if r == nil || r.chain == nil {
		return "none"
	}
	if named, ok := r.chain.(interface{ ActiveName(context.Context) string }); ok {
		return named.ActiveName(ctx)
	}
	if r.chain.Available(ctx) {
		return r.chain.Name()
	}
	return "none"
}

func (r *SharedReranker) retryAfterActiveFailure(ctx context.Context, query string, candidates []aisearch.RerankCandidate, firstErr error) ([]aisearch.RerankScore, error) {
	refreshable, ok := r.chain.(interface {
		ActiveName(context.Context) string
		ActiveUncached(context.Context) aisearch.Reranker
	})
	if !ok {
		return nil, firstErr
	}
	before := refreshable.ActiveName(ctx)
	active := refreshable.ActiveUncached(ctx)
	if active == nil {
		return nil, fmt.Errorf("%w; no fallback reranker available", firstErr)
	}
	after := active.Name()
	if after == "" || after == "none" || after == before {
		return nil, fmt.Errorf("%w; no alternate fallback reranker available", firstErr)
	}
	scores, err := r.chain.Rerank(ctx, query, candidates)
	if err != nil {
		return nil, fmt.Errorf("%w; fallback %s failed: %v", firstErr, after, err)
	}
	return scores, nil
}

func rerankCandidateText(h *routingv1.SearchHit) string {
	var parts []string
	if typ := strings.TrimSpace(h.GetType()); typ != "" {
		parts = append(parts, "type: "+typ)
	}
	title := strings.TrimSpace(h.GetTitle())
	if title == "" {
		title = strings.TrimSpace(h.GetId())
	}
	if title != "" {
		parts = append(parts, "title: "+title)
	}
	if path := strings.TrimSpace(h.GetPath()); path != "" && path != title {
		parts = append(parts, "path: "+path)
	}
	if snippet := truncateForErr(h.GetSnippet()); snippet != "" {
		parts = append(parts, "snippet: "+snippet)
	}
	return strings.Join(parts, "\n")
}

func applySharedRerank(candidates []*routingv1.SearchHit, scores []aisearch.RerankScore) []*routingv1.SearchHit {
	scoreByID := make(map[string]float64, len(scores))
	scoredByID := make(map[string]bool, len(scores))
	for _, s := range scores {
		scoreByID[s.ID] = s.Score
		scoredByID[s.ID] = true
	}
	type rankedHit struct {
		hit    *routingv1.SearchHit
		score  float64
		scored bool
		order  int
	}
	ranked := make([]rankedHit, len(candidates))
	for i, h := range candidates {
		score, scored := scoreByID[rerankCandidateKey(i, h.GetId())]
		if scored {
			h.RerankScore = score
		}
		ranked[i] = rankedHit{hit: h, score: score, scored: scored, order: i}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].scored != ranked[j].scored {
			return ranked[i].scored
		}
		if ranked[i].scored && ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		if ranked[i].hit.GetScore() != ranked[j].hit.GetScore() {
			return ranked[i].hit.GetScore() > ranked[j].hit.GetScore()
		}
		return ranked[i].order < ranked[j].order
	})
	out := make([]*routingv1.SearchHit, len(ranked))
	for i, item := range ranked {
		out[i] = item.hit
	}
	return out
}

// rerankCandidateKey is intentionally positional. A provider may return
// several passages whose document id is the same, and the reranker score is a
// judgment for one candidate passage, not for the document-id string. The
// position makes the adapter's contract one-to-one without changing the
// shared ai-go/search score type or leaking provider-specific identity into
// that package.
func rerankCandidateKey(index int, id string) string {
	return fmt.Sprintf("candidate:%d:%s", index, id)
}
