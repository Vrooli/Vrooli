package ai

import (
	"context"
	"sync"
	"time"
)

// MemConfigStore manages provider configuration and health in memory.
type MemConfigStore struct {
	mu      sync.RWMutex
	configs map[string]*Config
	health  map[string]*healthTracker
}

// NewMemConfigStore creates a config store with default provider settings.
func NewMemConfigStore() *MemConfigStore {
	return &MemConfigStore{
		configs: map[string]*Config{
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
		health: map[string]*healthTracker{
			"ollama":     {},
			"openrouter": {},
		},
	}
}

func (s *MemConfigStore) GetConfigs(_ context.Context) []Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Config, 0, len(s.configs))
	for _, c := range s.configs {
		result = append(result, *c)
	}
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Priority < result[i].Priority {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result
}

func (s *MemConfigStore) UpdateConfig(_ context.Context, name string, enabled bool, priority, timeoutSec, maxRetries int) bool {
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

func (s *MemConfigStore) RecordSuccess(_ context.Context, name string, latency time.Duration) {
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

func (s *MemConfigStore) RecordError(_ context.Context, name string) {
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

func (s *MemConfigStore) GetHealth(_ context.Context) []Health {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Health, 0, len(s.health))
	for name, h := range s.health {
		result = append(result, h.snapshot(name))
	}
	return result
}

func (s *MemConfigStore) GetProviderTimeout(_ context.Context, name string) time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.configs[name]
	if !ok {
		return DefaultProviderTimeout
	}
	return time.Duration(c.TimeoutSec) * time.Second
}

func (s *MemConfigStore) IsEnabled(_ context.Context, name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.configs[name]
	if !ok {
		return false
	}
	return c.Enabled
}
