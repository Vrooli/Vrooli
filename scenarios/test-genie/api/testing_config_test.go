package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestLoadTestingConfigParsesSettings(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, ".vrooli")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	payload := `{
  "phases": {
    "integration": {"enabled": false, "timeout": "90s"},
    "slowPhase": {"timeout": "2m"}
  },
  "presets": {
    "focused": ["Unit", " integration "]
  }
}`
	if err := os.WriteFile(filepath.Join(configDir, "testing.json"), []byte(payload), 0o644); err != nil {
		t.Fatalf("failed to write testing config: %v", err)
	}

	cfg, err := loadTestingConfig(root)
	if err != nil {
		t.Fatalf("loadTestingConfig failed: %v", err)
	}
	if cfg == nil {
		t.Fatalf("expected config to be parsed")
	}

	integration, ok := cfg.Phases["integration"]
	if !ok {
		t.Fatalf("expected integration phase settings")
	}
	if integration.Enabled == nil || *integration.Enabled != false {
		t.Fatalf("expected integration enabled flag to be false")
	}
	if integration.Timeout != 90*time.Second {
		t.Fatalf("expected integration timeout to equal 90s, got %s", integration.Timeout)
	}

	slow, ok := cfg.Phases["slowphase"]
	if !ok {
		t.Fatalf("expected slowphase settings")
	}
	if slow.Timeout != 2*time.Minute {
		t.Fatalf("expected slow timeout to equal 2m, got %s", slow.Timeout)
	}

	expectedPreset := []string{"unit", "integration"}
	if got := cfg.Presets["focused"]; !reflect.DeepEqual(expectedPreset, got) {
		t.Fatalf("unexpected preset phases: %v", got)
	}
}

func TestLoadTestingConfigMissingFile(t *testing.T) {
	root := t.TempDir()
	cfg, err := loadTestingConfig(root)
	if err != nil {
		t.Fatalf("expected no error when config missing: %v", err)
	}
	if cfg != nil {
		t.Fatalf("expected nil config when file missing")
	}
}

func TestParsePhaseTimeoutValidation(t *testing.T) {
	if _, err := parsePhaseTimeout("abc"); err == nil {
		t.Fatalf("expected error for invalid timeout")
	}
	if _, err := parsePhaseTimeout("10q"); err == nil {
		t.Fatalf("expected error for unknown unit")
	}
	duration, err := parsePhaseTimeout("15m")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if duration != 15*time.Minute {
		t.Fatalf("expected 15m duration, got %s", duration)
	}
}
