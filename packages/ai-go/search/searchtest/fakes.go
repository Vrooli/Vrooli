// Package searchtest provides deterministic fakes for adopters of the shared
// search contracts. Production packages must not import it.
package searchtest

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"sync"

	aisearch "github.com/vrooli/ai-go/search"
)

type Embedder struct {
	Vector         []float64
	Err            error
	AvailableValue bool

	mu    sync.Mutex
	Texts []string
}

func (e *Embedder) Embed(_ context.Context, text string) ([]float64, error) {
	e.mu.Lock()
	e.Texts = append(e.Texts, text)
	e.mu.Unlock()
	return append([]float64(nil), e.Vector...), e.Err
}

func (e *Embedder) Available(context.Context) bool { return e.AvailableValue }

type PagedSource struct {
	Documents []aisearch.SourceDoc
	Err       error
}

func (s PagedSource) LoadPage(_ context.Context, request aisearch.PageRequest) (aisearch.SourcePage, error) {
	if s.Err != nil {
		return aisearch.SourcePage{}, s.Err
	}
	start := 0
	if request.Cursor != "" {
		parsed, err := strconv.Atoi(request.Cursor)
		if err != nil {
			return aisearch.SourcePage{}, fmt.Errorf("parse cursor: %w", err)
		}
		start = parsed
	}
	if start < 0 || start > len(s.Documents) {
		return aisearch.SourcePage{}, fmt.Errorf("cursor %d exceeds corpus size %d", start, len(s.Documents))
	}
	limit := request.Limit
	if limit <= 0 {
		limit = aisearch.DefaultSourcePageSize
	}
	end := start + limit
	if end > len(s.Documents) {
		end = len(s.Documents)
	}
	page := aisearch.SourcePage{Documents: append([]aisearch.SourceDoc(nil), s.Documents[start:end]...), Done: end == len(s.Documents)}
	if !page.Done {
		page.NextCursor = strconv.Itoa(end)
	}
	return page, nil
}

type Lexical struct {
	Results []aisearch.SearchResult
	Err     error
}

func (s Lexical) SearchLexical(context.Context, aisearch.SearchQuery) ([]aisearch.SearchResult, error) {
	return cloneResults(s.Results), s.Err
}

type Semantic struct {
	Results []aisearch.SearchResult
	Err     error
}

func (s Semantic) SearchSemantic(context.Context, aisearch.SearchQuery) ([]aisearch.SearchResult, error) {
	return cloneResults(s.Results), s.Err
}

type VectorStore struct {
	mu sync.Mutex

	Points         map[string]aisearch.Point
	QueryResults   []aisearch.SearchResult
	Err            error
	AvailableValue bool
}

func NewVectorStore() *VectorStore {
	return &VectorStore{Points: map[string]aisearch.Point{}, AvailableValue: true}
}

func (s *VectorStore) EnsureCollection(context.Context, aisearch.CollectionSpec) error { return s.Err }

func (s *VectorStore) Upsert(_ context.Context, point aisearch.Point) error {
	if s.Err != nil {
		return s.Err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Points[point.ID] = clonePoint(point)
	return nil
}

func (s *VectorStore) SetPayload(_ context.Context, id string, payload map[string]any) error {
	if s.Err != nil {
		return s.Err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	point := s.Points[id]
	point.Payload = clonePayload(payload)
	s.Points[id] = point
	return nil
}

func (s *VectorStore) BatchDelete(_ context.Context, ids []string) error {
	if s.Err != nil {
		return s.Err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range ids {
		delete(s.Points, id)
	}
	return nil
}

func (s *VectorStore) Query(context.Context, aisearch.HybridQuery) ([]aisearch.SearchResult, error) {
	return cloneResults(s.QueryResults), s.Err
}

func (s *VectorStore) CountPoints(context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.Points), s.Err
}

func (s *VectorStore) ScrollIDs(context.Context) (map[string]aisearch.ScrollItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]aisearch.ScrollItem, len(s.Points))
	for id, point := range s.Points {
		item := aisearch.ScrollItem{}
		item.PayloadHash, _ = point.Payload["payload_hash"].(string)
		item.SourceID, _ = point.Payload["source_id"].(string)
		item.SourceHash, _ = point.Payload["source_hash"].(string)
		item.ChunkTotal, _ = point.Payload["chunk_total"].(int)
		out[id] = item
	}
	return out, s.Err
}

func (s *VectorStore) Available(context.Context) bool { return s.AvailableValue }

type Reranker struct {
	Leg            string
	Scores         map[string]float64
	Err            error
	AvailableValue bool
}

func (r Reranker) Name() string { return r.Leg }

func (r Reranker) Available(context.Context) bool { return r.AvailableValue }

func (r Reranker) Rerank(_ context.Context, _ string, candidates []aisearch.RerankCandidate) ([]aisearch.RerankScore, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	out := make([]aisearch.RerankScore, 0, len(candidates))
	for _, candidate := range candidates {
		out = append(out, aisearch.RerankScore{ID: candidate.ID, Score: r.Scores[candidate.ID]})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score == out[j].Score {
			return out[i].ID < out[j].ID
		}
		return out[i].Score > out[j].Score
	})
	return out, nil
}

func cloneResults(results []aisearch.SearchResult) []aisearch.SearchResult {
	out := make([]aisearch.SearchResult, len(results))
	for i, result := range results {
		out[i] = result
		out[i].Payload = clonePayload(result.Payload)
	}
	return out
}

func clonePoint(point aisearch.Point) aisearch.Point {
	copyPoint := point
	copyPoint.Dense = append([]float64(nil), point.Dense...)
	copyPoint.Payload = clonePayload(point.Payload)
	return copyPoint
}

func clonePayload(payload map[string]any) map[string]any {
	if payload == nil {
		return nil
	}
	out := make(map[string]any, len(payload))
	for key, value := range payload {
		out[key] = value
	}
	return out
}

var (
	_ aisearch.Embedder         = (*Embedder)(nil)
	_ aisearch.PagedSource      = PagedSource{}
	_ aisearch.LexicalSearcher  = Lexical{}
	_ aisearch.SemanticSearcher = Semantic{}
	_ aisearch.VectorStore      = (*VectorStore)(nil)
	_ aisearch.Reranker         = Reranker{}
)
