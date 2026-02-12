// Package services provides business logic orchestration.
// This file implements AI-powered skill suggestions based on conversation context.
package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"agent-inbox/config"
	"agent-inbox/integrations"
	"agent-inbox/persistence"
)

// SuggestRequest is the input for skill suggestion.
type SuggestRequest struct {
	ChatID          string   `json:"chatId,omitempty"`
	InputText       string   `json:"inputText,omitempty"`
	ExcludeSkillIDs []string `json:"excludeSkillIds,omitempty"`
}

// SuggestedSkill is a single skill suggestion with relevance score.
type SuggestedSkill struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Tags         []string `json:"tags,omitempty"`
	Modes        []string `json:"modes,omitempty"`
	Score        float64  `json:"score"`
	ScorePercent int      `json:"scorePercent"`
}

// SuggestResponse is the output of skill suggestion.
type SuggestResponse struct {
	Suggestions []SuggestedSkill `json:"suggestions"`
	QueryCount  int              `json:"queryCount"`
	Method      string           `json:"method"` // "ollama", "keyword", "direct"
}

// suggestCacheEntry is a cached suggestion result with TTL.
type suggestCacheEntry struct {
	response  *SuggestResponse
	expiresAt time.Time
}

// pmSearchResult represents a single result from prompt-manager AI search.
type pmSearchResult struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Tags         []string `json:"tags"`
	Modes        []string `json:"modes"`
	Score        float64  `json:"score"`
	ScorePercent int      `json:"scorePercent"`
}

// pmSearchResponse is the response from prompt-manager's AI search endpoint.
type pmSearchResponse struct {
	Results []pmSearchResult `json:"results"`
	Total   int              `json:"total"`
}

// SkillSuggestService provides AI-powered skill suggestions.
type SkillSuggestService struct {
	ollamaClient     *integrations.OllamaClient
	promptManagerURL string
	httpClient       *http.Client
	cfg              *config.SkillSuggestConfig
	cache            map[string]*suggestCacheEntry
	cacheMu          sync.RWMutex
}

// NewSkillSuggestService creates a new skill suggest service.
func NewSkillSuggestService(
	ollamaClient *integrations.OllamaClient,
	promptManagerURL string,
	cfg *config.SkillSuggestConfig,
) *SkillSuggestService {
	return &SkillSuggestService{
		ollamaClient:     ollamaClient,
		promptManagerURL: promptManagerURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		cfg:   cfg,
		cache: make(map[string]*suggestCacheEntry),
	}
}

// SuggestSkills generates skill suggestions based on conversation context.
func (s *SkillSuggestService) SuggestSkills(ctx context.Context, repo *persistence.Repository, req *SuggestRequest) (*SuggestResponse, error) {
	if !s.cfg.Enabled {
		return &SuggestResponse{Suggestions: []SuggestedSkill{}}, nil
	}

	// Build context summary from input text + chat messages
	summary := s.buildContextSummary(ctx, repo, req)
	if summary == "" {
		return &SuggestResponse{Suggestions: []SuggestedSkill{}}, nil
	}

	// Check cache
	cacheKey := s.cacheKey(summary, req.ExcludeSkillIDs)
	if cached := s.getCached(cacheKey); cached != nil {
		return cached, nil
	}

	// Generate search queries
	queries, method := s.generateSearchQueries(ctx, summary)
	if len(queries) == 0 {
		return &SuggestResponse{Suggestions: []SuggestedSkill{}, Method: method}, nil
	}

	// Search prompt-manager for each query
	var allResults []pmSearchResult
	for _, query := range queries {
		results, err := s.searchPromptManager(ctx, query, s.cfg.MaxSuggestions)
		if err != nil {
			log.Printf("skill suggest: search failed for query %q: %v", query, err)
			continue
		}
		allResults = append(allResults, results...)
	}

	// Merge, deduplicate, and rank
	suggestions := s.mergeResults(allResults, req.ExcludeSkillIDs, s.cfg.MaxSuggestions)

	resp := &SuggestResponse{
		Suggestions: suggestions,
		QueryCount:  len(queries),
		Method:      method,
	}

	// Cache the result
	s.setCached(cacheKey, resp)

	return resp, nil
}

// buildContextSummary constructs a text summary from the request.
func (s *SkillSuggestService) buildContextSummary(ctx context.Context, repo *persistence.Repository, req *SuggestRequest) string {
	var parts []string

	// Always include input text if present
	if req.InputText != "" {
		parts = append(parts, "Current input: "+req.InputText)
	}

	// If chat ID provided, include recent messages
	if req.ChatID != "" && repo != nil {
		messages, err := repo.GetMessages(ctx, req.ChatID)
		if err == nil && len(messages) > 0 {
			// Take last N messages
			start := 0
			if len(messages) > s.cfg.MaxMessages {
				start = len(messages) - s.cfg.MaxMessages
			}

			for _, msg := range messages[start:] {
				content := msg.Content
				if len(content) > s.cfg.MaxContentLen {
					content = content[:s.cfg.MaxContentLen] + "..."
				}
				parts = append(parts, fmt.Sprintf("%s: %s", msg.Role, content))
			}
		}
	}

	return strings.Join(parts, "\n")
}

// generateSearchQueries uses Ollama to generate targeted search queries.
// Falls back to keyword extraction if Ollama is unavailable.
func (s *SkillSuggestService) generateSearchQueries(ctx context.Context, summary string) ([]string, string) {
	// Short context: use directly as 1-2 queries
	if len(summary) < 100 {
		// Extract just the meaningful part for short inputs
		query := summary
		if strings.HasPrefix(query, "Current input: ") {
			query = strings.TrimPrefix(query, "Current input: ")
		}
		return []string{query}, "direct"
	}

	// Try Ollama first
	if s.ollamaClient != nil && s.ollamaClient.IsAvailable(ctx) {
		queries, err := s.generateQueriesViaOllama(ctx, summary)
		if err == nil && len(queries) > 0 {
			return queries, "ollama"
		}
		log.Printf("skill suggest: Ollama query generation failed: %v", err)
	}

	// Fallback: extract keywords from the summary
	return s.extractKeywords(summary), "keyword"
}

// generateQueriesViaOllama calls Ollama to generate search queries.
func (s *SkillSuggestService) generateQueriesViaOllama(ctx context.Context, summary string) ([]string, error) {
	// Truncate summary if very long
	if len(summary) > 2000 {
		summary = summary[:2000] + "..."
	}

	prompt := fmt.Sprintf(`You are a skill-matching assistant. Given a conversation, generate exactly %d short search queries that would find relevant knowledge/methodology skills.

Each query should target a DIFFERENT aspect:
1. The primary TOPIC (e.g., "React state management")
2. The TECHNIQUE or approach (e.g., "debugging performance")
3. The DOMAIN or broader context (e.g., "API design patterns")

Rules:
- 2-5 words each
- Focus on concepts/methodologies, not specific code
- Return ONLY %d queries, one per line, no numbering

Conversation:
%s

Queries:`, s.cfg.QueryCount, s.cfg.QueryCount, summary)

	result, err := s.ollamaClient.GenerateText(ctx, s.cfg.Model, prompt, 100)
	if err != nil {
		return nil, err
	}

	// Parse lines into queries
	var queries []string
	for _, line := range strings.Split(result, "\n") {
		line = strings.TrimSpace(line)
		// Skip empty lines and numbered prefixes
		line = strings.TrimLeft(line, "0123456789.-) ")
		line = strings.TrimSpace(line)
		if line != "" && len(line) >= 3 && len(line) <= 100 {
			queries = append(queries, line)
		}
		if len(queries) >= s.cfg.QueryCount {
			break
		}
	}

	return queries, nil
}

// extractKeywords provides a simple fallback when Ollama is unavailable.
func (s *SkillSuggestService) extractKeywords(summary string) []string {
	// Take first meaningful chunk and last meaningful chunk as separate queries
	lines := strings.Split(summary, "\n")

	var queries []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Strip role prefix
		if idx := strings.Index(line, ": "); idx >= 0 && idx < 20 {
			line = line[idx+2:]
		}
		line = strings.TrimSpace(line)
		if line != "" {
			// Take first 5 words as a query
			words := strings.Fields(line)
			if len(words) > 5 {
				words = words[:5]
			}
			queries = append(queries, strings.Join(words, " "))
			if len(queries) >= 2 {
				break
			}
		}
	}

	return queries
}

// searchPromptManager calls prompt-manager's AI search endpoint.
func (s *SkillSuggestService) searchPromptManager(ctx context.Context, query string, limit int) ([]pmSearchResult, error) {
	if s.promptManagerURL == "" {
		return nil, fmt.Errorf("prompt-manager URL not configured")
	}

	reqBody, err := json.Marshal(map[string]interface{}{
		"query": query,
		"limit": limit,
		"tag":   "skill",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/search/ai", s.promptManagerURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("prompt-manager returned %d: %s", resp.StatusCode, string(body))
	}

	var searchResp pmSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	return searchResp.Results, nil
}

// mergeResults deduplicates by skill ID, ranks by max score, and caps at max.
func (s *SkillSuggestService) mergeResults(allResults []pmSearchResult, excludeIDs []string, max int) []SuggestedSkill {
	excludeSet := make(map[string]bool, len(excludeIDs))
	for _, id := range excludeIDs {
		excludeSet[id] = true
	}

	// Deduplicate: keep highest score per ID
	bestByID := make(map[string]pmSearchResult)
	for _, r := range allResults {
		if excludeSet[r.ID] {
			continue
		}
		if existing, ok := bestByID[r.ID]; !ok || r.Score > existing.Score {
			bestByID[r.ID] = r
		}
	}

	// Convert to slice and sort by score descending
	suggestions := make([]SuggestedSkill, 0, len(bestByID))
	for _, r := range bestByID {
		suggestions = append(suggestions, SuggestedSkill{
			ID:           r.ID,
			Name:         r.Name,
			Description:  r.Description,
			Tags:         r.Tags,
			Modes:        r.Modes,
			Score:        r.Score,
			ScorePercent: r.ScorePercent,
		})
	}

	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Score > suggestions[j].Score
	})

	if len(suggestions) > max {
		suggestions = suggestions[:max]
	}

	return suggestions
}

// cacheKey creates a deterministic cache key from the context summary and excludes.
func (s *SkillSuggestService) cacheKey(summary string, excludeIDs []string) string {
	h := sha256.New()
	h.Write([]byte(summary))
	for _, id := range excludeIDs {
		h.Write([]byte("|"))
		h.Write([]byte(id))
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

// getCached returns a cached response if still valid.
func (s *SkillSuggestService) getCached(key string) *SuggestResponse {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	entry, ok := s.cache[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil
	}
	return entry.response
}

// setCached stores a response in the cache.
func (s *SkillSuggestService) setCached(key string, resp *SuggestResponse) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	s.cache[key] = &suggestCacheEntry{
		response:  resp,
		expiresAt: time.Now().Add(time.Duration(s.cfg.CacheTTLSeconds) * time.Second),
	}

	// Prune expired entries periodically (if cache grows)
	if len(s.cache) > 100 {
		now := time.Now()
		for k, v := range s.cache {
			if now.After(v.expiresAt) {
				delete(s.cache, k)
			}
		}
	}
}
