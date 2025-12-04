// Package smokeconfig provides shared configuration types for UI smoke testing.
// This package exists to break import cycles between structure and smoke packages.
package smokeconfig

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// UISmokeConfig holds UI smoke test settings loaded from .vrooli/testing.json.
type UISmokeConfig struct {
	// Enabled controls whether UI smoke testing runs.
	Enabled bool

	// TimeoutMs overrides the default UI smoke timeout.
	TimeoutMs int64

	// HandshakeTimeoutMs overrides the default handshake timeout.
	HandshakeTimeoutMs int64

	// HandshakeSignals lists custom JavaScript signals to check for handshake completion.
	// If empty, the default signals are used.
	HandshakeSignals []string
}

// rawTestingJSON represents the structure of .vrooli/testing.json for UI smoke config.
type rawTestingJSON struct {
	Structure struct {
		UISmoke struct {
			Enabled            *bool    `json:"enabled"`
			TimeoutMs          int64    `json:"timeout_ms"`
			HandshakeTimeoutMs int64    `json:"handshake_timeout_ms"`
			HandshakeSignals   []string `json:"handshake_signals"`
		} `json:"ui_smoke"`
	} `json:"structure"`
}

// LoadUISmokeConfig loads UI smoke configuration from .vrooli/testing.json.
// If the file doesn't exist or is invalid, it returns default configuration.
func LoadUISmokeConfig(scenarioDir string) UISmokeConfig {
	cfg := DefaultUISmokeConfig()

	configPath := filepath.Join(scenarioDir, ".vrooli", "testing.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg
		}
		return cfg
	}

	var raw rawTestingJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return cfg
	}

	if raw.Structure.UISmoke.Enabled != nil {
		cfg.Enabled = *raw.Structure.UISmoke.Enabled
	}
	if raw.Structure.UISmoke.TimeoutMs > 0 {
		cfg.TimeoutMs = raw.Structure.UISmoke.TimeoutMs
	}
	if raw.Structure.UISmoke.HandshakeTimeoutMs > 0 {
		cfg.HandshakeTimeoutMs = raw.Structure.UISmoke.HandshakeTimeoutMs
	}
	if len(raw.Structure.UISmoke.HandshakeSignals) > 0 {
		cfg.HandshakeSignals = raw.Structure.UISmoke.HandshakeSignals
	}

	return cfg
}

// DefaultUISmokeConfig returns the default UI smoke configuration.
func DefaultUISmokeConfig() UISmokeConfig {
	return UISmokeConfig{
		Enabled: true,
	}
}
