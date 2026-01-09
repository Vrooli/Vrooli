// Package recovery provides session state checkpointing for the playwright-driver sidecar.
//
// During recording sessions, the recovery system periodically saves the recorded
// actions to persistent storage. If the sidecar crashes, the UI can offer to
// resume from the last checkpoint.
package recovery

import (
	"os"
	"strconv"
	"time"
)

// Config holds configuration for session recovery.
type Config struct {
	// Enabled determines whether checkpointing is active.
	// Env: BAS_SIDECAR_CHECKPOINT_ENABLED (default: true)
	Enabled bool

	// CheckpointInterval is how often to save checkpoints during recording.
	// Env: BAS_SIDECAR_CHECKPOINT_INTERVAL_MS (default: 2000)
	CheckpointInterval time.Duration

	// Retention is how long to keep old checkpoints before cleanup.
	// Env: BAS_SIDECAR_CHECKPOINT_RETENTION_MS (default: 3600000, 1 hour)
	Retention time.Duration
}

// DefaultConfig returns a Config with all default values.
func DefaultConfig() Config {
	return Config{
		Enabled:            true,
		CheckpointInterval: 2 * time.Second,
		Retention:          1 * time.Hour,
	}
}

// LoadConfig loads configuration from environment variables.
func LoadConfig() Config {
	cfg := DefaultConfig()

	if v := os.Getenv("BAS_SIDECAR_CHECKPOINT_ENABLED"); v != "" {
		cfg.Enabled = parseBool(v, cfg.Enabled)
	}

	if v := os.Getenv("BAS_SIDECAR_CHECKPOINT_INTERVAL_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			cfg.CheckpointInterval = time.Duration(ms) * time.Millisecond
		}
	}

	if v := os.Getenv("BAS_SIDECAR_CHECKPOINT_RETENTION_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			cfg.Retention = time.Duration(ms) * time.Millisecond
		}
	}

	return cfg
}

// parseBool parses a string as a boolean, returning defaultVal on failure.
func parseBool(s string, defaultVal bool) bool {
	switch s {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return defaultVal
	}
}
