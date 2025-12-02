package orchestrator

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	phasespkg "test-genie/internal/orchestrator/phases"
	reqsync "test-genie/internal/orchestrator/requirements"
	workspacepkg "test-genie/internal/orchestrator/workspace"
)

type stubRequirementsSyncer struct {
	calls int
	last  reqsync.SyncInput
	err   error
}

func (s *stubRequirementsSyncer) Sync(ctx context.Context, input reqsync.SyncInput) error {
	s.calls++
	s.last = input
	return s.err
}

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
	requiredDirs := []string{
		"api",
		"cli",
		"requirements",
		"ui",
		"docs",
		filepath.Join("test", "phases"),
		filepath.Join("test", "lib"),
		filepath.Join("test", "playbooks"),
		".vrooli",
	}
	for _, rel := range requiredDirs {
		if err := os.MkdirAll(filepath.Join(scenarioDir, rel), 0o755); err != nil {
			t.Fatalf("failed to create required directory %s: %v", rel, err)
		}
	}
	apiEntryPoint := filepath.Join(scenarioDir, "api", "main.go")
	if err := os.WriteFile(apiEntryPoint, []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("failed to seed api/main.go: %v", err)
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
	cliPath := filepath.Join(scenarioDir, "cli", "test-genie")
	if err := os.WriteFile(cliPath, []byte("#!/usr/bin/env bash\necho cli\n"), 0o755); err != nil {
		t.Fatalf("failed to seed cli binary: %v", err)
	}
	scenarioCLI := filepath.Join(scenarioDir, "cli", name)
	if err := os.WriteFile(scenarioCLI, []byte("#!/usr/bin/env bash\necho scenario cli\n"), 0o755); err != nil {
		t.Fatalf("failed to seed scenario cli: %v", err)
	}
	installScript := filepath.Join(scenarioDir, "cli", "install.sh")
	if err := os.WriteFile(installScript, []byte("#!/usr/bin/env bash\necho install\n"), 0o755); err != nil {
		t.Fatalf("failed to seed cli/install.sh: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "test-genie.bats"), []byte("#!/usr/bin/env bash\n"), 0o644); err != nil {
		t.Fatalf("failed to seed cli/test-genie.bats: %v", err)
	}
	for _, phase := range []string{"structure", "dependencies", "unit", "integration", "playbooks", "business", "performance"} {
		writePhaseScript(t, filepath.Join(scenarioDir, "test", "phases"), "test-"+phase+".sh", "echo "+phase, 0)
	}
	playbookRegistry := fmt.Sprintf(`{"scenario":"%s","playbooks":[]}`, name)
	registryPath := filepath.Join(scenarioDir, "test", "playbooks", "registry.json")
	if err := os.WriteFile(registryPath, []byte(playbookRegistry), 0o644); err != nil {
		t.Fatalf("failed to seed playbook registry: %v", err)
	}
	libDir := filepath.Join(scenarioDir, "test", "lib")
	libFiles := map[string]string{
		"runtime.sh":      "#!/usr/bin/env bash\necho runtime\n",
		"orchestrator.sh": "#!/usr/bin/env bash\necho orchestrator\n",
	}
	for name, contents := range libFiles {
		if err := os.WriteFile(filepath.Join(libDir, name), []byte(contents), 0o755); err != nil {
			t.Fatalf("failed to seed %s: %v", name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "README.md"), []byte("# README\n"), 0o644); err != nil {
		t.Fatalf("failed to seed README: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "PRD.md"), []byte("# PRD\n"), 0o644); err != nil {
		t.Fatalf("failed to seed PRD: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "Makefile"), []byte("all:\n\techo ok\n"), 0o644); err != nil {
		t.Fatalf("failed to seed Makefile: %v", err)
	}
	testingConfig := filepath.Join(scenarioDir, ".vrooli", "testing.json")
	if err := os.WriteFile(testingConfig, []byte(`{"structure":{}}`), 0o644); err != nil {
		t.Fatalf("failed to seed testing config: %v", err)
	}
	return scenarioDir
}

func skipPlaybooksForTests(t *testing.T) {
	t.Helper()
	t.Setenv("TEST_GENIE_SKIP_PLAYBOOKS", "1")
}

func TestSuiteOrchestratorExecutesPhases(t *testing.T) {
	t.Run("[REQ:TESTGENIE-ORCH-P0] orchestrator runs go-native phases", func(t *testing.T) {
		skipPlaybooksForTests(t)
		root := t.TempDir()
		scenarioDir := createScenarioLayout(t, root, "demo")
		phaseDir := filepath.Join(scenarioDir, "test", "phases")
		writePhaseScript(t, phaseDir, "test-unit.sh", "echo 'unit phase'", 0)
		stubCommandLookup(t, func(name string) (string, error) {
			return "/tmp/" + name, nil
		})
		stubPhaseCommandExecutor(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			return nil
		})
		stubPhaseCommandCapture(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error) {
			switch {
			case strings.HasSuffix(name, filepath.Join("cli", "test-genie")) && len(args) > 0 && args[0] == "version":
				return "test-genie version 1.0.0", nil
			default:
				return "", nil
			}
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
		if len(result.Phases) != 7 {
			t.Fatalf("expected seven phases, got %d", len(result.Phases))
		}
		expected := []string{"structure", "dependencies", "unit", "integration", "playbooks", "business", "performance"}
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
	})
}

func TestSuiteOrchestratorSyncsRequirementsAfterFullRun(t *testing.T) {
	t.Run("[REQ:TESTGENIE-ORCH-P0] full suites trigger requirement sync", func(t *testing.T) {
		skipPlaybooksForTests(t)
		root := t.TempDir()
		scenarioDir := createScenarioLayout(t, root, "demo")
		phaseDir := filepath.Join(scenarioDir, "test", "phases")
		writePhaseScript(t, phaseDir, "test-unit.sh", "echo unit", 0)

		stubCommandLookup(t, func(name string) (string, error) {
			return "/tmp/" + name, nil
		})
		stubPhaseCommandExecutor(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			return nil
		})
		stubPhaseCommandCapture(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error) {
			switch {
			case strings.HasSuffix(name, filepath.Join("cli", "test-genie")) && len(args) > 0 && args[0] == "version":
				return "test-genie version 1.0.0", nil
			default:
				return "", nil
			}
		})

		orchestrator, err := NewSuiteOrchestrator(root)
		if err != nil {
			t.Fatalf("failed to init orchestrator: %v", err)
		}
		stubSyncer := &stubRequirementsSyncer{}
		orchestrator.requirements = stubSyncer

		result, err := orchestrator.Execute(context.Background(), SuiteExecutionRequest{
			ScenarioName: "demo",
		})
		if err != nil {
			t.Fatalf("execution failed: %v", err)
		}
		if !result.Success {
			t.Fatalf("expected orchestration to succeed")
		}
		if stubSyncer.calls != 1 {
			t.Fatalf("expected requirements sync to run once, got %d", stubSyncer.calls)
		}
		if stubSyncer.last.ScenarioName != "demo" {
			t.Fatalf("unexpected scenario name in sync payload: %#v", stubSyncer.last)
		}
		if len(stubSyncer.last.PhaseResults) != len(stubSyncer.last.PhaseDefinitions) {
			t.Fatalf("expected phase metadata to be recorded for sync")
		}
	})
}

func TestSuiteOrchestratorFailFastStopsExecution(t *testing.T) {
	t.Run("[REQ:TESTGENIE-ORCH-P0] fail-fast halts remaining phases", func(t *testing.T) {
		skipPlaybooksForTests(t)
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
	})
}

func TestSuiteOrchestratorPresetFromFile(t *testing.T) {
	t.Run("[REQ:TESTGENIE-ORCH-P0] custom presets are honored", func(t *testing.T) {
		root := t.TempDir()
		scenarioDir := createScenarioLayout(t, root, "demo")
		testDir := filepath.Join(scenarioDir, "test")
		phaseDir := filepath.Join(testDir, "phases")

		if err := os.WriteFile(filepath.Join(testDir, "presets.json"), []byte(`{"focused":["unit"]}`), 0o644); err != nil {
			t.Fatalf("failed to write preset: %v", err)
		}
		writePhaseScript(t, phaseDir, "test-structure.sh", "echo hi", 0)
		writePhaseScript(t, phaseDir, "test-unit.sh", "echo hi", 0)
		stubPhaseCommandExecutor(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			return nil
		})

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
	})
}

func TestSuiteOrchestratorRejectsInvalidScenarioNames(t *testing.T) {
	t.Run("[REQ:TESTGENIE-ORCH-P0] scenario names are validated", func(t *testing.T) {
		skipPlaybooksForTests(t)
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
	})
}

func TestSuiteOrchestratorHonorsTestingConfigPhaseToggles(t *testing.T) {
	t.Run("[REQ:TESTGENIE-ORCH-P0] testing config disables phases", func(t *testing.T) {
		skipPlaybooksForTests(t)
		root := t.TempDir()
		scenarioDir := createScenarioLayout(t, root, "demo")
		configPath := filepath.Join(scenarioDir, ".vrooli", "testing.json")
		if err := os.WriteFile(configPath, []byte(`{"phases":{"integration":{"enabled":false}}}`), 0o644); err != nil {
			t.Fatalf("failed to write testing config: %v", err)
		}
		stubCommandLookup(t, func(name string) (string, error) {
			return "/tmp/" + name, nil
		})
		stubPhaseCommandExecutor(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			return nil
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
		for _, phase := range result.Phases {
			if phase.Name == "integration" {
				t.Fatalf("expected integration phase to be disabled via testing config")
			}
		}
		if len(result.Phases) != 6 {
			t.Fatalf("expected six phases after disabling integration, got %d", len(result.Phases))
		}
	})
}

func TestSuiteOrchestratorHonorsTestingConfigPresets(t *testing.T) {
	t.Run("[REQ:TESTGENIE-ORCH-P0] config presets constrain execution order", func(t *testing.T) {
		skipPlaybooksForTests(t)
		root := t.TempDir()
		scenarioDir := createScenarioLayout(t, root, "demo")
		configPath := filepath.Join(scenarioDir, ".vrooli", "testing.json")
		if err := os.WriteFile(configPath, []byte(`{"presets":{"focused":["unit","performance"]}}`), 0o644); err != nil {
			t.Fatalf("failed to write testing config: %v", err)
		}
		stubCommandLookup(t, func(name string) (string, error) {
			return "/tmp/" + name, nil
		})
		stubPhaseCommandExecutor(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			return nil
		})

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
		if len(result.Phases) != 2 {
			t.Fatalf("expected preset to run two phases, got %d", len(result.Phases))
		}
		if result.Phases[0].Name != "unit" || result.Phases[1].Name != "performance" {
			t.Fatalf("unexpected preset order: %#v", result.Phases)
		}
		if result.PresetUsed != "focused" {
			t.Fatalf("expected presetUsed to be recorded, got %s", result.PresetUsed)
		}
	})
}

func TestSuiteOrchestratorRespectsPhaseTimeoutOverrides(t *testing.T) {
	t.Run("[REQ:TESTGENIE-ORCH-P0] per-phase timeouts guard against hangs", func(t *testing.T) {
		skipPlaybooksForTests(t)
		root := t.TempDir()
		scenarioDir := createScenarioLayout(t, root, "demo")
		configPath := filepath.Join(scenarioDir, ".vrooli", "testing.json")
		if err := os.WriteFile(configPath, []byte(`{"phases":{"slow":{"timeout":"1s"}}}`), 0o644); err != nil {
			t.Fatalf("failed to write testing config: %v", err)
		}

		orchestrator, err := NewSuiteOrchestrator(root)
		if err != nil {
			t.Fatalf("failed to init orchestrator: %v", err)
		}
		orchestrator.catalog.Register(phaseSpec{
			Name: PhaseName("slow"),
			Runner: func(ctx context.Context, env PhaseEnvironment, logWriter io.Writer) PhaseRunReport {
				select {
				case <-ctx.Done():
					return PhaseRunReport{Err: ctx.Err()}
				case <-time.After(2 * time.Second):
					return PhaseRunReport{}
				}
			},
		})

		result, err := orchestrator.Execute(context.Background(), SuiteExecutionRequest{
			ScenarioName: "demo",
			Phases:       []string{"slow"},
		})
		if err != nil {
			t.Fatalf("execution failed: %v", err)
		}
		if len(result.Phases) != 1 {
			t.Fatalf("expected only slow phase to run, got %d phases", len(result.Phases))
		}
		phase := result.Phases[0]
		if phase.Status != "failed" {
			t.Fatalf("expected slow phase to fail due to timeout, got %s", phase.Status)
		}
		if phase.Classification != failureClassTimeout {
			t.Fatalf("expected timeout classification, got %s", phase.Classification)
		}
		if !strings.Contains(phase.Error, "timed out") {
			t.Fatalf("expected timeout message, got %s", phase.Error)
		}
	})
}

func TestRequirementsSyncDecision(t *testing.T) {
	defs := []phaseDefinition{{Name: PhaseStructure}, {Name: PhaseUnit}}
	selected := append([]phaseDefinition(nil), defs...)
	fullPlan := &phasePlan{
		Definitions: defs,
		Selected:    selected,
	}
	results := []PhaseExecutionResult{
		{Name: "structure", Status: "passed"},
		{Name: "unit", Status: "failed"},
	}

	t.Run("[REQ:TESTGENIE-ORCH-P0] full suite attempts sync even on failure", func(t *testing.T) {
		t.Setenv("TESTING_REQUIREMENTS_SYNC", "")
		decision := newRequirementsSyncDecision(nil, fullPlan, results)
		if !decision.Execute {
			t.Fatalf("expected sync decision to allow execution: %+v", decision)
		}
	})

	t.Run("[REQ:TESTGENIE-ORCH-P0] config flag disables sync", func(t *testing.T) {
		cfg := &workspacepkg.Config{
			Requirements: workspacepkg.RequirementSettings{
				Sync: boolPtr(false),
			},
		}
		if decision := newRequirementsSyncDecision(cfg, fullPlan, results); decision.Execute {
			t.Fatalf("expected config-disabled sync to be skipped")
		}
	})

	t.Run("[REQ:TESTGENIE-ORCH-P0] env flag disables sync", func(t *testing.T) {
		t.Setenv("TESTING_REQUIREMENTS_SYNC", "0")
		if decision := newRequirementsSyncDecision(nil, fullPlan, results); decision.Execute {
			t.Fatalf("expected env-disabled sync to be skipped")
		}
	})

	t.Run("[REQ:TESTGENIE-ORCH-P0] missing required phases block sync", func(t *testing.T) {
		t.Setenv("TESTING_REQUIREMENTS_SYNC", "")
		partialResults := results[:1]
		if decision := newRequirementsSyncDecision(nil, fullPlan, partialResults); decision.Execute {
			t.Fatalf("expected missing phases to skip sync")
		}
	})

	t.Run("[REQ:TESTGENIE-ORCH-P0] force flag overrides missing phases", func(t *testing.T) {
		t.Setenv("TESTING_REQUIREMENTS_SYNC_FORCE", "1")
		partialResults := results[:1]
		decision := newRequirementsSyncDecision(nil, fullPlan, partialResults)
		if !decision.Execute || !decision.Forced {
			t.Fatalf("expected forced sync to execute: %+v", decision)
		}
	})
}

func TestBuildCommandHistory(t *testing.T) {
	req := SuiteExecutionRequest{
		ScenarioName: "demo",
		Phases:       []string{"structure"},
		Skip:         []string{"unit"},
		FailFast:     true,
	}
	plan := &phasePlan{
		PresetUsed: "quick",
		Selected:   []phaseDefinition{{Name: PhaseStructure}, {Name: PhaseDependencies}},
	}

	history := buildCommandHistory(req, plan)
	if len(history) != 2 {
		t.Fatalf("expected two history entries, got %d", len(history))
	}
	if history[0] == "" || history[1] == "" {
		t.Fatalf("history entries should not be empty: %#v", history)
	}
}

func TestSelectPhases(t *testing.T) {
	defs := []phaseDefinition{
		{Name: PhaseStructure},
		{Name: PhaseDependencies},
		{Name: PhaseUnit},
	}
	presets := map[string][]string{
		"quick": {"structure", "unit"},
	}

	t.Run("defaults to all phases when no hints provided", func(t *testing.T) {
		selected, preset, err := selectPhases(defs, presets, SuiteExecutionRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if preset != "" {
			t.Fatalf("expected empty preset usage, got %s", preset)
		}
		if len(selected) != len(defs) {
			t.Fatalf("expected %d phases, got %d", len(defs), len(selected))
		}
	})

	t.Run("resolves presets case-insensitively", func(t *testing.T) {
		selected, preset, err := selectPhases(defs, presets, SuiteExecutionRequest{Preset: "Quick"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if preset != "quick" {
			t.Fatalf("expected preset quick, got %s", preset)
		}
		if len(selected) != 2 || selected[0].Name != PhaseStructure || selected[1].Name != PhaseUnit {
			t.Fatalf("unexpected preset selection: %#v", selected)
		}
	})

	t.Run("errors when requested phase missing", func(t *testing.T) {
		_, _, err := selectPhases(defs, presets, SuiteExecutionRequest{Phases: []string{"structure", "invalid"}})
		if err == nil {
			t.Fatalf("expected error for invalid phase selection")
		}
	})

	t.Run("applies skip list", func(t *testing.T) {
		selected, _, err := selectPhases(defs, presets, SuiteExecutionRequest{Skip: []string{"dependencies"}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, def := range selected {
			if def.Name == PhaseDependencies {
				t.Fatalf("dependency phase should have been skipped")
			}
		}
	})
}

func boolPtr(value bool) *bool {
	v := value
	return &v
}

func stubCommandLookup(t *testing.T, fn func(string) (string, error)) {
	t.Helper()
	restore := phasespkg.OverrideCommandLookup(fn)
	t.Cleanup(restore)
}

func stubPhaseCommandExecutor(t *testing.T, fn func(context.Context, string, io.Writer, string, ...string) error) {
	t.Helper()
	restore := phasespkg.OverrideCommandExecutor(fn)
	t.Cleanup(restore)
}

func stubPhaseCommandCapture(t *testing.T, fn func(context.Context, string, io.Writer, string, ...string) (string, error)) {
	t.Helper()
	restore := phasespkg.OverrideCommandCapture(fn)
	t.Cleanup(restore)
}
