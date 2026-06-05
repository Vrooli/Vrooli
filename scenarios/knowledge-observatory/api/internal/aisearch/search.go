package aisearch

import (
	"context"
	"fmt"
	"sort"
	"strings"

	pkg "github.com/vrooli/aisearch-go"
)

// search.go is Phase 5 of the KO search cutover: the hybrid (dense+sparse RRF)
// documentation read path with reranking, scope/facet filters, an authority
// boost, and the auto fallback chain (hybrid -> dense -> grep). It implements
// the shared pkg.Service contract over the vrooli-docs collection the Phase-3
// Indexer populates, projecting hits into the search-hub federation shape
// (keys.go).

const (
	// defaultSearchLimit is the result count returned when a query omits Limit.
	defaultSearchLimit = 10
	// maxSearchLimit caps the returned result count.
	maxSearchLimit = 100
	// shortlistMultiplier widens the vector shortlist relative to the final
	// limit so the reranker has more candidates to reorder (two-stage retrieval).
	shortlistMultiplier = 4
	// minShortlist / maxShortlist bound the reranked candidate pool.
	minShortlist = 20
	maxShortlist = 60
	// prefetchLimit is the per-leg (dense/sparse) shortlist before RRF fusion.
	prefetchLimit = 60
	// snippetLen bounds the projected snippet length.
	snippetLen = 280
	// rerankCandidateLen bounds the contextual text handed to a reranker.
	rerankCandidateLen = 900

	// authorityActiveBoost / authorityCanonicalBoost nudge ranking toward
	// canonical, maintained docs (plan §3.3 authority boost). Applied as a
	// scale-relative multiplier to the final (post-rerank) score so the signal
	// survives both ~0.01 RRF scores and ~0..1 rerank scores without dominating.
	authorityActiveBoost    = 0.10
	authorityCanonicalBoost = 0.15
	// authorityProjectBoost favors root /docs (the canonical platform docs) over
	// scenario docs for corpus-wide queries — a scenario's same-named page (e.g.
	// its own QUICKSTART) should not outrank the project-level answer.
	authorityProjectBoost = 0.12
)

// TextFallback is the offline-safe keyword leg (the repurposed docsearch grep).
// It is injected so the search service stays decoupled from KO's filesystem
// search and unit tests need no on-disk corpus. NewDocsearchFallback adapts the
// concrete docsearch.Service to this shape.
type TextFallback func(ctx context.Context, q pkg.SearchQuery) ([]pkg.SearchHit, error)

// ServiceOptions configures a SearchService. Embedder/VectorStore are required
// for the vector legs; TextFallback (optional) provides the grep degradation;
// Reranker (optional) enables two-stage retrieval; Reconciler (optional) feeds
// Status's last-reconcile fields.
type ServiceOptions struct {
	Embedder     pkg.Embedder
	Sparse       pkg.SparseEncoder
	VectorStore  pkg.VectorStore
	Reranker     *pkg.RerankerChain
	TextFallback TextFallback
	Reconciler   *pkg.Reconciler
}

// SearchService is KO's documentation search read path. It satisfies
// pkg.Service.
type SearchService struct {
	embedder   pkg.Embedder
	sparse     pkg.SparseEncoder
	store      pkg.VectorStore
	reranker   *pkg.RerankerChain
	text       TextFallback
	reconciler *pkg.Reconciler
}

// NewSearchService wires the documentation search service. Sparse defaults to
// the BM25 encoder (the same one the indexer uses, so query/index weighting
// match).
func NewSearchService(opts ServiceOptions) *SearchService {
	sparse := opts.Sparse
	if sparse == nil {
		sparse = pkg.NewBM25SparseEncoder()
	}
	return &SearchService{
		embedder:   opts.Embedder,
		sparse:     sparse,
		store:      opts.VectorStore,
		reranker:   opts.Reranker,
		text:       opts.TextFallback,
		reconciler: opts.Reconciler,
	}
}

// NewDefaultReranker builds the production degradation chain: the cross-encoder
// TEI resource first, the always-available LLM reranker second. The chain
// returns the fused order when neither is reachable.
func NewDefaultReranker() *pkg.RerankerChain {
	return pkg.NewRerankerChain(pkg.NewCrossEncoderReranker(), pkg.NewLLMReranker(""))
}

func normalizeQuery(q pkg.SearchQuery) pkg.SearchQuery {
	q.Query = strings.TrimSpace(q.Query)
	if q.Mode == "" {
		q.Mode = pkg.ModeAuto
	}
	if q.Scope.Kind == "" {
		q.Scope.Kind = pkg.ScopeGlobal
	}
	if q.Limit <= 0 {
		q.Limit = defaultSearchLimit
	}
	if q.Limit > maxSearchLimit {
		q.Limit = maxSearchLimit
	}
	return q
}

// Search runs one documentation query, dispatching on mode and walking the auto
// fallback chain so it never hard-fails.
func (s *SearchService) Search(ctx context.Context, q pkg.SearchQuery) (pkg.SearchResponse, error) {
	q = normalizeQuery(q)
	if q.Query == "" {
		return pkg.SearchResponse{}, fmt.Errorf("query is required")
	}

	switch q.Mode {
	case pkg.ModeText:
		return s.textSearch(ctx, q)
	case pkg.ModeDense:
		return s.vectorSearch(ctx, q, false)
	case pkg.ModeHybrid:
		return s.vectorSearch(ctx, q, true)
	case pkg.ModeAuto:
		return s.autoSearch(ctx, q)
	default:
		return pkg.SearchResponse{}, fmt.Errorf("unknown search mode %q", q.Mode)
	}
}

// autoSearch walks hybrid -> dense -> grep, degrading on unavailability or
// error so search always returns something useful.
func (s *SearchService) autoSearch(ctx context.Context, q pkg.SearchQuery) (pkg.SearchResponse, error) {
	storeUp := s.store != nil && s.store.Available(ctx)
	embedUp := s.embedder != nil && s.embedder.Available(ctx)
	if storeUp && embedUp {
		if resp, err := s.vectorSearch(ctx, q, true); err == nil && len(resp.Results) > 0 {
			return resp, nil
		}
		// Hybrid empty/errored — try dense-only before grep.
		if resp, err := s.vectorSearch(ctx, q, false); err == nil && len(resp.Results) > 0 {
			return resp, nil
		}
	}
	return s.textSearch(ctx, q)
}

// vectorSearch runs the dense or hybrid vector leg, then authority-boosts,
// reranks, and projects the result.
func (s *SearchService) vectorSearch(ctx context.Context, q pkg.SearchQuery, hybrid bool) (pkg.SearchResponse, error) {
	if s.store == nil || s.embedder == nil {
		return pkg.SearchResponse{}, fmt.Errorf("vector search requires an embedder and vector store")
	}
	dense, err := s.embedder.Embed(ctx, q.Query)
	if err != nil {
		return pkg.SearchResponse{}, fmt.Errorf("embed query: %w", err)
	}

	shortlist := q.Limit * shortlistMultiplier
	if shortlist < minShortlist {
		shortlist = minShortlist
	}
	if shortlist > maxShortlist {
		shortlist = maxShortlist
	}
	// Path scope is filtered client-side (Qdrant keyword match has no prefix
	// op), so widen the shortlist to avoid losing in-prefix hits.
	if q.Scope.Kind == pkg.ScopePath {
		shortlist = maxShortlist
	}

	hq := pkg.HybridQuery{
		Dense:         dense,
		Filter:        docFilter(q.Scope, q.Facets),
		Limit:         shortlist,
		PrefetchLimit: prefetchLimit,
	}
	method := "dense"
	if hybrid {
		sparse := s.sparse.Encode(q.Query)
		hq.Sparse = &sparse
		hq.Fusion = "rrf"
		method = "hybrid"
	}

	raw, err := s.store.Query(ctx, hq)
	if err != nil {
		return pkg.SearchResponse{}, fmt.Errorf("vector query: %w", err)
	}

	hits := projectHits(raw)
	hits = filterPathScope(hits, q.Scope)
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })

	rerankerName := "none"
	if s.reranker != nil {
		if active := s.reranker.Active(ctx); active != nil {
			reordered, rerr := s.rerank(ctx, q.Query, hits, active)
			if rerr == nil {
				hits = reordered
				rerankerName = active.Name()
			}
			// On reranker error keep the fused order (rerankerName stays "none").
		}
	}

	// Authority is a final, reranker-agnostic nudge: it edges canonical,
	// maintained, and project-level docs up at the margins regardless of which
	// leg (fusion / cross-encoder / LLM) produced the order. A neutral
	// cross-encoder can't know a root /docs page is the canonical answer over a
	// scenario's same-named doc; this signal supplies that. Applied as a
	// scale-relative multiplier so it survives both ~0.01 RRF scores and ~0..1
	// rerank scores without dominating either.
	applyAuthorityBoost(hits)
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })

	total := len(hits)
	if len(hits) > q.Limit {
		hits = hits[:q.Limit]
	}
	return pkg.SearchResponse{
		Results:  hits,
		Total:    total,
		Query:    q.Query,
		Method:   method,
		Reranker: rerankerName,
	}, nil
}

// rerank scores the shortlist with the active reranker and reorders it,
// preserving fused order for any candidate the reranker omits.
func (s *SearchService) rerank(ctx context.Context, query string, hits []pkg.SearchHit, active pkg.Reranker) ([]pkg.SearchHit, error) {
	cands := make([]pkg.RerankCandidate, len(hits))
	for i, h := range hits {
		cands[i] = pkg.RerankCandidate{ID: h.ID, Text: rerankText(h)}
	}
	scores, err := active.Rerank(ctx, query, cands)
	if err != nil {
		return nil, err
	}
	return pkg.ApplyRerank(hits, scores), nil
}

// textSearch is the grep degradation leg.
func (s *SearchService) textSearch(ctx context.Context, q pkg.SearchQuery) (pkg.SearchResponse, error) {
	if s.text == nil {
		return pkg.SearchResponse{}, fmt.Errorf("text fallback is not configured")
	}
	hits, err := s.text(ctx, q)
	if err != nil {
		return pkg.SearchResponse{}, fmt.Errorf("text search: %w", err)
	}
	total := len(hits)
	if len(hits) > q.Limit {
		hits = hits[:q.Limit]
	}
	return pkg.SearchResponse{Results: hits, Total: total, Query: q.Query, Method: "text", Reranker: "none"}, nil
}

// Status reports backend availability for `search status`.
func (s *SearchService) Status(ctx context.Context) pkg.StatusReport {
	ollama := s.embedder != nil && s.embedder.Available(ctx)
	qdrant := s.store != nil && s.store.Available(ctx)
	report := pkg.StatusReport{
		Ollama:   ollama,
		Qdrant:   qdrant,
		Reranker: "none",
		// Search is available whenever a vector backend OR the grep fallback can
		// answer — it never hard-fails.
		Available: qdrant || s.text != nil,
	}
	if qdrant {
		if count, err := s.store.CountPoints(ctx); err == nil {
			report.IndexedCount = count
		}
	}
	if s.reranker != nil {
		report.Reranker = s.reranker.ActiveName(ctx)
	}
	if s.reconciler != nil {
		st := s.reconciler.Status()
		report.LastReconcileAt = st.FinishedAt
		switch {
		case st.Running:
			report.LastReconcileOutcome = "running"
		case st.Canceled:
			report.LastReconcileOutcome = "canceled"
		case st.LastError != "":
			report.LastReconcileOutcome = "error: " + st.LastError
		case st.FinishedAt != "":
			report.LastReconcileOutcome = "ok"
		}
	}
	return report
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
func filterPathScope(hits []pkg.SearchHit, scope pkg.Scope) []pkg.SearchHit {
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
func applyAuthorityBoost(hits []pkg.SearchHit) {
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

// projectHits maps raw vector-store results into the federation SearchHit
// shape, pulling the retrievable body, paths, and metadata from the payload.
func projectHits(raw []pkg.SearchResult) []pkg.SearchHit {
	out := make([]pkg.SearchHit, 0, len(raw))
	for _, r := range raw {
		rel := payloadString(r.Payload, MetaRelativePath)
		path := payloadString(r.Payload, MetaPath)
		if path == "" {
			path = rel
		}
		out = append(out, pkg.SearchHit{
			ID:           r.ID,
			RelativePath: rel,
			Score:        r.Score,
			Snippet:      snippet(payloadString(r.Payload, "body")),
			Path:         path,
			SourceID:     payloadString(r.Payload, "source_id"),
			Payload:      r.Payload,
		})
	}
	return out
}

// rerankText composes the contextual text a reranker scores: title, heading
// path, then the chunk body (bounded). Mirrors the indexing-time contextual
// prefix so the reranker sees the same self-contained context.
func rerankText(h pkg.SearchHit) string {
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
