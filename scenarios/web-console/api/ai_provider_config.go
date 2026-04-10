// DOC: docs/concepts/ARCHITECTURE.md#file-map
// DOC: docs/internal/STORAGE_AUDIT.md

package main

import (
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
