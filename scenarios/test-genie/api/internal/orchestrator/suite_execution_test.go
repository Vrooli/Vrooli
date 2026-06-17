package orchestrator

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	phasespkg "test-genie/internal/orchestrator/phases"
	reqsync "test-genie/internal/orchestrator/requirements"
	"test-genie/internal/orchestrator/runnability"
	"test-genie/internal/orchestrator/targetruntime"
	workspacepkg "test-genie/internal/orchestrator/workspace"
)

type stubRequirementsSyncer struct {
	calls         int
	snapshotCalls int
	last          reqsync.SyncInput
	err           error
	outcome       *reqsync.SyncOutcome
}

func (s *stubRequirementsSyncer) Sync(ctx context.Context, input reqsync.SyncInput) (*reqsync.SyncOutcome, error) {
	s.calls++
	s.last = input
	return s.outcome, s.err
}

func (s *stubRequirementsSyncer) Snapshot(ctx context.Context, input reqsync.SyncInput) (*reqsync.SyncOutcome, error) {
	s.snapshotCalls++
	s.last = input
	return s.outcome, s.err
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
		"test",
		"bas",
		filepath.Join("bas", "cases"),
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
	goMod := filepath.Join(scenarioDir, "api", "go.mod")
	if err := os.WriteFile(goMod, []byte("module "+name+"\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("failed to seed api/go.mod: %v", err)
	}
	manifest := fmt.Sprintf(`{"service":{"name":"%s"},"cli":{"enabled":true,"command":"%s","adapter":{"kind":"shell_script","script_path":"cli/%s","install_script":"cli/install.sh"},"invoke":{"kind":"installed_command","command":"%s"},"install":[{"kind":"command","run":"bash cli/install.sh"}]},"lifecycle":{"health":{"checks":[{"name":"api"}]}}}`, name, name, name, name)
	manifestPath := filepath.Join(scenarioDir, ".vrooli", "service.json")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}
	indexPath := filepath.Join(scenarioDir, "requirements", "index.json")
	if err := os.WriteFile(indexPath, []byte(`{"imports":["01-internal-orchestrator/module.json"]}`), 0o644); err != nil {
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
	scenarioCLI := filepath.Join(scenarioDir, "cli", name)
	cliScript := fmt.Sprintf(`#!/usr/bin/env bash
# Handle no arguments - print help
if [ -z "$1" ]; then
  echo "usage: %s <cmd>"
  exit 0
fi
# Handle known commands
case "$1" in
  version|--version|-v)
    echo "%s version 1.0.0"
    exit 0
    ;;
  help|--help|-h)
    echo "usage: %s <cmd>"
    exit 0
    ;;
  *)
    # Unknown command - return error
    echo "error: unknown command '$1'" >&2
    exit 1
    ;;
esac
`, name, name, name)
	if err := os.WriteFile(scenarioCLI, []byte(cliScript), 0o755); err != nil {
		t.Fatalf("failed to seed scenario cli: %v", err)
	}
	installScript := filepath.Join(scenarioDir, "cli", "install.sh")
	if err := os.WriteFile(installScript, []byte("#!/usr/bin/env bash\necho install\n"), 0o755); err != nil {
		t.Fatalf("failed to seed cli/install.sh: %v", err)
	}
	batsFile := filepath.Join(scenarioDir, "cli", name+".bats")
	if err := os.WriteFile(batsFile, []byte("#!/usr/bin/env bats\n"), 0o644); err != nil {
		t.Fatalf("failed to seed cli bats file: %v", err)
	}
	playbookRegistry := fmt.Sprintf(`{"scenario":"%s","playbooks":[]}`, name)
	registryPath := filepath.Join(scenarioDir, "bas", "registry.json")
	if err := os.WriteFile(registryPath, []byte(playbookRegistry), 0o644); err != nil {
		t.Fatalf("failed to seed playbook registry: %v", err)
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
	if err := os.WriteFile(testingConfig, []byte(`{"structure":{"ui_smoke":{"enabled":false}}}`), 0o644); err != nil {
		t.Fatalf("failed to seed testing config: %v", err)
	}
	return scenarioDir
}

func skipPlaybooksForTests(t *testing.T) {
	t.Helper()
	t.Setenv("TEST_GENIE_SKIP_PLAYBOOKS", "1")
}

func skipStandardsForTests(t *testing.T) {
	t.Helper()
	t.Setenv("TEST_GENIE_SKIP_STANDARDS", "1")
}

func skipDocsForTests(t *testing.T) {
	t.Helper()
	t.Setenv("TEST_GENIE_SKIP_DOCS", "1")
}

func skipArchitectureForTests(t *testing.T) {
	t.Helper()
	t.Setenv("TEST_GENIE_SKIP_ARCHITECTURE", "1")
}

func stubRuntimePhaseRunners(orchestrator *SuiteOrchestrator) {
	noOp := func(ctx context.Context, env workspacepkg.Environment, logWriter io.Writer) phasespkg.RunReport {
		return phasespkg.RunReport{}
	}
	// ui-health shells out to the ui-health binary against a real scenario; stub
	// it here (preserving its weight-20 position) so orchestration tests don't
	// depend on that external tool.
	orchestrator.catalog.Register(phasespkg.Spec{Name: phasespkg.UIHealth, Runner: noOp, Optional: false, Weight: 20})
	orchestrator.catalog.Register(phasespkg.Spec{Name: phasespkg.Quality, Runner: noOp, Optional: false, Weight: 60})
	orchestrator.catalog.Register(phasespkg.Spec{Name: phasespkg.Performance, Runner: noOp, Optional: true, Weight: 60})
	orchestrator.catalog.Register(phasespkg.Spec{Name: phasespkg.Smoke, Runner: noOp, Optional: true, Weight: 70})
	// unit delegates to the unit-health binary against a real scenario after the
	// hard cutover; stub it (preserving its weight-80 position) so orchestration
	// tests don't depend on that external tool.
	orchestrator.catalog.Register(phasespkg.Spec{Name: phasespkg.Unit, Runner: noOp, Optional: false, Weight: 80})
	orchestrator.catalog.Register(phasespkg.Spec{Name: phasespkg.Integration, Runner: noOp, Weight: 90})
	orchestrator.catalog.Register(phasespkg.Spec{Name: phasespkg.Playbooks, Runner: noOp, Weight: 100})
	orchestrator.catalog.Register(phasespkg.Spec{Name: phasespkg.Security, Runner: noOp, Optional: true, Weight: 140})
	orchestrator.catalog.Register(phasespkg.Spec{Name: phasespkg.Measures, Runner: noOp, Optional: true, Weight: 150})
	orchestrator.catalog.Register(phasespkg.Spec{Name: phasespkg.Proto, Runner: noOp, Optional: true, Weight: 160})
}

func TestSuiteOrchestratorExecutesPhases(t *testing.T) {
	t.Run("[REQ:TESTGENIE-ORCH-P0] orchestrator runs go-native phases", func(t *testing.T) {
		skipPlaybooksForTests(t)
		skipStandardsForTests(t)
		skipDocsForTests(t)
		skipArchitectureForTests(t)
		root := t.TempDir()
		createScenarioLayout(t, root, "demo")
		stubCommandLookup(t, func(name string) (string, error) {
			return "/tmp/" + name, nil
		})
		stubPhaseCommandExecutor(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			// Unknown command check expects failure for unknown commands
			if len(args) > 0 && strings.HasPrefix(args[0], "__test_genie") {
				return fmt.Errorf("unknown command: %s", args[0])
			}
			return nil
		})
		stubPhaseCommandCapture(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error) {
			switch {
			case name == "scenario-auditor":
				return `{"security":null,"standards":{"summary":{"total":0,"by_severity":{},"highest_severity":"","top_violations":[]}}}`, nil
			case isDependencyHealthCommand(name, args):
				return dependencyHealthStubOutput(args[1]), nil
			case strings.Contains(name, filepath.Join("cli", "demo")) && len(args) > 0 && args[0] == "version":
				return "demo version 1.0.0", nil
			case strings.Contains(name, filepath.Join("cli", "test-genie")) && len(args) > 0 && args[0] == "version":
				return "test-genie version 1.0.0", nil
			default:
				return "", nil
			}
		})

		orchestrator, err := NewSuiteOrchestrator(root)
		if err != nil {
			t.Fatalf("failed to init orchestrator: %v", err)
		}
		stubRuntimePhaseRunners(orchestrator)

		result, err := orchestrator.Execute(context.Background(), SuiteExecutionRequest{
			ScenarioName: "demo",
			UIURL:        "http://127.0.0.1:1",
			APIURL:       "http://127.0.0.1:2",
		})
		if err != nil {
			t.Fatalf("execution failed: %v", err)
		}
		if !result.Success {
			t.Fatalf("expected success, got failure: %#v", result)
		}
		// Derive expectations from the orchestrator's own catalog (the single
		// source of truth) rather than a hard-coded count and name list, so that
		// adding/removing/reordering a phase updates this assertion automatically.
		expected := make([]string, 0)
		for _, spec := range orchestrator.catalog.All() {
			expected = append(expected, spec.Name.String())
		}
		if len(result.Phases) != len(expected) {
			t.Fatalf("expected %d phases, got %d", len(expected), len(result.Phases))
		}
		for _, phase := range result.Phases {
			if phase.Status != "passed" {
				t.Fatalf("phase %s expected passed, got %s", phase.Name, phase.Status)
			}
			if phase.LogPath == "" {
				t.Fatalf("phase %s missing log path", phase.Name)
			}
		}
		actualNames := make([]string, len(result.Phases))
		for i, p := range result.Phases {
			actualNames[i] = p.Name
		}
		for i, name := range expected {
			if result.Phases[i].Name != name {
				t.Fatalf("expected phase %d to be %s but got %s (actual=%v)", i, name, result.Phases[i].Name, actualNames)
			}
		}
	})
}

func TestBuildWarningSummaryGroupsWarningObservationsByPhase(t *testing.T) {
	results := []PhaseExecutionResult{
		{
			Name:    "structure",
			LogPath: "coverage/logs/run/structure.log",
			Observations: []phasespkg.Observation{
				phasespkg.NewSuccessObservation("structure passed"),
				phasespkg.NewWarningObservation("starter BAS registry is empty"),
			},
		},
		{
			Name:    "performance",
			LogPath: "coverage/logs/run/performance.log",
			Observations: []phasespkg.Observation{
				phasespkg.NewWarningObservation("seo: 82% below warning threshold 90%"),
				phasespkg.NewInfoObservation("lighthouse artifact written"),
			},
		},
	}

	summary := BuildWarningSummary("20251208-151044-warnsum0", results)
	if summary.Total != 2 {
		t.Fatalf("expected two warnings, got %#v", summary)
	}
	if len(summary.Phases) != 2 {
		t.Fatalf("expected two phase groups, got %#v", summary.Phases)
	}
	if summary.Phases[0].Name != "structure" || summary.Phases[0].Count != 1 {
		t.Fatalf("unexpected first phase summary: %#v", summary.Phases[0])
	}
	warning := summary.Phases[1].Warnings[0]
	if warning.Message != "seo: 82% below warning threshold 90%" {
		t.Fatalf("unexpected warning message: %#v", warning)
	}
	if warning.Source != "observation" {
		t.Fatalf("expected observation source, got %q", warning.Source)
	}
	if warning.LogPath != "coverage/logs/run/performance.log" {
		t.Fatalf("expected log path to be copied, got %q", warning.LogPath)
	}
	if warning.ArtifactPath != filepath.Join("coverage", "runs", "20251208-151044-warnsum0", "phase-results", "performance.json") {
		t.Fatalf("expected artifact path, got %q", warning.ArtifactPath)
	}
}

func TestSuiteOrchestratorPreviewExecutionUsesConfiguredTimeouts(t *testing.T) {
	t.Run("[REQ:TESTGENIE-ORCH-PREVIEW-P0] preview reflects selected phases and timeout overrides", func(t *testing.T) {
		root := t.TempDir()
		scenarioDir := createScenarioLayout(t, root, "demo")
		testingConfig := `{
  "phases": {
    "unit": {"timeout": "25s"},
    "performance": {"enabled": false}
  },
  "presets": {
    "focused": ["unit", "performance"]
  }
}`
		if err := os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "testing.json"), []byte(testingConfig), 0o644); err != nil {
			t.Fatalf("failed to write testing config: %v", err)
		}

		orchestrator, err := NewSuiteOrchestrator(root)
		if err != nil {
			t.Fatalf("failed to init orchestrator: %v", err)
		}

		preview, err := orchestrator.PreviewExecution(SuiteExecutionRequest{
			ScenarioName: "demo",
			Preset:       "focused",
		})
		if err != nil {
			t.Fatalf("preview failed: %v", err)
		}
		if preview.PresetUsed != "focused" {
			t.Fatalf("expected preset to be preserved, got %q", preview.PresetUsed)
		}
		if len(preview.Phases) != 1 {
			t.Fatalf("expected disabled phase to be omitted, got %#v", preview.Phases)
		}
		if preview.Phases[0].Name != "unit" {
			t.Fatalf("expected unit phase, got %#v", preview.Phases[0])
		}
		if preview.Phases[0].TimeoutSeconds != 25 {
			t.Fatalf("expected configured timeout 25s, got %d", preview.Phases[0].TimeoutSeconds)
		}
	})
}

func TestSuiteOrchestratorExecuteCapturesSelectionMetadata(t *testing.T) {
	t.Run("[REQ:TESTGENIE-ORCH-META-P0] execution results retain requested and planned phase metadata", func(t *testing.T) {
		root := t.TempDir()
		createScenarioLayout(t, root, "demo")
		stubCommandLookup(t, func(name string) (string, error) {
			return "/tmp/" + name, nil
		})
		stubPhaseCommandExecutor(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			if len(args) > 0 && strings.HasPrefix(args[0], "__test_genie") {
				return fmt.Errorf("unknown command: %s", args[0])
			}
			return nil
		})
		stubPhaseCommandCapture(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error) {
			if strings.Contains(name, filepath.Join("cli", "demo")) && len(args) > 0 && args[0] == "version" {
				return "demo version 1.0.0", nil
			}
			if strings.Contains(name, filepath.Join("cli", "test-genie")) && len(args) > 0 && args[0] == "version" {
				return "test-genie version 1.0.0", nil
			}
			if isDependencyHealthCommand(name, args) {
				return dependencyHealthStubOutput(args[1]), nil
			}
			return "", nil
		})

		orchestrator, err := NewSuiteOrchestrator(root)
		if err != nil {
			t.Fatalf("failed to init orchestrator: %v", err)
		}
		stubRuntimePhaseRunners(orchestrator)

		result, err := orchestrator.Execute(context.Background(), SuiteExecutionRequest{
			ScenarioName: "demo",
			Phases:       []string{"structure", "unit", "integration"},
			Skip:         []string{"integration"},
			FailFast:     true,
		})
		if err != nil {
			t.Fatalf("execution failed: %v", err)
		}
		if !result.FailFast {
			t.Fatalf("expected failFast to be preserved")
		}
		expectedRequested := []string{"structure", "unit", "integration"}
		if strings.Join(result.RequestedPhases, ",") != strings.Join(expectedRequested, ",") {
			t.Fatalf("unexpected requested phases: %#v", result.RequestedPhases)
		}
		if strings.Join(result.RequestedSkipPhases, ",") != "integration" {
			t.Fatalf("unexpected requested skip phases: %#v", result.RequestedSkipPhases)
		}
		if strings.Join(result.PlannedPhases, ",") != "structure,unit" {
			t.Fatalf("unexpected planned phases: %#v", result.PlannedPhases)
		}
		if len(result.Phases) != 2 || result.Phases[0].Name != "structure" || result.Phases[1].Name != "unit" {
			t.Fatalf("unexpected executed phases: %#v", result.Phases)
		}
	})
}

func TestSuiteOrchestratorSyncsRequirementsAfterFullRun(t *testing.T) {
	t.Run("[REQ:TESTGENIE-ORCH-P0] full suites trigger requirement sync", func(t *testing.T) {
		skipPlaybooksForTests(t)
		skipStandardsForTests(t)
		skipDocsForTests(t)
		skipArchitectureForTests(t)
		root := t.TempDir()
		createScenarioLayout(t, root, "demo")
		stubCommandLookup(t, func(name string) (string, error) {
			return "/tmp/" + name, nil
		})
		stubPhaseCommandExecutor(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			// Unknown command check expects failure for unknown commands
			if len(args) > 0 && strings.HasPrefix(args[0], "__test_genie") {
				return fmt.Errorf("unknown command: %s", args[0])
			}
			return nil
		})
		stubPhaseCommandCapture(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error) {
			switch {
			case name == "scenario-auditor":
				return `{"security":null,"standards":{"summary":{"total":0,"by_severity":{},"highest_severity":"","top_violations":[]}}}`, nil
			case isDependencyHealthCommand(name, args):
				return dependencyHealthStubOutput(args[1]), nil
			case strings.Contains(name, filepath.Join("cli", "demo")) && len(args) > 0 && args[0] == "version":
				return "demo version 1.0.0", nil
			case strings.Contains(name, filepath.Join("cli", "test-genie")) && len(args) > 0 && args[0] == "version":
				return "test-genie version 1.0.0", nil
			default:
				return "", nil
			}
		})

		orchestrator, err := NewSuiteOrchestrator(root)
		if err != nil {
			t.Fatalf("failed to init orchestrator: %v", err)
		}
		stubRuntimePhaseRunners(orchestrator)
		stubSyncer := &stubRequirementsSyncer{
			outcome: &reqsync.SyncOutcome{Synced: true, OTComplete: 1, OTTotal: 3},
		}
		orchestrator.requirements = stubSyncer

		result, err := orchestrator.Execute(context.Background(), SuiteExecutionRequest{
			ScenarioName: "demo",
			UIURL:        "http://127.0.0.1:1",
			APIURL:       "http://127.0.0.1:2",
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
		// A full suite run syncs (not snapshots) and the outcome is attached to
		// the execution result for the report to render.
		if stubSyncer.snapshotCalls != 0 {
			t.Fatalf("expected no snapshot calls on a full suite, got %d", stubSyncer.snapshotCalls)
		}
		if result.Requirements == nil {
			t.Fatalf("expected requirements outcome attached to result")
		}
		if result.Requirements.OTTotal != 3 || result.Requirements.OTComplete != 1 {
			t.Fatalf("unexpected requirements counts on result: %#v", result.Requirements)
		}
		if stubSyncer.last.ScenarioName != "demo" {
			t.Fatalf("unexpected scenario name in sync payload: %#v", stubSyncer.last)
		}
		if len(stubSyncer.last.PhaseResults) != len(stubSyncer.last.PhaseDefinitions) {
			t.Fatalf("expected phase metadata to be recorded for sync")
		}
	})
}

func TestPrepareTargetRuntimeIgnoresGenericPortEnvironment(t *testing.T) {
	t.Setenv("UI_PORT", "21223")
	t.Setenv("API_PORT", "15421")

	var started bool
	orch := &SuiteOrchestrator{
		newRuntime: func(name, scenarioDir string) *targetruntime.Manager {
			return targetruntime.New(name, scenarioDir).
				WithHome(t.TempDir()).
				WithProbes(func(context.Context, int) bool { return true }, func(int) bool { return true }).
				WithCommandRunner(func(ctx context.Context, dir string, env map[string]string, logWriter io.Writer, command string, args ...string) error {
					started = true
					return fmt.Errorf("expected start because generic env vars are not runtime discovery")
				})
		},
	}

	env := workspacepkg.Environment{ScenarioName: "demo", ScenarioDir: filepath.Join(t.TempDir(), "demo")}
	smokeDef := phasespkg.Definition{Name: phasespkg.Smoke, Capabilities: runnability.PhaseCapabilities{Phase: phasespkg.Smoke.String(), NeedsUI: true}}
	_, _, _, _, err := orch.prepareTargetRuntime(context.Background(), env, []phasespkg.Definition{smokeDef}, SuiteExecutionRequest{}, io.Discard)
	if err == nil {
		t.Fatal("expected runtime start failure")
	}
	if !started {
		t.Fatal("expected target runtime start attempt")
	}
	if strings.Contains(err.Error(), "21223") || strings.Contains(err.Error(), "15421") {
		t.Fatalf("generic runtime env leaked into error: %v", err)
	}
}

func TestNewRunIDIsUniqueAndSortable(t *testing.T) {
	first := newRunID()
	time.Sleep(10 * time.Millisecond)
	second := newRunID()

	if first == second {
		t.Fatal("expected unique run IDs")
	}

	pattern := regexp.MustCompile(`^\d{8}-\d{6}-[0-9a-f]{8}$`)
	if !pattern.MatchString(first) {
		t.Fatalf("expected first run ID to match timestamp-uuid suffix, got %q", first)
	}
	if !pattern.MatchString(second) {
		t.Fatalf("expected second run ID to match timestamp-uuid suffix, got %q", second)
	}
}

func TestSuiteOrchestratorFailFastStopsExecution(t *testing.T) {
	t.Run("[REQ:TESTGENIE-ORCH-P0] fail-fast halts remaining phases", func(t *testing.T) {
		skipPlaybooksForTests(t)
		root := t.TempDir()
		scenarioDir := createScenarioLayout(t, root, "demo")
		stubCommandLookup(t, func(name string) (string, error) {
			return "/tmp/" + name, nil
		})
		// Force the structure phase to fail by removing a required file.
		requiredFile := filepath.Join(scenarioDir, "README.md")
		if err := os.Remove(requiredFile); err != nil {
			t.Fatalf("failed to remove %s: %v", requiredFile, err)
		}

		orchestrator, err := NewSuiteOrchestrator(root)
		if err != nil {
			t.Fatalf("failed to init orchestrator: %v", err)
		}
		stubRuntimePhaseRunners(orchestrator)

		result, err := orchestrator.Execute(context.Background(), SuiteExecutionRequest{
			ScenarioName: "demo",
			FailFast:     true,
			UIURL:        "http://127.0.0.1:1",
			APIURL:       "http://127.0.0.1:2",
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
		testDir := filepath.Join(scenarioDir, "coverage")
		if err := os.MkdirAll(testDir, 0o755); err != nil {
			t.Fatalf("failed to create coverage dir: %v", err)
		}

		if err := os.WriteFile(filepath.Join(testDir, "presets.json"), []byte(`{"focused":["unit"]}`), 0o644); err != nil {
			t.Fatalf("failed to write preset: %v", err)
		}
		stubCommandLookup(t, func(name string) (string, error) {
			return "/tmp/" + name, nil
		})
		stubPhaseCommandExecutor(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			// Unknown command check expects failure for unknown commands
			if len(args) > 0 && strings.HasPrefix(args[0], "__test_genie") {
				return fmt.Errorf("unknown command: %s", args[0])
			}
			return nil
		})
		stubPhaseCommandCapture(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error) {
			if name == "scenario-auditor" {
				return `{"security":null,"standards":{"summary":{"total":0,"by_severity":{},"highest_severity":"","top_violations":[]}}}`, nil
			}
			if isDependencyHealthCommand(name, args) {
				return dependencyHealthStubOutput(args[1]), nil
			}
			return "", nil
		})

		orchestrator, err := NewSuiteOrchestrator(root)
		if err != nil {
			t.Fatalf("failed to init orchestrator: %v", err)
		}
		stubRuntimePhaseRunners(orchestrator)

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
			// Unknown command check expects failure for unknown commands
			if len(args) > 0 && strings.HasPrefix(args[0], "__test_genie") {
				return fmt.Errorf("unknown command: %s", args[0])
			}
			return nil
		})

		orchestrator, err := NewSuiteOrchestrator(root)
		if err != nil {
			t.Fatalf("failed to init orchestrator: %v", err)
		}
		stubRuntimePhaseRunners(orchestrator)

		result, err := orchestrator.Execute(context.Background(), SuiteExecutionRequest{
			ScenarioName: "demo",
			UIURL:        "http://127.0.0.1:1",
			APIURL:       "http://127.0.0.1:2",
		})
		if err != nil {
			t.Fatalf("execution failed: %v", err)
		}
		for _, phase := range result.Phases {
			if phase.Name == "integration" {
				t.Fatalf("expected integration phase to be disabled via testing config")
			}
		}
		// All catalog phases run except the one disabled via testing config
		// (integration). Derive the expected count from the catalog so adding
		// a phase doesn't require a hand-edit here.
		expectedPhases := len(phasespkg.DefaultCatalog().All()) - 1
		if len(result.Phases) != expectedPhases {
			t.Fatalf("expected %d phases after disabling integration, got %d", expectedPhases, len(result.Phases))
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
			// Unknown command check expects failure for unknown commands
			if len(args) > 0 && strings.HasPrefix(args[0], "__test_genie") {
				return fmt.Errorf("unknown command: %s", args[0])
			}
			return nil
		})

		orchestrator, err := NewSuiteOrchestrator(root)
		if err != nil {
			t.Fatalf("failed to init orchestrator: %v", err)
		}

		result, err := orchestrator.Execute(context.Background(), SuiteExecutionRequest{
			ScenarioName: "demo",
			Preset:       "focused",
			UIURL:        "http://127.0.0.1:1",
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
		selected, preset, notices, err := selectPhases(defs, presets, SuiteExecutionRequest{}, PhaseToggleConfig{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if preset != "" {
			t.Fatalf("expected empty preset usage, got %s", preset)
		}
		if len(selected) != len(defs) {
			t.Fatalf("expected %d phases, got %d", len(defs), len(selected))
		}
		if len(notices.Skipped) != 0 || len(notices.Explicit) != 0 {
			t.Fatalf("expected no disabled notices, got %#v", notices)
		}
	})

	t.Run("resolves presets case-insensitively", func(t *testing.T) {
		selected, preset, notices, err := selectPhases(defs, presets, SuiteExecutionRequest{Preset: "Quick"}, PhaseToggleConfig{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if preset != "quick" {
			t.Fatalf("expected preset quick, got %s", preset)
		}
		if len(selected) != 2 || selected[0].Name != PhaseStructure || selected[1].Name != PhaseUnit {
			t.Fatalf("unexpected preset selection: %#v", selected)
		}
		if len(notices.Skipped) != 0 {
			t.Fatalf("expected no skipped phases, got %#v", notices.Skipped)
		}
	})

	t.Run("errors when requested phase missing", func(t *testing.T) {
		_, _, _, err := selectPhases(defs, presets, SuiteExecutionRequest{Phases: []string{"structure", "invalid"}}, PhaseToggleConfig{})
		if err == nil {
			t.Fatalf("expected error for invalid phase selection")
		}
	})

	t.Run("applies skip list", func(t *testing.T) {
		selected, _, _, err := selectPhases(defs, presets, SuiteExecutionRequest{Skip: []string{"dependencies"}}, PhaseToggleConfig{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, def := range selected {
			if def.Name == PhaseDependencies {
				t.Fatalf("dependency phase should have been skipped")
			}
		}
	})

	t.Run("honors global disabled phases", func(t *testing.T) {
		toggles := PhaseToggleConfig{
			Phases: map[string]PhaseToggle{
				"unit": {Disabled: true, Reason: "flaky"},
			},
		}
		selected, preset, notices, err := selectPhases(defs, presets, SuiteExecutionRequest{}, toggles)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if preset != "" {
			t.Fatalf("expected empty preset usage, got %s", preset)
		}
		if len(selected) != 2 {
			t.Fatalf("expected 2 phases after disabling unit, got %d", len(selected))
		}
		if len(notices.Skipped) != 1 || notices.Skipped[0].Name != "unit" {
			t.Fatalf("expected unit to be skipped with notice, got %#v", notices)
		}
	})

	t.Run("allows explicitly requesting disabled phases with notice", func(t *testing.T) {
		toggles := PhaseToggleConfig{
			Phases: map[string]PhaseToggle{
				"unit": {Disabled: true, Reason: "investigating"},
			},
		}
		selected, preset, notices, err := selectPhases(defs, presets, SuiteExecutionRequest{Phases: []string{"unit"}}, toggles)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if preset != "" {
			t.Fatalf("expected empty preset usage, got %s", preset)
		}
		if len(selected) != 1 || selected[0].Name != PhaseUnit {
			t.Fatalf("expected only unit to be selected, got %#v", selected)
		}
		if len(notices.Explicit) != 1 || notices.Explicit[0].Name != "unit" {
			t.Fatalf("expected explicit notice for unit, got %#v", notices)
		}
	})
}

func boolPtr(value bool) *bool {
	v := value
	return &v
}

func isDependencyHealthCommand(name string, args []string) bool {
	return name == "scenario-dependency-analyzer" &&
		len(args) >= 3 &&
		args[0] == "health" &&
		args[1] != "" &&
		args[2] == "--json"
}

func dependencyHealthStubOutput(scenario string) string {
	return fmt.Sprintf(`{
		"scenario": %q,
		"passed": true,
		"summary": {"sections": 6, "findings": 0, "errors": 0, "warnings": 0, "infos": 0},
		"sections": [
			{"id": "surfaces", "title": "Code surfaces", "status": "pass"},
			{"id": "readiness", "title": "Dependency readiness", "status": "pass"},
			{"id": "runtime", "title": "Runtime dependencies", "status": "pass"},
			{"id": "governance", "title": "Approved dependency governance", "status": "pass"},
			{"id": "release-age", "title": "Package release-age policy", "status": "pass"},
			{"id": "graph", "title": "Dependency graph drift", "status": "pass"}
		],
		"findings": [],
		"assessment": {
			"scenario": %q,
			"provider": "scenario-dependency-analyzer",
			"phase": "dependencies",
			"version": "1.0.0",
			"local": {"currentLevel": "L5", "nextLevel": ""}
		}
	}`, scenario, scenario)
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
