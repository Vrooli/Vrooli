package ai

import "time"

// ConfigStore abstracts AI provider configuration and health tracking.
// Config operations persist across restarts; health is ephemeral.
type ConfigStore interface {
	GetConfigs() []Config
	UpdateConfig(name string, enabled bool, priority, timeoutSec, maxRetries int) bool
	RecordSuccess(name string, latency time.Duration)
	RecordError(name string)
	GetHealth() []Health
	GetProviderTimeout(name string) time.Duration
	IsEnabled(name string) bool
}
