package workspace

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestLoadTestingConfigParsesSettings(t *testing.T) {
	t.Run("[REQ:TESTGENIE-ORCH-P0] loader parses phase toggles + presets", func(t *testing.T) {
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
  },
  "requirements": {
    "enforce": true,
    "sync": false
  }
}`
		if err := os.WriteFile(filepath.Join(configDir, "testing.json"), []byte(payload), 0o644); err != nil {
			t.Fatalf("failed to write testing config: %v", err)
		}

		cfg, err := LoadTestingConfig(root)
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

		if cfg.Requirements.Enforce == nil || !*cfg.Requirements.Enforce {
			t.Fatalf("expected requirements.enforce flag to be true")
		}
		if cfg.Requirements.Sync == nil || *cfg.Requirements.Sync {
			t.Fatalf("expected requirements.sync flag to be false")
		}
	})
}

func TestLoadTestingConfigIgnoresUnitPolicyProfile(t *testing.T) {
	t.Run("[REQ:TESTGENIE-ORCH-P0] unit policy profile does not affect phase compatibility", func(t *testing.T) {
		root := t.TempDir()
		configDir := filepath.Join(root, ".vrooli")
		if err := os.MkdirAll(configDir, 0o755); err != nil {
			t.Fatalf("failed to create config dir: %v", err)
		}
		payload := `{
  "unit": {
    "policy_profile": {
      "version": "1.0.0",
      "template": {"id": "react-vite", "scenario_class": "react-vite"},
      "required_roles": [
        {"role": "api", "policy_class": "go_service"},
        {"role": "ui", "policy_class": "react_vite_ui"}
      ],
      "policy_classes": {
        "go_service": {"language": "go", "framework": "go test"},
        "react_vite_ui": {"language": "typescript", "framework": "vitest"}
      },
      "customization": {"mode": "monotonic", "waivers": []}
    }
  },
  "phases": {
    "unit": {"enabled": true, "timeout": "120s"}
  },
  "presets": {
    "smoke": ["unit"]
  }
}`
		if err := os.WriteFile(filepath.Join(configDir, "testing.json"), []byte(payload), 0o644); err != nil {
			t.Fatalf("failed to write testing config: %v", err)
		}

		cfg, err := LoadTestingConfig(root)
		if err != nil {
			t.Fatalf("LoadTestingConfig failed: %v", err)
		}
		if cfg == nil {
			t.Fatalf("expected phase config to parse")
		}
		unit, ok := cfg.Phases["unit"]
		if !ok {
			t.Fatalf("expected unit phase settings")
		}
		if unit.Enabled == nil || !*unit.Enabled {
			t.Fatalf("expected unit phase enabled")
		}
		if unit.Timeout != 120*time.Second {
			t.Fatalf("expected unit timeout to equal 120s, got %s", unit.Timeout)
		}
		if got := cfg.Presets["smoke"]; !reflect.DeepEqual([]string{"unit"}, got) {
			t.Fatalf("unexpected smoke preset phases: %v", got)
		}
	})
}

func TestLoadTestingConfigMissingFile(t *testing.T) {
	t.Run("[REQ:TESTGENIE-ORCH-P0] loader tolerates missing config", func(t *testing.T) {
		root := t.TempDir()
		cfg, err := LoadTestingConfig(root)
		if err != nil {
			t.Fatalf("expected no error when config missing: %v", err)
		}
		if cfg != nil {
			t.Fatalf("expected nil config when file missing")
		}
	})
}

func TestParsePhaseTimeoutValidation(t *testing.T) {
	t.Run("[REQ:TESTGENIE-ORCH-P0] timeout parser enforces format", func(t *testing.T) {
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
	})
}
