package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/vrooli/api-core/storage"
)

// TTSSummarizeConfig holds configuration for TTS summarization of long responses.
type TTSSummarizeConfig struct {
	Enabled        bool   `json:"enabled"`        // default: false
	CharThreshold  int    `json:"charThreshold"`  // default: 500
	Level          string `json:"level"`          // "light" | "moderate" | "heavy"
	Model          string `json:"model"`          // default: env WC_TTS_SUMMARIZE_MODEL or "qwen3:1.7b"
	TimeoutSeconds int    `json:"timeoutSeconds"` // default: 5
}

// DefaultTTSSummarizeConfig returns the default TTS summarization config.
func DefaultTTSSummarizeConfig() TTSSummarizeConfig {
	model := os.Getenv("WC_TTS_SUMMARIZE_MODEL")
	if model == "" {
		model = "qwen3:1.7b"
	}
	return TTSSummarizeConfig{
		Enabled:        false,
		CharThreshold:  500,
		Level:          "moderate",
		Model:          model,
		TimeoutSeconds: 5,
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
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 5
	}
	return cfg, nil
}

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

// handleGetTTSSummarizeConfig returns the current TTS summarization config.
// GET /api/v1/tts/summarize/config
func (s *Server) handleGetTTSSummarizeConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.getTTSSummarizeConfig())
}

// handleUpdateTTSSummarizeConfig applies a partial update, persists, and returns updated config.
// PUT /api/v1/tts/summarize/config
func (s *Server) handleUpdateTTSSummarizeConfig(w http.ResponseWriter, r *http.Request) {
	var patch TTSSummarizeConfigPatch
	if !decodeJSON(w, r, &patch) {
		return
	}

	current := s.getTTSSummarizeConfig()
	updated := patch.Apply(current)

	if !validSummarizeLevels[updated.Level] {
		writeCatalogError(w, "invalid_body", "level must be light, moderate, or heavy")
		return
	}
	if updated.CharThreshold < 0 {
		writeCatalogError(w, "invalid_body", "charThreshold must be non-negative")
		return
	}
	if updated.TimeoutSeconds < 1 || updated.TimeoutSeconds > 60 {
		writeCatalogError(w, "invalid_body", "timeoutSeconds must be between 1 and 60")
		return
	}
	if updated.Model == "" {
		writeCatalogError(w, "invalid_body", "model must not be empty")
		return
	}

	s.setTTSSummarizeConfig(updated)
	if err := saveTTSSummarizeConfig(s.ttsSummarizePath, updated); err != nil {
		log.Printf("tts-summarize-config: persist failed (in-memory updated): %v", err)
	}
	log.Printf("tts-summarize-config: updated: enabled=%v threshold=%d level=%s model=%s timeout=%ds",
		updated.Enabled, updated.CharThreshold, updated.Level, updated.Model, updated.TimeoutSeconds)
	writeJSON(w, http.StatusOK, updated)
}

// resolveTTSSummarizeConfigPath returns the summarize config file path using api-core/storage.
func resolveTTSSummarizeConfigPath() string {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		log.Printf("tts-summarize-config: storage resolver failed, using fallback: %v", err)
		return fallbackTTSSummarizeConfigPath()
	}

	opts := storage.Options{ScenarioID: "web-console"}
	if _, err := storage.EnsureClassDir(resolver, opts, storage.ClassState, 0); err != nil {
		log.Printf("tts-summarize-config: ensure state dir failed, using fallback: %v", err)
		return fallbackTTSSummarizeConfigPath()
	}

	path, err := resolver.Path(opts, storage.ClassState, "tts-summarize-config.json")
	if err != nil {
		log.Printf("tts-summarize-config: resolve path failed, using fallback: %v", err)
		return fallbackTTSSummarizeConfigPath()
	}
	return path
}

func fallbackTTSSummarizeConfigPath() string {
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "..", "store", "tts-summarize-config.json")
}
