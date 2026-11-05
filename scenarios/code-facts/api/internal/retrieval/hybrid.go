package retrieval

import (
	"container/list"
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	aisearch "github.com/vrooli/ai-go/search"
)

type FreshnessFence interface {
	Current(context.Context, Candidate) (bool, error)
}

type GraphExpander interface {
	Expand(context.Context, string, string, []string, int) ([]Candidate, error)
}

type HybridResponse struct {
	Results         []Candidate
	Degraded        []string
	SuppressedStale int
}

type HybridEngine struct {
	Planner   QueryPlanner
	Lexical   LexicalSearcher
	Semantic  SemanticSearcher
	Reranker  Reranker
	Freshness FreshnessFence
	Graph     GraphExpander
	RRFK      int
	Admission Admission
	Cache     *ResultCache
	Flights   *QueryFlights
}

func (engine HybridEngine) Search(ctx context.Context, query Query) (HybridResponse, error) {
	normalized, err := NormalizeQuery(query)
	if err != nil {
		return HybridResponse{}, err
	}
	key := normalizedQueryKey(normalized)
	if normalized.Generation != "" && engine.Cache != nil {
		if cached, ok := engine.Cache.Get(key); ok {
			return cached, nil
		}
	}
	run := func(workCtx context.Context) (HybridResponse, error) {
		release := func() {}
		if engine.Admission != nil {
			var acquireErr error
			release, acquireErr = engine.Admission.Acquire(workCtx, "query", 1)
			if acquireErr != nil {
				return HybridResponse{}, acquireErr
			}
		}
		defer release()
		response, searchErr := engine.search(workCtx, normalized)
		if searchErr == nil && normalized.Generation != "" && engine.Cache != nil {
			engine.Cache.Put(key, response)
		}
		return response, searchErr
	}
	if engine.Flights != nil {
		return engine.Flights.Do(ctx, key, run)
	}
	return run(ctx)
}

func (engine HybridEngine) search(ctx context.Context, normalized Query) (HybridResponse, error) {
	plan := engine.Planner.Plan(normalized)
	if engine.Lexical == nil {
		return HybridResponse{}, fmt.Errorf("hybrid engine requires lexical retrieval")
	}
	if plan.Regime == RegimeExact || engine.Semantic == nil {
		results, err := engine.Lexical.SearchLexical(ctx, normalized)
		response := HybridResponse{Results: results}
		if engine.Semantic == nil && plan.Regime != RegimeExact {
			response.Degraded = append(response.Degraded, "semantic")
		}
		if err == nil {
			engine.expandGraph(ctx, normalized, plan, &response)
		}
		return response, err
	}
	shared := aisearch.ConcurrentFusion{
		Lexical:  lexicalAdapter{searcher: engine.Lexical, query: normalized},
		Semantic: semanticAdapter{searcher: engine.Semantic, query: normalized},
		RRFK:     engine.RRFK,
	}
	fused, err := shared.Search(ctx, aisearch.SearchQuery{Query: normalized.Text, Limit: normalized.Limit})
	if err != nil {
		return HybridResponse{Degraded: append([]string(nil), fused.Degraded...)}, err
	}
	response := HybridResponse{Degraded: append([]string(nil), fused.Degraded...)}
	for _, result := range fused.Results {
		candidate, ok := result.Result.Payload["candidate"].(Candidate)
		if !ok {
			continue
		}
		candidate.Regime = plan.Regime
		candidate.Score = result.Result.Score
		candidate.RankEvidence = make([]RankEvidence, len(result.Evidence))
		for index, evidence := range result.Evidence {
			candidate.RankEvidence[index] = RankEvidence{Leg: evidence.Leg, Rank: evidence.Rank, Score: evidence.Score}
		}
		candidate.ScoreFactors = map[string]float64{"rrf": candidate.Score}
		candidate.Explanation = "shared reciprocal-rank fusion"
		if engine.Freshness != nil {
			current, fenceErr := engine.Freshness.Current(ctx, candidate)
			if fenceErr != nil {
				return HybridResponse{}, fenceErr
			}
			if !current {
				response.SuppressedStale++
				continue
			}
		}
		response.Results = append(response.Results, candidate)
	}
	applyRankingPolicy(response.Results, normalized)
	if plan.UseReranker && engine.Reranker != nil && len(response.Results) > 1 {
		reranked, rerankErr := engine.Reranker.Rerank(ctx, normalized, append([]Candidate(nil), response.Results...))
		if rerankErr != nil {
			response.Degraded = append(response.Degraded, "reranker")
		} else {
			response.Results = reranked
		}
	}
	response.Degraded = uniqueSorted(response.Degraded)
	if normalized.Limit > 0 && len(response.Results) > normalized.Limit {
		response.Results = response.Results[:normalized.Limit]
	}
	engine.expandGraph(ctx, normalized, plan, &response)
	return response, nil
}

type resultCacheEntry struct {
	key      string
	response HybridResponse
	bytes    int64
}

// ResultCache is a deliberately small generation-keyed LRU. It accounts bytes
// on mutation, never scans entries, and returns defensive copies.
type ResultCache struct {
	mu         sync.Mutex
	maxEntries int
	maxBytes   int64
	bytes      int64
	entries    map[string]*list.Element
	order      *list.List
}

func NewResultCache(maxEntries int, maxBytes int64) *ResultCache {
	if maxEntries <= 0 {
		maxEntries = 64
	}
	if maxBytes <= 0 {
		maxBytes = 8 * 1024 * 1024
	}
	return &ResultCache{maxEntries: maxEntries, maxBytes: maxBytes, entries: map[string]*list.Element{}, order: list.New()}
}

func (c *ResultCache) Get(key string) (HybridResponse, bool) {
	if c == nil {
		return HybridResponse{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	element := c.entries[key]
	if element == nil {
		return HybridResponse{}, false
	}
	c.order.MoveToFront(element)
	return cloneHybridResponse(element.Value.(resultCacheEntry).response), true
}

func (c *ResultCache) Put(key string, response HybridResponse) {
	if c == nil || key == "" {
		return
	}
	entry := resultCacheEntry{key: key, response: cloneHybridResponse(response), bytes: estimateHybridBytes(response)}
	if entry.bytes > c.maxBytes {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if existing := c.entries[key]; existing != nil {
		c.bytes -= existing.Value.(resultCacheEntry).bytes
		existing.Value = entry
		c.bytes += entry.bytes
		c.order.MoveToFront(existing)
	} else {
		c.entries[key] = c.order.PushFront(entry)
		c.bytes += entry.bytes
	}
	for len(c.entries) > c.maxEntries || c.bytes > c.maxBytes {
		oldest := c.order.Back()
		if oldest == nil {
			break
		}
		victim := oldest.Value.(resultCacheEntry)
		delete(c.entries, victim.key)
		c.bytes -= victim.bytes
		c.order.Remove(oldest)
	}
}

// Reset invalidates every generation-keyed query result after an atomic
// ordinary-file refresh. Reads never mutate the cache; refresh owns the reset.
func (c *ResultCache) Reset() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = map[string]*list.Element{}
	c.order.Init()
	c.bytes = 0
}

type queryFlightCall struct {
	done     chan struct{}
	response HybridResponse
	err      error
}

type QueryFlights struct {
	mu    sync.Mutex
	calls map[string]*queryFlightCall
}

func NewQueryFlights() *QueryFlights { return &QueryFlights{calls: map[string]*queryFlightCall{}} }

func (g *QueryFlights) Do(ctx context.Context, key string, fn func(context.Context) (HybridResponse, error)) (HybridResponse, error) {
	g.mu.Lock()
	if call := g.calls[key]; call != nil {
		g.mu.Unlock()
		select {
		case <-ctx.Done():
			return HybridResponse{}, ctx.Err()
		case <-call.done:
			return cloneHybridResponse(call.response), call.err
		}
	}
	call := &queryFlightCall{done: make(chan struct{})}
	g.calls[key] = call
	g.mu.Unlock()
	call.response, call.err = fn(ctx)
	g.mu.Lock()
	delete(g.calls, key)
	close(call.done)
	g.mu.Unlock()
	return cloneHybridResponse(call.response), call.err
}

func normalizedQueryKey(query Query) string {
	roles, families, languages := append([]string(nil), query.Roles...), append([]string(nil), query.Families...), append([]string(nil), query.Languages...)
	sort.Strings(roles)
	sort.Strings(families)
	sort.Strings(languages)
	return strings.Join([]string{query.Generation, query.Text, query.Target, query.Scope, strings.Join(roles, ","), strings.Join(families, ","), strings.Join(languages, ","), fmt.Sprint(query.Limit)}, "\x00")
}

func estimateHybridBytes(response HybridResponse) int64 {
	var total int64 = int64(len(response.Degraded) * 32)
	for _, candidate := range response.Results {
		total += int64(len(candidate.ID) + len(candidate.Path) + len(candidate.Title) + len(candidate.Text) + len(candidate.Evidence) + len(candidate.Proof) + 256)
	}
	return total
}

func cloneHybridResponse(response HybridResponse) HybridResponse {
	clone := HybridResponse{Degraded: append([]string(nil), response.Degraded...), Results: append([]Candidate(nil), response.Results...), SuppressedStale: response.SuppressedStale}
	for index := range clone.Results {
		clone.Results[index].ScoreFactors = cloneFloatMap(clone.Results[index].ScoreFactors)
		clone.Results[index].RankEvidence = append([]RankEvidence(nil), clone.Results[index].RankEvidence...)
	}
	return clone
}

func cloneFloatMap(source map[string]float64) map[string]float64 {
	if source == nil {
		return nil
	}
	clone := make(map[string]float64, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func (engine HybridEngine) expandGraph(ctx context.Context, query Query, plan Plan, response *HybridResponse) {
	if plan.Regime != RegimeRelationship || response == nil || len(response.Results) == 0 {
		return
	}
	if engine.Graph == nil {
		response.Degraded = uniqueSorted(append(response.Degraded, "graph"))
		return
	}
	subject := response.Results[0].ID
	if response.Results[0].Title != "" {
		subject = response.Results[0].Title
	}
	expanded, err := engine.Graph.Expand(ctx, response.Results[0].Generation, subject, query.Families, query.Limit)
	if err != nil {
		response.Degraded = uniqueSorted(append(response.Degraded, "graph"))
		return
	}
	for index := range expanded {
		expanded[index].Regime = RegimeRelationship
		if expanded[index].Proof == "" {
			expanded[index].Proof = "analyzer_confirmed"
		}
	}
	response.Results = append(response.Results, expanded...)
	if query.Limit > 0 && len(response.Results) > query.Limit {
		response.Results = response.Results[:query.Limit]
	}
}

type lexicalAdapter struct {
	searcher LexicalSearcher
	query    Query
}

func (adapter lexicalAdapter) SearchLexical(ctx context.Context, _ aisearch.SearchQuery) ([]aisearch.SearchResult, error) {
	candidates, err := adapter.searcher.SearchLexical(ctx, adapter.query)
	return sharedResults(candidates), err
}

type semanticAdapter struct {
	searcher SemanticSearcher
	query    Query
}

func (adapter semanticAdapter) SearchSemantic(ctx context.Context, _ aisearch.SearchQuery) ([]aisearch.SearchResult, error) {
	candidates, err := adapter.searcher.SearchSemantic(ctx, adapter.query)
	return sharedResults(candidates), err
}

func sharedResults(candidates []Candidate) []aisearch.SearchResult {
	results := make([]aisearch.SearchResult, 0, len(candidates))
	for _, candidate := range candidates {
		results = append(results, aisearch.SearchResult{ID: candidate.ID, Score: candidate.Score, Path: candidate.Path, Snippet: candidate.Text, Payload: map[string]any{"candidate": candidate}})
	}
	return results
}

func uniqueSorted(values []string) []string {
	set := map[string]struct{}{}
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			set[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

var _ FreshnessFence = (*SQLiteIndex)(nil)
