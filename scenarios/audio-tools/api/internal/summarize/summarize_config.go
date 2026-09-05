package summarize

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"audio-tools/internal/envx"
	"audio-tools/internal/logx"
)

// configLogger is the package-level logx.Logger used by the rare cleanup
// fallback path in SaveSummarizeConfig. Production leaves it at
// logx.Std{}; tests use SetConfigLogger to capture the line.
var configLogger logx.Logger = logx.Std{}

// SetConfigLogger overrides the package config logger and returns the
// previous value so tests can restore via t.Cleanup.
func SetConfigLogger(l logx.Logger) logx.Logger {
	prev := configLogger
	configLogger = l
	return prev
}

const (
	MinSummarizeTimeoutSeconds     = 15
	DefaultSummarizeTimeoutSeconds = 120
	MaxSummarizeTimeoutSeconds     = 300
)

// SummarizeConfig is the operator-facing summarization config.
type SummarizeConfig struct {
	Enabled        bool
	CharThreshold  int
	Level          string
	Model          string
	TimeoutSeconds int
}

// SummarizeConfigPatch is the optional-update variant for SummarizeConfig.
type SummarizeConfigPatch struct {
	Enabled        *bool
	CharThreshold  *int
	Level          *string
	Model          *string
	TimeoutSeconds *int
}

// DefaultSummarizeConfig returns the default TTS summarization config,
// reading the model selection from the process environment via the
// canonical envx seam. Callers that need to inject a fake environment
// (deterministic tests, multi-tenant contexts) use
// DefaultSummarizeConfigWith and pass their own envx.Reader.
func DefaultSummarizeConfig() SummarizeConfig {
	return DefaultSummarizeConfigWith(envx.OS{})
}

// DefaultSummarizeConfigWith returns the default config but lets the
// caller substitute the env reader. Production wires envx.OS{}; tests
// wire mocks.FakeEnv to assert which keys the config reads.
func DefaultSummarizeConfigWith(env envx.Reader) SummarizeConfig {
	model := CoerceUnsafeStoredModel(env.Get("WC_TTS_SUMMARIZE_MODEL"), nil).Model
	return SummarizeConfig{
		Enabled:        true,
		CharThreshold:  500,
		Level:          "moderate",
		Model:          model,
		TimeoutSeconds: DefaultSummarizeTimeoutSeconds,
	}
}

// Apply merges non-nil patch fields into base, returning the updated config.
func (p SummarizeConfigPatch) Apply(base SummarizeConfig) SummarizeConfig {
	if p.Enabled != nil {
		base.Enabled = *p.Enabled
	}
	if p.CharThreshold != nil {
		base.CharThreshold = *p.CharThreshold
	}
	if p.Level != nil {
		base.Level = *p.Level
	}
	if p.Model != nil {
		if strings.TrimSpace(*p.Model) == "" {
			base.Model = DefaultSummarizeModel
		} else {
			base.Model = strings.TrimSpace(*p.Model)
		}
	}
	if p.TimeoutSeconds != nil {
		base.TimeoutSeconds = *p.TimeoutSeconds
	}
	return base
}

// LoadSummarizeConfig reads config from JSON file. Returns defaults if file missing.
func LoadSummarizeConfig(path string) (SummarizeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultSummarizeConfig(), nil
		}
		return DefaultSummarizeConfig(), fmt.Errorf("read tts summarize config: %w", err)
	}
	var raw struct {
		Enabled        bool   `json:"enabled"`
		CharThreshold  int    `json:"charThreshold"`
		Level          string `json:"level"`
		Model          string `json:"model"`
		TimeoutSeconds int    `json:"timeoutSeconds"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return DefaultSummarizeConfig(), fmt.Errorf("parse tts summarize config: %w", err)
	}
	cfg := SummarizeConfig{
		Enabled:        raw.Enabled,
		CharThreshold:  raw.CharThreshold,
		Level:          raw.Level,
		Model:          raw.Model,
		TimeoutSeconds: raw.TimeoutSeconds,
	}
	if cfg.CharThreshold <= 0 {
		cfg.CharThreshold = 500
	}
	if cfg.Level == "" {
		cfg.Level = "moderate"
	}
	if cfg.Model == "" {
		cfg.Model = DefaultSummarizeConfig().Model
	}
	cfg.Model = CoerceUnsafeStoredModel(cfg.Model, nil).Model
	// Clamp to a realistic floor so stale configs do not silently break the
	// feature after a model or timeout change.
	if cfg.TimeoutSeconds < MinSummarizeTimeoutSeconds {
		cfg.TimeoutSeconds = DefaultSummarizeTimeoutSeconds
	}
	return cfg, nil
}

// SaveSummarizeConfig writes config to JSON file atomically.
func SaveSummarizeConfig(path string, cfg SummarizeConfig) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	wire := struct {
		Enabled        bool   `json:"enabled"`
		CharThreshold  int    `json:"charThreshold"`
		Level          string `json:"level"`
		Model          string `json:"model"`
		TimeoutSeconds int    `json:"timeoutSeconds"`
	}{
		Enabled:        cfg.Enabled,
		CharThreshold:  cfg.CharThreshold,
		Level:          cfg.Level,
		Model:          cfg.Model,
		TimeoutSeconds: cfg.TimeoutSeconds,
	}
	data, err := json.MarshalIndent(wire, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal tts summarize config: %w", err)
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		if rmErr := os.Remove(tmp); rmErr != nil {
			configLogger.Printf("tts-summarize-config: failed to clean up temp file %s: %v", tmp, rmErr)
		}
		return fmt.Errorf("rename config file: %w", err)
	}
	return nil
}
