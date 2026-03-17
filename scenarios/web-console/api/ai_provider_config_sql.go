// DOC: docs/concepts/ARCHITECTURE.md#file-map
// DOC: docs/internal/STORAGE_AUDIT.md
// DOC: docs/internal/SECURITY-POSTURE.md#sql-injection-prevention

package main

import (
	"database/sql"
	"log"
	"sync"
	"time"
)

// [REQ:P1-003a] Provider Configuration Storage — SQLite implementation
// [REQ:P1-003b] Provider Health Dashboard (API support)

// SQLAIConfigStore persists AI provider configuration in SQLite and tracks
// health metrics in memory. Config operations survive restarts; health data
// resets on restart (it reflects runtime provider availability, not durable state).
type SQLAIConfigStore struct {
	db     *sql.DB
	mu     sync.RWMutex
	health map[string]*providerHealthTracker
}

// NewSQLAIConfigStore creates a SQLite-backed AI config store.
// Health trackers are initialized for all configured providers.
func NewSQLAIConfigStore(db *sql.DB) *SQLAIConfigStore {
	store := &SQLAIConfigStore{
		db:     db,
		health: make(map[string]*providerHealthTracker),
	}
	// Initialize health trackers from existing configs
	rows, err := db.Query(`SELECT name FROM ai_provider_configs`)
	if err != nil {
		log.Printf("SQLAIConfigStore: failed to load provider names: %v", err)
		// Fall back to known defaults
		store.health["ollama"] = &providerHealthTracker{}
		store.health["openrouter"] = &providerHealthTracker{}
		return store
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		store.health[name] = &providerHealthTracker{}
	}
	return store
}

// GetConfigs returns all provider configurations sorted by priority.
func (s *SQLAIConfigStore) GetConfigs() []ProviderConfig {
	rows, err := s.db.Query(`SELECT name, enabled, priority, timeout_sec, max_retries FROM ai_provider_configs ORDER BY priority`)
	if err != nil {
		log.Printf("SQLAIConfigStore.GetConfigs: query failed: %v", err)
		return nil
	}
	defer rows.Close()

	var configs []ProviderConfig
	for rows.Next() {
		var c ProviderConfig
		var enabled int
		if err := rows.Scan(&c.Name, &enabled, &c.Priority, &c.TimeoutSec, &c.MaxRetries); err != nil {
			log.Printf("SQLAIConfigStore.GetConfigs: scan failed: %v", err)
			continue
		}
		c.Enabled = enabled != 0
		configs = append(configs, c)
	}
	if configs == nil {
		configs = make([]ProviderConfig, 0)
	}
	return configs
}

// UpdateConfig updates a single provider's configuration in SQLite.
func (s *SQLAIConfigStore) UpdateConfig(name string, enabled bool, priority, timeoutSec, maxRetries int) bool {
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	result, err := s.db.Exec(`
		UPDATE ai_provider_configs
		SET enabled = ?, priority = ?, timeout_sec = ?, max_retries = ?
		WHERE name = ?`,
		enabledInt, priority, timeoutSec, maxRetries, name)
	if err != nil {
		log.Printf("SQLAIConfigStore.UpdateConfig: exec failed: %v", err)
		return false
	}
	n, _ := result.RowsAffected()
	return n > 0
}

// RecordSuccess records a successful provider call (in-memory only).
func (s *SQLAIConfigStore) RecordSuccess(name string, latency time.Duration) {
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

// RecordError records a failed provider call (in-memory only).
func (s *SQLAIConfigStore) RecordError(name string) {
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
func (s *SQLAIConfigStore) GetHealth() []ProviderHealth {
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

// GetProviderTimeout returns the configured timeout for a provider from SQLite.
func (s *SQLAIConfigStore) GetProviderTimeout(name string) time.Duration {
	var timeoutSec int
	err := s.db.QueryRow(`SELECT timeout_sec FROM ai_provider_configs WHERE name = ?`, name).Scan(&timeoutSec)
	if err != nil {
		return defaultProviderTimeout
	}
	return time.Duration(timeoutSec) * time.Second
}

// IsEnabled returns whether a provider is enabled in SQLite.
func (s *SQLAIConfigStore) IsEnabled(name string) bool {
	var enabled int
	err := s.db.QueryRow(`SELECT enabled FROM ai_provider_configs WHERE name = ?`, name).Scan(&enabled)
	if err != nil {
		return false
	}
	return enabled != 0
}
