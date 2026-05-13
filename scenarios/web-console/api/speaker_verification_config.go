package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type SpeakerVerificationConfig struct {
	Enabled                     bool     `json:"enabled"`
	ProfileIDs                  []string `json:"profileIds"`
	Threshold                   float64  `json:"threshold"`
	Mode                        string   `json:"mode"`
	RejectBehavior              string   `json:"rejectBehavior"`
	FallbackWithoutVerification bool     `json:"fallbackWithoutVerification"`
	ExtractionEnabled           bool     `json:"extractionEnabled"`
}

func DefaultSpeakerVerificationConfig() SpeakerVerificationConfig {
	return SpeakerVerificationConfig{
		Enabled:                     false,
		Threshold:                   0.35,
		Mode:                        "filter",
		RejectBehavior:              "drop",
		FallbackWithoutVerification: false,
	}
}

func (c SpeakerVerificationConfig) Validate() error {
	if c.Threshold < 0 || c.Threshold > 1 {
		return fmt.Errorf("threshold must be between 0 and 1, got %.3f", c.Threshold)
	}
	switch c.Mode {
	case "", "off", "filter", "advisory":
	default:
		return fmt.Errorf("mode must be off, filter, or advisory")
	}
	switch c.RejectBehavior {
	case "", "drop", "show-muted":
	default:
		return fmt.Errorf("rejectBehavior must be drop or show-muted")
	}
	if c.Enabled && len(c.ProfileIDs) == 0 {
		return fmt.Errorf("at least one profileId is required when speaker verification is enabled")
	}
	return nil
}

type SpeakerVerificationConfigPatch struct {
	Enabled                     *bool     `json:"enabled,omitempty"`
	ProfileIDs                  *[]string `json:"profileIds,omitempty"`
	Threshold                   *float64  `json:"threshold,omitempty"`
	Mode                        *string   `json:"mode,omitempty"`
	RejectBehavior              *string   `json:"rejectBehavior,omitempty"`
	FallbackWithoutVerification *bool     `json:"fallbackWithoutVerification,omitempty"`
	ExtractionEnabled           *bool     `json:"extractionEnabled,omitempty"`
}

func (p SpeakerVerificationConfigPatch) Apply(base SpeakerVerificationConfig) SpeakerVerificationConfig {
	if p.Enabled != nil {
		base.Enabled = *p.Enabled
	}
	if p.ProfileIDs != nil {
		base.ProfileIDs = *p.ProfileIDs
	}
	if p.Threshold != nil {
		base.Threshold = *p.Threshold
	}
	if p.Mode != nil {
		base.Mode = *p.Mode
	}
	if p.RejectBehavior != nil {
		base.RejectBehavior = *p.RejectBehavior
	}
	if p.FallbackWithoutVerification != nil {
		base.FallbackWithoutVerification = *p.FallbackWithoutVerification
	}
	if p.ExtractionEnabled != nil {
		base.ExtractionEnabled = *p.ExtractionEnabled
	}
	return base
}

func loadSpeakerVerificationConfig(path string) (SpeakerVerificationConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultSpeakerVerificationConfig(), nil
		}
		return DefaultSpeakerVerificationConfig(), fmt.Errorf("read speaker verification config: %w", err)
	}
	var cfg SpeakerVerificationConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultSpeakerVerificationConfig(), fmt.Errorf("parse speaker verification config: %w", err)
	}
	if cfg.Mode == "" {
		cfg.Mode = "filter"
	}
	if cfg.RejectBehavior == "" {
		cfg.RejectBehavior = "drop"
	}
	if cfg.Threshold == 0 {
		cfg.Threshold = 0.35
	}
	if err := cfg.Validate(); err != nil {
		return DefaultSpeakerVerificationConfig(), fmt.Errorf("speaker verification config validation: %w", err)
	}
	return cfg, nil
}

func saveSpeakerVerificationConfig(path string, cfg SpeakerVerificationConfig) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal speaker verification config: %w", err)
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename config file: %w", err)
	}
	return nil
}

func (s *Server) getSpeakerVerificationConfig() SpeakerVerificationConfig {
	s.speakerVerificationConfigMu.RLock()
	defer s.speakerVerificationConfigMu.RUnlock()
	return s.speakerVerificationConfig
}

func (s *Server) setSpeakerVerificationConfig(cfg SpeakerVerificationConfig) {
	s.speakerVerificationConfigMu.Lock()
	defer s.speakerVerificationConfigMu.Unlock()
	s.speakerVerificationConfig = cfg
}

// HTTP handlers for /voice/speaker/config and /voice/speaker/status have moved
// to the Connect VoiceService (see handlers/voice and voice_adapter.go).
// Types, validation, and persistence helpers above remain shared.
