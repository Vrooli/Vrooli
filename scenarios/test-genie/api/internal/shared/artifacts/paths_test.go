package artifacts

import (
	"path/filepath"
	"testing"
)

func TestPhaseResultsPath(t *testing.T) {
	tests := []struct {
		name        string
		scenarioDir string
		filename    string
		want        string
	}{
		{
			name:        "smoke phase",
			scenarioDir: "/scenarios/test-scenario",
			filename:    "smoke.json",
			want:        "/scenarios/test-scenario/coverage/phase-results/smoke.json",
		},
		{
			name:        "unit phase",
			scenarioDir: "/scenarios/my-app",
			filename:    "unit.json",
			want:        "/scenarios/my-app/coverage/phase-results/unit.json",
		},
		{
			name:        "playbooks phase",
			scenarioDir: "/home/user/project",
			filename:    "playbooks.json",
			want:        "/home/user/project/coverage/phase-results/playbooks.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PhaseResultsPath(tt.scenarioDir, tt.filename)
			want := filepath.FromSlash(tt.want)
			if got != want {
				t.Errorf("PhaseResultsPath() = %q, want %q", got, want)
			}
		})
	}
}

func TestUISmokeArtifactPath(t *testing.T) {
	tests := []struct {
		name        string
		scenarioDir string
		filename    string
		want        string
	}{
		{
			name:        "screenshot",
			scenarioDir: "/scenarios/test",
			filename:    "screenshot.png",
			want:        "/scenarios/test/coverage/ui-smoke/screenshot.png",
		},
		{
			name:        "console logs",
			scenarioDir: "/scenarios/test",
			filename:    "console.json",
			want:        "/scenarios/test/coverage/ui-smoke/console.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UISmokeArtifactPath(tt.scenarioDir, tt.filename)
			want := filepath.FromSlash(tt.want)
			if got != want {
				t.Errorf("UISmokeArtifactPath() = %q, want %q", got, want)
			}
		})
	}
}

func TestVitestRequirementsPaths(t *testing.T) {
	scenarioDir := "/scenarios/test"
	paths := VitestRequirementsPaths(scenarioDir)

	// Should have exactly 3 paths
	if len(paths) != 3 {
		t.Errorf("VitestRequirementsPaths() returned %d paths, want 3", len(paths))
	}

	// First path should be the UI subdirectory (most common location)
	uiPath := filepath.FromSlash("/scenarios/test/ui/coverage/vitest-requirements.json")
	if paths[0] != uiPath {
		t.Errorf("First path should be UI location: got %q, want %q", paths[0], uiPath)
	}
}

func TestAllCoverageSubdirs(t *testing.T) {
	scenarioDir := "/scenarios/test"
	dirs := AllCoverageSubdirs(scenarioDir)

	// Should have 10 directories
	if len(dirs) != 10 {
		t.Errorf("AllCoverageSubdirs() returned %d dirs, want 10", len(dirs))
	}

	// Verify key directories are included
	expected := []string{
		"/scenarios/test/coverage/logs",
		"/scenarios/test/coverage/latest",
		"/scenarios/test/coverage/phase-results",
		"/scenarios/test/coverage/ui-smoke",
		"/scenarios/test/coverage/automation",
		"/scenarios/test/coverage/lighthouse",
		"/scenarios/test/coverage/unit",
		"/scenarios/test/coverage/sync",
		"/scenarios/test/coverage/manual-validations",
		"/scenarios/test/coverage/runtime",
	}

	for i, exp := range expected {
		want := filepath.FromSlash(exp)
		if dirs[i] != want {
			t.Errorf("AllCoverageSubdirs()[%d] = %q, want %q", i, dirs[i], want)
		}
	}
}

func TestRelativePaths(t *testing.T) {
	tests := []struct {
		name     string
		fn       func(string) string
		filename string
		want     string
	}{
		{
			name:     "phase results",
			fn:       RelativePhaseResultsPath,
			filename: "unit.json",
			want:     "coverage/phase-results/unit.json",
		},
		{
			name:     "ui smoke",
			fn:       RelativeUISmokeArtifactPath,
			filename: "screenshot.png",
			want:     "coverage/ui-smoke/screenshot.png",
		},
		{
			name:     "automation",
			fn:       RelativeAutomationArtifactPath,
			filename: "workflow.timeline.json",
			want:     "coverage/automation/workflow.timeline.json",
		},
		{
			name:     "lighthouse",
			fn:       RelativeLighthouseArtifactPath,
			filename: "home.json",
			want:     "coverage/lighthouse/home.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn(tt.filename)
			want := filepath.FromSlash(tt.want)
			if got != want {
				t.Errorf("%s() = %q, want %q", tt.name, got, want)
			}
		})
	}
}

func TestSeedStatePath(t *testing.T) {
	scenarioDir := "/scenarios/browser-automation"
	got := SeedStatePath(scenarioDir)
	want := filepath.FromSlash("/scenarios/browser-automation/coverage/runtime/seed-state.json")
	if got != want {
		t.Errorf("SeedStatePath() = %q, want %q", got, want)
	}
}

func TestSyncMetadataPath(t *testing.T) {
	scenarioDir := "/scenarios/my-app"
	got := SyncMetadataPath(scenarioDir)
	want := filepath.FromSlash("/scenarios/my-app/coverage/sync/latest.json")
	if got != want {
		t.Errorf("SyncMetadataPath() = %q, want %q", got, want)
	}
}

func TestManualValidationsPath(t *testing.T) {
	scenarioDir := "/scenarios/my-app"
	got := ManualValidationsPath(scenarioDir)
	want := filepath.FromSlash("/scenarios/my-app/coverage/manual-validations/log.jsonl")
	if got != want {
		t.Errorf("ManualValidationsPath() = %q, want %q", got, want)
	}
}

func TestConstantConsistency(t *testing.T) {
	// Verify that PhaseResultsDir constant matches what's used in paths.go and writer.go
	if PhaseResultsDir != "coverage/phase-results" {
		t.Errorf("PhaseResultsDir = %q, want %q", PhaseResultsDir, "coverage/phase-results")
	}

	// Verify automation dir matches playbooks constant
	if AutomationDir != "coverage/automation" {
		t.Errorf("AutomationDir = %q, want %q", AutomationDir, "coverage/automation")
	}

	// Verify lighthouse dir
	if LighthouseDir != "coverage/lighthouse" {
		t.Errorf("LighthouseDir = %q, want %q", LighthouseDir, "coverage/lighthouse")
	}
}

func TestLogsAndLatestPaths(t *testing.T) {
	scenarioDir := "/scenarios/demo"
	runID := "20251208-151044"

	if got, want := RunLogsDir(scenarioDir, runID), filepath.FromSlash("/scenarios/demo/coverage/logs/20251208-151044"); got != want {
		t.Errorf("RunLogsDir() = %q, want %q", got, want)
	}

	if got, want := LatestDirPath(scenarioDir), filepath.FromSlash("/scenarios/demo/coverage/latest"); got != want {
		t.Errorf("LatestDirPath() = %q, want %q", got, want)
	}

	if got, want := LatestManifestPath(scenarioDir), filepath.FromSlash("/scenarios/demo/coverage/latest/manifest.json"); got != want {
		t.Errorf("LatestManifestPath() = %q, want %q", got, want)
	}
}
