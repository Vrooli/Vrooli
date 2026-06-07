package aisearch

import (
	"strings"

	pkg "github.com/vrooli/aisearch-go"
)

// search.go is KO's documentation read path. The query -> rerank -> floor ->
// project -> fallback ORCHESTRATION now lives in the shared engine
// (pkg.Service); this file supplies only the doc-specific shaping the engine is
// parameterized over: the scope/facet payload filter, the client-side path-scope
// trim, the authority boost, the federation projection, and the reranker
// contextual text. NewSearchService wires those seams onto a pkg.Service over
// the vrooli-docs collection.

const (
	// defaultSearchLimit / maxSearchLimit bound the returned result count.
	defaultSearchLimit = 10
	maxSearchLimit     = 100
	// docShortlist is the over-fetch depth: the doc read path always widens the
	// vector shortlist (the engine over-fetches because a PostFilter + Decorator
	// run before the page is cut) so path-scope trimming and the authority boost
	// have headroom, and the reranker (when enabled) reorders a real pool.
	docShortlist = 60
	// prefetchLimit is the per-leg (dense/sparse) shortlist before RRF fusion.
	prefetchLimit = 60
	// snippetLen bounds the projected snippet length.
	snippetLen = 280
	// rerankCandidateLen bounds the contextual text handed to a reranker.
	rerankCandidateLen = 900

	// authorityActiveBoost / authorityCanonicalBoost nudge ranking toward
	// canonical, maintained docs. Applied as a scale-relative multiplier to the
	// final (post-rerank) score so the signal survives both ~0.01 RRF scores and
	// ~0..1 rerank scores without dominating.
	authorityActiveBoost    = 0.10
	authorityCanonicalBoost = 0.15
	// authorityProjectBoost favors root /docs over scenario docs for corpus-wide
	// queries (a scenario's same-named page should not outrank the project answer).
	authorityProjectBoost = 0.12
)

// TextFallback is the offline-safe keyword leg (the repurposed docsearch grep).
// Its signature matches pkg.TextFallbackFunc so it plugs straight into the
// shared Service. NewDocsearchFallback adapts the concrete docsearch.Service.
type TextFallback = pkg.TextFallbackFunc

// ServiceOptions configures KO's documentation search. Embedder/VectorStore are
// required for the vector legs; Sparse defaults to the BM25 encoder (matching
// the indexer); RerankEnabled gates the rerank pass and Reranker supplies the
// degradation chain (cross-encoder -> llm -> fused order); TextFallback provides
// the grep degradation; Reconciler feeds Status's last-reconcile fields.
type ServiceOptions struct {
	Embedder    pkg.Embedder
	Sparse      pkg.SparseEncoder
	VectorStore pkg.VectorStore
	// RerankEnabled is the explicit rerank-on/off lever (the shared convention —
	// matches cli-health — instead of inferring it from a nil Reranker). The
	// caller wires it from KO_DOCS_RERANK_ENABLED; see server.go.
	RerankEnabled bool
	Reranker      *pkg.RerankerChain
	TextFallback  TextFallback
	Reconciler    *pkg.Reconciler
}

// NewSearchService wires the documentation read path onto the shared engine.
// Reranking is gated by the explicit RerankEnabled flag (the rerank-on/off
// decision is the caller's — see KO_DOCS_RERANK_ENABLED in server.go), default
// OFF for this corpus. The relevance floor is ON and correct: the doc corpus runs
// RRF-fused hybrid, and the shared Service now classifies that rerank-off fused
// leg into the fusion floor band (relative MaxGap only, the absolute cosine
// HardFloor that would have annihilated real fused hits is disabled), so the
// floor trims only the far tail. Ranking quality still comes from fusion + the
// authority boost (+ rerank when enabled).
func NewSearchService(opts ServiceOptions) *pkg.Service {
	sparse := opts.Sparse
	if sparse == nil {
		sparse = pkg.NewBM25SparseEncoder()
	}
	return pkg.NewService(pkg.ServiceOptions{
		Embedder:      opts.Embedder,
		SparseEncoder: sparse,
		VectorStore:   opts.VectorStore,
		Reranker:      opts.Reranker,
		Reconciler:    opts.Reconciler,
		RerankEnabled: opts.RerankEnabled,
		ApplyFloor:    true,
		Shortlist:     docShortlist,
		PrefetchLimit: prefetchLimit,
		DefaultLimit:  defaultSearchLimit,
		MaxLimit:      maxSearchLimit,
		Project:       docProject,
		Filter:        func(q pkg.SearchQuery) *pkg.QueryFilter { return docFilter(q.Scope, q.Facets) },
		PostFilter: func(hits []pkg.SearchResult, q pkg.SearchQuery) []pkg.SearchResult {
			return filterPathScope(hits, q.Scope)
		},
		Decorate:     applyAuthorityBoost,
		RerankText:   rerankText,
		TextFallback: opts.TextFallback,
	})
}

// NewDefaultReranker builds the production degradation chain: the cross-encoder
// TEI resource first, the always-available LLM reranker second. The chain
// returns the fused order when neither is reachable.
func NewDefaultReranker() *pkg.RerankerChain {
	return pkg.NewRerankerChain(pkg.NewCrossEncoderReranker("", ""), pkg.NewLLMReranker(""))
}

// docProject maps a raw vector-store result into the federation hit shape,
// pulling the retrievable body, paths, and source id from the payload. The first
// fields (id, relative_path, score, snippet, path) are the search-hub federation
// contract for the KO `doc` leaf — keep them stable (search_surface.go's
// docSearchHit serializes them).
func docProject(r pkg.SearchResult) pkg.SearchResult {
	rel := payloadString(r.Payload, MetaRelativePath)
	path := payloadString(r.Payload, MetaPath)
	if path == "" {
		path = rel
	}
	r.RelativePath = rel
	r.Path = path
	r.Snippet = snippet(payloadString(r.Payload, "body"))
	r.SourceID = payloadString(r.Payload, "source_id")
	return r
}

// docFilter builds the Qdrant payload filter from scope + facets. Path scope is
// filtered server-side via the path_prefixes payload array (an exact segment
// match) so in-prefix docs are retrieved even when they fall outside the global
// shortlist; filterPathScope still applies an exact client-side trim afterward.
func docFilter(scope pkg.Scope, facets pkg.Facets) *pkg.QueryFilter {
	var must []pkg.FieldMatch
	if scope.Kind == pkg.ScopeScenario && strings.TrimSpace(scope.Value) != "" {
		must = append(must, pkg.FieldMatch{Key: MetaScenario, Value: scope.Value})
	}
	if scope.Kind == pkg.ScopePath && strings.TrimSpace(scope.Value) != "" {
		prefix := strings.Trim(strings.TrimSpace(scope.Value), "/")
		must = append(must, pkg.FieldMatch{Key: MetaPathPrefixes, Value: prefix})
	}
	addAny := func(key string, vals []string) {
		if len(vals) == 0 {
			return
		}
		anyOf := make([]any, 0, len(vals))
		for _, v := range vals {
			if v = strings.TrimSpace(v); v != "" {
				anyOf = append(anyOf, v)
			}
		}
		if len(anyOf) > 0 {
			must = append(must, pkg.FieldMatch{Key: key, AnyOf: anyOf})
		}
	}
	addAny(MetaDocType, facets.DocType)
	addAny(MetaAudience, facets.Audience)
	addAny(MetaMaturity, facets.Maturity)
	addAny(MetaCanonicalFor, facets.CanonicalFor)
	if len(must) == 0 {
		return nil
	}
	return &pkg.QueryFilter{Must: must}
}

// filterPathScope keeps only hits whose relative path is under the prefix when
// the query uses path scope. A no-op for other scopes.
func filterPathScope(hits []pkg.SearchResult, scope pkg.Scope) []pkg.SearchResult {
	if scope.Kind != pkg.ScopePath || strings.TrimSpace(scope.Value) == "" {
		return hits
	}
	prefix := strings.TrimPrefix(strings.TrimSpace(scope.Value), "/")
	out := hits[:0]
	for _, h := range hits {
		if strings.HasPrefix(h.RelativePath, prefix) || strings.HasPrefix(h.Path, prefix) {
			out = append(out, h)
		}
	}
	return out
}

// applyAuthorityBoost nudges the fused score toward canonical, maintained docs.
// It is purely facet-driven (maturity/canonicalFor/scope) so it ignores the
// query the shared seam now passes.
func applyAuthorityBoost(hits []pkg.SearchResult, _ pkg.SearchQuery) {
	for i := range hits {
		factor := 1.0
		if maturity, _ := hits[i].Payload[MetaMaturity].(string); strings.EqualFold(maturity, "active") {
			factor += authorityActiveBoost
		}
		if hasAny(hits[i].Payload[MetaCanonicalFor]) {
			factor += authorityCanonicalBoost
		}
		if scope, _ := hits[i].Payload[MetaScope].(string); scope == ScopeProject {
			factor += authorityProjectBoost
		}
		hits[i].Score *= factor
	}
}

func hasAny(v any) bool {
	switch t := v.(type) {
	case []any:
		return len(t) > 0
	case []string:
		return len(t) > 0
	case string:
		return strings.TrimSpace(t) != ""
	default:
		return false
	}
}

// rerankText composes the contextual text a reranker scores: title, heading
// path, then the chunk body (bounded). Mirrors the indexing-time contextual
// prefix so the reranker sees the same self-contained context.
func rerankText(h pkg.SearchResult) string {
	var b strings.Builder
	if title := payloadString(h.Payload, MetaTitle); title != "" {
		b.WriteString(title)
		b.WriteString("\n")
	}
	if hp := payloadString(h.Payload, MetaHeadingPath); hp != "" {
		b.WriteString(hp)
		b.WriteString("\n")
	}
	body := payloadString(h.Payload, "body")
	if body == "" {
		body = h.Snippet
	}
	b.WriteString(body)
	text := b.String()
	if len(text) > rerankCandidateLen {
		text = text[:rerankCandidateLen]
	}
	return text
}

func snippet(body string) string {
	body = strings.TrimSpace(body)
	if len(body) <= snippetLen {
		return body
	}
	return strings.TrimSpace(body[:snippetLen]) + "…"
}

func payloadString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	if v, ok := payload[key].(string); ok {
		return v
	}
	return ""
}
