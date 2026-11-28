// Package health provides health monitoring for the playwright-driver sidecar.
//
// The health monitor polls the sidecar's /health endpoint and broadcasts
// status changes to subscribers. It integrates with the supervisor to
// include restart count and state information.
package health

import (
	"time"
)

// Config holds configuration for the health monitor.
type Config struct {
	// PollInterval is how often to check the sidecar's health.
	PollInterval time.Duration

	// Timeout is the timeout for each health check request.
	Timeout time.Duration

	// FailureThreshold is the number of consecutive failures before
	// the sidecar is considered unhealthy.
	FailureThreshold int

	// Debounce is the minimum time between state change broadcasts.
	// This prevents UI flicker from rapid state changes.
	Debounce time.Duration
}

// DefaultConfig returns a Config with all default values.
func DefaultConfig() Config {
	return Config{
		PollInterval:     5 * time.Second,
		Timeout:          2 * time.Second,
		FailureThreshold: 3,
		Debounce:         1 * time.Second,
	}
}

func LoadConfig(settings map[string]any) Config {
	cfg := DefaultConfig()
	cfg.PollInterval = milliseconds(settings, "sidecar_health_poll_interval_ms", cfg.PollInterval)
	cfg.Timeout = milliseconds(settings, "sidecar_health_timeout_ms", cfg.Timeout)
	cfg.FailureThreshold = positiveInt(settings, "sidecar_health_failure_threshold", cfg.FailureThreshold)
	cfg.Debounce = milliseconds(settings, "sidecar_health_debounce_ms", cfg.Debounce)
	return cfg
}

func milliseconds(settings map[string]any, key string, fallback time.Duration) time.Duration {
	if value, ok := settings[key].(float64); ok && value >= 0 {
		return time.Duration(value) * time.Millisecond
	}
	return fallback
}

func positiveInt(settings map[string]any, key string, fallback int) int {
	if value, ok := settings[key].(float64); ok && value > 0 {
		return int(value)
	}
	return fallback
}
