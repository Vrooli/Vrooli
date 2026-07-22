package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
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
	sharedartifacts "test-genie/internal/shared/artifacts"
	sharedruns "test-genie/internal/shared/runs"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

func TestIsLinkedWorktreeDistinguishesPrimaryAndStrictWorktree(t *testing.T) {
	primary := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", primary}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	runGit("init")
	runGit("config", "user.email", "tests@example.invalid")
	runGit("config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(primary, "tracked.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit("add", "tracked.txt")
	runGit("commit", "-m", "fixture")
	linked := filepath.Join(t.TempDir(), "linked")
	runGit("worktree", "add", linked)
	if isLinkedWorktree(primary) {
		t.Fatal("primary checkout must not satisfy strict linked-worktree provenance")
	}
	if !isLinkedWorktree(linked) {
		t.Fatal("linked worktree must satisfy strict provenance precondition")
	}
}

type stubRequirementsSyncer struct {
	calls         int
	snapshotCalls int
	last          reqsync.SyncInput
	err           error
	outcome       *reqsync.SyncOutcome
}

func TestCompactTerminalSnapshotDropsDetailedPhasePayloads(t *testing.T) {
	result := &SuiteExecutionResult{Phases: []PhaseExecutionResult{{
		Name: "security", Status: "failed", DurationSeconds: 3,
		Observations: []phasespkg.Observation{phasespkg.NewErrorObservation("large observation")},
		Findings:     []*architecturev1.ArchitectureFinding{{Code: "detailed.finding"}},
		LogPath:      "coverage/logs/security.log",
	}}}
	compact := CompactTerminalSnapshot(result)
	if compact == result || len(compact.Phases) != 1 {
		t.Fatalf("compact snapshot = %+v", compact)
	}
	phase := compact.Phases[0]
	if phase.LogPath != "" || len(phase.Observations) != 0 || len(phase.Findings) != 0 || phase.Metrics != nil {
		t.Fatalf("snapshot retained detailed phase payload: %+v", phase)
	}
	if phase.Name != "security" || phase.Status != "failed" || phase.DurationSeconds != 3 {
		t.Fatalf("snapshot lost compact phase summary: %+v", phase)
	}
}

func TestFinalizeRunRecordPersistsCanonicalTerminalSnapshot(t *testing.T) {
	dir := t.TempDir()
	runID := "20260710-142937-ae6a753e"
	started := time.Now().UTC().Add(-12 * time.Second).Truncate(time.Second)
	completed := started.Add(12 * time.Second)
	idx := sharedruns.NewIndex(dir)
	if err := idx.Append(sharedruns.RunRecord{
		RunID: runID, Scenario: "demo", StartedAt: started, Status: sharedruns.StatusInProgress,
	}); err != nil {
		t.Fatalf("append: %v", err)
	}
	result := &SuiteExecutionResult{
		RunID: runID, ScenarioName: "demo", StartedAt: started, CompletedAt: completed,
		Success: false, Verdict: SuiteVerdictFail, PlannedPhases: []string{"unit"},
		Phases: []PhaseExecutionResult{{Name: "unit", Status: "failed", DurationSeconds: 12}},
	}
	(&SuiteOrchestrator{}).finalizeRunRecord(dir, runID, result, []phaseResultView{{
		Name: "unit", Status: "failed", DurationSeconds: 12,
	}})

	rec, err := idx.Find(runID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if rec.Status != sharedruns.StatusFailed || len(rec.Phases) != 1 {
		t.Fatalf("terminal index = %+v", rec)
	}
	snapshot, err := idx.ReadTerminalSnapshot(runID)
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	var persisted SuiteExecutionResult
	if err := json.Unmarshal(snapshot.Result, &persisted); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if persisted.RunID != runID || len(persisted.Phases) != 1 || persisted.Phases[0].DurationSeconds != 12 {
		t.Fatalf("persisted terminal result = %+v", persisted)
	}
}

func TestWriteArtifactCatalogUsesCapturedDescriptorEvidenceKinds(t *testing.T) { // [REQ:TESTGENIE-TYPED-EVIDENCE-P0]
	dir := t.TempDir()
	runID := "run-artifacts"
	snapshot, err := sharedruns.NewDescriptorSnapshot([]sharedruns.PhaseDescriptorSnapshot{{
		Phase: "future-visual", EvidenceKinds: []string{sharedartifacts.ArtifactKindScreenshot},
		Applicability: sharedruns.ApplicabilityDecisionSnapshot{Status: "applies", Planned: true},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := sharedruns.WriteDescriptorSnapshot(dir, runID, snapshot); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(sharedartifacts.RunUISmokePagesDir(dir, runID), "home", "screenshot.png")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeArtifactCatalog(dir, runID, time.Unix(100, 0)); err != nil {
		t.Fatalf("writeArtifactCatalog: %v", err)
	}
	catalog, err := sharedartifacts.ReadArtifactCatalog(dir, runID)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Artifacts) != 1 || catalog.Artifacts[0].ProducingPhase != "future-visual" {
		t.Fatalf("catalog = %+v", catalog)
	}
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
	manifest := fmt.Sprintf(`{"service":{"name":"%s"},"cli":{"enabled":true,"command":"%s","adapter":{"kind":"go_module","module_dir":"cli"},"source_build":{"kind":"go_module"},"invoke":{"kind":"installed_command","command":"%s"},"freshness":{"inputs":["cli/**",".vrooli/service.json"]}},"lifecycle":{"health":{"checks":[{"name":"api"}]}}}`, name, name, name)
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
	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "go.mod"), []byte("module "+name+"/cli\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("failed to seed scenario CLI module: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("failed to seed scenario CLI source: %v", err)
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

func stubRuntimePhaseRunners(orchestrator *SuiteOrchestrator) {
	noOp := func(ctx context.Context, env workspacepkg.Environment, logWriter io.Writer) phasespkg.RunReport {
		return phasespkg.RunReport{}
	}
	// Provider-backed phases are covered in the phase package. Orchestration
	// tests replace every discovered runner and detach its provider transport,
	// so a future provider phase needs no fixture registration and a minimal
	// fake scenario never depends on live provider APIs.
	for _, spec := range orchestrator.catalog.All() {
		spec.Runner = noOp
		spec.Delegated = nil
		orchestrator.catalog.Register(spec)
	}
}

func TestDiscoverPhaseDefinitionsIgnoresScenarioLocalScripts(t *testing.T) {
	testDir := t.TempDir()
	phaseDir := filepath.Join(testDir, "phases")
	if err := os.MkdirAll(phaseDir, 0o755); err != nil {
		t.Fatalf("create phase dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(phaseDir, "test-unit.sh"), []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write script phase: %v", err)
	}

	orchestrator := &SuiteOrchestrator{phaseTimeout: time.Minute}
	defs, err := orchestrator.discoverPhaseDefinitions(workspacepkg.Environment{TestDir: testDir})
	if err != nil {
		t.Fatalf("discover phases: %v", err)
	}
	if len(defs) != 0 {
		t.Fatalf("script phases must be ignored, got %d definitions: %#v", len(defs), defs)
	}
}

func TestSuiteOrchestratorExecutesPhases(t *testing.T) {
	t.Run("[REQ:TESTGENIE-ORCH-P0] orchestrator runs catalog phases", func(t *testing.T) {
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
			case isAPIHealthCommand(name, args):
				return apiHealthStubOutput(args[2]), nil
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
		futurePhase := phasespkg.Name("future-provider-fixture")
		orchestrator.catalog.Register(phasespkg.Spec{
			Name: futurePhase,
			Runner: func(context.Context, workspacepkg.Environment, io.Writer) phasespkg.RunReport {
				return phasespkg.RunReport{}
			},
		})
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
		// Derive expectations from the execution plan so descriptor applicability
		// changes, such as search applying only to search-enabled scenarios, do
		// not require a hand-maintained fixture list.
		expected := append([]string(nil), result.PlannedPhases...)
		futurePlanned := false
		for _, name := range expected {
			if name == futurePhase.String() {
				futurePlanned = true
				break
			}
		}
		if !futurePlanned {
			t.Fatalf("synthetic phase %q was not planned; orchestration fixtures must accept catalog extensions without registration", futurePhase)
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

func TestSuiteOrchestratorPreviewExecutionAppliesDescriptorApplicability(t *testing.T) {
	t.Run("[REQ:TESTGENIE-ORCH-PREVIEW-P0] descriptor phase is omitted when target is not applicable", func(t *testing.T) {
		repoRoot := t.TempDir()
		root := filepath.Join(repoRoot, "scenarios")
		createScenarioLayout(t, root, "demo")
		writeTestGenieDescriptor(t, root, "unit-health", testPhaseDescriptor("unit-health", "unit", `"applicability":{"default":"applies"}`))
		writeTestGenieDescriptor(t, root, "search-hub", searchDescriptor("search-hub"))

		orchestrator, err := NewSuiteOrchestrator(root)
		if err != nil {
			t.Fatalf("failed to init orchestrator: %v", err)
		}

		preview, err := orchestrator.PreviewExecution(SuiteExecutionRequest{ScenarioName: "demo"})
		if err != nil {
			t.Fatalf("preview failed: %v", err)
		}
		if hasPlannedPhase(preview.Phases, "search") {
			t.Fatalf("search phase should not be selected for non-search target: %#v", preview.Phases)
		}
		search, ok := plannedPhase(preview.NotApplicablePhases, "search")
		if !ok {
			t.Fatalf("search phase should be reported as not applicable: %#v", preview.NotApplicablePhases)
		}
		if search.ApplicabilityStatus != "not_applicable" {
			t.Fatalf("search applicability = %q, want not_applicable", search.ApplicabilityStatus)
		}
	})

	t.Run("[REQ:TESTGENIE-ORCH-PREVIEW-P0] descriptor phase is selected when target applicability matches", func(t *testing.T) {
		repoRoot := t.TempDir()
		root := filepath.Join(repoRoot, "scenarios")
		scenarioDir := createScenarioLayout(t, root, "demo")
		if err := os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "search.json"), []byte(`{"enabled":true}`), 0o644); err != nil {
			t.Fatalf("failed to seed search config: %v", err)
		}
		writeTestGenieDescriptor(t, root, "unit-health", testPhaseDescriptor("unit-health", "unit", `"applicability":{"default":"applies"}`))
		writeTestGenieDescriptor(t, root, "search-hub", searchDescriptor("search-hub"))

		orchestrator, err := NewSuiteOrchestrator(root)
		if err != nil {
			t.Fatalf("failed to init orchestrator: %v", err)
		}

		preview, err := orchestrator.PreviewExecution(SuiteExecutionRequest{ScenarioName: "demo"})
		if err != nil {
			t.Fatalf("preview failed: %v", err)
		}
		search, ok := plannedPhase(preview.Phases, "search")
		if !ok {
			t.Fatalf("search phase should be selected for search target: %#v", preview.Phases)
		}
		if search.ApplicabilityStatus != "applies" {
			t.Fatalf("search applicability = %q, want applies", search.ApplicabilityStatus)
		}
		if search.ProviderReadiness != "required_when_applicable" {
			t.Fatalf("provider readiness = %q, want required_when_applicable", search.ProviderReadiness)
		}
	})
}

func TestSuiteOrchestratorPhasePlanRequiresDescriptorMetadata(t *testing.T) {
	catalog := phasespkg.NewCatalogFromSpecs(time.Minute, phasespkg.Spec{
		Name:        phasespkg.Search,
		Description: "catalog-only fixture",
		Source:      "validation-provider",
		Delegated:   &phasespkg.Delegated{ProviderScenario: "search-hub"},
	})
	orchestrator := &SuiteOrchestrator{catalog: catalog}

	_, err := orchestrator.buildPhasePlan(workspacepkg.Environment{
		ScenarioName: "demo",
		ScenarioDir:  t.TempDir(),
	}, &workspacepkg.Config{}, SuiteExecutionRequest{})
	if err == nil {
		t.Fatal("buildPhasePlan succeeded without descriptor metadata")
	}
	if !strings.Contains(err.Error(), "phase_applicability_descriptor_missing") {
		t.Fatalf("error = %q, want phase_applicability_descriptor_missing", err.Error())
	}
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
			Phases:       []string{"structure", "unit", "docs"},
			Skip:         []string{"docs"},
			FailFast:     true,
		})
		if err != nil {
			t.Fatalf("execution failed: %v", err)
		}
		if !result.FailFast {
			t.Fatalf("expected failFast to be preserved")
		}
		expectedRequested := []string{"structure", "unit", "docs"}
		if strings.Join(result.RequestedPhases, ",") != strings.Join(expectedRequested, ",") {
			t.Fatalf("unexpected requested phases: %#v", result.RequestedPhases)
		}
		if strings.Join(result.RequestedSkipPhases, ",") != "docs" {
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

func writeTestGenieDescriptor(t *testing.T, root, scenario, body string) {
	t.Helper()
	dir := filepath.Join(root, scenario, ".vrooli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "test-genie.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func searchDescriptor(scenario string) string {
	return `{
  "schemaVersion":"1.0.0",
  "scenario":"` + scenario + `",
  "phase":"search",
  "description":"Validates search registration.",
  "source":"validation-provider",
  "orderHint":2000,
  "timeout":"120s",
  "validation":{"contract":"scenario-validation/v1","includeExecution":true},
  "applicability":{"default":"not_applicable","any":[{"fileExists":".vrooli/search.json"},{"serviceCapability":"search"}]},
  "policy":{
    "selection":"default_when_applicable",
    "providerReadiness":"required_when_applicable",
    "providerLifecycle":"start_if_needed",
    "freshness":"require_live_contract",
    "resultGating":"gating",
    "unavailable":"fail"
  },
  "runnability":{"needsUI":false,"needsAPI":false,"requiredResources":[]},
  "docs":{"path":"scenarios/test-genie/docs/phases/search/README.md"},
  "maturity":{
    "version":"2.0.0",
    "capabilities":[{"id":"registration","label":"Search registration","levels":[{"id":"L0","name":"Missing","description":"Missing search registration.","entry_criteria":[],"exit_criteria":[]}]}],
    "findings":{
      "SEARCH_REGISTRATION_MISSING":{
        "capability_id":"registration",
        "local_level_impact":"L0",
        "global_impact":"capability_gap",
        "dimension":"operational-targets",
        "severity_default":"SEVERITY_ERROR",
        "recommended_skill_ids":["search"],
        "clean_requirement":"required",
        "fix_class":"manual",
        "reason":"Requires provider-specific judgment."
      }
    },
    "fallback":{"capability_id":"registration","local_level_impact":"L0","global_impact":"unknown","dimension":"operational-targets","severity_default":"SEVERITY_WARNING","clean_requirement":"advisory"}
  }
}`
}

func testPhaseDescriptor(scenario, phase, applicability string) string {
	return `{
  "schemaVersion":"1.0.0",
  "scenario":"` + scenario + `",
  "phase":"` + phase + `",
  "description":"Validates ` + phase + ` health.",
  "source":"validation-provider",
  "orderHint":100,
  "timeout":"120s",
  "validation":{"contract":"scenario-validation/v1","includeExecution":true},
  ` + applicability + `,
  "policy":{
    "selection":"default_when_applicable",
    "providerReadiness":"required_when_applicable",
    "providerLifecycle":"start_if_needed",
    "freshness":"require_live_contract",
    "resultGating":"gating",
    "unavailable":"fail"
  },
  "runnability":{"needsUI":false,"needsAPI":false,"requiredResources":[]},
  "docs":{"path":"scenarios/test-genie/docs/phases/` + phase + `/README.md"},
  "maturity":{
    "version":"2.0.0",
    "capabilities":[{"id":"contract","label":"Contract","levels":[{"id":"L0","name":"Missing","description":"Missing contract.","entry_criteria":[],"exit_criteria":[]}]}],
    "findings":{
      "TEST_FINDING":{
        "capability_id":"contract",
        "local_level_impact":"L0",
        "global_impact":"capability_gap",
        "dimension":"operational-targets",
        "severity_default":"SEVERITY_ERROR",
        "recommended_skill_ids":["test"],
        "clean_requirement":"required",
        "fix_class":"manual",
        "reason":"Requires provider-specific judgment."
      }
    },
    "fallback":{"capability_id":"contract","local_level_impact":"L0","global_impact":"unknown","dimension":"operational-targets","severity_default":"SEVERITY_WARNING","clean_requirement":"advisory"}
  }
}`
}

func hasPlannedPhase(phases []PlannedPhase, name string) bool {
	_, ok := plannedPhase(phases, name)
	return ok
}

func plannedPhase(phases []PlannedPhase, name string) (PlannedPhase, bool) {
	for _, phase := range phases {
		if phase.Name == name {
			return phase, true
		}
	}
	return PlannedPhase{}, false
}

func TestSuiteOrchestratorSyncsRequirementsAfterFullRun(t *testing.T) {
	t.Run("[REQ:TESTGENIE-ORCH-P0] full suites trigger requirement sync", func(t *testing.T) {
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
			case isAPIHealthCommand(name, args):
				return apiHealthStubOutput(args[2]), nil
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
	uiPhaseDef := phasespkg.Definition{Name: phasespkg.Performance, Capabilities: runnability.PhaseCapabilities{Phase: phasespkg.Performance.String(), NeedsUI: true}}
	_, _, _, _, err := orch.prepareTargetRuntime(context.Background(), env, []phasespkg.Definition{uiPhaseDef}, SuiteExecutionRequest{}, io.Discard)
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
		root := t.TempDir()
		createScenarioLayout(t, root, "demo")
		stubCommandLookup(t, func(name string) (string, error) {
			return "/tmp/" + name, nil
		})

		orchestrator, err := NewSuiteOrchestrator(root)
		if err != nil {
			t.Fatalf("failed to init orchestrator: %v", err)
		}
		stubRuntimePhaseRunners(orchestrator)
		// Force the first phase (structure) to fail so fail-fast halts the rest.
		// Structure is now delegated to structure-health, so the failure is
		// injected via the runner rather than by corrupting the fake layout.
		orchestrator.catalog.Register(phasespkg.Spec{Name: phasespkg.Structure, Runner: func(ctx context.Context, env workspacepkg.Environment, logWriter io.Writer) phasespkg.RunReport {
			return phasespkg.RunReport{Err: fmt.Errorf("forced structure failure")}
		}})

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
			if isAPIHealthCommand(name, args) {
				return apiHealthStubOutput(args[2]), nil
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
		root := t.TempDir()
		scenarioDir := createScenarioLayout(t, root, "demo")
		configPath := filepath.Join(scenarioDir, ".vrooli", "testing.json")
		if err := os.WriteFile(configPath, []byte(`{"phases":{"docs":{"enabled":false}}}`), 0o644); err != nil {
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
			if phase.Name == "docs" {
				t.Fatalf("expected docs phase to be disabled via testing config")
			}
		}
		// All applicable planned phases run except the one disabled via testing
		// config (docs). Descriptor-only non-applicable phases, such as search on
		// a non-search target, are omitted before execution.
		expectedPhases := len(result.PlannedPhases)
		if len(result.Phases) != expectedPhases {
			t.Fatalf("expected %d phases after disabling docs, got %d", expectedPhases, len(result.Phases))
		}
	})
}

func TestSuiteOrchestratorHonorsTestingConfigPresets(t *testing.T) {
	t.Run("[REQ:TESTGENIE-ORCH-P0] config presets constrain execution order", func(t *testing.T) {
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

	t.Run("records skip list notices", func(t *testing.T) {
		selected, _, notices, err := selectPhases(defs, presets, SuiteExecutionRequest{Skip: []string{"dependencies"}}, PhaseToggleConfig{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(selected) != 2 {
			t.Fatalf("expected 2 selected phases, got %d", len(selected))
		}
		if len(notices.Skipped) != 1 || notices.Skipped[0].Name != "dependencies" || !notices.Skipped[0].Requested {
			t.Fatalf("expected requested skip notice for dependencies, got %#v", notices.Skipped)
		}
		if warnings := buildPlanWarnings(&phasePlan{DisabledByDefault: notices.Skipped}); len(warnings) != 1 || !strings.Contains(warnings[0], "skipped by request") {
			t.Fatalf("expected requested skip warning, got %#v", warnings)
		}
	})

	t.Run("honors env disabled phases before execution", func(t *testing.T) {
		t.Setenv("TEST_GENIE_SKIP_UNIT", "1")
		defsWithEnv := append([]phaseDefinition(nil), defs...)
		for i := range defsWithEnv {
			if defsWithEnv[i].Name == PhaseUnit {
				defsWithEnv[i].SkipEnvVar = "TEST_GENIE_SKIP_UNIT"
			}
		}
		selected, _, notices, err := selectPhases(defsWithEnv, presets, SuiteExecutionRequest{}, PhaseToggleConfig{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, def := range selected {
			if def.Name == PhaseUnit {
				t.Fatalf("unit phase should have been skipped via env before execution")
			}
		}
		if len(notices.Skipped) != 1 || notices.Skipped[0].Name != "unit" || notices.Skipped[0].EnvVar != "TEST_GENIE_SKIP_UNIT" {
			t.Fatalf("expected env skip notice for unit, got %#v", notices.Skipped)
		}
		if warnings := buildPlanWarnings(&phasePlan{DisabledByDefault: notices.Skipped}); len(warnings) != 1 || !strings.Contains(warnings[0], "TEST_GENIE_SKIP_UNIT=1") {
			t.Fatalf("expected env skip warning, got %#v", warnings)
		}
	})

	t.Run("env disabled phase is skipped even when explicit", func(t *testing.T) {
		t.Setenv("TEST_GENIE_SKIP_UNIT", "1")
		defsWithEnv := append([]phaseDefinition(nil), defs...)
		for i := range defsWithEnv {
			if defsWithEnv[i].Name == PhaseUnit {
				defsWithEnv[i].SkipEnvVar = "TEST_GENIE_SKIP_UNIT"
			}
		}
		selected, _, notices, err := selectPhases(defsWithEnv, presets, SuiteExecutionRequest{Phases: []string{"unit"}}, PhaseToggleConfig{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(selected) != 0 {
			t.Fatalf("expected explicit env-disabled unit to stay unselected, got %#v", selected)
		}
		if len(notices.Skipped) != 1 || notices.Skipped[0].EnvVar != "TEST_GENIE_SKIP_UNIT" {
			t.Fatalf("expected env skip notice, got %#v", notices.Skipped)
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

func TestValidateTestingConfigPhasesReportsUnknownKeys(t *testing.T) {
	defs := []phaseDefinition{
		{Name: PhaseStructure},
		{Name: PhaseUnit},
	}
	cfg := &workspacepkg.Config{Phases: map[string]workspacepkg.PhaseSettings{
		"strcuture": {},
	}}

	err := validateTestingConfigPhases(defs, cfg)
	if err == nil {
		t.Fatal("expected unknown phase error")
	}
	msg := err.Error()
	for _, needle := range []string{`unknown phase "strcuture"`, "available phases: structure, unit"} {
		if !strings.Contains(msg, needle) {
			t.Fatalf("error %q missing %q", msg, needle)
		}
	}
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

func isAPIHealthCommand(name string, args []string) bool {
	return name == "api-health" &&
		len(args) >= 4 &&
		args[0] == "validate" &&
		args[1] == "scenario" &&
		args[2] != "" &&
		args[3] == "--json"
}

func apiHealthStubOutput(scenario string) string {
	return fmt.Sprintf(`{
		"scenario": %q,
		"status": "VALIDATION_STATUS_PASSED",
		"assessment": {
			"scenario": %q,
			"provider": "api-health",
			"phase": "api",
			"version": "1.0.0",
			"local": {"currentLevel": "L5", "nextLevel": ""}
		},
		"metrics": {"wallClockMs": "1"}
	}`, scenario, scenario)
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

func TestSafePathGlobRejectsEscapesAndOnlyReturnsContainedFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".vrooli", "agent-profiles"), 0o755); err != nil {
		t.Fatal(err)
	}
	profile := filepath.Join(root, ".vrooli", "agent-profiles", "default.json")
	if err := os.WriteFile(profile, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	matches, err := safePathGlob(root, ".vrooli/agent-profiles/*.json")
	if err != nil || len(matches) != 1 || matches[0] != ".vrooli/agent-profiles/default.json" {
		t.Fatalf("safePathGlob = %v, %v", matches, err)
	}
	for _, pattern := range []string{"/etc/*", "../*.json", "["} {
		if _, err := safePathGlob(root, pattern); err == nil {
			t.Fatalf("safePathGlob(%q) succeeded, want rejection", pattern)
		}
	}
}
