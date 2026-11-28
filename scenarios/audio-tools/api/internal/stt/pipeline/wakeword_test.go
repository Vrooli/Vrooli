package pipeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validSample() AudioFeatures {
	return AudioFeatures{Kind: "mfcc-v1", Data: []float64{0.1, 0.2}, SampleRate: 16000, DurationSec: 1.0}
}

func validTemplate() *WakeWordTemplate {
	return &WakeWordTemplate{
		Samples:   []AudioFeatures{validSample(), validSample(), validSample()},
		Label:     "hello",
		Threshold: 0.6,
		UpdatedAt: "2026-05-16T00:00:00Z",
	}
}

func TestValidateWakeWordTemplate(t *testing.T) {
	if err := ValidateWakeWordTemplate(validTemplate()); err != nil {
		t.Fatalf("valid template rejected: %v", err)
	}

	bad := validTemplate()
	bad.Samples = bad.Samples[:2]
	if err := ValidateWakeWordTemplate(bad); err == nil {
		t.Fatalf("expected too-few-samples rejection")
	}

	bad = validTemplate()
	bad.Samples = append(bad.Samples, validSample(), validSample(), validSample())
	if err := ValidateWakeWordTemplate(bad); err == nil {
		t.Fatalf("expected too-many-samples rejection")
	}

	bad = validTemplate()
	bad.Threshold = 0.0
	if err := ValidateWakeWordTemplate(bad); err == nil {
		t.Fatalf("expected low-threshold rejection")
	}

	bad = validTemplate()
	bad.Samples[0].Kind = "unknown"
	if err := ValidateWakeWordTemplate(bad); err == nil {
		t.Fatalf("expected unknown-kind rejection")
	}

	bad = validTemplate()
	bad.Samples[0].SampleRate = 0
	if err := ValidateWakeWordTemplate(bad); err == nil {
		t.Fatalf("expected zero-sampleRate rejection")
	}
}

func TestWakeWordToTransport(t *testing.T) {
	c := WakeWordToTransport(nil)
	if c.Configured || c.TemplateJSON != "" {
		t.Fatalf("nil template should produce empty config, got %+v", c)
	}

	tmpl := validTemplate()
	c = WakeWordToTransport(tmpl)
	if !c.Configured {
		t.Fatalf("expected Configured=true")
	}
	if !strings.Contains(c.TemplateJSON, "\"label\":\"hello\"") {
		t.Fatalf("expected JSON to include label, got %q", c.TemplateJSON)
	}
}

func TestLoadSaveDeleteWakeWordTemplate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wakeword.json")

	got, err := LoadWakeWordTemplate(path)
	if err != nil {
		t.Fatalf("Load on missing path: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil on missing path, got %+v", got)
	}

	tmpl := validTemplate()
	if err := SaveWakeWordTemplate(path, tmpl); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := LoadWakeWordTemplate(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded == nil || loaded.Label != "hello" {
		t.Fatalf("unexpected load: %+v", loaded)
	}

	if err := DeleteWakeWordTemplate(path); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file removed, stat err: %v", err)
	}
	if err := DeleteWakeWordTemplate(path); err != nil {
		t.Fatalf("Delete on missing path should be no-op: %v", err)
	}
}

func TestLoadWakeWordTemplateRejectsCorrupt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadWakeWordTemplate(path); err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestLoadWakeWordTemplateRejectsInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.json")
	bad := validTemplate()
	bad.Threshold = 5
	raw, _ := json.Marshal(bad)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadWakeWordTemplate(path); err == nil {
		t.Fatalf("expected validation error")
	}
}
