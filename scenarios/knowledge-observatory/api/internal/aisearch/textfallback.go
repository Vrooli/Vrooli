package aisearch

import (
	"context"
	"strconv"
	"strings"

	pkg "github.com/vrooli/ai-go/search"

	"knowledge-observatory/internal/services/docsearch"
)

// NewDocsearchFallback adapts the repurposed filesystem grep service
// (internal/services/docsearch) to the search service's TextFallback shape. It
// is the offline-safe leg of the auto chain (plan §3.3): when Ollama/Qdrant are
// down, search degrades to keyword grep and still returns path-anchored hits.
//
// docsearch already spans project-level /docs + scenario docs and supports the
// same global/scenario/path scopes, so the mapping is direct; the only shaping
// is projecting a TextSearchMatch into the federation SearchHit (keys.go).
func NewDocsearchFallback(svc *docsearch.Service) TextFallback {
	return func(ctx context.Context, q pkg.SearchQuery) ([]pkg.SearchResult, error) {
		req := docsearch.TextSearchRequest{
			Query:        q.Query,
			Scope:        scopeToDocsearch(q.Scope.Kind),
			Limit:        q.Limit,
			ContextLines: 1,
		}
		switch q.Scope.Kind {
		case pkg.ScopeScenario:
			req.Scenario = q.Scope.Value
		case pkg.ScopePath:
			req.BasePath = q.Scope.Value
		}
		matches, err := svc.SearchText(ctx, req)
		if err != nil {
			return nil, err
		}
		hits := make([]pkg.SearchResult, 0, len(matches))
		for i, m := range matches {
			body := strings.TrimSpace(strings.Join([]string{m.ContextBefore, m.Content, m.ContextAfter}, "\n"))
			hits = append(hits, pkg.SearchResult{
				ID:           grepID(m.Path, m.LineNumber),
				RelativePath: m.RelativePath,
				// Grep has no relevance score; preserve match order with a
				// gently decreasing synthetic score so downstream sorts are stable.
				Score:   1.0 - float64(i)*0.001,
				Snippet: snippet(body),
				Path:    m.Path,
				Payload: map[string]any{
					MetaRelativePath: m.RelativePath,
					MetaPath:         m.Path,
					MetaScenario:     m.Scenario,
					"body":           body,
					"line_number":    m.LineNumber,
				},
			})
		}
		return hits, nil
	}
}

func scopeToDocsearch(kind pkg.ScopeKind) string {
	switch kind {
	case pkg.ScopeScenario:
		return docsearch.ScopeScenario
	case pkg.ScopePath:
		return docsearch.ScopePath
	default:
		return docsearch.ScopeGlobal
	}
}

func grepID(path string, line int) string {
	return IDPrefix + "grep:" + path + ":" + strconv.Itoa(line)
}
