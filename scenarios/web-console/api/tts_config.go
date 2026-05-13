package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
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

// HTTP handlers for the TTS config/status/update endpoints have moved to
// handlers/tts (Connect-RPC). The legacy types and validation now live in
// tts_adapter.go; this file keeps only the config struct, patch type, and
// persistence helpers.
