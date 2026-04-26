package aisearch

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"
)

// TextSearcher is the fallback surface used when the embedder or vector store
// is unavailable. Implementations should perform a simple, cheap lookup (e.g.
// substring match) across backlog items and initiatives and return
// already-scored results. A nil TextSearcher causes fallback to return an
// empty response with Fallback = FallbackUnavailable.
type TextSearcher interface {
	Search(ctx context.Context, query string, entity EntityType, limit int, filters SearchFilters) ([]AISearchResult, error)
}

// Service orchestrates embedding, vector upsert/search, and reindex lifecycle
// for backlog items and initiatives.
type Service struct {
	embedder         *Embedder
	backlogStore     *VectorStore
	initiativeStore  *VectorStore
	backlogReader    BacklogReader
	initiativeReader InitiativeReader
	textSearcher     TextSearcher
	threshold        float64
	reindex          *reindexState
}

// NewService creates a new AI search service. Any of backlogStore,
// initiativeStore, backlogReader, initiativeReader may be nil to disable
// that surface; the service degrades gracefully.
func NewService(
	embedder *Embedder,
	backlogStore *VectorStore,
	initiativeStore *VectorStore,
	backlogReader BacklogReader,
	initiativeReader InitiativeReader,
	threshold float64,
) *Service {
	if threshold <= 0 {
		threshold = 0.5
	}
	return &Service{
		embedder:         embedder,
		backlogStore:     backlogStore,
		initiativeStore:  initiativeStore,
		backlogReader:    backlogReader,
		initiativeReader: initiativeReader,
		threshold:        threshold,
		reindex:          &reindexState{},
	}
}

// SetTextSearcher wires a fallback text-search implementation. Safe to call
// after construction.
func (s *Service) SetTextSearcher(t TextSearcher) {
	s.textSearcher = t
}

// Search performs a semantic search over the requested entity (backlog,
// initiative, or both). On embedder/vector-store failure it falls back to
// text search when a TextSearcher is configured, or to an empty response
// marked FallbackUnavailable otherwise.
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
		return nil, fmt.Errorf("invalid entity %q: must be backlog, initiative, or both", req.Entity)
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
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

	// Initialized non-nil so zero-match searches serialize to `"results":[]`, not
	// `"results":null`. The JSON contract is "always an array" — a nil slice here
	// becomes JSON null and crashes clients that do `results.length`.
	results := make([]AISearchResult, 0)
	var searchErr error

	if entity == EntityBacklog || entity == EntityBoth {
		if s.backlogStore != nil {
			r, err := s.searchStore(ctx, s.backlogStore, EntityBacklog, vector, limit, threshold)
			if err != nil {
				searchErr = err
			} else {
				results = append(results, r...)
			}
		}
	}
	if entity == EntityInitiative || entity == EntityBoth {
		if s.initiativeStore != nil {
			r, err := s.searchStore(ctx, s.initiativeStore, EntityInitiative, vector, limit, threshold)
			if err != nil {
				searchErr = err
			} else {
				results = append(results, r...)
			}
		}
	}

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

func (s *Service) searchStore(ctx context.Context, store *VectorStore, entity EntityType, vector []float64, limit int, threshold float64) ([]AISearchResult, error) {
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
	initiative := strings.TrimSpace(f.Initiative)
	target := strings.TrimSpace(f.TargetScenario)

	out := results[:0]
	for _, r := range results {
		if r.Payload != nil {
			archived, _ := r.Payload["archived"].(bool)
			if archived && !f.IncludeArchived {
				continue
			}
			if len(statusSet) > 0 {
				status, _ := r.Payload["status"].(string)
				if !statusSet[status] {
					continue
				}
			}
			if len(kindSet) > 0 && r.Entity == EntityBacklog {
				kind, _ := r.Payload["kind"].(string)
				if !kindSet[kind] {
					continue
				}
			}
			if initiative != "" && r.Entity == EntityBacklog {
				rInit, _ := r.Payload["initiative"].(string)
				if rInit != initiative {
					continue
				}
			}
			if target != "" && r.Entity == EntityBacklog {
				if !payloadTargetsScenario(r.Payload, target) {
					continue
				}
			}
		}
		out = append(out, r)
	}
	return out
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
	var indexedBacklog, indexedInitiatives int
	if s.backlogStore != nil && s.backlogStore.Available(ctx) {
		qdrantOK = true
		if n, err := s.backlogStore.CountPoints(ctx); err == nil {
			indexedBacklog = n
		}
	}
	if s.initiativeStore != nil && s.initiativeStore.Available(ctx) {
		qdrantOK = true
		if n, err := s.initiativeStore.CountPoints(ctx); err == nil {
			indexedInitiatives = n
		}
	}

	var onDiskBacklog, onDiskInitiatives int
	if s.backlogReader != nil {
		if items, err := s.backlogReader.LoadAll(); err == nil {
			onDiskBacklog = len(items)
		}
	}
	if s.initiativeReader != nil {
		if inits, err := s.initiativeReader.List(); err == nil {
			onDiskInitiatives = len(inits)
		}
	}

	status := &AvailabilityStatus{
		Available:          ollamaOK && qdrantOK,
		Ollama:             ollamaOK,
		Qdrant:             qdrantOK,
		IndexedBacklog:     indexedBacklog,
		IndexedInitiatives: indexedInitiatives,
		OnDiskBacklog:      onDiskBacklog,
		OnDiskInitiatives:  onDiskInitiatives,
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
	if s.initiativeStore != nil && !s.initiativeStore.Available(ctx) {
		return false
	}
	return true
}

// NeedsReindex reports whether the vector index count diverges from the
// on-disk entity count for either collection. Returns (needs, totalIndexed,
// totalOnDisk, err).
func (s *Service) NeedsReindex(ctx context.Context) (bool, int, int, error) {
	var totalIndexed, totalDisk int

	if s.backlogStore != nil {
		n, err := s.backlogStore.CountPoints(ctx)
		if err != nil {
			return false, 0, 0, fmt.Errorf("count backlog points: %w", err)
		}
		totalIndexed += n
	}
	if s.initiativeStore != nil {
		n, err := s.initiativeStore.CountPoints(ctx)
		if err != nil {
			return false, 0, 0, fmt.Errorf("count initiative points: %w", err)
		}
		totalIndexed += n
	}

	if s.backlogReader != nil {
		items, err := s.backlogReader.LoadAll()
		if err != nil {
			return false, 0, 0, fmt.Errorf("load backlog items: %w", err)
		}
		totalDisk += len(items)
	}
	if s.initiativeReader != nil {
		inits, err := s.initiativeReader.List()
		if err != nil {
			return false, 0, 0, fmt.Errorf("list initiatives: %w", err)
		}
		totalDisk += len(inits)
	}

	return totalIndexed != totalDisk, totalIndexed, totalDisk, nil
}

// --- Reindex lifecycle ---

type reindexState struct {
	mu         sync.Mutex
	running    bool
	canceled   bool
	startedAt  time.Time
	finishedAt time.Time
	indexed    int
	skipped    int
	errors     int
	total      int
	message    string
	lastError  string
	cancel     context.CancelFunc
}

// StartReindex begins a singleton reindex job. If one is already running,
// returns its current status with started=false. Otherwise spawns a goroutine
// and returns the initial status with started=true.
func (s *Service) StartReindex() (ReindexStatus, bool) {
	s.reindex.mu.Lock()
	defer s.reindex.mu.Unlock()

	if s.reindex.running {
		return s.reindex.statusLocked(), false
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.reindex.running = true
	s.reindex.canceled = false
	s.reindex.startedAt = time.Now()
	s.reindex.finishedAt = time.Time{}
	s.reindex.indexed = 0
	s.reindex.skipped = 0
	s.reindex.errors = 0
	s.reindex.total = 0
	s.reindex.message = "Reindex started"
	s.reindex.lastError = ""
	s.reindex.cancel = cancel

	go s.runReindexJob(ctx)

	return s.reindex.statusLocked(), true
}

// CancelReindex requests cancellation of an in-flight reindex. No-op if no
// job is running.
func (s *Service) CancelReindex() ReindexStatus {
	s.reindex.mu.Lock()
	defer s.reindex.mu.Unlock()

	if s.reindex.running && s.reindex.cancel != nil {
		s.reindex.canceled = true
		s.reindex.message = "Reindex cancel requested"
		s.reindex.cancel()
	}
	return s.reindex.statusLocked()
}

// ReindexStatus returns a snapshot of the current reindex job state.
func (s *Service) ReindexStatus() ReindexStatus {
	s.reindex.mu.Lock()
	defer s.reindex.mu.Unlock()
	return s.reindex.statusLocked()
}

func (s *Service) runReindexJob(ctx context.Context) {
	resp, err := s.reindexAllWithProgress(ctx, func(indexed, skipped, errorsCount int) {
		s.reindex.mu.Lock()
		s.reindex.indexed = indexed
		s.reindex.skipped = skipped
		s.reindex.errors = errorsCount
		s.reindex.mu.Unlock()
	}, func(total int) {
		s.reindex.mu.Lock()
		s.reindex.total = total
		s.reindex.mu.Unlock()
	})

	s.reindex.mu.Lock()
	defer s.reindex.mu.Unlock()

	if resp != nil {
		s.reindex.indexed = resp.Indexed
		s.reindex.skipped = resp.Skipped
		s.reindex.errors = resp.Errors
		s.reindex.message = resp.Message
	}

	if err != nil {
		if errors.Is(err, context.Canceled) {
			s.reindex.canceled = true
			if s.reindex.message == "" {
				s.reindex.message = "Reindex canceled"
			}
		} else {
			s.reindex.lastError = err.Error()
		}
	}

	s.reindex.running = false
	s.reindex.finishedAt = time.Now()
	s.reindex.cancel = nil
}

func (rs *reindexState) statusLocked() ReindexStatus {
	status := ReindexStatus{
		Running:    rs.running,
		StartedAt:  formatReindexTime(rs.startedAt),
		FinishedAt: formatReindexTime(rs.finishedAt),
		Indexed:    rs.indexed,
		Skipped:    rs.skipped,
		Errors:     rs.errors,
		Total:      rs.total,
		Message:    rs.message,
		Canceled:   rs.canceled,
	}
	if rs.lastError != "" {
		status.Error = rs.lastError
	}
	return status
}

func formatReindexTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// StartPeriodicSync runs a background goroutine that reconciles index drift
// on the given interval. Exits when ctx is canceled.
func (s *Service) StartPeriodicSync(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				needs, indexed, disk, err := s.NeedsReindex(ctx)
				if err != nil {
					slog.Warn("[aisearch] periodic sync check failed", "err", err)
					continue
				}
				if needs {
					slog.Info("[aisearch] periodic sync: index drift detected, reindexing",
						"indexed", indexed, "on_disk", disk)
					s.StartReindex()
				}
			}
		}
	}()
}
