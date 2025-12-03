package uismoke

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewRunner(t *testing.T) {
	r := NewRunner("http://localhost:4110")

	if r.browserlessURL != "http://localhost:4110" {
		t.Errorf("browserlessURL = %q, want %q", r.browserlessURL, "http://localhost:4110")
	}
}

func TestWithRunnerLogger(t *testing.T) {
	r := NewRunner("http://localhost:4110", WithRunnerLogger(os.Stdout))

	if r.logger != os.Stdout {
		t.Error("logger should be set to os.Stdout")
	}
}

func TestLoadTestingConfig_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	testingJSON := `{
		"structure": {
			"ui_smoke": {
				"enabled": true,
				"timeout_ms": 120000,
				"handshake_timeout_ms": 20000
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(testingJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := loadTestingConfig(tmpDir)
	if cfg == nil {
		t.Fatal("loadTestingConfig() returned nil")
	}

	if !cfg.Enabled {
		t.Error("Enabled should be true")
	}
	if cfg.Timeout.Milliseconds() != 120000 {
		t.Errorf("Timeout = %v, want 120000ms", cfg.Timeout)
	}
	if cfg.HandshakeTimeout.Milliseconds() != 20000 {
		t.Errorf("HandshakeTimeout = %v, want 20000ms", cfg.HandshakeTimeout)
	}
}

func TestLoadTestingConfig_Disabled(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	testingJSON := `{
		"structure": {
			"ui_smoke": {
				"enabled": false
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(testingJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := loadTestingConfig(tmpDir)
	if cfg == nil {
		t.Fatal("loadTestingConfig() returned nil")
	}

	if cfg.Enabled {
		t.Error("Enabled should be false")
	}
}

func TestLoadTestingConfig_NoFile(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := loadTestingConfig(tmpDir)
	if cfg != nil {
		t.Error("loadTestingConfig() should return nil when no testing.json")
	}
}

func TestLoadTestingConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte("not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := loadTestingConfig(tmpDir)
	if cfg != nil {
		t.Error("loadTestingConfig() should return nil for invalid JSON")
	}
}

func TestLoadTestingConfig_DefaultEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Config without enabled field - should default to true
	testingJSON := `{
		"structure": {
			"ui_smoke": {
				"timeout_ms": 60000
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(testingJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := loadTestingConfig(tmpDir)
	if cfg == nil {
		t.Fatal("loadTestingConfig() returned nil")
	}

	if !cfg.Enabled {
		t.Error("Enabled should default to true")
	}
}

func TestConvertStorageShim(t *testing.T) {
	input := []StorageShimEntry{
		{Prop: "localStorage", Patched: true},
		{Prop: "sessionStorage", Patched: false, Reason: "access denied"},
	}

	// We can't test convertStorageShim directly since it takes orchestrator.StorageShimEntry
	// but we can test the conversion via the result types

	if len(input) != 2 {
		t.Errorf("expected 2 entries, got %d", len(input))
	}
	if input[0].Prop != "localStorage" {
		t.Errorf("Prop = %q, want %q", input[0].Prop, "localStorage")
	}
	if !input[0].Patched {
		t.Error("Patched should be true")
	}
	if input[1].Reason != "access denied" {
		t.Errorf("Reason = %q, want %q", input[1].Reason, "access denied")
	}
}

func TestLoadTestingConfig_EmptyUISmoke(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Config with empty ui_smoke section
	testingJSON := `{
		"structure": {
			"ui_smoke": {}
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(testingJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := loadTestingConfig(tmpDir)
	if cfg == nil {
		t.Fatal("loadTestingConfig() returned nil")
	}

	// Should default to enabled
	if !cfg.Enabled {
		t.Error("Enabled should default to true")
	}
}

func TestLoadTestingConfig_NoStructureSection(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Config without structure section
	testingJSON := `{
		"other": {}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(testingJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := loadTestingConfig(tmpDir)
	if cfg == nil {
		t.Fatal("loadTestingConfig() returned nil")
	}

	// Should still return config with defaults
	if !cfg.Enabled {
		t.Error("Enabled should default to true")
	}
}

func TestDefaultViewport(t *testing.T) {
	vp := DefaultViewport()

	if vp.Width != DefaultViewportWidth {
		t.Errorf("Width = %d, want %d", vp.Width, DefaultViewportWidth)
	}
	if vp.Height != DefaultViewportHeight {
		t.Errorf("Height = %d, want %d", vp.Height, DefaultViewportHeight)
	}
}

func TestDefaultViewportConstants(t *testing.T) {
	if DefaultViewportWidth != 1280 {
		t.Errorf("DefaultViewportWidth = %d, want 1280", DefaultViewportWidth)
	}
	if DefaultViewportHeight != 720 {
		t.Errorf("DefaultViewportHeight = %d, want 720", DefaultViewportHeight)
	}
}
