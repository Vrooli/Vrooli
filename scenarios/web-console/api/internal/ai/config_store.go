package ai

import (
	"context"
	"time"
)

// ConfigStore abstracts AI provider configuration and health tracking.
// Config operations persist across restarts; health is ephemeral.
type ConfigStore interface {
	GetConfigs(ctx context.Context) []Config
	UpdateConfig(ctx context.Context, name string, enabled bool, priority, timeoutSec, maxRetries int) bool
	RecordSuccess(ctx context.Context, name string, latency time.Duration)
	RecordError(ctx context.Context, name string)
	GetHealth(ctx context.Context) []Health
	GetProviderTimeout(ctx context.Context, name string) time.Duration
	IsEnabled(ctx context.Context, name string) bool
}
