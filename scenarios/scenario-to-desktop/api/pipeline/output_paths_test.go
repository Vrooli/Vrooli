package pipeline

import (
	"path/filepath"
	"testing"

	"scenario-to-desktop-api/generation"
)

func TestResolvePipelineOutputPaths_StagingUsesStorageRoot(t *testing.T) {
	storageRoot := t.TempDir()
	t.Setenv("VROOLI_STORAGE_ROOT", storageRoot)
	// The test owns the live namespace. Do not let a lifecycle-injected
	// namespace from the surrounding process redirect this fixture.
	t.Setenv("VROOLI_STORAGE_NAMESPACE", "")
	t.Setenv("VROOLI_SCENARIO", "")

	config := &PipelineConfig{
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

func TestResolvePipelineOutputPathsUsesRegisteredStrategy(t *testing.T) {
	config := &PipelineConfig{ScenarioName: "demo", StrategyID: generation.DefaultStrategyID}
	_, path := resolvePipelineOutputPaths(config, "/scenarios/demo", "", config.StrategyID)
	want := filepath.Join("/scenarios/demo", "platforms", "electron")
	if path != want {
		t.Fatalf("path=%q want %q", path, want)
	}
	config.StrategyID = "unregistered"
	_, path = resolvePipelineOutputPaths(config, "/scenarios/demo", "", config.StrategyID)
	if path != "" {
		t.Fatalf("unregistered strategy path=%q", path)
	}
}

func TestResolvePipelineOutputPaths_ProperUsesScenarioPath(t *testing.T) {
	scenarioPath := "/repo/scenarios/demo-scenario"
	outputRoot, desktopPath := resolvePipelineOutputPaths(&PipelineConfig{
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
