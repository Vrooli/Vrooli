package ai

import (
	"database/sql"
	"log"
	"sync"
	"time"
)

// SQLConfigStore persists AI provider configuration in SQLite and tracks
// health metrics in memory.
type SQLConfigStore struct {
	db     *sql.DB
	mu     sync.RWMutex
	health map[string]*healthTracker
}

// NewSQLConfigStore creates a SQLite-backed AI config store. Health
// trackers are initialized for all configured providers.
func NewSQLConfigStore(db *sql.DB) *SQLConfigStore {
	store := &SQLConfigStore{
		db:     db,
		health: make(map[string]*healthTracker),
	}
	rows, err := db.Query(`SELECT name FROM ai_provider_configs`)
	if err != nil {
		log.Printf("SQLConfigStore: failed to load provider names: %v", err)
		store.health["ollama"] = &healthTracker{}
		store.health["openrouter"] = &healthTracker{}
		return store
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		store.health[name] = &healthTracker{}
	}
	return store
}

func (s *SQLConfigStore) GetConfigs() []Config {
	rows, err := s.db.Query(`SELECT name, enabled, priority, timeout_sec, max_retries FROM ai_provider_configs ORDER BY priority`)
	if err != nil {
		log.Printf("SQLConfigStore.GetConfigs: query failed: %v", err)
		return nil
	}
	defer rows.Close()

	var configs []Config
	for rows.Next() {
		var c Config
		var enabled int
		if err := rows.Scan(&c.Name, &enabled, &c.Priority, &c.TimeoutSec, &c.MaxRetries); err != nil {
			log.Printf("SQLConfigStore.GetConfigs: scan failed: %v", err)
			continue
		}
		c.Enabled = enabled != 0
		configs = append(configs, c)
	}
	if configs == nil {
		configs = make([]Config, 0)
	}
	return configs
}

func (s *SQLConfigStore) UpdateConfig(name string, enabled bool, priority, timeoutSec, maxRetries int) bool {
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
		log.Printf("SQLConfigStore.UpdateConfig: exec failed: %v", err)
		return false
	}
	n, _ := result.RowsAffected()
	return n > 0
}

func (s *SQLConfigStore) RecordSuccess(name string, latency time.Duration) {
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

func (s *SQLConfigStore) RecordError(name string) {
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

func (s *SQLConfigStore) GetHealth() []Health {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Health, 0, len(s.health))
	for name, h := range s.health {
		result = append(result, h.snapshot(name))
	}
	return result
}

func (s *SQLConfigStore) GetProviderTimeout(name string) time.Duration {
	var timeoutSec int
	err := s.db.QueryRow(`SELECT timeout_sec FROM ai_provider_configs WHERE name = ?`, name).Scan(&timeoutSec)
	if err != nil {
		return DefaultProviderTimeout
	}
	return time.Duration(timeoutSec) * time.Second
}

func (s *SQLConfigStore) IsEnabled(name string) bool {
	var enabled int
	err := s.db.QueryRow(`SELECT enabled FROM ai_provider_configs WHERE name = ?`, name).Scan(&enabled)
	if err != nil {
		return false
	}
	return enabled != 0
}
