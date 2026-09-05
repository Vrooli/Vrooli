// DOC: docs/internal/SEAMS.md#wake-word-engine-seam
package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// AudioFeatures holds MFCC (or future embedding) features extracted from
// audio. Kind enables runtime validation when loading persisted data.
type AudioFeatures struct {
	Kind        string  `json:"kind"`
	Data        any     `json:"data"`
	SampleRate  int     `json:"sampleRate"`
	DurationSec float64 `json:"durationSec"`
}

type WakeWordTemplate struct {
	Samples   []AudioFeatures `json:"samples"`
	Label     string          `json:"label"`
	Threshold float64         `json:"threshold"`
	UpdatedAt string          `json:"updatedAt"`
}

// WakeWordToTransport projects an internal *WakeWordTemplate to the
// transport-visible WakeWordConfig shape used by handlers and the UI.
func WakeWordToTransport(tmpl *WakeWordTemplate) WakeWordConfig {
	if tmpl == nil {
		return WakeWordConfig{}
	}
	payload, err := json.Marshal(tmpl)
	if err != nil {
		return WakeWordConfig{Configured: true}
	}
	return WakeWordConfig{Configured: true, TemplateJSON: string(payload)}
}

func LoadWakeWordTemplate(path string) (*WakeWordTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read wakeword template: %w", err)
	}
	var tmpl WakeWordTemplate
	if err := json.Unmarshal(data, &tmpl); err != nil {
		return nil, fmt.Errorf("parse wakeword template: %w", err)
	}
	if err := ValidateWakeWordTemplate(&tmpl); err != nil {
		return nil, fmt.Errorf("wakeword template validation: %w", err)
	}
	return &tmpl, nil
}

func SaveWakeWordTemplate(path string, tmpl *WakeWordTemplate) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create wakeword directory: %w", err)
	}
	data, err := json.MarshalIndent(tmpl, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal wakeword template: %w", err)
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename wakeword template file: %w", err)
	}
	return nil
}

func DeleteWakeWordTemplate(path string) error {
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete wakeword template: %w", err)
	}
	return nil
}

func ValidateWakeWordTemplate(tmpl *WakeWordTemplate) error {
	if len(tmpl.Samples) < 3 {
		return fmt.Errorf("at least 3 samples required, got %d", len(tmpl.Samples))
	}
	if len(tmpl.Samples) > 5 {
		return fmt.Errorf("at most 5 samples allowed, got %d", len(tmpl.Samples))
	}
	if tmpl.Threshold < 0.1 || tmpl.Threshold > 0.95 {
		return fmt.Errorf("threshold must be between 0.1 and 0.95, got %f", tmpl.Threshold)
	}
	for i, s := range tmpl.Samples {
		if s.Kind != "mfcc-v1" && s.Kind != "embedding-v1" {
			return fmt.Errorf("sample %d: unknown feature kind %q", i, s.Kind)
		}
		if s.SampleRate <= 0 {
			return fmt.Errorf("sample %d: invalid sampleRate %d", i, s.SampleRate)
		}
	}
	return nil
}
