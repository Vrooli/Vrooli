package main

import (
	"net/http"
	"sync"
	"time"
)

// [REQ:P1-003a] Provider Configuration Storage
// [REQ:P1-003b] Provider Health Dashboard (API support)
//
// Provider configuration is stored in-memory with per-provider settings
// for priority, timeout, and max retries. Health status is tracked from
// actual provider calls.

// ProviderConfig holds the configuration for a single AI provider.
type ProviderConfig struct {
	Name       string `json:"name"`
	Enabled    bool   `json:"enabled"`
	Priority   int    `json:"priority"` // lower = tried first
	TimeoutSec int    `json:"timeout_sec"`
	MaxRetries int    `json:"max_retries"`
}

// ProviderHealth tracks the health status of a single AI provider.
type ProviderHealth struct {
	Name         string  `json:"name"`
	Available    bool    `json:"available"`
	LastCheck    string  `json:"last_check,omitempty"`
	LastLatency  string  `json:"last_latency,omitempty"`
	ErrorCount   int64   `json:"error_count"`
	SuccessCount int64   `json:"success_count"`
	ErrorRate    float64 `json:"error_rate"`
}

// AIProviderConfigStore manages provider configuration and health in memory.
type AIProviderConfigStore struct {
	mu      sync.RWMutex
	configs map[string]*ProviderConfig
	health  map[string]*providerHealthTracker
}

type providerHealthTracker struct {
	available    bool
	lastCheck    time.Time
	lastLatency  time.Duration
	errorCount   int64
	successCount int64
}

// NewAIProviderConfigStore creates a config store with default provider settings.
func NewAIProviderConfigStore() *AIProviderConfigStore {
	store := &AIProviderConfigStore{
		configs: map[string]*ProviderConfig{
			"ollama": {
				Name:       "ollama",
				Enabled:    true,
				Priority:   1,
				TimeoutSec: 30,
				MaxRetries: 0,
			},
			"openrouter": {
				Name:       "openrouter",
				Enabled:    true,
				Priority:   2,
				TimeoutSec: 30,
				MaxRetries: 0,
			},
		},
		health: map[string]*providerHealthTracker{
			"ollama":     {},
			"openrouter": {},
		},
	}
	return store
}

// GetConfigs returns all provider configurations sorted by priority.
func (s *AIProviderConfigStore) GetConfigs() []ProviderConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]ProviderConfig, 0, len(s.configs))
	for _, c := range s.configs {
		result = append(result, *c)
	}
	// Sort by priority ascending
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Priority < result[i].Priority {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result
}

// UpdateConfig updates a single provider's configuration.
func (s *AIProviderConfigStore) UpdateConfig(name string, enabled bool, priority, timeoutSec, maxRetries int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.configs[name]
	if !ok {
		return false
	}
	c.Enabled = enabled
	c.Priority = priority
	c.TimeoutSec = timeoutSec
	c.MaxRetries = maxRetries
	return true
}

// RecordSuccess records a successful provider call.
func (s *AIProviderConfigStore) RecordSuccess(name string, latency time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	h, ok := s.health[name]
	if !ok {
		return
	}
	h.available = true
	h.lastCheck = time.Now()
	h.lastLatency = latency
	h.successCount++
}

// RecordError records a failed provider call.
func (s *AIProviderConfigStore) RecordError(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	h, ok := s.health[name]
	if !ok {
		return
	}
	h.available = false
	h.lastCheck = time.Now()
	h.errorCount++
}

// GetHealth returns health status for all providers.
func (s *AIProviderConfigStore) GetHealth() []ProviderHealth {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]ProviderHealth, 0, len(s.health))
	for name, h := range s.health {
		total := h.successCount + h.errorCount
		var errorRate float64
		if total > 0 {
			errorRate = float64(h.errorCount) / float64(total)
		}
		ph := ProviderHealth{
			Name:         name,
			Available:    h.available,
			ErrorCount:   h.errorCount,
			SuccessCount: h.successCount,
			ErrorRate:    errorRate,
		}
		if !h.lastCheck.IsZero() {
			ph.LastCheck = h.lastCheck.UTC().Format(time.RFC3339)
			ph.LastLatency = h.lastLatency.Truncate(time.Millisecond).String()
		}
		result = append(result, ph)
	}
	return result
}

// GetProviderTimeout returns the configured timeout for a provider.
func (s *AIProviderConfigStore) GetProviderTimeout(name string) time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.configs[name]
	if !ok {
		return defaultProviderTimeout
	}
	return time.Duration(c.TimeoutSec) * time.Second
}

// IsEnabled returns whether a provider is enabled.
func (s *AIProviderConfigStore) IsEnabled(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.configs[name]
	if !ok {
		return false
	}
	return c.Enabled
}

// AIProviderConfigResponse wraps config + health for the API response.
type AIProviderConfigResponse struct {
	Providers []ProviderConfig `json:"providers"`
	Health    []ProviderHealth `json:"health"`
}

// ProviderConfigUpdateRequest is the JSON body for updating a provider config.
type ProviderConfigUpdateRequest struct {
	Name       string `json:"name"`
	Enabled    *bool  `json:"enabled,omitempty"`
	Priority   *int   `json:"priority,omitempty"`
	TimeoutSec *int   `json:"timeout_sec,omitempty"`
	MaxRetries *int   `json:"max_retries,omitempty"`
}

// handleGetAIConfig returns provider configuration and health.
// GET /api/v1/ai/config
// [REQ:P1-003a] Provider Configuration Storage
// [REQ:P1-003b] Provider Health Dashboard
func (s *Server) handleGetAIConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, AIProviderConfigResponse{
		Providers: s.aiConfig.GetConfigs(),
		Health:    s.aiConfig.GetHealth(),
	})
}

// handleUpdateAIConfig updates a provider's configuration.
// PUT /api/v1/ai/config
// [REQ:P1-003a] Provider Configuration Storage
func (s *Server) handleUpdateAIConfig(w http.ResponseWriter, r *http.Request) {
	var req ProviderConfigUpdateRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if req.Name == "" {
		writeCatalogError(w, "invalid_body", "Provider name is required")
		return
	}

	// Read current config and apply partial updates
	configs := s.aiConfig.GetConfigs()
	var current *ProviderConfig
	for i := range configs {
		if configs[i].Name == req.Name {
			current = &configs[i]
			break
		}
	}
	if current == nil {
		writeCatalogError(w, "invalid_body", "Unknown provider: "+req.Name)
		return
	}

	enabled := current.Enabled
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	priority := current.Priority
	if req.Priority != nil {
		priority = *req.Priority
	}
	timeoutSec := current.TimeoutSec
	if req.TimeoutSec != nil {
		if *req.TimeoutSec < 1 || *req.TimeoutSec > 120 {
			writeCatalogError(w, "invalid_body", "Timeout must be between 1 and 120 seconds")
			return
		}
		timeoutSec = *req.TimeoutSec
	}
	maxRetries := current.MaxRetries
	if req.MaxRetries != nil {
		if *req.MaxRetries < 0 || *req.MaxRetries > 5 {
			writeCatalogError(w, "invalid_body", "Max retries must be between 0 and 5")
			return
		}
		maxRetries = *req.MaxRetries
	}

	s.aiConfig.UpdateConfig(req.Name, enabled, priority, timeoutSec, maxRetries)

	writeJSON(w, http.StatusOK, AIProviderConfigResponse{
		Providers: s.aiConfig.GetConfigs(),
		Health:    s.aiConfig.GetHealth(),
	})
}

// handleGetAIHealth returns provider health status only.
// GET /api/v1/ai/health
// [REQ:P1-003b] Provider Health Dashboard
func (s *Server) handleGetAIHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.aiConfig.GetHealth())
}
