// Package services provides business logic orchestration.
// This file implements AI-powered skill suggestions based on conversation context.
package services

import (
	"agent-inbox/persistence"
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
)

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
	// Fast path: if the first query is already high-confidence, skip extra network calls.
	var allResults []pmSearchResult
	firstResults, err := s.searchPromptManager(ctx, queries[0], s.cfg.MaxSuggestions)
	if err != nil {
		log.Printf("skill suggest: search failed for query %q: %v", queries[0], err)
	} else {
		allResults = append(allResults, firstResults...)
	}

	if len(queries) > 1 && !hasHighConfidenceResults(firstResults, minInt(3, s.cfg.MaxSuggestions), 0.85) {
		var mu sync.Mutex
		var wg sync.WaitGroup

		for _, query := range queries[1:] {
			query := query
			wg.Add(1)
			go func() {
				defer wg.Done()
				results, err := s.searchPromptManager(ctx, query, s.cfg.MaxSuggestions)
				if err != nil {
					log.Printf("skill suggest: search failed for query %q: %v", query, err)
					return
				}
				mu.Lock()
				allResults = append(allResults, results...)
				mu.Unlock()
			}()
		}

		wg.Wait()
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

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func hasHighConfidenceResults(results []pmSearchResult, minCount int, threshold float64) bool {
	if minCount <= 0 {
		return false
	}

	count := 0
	for _, result := range results {
		if result.Score >= threshold || result.ScorePercent >= int(threshold*100) {
			count++
			if count >= minCount {
				return true
			}
		}
	}

	return false
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
		query = strings.TrimPrefix(query, "Current input: ")
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
