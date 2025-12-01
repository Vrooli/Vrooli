package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writePhaseScript(t *testing.T, dir, name, content string, exitCode int) {
	t.Helper()
	path := filepath.Join(dir, name)
	payload := fmt.Sprintf("#!/usr/bin/env bash\nset -euo pipefail\n%s\nexit %d\n", content, exitCode)
	if err := os.WriteFile(path, []byte(payload), 0o755); err != nil {
		t.Fatalf("failed to write script %s: %v", path, err)
	}
}

func createScenarioLayout(t *testing.T, root, name string) string {
	t.Helper()
	scenarioDir := filepath.Join(root, name)
	requiredDirs := []string{"api", "cli", "requirements", "ui", filepath.Join("test", "phases"), ".vrooli"}
	for _, rel := range requiredDirs {
		if err := os.MkdirAll(filepath.Join(scenarioDir, rel), 0o755); err != nil {
			t.Fatalf("failed to create required directory %s: %v", rel, err)
		}
	}
	manifest := fmt.Sprintf(`{"service":{"name":"%s"},"lifecycle":{"health":{"checks":[{"name":"api"}]}}}`, name)
	manifestPath := filepath.Join(scenarioDir, ".vrooli", "service.json")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}
	indexPath := filepath.Join(scenarioDir, "requirements", "index.json")
	if err := os.WriteFile(indexPath, []byte(`{"modules":["01-internal-orchestrator"]}`), 0o644); err != nil {
		t.Fatalf("failed to seed requirements index: %v", err)
	}
	moduleDir := filepath.Join(scenarioDir, "requirements", "01-internal-orchestrator")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("failed to create module dir: %v", err)
	}
	module := `{
  "_metadata": {"module_name":"Sample"},
  "requirements": [
    {
      "id": "REQ-1",
      "title": "Seed requirement",
      "criticality": "P0",
      "status": "planned",
      "validation": [{"type":"test","ref":"test/sample.sh"}]
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(moduleDir, "module.json"), []byte(module), 0o644); err != nil {
		t.Fatalf("failed to seed module.json: %v", err)
	}
	runnerPath := filepath.Join(scenarioDir, "test", "run-tests.sh")
	f, err := os.OpenFile(runnerPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		t.Fatalf("failed to create test runner: %v", err)
	}
	defer f.Close()
	if _, err := f.WriteString("#!/usr/bin/env bash\necho 'using scenario-local orchestrator'\n"); err != nil {
		t.Fatalf("failed to seed run-tests.sh: %v", err)
	}
	return scenarioDir
}

func TestSuiteOrchestratorExecutesPhases(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")
	phaseDir := filepath.Join(scenarioDir, "test", "phases")
	writePhaseScript(t, phaseDir, "test-unit.sh", "echo 'unit phase'", 0)
	stubCommandLookup(t, func(name string) (string, error) {
		return "/tmp/" + name, nil
	})

	orchestrator, err := NewSuiteOrchestrator(root)
	if err != nil {
		t.Fatalf("failed to init orchestrator: %v", err)
	}

	result, err := orchestrator.Execute(context.Background(), SuiteExecutionRequest{
		ScenarioName: "demo",
	})
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got failure: %#v", result)
	}
	if len(result.Phases) != 4 {
		t.Fatalf("expected four phases, got %d", len(result.Phases))
	}
	expected := []string{"structure", "dependencies", "business", "unit"}
	for _, phase := range result.Phases {
		if phase.Status != "passed" {
			t.Fatalf("phase %s expected passed, got %s", phase.Name, phase.Status)
		}
		if phase.LogPath == "" {
			t.Fatalf("phase %s missing log path", phase.Name)
		}
	}
	for i, name := range expected {
		if result.Phases[i].Name != name {
			t.Fatalf("expected phase %d to be %s but got %s", i, name, result.Phases[i].Name)
		}
	}
}

func TestSuiteOrchestratorFailFastStopsExecution(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")
	phaseDir := filepath.Join(scenarioDir, "test", "phases")
	writePhaseScript(t, phaseDir, "test-unit.sh", "echo 'unit'", 0)
	stubCommandLookup(t, func(name string) (string, error) {
		return "/tmp/" + name, nil
	})
	// Force the structure phase to fail by reintroducing the forbidden runner reference.
	runnerPath := filepath.Join(scenarioDir, "test", "run-tests.sh")
	if err := os.WriteFile(runnerPath, []byte("#!/usr/bin/env bash\necho 'scripts/scenarios/testing'\n"), 0o644); err != nil {
		t.Fatalf("failed to poison runner: %v", err)
	}

	orchestrator, err := NewSuiteOrchestrator(root)
	if err != nil {
		t.Fatalf("failed to init orchestrator: %v", err)
	}

	result, err := orchestrator.Execute(context.Background(), SuiteExecutionRequest{
		ScenarioName: "demo",
		FailFast:     true,
	})
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}
	if result.Success {
		t.Fatalf("expected failure when first phase exits non-zero")
	}
	if len(result.Phases) != 1 {
		t.Fatalf("expected a single executed phase due to fail-fast, got %d", len(result.Phases))
	}
	if result.Phases[0].Name != "structure" || result.Phases[0].Status != "failed" {
		t.Fatalf("unexpected phase result: %#v", result.Phases[0])
	}
}

func TestSuiteOrchestratorPresetFromFile(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")
	testDir := filepath.Join(scenarioDir, "test")
	phaseDir := filepath.Join(testDir, "phases")

	if err := os.WriteFile(filepath.Join(testDir, "presets.json"), []byte(`{"focused":["unit"]}`), 0o644); err != nil {
		t.Fatalf("failed to write preset: %v", err)
	}
	writePhaseScript(t, phaseDir, "test-structure.sh", "echo hi", 0)
	writePhaseScript(t, phaseDir, "test-unit.sh", "echo hi", 0)

	orchestrator, err := NewSuiteOrchestrator(root)
	if err != nil {
		t.Fatalf("failed to init orchestrator: %v", err)
	}

	result, err := orchestrator.Execute(context.Background(), SuiteExecutionRequest{
		ScenarioName: "demo",
		Preset:       "focused",
	})
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}
	if len(result.Phases) != 1 || result.Phases[0].Name != "unit" {
		t.Fatalf("expected only unit phase, got %#v", result.Phases)
	}
}

func TestSuiteOrchestratorRejectsInvalidScenarioNames(t *testing.T) {
	root := t.TempDir()
	orchestrator, err := NewSuiteOrchestrator(root)
	if err != nil {
		t.Fatalf("failed to init orchestrator: %v", err)
	}

	_, err = orchestrator.Execute(context.Background(), SuiteExecutionRequest{
		ScenarioName: "../bad",
	})
	if err == nil || !strings.Contains(err.Error(), "scenarioName") {
		t.Fatalf("expected validation error for invalid scenario name, got %v", err)
	}
}
