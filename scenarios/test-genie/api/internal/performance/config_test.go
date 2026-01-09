package performance

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadExpectations_DefaultsWhenNoFile(t *testing.T) {
	scenarioDir := t.TempDir()

	exp, err := LoadExpectations(scenarioDir)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exp.GoBuildMaxDuration != 90*time.Second {
		t.Errorf("expected GoBuildMaxDuration 90s, got %s", exp.GoBuildMaxDuration)
	}
	if exp.UIBuildMaxDuration != 180*time.Second {
		t.Errorf("expected UIBuildMaxDuration 180s, got %s", exp.UIBuildMaxDuration)
	}
	if !exp.RequireGoBuild {
		t.Error("expected RequireGoBuild to default to true")
	}
	if exp.RequireUIBuild {
		t.Error("expected RequireUIBuild to default to false")
	}
}

func TestLoadExpectations_LoadsFromFile(t *testing.T) {
	scenarioDir := t.TempDir()
	vrooliDir := filepath.Join(scenarioDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}

	content := `{
		"performance": {
			"go_build_max_seconds": 120,
			"ui_build_max_seconds": 240,
			"require_go_build": false,
			"require_ui_build": true
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write testing.json: %v", err)
	}

	exp, err := LoadExpectations(scenarioDir)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exp.GoBuildMaxDuration != 120*time.Second {
		t.Errorf("expected GoBuildMaxDuration 120s, got %s", exp.GoBuildMaxDuration)
	}
	if exp.UIBuildMaxDuration != 240*time.Second {
		t.Errorf("expected UIBuildMaxDuration 240s, got %s", exp.UIBuildMaxDuration)
	}
	if exp.RequireGoBuild {
		t.Error("expected RequireGoBuild to be false")
	}
	if !exp.RequireUIBuild {
		t.Error("expected RequireUIBuild to be true")
	}
}

func TestLoadExpectations_PartialOverrides(t *testing.T) {
	scenarioDir := t.TempDir()
	vrooliDir := filepath.Join(scenarioDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}

	// Only override go_build_max_seconds
	content := `{
		"performance": {
			"go_build_max_seconds": 60
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write testing.json: %v", err)
	}

	exp, err := LoadExpectations(scenarioDir)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exp.GoBuildMaxDuration != 60*time.Second {
		t.Errorf("expected GoBuildMaxDuration 60s, got %s", exp.GoBuildMaxDuration)
	}
	// Others should be defaults
	if exp.UIBuildMaxDuration != 180*time.Second {
		t.Errorf("expected UIBuildMaxDuration 180s (default), got %s", exp.UIBuildMaxDuration)
	}
}

func TestLoadExpectations_InvalidJSON(t *testing.T) {
	scenarioDir := t.TempDir()
	vrooliDir := filepath.Join(scenarioDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte("not json"), 0o644); err != nil {
		t.Fatalf("failed to write testing.json: %v", err)
	}

	_, err := LoadExpectations(scenarioDir)

	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadExpectations_EmptyPerformanceSection(t *testing.T) {
	scenarioDir := t.TempDir()
	vrooliDir := filepath.Join(scenarioDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}

	// Valid JSON but empty performance section
	content := `{"version": "1.0.0"}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write testing.json: %v", err)
	}

	exp, err := LoadExpectations(scenarioDir)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should use all defaults
	if exp.GoBuildMaxDuration != 90*time.Second {
		t.Errorf("expected GoBuildMaxDuration 90s, got %s", exp.GoBuildMaxDuration)
	}
}

func TestDefaultExpectations(t *testing.T) {
	exp := DefaultExpectations()

	if exp.GoBuildMaxDuration != 90*time.Second {
		t.Errorf("expected GoBuildMaxDuration 90s, got %s", exp.GoBuildMaxDuration)
	}
	if exp.UIBuildMaxDuration != 180*time.Second {
		t.Errorf("expected UIBuildMaxDuration 180s, got %s", exp.UIBuildMaxDuration)
	}
	if !exp.RequireGoBuild {
		t.Error("expected RequireGoBuild to be true")
	}
	if exp.RequireUIBuild {
		t.Error("expected RequireUIBuild to be false")
	}
}
