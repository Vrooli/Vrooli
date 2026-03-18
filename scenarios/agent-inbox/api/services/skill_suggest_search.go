// Package services provides business logic orchestration.
// This file implements search, merge, and cache operations for the skill suggestion service.
package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"
)

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
		suggestions = append(suggestions, SuggestedSkill(r))
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
