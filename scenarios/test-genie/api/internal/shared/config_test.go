package shared

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

type phaseConfigFixture struct {
	Enabled *bool  `json:"enabled"`
	Timeout string `json:"timeout"`
}

func TestLoadPhaseConfigIgnoresUnitPolicyProfile(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, ".vrooli")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	payload := `{
  "unit": {
    "policy_profile": {
      "version": "1.0.0",
      "template": {"id": "react-vite", "scenario_class": "react-vite"},
      "required_roles": [{"role": "ui", "policy_class": "react_vite_ui"}],
      "policy_classes": {
        "react_vite_ui": {
          "language": "typescript",
          "framework": "vitest",
          "coverage": {"minimum_percent": 85}
        }
      },
      "customization": {"mode": "monotonic", "waivers": []}
    }
  },
  "phases": {
    "unit": {"enabled": true, "timeout": "120s"}
  }
}`
	if err := os.WriteFile(filepath.Join(configDir, "testing.json"), []byte(payload), 0o644); err != nil {
		t.Fatalf("write testing config: %v", err)
	}

	cfg, err := LoadPhaseConfig(root, "phases", map[string]phaseConfigFixture{})
	if err != nil {
		t.Fatalf("LoadPhaseConfig: %v", err)
	}
	unit, ok := cfg["unit"]
	if !ok {
		t.Fatalf("unit phase config missing: %#v", cfg)
	}
	if unit.Enabled == nil || !*unit.Enabled {
		t.Fatalf("unit phase enabled = %v, want true", unit.Enabled)
	}
	if unit.Timeout != "120s" {
		t.Fatalf("unit timeout = %q, want 120s", unit.Timeout)
	}
}

func TestMergePhaseConfigIgnoresUnitPolicyProfile(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, ".vrooli")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	payload := `{
  "unit": {
    "policy_profile": {
      "version": "1.0.0",
      "template": {"id": "react-vite", "scenario_class": "react-vite"},
      "required_roles": [{"role": "api", "policy_class": "go_service"}],
      "policy_classes": {"go_service": {"language": "go", "framework": "go test"}},
      "customization": {"mode": "monotonic", "waivers": []}
    }
  },
  "performance": {"enabled": false}
}`
	if err := os.WriteFile(filepath.Join(configDir, "testing.json"), []byte(payload), 0o644); err != nil {
		t.Fatalf("write testing config: %v", err)
	}

	cfg := struct {
		Enabled bool          `json:"enabled"`
		Timeout time.Duration `json:"-"`
	}{Enabled: true, Timeout: time.Minute}
	if err := MergePhaseConfig(root, "performance", &cfg); err != nil {
		t.Fatalf("MergePhaseConfig: %v", err)
	}
	if cfg.Enabled {
		t.Fatal("performance.enabled should merge to false")
	}
	if cfg.Timeout != time.Minute {
		t.Fatalf("timeout changed unexpectedly: %s", cfg.Timeout)
	}
}
