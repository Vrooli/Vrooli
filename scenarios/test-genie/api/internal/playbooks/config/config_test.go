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
	if err := os.MkdirAll(vrooliDir, 0755); err != nil {
		t.Fatal(err)
	}

	testingJSON := `{"version": "1.0.0", "unit": {"languages": {"go": {}}}}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(testingJSON), 0644); err != nil {
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

func TestLoadWithPlaybooksSection(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0755); err != nil {
		t.Fatal(err)
	}

	testingJSON := `{
		"version": "1.0.0",
		"playbooks": {
			"enabled": true,
			"bas": {
				"endpoint": "http://custom:9999/api/v1",
				"timeout_ms": 60000
			},
			"execution": {
				"stop_on_first_failure": true,
				"default_step_timeout_ms": 15000
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(testingJSON), 0644); err != nil {
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
	if !cfg.Execution.StopOnFirstFailure {
		t.Error("expected StopOnFirstFailure to be true")
	}
	if cfg.Execution.DefaultStepTimeoutMs != 15000 {
		t.Errorf("DefaultStepTimeoutMs = %d, want 15000", cfg.Execution.DefaultStepTimeoutMs)
	}
	// Unset values should use defaults
	if cfg.BAS.LaunchTimeoutMs != DefaultBASLaunchTimeoutMs {
		t.Errorf("LaunchTimeoutMs = %d, want default %d", cfg.BAS.LaunchTimeoutMs, DefaultBASLaunchTimeoutMs)
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(tmpDir)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
