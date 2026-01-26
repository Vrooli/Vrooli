package aisearch

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"prompt-manager/search"
	"prompt-manager/skills"
)

// Service provides AI-powered search with graceful fallback to text search.
type Service struct {
	embedder      *Embedder
	vectorStore   *VectorStore
	skillStore    skills.SkillStore
	searchService *search.Service
	threshold     float64
	reindex       *reindexState
}

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

// NewService creates a new AI search service.
func NewService(
	embedder *Embedder,
	vectorStore *VectorStore,
	skillStore skills.SkillStore,
	searchService *search.Service,
	threshold float64,
) *Service {
	if threshold <= 0 {
		threshold = 0.5
	}
	return &Service{
		embedder:      embedder,
		vectorStore:   vectorStore,
		skillStore:    skillStore,
		searchService: searchService,
		threshold:     threshold,
		reindex:       &reindexState{},
	}
}

// Search performs AI semantic search with fallback to text search.
func (s *Service) Search(ctx context.Context, query string, limit int) (*AISearchResponse, error) {
	if limit <= 0 {
		limit = 5
	}

	// Try AI search first
	vector, err := s.embedder.Embed(ctx, query)
	if err != nil {
		log.Printf("[aisearch] Embedding failed, falling back to text search: %v", err)
		return s.fallbackToTextSearch(ctx, query, limit)
	}

	results, err := s.vectorStore.Search(ctx, vector, limit, s.threshold)
	if err != nil {
		log.Printf("[aisearch] Vector search failed, falling back to text search: %v", err)
		return s.fallbackToTextSearch(ctx, query, limit)
	}

	// Convert to AI search results
	aiResults := make([]AISearchResult, 0, len(results))
	for _, r := range results {
		aiResults = append(aiResults, s.toAISearchResult(r))
	}

	return &AISearchResponse{
		Results: aiResults,
		Total:   len(aiResults),
		Query:   query,
		Method:  "ai",
	}, nil
}

// fallbackToTextSearch uses the existing text search when AI is unavailable.
func (s *Service) fallbackToTextSearch(ctx context.Context, query string, limit int) (*AISearchResponse, error) {
	textResp, err := s.searchService.Search(search.SearchQuery{Query: query})
	if err != nil {
		return nil, fmt.Errorf("text search failed: %w", err)
	}

	// Limit results
	results := textResp.Results
	if len(results) > limit {
		results = results[:limit]
	}

	// Convert to AI search result format
	aiResults := make([]AISearchResult, 0, len(results))
	for _, r := range results {
		aiResults = append(aiResults, AISearchResult{
			ID:           r.ID,
			Name:         r.Name,
			Description:  r.Description,
			Folder:       r.Folder,
			Tags:         r.Tags,
			Modes:        r.Modes,
			Score:        r.Score / 10.0, // Normalize text search score to 0-1 range
			ScorePercent: int(r.Score * 10),
		})
	}

	return &AISearchResponse{
		Results: aiResults,
		Total:   len(aiResults),
		Query:   query,
		Method:  "text",
	}, nil
}

// toAISearchResult converts a vector search result to an AI search result.
func (s *Service) toAISearchResult(r SearchResult) AISearchResult {
	payload := r.Payload

	// Extract fields from payload
	resultID := r.ID
	if payloadSkillID, ok := payload["skill_id"].(string); ok && payloadSkillID != "" {
		resultID = payloadSkillID
	}
	name, _ := payload["name"].(string)
	description, _ := payload["description"].(string)
	folder, _ := payload["folder"].(string)

	var tags []string
	if tagsRaw, ok := payload["tags"].([]interface{}); ok {
		for _, t := range tagsRaw {
			if ts, ok := t.(string); ok {
				tags = append(tags, ts)
			}
		}
	}

	var modes []string
	if modesRaw, ok := payload["modes"].([]interface{}); ok {
		for _, m := range modesRaw {
			if ms, ok := m.(string); ok {
				modes = append(modes, ms)
			}
		}
	}

	// Convert cosine similarity (0-1) to percentage
	scorePercent := int(r.Score * 100)
	if scorePercent > 100 {
		scorePercent = 100
	}

	return AISearchResult{
		ID:           resultID,
		Name:         name,
		Description:  description,
		Folder:       folder,
		Tags:         tags,
		Modes:        modes,
		Score:        r.Score,
		ScorePercent: scorePercent,
	}
}

// GetStatus returns the availability status of the AI search system.
func (s *Service) GetStatus(ctx context.Context) *AvailabilityStatus {
	ollamaAvailable := s.embedder.Available(ctx)
	qdrantAvailable := s.vectorStore.Available(ctx)

	var indexedCount int
	if qdrantAvailable {
		count, err := s.vectorStore.CountPoints(ctx)
		if err == nil {
			indexedCount = count
		}
	}

	available := ollamaAvailable && qdrantAvailable

	var message string
	if !available {
		var missing []string
		if !ollamaAvailable {
			missing = append(missing, "Ollama")
		}
		if !qdrantAvailable {
			missing = append(missing, "Qdrant")
		}
		message = fmt.Sprintf("AI search unavailable: %s not reachable", strings.Join(missing, " and "))
	}

	return &AvailabilityStatus{
		Available:    available,
		Ollama:       ollamaAvailable,
		Qdrant:       qdrantAvailable,
		IndexedCount: indexedCount,
		Message:      message,
	}
}

// Available returns true if AI search is available.
func (s *Service) Available(ctx context.Context) bool {
	return s.embedder.Available(ctx) && s.vectorStore.Available(ctx)
}

// StartReindex begins a singleton reindex job if one is not already running.
// Returns the current status and whether a new job was started.
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

// CancelReindex requests cancellation of the active reindex job.
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

// ReindexStatus returns the current reindex status.
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

func formatReindexTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
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
