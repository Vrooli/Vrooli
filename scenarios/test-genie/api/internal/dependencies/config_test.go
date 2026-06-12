package dependencies

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSettingsRejectsInvalidPolicy(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".vrooli", "testing.json"), `{
  "dependencies": {
    "resources": {"health_policy": "maybe"}
  }
}`)
	_, err := LoadSettings(dir)
	if err == nil {
		t.Fatal("expected invalid policy error")
	}
	if !strings.Contains(err.Error(), "resources.health_policy") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadSettingsAppliesRuntimeVersionOverride(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".vrooli", "testing.json"), `{
  "dependencies": {
    "runtime_versions": {"go": ">=1.25.0"}
  }
}`)
	settings, err := LoadSettings(dir)
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	if settings.RuntimeVersions["go"] != ">=1.25.0" {
		t.Fatalf("go runtime version = %q", settings.RuntimeVersions["go"])
	}
}
