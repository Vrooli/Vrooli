package aisearch

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"
)

// TextSearcher is the fallback surface used when the embedder or vector store
// is unavailable. Implementations should perform a simple, cheap lookup (e.g.
// substring match) across backlog items and goals and return
// already-scored results. A nil TextSearcher causes fallback to return an
// empty response with Fallback = FallbackUnavailable.
type TextSearcher interface {
	Search(ctx context.Context, query string, entity EntityType, limit int, filters SearchFilters) ([]AISearchResult, error)
}

// Service orchestrates semantic search and write-through indexing hooks. The
// reconcile lifecycle (drift detection, plan/apply, periodic sync) lives in
// Reconciler and SyncLoop — Service does not own that decision boundary.
type Service struct {
	embedder      Embedder
	backlogStore  VectorStore
	goalStore     VectorStore
	recordStore   VectorStore
	backlogReader BacklogReader
	goalReader    GoalReader
	textSearcher  TextSearcher
	threshold     float64
}

// SetRecordStore wires a VectorStore for the records collection. Optional;
// when nil, IndexRecord/DeleteRecord return a configured-off error and the
// service degrades silently for that surface. Wired post-construction so the
// NewService signature stays stable.
func (s *Service) SetRecordStore(store VectorStore) {
	s.recordStore = store
}

// NewService creates a new AI search service. Any of backlogStore,
// goalStore, backlogReader, goalReader may be nil to disable
// that surface; the service degrades gracefully.
func NewService(
	embedder Embedder,
	backlogStore VectorStore,
	goalStore VectorStore,
	backlogReader BacklogReader,
	goalReader GoalReader,
	threshold float64,
) *Service {
	if threshold <= 0 {
		threshold = 0.5
	}
	return &Service{
		embedder:      embedder,
		backlogStore:  backlogStore,
		goalStore:     goalStore,
		backlogReader: backlogReader,
		goalReader:    goalReader,
		threshold:     threshold,
	}
}

// SetTextSearcher wires a fallback text-search implementation. Safe to call
// after construction.
func (s *Service) SetTextSearcher(t TextSearcher) {
	s.textSearcher = t
}

// Search performs a semantic search over the requested entity (backlog,
// goal, record, or both). EntityBoth fans out to every wired store
// — records are included when a recordStore is configured. On embedder /
// vector-store failure it falls back to text search when a TextSearcher
// is configured, or to an empty response marked FallbackUnavailable
// otherwise.
func (s *Service) Search(ctx context.Context, req AISearchRequest) (*AISearchResponse, error) {
	start := time.Now()
	query := strings.TrimSpace(req.Query)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}

	entity := req.Entity
	if entity == "" {
		entity = EntityBoth
	}
	if !entity.Valid() {
		return nil, fmt.Errorf("invalid entity %q: must be backlog, goal, record, or both", req.Entity)
	}

	limit := normalizeLimit(req.Limit)
	threshold := req.Threshold
	if threshold <= 0 {
		threshold = s.threshold
	}

	if s.embedder == nil {
		return s.fallback(ctx, req, entity, limit, start), nil
	}
	vector, err := s.embedder.Embed(ctx, query)
	if err != nil {
		slog.Warn("[aisearch] embed failed, falling back", "err", err)
		return s.fallback(ctx, req, entity, limit, start), nil
	}

	results, searchErr := s.searchStores(ctx, entity, vector, limit, threshold)
	if searchErr != nil && len(results) == 0 {
		slog.Warn("[aisearch] vector search failed, falling back", "err", searchErr)
		return s.fallback(ctx, req, entity, limit, start), nil
	}

	results = applyFilters(results, normalizeFilters(req.Filters))
	sortByScoreDesc(results)
	if len(results) > limit {
		results = results[:limit]
	}

	return &AISearchResponse{
		Results:   results,
		Total:     len(results),
		Query:     query,
		Entity:    entity,
		Fallback:  FallbackNone,
		LatencyMs: time.Since(start).Milliseconds(),
	}, nil
}

// SimilarTo re-embeds the target's canonical indexed text and searches every
// collection, including archived work. It returns Degraded rather than using
// Search's text fallback when Ollama or Qdrant is unavailable.
func (s *Service) SimilarTo(ctx context.Context, target SimilarTarget, limit int) (*SimilarResponse, error) {
	if target.Entity != EntityBacklog && target.Entity != EntityGoal {
		return nil, fmt.Errorf("similar target entity must be backlog or goal")
	}
	if s.embedder == nil || s.backlogStore == nil || s.goalStore == nil {
		return &SimilarResponse{Results: []AISearchResult{}, Degraded: true}, nil
	}
	var text, self string
	if target.Entity == EntityBacklog {
		item, err := s.backlogReader.LoadItem(target.BacklogKind, target.Name)
		if err != nil {
			return nil, err
		}
		text, self = composeBacklogText(item), backlogPointID(target.BacklogKind, target.Name)
	} else {
		goal, err := s.goalReader.Get(target.Name)
		if err != nil {
			return nil, err
		}
		text, self = composeGoalText(*goal), goalPointID(target.Name)
	}
	vector, err := s.embedder.Embed(ctx, text)
	if err != nil {
		return &SimilarResponse{Results: []AISearchResult{}, Degraded: true}, nil
	}
	results, err := s.searchStores(ctx, EntityBoth, vector, normalizeLimit(limit), s.threshold)
	if err != nil {
		return &SimilarResponse{Results: []AISearchResult{}, Degraded: true}, nil
	}
	filtered := make([]AISearchResult, 0, len(results))
	for _, result := range results {
		if result.ID == self {
			continue
		}
		if target.Entity == EntityBacklog && result.Entity == EntityBacklog && result.ID == target.Name {
			continue
		}
		if target.Entity == EntityGoal && result.Entity == EntityGoal && result.ID == target.Name {
			continue
		}
		filtered = append(filtered, result)
	}
	sortByScoreDesc(filtered)
	if limit > 0 && len(filtered) > normalizeLimit(limit) {
		filtered = filtered[:normalizeLimit(limit)]
	}
	return &SimilarResponse{Results: filtered}, nil
}

// normalizeLimit clamps a requested result limit into the supported range,
// applying the default of 20 when unset and capping at 100.
func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}

// searchStores fans the embedded query out to every store wired for the
// requested entity, concatenating their results. It returns the last
// per-store error encountered (if any); callers decide whether to fall back
// based on whether any results were produced.
func (s *Service) searchStores(ctx context.Context, entity EntityType, vector []float64, limit int, threshold float64) ([]AISearchResult, error) {
	// Initialized non-nil so zero-match searches serialize to `"results":[]`, not
	// `"results":null`. The JSON contract is "always an array" — a nil slice here
	// becomes JSON null and crashes clients that do `results.length`.
	results := make([]AISearchResult, 0)
	var searchErr error

	stores := []struct {
		entity EntityType
		store  VectorStore
	}{
		{EntityBacklog, s.backlogStore},
		{EntityGoal, s.goalStore},
		{EntityRecord, s.recordStore},
	}
	for _, st := range stores {
		if entity != st.entity && entity != EntityBoth {
			continue
		}
		if st.store == nil {
			continue
		}
		r, err := s.searchStore(ctx, st.store, st.entity, vector, limit, threshold)
		if err != nil {
			searchErr = err
		} else {
			results = append(results, r...)
		}
	}
	return results, searchErr
}

func (s *Service) searchStore(ctx context.Context, store VectorStore, entity EntityType, vector []float64, limit int, threshold float64) ([]AISearchResult, error) {
	raw, err := store.Search(ctx, vector, limit, threshold)
	if err != nil {
		return nil, err
	}
	out := make([]AISearchResult, 0, len(raw))
	for _, r := range raw {
		out = append(out, toAISearchResult(entity, r))
	}
	return out, nil
}

func (s *Service) fallback(ctx context.Context, req AISearchRequest, entity EntityType, limit int, start time.Time) *AISearchResponse {
	if s.textSearcher == nil {
		return &AISearchResponse{
			Results:   []AISearchResult{},
			Total:     0,
			Query:     req.Query,
			Entity:    entity,
			Fallback:  FallbackUnavailable,
			LatencyMs: time.Since(start).Milliseconds(),
		}
	}
	results, err := s.textSearcher.Search(ctx, req.Query, entity, limit, normalizeFilters(req.Filters))
	if err != nil {
		slog.Warn("[aisearch] text search fallback failed", "err", err)
		results = []AISearchResult{}
	}
	// Upstream may also hand back a typed nil; normalize before serialization
	// so the JSON always contains an array.
	if results == nil {
		results = []AISearchResult{}
	}
	return &AISearchResponse{
		Results:   results,
		Total:     len(results),
		Query:     req.Query,
		Entity:    entity,
		Fallback:  FallbackTextSearch,
		LatencyMs: time.Since(start).Milliseconds(),
	}
}

// toAISearchResult converts a raw Qdrant SearchResult into the API shape,
// annotating it with the entity type it came from.
func toAISearchResult(entity EntityType, r SearchResult) AISearchResult {
	percent := int(r.Score * 100)
	if percent > 100 {
		percent = 100
	}
	if percent < 0 {
		percent = 0
	}
	id := r.ID
	if r.Payload != nil {
		if name, ok := r.Payload["name"].(string); ok && name != "" {
			id = name
		}
	}
	return AISearchResult{
		Entity:       entity,
		ID:           id,
		Score:        r.Score,
		ScorePercent: percent,
		Payload:      r.Payload,
	}
}

// applyFilters removes results that do not match any of the provided filters.
// Empty filter fields match everything.
func applyFilters(results []AISearchResult, f SearchFilters) []AISearchResult {
	if len(results) == 0 {
		return results
	}
	statusSet := toStringSet(f.Status)
	kindSet := toStringSet(f.Kind)
	goal := strings.TrimSpace(f.Goal)
	target := strings.TrimSpace(f.TargetScenario)

	out := results[:0]
	for _, r := range results {
		if r.Payload == nil || resultMatchesFilters(r, f, statusSet, kindSet, goal, target) {
			out = append(out, r)
		}
	}
	return out
}

// resultMatchesFilters reports whether a result with a non-nil payload passes
// every active filter predicate. Empty/zero filter fields match everything.
func resultMatchesFilters(r AISearchResult, f SearchFilters, statusSet, kindSet map[string]bool, goal, target string) bool {
	archived, _ := r.Payload["archived"].(bool)
	if archived && !f.IncludeArchived {
		return false
	}
	return matchesStatusFilter(r, statusSet) &&
		matchesKindFilter(r, kindSet) &&
		matchesGoalFilter(r, goal) &&
		matchesTargetFilter(r, target)
}

// matchesStatusFilter applies the status filter. Records don't have a status,
// so the check is skipped for them.
func matchesStatusFilter(r AISearchResult, statusSet map[string]bool) bool {
	if len(statusSet) == 0 || r.Entity == EntityRecord {
		return true
	}
	status, _ := r.Payload["status"].(string)
	return statusSet[status]
}

// matchesKindFilter applies the kind filter, which covers both backlog items
// and records (same enum).
func matchesKindFilter(r AISearchResult, kindSet map[string]bool) bool {
	if len(kindSet) == 0 || (r.Entity != EntityBacklog && r.Entity != EntityRecord) {
		return true
	}
	kind, _ := r.Payload["kind"].(string)
	return kindSet[kind]
}

// matchesGoalFilter narrows backlog items by their goal. Backlog items are
// associated through goal scopes rather than a persisted item field, so this
// filter is currently meaningful only to goal payloads.
func matchesGoalFilter(r AISearchResult, goal string) bool {
	if goal == "" || r.Entity != EntityGoal {
		return true
	}
	name, _ := r.Payload["name"].(string)
	return name == goal
}

// matchesTargetFilter narrows results by target scenario. Backlog items carry
// a target_scenarios array; records carry a single "scenario" field, but the
// same --target-scenario CLI flag should narrow both.
func matchesTargetFilter(r AISearchResult, target string) bool {
	if target == "" {
		return true
	}
	switch r.Entity {
	case EntityBacklog:
		return payloadTargetsScenario(r.Payload, target)
	case EntityRecord:
		rs, _ := r.Payload["scenario"].(string)
		return rs == target
	}
	return true
}

// payloadTargetsScenario reports whether the indexed target_scenarios array
// contains the given scenario. Tolerant of either []string or []interface{}
// since Qdrant payloads round-trip through map[string]interface{}.
func payloadTargetsScenario(payload map[string]interface{}, scenario string) bool {
	raw, ok := payload["target_scenarios"]
	if !ok {
		return false
	}
	switch v := raw.(type) {
	case []string:
		for _, s := range v {
			if s == scenario {
				return true
			}
		}
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok && s == scenario {
				return true
			}
		}
	}
	return false
}

// normalizeFilters applies implicit defaults that encode product rules in one
// place. Today: when Kind is exactly ["fix"] and IncludeArchived is unset by
// the caller, default it to true — fix history work always wants archived
// items by default. Callers can still pass IncludeArchived=false explicitly
// to opt out.
func normalizeFilters(f SearchFilters) SearchFilters {
	if !f.IncludeArchived && len(f.Kind) == 1 && strings.TrimSpace(f.Kind[0]) == "fix" {
		f.IncludeArchived = true
	}
	return f
}

func toStringSet(ss []string) map[string]bool {
	if len(ss) == 0 {
		return nil
	}
	set := make(map[string]bool, len(ss))
	for _, s := range ss {
		set[s] = true
	}
	return set
}

func sortByScoreDesc(results []AISearchResult) {
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
}

// GetStatus returns the current availability of the AI search stack.
func (s *Service) GetStatus(ctx context.Context) *AvailabilityStatus {
	var ollamaOK bool
	if s.embedder != nil {
		ollamaOK = s.embedder.Available(ctx)
	}

	var qdrantOK bool
	var indexedBacklog, indexedGoals int
	if s.backlogStore != nil && s.backlogStore.Available(ctx) {
		qdrantOK = true
		if n, err := s.backlogStore.CountPoints(ctx); err == nil {
			indexedBacklog = n
		}
	}
	if s.goalStore != nil && s.goalStore.Available(ctx) {
		qdrantOK = true
		if n, err := s.goalStore.CountPoints(ctx); err == nil {
			indexedGoals = n
		}
	}

	var onDiskBacklog, onDiskGoals int
	if s.backlogReader != nil {
		if items, err := s.backlogReader.LoadAll(); err == nil {
			onDiskBacklog = len(items)
		}
	}
	if s.goalReader != nil {
		if goalsList, err := s.goalReader.List(); err == nil {
			onDiskGoals = len(goalsList)
		}
	}

	status := &AvailabilityStatus{
		Available:      ollamaOK && qdrantOK,
		Ollama:         ollamaOK,
		Qdrant:         qdrantOK,
		IndexedBacklog: indexedBacklog,
		IndexedGoals:   indexedGoals,
		OnDiskBacklog:  onDiskBacklog,
		OnDiskGoals:    onDiskGoals,
	}
	if !status.Available {
		var missing []string
		if !ollamaOK {
			missing = append(missing, "Ollama")
		}
		if !qdrantOK {
			missing = append(missing, "Qdrant")
		}
		status.Message = fmt.Sprintf("AI search unavailable: %s not reachable", strings.Join(missing, " and "))
	}
	return status
}

// Available reports whether AI search can be performed right now. Useful for
// cheap gating decisions; for detailed diagnostics use GetStatus.
func (s *Service) Available(ctx context.Context) bool {
	if s.embedder == nil || !s.embedder.Available(ctx) {
		return false
	}
	if s.backlogStore != nil && !s.backlogStore.Available(ctx) {
		return false
	}
	if s.goalStore != nil && !s.goalStore.Available(ctx) {
		return false
	}
	return true
}
