package pipeline

import (
	"path/filepath"
	"testing"
)

func TestResolvePipelineOutputPaths_StagingUsesStorageRoot(t *testing.T) {
	storageRoot := t.TempDir()
	t.Setenv("VROOLI_STORAGE_ROOT", storageRoot)

	config := &Config{
		ScenarioName: "demo-scenario",
		LocationMode: "staging",
	}

	outputRoot, desktopPath := resolvePipelineOutputPaths(config, "/unused/scenario", "pipe-123", FrameworkElectron)

	wantRoot := filepath.Join(storageRoot, "cache", "vrooli", "scenario-to-desktop", "staging", "demo-scenario", "pipe-123")
	wantDesktop := filepath.Join(wantRoot, "platforms", FrameworkElectron)
	if outputRoot != wantRoot {
		t.Fatalf("outputRoot = %q, want %q", outputRoot, wantRoot)
	}
	if desktopPath != wantDesktop {
		t.Fatalf("desktopPath = %q, want %q", desktopPath, wantDesktop)
	}
}

func TestResolvePipelineOutputPaths_ProperUsesScenarioPath(t *testing.T) {
	scenarioPath := "/repo/scenarios/demo-scenario"
	outputRoot, desktopPath := resolvePipelineOutputPaths(&Config{
		ScenarioName: "demo-scenario",
		LocationMode: "proper",
	}, scenarioPath, "pipe-123", FrameworkElectron)

	if outputRoot != scenarioPath {
		t.Fatalf("outputRoot = %q, want %q", outputRoot, scenarioPath)
	}
	wantDesktop := filepath.Join(scenarioPath, "platforms", FrameworkElectron)
	if desktopPath != wantDesktop {
		t.Fatalf("desktopPath = %q, want %q", desktopPath, wantDesktop)
	}
}
