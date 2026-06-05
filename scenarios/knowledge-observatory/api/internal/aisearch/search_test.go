package aisearch

import (
	"context"
	"errors"
	"testing"

	pkg "github.com/vrooli/aisearch-go"
)

// searchEmbedder returns a fixed vector and a controllable availability.
type searchEmbedder struct {
	available bool
	err       error
}

func (f *searchEmbedder) Embed(context.Context, string) ([]float64, error) {
	if f.err != nil {
		return nil, f.err
	}
	return []float64{0.1, 0.2, 0.3}, nil
}
func (f *searchEmbedder) Available(context.Context) bool { return f.available }

// searchStore records the last query and returns canned results.
type searchStore struct {
	available bool
	results   []pkg.SearchResult
	lastQuery pkg.HybridQuery
	count     int
	queryErr  error
}

func (f *searchStore) EnsureCollection(context.Context, pkg.CollectionSpec) error { return nil }
func (f *searchStore) Upsert(context.Context, pkg.Point) error                    { return nil }
func (f *searchStore) SetPayload(context.Context, string, map[string]any) error   { return nil }
func (f *searchStore) BatchDelete(context.Context, []string) error                { return nil }
func (f *searchStore) Query(_ context.Context, q pkg.HybridQuery) ([]pkg.SearchResult, error) {
	f.lastQuery = q
	if f.queryErr != nil {
		return nil, f.queryErr
	}
	return f.results, nil
}
func (f *searchStore) CountPoints(context.Context) (int, error) { return f.count, nil }
func (f *searchStore) ScrollIDs(context.Context) (map[string]pkg.ScrollItem, error) {
	return map[string]pkg.ScrollItem{}, nil
}
func (f *searchStore) Available(context.Context) bool { return f.available }

func docResult(id, rel, body, maturity string) pkg.SearchResult {
	return pkg.SearchResult{
		ID:    id,
		Score: 0.5,
		Payload: map[string]any{
			MetaRelativePath: rel,
			MetaPath:         rel,
			MetaScenario:     "cli-health",
			MetaMaturity:     maturity,
			"body":           body,
			"source_id":      "ko-docs:" + rel,
		},
	}
}

func TestVectorSearchHybridProjectsAndFuses(t *testing.T) {
	store := &searchStore{available: true, results: []pkg.SearchResult{
		docResult("ko-docs:a#0", "docs/a.md", "alpha body content", "draft"),
		docResult("ko-docs:b#0", "docs/b.md", "bravo body content", "active"),
	}}
	svc := NewSearchService(ServiceOptions{
		Embedder:    &searchEmbedder{available: true},
		VectorStore: store,
	})
	resp, err := svc.Search(context.Background(), pkg.SearchQuery{Query: "content", Mode: pkg.ModeHybrid})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if resp.Method != "hybrid" {
		t.Fatalf("method = %q, want hybrid", resp.Method)
	}
	if store.lastQuery.Sparse == nil || store.lastQuery.Fusion != "rrf" {
		t.Fatalf("expected sparse+rrf in hybrid query, got %+v", store.lastQuery)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("got %d results, want 2", len(resp.Results))
	}
	// Authority boost should float the active doc (b) above the draft (a),
	// despite equal base scores.
	if resp.Results[0].RelativePath != "docs/b.md" {
		t.Fatalf("authority boost failed: top = %q, want docs/b.md", resp.Results[0].RelativePath)
	}
	// Federation contract fields are populated.
	if resp.Results[0].Snippet == "" || resp.Results[0].Path == "" {
		t.Fatalf("projection missing snippet/path: %+v", resp.Results[0])
	}
}

func TestVectorSearchDenseOmitsSparse(t *testing.T) {
	store := &searchStore{available: true, results: []pkg.SearchResult{docResult("ko-docs:a#0", "docs/a.md", "body", "active")}}
	svc := NewSearchService(ServiceOptions{Embedder: &searchEmbedder{available: true}, VectorStore: store})
	resp, err := svc.Search(context.Background(), pkg.SearchQuery{Query: "x", Mode: pkg.ModeDense})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if resp.Method != "dense" {
		t.Fatalf("method = %q, want dense", resp.Method)
	}
	if store.lastQuery.Sparse != nil || store.lastQuery.Fusion != "" {
		t.Fatalf("dense query must not set sparse/fusion: %+v", store.lastQuery)
	}
}

func TestScopeAndFacetFilter(t *testing.T) {
	store := &searchStore{available: true}
	svc := NewSearchService(ServiceOptions{Embedder: &searchEmbedder{available: true}, VectorStore: store})
	_, _ = svc.Search(context.Background(), pkg.SearchQuery{
		Query:  "x",
		Mode:   pkg.ModeDense,
		Scope:  pkg.Scope{Kind: pkg.ScopeScenario, Value: "cli-health"},
		Facets: pkg.Facets{DocType: []string{"reference"}, Maturity: []string{"active"}},
	})
	f := store.lastQuery.Filter
	if f == nil {
		t.Fatal("expected a filter for scenario scope + facets")
	}
	keys := map[string]any{}
	for _, m := range f.Must {
		if m.Value != nil {
			keys[m.Key] = m.Value
		} else {
			keys[m.Key] = m.AnyOf
		}
	}
	if keys[MetaScenario] != "cli-health" {
		t.Fatalf("scenario filter missing: %+v", keys)
	}
	if _, ok := keys[MetaDocType]; !ok {
		t.Fatalf("doc_type facet missing: %+v", keys)
	}
	if _, ok := keys[MetaMaturity]; !ok {
		t.Fatalf("maturity facet missing: %+v", keys)
	}
}

func TestPathScopeFiltersClientSide(t *testing.T) {
	store := &searchStore{available: true, results: []pkg.SearchResult{
		docResult("ko-docs:a#0", "docs/guides/a.md", "body", "active"),
		docResult("ko-docs:b#0", "scenarios/x/README.md", "body", "active"),
	}}
	svc := NewSearchService(ServiceOptions{Embedder: &searchEmbedder{available: true}, VectorStore: store})
	resp, err := svc.Search(context.Background(), pkg.SearchQuery{
		Query: "x", Mode: pkg.ModeDense,
		Scope: pkg.Scope{Kind: pkg.ScopePath, Value: "docs/guides"},
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(resp.Results) != 1 || resp.Results[0].RelativePath != "docs/guides/a.md" {
		t.Fatalf("path scope failed: %+v", resp.Results)
	}
	// Path scope is now pushed into the qdrant filter as a path_prefixes segment
	// match (so in-prefix docs are retrieved), with filterPathScope still doing
	// the exact client-side trim above.
	if store.lastQuery.Filter == nil {
		t.Fatal("path scope should produce a server filter on path_prefixes")
	}
	var sawPrefix bool
	for _, m := range store.lastQuery.Filter.Must {
		if m.Key == MetaPathPrefixes && m.Value == "docs/guides" {
			sawPrefix = true
		}
	}
	if !sawPrefix {
		t.Fatalf("path scope filter missing path_prefixes=docs/guides: %+v", store.lastQuery.Filter.Must)
	}
}

func TestRerankReorders(t *testing.T) {
	store := &searchStore{available: true, results: []pkg.SearchResult{
		docResult("ko-docs:a#0", "docs/a.md", "alpha", "draft"),
		docResult("ko-docs:b#0", "docs/b.md", "bravo", "draft"),
	}}
	// Reranker promotes b strongly.
	rr := pkg.NewRerankerChain(&fixedReranker{
		name:      "test",
		available: true,
		scores:    []pkg.RerankScore{{ID: "ko-docs:b#0", Score: 0.99}, {ID: "ko-docs:a#0", Score: 0.01}},
	})
	svc := NewSearchService(ServiceOptions{Embedder: &searchEmbedder{available: true}, VectorStore: store, Reranker: rr})
	resp, err := svc.Search(context.Background(), pkg.SearchQuery{Query: "x", Mode: pkg.ModeDense})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if resp.Reranker != "test" {
		t.Fatalf("reranker name = %q, want test", resp.Reranker)
	}
	if resp.Results[0].RelativePath != "docs/b.md" {
		t.Fatalf("rerank did not reorder: %+v", resp.Results)
	}
}

func TestAutoFallsBackToText(t *testing.T) {
	// Store down + embedder down => auto must use the text fallback.
	store := &searchStore{available: false}
	called := false
	text := func(_ context.Context, q pkg.SearchQuery) ([]pkg.SearchHit, error) {
		called = true
		return []pkg.SearchHit{{ID: "grep:1", RelativePath: "docs/z.md", Score: 1, Path: "docs/z.md"}}, nil
	}
	svc := NewSearchService(ServiceOptions{
		Embedder: &searchEmbedder{available: false}, VectorStore: store, TextFallback: text,
	})
	resp, err := svc.Search(context.Background(), pkg.SearchQuery{Query: "x"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if !called || resp.Method != "text" {
		t.Fatalf("expected text fallback; called=%v method=%q", called, resp.Method)
	}
}

func TestAutoFallsBackWhenHybridErrors(t *testing.T) {
	store := &searchStore{available: true, queryErr: errors.New("qdrant boom")}
	text := func(_ context.Context, _ pkg.SearchQuery) ([]pkg.SearchHit, error) {
		return []pkg.SearchHit{{ID: "grep:1", RelativePath: "docs/z.md", Score: 1}}, nil
	}
	svc := NewSearchService(ServiceOptions{
		Embedder: &searchEmbedder{available: true}, VectorStore: store, TextFallback: text,
	})
	resp, err := svc.Search(context.Background(), pkg.SearchQuery{Query: "x"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if resp.Method != "text" {
		t.Fatalf("expected text fallback after hybrid error; got %q", resp.Method)
	}
}

func TestStatusReportsBackends(t *testing.T) {
	svc := NewSearchService(ServiceOptions{
		Embedder:    &searchEmbedder{available: true},
		VectorStore: &searchStore{available: true, count: 6373},
		Reranker:    pkg.NewRerankerChain(&fixedReranker{name: "cross", available: true}),
	})
	st := svc.Status(context.Background())
	if !st.Available || !st.Ollama || !st.Qdrant {
		t.Fatalf("status backends wrong: %+v", st)
	}
	if st.IndexedCount != 6373 {
		t.Fatalf("indexed count = %d, want 6373", st.IndexedCount)
	}
	if st.Reranker != "cross" {
		t.Fatalf("reranker = %q, want cross", st.Reranker)
	}
}

func TestEmptyQueryRejected(t *testing.T) {
	svc := NewSearchService(ServiceOptions{Embedder: &searchEmbedder{available: true}, VectorStore: &searchStore{available: true}})
	if _, err := svc.Search(context.Background(), pkg.SearchQuery{Query: "  "}); err == nil {
		t.Fatal("expected error for empty query")
	}
}

// fixedReranker is a deterministic Reranker for service tests.
type fixedReranker struct {
	name      string
	available bool
	scores    []pkg.RerankScore
	err       error
}

func (r *fixedReranker) Name() string                   { return r.name }
func (r *fixedReranker) Available(context.Context) bool { return r.available }
func (r *fixedReranker) Rerank(context.Context, string, []pkg.RerankCandidate) ([]pkg.RerankScore, error) {
	return r.scores, r.err
}
