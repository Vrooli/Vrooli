package aisearch

import (
	"context"
	"sort"
	"strings"
	"time"
)

// StatusReport is the truthful status of the live catalog projection. Both
// Generation and LastIndexedAt are derived from the catalog's own data, so
// rereading status without a source change returns the same values.
type StatusReport struct {
	Available     bool
	IndexedCount  int
	Generation    string
	LastIndexedAt time.Time
}

// SearchHit is one ranked result over the projected corpus.
type SearchHit struct {
	ID         string
	Kind       string
	Title      string
	Snippet    string
	FollowUp   string
	Score      float64
	Freshness  string
	Historical bool
	Metadata   map[string]any
}

// SearchResponse carries the hits plus the corpus generation the ranking read,
// so a caller can tell whether results moved because content moved.
type SearchResponse struct {
	Hits           []SearchHit
	Generation     string
	MaterializedAt time.Time
}

// Service ranks records from a live catalog snapshot. It is the degraded,
// no-vector fallback: when the shared ai-go/search index is unavailable it
// still answers over the authoritative corpus instead of returning nothing.
type Service struct {
	source *StoreSource
}

// NewService builds a service over a catalog source.
func NewService(source *StoreSource) *Service { return &Service{source: source} }

// Snapshot loads the current corpus. Exposed so a handler can report status
// without duplicating projection logic.
func (s *Service) Snapshot(ctx context.Context) (*Snapshot, error) {
	return s.source.Load(ctx)
}

// Status reports the truthful corpus state.
func (s *Service) Status(ctx context.Context) (StatusReport, error) {
	snapshot, err := s.source.Load(ctx)
	if err != nil {
		return StatusReport{Available: false}, err
	}
	return StatusReport{
		Available:     true,
		IndexedCount:  snapshot.IndexedCount(),
		Generation:    snapshot.Generation,
		LastIndexedAt: snapshot.MaterializedAt,
	}, nil
}

// Search ranks the live corpus for query. An empty query returns the first
// limit records in stable id order (a browse read). A query whose tokens are
// mostly unmatched returns no hits rather than an accidental weak match.
func (s *Service) Search(ctx context.Context, query string, limit int) (*SearchResponse, error) {
	snapshot, err := s.source.Load(ctx)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	tokens := tokenize(query)
	if len(tokens) == 0 {
		hits := make([]SearchHit, 0, limit)
		for _, r := range snapshot.Records {
			if len(hits) >= limit {
				break
			}
			hits = append(hits, hitFor(r, 0))
		}
		return &SearchResponse{Hits: hits, Generation: snapshot.Generation, MaterializedAt: snapshot.MaterializedAt}, nil
	}
	type scored struct {
		hit   SearchHit
		score float64
	}
	var ranked []scored
	for _, r := range snapshot.Records {
		score := lexicalScore(tokens, r)
		if score <= 0 {
			continue
		}
		ranked = append(ranked, scored{hit: hitFor(r, score), score: score})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].hit.ID < ranked[j].hit.ID
	})
	hits := make([]SearchHit, 0, limit)
	for _, r := range ranked {
		if len(hits) >= limit {
			break
		}
		hits = append(hits, r.hit)
	}
	return &SearchResponse{Hits: hits, Generation: snapshot.Generation, MaterializedAt: snapshot.MaterializedAt}, nil
}

func hitFor(r Record, score float64) SearchHit {
	return SearchHit{
		ID:         r.ID,
		Kind:       r.Kind,
		Title:      r.Title,
		Snippet:    r.Snippet,
		FollowUp:   r.FollowUp,
		Score:      score,
		Freshness:  r.Freshness,
		Historical: r.Historical,
		Metadata:   r.Metadata,
	}
}

// lexicalScore is the fraction of query tokens present in the record text,
// with a title match weighted double. It is intentionally simple and
// deterministic; the shared vector index owns quality ranking.
func lexicalScore(tokens []string, r Record) float64 {
	title := strings.ToLower(r.Title)
	text := strings.ToLower(r.Title + " " + r.Snippet + " " + r.Body)
	matched := 0.0
	for _, token := range tokens {
		switch {
		case strings.Contains(title, token):
			matched += 2
		case strings.Contains(text, token):
			matched += 1
		}
	}
	// Require at least half the tokens to land somewhere, so gibberish with a
	// single accidental token does not surface as a hit.
	if matched < float64(len(tokens)) {
		return 0
	}
	return matched / float64(len(tokens)*2)
}

func tokenize(query string) []string {
	fields := strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if len(f) < 2 {
			continue
		}
		out = append(out, f)
	}
	return out
}
