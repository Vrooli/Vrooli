package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TTSConfig holds configuration for automatic text-to-speech delivery.
type TTSConfig struct {
	AutoEnabled bool    `json:"autoEnabled"`
	Backend     string  `json:"backend"`     // "auto", "kokoro", or "browser"
	KokoroVoice string  `json:"kokoroVoice"` // Kokoro voice ID
	KokoroSpeed float64 `json:"kokoroSpeed"` // 0.5-2.0
}

// DefaultTTSConfig returns the default TTS configuration with auto-TTS disabled.
func DefaultTTSConfig() TTSConfig {
	return TTSConfig{
		AutoEnabled: false,
		Backend:     "auto",
		KokoroVoice: "af_heart",
		KokoroSpeed: 1.0,
	}
}

// TTSConfigPatch is the partial update type for TTS config. Pointer fields
// allow distinguishing "not provided" from "set to zero value".
type TTSConfigPatch struct {
	AutoEnabled *bool    `json:"autoEnabled,omitempty"`
	Backend     *string  `json:"backend,omitempty"`
	KokoroVoice *string  `json:"kokoroVoice,omitempty"`
	KokoroSpeed *float64 `json:"kokoroSpeed,omitempty"`
}

// Apply merges non-nil patch fields into base, returning the updated config.
func (p TTSConfigPatch) Apply(base TTSConfig) TTSConfig {
	if p.AutoEnabled != nil {
		base.AutoEnabled = *p.AutoEnabled
	}
	if p.Backend != nil {
		base.Backend = *p.Backend
	}
	if p.KokoroVoice != nil {
		base.KokoroVoice = *p.KokoroVoice
	}
	if p.KokoroSpeed != nil {
		base.KokoroSpeed = *p.KokoroSpeed
	}
	return base
}

// loadTTSConfig reads a TTSConfig from the given JSON file path.
// If the file does not exist, returns defaults without error.
func loadTTSConfig(path string) (TTSConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultTTSConfig(), nil
		}
		return DefaultTTSConfig(), fmt.Errorf("read tts config: %w", err)
	}
	var cfg TTSConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultTTSConfig(), fmt.Errorf("parse tts config: %w", err)
	}
	if cfg.Backend == "" {
		cfg.Backend = "auto"
	}
	if cfg.KokoroVoice == "" {
		cfg.KokoroVoice = "af_heart"
	}
	if cfg.KokoroSpeed <= 0 {
		cfg.KokoroSpeed = 1.0
	}
	return cfg, nil
}

// saveTTSConfig writes a TTSConfig to the given JSON file path.
// The parent directory is created if it doesn't exist. Writes atomically
// via a temporary file to prevent corruption from concurrent writes or crashes.
func saveTTSConfig(path string, cfg TTSConfig) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal tts config: %w", err)
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		if rmErr := os.Remove(tmp); rmErr != nil {
			log.Printf("tts-config: failed to clean up temp file %s: %v", tmp, rmErr)
		}
		return fmt.Errorf("rename config file: %w", err)
	}
	return nil
}

// ttsConfigMu and ttsConfig are embedded in the Server struct (see main.go).
// These accessor methods provide thread-safe read/write access.

func (s *Server) getTTSConfig() TTSConfig {
	s.ttsConfigMu.RLock()
	defer s.ttsConfigMu.RUnlock()
	return s.ttsConfig
}

func (s *Server) setTTSConfig(cfg TTSConfig) {
	s.ttsConfigMu.Lock()
	defer s.ttsConfigMu.Unlock()
	s.ttsConfig = cfg
}

// handleGetTTSConfig returns the current TTS configuration.
// GET /api/v1/tts/config
func (s *Server) handleGetTTSConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.getTTSConfig())
}

type TTSRuntimeStatus struct {
	Config                TTSConfig          `json:"config"`
	HookRegistered        bool               `json:"hookRegistered"`
	HookReason            string             `json:"hookReason"`
	HookSettingsPath      string             `json:"hookSettingsPath,omitempty"`
	LastDelivery          *TTSDeliveryResult `json:"lastDelivery,omitempty"`
	LastDeliveryAt        string             `json:"lastDeliveryAt,omitempty"`
	KokoroCapability      string             `json:"kokoroCapability,omitempty"`
	KokoroCapabilityLabel string             `json:"kokoroCapabilityLabel,omitempty"`
}

func (s *Server) handleGetTTSStatus(w http.ResponseWriter, r *http.Request) {
	hookRegistered, hookReason, hookSettingsPath := s.getClaudeHookStatus()
	lastDelivery, lastDeliveryAt := s.getLastTTSDelivery()

	kokoroCapability := "unknown"
	kokoroCapabilityLabel := "Kokoro status not checked yet"
	for _, cap := range s.capabilities.ResolveLiveness(r.Context()) {
		if cap.ID == "kokoro-tts" {
			kokoroCapability = string(cap.Status)
			kokoroCapabilityLabel = strings.TrimSpace(cap.Message)
			if kokoroCapabilityLabel == "" {
				kokoroCapabilityLabel = "Kokoro status available"
			}
			break
		}
	}

	status := TTSRuntimeStatus{
		Config:                s.getTTSConfig(),
		HookRegistered:        hookRegistered,
		HookReason:            hookReason,
		HookSettingsPath:      hookSettingsPath,
		LastDelivery:          lastDelivery,
		KokoroCapability:      kokoroCapability,
		KokoroCapabilityLabel: kokoroCapabilityLabel,
	}
	if !lastDeliveryAt.IsZero() {
		status.LastDeliveryAt = lastDeliveryAt.UTC().Format(time.RFC3339)
	}
	writeJSON(w, http.StatusOK, status)
}

// handleUpdateTTSConfig applies a partial update to TTS config,
// persists to disk, and returns the updated config.
// PUT /api/v1/tts/config
func (s *Server) handleUpdateTTSConfig(w http.ResponseWriter, r *http.Request) {
	var patch TTSConfigPatch
	if !decodeJSON(w, r, &patch) {
		return
	}
	current := s.getTTSConfig()
	updated := patch.Apply(current)
	switch updated.Backend {
	case "", "auto", "kokoro", "browser":
		if updated.Backend == "" {
			updated.Backend = "auto"
		}
	default:
		writeCatalogError(w, "invalid_body", "backend must be auto, kokoro, or browser")
		return
	}
	if updated.KokoroVoice == "" {
		updated.KokoroVoice = "af_heart"
	}
	if updated.KokoroSpeed < 0.5 || updated.KokoroSpeed > 4.0 {
		writeCatalogError(w, "invalid_body", "kokoroSpeed must be between 0.5 and 4.0")
		return
	}
	s.setTTSConfig(updated)
	if err := saveTTSConfig(s.ttsConfigPath, updated); err != nil {
		log.Printf("tts-config: persist failed (in-memory updated): %v", err)
	}
	log.Printf("tts-config: updated: autoEnabled=%v backend=%s voice=%s speed=%.1f",
		updated.AutoEnabled, updated.Backend, updated.KokoroVoice, updated.KokoroSpeed)
	writeJSON(w, http.StatusOK, updated)
}
