// Package health provides health monitoring for the playwright-driver sidecar.
//
// The health monitor polls the sidecar's /health endpoint and broadcasts
// status changes to subscribers. It integrates with the supervisor to
// include restart count and state information.
package health

import (
	"os"
	"strconv"
	"time"
)

// Config holds configuration for the health monitor.
type Config struct {
	// PollInterval is how often to check the sidecar's health.
	// Env: BAS_SIDECAR_HEALTH_POLL_INTERVAL_MS (default: 5000)
	PollInterval time.Duration

	// Timeout is the timeout for each health check request.
	// Env: BAS_SIDECAR_HEALTH_TIMEOUT_MS (default: 2000)
	Timeout time.Duration

	// FailureThreshold is the number of consecutive failures before
	// the sidecar is considered unhealthy.
	// Env: BAS_SIDECAR_HEALTH_FAILURE_THRESHOLD (default: 3)
	FailureThreshold int

	// Debounce is the minimum time between state change broadcasts.
	// This prevents UI flicker from rapid state changes.
	// Env: BAS_SIDECAR_HEALTH_DEBOUNCE_MS (default: 1000)
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

// LoadConfig loads configuration from environment variables.
func LoadConfig() Config {
	cfg := DefaultConfig()

	if v := os.Getenv("BAS_SIDECAR_HEALTH_POLL_INTERVAL_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			cfg.PollInterval = time.Duration(ms) * time.Millisecond
		}
	}

	if v := os.Getenv("BAS_SIDECAR_HEALTH_TIMEOUT_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			cfg.Timeout = time.Duration(ms) * time.Millisecond
		}
	}

	if v := os.Getenv("BAS_SIDECAR_HEALTH_FAILURE_THRESHOLD"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.FailureThreshold = n
		}
	}

	if v := os.Getenv("BAS_SIDECAR_HEALTH_DEBOUNCE_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms >= 0 {
			cfg.Debounce = time.Duration(ms) * time.Millisecond
		}
	}

	return cfg
}
