package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Session-decision defaults applied when the config leaves them zero. The
// warm-up window holds off any rejection until this much voiced audio has
// accrued; the smoothing alpha is the EMA weight on each new segment score.
const (
	DefaultMinDecisionSeconds = 3.0
	DefaultScoreSmoothing     = 0.4
)

type SpeakerConfig struct {
	Enabled                     bool     `json:"enabled"`
	ProfileIDs                  []string `json:"profileIds"`
	Threshold                   float64  `json:"threshold"`
	Mode                        string   `json:"mode"`
	RejectBehavior              string   `json:"rejectBehavior"`
	FallbackWithoutVerification bool     `json:"fallbackWithoutVerification"`
	ExtractionEnabled           bool     `json:"extractionEnabled"`
	// Session-decision tuning. Zero means "use the default" (see NewSessionSpeakerState).
	MinDecisionSeconds float64 `json:"minDecisionSeconds"`
	ScoreSmoothing     float64 `json:"scoreSmoothing"`
}

func DefaultSpeakerConfig() SpeakerConfig {
	return SpeakerConfig{
		Enabled:                     false,
		Threshold:                   0.5,
		Mode:                        "filter",
		RejectBehavior:              "drop",
		FallbackWithoutVerification: false,
		MinDecisionSeconds:          DefaultMinDecisionSeconds,
		ScoreSmoothing:              DefaultScoreSmoothing,
	}
}

func (c SpeakerConfig) Validate() error {
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

type SpeakerConfigPatch struct {
	Enabled                     *bool     `json:"enabled,omitempty"`
	ProfileIDs                  *[]string `json:"profileIds,omitempty"`
	Threshold                   *float64  `json:"threshold,omitempty"`
	Mode                        *string   `json:"mode,omitempty"`
	RejectBehavior              *string   `json:"rejectBehavior,omitempty"`
	FallbackWithoutVerification *bool     `json:"fallbackWithoutVerification,omitempty"`
	ExtractionEnabled           *bool     `json:"extractionEnabled,omitempty"`
	MinDecisionSeconds          *float64  `json:"minDecisionSeconds,omitempty"`
	ScoreSmoothing              *float64  `json:"scoreSmoothing,omitempty"`
}

func (p SpeakerConfigPatch) Apply(base SpeakerConfig) SpeakerConfig {
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
	if p.MinDecisionSeconds != nil {
		base.MinDecisionSeconds = *p.MinDecisionSeconds
	}
	if p.ScoreSmoothing != nil {
		base.ScoreSmoothing = *p.ScoreSmoothing
	}
	return base
}

func LoadSpeakerConfig(path string) (SpeakerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultSpeakerConfig(), nil
		}
		return DefaultSpeakerConfig(), fmt.Errorf("read speaker verification config: %w", err)
	}
	var cfg SpeakerConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultSpeakerConfig(), fmt.Errorf("parse speaker verification config: %w", err)
	}
	if cfg.Mode == "" {
		cfg.Mode = "filter"
	}
	if cfg.RejectBehavior == "" {
		cfg.RejectBehavior = "drop"
	}
	if cfg.Threshold == 0 {
		cfg.Threshold = 0.5
	}
	if cfg.MinDecisionSeconds == 0 {
		cfg.MinDecisionSeconds = DefaultMinDecisionSeconds
	}
	if cfg.ScoreSmoothing == 0 {
		cfg.ScoreSmoothing = DefaultScoreSmoothing
	}
	if err := cfg.Validate(); err != nil {
		return DefaultSpeakerConfig(), fmt.Errorf("speaker verification config validation: %w", err)
	}
	return cfg, nil
}

func SaveSpeakerConfig(path string, cfg SpeakerConfig) error {
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

func DefaultSpeakerProfileID() string {
	return "default"
}
