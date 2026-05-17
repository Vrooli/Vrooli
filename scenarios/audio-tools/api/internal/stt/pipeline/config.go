// Package pipeline owns the STT pipeline state and behaviour: stream
// config, wake-word templates, speaker-verification config and resource
// client, the Whisper transcribe path, and the WebSocket-bridge
// constants used by the browser transport.
package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds tunable parameters for the voice streaming pipeline. Values
// are validated on update to stay within safe operating bounds. The pipeline
// snapshots config once per WebSocket session, so mid-session changes take
// effect on the next recording.
type Config struct {
	FlushIntervalMs   int     `json:"flushIntervalMs"`
	MinDeltaBytes     int     `json:"minDeltaBytes"`
	OverlapBytes      int     `json:"overlapBytes"`
	PersistentMode    bool    `json:"persistentMode"`
	WakeWordEnabled   bool    `json:"wakeWordEnabled"`
	WakeWordThreshold float64 `json:"wakeWordThreshold"`
	SegmentSilenceMs  int     `json:"segmentSilenceMs"`
}

func DefaultConfig() Config {
	return Config{
		FlushIntervalMs:   500,
		MinDeltaBytes:     4096,
		OverlapBytes:      2048,
		PersistentMode:    false,
		WakeWordEnabled:   false,
		WakeWordThreshold: 0.65,
		SegmentSilenceMs:  1500,
	}
}

func (c Config) Validate() error {
	if c.FlushIntervalMs < 100 || c.FlushIntervalMs > 5000 {
		return fmt.Errorf("flushIntervalMs must be between 100 and 5000, got %d", c.FlushIntervalMs)
	}
	if c.MinDeltaBytes < 512 || c.MinDeltaBytes > 32768 {
		return fmt.Errorf("minDeltaBytes must be between 512 and 32768, got %d", c.MinDeltaBytes)
	}
	if c.OverlapBytes < 0 || c.OverlapBytes > 16384 {
		return fmt.Errorf("overlapBytes must be between 0 and 16384, got %d", c.OverlapBytes)
	}
	if c.SegmentSilenceMs != 0 && (c.SegmentSilenceMs < 800 || c.SegmentSilenceMs > 3000) {
		return fmt.Errorf("segmentSilenceMs must be between 800 and 3000, got %d", c.SegmentSilenceMs)
	}
	if c.WakeWordThreshold != 0 && (c.WakeWordThreshold < 0.1 || c.WakeWordThreshold > 0.95) {
		return fmt.Errorf("wakeWordThreshold must be between 0.1 and 0.95, got %f", c.WakeWordThreshold)
	}
	return nil
}

type ConfigPatch struct {
	FlushIntervalMs   *int     `json:"flushIntervalMs,omitempty"`
	MinDeltaBytes     *int     `json:"minDeltaBytes,omitempty"`
	OverlapBytes      *int     `json:"overlapBytes,omitempty"`
	PersistentMode    *bool    `json:"persistentMode,omitempty"`
	WakeWordEnabled   *bool    `json:"wakeWordEnabled,omitempty"`
	WakeWordThreshold *float64 `json:"wakeWordThreshold,omitempty"`
	SegmentSilenceMs  *int     `json:"segmentSilenceMs,omitempty"`
}

func (p ConfigPatch) Apply(base Config) Config {
	if p.FlushIntervalMs != nil {
		base.FlushIntervalMs = *p.FlushIntervalMs
	}
	if p.MinDeltaBytes != nil {
		base.MinDeltaBytes = *p.MinDeltaBytes
	}
	if p.OverlapBytes != nil {
		base.OverlapBytes = *p.OverlapBytes
	}
	if p.PersistentMode != nil {
		base.PersistentMode = *p.PersistentMode
	}
	if p.WakeWordEnabled != nil {
		base.WakeWordEnabled = *p.WakeWordEnabled
	}
	if p.WakeWordThreshold != nil {
		base.WakeWordThreshold = *p.WakeWordThreshold
	}
	if p.SegmentSilenceMs != nil {
		base.SegmentSilenceMs = *p.SegmentSilenceMs
	}
	return base
}

func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return DefaultConfig(), fmt.Errorf("read voice config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig(), fmt.Errorf("parse voice config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return DefaultConfig(), fmt.Errorf("voice config validation: %w", err)
	}
	return cfg, nil
}

func SaveConfig(path string, cfg Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal voice config: %w", err)
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename config file: %w", err)
	}
	return nil
}
