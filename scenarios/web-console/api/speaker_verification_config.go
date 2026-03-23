package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type SpeakerVerificationConfig struct {
	Enabled                     bool    `json:"enabled"`
	ProfileID                   string  `json:"profileId"`
	Threshold                   float64 `json:"threshold"`
	Mode                        string  `json:"mode"`
	RejectBehavior              string  `json:"rejectBehavior"`
	FallbackWithoutVerification bool    `json:"fallbackWithoutVerification"`
	ExtractionEnabled           bool    `json:"extractionEnabled"`
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
	if c.Enabled && c.ProfileID == "" {
		return fmt.Errorf("profileId is required when speaker verification is enabled")
	}
	return nil
}

type SpeakerVerificationConfigPatch struct {
	Enabled                     *bool    `json:"enabled,omitempty"`
	ProfileID                   *string  `json:"profileId,omitempty"`
	Threshold                   *float64 `json:"threshold,omitempty"`
	Mode                        *string  `json:"mode,omitempty"`
	RejectBehavior              *string  `json:"rejectBehavior,omitempty"`
	FallbackWithoutVerification *bool    `json:"fallbackWithoutVerification,omitempty"`
	ExtractionEnabled           *bool    `json:"extractionEnabled,omitempty"`
}

func (p SpeakerVerificationConfigPatch) Apply(base SpeakerVerificationConfig) SpeakerVerificationConfig {
	if p.Enabled != nil {
		base.Enabled = *p.Enabled
	}
	if p.ProfileID != nil {
		base.ProfileID = *p.ProfileID
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

func (s *Server) handleGetSpeakerVerificationConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.getSpeakerVerificationConfig())
}

func (s *Server) handleUpdateSpeakerVerificationConfig(w http.ResponseWriter, r *http.Request) {
	var patch SpeakerVerificationConfigPatch
	if !decodeJSON(w, r, &patch) {
		return
	}
	current := s.getSpeakerVerificationConfig()
	updated := patch.Apply(current)
	if updated.Mode == "" {
		updated.Mode = "filter"
	}
	if updated.RejectBehavior == "" {
		updated.RejectBehavior = "drop"
	}
	if err := updated.Validate(); err != nil {
		writeCatalogError(w, "invalid_body", err.Error())
		return
	}
	s.setSpeakerVerificationConfig(updated)
	if err := saveSpeakerVerificationConfig(s.speakerVerificationConfigPath, updated); err != nil {
		log.Printf("speaker-verification-config: persist failed (in-memory updated): %v", err)
	}
	writeJSON(w, http.StatusOK, updated)
}

type SpeakerVerificationStatusResponse struct {
	Config            SpeakerVerificationConfig        `json:"config"`
	Capability        string                           `json:"capability"`
	CapabilityLabel   string                           `json:"capabilityLabel,omitempty"`
	ResourceReady     bool                             `json:"resourceReady"`
	ProfileConfigured bool                             `json:"profileConfigured"`
	ProfileExists     bool                             `json:"profileExists"`
	ProfileCount      int                              `json:"profileCount"`
	Profiles          []SpeakerVerificationProfile     `json:"profiles,omitempty"`
	Info              *SpeakerVerificationResourceInfo `json:"info,omitempty"`
	CheckedAt         string                           `json:"checkedAt"`
}

func (s *Server) handleGetSpeakerVerificationStatus(w http.ResponseWriter, r *http.Request) {
	cfg := s.getSpeakerVerificationConfig()
	resp := SpeakerVerificationStatusResponse{
		Config:            cfg,
		Capability:        string(StatusUnknown),
		ProfileConfigured: cfg.ProfileID != "",
		CheckedAt:         time.Now().UTC().Format(time.RFC3339),
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	for _, cap := range s.capabilities.ResolveLiveness(ctx) {
		if cap.ID != "speaker-verification" {
			continue
		}
		resp.Capability = string(cap.Status)
		resp.CapabilityLabel = cap.Message
		break
	}

	if s.speakerVerification == nil || resp.Capability != string(StatusAvailable) {
		writeJSON(w, http.StatusOK, resp)
		return
	}

	ready, err := s.speakerVerification.Ready(ctx)
	if err == nil && ready.Status == "ready" {
		resp.ResourceReady = true
	}

	profiles, err := s.speakerVerification.ListProfiles(ctx)
	if err == nil {
		resp.ProfileCount = profiles.Count
		resp.Profiles = profiles.Profiles
		for _, profile := range profiles.Profiles {
			if profile.ID == cfg.ProfileID {
				resp.ProfileExists = true
				break
			}
		}
	}

	info, err := s.speakerVerification.Info(ctx)
	if err == nil {
		resp.Info = &info
	}

	writeJSON(w, http.StatusOK, resp)
}
