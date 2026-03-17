package main

import "time"

// DOC: docs/internal/STORAGE_AUDIT.md
// DOC: docs/internal/SEAMS.md#axis-5-storage-abstraction

// ShortcutStore abstracts shortcut profile storage. Implementations may be
// in-memory (for tests) or SQLite-backed (for production persistence).
type ShortcutStore interface {
	List() []*ShortcutProfile
	Get(id string) (*ShortcutProfile, bool)
	Upsert(id, scope, name string, shortcuts []ShortcutEntry) *ShortcutProfile
	Delete(id string) bool
	Effective() []ShortcutEntry
}

// AIConfigStore abstracts AI provider configuration and health tracking.
// Config operations (GetConfigs, UpdateConfig, GetProviderTimeout, IsEnabled)
// persist across restarts. Health tracking (RecordSuccess, RecordError, GetHealth)
// is ephemeral and resets on restart.
type AIConfigStore interface {
	GetConfigs() []ProviderConfig
	UpdateConfig(name string, enabled bool, priority, timeoutSec, maxRetries int) bool
	RecordSuccess(name string, latency time.Duration)
	RecordError(name string)
	GetHealth() []ProviderHealth
	GetProviderTimeout(name string) time.Duration
	IsEnabled(name string) bool
}
