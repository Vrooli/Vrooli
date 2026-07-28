package retrieval

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"signal-inbox/internal/clock"
)

const (
	defaultLimit = 50
	corpusLimit  = 100000
)

type Service struct {
	repo     Repository
	clock    clock.Clock
	semantic SemanticSearch
}

func NewService(repo Repository, clk clock.Clock, semantic ...SemanticSearch) *Service {
	service := &Service{repo: repo, clock: clk}
	if len(semantic) > 0 {
		service.semantic = semantic[0]
	}
	return service
}

func (s *Service) Search(ctx context.Context, filter Filter) ([]Result, error) {
	page, err := s.SearchPage(ctx, filter)
	return page.Results, err
}

func (s *Service) SearchPage(ctx context.Context, filter Filter) (Page, error) {
	if filter.Limit <= 0 {
		filter.Limit = defaultLimit
	}
	if filter.PageAfter != "" {
		capturedAt, signalID, err := parsePageAfter(filter.PageAfter)
		if err != nil {
			return Page{}, err
		}
		filter.PageAfterCapturedAt, filter.PageAfterSignalID = &capturedAt, signalID
	}
	if err := s.repo.Rebuild(ctx); err != nil {
		return Page{}, err
	}
	limit := filter.Limit
	filter.Limit++
	var results []Result
	var err error
	if s.semantic != nil && strings.TrimSpace(filter.Text) != "" {
		results, err = s.semanticSearch(ctx, filter)
	} else {
		results, err = s.repo.Search(ctx, filter)
	}
	if err != nil {
		return Page{}, err
	}
	page := Page{Results: results}
	if len(page.Results) > limit {
		page.Results = page.Results[:limit]
		last := page.Results[len(page.Results)-1].Signal
		page.NextPageAfter = encodePageAfter(last.CapturedAt, last.ID)
	}
	return page, nil
}

func encodePageAfter(capturedAt time.Time, signalID string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(capturedAt.UTC().Format(time.RFC3339Nano) + "\n" + signalID))
}

func parsePageAfter(value string) (time.Time, string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invalid page_after cursor")
	}
	capturedAt, signalID, ok := strings.Cut(string(raw), "\n")
	if !ok || signalID == "" {
		return time.Time{}, "", fmt.Errorf("invalid page_after cursor")
	}
	parsed, err := time.Parse(time.RFC3339Nano, capturedAt)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invalid page_after cursor")
	}
	return parsed.UTC(), signalID, nil
}

func (s *Service) semanticSearch(ctx context.Context, filter Filter) ([]Result, error) {
	// First select the full immutable corpus for indexing. Category and
	// disposition are intentionally absent from this query: they are ambient
	// presentation state, never index eligibility.
	corpus, err := s.repo.Search(ctx, Filter{Limit: corpusLimit})
	if err != nil {
		return nil, err
	}
	hits, semanticErr := s.semantic.Search(ctx, filter.Text, corpus, filter.Limit)
	if semanticErr != nil {
		// Exact FTS remains an honest degraded path when ai-gateway or Qdrant is
		// unavailable. It does not claim a semantic result.
		return s.repo.Search(ctx, filter)
	}
	// Apply structured constraints after semantic ranking. The filtered result
	// set is read from the journal projection, never from category/disposition
	// state in the vector index.
	structured := filter
	structured.Text = ""
	structured.Limit = corpusLimit
	candidates, err := s.repo.Search(ctx, structured)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]Result, len(candidates))
	for _, candidate := range candidates {
		byID[candidate.Signal.ID] = candidate
	}
	results := make([]Result, 0, min(filter.Limit, len(hits)))
	for _, hit := range hits {
		candidate, ok := byID[hit.SignalID]
		if !ok {
			continue
		}
		candidate.Score = hit.Score
		results = append(results, candidate)
		if len(results) == filter.Limit {
			break
		}
	}
	return results, nil
}

func (s *Service) Ambient(ctx context.Context, categoryID string, budget int) ([]Result, error) {
	if budget <= 0 {
		budget = defaultLimit
	}
	return s.repo.Ambient(ctx, categoryID, budget, s.clock.Now().UTC())
}

func (s *Service) Coverage(ctx context.Context) (indexed, journal int, err error) {
	if err = s.repo.Rebuild(ctx); err != nil {
		return 0, 0, err
	}
	if indexed, err = s.repo.IndexedCount(ctx); err != nil {
		return 0, 0, err
	}
	journal, err = s.repo.JournalCount(ctx)
	return indexed, journal, err
}
