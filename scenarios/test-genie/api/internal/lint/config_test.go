package lint

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultSettings(t *testing.T) {
	settings := DefaultSettings()

	// By default, all languages are enabled
	if !settings.Go.IsEnabled() {
		t.Error("Go should be enabled by default")
	}
	if !settings.Node.IsEnabled() {
		t.Error("Node should be enabled by default")
	}
	if !settings.Python.IsEnabled() {
		t.Error("Python should be enabled by default")
	}

	// By default, strict mode is off
	if settings.Go.Strict {
		t.Error("Go strict mode should be off by default")
	}
	if settings.Node.Strict {
		t.Error("Node strict mode should be off by default")
	}
	if settings.Python.Strict {
		t.Error("Python strict mode should be off by default")
	}
}

func TestLanguageSettings_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		enabled  *bool
		expected bool
	}{
		{"nil is enabled", nil, true},
		{"true is enabled", boolPtr(true), true},
		{"false is disabled", boolPtr(false), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := LanguageSettings{Enabled: tt.enabled}
			if s.IsEnabled() != tt.expected {
				t.Errorf("IsEnabled() = %v, want %v", s.IsEnabled(), tt.expected)
			}
		})
	}
}

func TestLoadSettings_NoFile(t *testing.T) {
	tempDir := t.TempDir()

	settings, err := LoadSettings(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return default settings when no file exists
	if !settings.Go.IsEnabled() {
		t.Error("Go should be enabled when no config file")
	}
}

func TestLoadSettings_EmptyLintSection(t *testing.T) {
	tempDir := t.TempDir()
	vrooliDir := filepath.Join(tempDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}

	// testing.json without lint section
	config := `{"structure": {}}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(config), 0o644); err != nil {
		t.Fatalf("failed to write testing.json: %v", err)
	}

	settings, err := LoadSettings(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return default settings when lint section is missing
	if !settings.Go.IsEnabled() {
		t.Error("Go should be enabled when lint section missing")
	}
}

func TestLoadSettings_DisableLanguage(t *testing.T) {
	tempDir := t.TempDir()
	vrooliDir := filepath.Join(tempDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}

	config := `{
		"lint": {
			"languages": {
				"go": {"enabled": false},
				"node": {"enabled": true}
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(config), 0o644); err != nil {
		t.Fatalf("failed to write testing.json: %v", err)
	}

	settings, err := LoadSettings(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if settings.Go.IsEnabled() {
		t.Error("Go should be disabled")
	}
	if !settings.Node.IsEnabled() {
		t.Error("Node should be enabled")
	}
	// Python not specified, should use default (enabled)
	if !settings.Python.IsEnabled() {
		t.Error("Python should be enabled by default")
	}
}

func TestLoadSettings_StrictMode(t *testing.T) {
	tempDir := t.TempDir()
	vrooliDir := filepath.Join(tempDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}

	config := `{
		"lint": {
			"languages": {
				"go": {"strict": true},
				"node": {"strict": false}
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(config), 0o644); err != nil {
		t.Fatalf("failed to write testing.json: %v", err)
	}

	settings, err := LoadSettings(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !settings.Go.Strict {
		t.Error("Go strict should be true")
	}
	if settings.Node.Strict {
		t.Error("Node strict should be false")
	}
	// Python not specified, should use default (not strict)
	if settings.Python.Strict {
		t.Error("Python strict should be false by default")
	}
}

func TestLoadSettings_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	vrooliDir := filepath.Join(tempDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte("not json"), 0o644); err != nil {
		t.Fatalf("failed to write testing.json: %v", err)
	}

	_, err := LoadSettings(tempDir)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func boolPtr(b bool) *bool {
	return &b
}
