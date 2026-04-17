// Package services provides business logic orchestration.
// This file defines types for the skill suggestion service.
package services

import (
	"agent-inbox/config"
	"agent-inbox/integrations"
	"net/http"
	"sync"
	"time"
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
