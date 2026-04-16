package lint

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultSettings(t *testing.T) {
	settings := DefaultSettings()

	if !settings.Handlers[HandlerGoModule].EnabledOrDefault() {
		t.Fatal("expected go_module handler enabled by default")
	}
	if settings.Policy.UnconfiguredCommonComponents["api"] != PolicySeverityError {
		t.Fatalf("expected api policy to default to error, got %q", settings.Policy.UnconfiguredCommonComponents["api"])
	}
	if settings.Policy.UnmatchedCodeComponents != PolicySeverityWarning {
		t.Fatalf("expected unmatched code policy warning, got %q", settings.Policy.UnmatchedCodeComponents)
	}
	if len(settings.Ignore) == 0 {
		t.Fatal("expected default ignore list")
	}
}

func TestHandlerSettingsEnabledOrDefault(t *testing.T) {
	if !(HandlerSettings{}).EnabledOrDefault() {
		t.Fatal("nil enabled should default to true")
	}
	disabled := false
	if (HandlerSettings{Enabled: &disabled}).EnabledOrDefault() {
		t.Fatal("expected explicit false to disable handler")
	}
}

func TestLoadSettings_NoFile(t *testing.T) {
	tempDir := t.TempDir()
	settings, err := LoadSettings(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !settings.Handlers[HandlerNodePackage].EnabledOrDefault() {
		t.Fatal("expected default settings when file absent")
	}
}

func TestLoadSettings_OverridesHandlersPolicyAndComponents(t *testing.T) {
	tempDir := t.TempDir()
	vrooliDir := filepath.Join(tempDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}

	config := `{
		"lint": {
			"handlers": {
				"go_module": {"strict": true},
				"python_project": {"enabled": false}
			},
			"policy": {
				"unconfigured_common_components": {
					"api": "error",
					"ui": "info"
				},
				"unmatched_code_components": "info"
			},
			"components": {
				"worker": {"handler": "go_module", "strict": true}
			},
			"ignore": ["docs", "fixtures"]
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(config), 0o644); err != nil {
		t.Fatalf("failed to write testing.json: %v", err)
	}

	settings, err := LoadSettings(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !settings.Handlers[HandlerGoModule].Strict {
		t.Fatal("expected go_module strict override")
	}
	if settings.Handlers[HandlerPythonProject].EnabledOrDefault() {
		t.Fatal("expected python_project to be disabled")
	}
	if settings.Policy.UnconfiguredCommonComponents["ui"] != PolicySeverityInfo {
		t.Fatal("expected ui policy override")
	}
	if settings.Policy.UnmatchedCodeComponents != PolicySeverityInfo {
		t.Fatal("expected unmatched code policy override")
	}
	if settings.Components["worker"].Handler != HandlerGoModule {
		t.Fatal("expected worker handler override")
	}
	if len(settings.Ignore) != 2 || settings.Ignore[1] != "fixtures" {
		t.Fatalf("unexpected ignore list: %v", settings.Ignore)
	}
}

func TestLoadSettings_InvalidPolicySeverity(t *testing.T) {
	tempDir := t.TempDir()
	vrooliDir := filepath.Join(tempDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}
	config := `{"lint": {"policy": {"unmatched_code_components": "fatal"}}}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(config), 0o644); err != nil {
		t.Fatalf("failed to write testing.json: %v", err)
	}

	if _, err := LoadSettings(tempDir); err == nil {
		t.Fatal("expected invalid policy severity error")
	}
}
