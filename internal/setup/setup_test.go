package setup

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/lifecycle"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
	"github.com/vrooli/vrooli/internal/scenario"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=2 | LAST: 2026-04-11

func TestParseOptionsAcceptsSetupFlags(t *testing.T) {
	opts, err := parseOptions("setup", []string{"--environment", "minimal", "--resources", "none", "--yes", "yes", "--sudo-mode", "skip", "--dry-run"})
	if err != nil {
		t.Fatalf("parseOptions: %v", err)
	}
	if opts.Environment != "minimal" || opts.Resources != "none" || opts.Yes != "yes" || opts.SudoMode != "skip" || !opts.DryRun {
		t.Fatalf("opts = %+v", opts)
	}
}

func TestApplyEnvironmentSetsDefaultsAndRestoresState(t *testing.T) {
	t.Setenv("TARGET", "")
	t.Setenv("LOCATION", "")
	root := t.TempDir()
	restore, err := applyEnvironment(root, filepath.Join(root, ".vrooli", "service.json"), options{
		Environment: "production",
		Resources:   "none",
		Yes:         "yes",
		SudoMode:    "skip",
		DryRun:      true,
	})
	if err != nil {
		t.Fatalf("applyEnvironment: %v", err)
	}

	if got := os.Getenv("APP_ROOT"); got != root {
		t.Fatalf("APP_ROOT = %q", got)
	}
	if got := os.Getenv("TARGET"); got != defaultTarget {
		t.Fatalf("TARGET = %q", got)
	}
	if got := os.Getenv("LOCATION"); got != defaultLocation {
		t.Fatalf("LOCATION = %q", got)
	}
	if got := os.Getenv("ENVIRONMENT"); got != "production" {
		t.Fatalf("ENVIRONMENT = %q", got)
	}
	if got := os.Getenv("RESOURCES"); got != "none" {
		t.Fatalf("RESOURCES = %q", got)
	}
	if got := os.Getenv("YES"); got != "yes" {
		t.Fatalf("YES = %q", got)
	}
	if got := os.Getenv("SUDO_MODE"); got != "skip" {
		t.Fatalf("SUDO_MODE = %q", got)
	}
	if got := os.Getenv("DRY_RUN"); got != "true" {
		t.Fatalf("DRY_RUN = %q", got)
	}

	restore()

	if got := os.Getenv("APP_ROOT"); got != "" {
		t.Fatalf("APP_ROOT after restore = %q", got)
	}
	if got := os.Getenv("TARGET"); got != "" {
		t.Fatalf("TARGET after restore = %q", got)
	}
}

func TestMarkCompleteWritesSetupAndResourceMarkers(t *testing.T) {
	root := t.TempDir()
	manifest := scenario.ServiceManifest{
		Lifecycle: scenario.Lifecycle{
			Setup: scenario.Phase{
				Steps: []scenario.PhaseStep{
					{Name: "base-setup"},
					{Name: "add-data"},
				},
			},
		},
	}

	if err := markComplete(root, manifest); err != nil {
		t.Fatalf("markComplete: %v", err)
	}

	setupMarker := filepath.Join(root, "data", ".setup-complete")
	data, err := os.ReadFile(setupMarker)
	if err != nil {
		t.Fatalf("read setup marker: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal setup marker: %v", err)
	}
	if payload["setup_version"] != "2.0.0" {
		t.Fatalf("setup_version = %v", payload["setup_version"])
	}
	if _, err := os.Stat(filepath.Join(root, "data", ".resources-populated")); err != nil {
		t.Fatalf("expected resources marker: %v", err)
	}
}

func TestRunSetupUsesLifecycleRunnerAndMarksComplete(t *testing.T) {
	restore := stubSetupDeps(t)
	defer restore()

	root := t.TempDir()
	home := t.TempDir()
	fake := &fakePhaseRunner{}
	projectScenario := scenario.Scenario{
		Slug:        "project-alpha",
		Path:        root,
		ServicePath: filepath.Join(root, ".vrooli", "service.json"),
		Manifest: scenario.ServiceManifest{
			Service: scenario.ServiceMetadata{Name: "project-alpha"},
		},
	}

	currentHostFn = func() vrooliruntime.Host { return vrooliruntime.Host{SupportsSetup: true, SupportsDevelop: true} }
	loadProjectFn = func(root string) (scenario.Scenario, error) { return projectScenario, nil }
	newPhaseRunnerFn = func(root, home string, stdout, stderr io.Writer) (phaseRunner, error) { return fake, nil }

	marked := false
	markCompleteFn = func(root string, manifest scenario.ServiceManifest) error {
		marked = true
		return nil
	}

	if err := RunSetup(root, home, []string{"--dry-run"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("RunSetup: %v", err)
	}
	if !marked {
		t.Fatal("expected markCompleteFn to be called")
	}
	if len(fake.runPhaseCalls) != 1 {
		t.Fatalf("len(fake.runPhaseCalls) = %d, want 1", len(fake.runPhaseCalls))
	}
	if got := fake.runPhaseCalls[0]; got.phase != "setup" || got.opts.CustomPath != root || !got.opts.ProjectMode {
		t.Fatalf("runPhase call = %+v", got)
	}
}

func TestRunDevelopRunsSetupWhenNeeded(t *testing.T) {
	restore := stubSetupDeps(t)
	defer restore()

	root := t.TempDir()
	home := t.TempDir()
	stdout := &strings.Builder{}
	fake := &fakePhaseRunner{
		setupNeeded: true,
		reasons:     []string{"forced"},
	}
	projectScenario := scenario.Scenario{
		Slug:        "project-alpha",
		Path:        root,
		ServicePath: filepath.Join(root, ".vrooli", "service.json"),
		Manifest: scenario.ServiceManifest{
			Service: scenario.ServiceMetadata{Name: "project-alpha"},
		},
	}

	currentHostFn = func() vrooliruntime.Host { return vrooliruntime.Host{SupportsSetup: true, SupportsDevelop: true} }
	loadProjectFn = func(root string) (scenario.Scenario, error) { return projectScenario, nil }
	newPhaseRunnerFn = func(root, home string, stdout, stderr io.Writer) (phaseRunner, error) { return fake, nil }
	markCompleteFn = func(root string, manifest scenario.ServiceManifest) error { return nil }

	if err := RunDevelop(root, home, nil, stdout, io.Discard); err != nil {
		t.Fatalf("RunDevelop: %v", err)
	}
	if len(fake.runPhaseCalls) != 2 {
		t.Fatalf("len(fake.runPhaseCalls) = %d, want 2", len(fake.runPhaseCalls))
	}
	if fake.runPhaseCalls[0].phase != "setup" || fake.runPhaseCalls[1].phase != "develop" {
		t.Fatalf("runPhase sequence = %+v", fake.runPhaseCalls)
	}
	if !strings.Contains(stdout.String(), "Running setup before develop") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunDevelopSkipsSetupWhenNotNeeded(t *testing.T) {
	restore := stubSetupDeps(t)
	defer restore()

	root := t.TempDir()
	home := t.TempDir()
	fake := &fakePhaseRunner{}
	projectScenario := scenario.Scenario{
		Slug:        "project-alpha",
		Path:        root,
		ServicePath: filepath.Join(root, ".vrooli", "service.json"),
		Manifest: scenario.ServiceManifest{
			Service: scenario.ServiceMetadata{Name: "project-alpha"},
		},
	}

	currentHostFn = func() vrooliruntime.Host { return vrooliruntime.Host{SupportsSetup: true, SupportsDevelop: true} }
	loadProjectFn = func(root string) (scenario.Scenario, error) { return projectScenario, nil }
	newPhaseRunnerFn = func(root, home string, stdout, stderr io.Writer) (phaseRunner, error) { return fake, nil }

	if err := RunDevelop(root, home, nil, io.Discard, io.Discard); err != nil {
		t.Fatalf("RunDevelop: %v", err)
	}
	if len(fake.runPhaseCalls) != 1 || fake.runPhaseCalls[0].phase != "develop" {
		t.Fatalf("runPhase calls = %+v", fake.runPhaseCalls)
	}
}

func TestRunSetupRejectsUnsupportedHost(t *testing.T) {
	restore := stubSetupDeps(t)
	defer restore()

	currentHostFn = func() vrooliruntime.Host {
		return vrooliruntime.Host{OS: "darwin", SupportsSetup: false, Notes: []string{"unsupported in test"}}
	}

	err := RunSetup(t.TempDir(), t.TempDir(), nil, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("expected unsupported host error")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("err = %v", err)
	}
}

type fakePhaseRunner struct {
	setupNeeded    bool
	reasons        []string
	setupNeededErr error
	runPhaseErr    error
	runPhaseCalls  []phaseInvocation
}

type phaseInvocation struct {
	name  string
	phase string
	opts  lifecycle.PhaseOptions
}

func (f *fakePhaseRunner) RunPhase(name, phaseName string, opts lifecycle.PhaseOptions) error {
	f.runPhaseCalls = append(f.runPhaseCalls, phaseInvocation{name: name, phase: phaseName, opts: opts})
	return f.runPhaseErr
}

func (f *fakePhaseRunner) SetupNeeded(item scenario.Scenario, force bool) (bool, []string, error) {
	if f.setupNeededErr != nil {
		return false, nil, f.setupNeededErr
	}
	return f.setupNeeded, append([]string(nil), f.reasons...), nil
}

func stubSetupDeps(t *testing.T) func() {
	t.Helper()
	originalCurrentHostFn := currentHostFn
	originalLoadProjectFn := loadProjectFn
	originalMarkCompleteFn := markCompleteFn
	originalNewPhaseRunnerFn := newPhaseRunnerFn
	return func() {
		currentHostFn = originalCurrentHostFn
		loadProjectFn = originalLoadProjectFn
		markCompleteFn = originalMarkCompleteFn
		newPhaseRunnerFn = originalNewPhaseRunnerFn
	}
}
