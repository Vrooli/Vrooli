package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/vrooli/api-core/storage"
)

// TTSSummarizeConfig holds configuration for TTS summarization of long responses.
type TTSSummarizeConfig struct {
	Enabled        bool   `json:"enabled"`        // default: true
	CharThreshold  int    `json:"charThreshold"`  // default: 500
	Level          string `json:"level"`          // "light" | "moderate" | "heavy"
	Model          string `json:"model"`          // default: env WC_TTS_SUMMARIZE_MODEL or "qwen3:4b"
	TimeoutSeconds int    `json:"timeoutSeconds"` // default: 120
}

// DefaultTTSSummarizeConfig returns the default TTS summarization config.
func DefaultTTSSummarizeConfig() TTSSummarizeConfig {
	model := os.Getenv("WC_TTS_SUMMARIZE_MODEL")
	if model == "" {
		// qwen3:4b follows length/budget instructions much more reliably than
		// qwen3:1.7b. Users on memory-constrained boxes can set
		// WC_TTS_SUMMARIZE_MODEL to downshift.
		model = "qwen3:4b"
	}
	return TTSSummarizeConfig{
		Enabled:        true,
		CharThreshold:  500,
		Level:          "moderate",
		Model:          model,
		TimeoutSeconds: defaultSummarizeTimeoutSeconds,
	}
}

// TTSSummarizeConfigPatch is the partial update type. Pointer fields allow
// distinguishing "not provided" from "set to zero value".
type TTSSummarizeConfigPatch struct {
	Enabled        *bool   `json:"enabled,omitempty"`
	CharThreshold  *int    `json:"charThreshold,omitempty"`
	Level          *string `json:"level,omitempty"`
	Model          *string `json:"model,omitempty"`
	TimeoutSeconds *int    `json:"timeoutSeconds,omitempty"`
}

// Apply merges non-nil patch fields into base, returning the updated config.
func (p TTSSummarizeConfigPatch) Apply(base TTSSummarizeConfig) TTSSummarizeConfig {
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

// loadTTSSummarizeConfig reads config from JSON file. Returns defaults if file missing.
func loadTTSSummarizeConfig(path string) (TTSSummarizeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultTTSSummarizeConfig(), nil
		}
		return DefaultTTSSummarizeConfig(), fmt.Errorf("read tts summarize config: %w", err)
	}
	var cfg TTSSummarizeConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultTTSSummarizeConfig(), fmt.Errorf("parse tts summarize config: %w", err)
	}
	if cfg.CharThreshold <= 0 {
		cfg.CharThreshold = 500
	}
	if cfg.Level == "" {
		cfg.Level = "moderate"
	}
	if cfg.Model == "" {
		cfg.Model = DefaultTTSSummarizeConfig().Model
	}
	// Clamp to a realistic floor. Reasoning models (qwen3 family) emit
	// hundreds of <think>…</think> tokens before their actual answer, so
	// anything under ~15 s is a near-guaranteed timeout on CPU inference for
	// non-trivial inputs. We clamp silently on load so stale configs don't
	// silently break the feature after a model change or the user picking a
	// reasoning-capable default.
	if cfg.TimeoutSeconds < minSummarizeTimeoutSeconds {
		cfg.TimeoutSeconds = defaultSummarizeTimeoutSeconds
	}
	return cfg, nil
}

const (
	minSummarizeTimeoutSeconds     = 15
	defaultSummarizeTimeoutSeconds = 120
	maxSummarizeTimeoutSeconds     = 300
)

// saveTTSSummarizeConfig writes config to JSON file atomically.
func saveTTSSummarizeConfig(path string, cfg TTSSummarizeConfig) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
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

var validSummarizeLevels = map[string]bool{
	"light":    true,
	"moderate": true,
	"heavy":    true,
}

func (s *Server) getTTSSummarizeConfig() TTSSummarizeConfig {
	s.ttsSummarizeMu.RLock()
	defer s.ttsSummarizeMu.RUnlock()
	return s.ttsSummarizeConfig
}

func (s *Server) setTTSSummarizeConfig(cfg TTSSummarizeConfig) {
	s.ttsSummarizeMu.Lock()
	defer s.ttsSummarizeMu.Unlock()
	s.ttsSummarizeConfig = cfg
}

// HTTP handlers for /api/v1/tts/summarize/config moved to handlers/tts.
// The validation logic now lives in tts_adapter.go's UpdateSummarizeConfig.

// resolveTTSSummarizeConfigPath returns the summarize config file path using api-core/storage.
func resolveTTSSummarizeConfigPath() string {
	return mustResolveScenarioStoragePath(storage.ClassState, "tts-summarize-config.json")
}
