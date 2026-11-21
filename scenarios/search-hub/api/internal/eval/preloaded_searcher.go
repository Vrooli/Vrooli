package eval

import (
	"context"
	"fmt"

	aisearch "github.com/vrooli/ai-go/search"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

type cachedSearchResult struct {
	response aisearch.SearchResponse
	err      error
}

// preloadedSuiteSearcher adapts the already-fetched provider results to the
// shared ai-go fixed-denominator grader, so the runner grades each real request
// once and does not issue a second search to compute acceptance recall.
type preloadedSuiteSearcher struct {
	entries map[preloadedQueryKey][]cachedSearchResult
	readAt  map[preloadedQueryKey]int
}

type preloadedQueryKey struct {
	query string
	scope aisearch.Scope
}

func newPreloadedSuiteSearcher() *preloadedSuiteSearcher {
	return &preloadedSuiteSearcher{entries: map[preloadedQueryKey][]cachedSearchResult{}, readAt: map[preloadedQueryKey]int{}}
}

func (s *preloadedSuiteSearcher) Add(c *evalv1.EvalCase, hits []*routingv1.SearchHit, searchErr error) {
	if c == nil {
		return
	}
	testCase := protoCaseToSearchCase(c)
	key := preloadedQueryKey{query: testCase.Query, scope: testCase.ResolvedScope()}
	results := make([]aisearch.SearchResult, 0, len(hits))
	for _, hit := range hits {
		if hit == nil {
			continue
		}
		id := hit.GetId()
		if matched := matchingExpectedIdentity(c, hit); matched != "" {
			id = matched
		}
		results = append(results, aisearch.SearchResult{ID: id, Score: hit.GetScore()})
	}
	s.entries[key] = append(s.entries[key], cachedSearchResult{response: aisearch.SearchResponse{Results: results}, err: searchErr})
}

func (s *preloadedSuiteSearcher) Search(_ context.Context, query aisearch.SearchQuery, _ ...aisearch.SearchOption) (aisearch.SearchResponse, error) {
	key := preloadedQueryKey{query: query.Query, scope: query.Scope}
	entries := s.entries[key]
	index := s.readAt[key]
	if index >= len(entries) {
		return aisearch.SearchResponse{}, fmt.Errorf("no preloaded provider result for fixed-denominator query")
	}
	s.readAt[key] = index + 1
	return entries[index].response, entries[index].err
}
