package packageapp

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/orchestrator"
	"github.com/vrooli/vrooli/internal/packagegov"
	testpackage "github.com/vrooli/vrooli/internal/packagegov/packagegovtest"
	"github.com/vrooli/vrooli/internal/process"
	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
)

type fakeScenarioRuntime struct {
	started []string
}

func (f *fakeScenarioRuntime) Lookup(name string) (orchestrator.Detail, bool, error) {
	return orchestrator.Detail{
		Scenario: scenariomodel.Scenario{Slug: name},
		Runtime:  process.ScenarioRuntime{ProcessCount: 1},
	}, true, nil
}

func (f *fakeScenarioRuntime) StartDetailed(name string, opts lifecycle.StartOptions) (orchestrator.StartResult, error) {
	f.started = append(f.started, name)
	return orchestrator.StartResult{Scenario: scenariomodel.Scenario{Slug: name}}, nil
}

type fakeScenarioRunner struct {
	stopped []string
	phases  []string
}

func (f *fakeScenarioRunner) Stop(name string, opts lifecycle.StopOptions) error {
	f.stopped = append(f.stopped, name)
	return nil
}

func (f *fakeScenarioRunner) RunPhaseDetailed(name, phase string, opts lifecycle.PhaseOptions) (lifecycle.PhaseResult, error) {
	f.phases = append(f.phases, name+":"+phase)
	return lifecycle.PhaseResult{ExecutedSteps: 1}, nil
}

func (f *fakeScenarioRunner) SetupAdmissionBlockers(string, string) ([]string, error) {
	return nil, nil
}

// variantScenarioRunner models a scenario whose shadow variant is running:
// setup admission names the shadow, and stopped-only setup defers.
type variantScenarioRunner struct {
	fakeScenarioRunner
	blockers []string
}

func (v *variantScenarioRunner) SetupAdmissionBlockers(string, string) ([]string, error) {
	return v.blockers, nil
}

func (v *variantScenarioRunner) RunSetupIfStopped(name string, opts lifecycle.PhaseOptions) (lifecycle.PhaseResult, error) {
	v.phases = append(v.phases, name+":setup-if-stopped")
	return lifecycle.PhaseResult{Scenario: name, Phase: "setup", Status: lifecycle.PhaseExecutionRunningDeferred}, nil
}

func restartingRefresh(service ScenarioRuntime, runner ScenarioPhaseRunner) *refreshRuntime {
	item := packagegov.Package{Name: "alpha"}
	item.Manifest.Package.Refresh.RestartRunningConsumers = true
	return &refreshRuntime{item: item, service: service, runner: runner}
}

// knw-1794078160449965368: a refresh with --restart stopped the live site,
// setup deferred because a shadow variant was running, and nothing restarted
// the live instance. The refresh must refuse before stopping anything.
func TestRefreshRestartRefusesBeforeStoppingWhenAnotherVariantRuns(t *testing.T) {
	service := &fakeScenarioRuntime{}
	runner := &variantScenarioRunner{blockers: []string{"landing-page-business-suite@shadow"}}

	status, err := restartingRefresh(service, runner).setupScenario("landing-page-business-suite")
	if err == nil || !strings.Contains(err.Error(), "landing-page-business-suite@shadow") {
		t.Fatalf("setupScenario = %q, %v; want a refusal naming the running shadow", status, err)
	}
	if len(runner.stopped) != 0 || len(runner.phases) != 0 || len(service.started) != 0 {
		t.Fatalf("refused refresh still acted: stopped=%v phases=%v started=%v", runner.stopped, runner.phases, service.started)
	}
}

// A variant that starts after the admission check still defers setup; the
// live instance this refresh stopped is started again.
func TestRefreshRestartRestoresLiveInstanceWhenSetupDefers(t *testing.T) {
	service := &fakeScenarioRuntime{}
	runner := &variantScenarioRunner{}

	status, err := restartingRefresh(service, runner).setupScenario("landing-page-business-suite")
	if err != nil || status != "running_setup_deferred" {
		t.Fatalf("setupScenario = %q, %v; want running_setup_deferred", status, err)
	}
	if len(runner.stopped) != 1 || len(service.started) != 1 {
		t.Fatalf("stopped=%v started=%v; the stopped live instance was not restored", runner.stopped, service.started)
	}
}

func TestRefreshUsesInterfaceBasedScenarioDependencies(t *testing.T) {
	root := t.TempDir()
	svc := Service{
		Root:   root,
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		ScenarioService: func() (ScenarioRuntime, error) {
			return &fakeScenarioRuntime{}, nil
		},
		ScenarioRunner: func() (ScenarioPhaseRunner, error) {
			return &fakeScenarioRunner{}, nil
		},
	}

	if _, ok := any(svc.ScenarioService).(func() (ScenarioRuntime, error)); !ok {
		t.Fatal("ScenarioService is not interface-based")
	}
	if _, ok := any(svc.ScenarioRunner).(func() (ScenarioPhaseRunner, error)); !ok {
		t.Fatal("ScenarioRunner is not interface-based")
	}
}

func TestTestUsesServerOwnedTestGenieTarget(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testpackage.WritePackageManifest(t, fixture.Root, "envkit-go", testpackage.PackageManifest("envkit-go"))
	var target string
	svc := Service{
		Root:   fixture.Root,
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		TestGenieRunner: func(got string, _, _ io.Writer) error {
			target = got
			return nil
		},
	}

	resp, err := svc.Test(" envkit-go ")
	if err != nil {
		t.Fatal(err)
	}
	if target != "package:envkit-go" {
		t.Fatalf("target = %q, want package:envkit-go", target)
	}
	if resp.Action != "test-genie" {
		t.Fatalf("action = %q, want test-genie", resp.Action)
	}
}

func TestEnsureGeneratedPackageCurrentRunsDeclaredGenerator(t *testing.T) {
	root := t.TempDir()
	marker := filepath.Join(root, "generated.marker")
	svc := Service{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	item := packagegov.Package{
		RootPath: root,
		Manifest: packagegov.Manifest{Package: packagegov.ManifestEntry{
			Name:             "generated-fixture",
			GeneratedOutputs: []packagegov.GeneratedOutput{{Name: "fixture"}},
			Lifecycle: packagegov.LifecyclePolicy{Generate: []packagegov.CommandSpec{{
				Name: "generate",
				Run:  []string{"sh", "-c", "printf generated > generated.marker"},
			}}},
		}},
	}

	if err := svc.ensureGeneratedPackageCurrent(item); err != nil {
		t.Fatalf("ensureGeneratedPackageCurrent: %v", err)
	}
	contents, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read generated marker: %v", err)
	}
	if string(contents) != "generated" {
		t.Fatalf("generated marker = %q, want generated", contents)
	}
}

func TestEnsureGeneratedPackageCurrentFallsBackToBuild(t *testing.T) {
	root := t.TempDir()
	marker := filepath.Join(root, "built.marker")
	svc := Service{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	item := packagegov.Package{
		Name:     "runtime-artifact-fixture",
		RootPath: root,
		Manifest: packagegov.Manifest{Package: packagegov.ManifestEntry{
			Name:             "runtime-artifact-fixture",
			GeneratedOutputs: []packagegov.GeneratedOutput{{Name: "runtime"}},
			Lifecycle: packagegov.LifecyclePolicy{Build: []packagegov.CommandSpec{{
				Name: "build",
				Run:  []string{"sh", "-c", "printf built > built.marker"},
			}}},
		}},
	}

	if err := svc.ensureGeneratedPackageCurrent(item); err != nil {
		t.Fatalf("ensureGeneratedPackageCurrent: %v", err)
	}
	contents, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read built marker: %v", err)
	}
	if string(contents) != "built" {
		t.Fatalf("built marker = %q, want built", contents)
	}
}

func TestTestPreflightsGeneratedPackageBeforeTestGenie(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testpackage.WritePackageManifest(t, fixture.Root, "generated-fixture", testpackage.PackageManifest(
		"generated-fixture",
		testpackage.WithPackageGeneratedOutputs(packagegov.GeneratedOutput{Name: "fixture"}),
		testpackage.WithPackageGenerateCommands(packagegov.CommandSpec{
			Name: "generate",
			Run:  []string{"sh", "-c", "printf generated > generated.marker"},
		}),
	))

	called := false
	svc := Service{
		Root:   fixture.Root,
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		TestGenieRunner: func(target string, _, _ io.Writer) error {
			called = true
			if target != "package:generated-fixture" {
				t.Fatalf("target = %q, want package:generated-fixture", target)
			}
			return nil
		},
	}

	if _, err := svc.Test("generated-fixture"); err != nil {
		t.Fatalf("Test: %v", err)
	}
	if !called {
		t.Fatal("expected Test Genie runner to be called")
	}
	if _, err := os.Stat(filepath.Join(fixture.Root, "packages", "generated-fixture", "generated.marker")); err != nil {
		t.Fatalf("generated marker missing: %v", err)
	}
}
