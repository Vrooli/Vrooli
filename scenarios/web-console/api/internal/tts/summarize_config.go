package tts

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

const (
	MinSummarizeTimeoutSeconds     = 15
	DefaultSummarizeTimeoutSeconds = 120
	MaxSummarizeTimeoutSeconds     = 300
)

// DefaultSummarizeConfig returns the default TTS summarization config.
func DefaultSummarizeConfig() SummarizeConfig {
	model := os.Getenv("WC_TTS_SUMMARIZE_MODEL")
	if model == "" {
		// qwen3:4b follows length/budget instructions much more reliably than
		// qwen3:1.7b. Users on memory-constrained boxes can set
		// WC_TTS_SUMMARIZE_MODEL to downshift.
		model = "qwen3:4b"
	}
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
		base.Model = *p.Model
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
	// Clamp to a realistic floor. Reasoning models (qwen3 family) emit
	// hundreds of <think>…</think> tokens before their actual answer, so
	// anything under ~15 s is a near-guaranteed timeout on CPU inference for
	// non-trivial inputs. We clamp silently on load so stale configs don't
	// silently break the feature after a model change or the user picking a
	// reasoning-capable default.
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
			log.Printf("tts-summarize-config: failed to clean up temp file %s: %v", tmp, rmErr)
		}
		return fmt.Errorf("rename config file: %w", err)
	}
	return nil
}
