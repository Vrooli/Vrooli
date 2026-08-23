package aisearch

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"golang.org/x/sync/semaphore"
)

// WeightedAdmission is a process-local weighted budget for expensive search,
// embedding, reranking, and indexing work.
type WeightedAdmission struct {
	capacity int64
	sem      *semaphore.Weighted
}

func NewWeightedAdmission(capacity int64) *WeightedAdmission {
	if capacity <= 0 {
		capacity = 1
	}
	return &WeightedAdmission{capacity: capacity, sem: semaphore.NewWeighted(capacity)}
}

func (a *WeightedAdmission) Acquire(ctx context.Context, weight int64) (func(), error) {
	if a == nil || a.sem == nil {
		return nil, fmt.Errorf("admission budget is not configured")
	}
	if weight <= 0 || weight > a.capacity {
		return nil, fmt.Errorf("admission weight %d must be within 1..%d", weight, a.capacity)
	}
	if err := a.sem.Acquire(ctx, weight); err != nil {
		return nil, err
	}
	return func() { a.sem.Release(weight) }, nil
}

// FusionResponse reports fused results and any unavailable retrieval legs.
// One healthy leg is sufficient; both failing returns an error.
type FusionResponse struct {
	Results  []FusedResult `json:"results"`
	Degraded []string      `json:"degraded,omitempty"`
}

// ConcurrentFusion runs lexical and semantic retrieval concurrently and
// combines them with reciprocal rank fusion. The evidence on every result
// makes the ranking contribution inspectable.
type ConcurrentFusion struct {
	Lexical  LexicalSearcher
	Semantic SemanticSearcher
	RRFK     int
}

type retrievalLegResult struct {
	name string
	hits []SearchResult
	err  error
}

func (f ConcurrentFusion) Search(ctx context.Context, query SearchQuery) (FusionResponse, error) {
	if f.Lexical == nil && f.Semantic == nil {
		return FusionResponse{}, fmt.Errorf("fusion requires at least one retrieval leg")
	}
	legCount := 0
	results := make(chan retrievalLegResult, 2)
	if f.Lexical != nil {
		legCount++
		go func() {
			hits, err := f.Lexical.SearchLexical(ctx, query)
			results <- retrievalLegResult{name: "lexical", hits: hits, err: err}
		}()
	}
	if f.Semantic != nil {
		legCount++
		go func() {
			hits, err := f.Semantic.SearchSemantic(ctx, query)
			results <- retrievalLegResult{name: "semantic", hits: hits, err: err}
		}()
	}

	legs := make([]retrievalLegResult, 0, legCount)
	var failures []error
	response := FusionResponse{}
	for range legCount {
		select {
		case <-ctx.Done():
			return FusionResponse{}, ctx.Err()
		case leg := <-results:
			if leg.err != nil {
				response.Degraded = append(response.Degraded, leg.name)
				failures = append(failures, fmt.Errorf("%s retrieval: %w", leg.name, leg.err))
				continue
			}
			legs = append(legs, leg)
		}
	}
	if len(legs) == 0 {
		return response, errors.Join(failures...)
	}
	sort.Strings(response.Degraded)
	response.Results = fuseRankedResults(legs, f.rrfK(), query.Limit)
	return response, nil
}

func (f ConcurrentFusion) rrfK() int {
	if f.RRFK > 0 {
		return f.RRFK
	}
	return DefaultRRFK
}

func fuseRankedResults(legs []retrievalLegResult, k, limit int) []FusedResult {
	type aggregate struct {
		result   SearchResult
		score    float64
		evidence []RankEvidence
	}
	byID := make(map[string]*aggregate)
	for _, leg := range legs {
		for index, hit := range leg.hits {
			id := strings.TrimSpace(hit.ID)
			if id == "" {
				continue
			}
			agg := byID[id]
			if agg == nil {
				copyHit := hit
				agg = &aggregate{result: copyHit}
				byID[id] = agg
			}
			rank := index + 1
			agg.score += 1 / float64(k+rank)
			agg.evidence = append(agg.evidence, RankEvidence{Leg: leg.name, Rank: rank, Score: hit.Score})
		}
	}
	out := make([]FusedResult, 0, len(byID))
	for _, agg := range byID {
		agg.result.Score = agg.score
		sort.Slice(agg.evidence, func(i, j int) bool { return agg.evidence[i].Leg < agg.evidence[j].Leg })
		out = append(out, FusedResult{Result: agg.result, Evidence: agg.evidence})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Result.Score == out[j].Result.Score {
			return out[i].Result.ID < out[j].Result.ID
		}
		return out[i].Result.Score > out[j].Result.Score
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}
