package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if !cfg.Enabled {
		t.Error("expected Enabled to be true by default")
	}
	if cfg.BAS.Endpoint != DefaultBASEndpoint {
		t.Errorf("BAS.Endpoint = %q, want %q", cfg.BAS.Endpoint, DefaultBASEndpoint)
	}
	if cfg.BAS.TimeoutMs != DefaultBASTimeoutMs {
		t.Errorf("BAS.TimeoutMs = %d, want %d", cfg.BAS.TimeoutMs, DefaultBASTimeoutMs)
	}
	if !cfg.Seeds.Enabled {
		t.Error("expected Seeds.Enabled to be true by default")
	}
	if !cfg.Artifacts.Screenshots {
		t.Error("expected Artifacts.Screenshots to be true by default")
	}
}

func TestBASConfigTimeout(t *testing.T) {
	cfg := &BASConfig{TimeoutMs: 5000}
	got := cfg.Timeout()
	want := 5 * time.Second
	if got != want {
		t.Errorf("Timeout() = %v, want %v", got, want)
	}
}

func TestBASConfigTimeoutDefault(t *testing.T) {
	cfg := &BASConfig{TimeoutMs: 0}
	got := cfg.Timeout()
	want := time.Duration(DefaultBASTimeoutMs) * time.Millisecond
	if got != want {
		t.Errorf("Timeout() with zero = %v, want %v", got, want)
	}
}

func TestLoadNoFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}
	// Should return defaults when no file exists
	if cfg.BAS.Endpoint != DefaultBASEndpoint {
		t.Errorf("BAS.Endpoint = %q, want default %q", cfg.BAS.Endpoint, DefaultBASEndpoint)
	}
}

func TestLoadNoPlaybooksSection(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	testingJSON := `{
		"version": "1.0.0",
		"unit": {
			"policy_profile": {
				"version": "1.0.0",
				"template": {"id": "react-vite", "scenario_class": "react-vite"},
				"required_roles": [{"role": "api", "policy_class": "go_service"}],
				"policy_classes": {"go_service": {"language": "go", "framework": "go test"}},
				"customization": {"mode": "monotonic", "waivers": []}
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(testingJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	// Should return defaults when no playbooks section
	if cfg.BAS.Endpoint != DefaultBASEndpoint {
		t.Errorf("BAS.Endpoint = %q, want default %q", cfg.BAS.Endpoint, DefaultBASEndpoint)
	}
}

func TestLoadIgnoresUnitPolicyProfile(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	testingJSON := `{
		"version": "1.0.0",
		"unit": {
			"policy_profile": {
				"version": "1.0.0",
				"template": {"id": "react-vite", "scenario_class": "react-vite"},
				"required_roles": [
					{"role": "api", "policy_class": "go_service"},
					{"role": "cli", "policy_class": "go_cli"},
					{"role": "ui", "policy_class": "react_vite_ui"}
				],
				"policy_classes": {
					"go_service": {"language": "go", "framework": "go test"},
					"go_cli": {"language": "go", "framework": "go test"},
					"react_vite_ui": {"language": "typescript", "framework": "vitest"}
				},
				"customization": {"mode": "monotonic", "waivers": []}
			}
		},
		"playbooks": {
			"enabled": false,
			"diagnostics": {"console": true}
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(testingJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Enabled {
		t.Fatal("playbooks.enabled should still be parsed when unit.policy_profile is present")
	}
	if !cfg.Diagnostics.Console {
		t.Fatal("playbooks diagnostics should still be parsed when unit.policy_profile is present")
	}
}

func TestLoadWithPlaybooksSection(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	testingJSON := `{
		"version": "1.0.0",
		"playbooks": {
			"enabled": true,
			"bas": {
				"endpoint": "http://custom:9999/api/v1",
				"timeout_ms": 60000
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(testingJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.BAS.Endpoint != "http://custom:9999/api/v1" {
		t.Errorf("BAS.Endpoint = %q, want custom endpoint", cfg.BAS.Endpoint)
	}
	if cfg.BAS.TimeoutMs != 60000 {
		t.Errorf("BAS.TimeoutMs = %d, want 60000", cfg.BAS.TimeoutMs)
	}
	// Unset values should use defaults
	if cfg.BAS.LaunchTimeoutMs != DefaultBASLaunchTimeoutMs {
		t.Errorf("LaunchTimeoutMs = %d, want default %d", cfg.BAS.LaunchTimeoutMs, DefaultBASLaunchTimeoutMs)
	}
}

func TestLoadPlaybooksDoesNotDisableDefaultsWhenUnset(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// This intentionally omits most boolean fields. Historically this could
	// accidentally flip defaults to false when a playbooks section existed.
	testingJSON := `{
		"version": "1.0.0",
		"playbooks": {
			"bas": {
				"timeout_ms": 60000
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(testingJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.Enabled {
		t.Error("expected playbooks.Enabled to remain true by default when unset")
	}
	if !cfg.Seeds.Enabled {
		t.Error("expected playbooks.seeds.enabled to remain true by default when unset")
	}
	if !cfg.Artifacts.Screenshots {
		t.Error("expected playbooks.artifacts.screenshots to remain true by default when unset")
	}
	if !cfg.Artifacts.DOMSnapshots {
		t.Error("expected playbooks.artifacts.dom_snapshots to remain true by default when unset")
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(tmpDir)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadAllowEmptyTestPool(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	testingJSON := `{"playbooks": {"allow_empty_test_pool": true}}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(testingJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.AllowEmptyTestPool {
		t.Error("expected AllowEmptyTestPool to be true")
	}
}

func TestLoadAllowEmptyTestPoolDefaultsFalse(t *testing.T) {
	tmpDir := t.TempDir()
	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.AllowEmptyTestPool {
		t.Error("expected AllowEmptyTestPool to default to false (strict)")
	}
}

func TestDiagnosticsDefault(t *testing.T) {
	cfg := Default()
	if !cfg.Diagnostics.Console {
		t.Error("expected console diagnostics on by default (cheap)")
	}
	if cfg.Diagnostics.Video || cfg.Diagnostics.HAR || cfg.Diagnostics.Trace || cfg.Diagnostics.Network || cfg.Diagnostics.DOM {
		t.Errorf("expected only console enabled by default, got %#v", cfg.Diagnostics)
	}
}

func TestDiagnosticsPreset(t *testing.T) {
	cases := []struct {
		name      string
		wantOK    bool
		wantVideo bool
		wantCons  bool
	}{
		{name: "none", wantOK: true, wantVideo: false, wantCons: false},
		{name: "light", wantOK: true, wantVideo: false, wantCons: true},
		{name: "", wantOK: true, wantVideo: false, wantCons: true},
		{name: "full", wantOK: true, wantVideo: true, wantCons: true},
		{name: "bogus", wantOK: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, ok := DiagnosticsPreset(tc.name)
			if ok != tc.wantOK {
				t.Fatalf("DiagnosticsPreset(%q) ok=%v, want %v", tc.name, ok, tc.wantOK)
			}
			if !tc.wantOK {
				return
			}
			if d.Video != tc.wantVideo {
				t.Errorf("preset %q video=%v, want %v", tc.name, d.Video, tc.wantVideo)
			}
			if d.Console != tc.wantCons {
				t.Errorf("preset %q console=%v, want %v", tc.name, d.Console, tc.wantCons)
			}
		})
	}
	// "full" should enable every diagnostic.
	full, _ := DiagnosticsPreset("full")
	if !(full.Video && full.Console && full.Network && full.HAR && full.Trace && full.DOM) {
		t.Errorf("full preset should enable everything, got %#v", full)
	}
}

func TestLoadDiagnosticsFromTestingJSON(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	testingJSON := `{"playbooks": {"diagnostics": {"video": true, "console": false}, "continue_on_failure": true, "workflow_filter": ["login"]}}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(testingJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.Diagnostics.Video {
		t.Error("expected video diagnostics enabled from config")
	}
	if cfg.Diagnostics.Console {
		t.Error("expected console explicitly disabled by config")
	}
	if !cfg.ContinueOnFailure {
		t.Error("expected continue_on_failure true")
	}
	if len(cfg.WorkflowFilter) != 1 || cfg.WorkflowFilter[0] != "login" {
		t.Errorf("expected workflow_filter [login], got %v", cfg.WorkflowFilter)
	}
}
