package phases

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"test-genie/internal/orchestrator/workspace"
)

func TestRunStructurePhaseValidScenario(t *testing.T) {
	h := newStructureTestHarness(t)
	report := runStructurePhase(context.Background(), h.env, io.Discard)
	if report.Err != nil {
		t.Fatalf("expected success, got error: %v", report.Err)
	}
	if len(report.Observations) == 0 {
		t.Fatalf("expected observations to be recorded")
	}
}

func TestRunStructurePhaseHonorsAdditionalRequirements(t *testing.T) {
	h := newStructureTestHarness(t)
	h.writeTestingConfig(t, `{
  "structure": {
    "additional_dirs": ["extensions"],
    "additional_files": ["configs/custom.json"],
    "ui_smoke": {"enabled": false}
  }
}`)
	report := runStructurePhase(context.Background(), h.env, io.Discard)
	if report.Err == nil {
		t.Fatalf("expected failure when required paths missing")
	}
	if report.FailureClassification != FailureClassMisconfiguration {
		t.Fatalf("expected misconfiguration classification, got %s", report.FailureClassification)
	}
	if !strings.Contains(report.Err.Error(), "extensions") {
		t.Fatalf("expected missing directory error, got %v", report.Err)
	}

	mustMkdirAll(t, filepath.Join(h.scenarioDir, "extensions"))
	writeFile(t, filepath.Join(h.scenarioDir, "configs", "custom.json"), `{}`)
	report = runStructurePhase(context.Background(), h.env, io.Discard)
	if report.Err != nil {
		t.Fatalf("expected success after creating additional paths: %v", report.Err)
	}
}

func TestRunStructurePhaseSupportsExclusions(t *testing.T) {
	h := newStructureTestHarness(t)
	if err := os.Remove(filepath.Join(h.scenarioDir, "PRD.md")); err != nil {
		t.Fatalf("failed to remove PRD.md: %v", err)
	}
	h.writeTestingConfig(t, `{
  "structure": {
    "exclude_files": ["PRD.md"],
    "ui_smoke": {"enabled": false}
  }
}`)
	report := runStructurePhase(context.Background(), h.env, io.Discard)
	if report.Err != nil {
		t.Fatalf("expected success when PRD.md is excluded: %v", report.Err)
	}
}

func TestRunStructurePhaseDetectsInvalidJSON(t *testing.T) {
	h := newStructureTestHarness(t)
	writeFile(t, filepath.Join(h.scenarioDir, "ui", "broken.json"), `{"invalid"`)
	report := runStructurePhase(context.Background(), h.env, io.Discard)
	if report.Err == nil {
		t.Fatalf("expected failure for invalid JSON")
	}
	if !strings.Contains(report.Err.Error(), "broken.json") {
		t.Fatalf("expected broken.json to be referenced, got %v", report.Err)
	}
}

func TestRunStructurePhaseCanSkipJSONValidation(t *testing.T) {
	h := newStructureTestHarness(t)
	writeFile(t, filepath.Join(h.scenarioDir, "ui", "broken.json"), `{"invalid"`)
	h.writeTestingConfig(t, `{
  "structure": {
    "validations": {
      "check_json_validity": false
    },
    "ui_smoke": {"enabled": false}
  }
}`)
	report := runStructurePhase(context.Background(), h.env, io.Discard)
	if report.Err != nil {
		t.Fatalf("expected success when JSON validation disabled: %v", report.Err)
	}
}

func TestRunStructurePhaseCanSkipServiceNameValidation(t *testing.T) {
	h := newStructureTestHarness(t)
	writeFile(t, h.manifestPath, `{
  "service": {"name": "legacy-service"},
  "lifecycle": {"health": {"checks": [{}]}}
}`)
	h.writeTestingConfig(t, `{
  "structure": {
    "validations": {
      "service_json_name_matches_directory": false
    },
    "ui_smoke": {"enabled": false}
  }
}`)
	report := runStructurePhase(context.Background(), h.env, io.Discard)
	if report.Err != nil {
		t.Fatalf("expected success when service name validation disabled: %v", report.Err)
	}
}

type structureTestHarness struct {
	env               workspace.Environment
	scenarioDir       string
	testingConfigPath string
	manifestPath      string
}

func newStructureTestHarness(t *testing.T) *structureTestHarness {
	t.Helper()
	projectRoot := t.TempDir()
	scenarioName := "demo-scenario"
	scenarioDir := filepath.Join(projectRoot, "scenarios", scenarioName)
	mustMkdirAll(t, scenarioDir)
	subDirs := []string{
		"api",
		"cli",
		"docs",
		"requirements",
		"test/phases",
		"ui",
		filepath.Join(".vrooli"),
	}
	for _, dir := range subDirs {
		mustMkdirAll(t, filepath.Join(scenarioDir, dir))
	}

	writeFile(t, filepath.Join(scenarioDir, "api", "main.go"), "package main\nfunc main() {}\n")
	writeExecutable(t, filepath.Join(scenarioDir, "cli", "install.sh"), "#!/usr/bin/env bash\n")
	writeExecutable(t, filepath.Join(scenarioDir, "cli", scenarioName), "#!/usr/bin/env bash\necho cli\n")
	writeExecutable(t, filepath.Join(scenarioDir, "test", "run-tests.sh"), "#!/usr/bin/env bash\necho local orchestrator\n")

	for _, phase := range []string{"structure", "dependencies", "unit", "integration", "business", "performance"} {
		writeExecutable(t, filepath.Join(scenarioDir, "test", "phases", "test-"+phase+".sh"), "#!/usr/bin/env bash\n")
	}

	writeFile(t, filepath.Join(scenarioDir, "README.md"), "# Demo\n")
	writeFile(t, filepath.Join(scenarioDir, "PRD.md"), "# PRD\n")
	writeFile(t, filepath.Join(scenarioDir, "Makefile"), "all:\n\techo ok\n")
	writeFile(t, filepath.Join(scenarioDir, "ui", "sample.json"), `{"ok": true}`)

	manifestPath := filepath.Join(scenarioDir, ".vrooli", "service.json")
	writeFile(t, manifestPath, fmt.Sprintf(`{
  "service": {"name": "%s"},
  "lifecycle": {"health": {"checks": [{}]}}
}`, scenarioName))

	configPath := filepath.Join(scenarioDir, ".vrooli", "testing.json")
	writeFile(t, configPath, `{
  "structure": {
    "additional_dirs": [],
    "additional_files": [],
    "validations": {
      "service_json_name_matches_directory": true,
      "check_json_validity": true
    },
    "ui_smoke": {
      "enabled": false
    }
  }
}`)

	return &structureTestHarness{
		env: workspace.Environment{
			ScenarioName: scenarioName,
			ScenarioDir:  scenarioDir,
			TestDir:      filepath.Join(scenarioDir, "test"),
			AppRoot:      filepath.Dir(filepath.Dir(scenarioDir)),
		},
		scenarioDir:       scenarioDir,
		testingConfigPath: configPath,
		manifestPath:      manifestPath,
	}
}

func (h *structureTestHarness) writeTestingConfig(t *testing.T, content string) {
	t.Helper()
	writeFile(t, h.testingConfigPath, content)
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("failed to create directory %s: %v", path, err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	mustMkdirAll(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	mustMkdirAll(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("failed to write executable %s: %v", path, err)
	}
}
