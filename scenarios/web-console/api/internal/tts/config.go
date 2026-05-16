package tts

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// DefaultConfig returns the default TTS configuration with auto-TTS disabled.
func DefaultConfig() Config {
	return Config{
		AutoEnabled: false,
		Backend:     "auto",
		KokoroVoice: "af_heart",
		KokoroSpeed: 1.0,
	}
}

// Apply merges non-nil patch fields into base, returning the updated config.
func (p ConfigPatch) Apply(base Config) Config {
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

// LoadConfig reads a Config from the given JSON file path. If the file does
// not exist, returns defaults without error.
func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return DefaultConfig(), fmt.Errorf("read tts config: %w", err)
	}
	var raw struct {
		AutoEnabled bool    `json:"autoEnabled"`
		Backend     string  `json:"backend"`
		KokoroVoice string  `json:"kokoroVoice"`
		KokoroSpeed float64 `json:"kokoroSpeed"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return DefaultConfig(), fmt.Errorf("parse tts config: %w", err)
	}
	cfg := Config{
		AutoEnabled: raw.AutoEnabled,
		Backend:     raw.Backend,
		KokoroVoice: raw.KokoroVoice,
		KokoroSpeed: raw.KokoroSpeed,
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

// SaveConfig writes a Config to the given JSON file path. The parent
// directory is created if it doesn't exist. Writes atomically via a
// temporary file to prevent corruption from concurrent writes or crashes.
func SaveConfig(path string, cfg Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	wire := struct {
		AutoEnabled bool    `json:"autoEnabled"`
		Backend     string  `json:"backend"`
		KokoroVoice string  `json:"kokoroVoice"`
		KokoroSpeed float64 `json:"kokoroSpeed"`
	}{
		AutoEnabled: cfg.AutoEnabled,
		Backend:     cfg.Backend,
		KokoroVoice: cfg.KokoroVoice,
		KokoroSpeed: cfg.KokoroSpeed,
	}
	data, err := json.MarshalIndent(wire, "", "  ")
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
