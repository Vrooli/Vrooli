// DOC: docs/internal/SEAMS.md#wake-word-engine-seam
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

// ---------------------------------------------------------------------------
// Wake word template types
// ---------------------------------------------------------------------------

// AudioFeatures holds MFCC (or future embedding) features extracted from audio.
// The Kind discriminator enables runtime validation when loading persisted data.
type AudioFeatures struct {
	Kind        string      `json:"kind"`       // "mfcc-v1" or "embedding-v1"
	Data        interface{} `json:"data"`       // [][]float64 for MFCC, []float64 for embeddings
	SampleRate  int         `json:"sampleRate"` // e.g. 16000
	DurationSec float64     `json:"durationSec"`
}

// WakeWordTemplate stores the enrolled wake word samples and configuration.
type WakeWordTemplate struct {
	Samples   []AudioFeatures `json:"samples"`
	Label     string          `json:"label"`
	Threshold float64         `json:"threshold"`
	UpdatedAt string          `json:"updatedAt"`
}

// WakeWordConfig is the top-level config returned by the GET endpoint.
type WakeWordConfig struct {
	Configured bool              `json:"configured"`
	Template   *WakeWordTemplate `json:"template,omitempty"`
}

// ---------------------------------------------------------------------------
// Persistence helpers
// ---------------------------------------------------------------------------

func resolveWakeWordTemplatePath() string {
	return mustResolveScenarioStoragePath(storage.ClassState, "wakeword-template.json")
}

func loadWakeWordTemplate(path string) (*WakeWordTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No template configured
		}
		return nil, fmt.Errorf("read wakeword template: %w", err)
	}
	var tmpl WakeWordTemplate
	if err := json.Unmarshal(data, &tmpl); err != nil {
		return nil, fmt.Errorf("parse wakeword template: %w", err)
	}
	if err := validateWakeWordTemplate(&tmpl); err != nil {
		return nil, fmt.Errorf("wakeword template validation: %w", err)
	}
	return &tmpl, nil
}

func saveWakeWordTemplate(path string, tmpl *WakeWordTemplate) error {
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

func deleteWakeWordTemplate(path string) error {
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete wakeword template: %w", err)
	}
	return nil
}

func validateWakeWordTemplate(tmpl *WakeWordTemplate) error {
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

// ---------------------------------------------------------------------------
// Server accessors (thread-safe)
// ---------------------------------------------------------------------------

func (s *Server) getWakeWordTemplate() *WakeWordTemplate {
	s.wakeWordTemplateMu.RLock()
	defer s.wakeWordTemplateMu.RUnlock()
	return s.wakeWordTemplate
}

func (s *Server) setWakeWordTemplate(tmpl *WakeWordTemplate) {
	s.wakeWordTemplateMu.Lock()
	defer s.wakeWordTemplateMu.Unlock()
	s.wakeWordTemplate = tmpl
}

// ---------------------------------------------------------------------------
// HTTP handlers
// ---------------------------------------------------------------------------

// handleGetWakeWordConfig returns the current wake word configuration.
// GET /api/v1/voice/wakeword
func (s *Server) handleGetWakeWordConfig(w http.ResponseWriter, _ *http.Request) {
	tmpl := s.getWakeWordTemplate()
	cfg := WakeWordConfig{
		Configured: tmpl != nil,
		Template:   tmpl,
	}
	writeJSON(w, http.StatusOK, cfg)
}

// handleUpdateWakeWordTemplate saves a new wake word template.
// PUT /api/v1/voice/wakeword
func (s *Server) handleUpdateWakeWordTemplate(w http.ResponseWriter, r *http.Request) {
	var tmpl WakeWordTemplate
	if !decodeJSON(w, r, &tmpl) {
		return
	}
	if err := validateWakeWordTemplate(&tmpl); err != nil {
		writeCatalogError(w, "invalid_body", err.Error())
		return
	}
	s.setWakeWordTemplate(&tmpl)
	if err := saveWakeWordTemplate(s.wakeWordTemplatePath, &tmpl); err != nil {
		log.Printf("wakeword: persist failed (in-memory updated): %v", err)
	}
	log.Printf("wakeword: template saved: label=%q samples=%d threshold=%.2f",
		tmpl.Label, len(tmpl.Samples), tmpl.Threshold)
	writeJSON(w, http.StatusOK, WakeWordConfig{
		Configured: true,
		Template:   &tmpl,
	})
}

// handleDeleteWakeWordTemplate clears the stored wake word template.
// DELETE /api/v1/voice/wakeword
func (s *Server) handleDeleteWakeWordTemplate(w http.ResponseWriter, _ *http.Request) {
	s.setWakeWordTemplate(nil)
	if err := deleteWakeWordTemplate(s.wakeWordTemplatePath); err != nil {
		log.Printf("wakeword: delete failed: %v", err)
	}
	log.Printf("wakeword: template cleared")
	writeJSON(w, http.StatusOK, WakeWordConfig{
		Configured: false,
	})
}
