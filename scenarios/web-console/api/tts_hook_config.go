package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/vrooli/api-core/storage"
)

// TTSHookConfig holds the small web-console-internal preference triple
// that drives auto-TTS routing on the Claude hook + Codex tailer side.
// Voice / speed / response-format / summarization knobs all live in
// audio-tools — this config is only for routing-vs-not and which backend
// the UI defaults to client-side.
type TTSHookConfig struct {
	AutoEnabled bool   `json:"autoEnabled"`
	Backend     string `json:"backend"`
	StartMuted  bool   `json:"startMuted"`
}

// TTSHookConfigPatch is the optional-field patch shape accepted by
// PUT /api/v1/tts-hook/config.
type TTSHookConfigPatch struct {
	AutoEnabled *bool   `json:"autoEnabled,omitempty"`
	Backend     *string `json:"backend,omitempty"`
	StartMuted  *bool   `json:"startMuted,omitempty"`
}

// DefaultTTSHookConfig returns the conservative default: auto off, backend
// resolved via the UI's preference resolver, audio starts muted so the
// first interaction unlocks the autoplay policy cleanly.
func DefaultTTSHookConfig() TTSHookConfig {
	return TTSHookConfig{
		AutoEnabled: false,
		Backend:     "auto",
		StartMuted:  true,
	}
}

// Apply merges non-nil patch fields into base. Backend values outside the
// known set are dropped so a malformed PUT can never poison state.
func (p TTSHookConfigPatch) Apply(base TTSHookConfig) TTSHookConfig {
	if p.AutoEnabled != nil {
		base.AutoEnabled = *p.AutoEnabled
	}
	if p.Backend != nil {
		switch *p.Backend {
		case "auto", "kokoro", "browser":
			base.Backend = *p.Backend
		}
	}
	if p.StartMuted != nil {
		base.StartMuted = *p.StartMuted
	}
	return base
}

// loadTTSHookConfig reads the TTS hook config from disk; missing-file
// returns defaults without error. Malformed JSON falls back to defaults
// with a logged warning so a corrupt file never blocks startup.
func loadTTSHookConfig(path string) (TTSHookConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultTTSHookConfig(), nil
		}
		return DefaultTTSHookConfig(), fmt.Errorf("read tts-hook config: %w", err)
	}
	var cfg TTSHookConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("tts-hook-config: failed to parse %s, using defaults: %v", path, err)
		return DefaultTTSHookConfig(), nil
	}
	if cfg.Backend == "" {
		cfg.Backend = "auto"
	}
	return cfg, nil
}

// saveTTSHookConfig writes the TTS hook config atomically (write+rename)
// so a crashed half-write never corrupts the live file.
func saveTTSHookConfig(path string, cfg TTSHookConfig) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal tts-hook config: %w", err)
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		if rmErr := os.Remove(tmp); rmErr != nil {
			log.Printf("tts-hook-config: failed to clean up temp file %s: %v", tmp, rmErr)
		}
		return fmt.Errorf("rename config file: %w", err)
	}
	return nil
}

func resolveTTSHookConfigPath() string {
	return mustResolveScenarioStoragePath(storage.ClassState, "tts-hook-config.json")
}

// hookConfigMutex serializes access to Server.ttsHookConfig.
type hookConfigState struct {
	mu   sync.RWMutex
	cfg  TTSHookConfig
	path string
}

func (s *Server) getTTSHookConfig() TTSHookConfig {
	s.ttsHookConfigState.mu.RLock()
	defer s.ttsHookConfigState.mu.RUnlock()
	return s.ttsHookConfigState.cfg
}

func (s *Server) setTTSHookConfig(cfg TTSHookConfig) error {
	s.ttsHookConfigState.mu.Lock()
	defer s.ttsHookConfigState.mu.Unlock()
	if err := saveTTSHookConfig(s.ttsHookConfigState.path, cfg); err != nil {
		return err
	}
	s.ttsHookConfigState.cfg = cfg
	return nil
}
