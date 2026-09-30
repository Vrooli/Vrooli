// Package recovery provides session state checkpointing for the playwright-driver sidecar.
//
// During recording sessions, the recovery system periodically saves the recorded
// actions to persistent storage. If the sidecar crashes, the UI can offer to
// resume from the last checkpoint.
package recovery

import "time"

// Config holds configuration for session recovery.
type Config struct {
	// Enabled determines whether checkpointing is active.
	Enabled bool

	// CheckpointInterval is how often to save checkpoints during recording.
	CheckpointInterval time.Duration

	// Retention is how long to keep old checkpoints before cleanup.
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

func LoadConfig(settings map[string]any) Config {
	cfg := DefaultConfig()
	if value, ok := settings["sidecar_checkpoint_enabled"].(bool); ok {
		cfg.Enabled = value
	}
	if value, ok := settings["sidecar_checkpoint_interval_ms"].(float64); ok && value > 0 {
		cfg.CheckpointInterval = time.Duration(value) * time.Millisecond
	}
	if value, ok := settings["sidecar_checkpoint_retention_ms"].(float64); ok && value > 0 {
		cfg.Retention = time.Duration(value) * time.Millisecond
	}
	return cfg
}
