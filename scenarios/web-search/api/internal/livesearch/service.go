package livesearch

import (
	"context"
	"log"
	"strings"
)

// DefaultLimit is the result count used when a request omits a positive limit.
const DefaultLimit = 10

// degradedBudgetReason is the degraded_reason surfaced when the governor
// declines a live call.
const degradedBudgetReason = "live search budget exhausted; try again shortly"

// Service orchestrates the L0/L1 live-search read path over its injected seams:
//
//	governor.Allow() -> cache.Get() -> client.Search() -> normalize ->
//	cache.Put() -> (optional) synthesizer.Synthesize()
//
// Raw results are always returned (when available); synthesis is additive and
// off unless requested. A synthesis failure never blocks the raw results.
type Service struct {
	client      SearxngClient
	cache       *Cache
	governor    *Governor
	synthesizer Synthesizer
	logger      *log.Logger
}

// Deps wires the service's seams. Client is required; Cache, Governor, and
// Synthesizer are optional (a nil Cache/Governor disables that shield, a nil
// Synthesizer makes synthesis a no-op).
type Deps struct {
	Client      SearxngClient
	Cache       *Cache
	Governor    *Governor
	Synthesizer Synthesizer
	Logger      *log.Logger
}

// NewService constructs the live-search service.
func NewService(d Deps) *Service {
	logger := d.Logger
	if logger == nil {
		logger = log.Default()
	}
	return &Service{
		client:      d.Client,
		cache:       d.Cache,
		governor:    d.Governor,
		synthesizer: d.Synthesizer,
		logger:      logger,
	}
}

// SearchInput is the service-level request.
type SearchInput struct {
	Query      string
	Limit      int
	Synthesize bool
}

// Search runs the L0 live web search and optional L1 synthesis.
func (s *Service) Search(ctx context.Context, in SearchInput) (SearchOutcome, error) {
	query := strings.TrimSpace(in.Query)
	limit := in.Limit
	if limit <= 0 {
		limit = DefaultLimit
	}

	// Cache hit serves without spending budget or touching SearXNG.
	if s.cache != nil {
		if cached, ok := s.cache.Get(query, limit); ok {
			return s.withSynthesis(ctx, query, cached, SearchOutcome{Results: cached, Cached: true}, in.Synthesize), nil
		}
	}

	// Budget governor: on exhaustion, degrade gracefully WITHOUT calling
	// SearXNG. Synthesis is skipped (no results to ground it).
	if s.governor != nil && !s.governor.Allow() {
		return SearchOutcome{
			Results:        nil,
			Degraded:       true,
			DegradedReason: degradedBudgetReason,
		}, nil
	}

	raw, err := s.client.Search(ctx, query, limit)
	if err != nil {
		return SearchOutcome{}, err
	}
	results := normalizeAll(raw)
	if s.cache != nil {
		s.cache.Put(query, limit, results)
	}

	return s.withSynthesis(ctx, query, results, SearchOutcome{Results: results}, in.Synthesize), nil
}

// withSynthesis attaches the optional L1 synthesis to an outcome. Synthesis is
// additive: a failure is logged and the raw results are returned unchanged.
func (s *Service) withSynthesis(ctx context.Context, query string, results []Result, out SearchOutcome, synthesize bool) SearchOutcome {
	if !synthesize || s.synthesizer == nil || len(results) == 0 {
		return out
	}
	syn, err := s.synthesizer.Synthesize(ctx, query, results)
	if err != nil {
		s.logger.Printf("livesearch: synthesis failed (returning raw results): %v", err)
		return out
	}
	out.Synthesis = syn
	return out
}
