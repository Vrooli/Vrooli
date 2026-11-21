package aisearch

import (
	"context"
	"strings"
	"time"

	pkg "github.com/vrooli/ai-go/search"
)

// Searcher is the read path the search handler serves: ranked hits over the
// authoritative catalog plus the truthful corpus status. Both the lexical
// Service (the offline projection) and the engine-backed LiveSearch satisfy it.
type Searcher interface {
	Search(ctx context.Context, query string, limit int) (*SearchResponse, error)
	Status(ctx context.Context) (StatusReport, error)
}

// LiveSearch is the engine-backed read path. The shared engine ranks and
// projects the catalog, while the lexical Service stays the authoritative
// status/generation source and the engine's offline TextFallback. When no engine
// could be assembled (an unresolved embedding policy or an unreachable vector
// store) the LiveSearch serves the lexical Service directly, so retrieval is
// never worse than the pre-engine baseline.
type LiveSearch struct {
	engine  *Engine
	lexical *Service
}

// NewLiveSearch pairs an optional engine with its authoritative lexical leg.
// lexical must be non-nil; engine may be nil.
func NewLiveSearch(engine *Engine, lexical *Service) *LiveSearch {
	return &LiveSearch{engine: engine, lexical: lexical}
}

// NewLexicalSearch is the engine-less read path: the lexical projection only.
func NewLexicalSearch(lexical *Service) *LiveSearch { return NewLiveSearch(nil, lexical) }

// EngineAvailable reports whether vector retrieval is wired. It is diagnostics,
// not a search result: status truth stays derived from the catalog.
func (l *LiveSearch) EngineAvailable() bool { return l != nil && l.engine != nil }

// Search ranks the corpus. With an engine it projects engine hits and attaches
// the corpus generation the lexical snapshot read; on any engine error it
// degrades to the lexical ranking rather than failing the request.
func (l *LiveSearch) Search(ctx context.Context, query string, limit int) (*SearchResponse, error) {
	if !l.EngineAvailable() {
		return l.lexical.Search(ctx, query, limit)
	}
	hits, _, err := l.engine.Search(ctx, query, limit)
	if err != nil {
		return l.lexical.Search(ctx, query, limit)
	}
	snapshot, err := l.lexical.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	return &SearchResponse{
		Hits:           hits,
		Generation:     snapshot.Generation,
		MaterializedAt: snapshot.MaterializedAt,
	}, nil
}

// Status reports the authoritative catalog state plus the index's currency.
// The registered index-timestamp contract is the LAST SUCCESSFUL reconcile, not
// the corpus's newest content mutation: a live-reconciled corpus whose source
// has not changed is still freshly indexed. Corpus content identity stays in
// Generation (the source materialization rides SearchResponse.MaterializedAt),
// while LastIndexedAt reports the engine reconcile time so Search Hub's
// freshness gate measures index currency rather than source age. Without an
// engine there is no reconcile, so the source materialization remains the only
// honest timestamp.
func (l *LiveSearch) Status(ctx context.Context) (StatusReport, error) {
	status, err := l.lexical.Status(ctx)
	if err != nil {
		return status, err
	}
	if l.EngineAvailable() {
		status = applyIndexTime(status, l.engine.Status(ctx).LastReconcileAt)
	}
	return status, nil
}

// applyIndexTime overrides the corpus-derived index timestamp with the engine's
// last successful reconcile time when it is well-formed. An empty or unparsable
// reconcile time leaves the source materialization in place, so a degraded boot
// or a reconcile still in flight never fabricates an index time.
func applyIndexTime(status StatusReport, reconciled string) StatusReport {
	reconciled = strings.TrimSpace(reconciled)
	if reconciled == "" {
		return status
	}
	at, err := time.Parse(time.RFC3339, reconciled)
	if err != nil {
		return status
	}
	status.LastIndexedAt = at.UTC()
	return status
}

// textFallbackOf adapts the lexical Service to the shared engine's offline leg:
// it runs the same catalog projection and returns the shared SearchResult shape
// (with the typed projection in Payload) so a down vector backend yields
// lexical hits instead of an empty page.
func textFallbackOf(svc *Service) pkg.TextFallbackFunc {
	return func(ctx context.Context, query pkg.SearchQuery) ([]pkg.SearchResult, error) {
		response, err := svc.Search(ctx, query.Query, query.Limit)
		if err != nil {
			return nil, err
		}
		out := make([]pkg.SearchResult, 0, len(response.Hits))
		for _, hit := range response.Hits {
			out = append(out, pkg.SearchResult{
				ID:           hit.ID,
				Score:        hit.Score,
				Payload:      hitPayload(hit),
				RelativePath: hit.FollowUp,
				Path:         hit.FollowUp,
				Snippet:      hit.Snippet,
				SourceID:     hit.ID,
			})
		}
		return out, nil
	}
}

// hitPayload mirrors the engine's sourceMetaFor projection so a lexical fallback
// hit survives the engine's typed projection unchanged.
func hitPayload(hit SearchHit) map[string]any {
	payload := map[string]any{
		"id":         hit.ID,
		"kind":       hit.Kind,
		"title":      hit.Title,
		"snippet":    hit.Snippet,
		"follow_up":  hit.FollowUp,
		"freshness":  hit.Freshness,
		"historical": hit.Historical,
	}
	if len(hit.Metadata) > 0 {
		payload["record"] = hit.Metadata
	}
	return payload
}
