package lifecycle

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostreqrun"
	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/ports"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/projectstate"
	"github.com/vrooli/vrooli/internal/resources"
	resourcecontrol "github.com/vrooli/vrooli/internal/resources/control"
	resourcemanifest "github.com/vrooli/vrooli/internal/resources/manifest"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
	testpackage "github.com/vrooli/vrooli/packages/testkit-go/packagefixture"
	testresource "github.com/vrooli/vrooli/packages/testkit-go/resourcefixture"
	testscenario "github.com/vrooli/vrooli/packages/testkit-go/scenariofixture"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=9 | LAST: 2026-04-13

func newLifecycleRunnerForTest(t *testing.T, root, home string, mutate func(*lifecycleDeps), logger ...*slog.Logger) *Runner {
	t.Helper()
	deps := defaultLifecycleDeps()
	// Stub host-requirement enforcement by default so tests don't need a root
	// manifest on disk. Tests exercising the enforcement hook override this.
	deps.enforceHostRequirements = func(_ hostreqrun.Options) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{}, nil
	}
	if mutate != nil {
		mutate(&deps)
	}
	runner, err := newRunnerWithDeps(root, home, io.Discard, io.Discard, deps, logger...)
	if err != nil {
		t.Fatalf("newRunnerWithDeps: %v", err)
	}
	return runner
}

func intPtr(value int) *int {
	return &value
}

type fakeHostNode struct {
	modTime time.Time
	mode    os.FileMode
	data    []byte
}

type fakeHostProbe struct {
	nodes    map[string]fakeHostNode
	lookPath map[string]string
	env      map[string]string
	home     string
}

func newFakeHostProbe() *fakeHostProbe {
	return &fakeHostProbe{
		nodes:    map[string]fakeHostNode{},
		lookPath: map[string]string{},
		env:      map[string]string{},
		home:     "/home/tester",
	}
}

func (p *fakeHostProbe) addDir(path string, modTime time.Time) {
	p.nodes[filepath.Clean(path)] = fakeHostNode{modTime: modTime, mode: os.ModeDir | 0o755}
}

func (p *fakeHostProbe) addFile(path string, modTime time.Time, mode os.FileMode, data []byte) {
	p.nodes[filepath.Clean(path)] = fakeHostNode{modTime: modTime, mode: mode, data: append([]byte(nil), data...)}
}

func (p *fakeHostProbe) deps() hostProbeDeps {
	return hostProbeDeps{
		stat:        p.stat,
		readFile:    p.readFile,
		lookPath:    p.lookup,
		getenv:      p.getenv,
		userHomeDir: p.userHomeDir,
		walkDir:     p.walkDir,
	}
}

func (p *fakeHostProbe) stat(path string) (os.FileInfo, error) {
	node, ok := p.nodes[filepath.Clean(path)]
	if !ok {
		return nil, os.ErrNotExist
	}
	return fakeFileInfo{name: filepath.Base(path), node: node}, nil
}

func (p *fakeHostProbe) readFile(path string) ([]byte, error) {
	node, ok := p.nodes[filepath.Clean(path)]
	if !ok || node.mode.IsDir() {
		return nil, os.ErrNotExist
	}
	return append([]byte(nil), node.data...), nil
}

func (p *fakeHostProbe) lookup(name string) (string, error) {
	path, ok := p.lookPath[name]
	if !ok {
		return "", exec.ErrNotFound
	}
	return path, nil
}

func (p *fakeHostProbe) getenv(key string) string {
	return p.env[key]
}

func (p *fakeHostProbe) userHomeDir() (string, error) {
	return p.home, nil
}

func (p *fakeHostProbe) walkDir(root string, walkFn fs.WalkDirFunc) error {
	root = filepath.Clean(root)
	if _, ok := p.nodes[root]; !ok {
		return os.ErrNotExist
	}
	paths := make([]string, 0, len(p.nodes))
	for path := range p.nodes {
		if path == root || strings.HasPrefix(path, root+string(filepath.Separator)) {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	for _, path := range paths {
		node := p.nodes[path]
		entry := fakeDirEntry{name: filepath.Base(path), node: node}
		if err := walkFn(path, entry, nil); err != nil {
			return err
		}
	}
	return nil
}

type fakeFileInfo struct {
	name string
	node fakeHostNode
}

func (f fakeFileInfo) Name() string       { return f.name }
func (f fakeFileInfo) Size() int64        { return int64(len(f.node.data)) }
func (f fakeFileInfo) Mode() os.FileMode  { return f.node.mode }
func (f fakeFileInfo) ModTime() time.Time { return f.node.modTime }
func (f fakeFileInfo) IsDir() bool        { return f.node.mode.IsDir() }
func (f fakeFileInfo) Sys() any           { return nil }

type fakeDirEntry struct {
	name string
	node fakeHostNode
}

func (d fakeDirEntry) Name() string      { return d.name }
func (d fakeDirEntry) IsDir() bool       { return d.node.mode.IsDir() }
func (d fakeDirEntry) Type() fs.FileMode { return fs.FileMode(d.node.mode) }
func (d fakeDirEntry) Info() (fs.FileInfo, error) {
	return fakeFileInfo(d), nil
}

func TestRunPhaseDetailedReportsUndefinedPhase(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testresource.WritePortRegistry(t, root, nil)
	writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "alpha"},
	})

	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	result, err := runner.RunPhaseDetailed("alpha", "setup", PhaseOptions{})
	if err != nil {
		t.Fatalf("RunPhaseDetailed: %v", err)
	}
	if result.Defined {
		t.Fatalf("result.Defined = %v, want false", result.Defined)
	}
	if result.Status != PhaseExecutionUndefined {
		t.Fatalf("result.Status = %q, want %q", result.Status, PhaseExecutionUndefined)
	}
}

func TestExecutePhaseDetailedReplaysTailToErrOnFailure(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testresource.WritePortRegistry(t, root, nil)
	writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "alpha"},
		Lifecycle: scenario.Lifecycle{
			Setup: scenario.Phase{
				Steps: []scenario.PhaseStep{
					{Name: "noisy-fail", Run: "echo 'alpha line'; echo 'beta line'; echo 'gamma line'; exit 3"},
				},
			},
		},
	})

	var stderr bytes.Buffer
	runner, err := NewRunner(root, home, io.Discard, &stderr)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner = runner.WithVerbosity(VerbosityNormal)

	item, err := scenario.Load(root, "alpha", scenario.SandboxEnv{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	_, err = runner.ExecutePhaseDetailed(item, "setup", map[string]string{}, nil, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("expected phase to fail")
	}
	replay := stderr.String()
	for _, line := range []string{"alpha line", "beta line", "gamma line"} {
		if !strings.Contains(replay, line) {
			t.Errorf("stderr missing replay line %q: %q", line, replay)
		}
	}
	if !strings.Contains(replay, "full log:") {
		t.Errorf("stderr missing log pointer: %q", replay)
	}
}

func TestExecutePhaseDetailedSkipsReplayAtVerbose(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testresource.WritePortRegistry(t, root, nil)
	writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "alpha"},
		Lifecycle: scenario.Lifecycle{
			Setup: scenario.Phase{
				Steps: []scenario.PhaseStep{
					{Name: "noisy-fail", Run: "echo 'zeta line'; exit 3"},
				},
			},
		},
	})

	var stderr bytes.Buffer
	runner, err := NewRunner(root, home, io.Discard, &stderr)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner = runner.WithVerbosity(VerbosityVerbose)

	item, err := scenario.Load(root, "alpha", scenario.SandboxEnv{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	_, err = runner.ExecutePhaseDetailed(item, "setup", map[string]string{}, nil, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("expected phase to fail")
	}
	if strings.Contains(stderr.String(), "full log:") {
		t.Errorf("verbose mode should not replay (tool stdout already teed): %q", stderr.String())
	}
}

func TestExecutePhaseDetailedWrapsStepFailuresWithContext(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testresource.WritePortRegistry(t, root, nil)
	writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "alpha"},
		Lifecycle: scenario.Lifecycle{
			Setup: scenario.Phase{
				Steps: []scenario.PhaseStep{
					{Name: "explode", Run: "exit 7"},
				},
			},
		},
	})

	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	item, err := scenario.Load(root, "alpha", scenario.SandboxEnv{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	result, err := runner.ExecutePhaseDetailed(item, "setup", map[string]string{}, nil, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("expected wrapped phase error")
	}
	if result.Defined != true {
		t.Fatalf("result = %+v", result)
	}
	var phaseErr *PhaseStepError
	if !errors.As(err, &phaseErr) {
		t.Fatalf("expected PhaseStepError, got %T", err)
	}
	if phaseErr.Scenario != "alpha" || phaseErr.Phase != "setup" || phaseErr.Step != "explode" {
		t.Fatalf("phaseErr = %+v", phaseErr)
	}
	if phaseErr.ExitCode() != 7 {
		t.Fatalf("phaseErr.ExitCode() = %d, want 7", phaseErr.ExitCode())
	}
	if !strings.Contains(phaseErr.Error(), process.ScenarioLifecycleLogPath(home, "alpha")) {
		t.Fatalf("phaseErr.Error() = %q", phaseErr.Error())
	}
}

func TestRunPhaseDetailedWritesLifecycleRunBoundaries(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testresource.WritePortRegistry(t, root, nil)
	writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "alpha"},
		Lifecycle: scenario.Lifecycle{
			Setup: scenario.Phase{
				Steps: []scenario.PhaseStep{
					{Name: "ok", Run: "echo 'setup ok'"},
				},
			},
		},
	})

	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	if _, err := runner.RunPhaseDetailed("alpha", "setup", PhaseOptions{}); err != nil {
		t.Fatalf("RunPhaseDetailed: %v", err)
	}

	data, err := os.ReadFile(process.ScenarioLifecycleLogPath(home, "alpha"))
	if err != nil {
		t.Fatalf("read lifecycle log: %v", err)
	}
	log := string(data)
	for _, want := range []string{
		"=== VROOLI LIFECYCLE RUN START ===",
		"scenario: alpha",
		"operation: phase",
		"phase: setup",
		"setup ok",
		"=== VROOLI LIFECYCLE RUN END ===",
		"status: completed",
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("lifecycle log missing %q:\n%s", want, log)
		}
	}
}

func TestRunPhaseDetailedWritesFailureBoundaryDetails(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testresource.WritePortRegistry(t, root, nil)
	writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "alpha"},
		Lifecycle: scenario.Lifecycle{
			Setup: scenario.Phase{
				Steps: []scenario.PhaseStep{
					{Name: "explode", Run: "echo 'before fail'; exit 7"},
				},
			},
		},
	})

	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	if _, err := runner.RunPhaseDetailed("alpha", "setup", PhaseOptions{}); err == nil {
		t.Fatal("expected RunPhaseDetailed to fail")
	}

	data, err := os.ReadFile(process.ScenarioLifecycleLogPath(home, "alpha"))
	if err != nil {
		t.Fatalf("read lifecycle log: %v", err)
	}
	log := string(data)
	for _, want := range []string{
		"before fail",
		"status: failed",
		"step: explode",
		"exit_code: 7",
		"error: scenario \"alpha\" phase \"setup\" step \"explode\" failed with exit code 7",
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("lifecycle log missing %q:\n%s", want, log)
		}
	}
}

func TestEnsureDependenciesBestEffortMarksMissingRequiredDependencyAsFailed(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixtureManifest(t, root, testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithDependencies(scenario.Dependencies{
			Scenarios: map[string]scenario.Dependency{
				"missing-beta": {Required: true},
			},
		}),
	))

	runner := newLifecycleRunnerForTest(t, root, home, nil)
	item, err := scenario.Load(root, "alpha", scenario.SandboxEnv{})
	if err != nil {
		t.Fatalf("Load(alpha): %v", err)
	}

	failed, err := runner.ensureDependencies(item, StartOptions{BestEffort: true}, map[string]struct{}{}, []string{"alpha"})
	if err != nil {
		t.Fatalf("ensureDependencies(best-effort): %v", err)
	}
	if len(failed) != 1 || failed[0] != "missing-beta" {
		t.Fatalf("failed dependencies = %#v, want [missing-beta]", failed)
	}
}

func TestEnsureDependenciesTryStartMarksMissingOptionalDependencyAsFailed(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixtureManifest(t, root, testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithDependencies(scenario.Dependencies{
			Scenarios: map[string]scenario.Dependency{
				"missing-beta": {Required: false, StartupPolicy: "try_start"},
			},
		}),
	))

	runner := newLifecycleRunnerForTest(t, root, home, nil)
	item, err := scenario.Load(root, "alpha", scenario.SandboxEnv{})
	if err != nil {
		t.Fatalf("Load(alpha): %v", err)
	}

	failed, err := runner.ensureDependencies(item, StartOptions{}, map[string]struct{}{}, []string{"alpha"})
	if err != nil {
		t.Fatalf("ensureDependencies(try-start): %v", err)
	}
	if len(failed) != 1 || failed[0] != "missing-beta" {
		t.Fatalf("failed dependencies = %#v, want [missing-beta]", failed)
	}
}

func TestEnsureDependenciesIgnoreSkipsMissingOptionalDependency(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixtureManifest(t, root, testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithDependencies(scenario.Dependencies{
			Scenarios: map[string]scenario.Dependency{
				"missing-beta": {Required: false, StartupPolicy: "ignore"},
			},
		}),
	))

	runner := newLifecycleRunnerForTest(t, root, home, nil)
	item, err := scenario.Load(root, "alpha", scenario.SandboxEnv{})
	if err != nil {
		t.Fatalf("Load(alpha): %v", err)
	}

	failed, err := runner.ensureDependencies(item, StartOptions{}, map[string]struct{}{}, []string{"alpha"})
	if err != nil {
		t.Fatalf("ensureDependencies(ignore): %v", err)
	}
	if len(failed) != 0 {
		t.Fatalf("failed dependencies = %#v, want none", failed)
	}
}

func TestResolveDependencyDecisionUsesNormalizedContract(t *testing.T) {
	required := resolveDependencyDecision(scenario.Dependency{Enabled: true, Required: true}, false)
	if required.policy != scenario.DependencyStartupPolicyMustStart || required.skip || required.continueOnFailure {
		t.Fatalf("required decision = %+v", required)
	}

	tryStart := resolveDependencyDecision(scenario.Dependency{
		Enabled:       true,
		Required:      false,
		StartupPolicy: scenario.DependencyStartupPolicyTryStart,
	}, false)
	if tryStart.policy != scenario.DependencyStartupPolicyTryStart || tryStart.skip || !tryStart.continueOnFailure {
		t.Fatalf("try-start decision = %+v", tryStart)
	}

	disabled := resolveDependencyDecision(scenario.Dependency{
		Enabled:       false,
		Required:      true,
		StartupPolicy: scenario.DependencyStartupPolicyMustStart,
	}, false)
	if disabled.policy != scenario.DependencyStartupPolicyIgnore || !disabled.skip || disabled.continueOnFailure {
		t.Fatalf("disabled decision = %+v", disabled)
	}

	bestEffort := resolveDependencyDecision(scenario.Dependency{Enabled: true, Required: true}, true)
	if !bestEffort.continueOnFailure {
		t.Fatalf("best-effort decision = %+v", bestEffort)
	}
}

func TestEnsureResourceDependenciesBlocksRequiredUnavailableResource(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixtureManifest(t, root, testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithDependencies(scenario.Dependencies{
			Resources: map[string]scenario.Dependency{
				"postgres": {Enabled: true, Required: true},
			},
		}),
	))

	now := time.Unix(0, 0)
	runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.resourceStatus = func(name string, fast bool) (resourcecontrol.Status, error) {
			return resourcecontrol.Status{Resource: resources.Resource{Name: name}, Running: false, StatusCode: resourcecontrol.StatusCodeUnavailable}, nil
		}
		deps.runResource = func(name string, args []string, stdout, stderr io.Writer) error {
			return nil
		}
		deps.now = func() time.Time { return now }
		deps.sleep = func(d time.Duration) { now = now.Add(resourceDependencyReadyTimeout) }
	})
	item, err := scenario.Load(root, "alpha", scenario.SandboxEnv{})
	if err != nil {
		t.Fatalf("Load(alpha): %v", err)
	}

	_, err = runner.ensureResourceDependencies(item, StartOptions{})
	if err == nil || !strings.Contains(err.Error(), "not ready after start") {
		t.Fatalf("ensureResourceDependencies error = %v", err)
	}
}

func TestEnsureResourceDependenciesTryStartMarksUnavailableResourceAsFailed(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixtureManifest(t, root, testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithDependencies(scenario.Dependencies{
			Resources: map[string]scenario.Dependency{
				"qdrant": {Enabled: true, StartupPolicy: scenario.DependencyStartupPolicyTryStart},
			},
		}),
	))

	statusCalls := 0
	now := time.Unix(0, 0)
	runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.resourceStatus = func(name string, fast bool) (resourcecontrol.Status, error) {
			statusCalls++
			return resourcecontrol.Status{Resource: resources.Resource{Name: name}, Running: false, StatusCode: resourcecontrol.StatusCodeUnavailable}, nil
		}
		deps.runResource = func(name string, args []string, stdout, stderr io.Writer) error {
			return nil
		}
		deps.now = func() time.Time { return now }
		deps.sleep = func(d time.Duration) { now = now.Add(resourceDependencyReadyTimeout) }
	})
	item, err := scenario.Load(root, "alpha", scenario.SandboxEnv{})
	if err != nil {
		t.Fatalf("Load(alpha): %v", err)
	}

	failed, err := runner.ensureResourceDependencies(item, StartOptions{})
	if err != nil {
		t.Fatalf("ensureResourceDependencies: %v", err)
	}
	if len(failed) != 1 || failed[0] != "qdrant" {
		t.Fatalf("failed resources = %#v, want [qdrant]", failed)
	}
	if statusCalls < 2 {
		t.Fatalf("statusCalls = %d, want at least 2", statusCalls)
	}
}

func TestEnsureResourceDependenciesIgnoreSkipsOptionalResource(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixtureManifest(t, root, testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithDependencies(scenario.Dependencies{
			Resources: map[string]scenario.Dependency{
				"redis": {Enabled: true, StartupPolicy: scenario.DependencyStartupPolicyIgnore},
			},
		}),
	))

	runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.resourceStatus = func(name string, fast bool) (resourcecontrol.Status, error) {
			t.Fatalf("resourceStatus should not be called for ignored resource %s", name)
			return resourcecontrol.Status{}, nil
		}
		deps.runResource = func(name string, args []string, stdout, stderr io.Writer) error {
			t.Fatalf("runResource should not be called for ignored resource %s", name)
			return nil
		}
	})
	item, err := scenario.Load(root, "alpha", scenario.SandboxEnv{})
	if err != nil {
		t.Fatalf("Load(alpha): %v", err)
	}

	failed, err := runner.ensureResourceDependencies(item, StartOptions{})
	if err != nil {
		t.Fatalf("ensureResourceDependencies: %v", err)
	}
	if len(failed) != 0 {
		t.Fatalf("failed resources = %#v, want none", failed)
	}
}

func TestEnsureResourceDependenciesCallsEnsureWhenCapAndConfigPresent(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixtureManifest(t, root, testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithDependencies(scenario.Dependencies{
			Resources: map[string]scenario.Dependency{
				"ollama": {
					Enabled:       true,
					Required:      true,
					StartupPolicy: scenario.DependencyStartupPolicyMustStart,
					Config:        []byte(`{"models":["qwen3:4b"]}`),
				},
			},
		}),
	))

	var ranArgs []string
	runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.resourceStatus = func(name string, _ bool) (resourcecontrol.Status, error) {
			healthy := true
			return resourcecontrol.Status{
				Resource: resources.Resource{Name: name},
				Running:  true,
				Healthy:  &healthy,
			}, nil
		}
		deps.resourceManifest = func(_ string) (resourcemanifest.ResourceManifest, error) {
			return resourcemanifest.ResourceManifest{
				Capabilities: resourcemanifest.ResourceManifestCapabilities{SupportsEnsure: true},
			}, nil
		}
		deps.runResourceCLI = func(_ string, args []string, _, _ io.Writer) error {
			ranArgs = append([]string(nil), args...)
			return nil
		}
	})
	item, err := scenario.Load(root, "alpha", scenario.SandboxEnv{})
	if err != nil {
		t.Fatalf("Load(alpha): %v", err)
	}
	if _, err := runner.ensureResourceDependencies(item, StartOptions{}); err != nil {
		t.Fatalf("ensureResourceDependencies: %v", err)
	}

	if len(ranArgs) != 3 || ranArgs[0] != "ensure" || ranArgs[1] != "--config-base64" {
		t.Fatalf("ensure argv = %#v, want [ensure --config-base64 <b64>]", ranArgs)
	}
	decoded, err := base64.StdEncoding.DecodeString(ranArgs[2])
	if err != nil {
		t.Fatalf("decode config-base64: %v", err)
	}
	if !strings.Contains(string(decoded), `"qwen3:4b"`) {
		t.Errorf("decoded config missing model: %s", decoded)
	}
}

func TestEnsureResourceDependenciesSkipsEnsureWhenCapabilityOff(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixtureManifest(t, root, testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithDependencies(scenario.Dependencies{
			Resources: map[string]scenario.Dependency{
				"postgres": {
					Enabled:       true,
					Required:      true,
					StartupPolicy: scenario.DependencyStartupPolicyMustStart,
					Config:        []byte(`{"whatever":true}`),
				},
			},
		}),
	))

	var ensureCalled bool
	runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.resourceStatus = func(name string, _ bool) (resourcecontrol.Status, error) {
			healthy := true
			return resourcecontrol.Status{Resource: resources.Resource{Name: name}, Running: true, Healthy: &healthy}, nil
		}
		deps.resourceManifest = func(_ string) (resourcemanifest.ResourceManifest, error) {
			return resourcemanifest.ResourceManifest{}, nil // SupportsEnsure=false
		}
		deps.runResourceCLI = func(_ string, _ []string, _, _ io.Writer) error {
			ensureCalled = true
			return nil
		}
	})
	item, err := scenario.Load(root, "alpha", scenario.SandboxEnv{})
	if err != nil {
		t.Fatalf("Load(alpha): %v", err)
	}
	if _, err := runner.ensureResourceDependencies(item, StartOptions{}); err != nil {
		t.Fatalf("ensureResourceDependencies: %v", err)
	}
	if ensureCalled {
		t.Error("ensure must not be invoked when SupportsEnsure=false")
	}
}

func TestEnsureResourceDependenciesSkipsEnsureWhenConfigEmpty(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixtureManifest(t, root, testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithDependencies(scenario.Dependencies{
			Resources: map[string]scenario.Dependency{
				"ollama": {
					Enabled:       true,
					Required:      true,
					StartupPolicy: scenario.DependencyStartupPolicyMustStart,
					// no Config → no ensure should run
				},
			},
		}),
	))

	var manifestCalled bool
	var ensureCalled bool
	runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.resourceStatus = func(name string, _ bool) (resourcecontrol.Status, error) {
			healthy := true
			return resourcecontrol.Status{Resource: resources.Resource{Name: name}, Running: true, Healthy: &healthy}, nil
		}
		deps.resourceManifest = func(_ string) (resourcemanifest.ResourceManifest, error) {
			manifestCalled = true
			return resourcemanifest.ResourceManifest{
				Capabilities: resourcemanifest.ResourceManifestCapabilities{SupportsEnsure: true},
			}, nil
		}
		deps.runResourceCLI = func(_ string, _ []string, _, _ io.Writer) error {
			ensureCalled = true
			return nil
		}
	})
	item, err := scenario.Load(root, "alpha", scenario.SandboxEnv{})
	if err != nil {
		t.Fatalf("Load(alpha): %v", err)
	}
	if _, err := runner.ensureResourceDependencies(item, StartOptions{}); err != nil {
		t.Fatalf("ensureResourceDependencies: %v", err)
	}
	if manifestCalled {
		t.Error("resourceManifest should not be loaded when Config is empty (short-circuit)")
	}
	if ensureCalled {
		t.Error("ensure must not be invoked when Config is empty")
	}
}

func TestEnsureResourceDependenciesFailsRequiredDependencyWhenEnsureFails(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixtureManifest(t, root, testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithDependencies(scenario.Dependencies{
			Resources: map[string]scenario.Dependency{
				"ollama": {
					Enabled:       true,
					Required:      true,
					StartupPolicy: scenario.DependencyStartupPolicyMustStart,
					Config:        []byte(`{"models":["qwen3:4b"]}`),
				},
			},
		}),
	))

	runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.resourceStatus = func(name string, _ bool) (resourcecontrol.Status, error) {
			healthy := true
			return resourcecontrol.Status{Resource: resources.Resource{Name: name}, Running: true, Healthy: &healthy}, nil
		}
		deps.resourceManifest = func(_ string) (resourcemanifest.ResourceManifest, error) {
			return resourcemanifest.ResourceManifest{
				Capabilities: resourcemanifest.ResourceManifestCapabilities{SupportsEnsure: true},
			}, nil
		}
		deps.runResourceCLI = func(_ string, _ []string, _, _ io.Writer) error {
			return errors.New("pull failed")
		}
	})
	item, err := scenario.Load(root, "alpha", scenario.SandboxEnv{})
	if err != nil {
		t.Fatalf("Load(alpha): %v", err)
	}
	_, err = runner.ensureResourceDependencies(item, StartOptions{})
	if err == nil || !strings.Contains(err.Error(), "ensure resource dependency ollama") {
		t.Fatalf("expected ensure failure to surface on required dep, got %v", err)
	}
}

func TestEnsureResourceDependenciesContinuesOnEnsureFailureWhenBestEffort(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixtureManifest(t, root, testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithDependencies(scenario.Dependencies{
			Resources: map[string]scenario.Dependency{
				"ollama": {
					Enabled:          true,
					Required:         true,
					StartupPolicy:    scenario.DependencyStartupPolicyTryStart,
					DegradedBehavior: "fallback to cloud",
					Config:           []byte(`{"models":["qwen3:4b"]}`),
				},
			},
		}),
	))

	runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.resourceStatus = func(name string, _ bool) (resourcecontrol.Status, error) {
			healthy := true
			return resourcecontrol.Status{Resource: resources.Resource{Name: name}, Running: true, Healthy: &healthy}, nil
		}
		deps.resourceManifest = func(_ string) (resourcemanifest.ResourceManifest, error) {
			return resourcemanifest.ResourceManifest{
				Capabilities: resourcemanifest.ResourceManifestCapabilities{SupportsEnsure: true},
			}, nil
		}
		deps.runResourceCLI = func(_ string, _ []string, _, _ io.Writer) error {
			return errors.New("transient pull failure")
		}
	})
	item, err := scenario.Load(root, "alpha", scenario.SandboxEnv{})
	if err != nil {
		t.Fatalf("Load(alpha): %v", err)
	}
	failed, err := runner.ensureResourceDependencies(item, StartOptions{})
	if err != nil {
		t.Fatalf("try_start ensure failure should be best-effort, got %v", err)
	}
	if len(failed) != 1 || failed[0] != "ollama" {
		t.Fatalf("failed resources = %#v, want [ollama]", failed)
	}
}

func TestRunnerStartContinuesWithTryStartResourceDependency(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixtureManifest(t, root, testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithDependencies(scenario.Dependencies{
			Resources: map[string]scenario.Dependency{
				"qdrant": {Enabled: true, StartupPolicy: scenario.DependencyStartupPolicyTryStart},
			},
		}),
	))

	now := time.Unix(0, 0)
	runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.resourceStatus = func(name string, fast bool) (resourcecontrol.Status, error) {
			return resourcecontrol.Status{Resource: resources.Resource{Name: name}, Running: false, StatusCode: resourcecontrol.StatusCodeUnavailable}, nil
		}
		deps.runResource = func(name string, args []string, stdout, stderr io.Writer) error {
			return nil
		}
		deps.now = func() time.Time { return now }
		deps.sleep = func(d time.Duration) { now = now.Add(resourceDependencyReadyTimeout) }
	})

	result, err := runner.Start("alpha", StartOptions{})
	if err != nil {
		t.Fatalf("Start(alpha): %v", err)
	}
	if len(result.FailedResources) != 1 || result.FailedResources[0] != "qdrant" {
		t.Fatalf("FailedResources = %#v, want [qdrant]", result.FailedResources)
	}
}

func TestRunnerStartRollsBackBackgroundProcessRecordsAndLocksOnHealthFailure(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixture(t, root, "alpha")

	item, err := scenario.Load(root, "alpha", scenario.SandboxEnv{})
	if err != nil {
		t.Fatalf("Load(alpha): %v", err)
	}
	item.Manifest.Lifecycle.Health = &scenario.HealthConfig{
		Checks: []scenario.HealthCheck{{
			Name:     "api",
			Type:     "unsupported",
			Critical: true,
		}},
		Timeout:  25,
		Interval: 1,
	}
	writeLifecycleFixtureManifest(t, root, item.Manifest)

	runner := newLifecycleRunnerForTest(t, root, home, nil)
	_, err = runner.Start("alpha", StartOptions{})
	if err == nil {
		t.Fatal("expected health failure")
	}

	records, readErr := process.ReadScenarioRecords(home, "alpha")
	if readErr != nil {
		t.Fatalf("ReadScenarioRecords(alpha): %v", readErr)
	}
	if len(records) != 0 {
		t.Fatalf("records after failed start = %#v, want none", records)
	}

	locks, lockErr := runner.Ports.LocksForScenario("alpha")
	if lockErr != nil {
		t.Fatalf("LocksForScenario(alpha): %v", lockErr)
	}
	if len(locks) != 0 {
		t.Fatalf("locks after failed start = %#v, want none", locks)
	}
}

func TestRunnerStartRollsBackLocksOnSetupFailure(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixture(t, root, "alpha")

	item, err := scenario.Load(root, "alpha", scenario.SandboxEnv{})
	if err != nil {
		t.Fatalf("Load(alpha): %v", err)
	}
	item.Manifest.Lifecycle.Setup = scenario.Phase{
		Condition: &scenario.Condition{
			Checks: []scenario.ConditionCheck{
				{Type: "binaries", Targets: []string{"api/mock-api"}},
			},
		},
		Steps: []scenario.PhaseStep{
			{Name: "explode", Run: "exit 9"},
		},
	}
	writeLifecycleFixtureManifest(t, root, item.Manifest)

	runner := newLifecycleRunnerForTest(t, root, home, nil)
	_, err = runner.Start("alpha", StartOptions{})
	if err == nil {
		t.Fatal("expected setup failure")
	}

	records, readErr := process.ReadScenarioRecords(home, "alpha")
	if readErr != nil {
		t.Fatalf("ReadScenarioRecords(alpha): %v", readErr)
	}
	if len(records) != 0 {
		t.Fatalf("records after failed setup = %#v, want none", records)
	}

	locks, lockErr := runner.Ports.LocksForScenario("alpha")
	if lockErr != nil {
		t.Fatalf("LocksForScenario(alpha): %v", lockErr)
	}
	if len(locks) != 0 {
		t.Fatalf("locks after failed setup = %#v, want none", locks)
	}
}

func TestRunnerStartDualWritesScenarioRuntimeRegistry(t *testing.T) {
	t.Setenv(runtimeRegistryModeEnv, runtimeRegistryModeDual)
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixture(t, root, "alpha")

	runner := newLifecycleRunnerForTest(t, root, home, nil)
	result, err := runner.Start("alpha", StartOptions{})
	if err != nil {
		t.Fatalf("Start(alpha): %v", err)
	}
	defer func() {
		if err := runner.Stop("alpha", StopOptions{}); err != nil {
			t.Fatalf("Stop(alpha): %v", err)
		}
	}()
	if result.AlreadyRunning {
		t.Fatal("result.AlreadyRunning = true, want fresh start")
	}

	store, err := scenarioruntime.NewSQLiteStore(context.Background(), scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer store.Close()

	instances, err := store.ListInstances(context.Background(), scenarioruntime.InstanceFilter{
		Scenario: "alpha",
		Statuses: []string{scenarioruntime.StatusRunning},
	})
	if err != nil {
		t.Fatalf("ListInstances(running): %v", err)
	}
	if len(instances) != 1 {
		t.Fatalf("running instances = %#v, want one", instances)
	}
	instance := instances[0]
	if instance.WorkingDir == "" || instance.ScopePath == "" || instance.OwnerPID == nil || instance.HostBootID == "" || instance.HostSessionID == "" {
		t.Fatalf("instance missing lifecycle metadata: %#v", instance)
	}

	claims, err := store.ListPortClaims(context.Background(), scenarioruntime.PortClaimFilter{InstanceID: instance.InstanceID})
	if err != nil {
		t.Fatalf("ListPortClaims: %v", err)
	}
	if len(claims) != 1 {
		t.Fatalf("claims = %#v, want one", claims)
	}
	if claims[0].Status != scenarioruntime.ClaimStatusBound || claims[0].PortName != "api" || claims[0].EnvVar != "API_PORT" {
		t.Fatalf("claim = %#v, want bound api/API_PORT claim", claims[0])
	}
	if result.AllocatedPorts["api"] != claims[0].Port {
		t.Fatalf("claim port = %d, result api port = %d", claims[0].Port, result.AllocatedPorts["api"])
	}

	refs, err := store.ListProcessRefs(context.Background(), instance.InstanceID)
	if err != nil {
		t.Fatalf("ListProcessRefs: %v", err)
	}
	if len(refs) != 1 || refs[0].Step != "start-api" || refs[0].PID == nil || refs[0].HostBootID == "" {
		t.Fatalf("process refs = %#v, want start-api with pid", refs)
	}

	health, err := store.GetHealthSnapshot(context.Background(), instance.InstanceID)
	if err != nil {
		t.Fatalf("GetHealthSnapshot: %v", err)
	}
	if health.Status != scenarioruntime.HealthStatusHealthy || health.Readiness == nil || !*health.Readiness {
		t.Fatalf("health = %#v, want healthy and ready", health)
	}
}

func TestRunnerStartHonorsRuntimeRegistryAllowlist(t *testing.T) {
	t.Setenv(runtimeRegistryModeEnv, runtimeRegistryModeDual)
	t.Setenv(scenarioruntime.AllowlistEnv, "beta")
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixture(t, root, "alpha")

	runner := newLifecycleRunnerForTest(t, root, home, nil)
	if _, err := runner.Start("alpha", StartOptions{}); err != nil {
		t.Fatalf("Start(alpha): %v", err)
	}
	defer func() {
		if err := runner.Stop("alpha", StopOptions{}); err != nil {
			t.Fatalf("Stop(alpha): %v", err)
		}
	}()

	dbPath, err := scenarioruntime.DefaultDBPath(home)
	if err != nil {
		t.Fatalf("DefaultDBPath: %v", err)
	}
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Fatalf("runtime registry db stat err = %v, want not exist for non-allowlisted start", err)
	}
}

func TestRunnerStopDualWritesStoppedRuntimeRegistry(t *testing.T) {
	t.Setenv(runtimeRegistryModeEnv, runtimeRegistryModeDual)
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixture(t, root, "alpha")

	runner := newLifecycleRunnerForTest(t, root, home, nil)
	if _, err := runner.Start("alpha", StartOptions{}); err != nil {
		t.Fatalf("Start(alpha): %v", err)
	}
	if err := runner.Stop("alpha", StopOptions{}); err != nil {
		t.Fatalf("Stop(alpha): %v", err)
	}

	store, err := scenarioruntime.NewSQLiteStore(context.Background(), scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer store.Close()

	instances, err := store.ListInstances(context.Background(), scenarioruntime.InstanceFilter{Scenario: "alpha"})
	if err != nil {
		t.Fatalf("ListInstances: %v", err)
	}
	if len(instances) != 1 {
		t.Fatalf("instances = %#v, want one", instances)
	}
	if instances[0].Status != scenarioruntime.StatusStopped || instances[0].StoppedAt == nil {
		t.Fatalf("instance = %#v, want stopped with stopped_at", instances[0])
	}

	claims, err := store.ListPortClaims(context.Background(), scenarioruntime.PortClaimFilter{InstanceID: instances[0].InstanceID})
	if err != nil {
		t.Fatalf("ListPortClaims: %v", err)
	}
	if len(claims) != 1 || claims[0].Status != scenarioruntime.ClaimStatusReleased {
		t.Fatalf("claims = %#v, want released claim", claims)
	}
}

func TestRunnerStartHealthFailureDualWritesFailedRuntimeRegistry(t *testing.T) {
	t.Setenv(runtimeRegistryModeEnv, runtimeRegistryModeDual)
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixture(t, root, "alpha")

	item, err := scenario.Load(root, "alpha", scenario.SandboxEnv{})
	if err != nil {
		t.Fatalf("Load(alpha): %v", err)
	}
	item.Manifest.Lifecycle.Health = &scenario.HealthConfig{
		Checks: []scenario.HealthCheck{{
			Name:     "api",
			Type:     "unsupported",
			Critical: true,
		}},
		Timeout:  25,
		Interval: 1,
	}
	writeLifecycleFixtureManifest(t, root, item.Manifest)

	runner := newLifecycleRunnerForTest(t, root, home, nil)
	_, err = runner.Start("alpha", StartOptions{})
	if err == nil {
		t.Fatal("expected health failure")
	}

	store, openErr := scenarioruntime.NewSQLiteStore(context.Background(), scenarioruntime.Config{HomeDir: home})
	if openErr != nil {
		t.Fatalf("NewSQLiteStore: %v", openErr)
	}
	defer store.Close()

	instances, listErr := store.ListInstances(context.Background(), scenarioruntime.InstanceFilter{Scenario: "alpha"})
	if listErr != nil {
		t.Fatalf("ListInstances: %v", listErr)
	}
	if len(instances) != 1 || instances[0].Status != scenarioruntime.StatusFailed {
		t.Fatalf("instances = %#v, want one failed instance", instances)
	}
	claims, claimErr := store.ListPortClaims(context.Background(), scenarioruntime.PortClaimFilter{InstanceID: instances[0].InstanceID})
	if claimErr != nil {
		t.Fatalf("ListPortClaims: %v", claimErr)
	}
	if len(claims) != 1 || claims[0].Status != scenarioruntime.ClaimStatusReleased {
		t.Fatalf("claims = %#v, want released claim after failed start", claims)
	}
}

func TestRunPhaseDetailedBootstrapsResourceDependencies(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixtureManifest(t, root, testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithDependencies(scenario.Dependencies{
			Resources: map[string]scenario.Dependency{
				"postgres": {Enabled: true, Required: true},
			},
		}),
	))

	statusCalls := 0
	runCalls := 0
	runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.resourceStatus = func(name string, fast bool) (resourcecontrol.Status, error) {
			statusCalls++
			if statusCalls == 1 {
				return resourcecontrol.Status{Resource: resources.Resource{Name: name}, Running: false, StatusCode: resourcecontrol.StatusCodeUnavailable}, nil
			}
			healthy := true
			return resourcecontrol.Status{Resource: resources.Resource{Name: name}, Running: true, Healthy: &healthy, StatusCode: resourcecontrol.StatusCodeOK}, nil
		}
		deps.runResource = func(name string, args []string, stdout, stderr io.Writer) error {
			runCalls++
			return nil
		}
	})

	result, err := runner.RunPhaseDetailed("alpha", "test", PhaseOptions{})
	if err != nil {
		t.Fatalf("RunPhaseDetailed(test): %v", err)
	}
	if result.Status != PhaseExecutionUndefined {
		t.Fatalf("result.Status = %q, want %q", result.Status, PhaseExecutionUndefined)
	}
	if runCalls != 1 {
		t.Fatalf("runCalls = %d, want 1", runCalls)
	}
	if statusCalls < 2 {
		t.Fatalf("statusCalls = %d, want at least 2", statusCalls)
	}
}

func TestEnsureResourceDependenciesWaitsForStartedResourceReadiness(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixtureManifest(t, root, testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithDependencies(scenario.Dependencies{
			Resources: map[string]scenario.Dependency{
				"postgres": {Enabled: true, Required: true},
			},
		}),
	))

	statusCalls := 0
	now := time.Unix(0, 0)
	runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.resourceStatus = func(name string, fast bool) (resourcecontrol.Status, error) {
			statusCalls++
			if statusCalls < 3 {
				healthy := false
				return resourcecontrol.Status{
					Resource:   resources.Resource{Name: name},
					Running:    true,
					Healthy:    &healthy,
					StatusCode: resourcecontrol.StatusCodeUnavailable,
					Health:     "starting",
				}, nil
			}
			healthy := true
			return resourcecontrol.Status{
				Resource:   resources.Resource{Name: name},
				Running:    true,
				Healthy:    &healthy,
				StatusCode: resourcecontrol.StatusCodeOK,
				Health:     "healthy",
			}, nil
		}
		deps.runResource = func(name string, args []string, stdout, stderr io.Writer) error {
			return nil
		}
		deps.now = func() time.Time { return now }
		deps.sleep = func(d time.Duration) { now = now.Add(d) }
	})
	item, err := scenario.Load(root, "alpha", scenario.SandboxEnv{})
	if err != nil {
		t.Fatalf("Load(alpha): %v", err)
	}

	failed, err := runner.ensureResourceDependencies(item, StartOptions{})
	if err != nil {
		t.Fatalf("ensureResourceDependencies: %v", err)
	}
	if len(failed) != 0 {
		t.Fatalf("failed resources = %#v, want none", failed)
	}
	if statusCalls != 3 {
		t.Fatalf("statusCalls = %d, want 3", statusCalls)
	}
}

func TestEnsureResourceDependenciesTryStartWaitsThenDegradesOnTimeout(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixtureManifest(t, root, testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithDependencies(scenario.Dependencies{
			Resources: map[string]scenario.Dependency{
				"qdrant": {Enabled: true, StartupPolicy: scenario.DependencyStartupPolicyTryStart},
			},
		}),
	))

	statusCalls := 0
	now := time.Unix(0, 0)
	runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.resourceStatus = func(name string, fast bool) (resourcecontrol.Status, error) {
			statusCalls++
			healthy := false
			return resourcecontrol.Status{
				Resource:   resources.Resource{Name: name},
				Running:    true,
				Healthy:    &healthy,
				StatusCode: resourcecontrol.StatusCodeUnavailable,
				Health:     "starting",
			}, nil
		}
		deps.runResource = func(name string, args []string, stdout, stderr io.Writer) error {
			return nil
		}
		deps.now = func() time.Time { return now }
		deps.sleep = func(d time.Duration) { now = now.Add(resourceDependencyReadyTimeout) }
	})
	item, err := scenario.Load(root, "alpha", scenario.SandboxEnv{})
	if err != nil {
		t.Fatalf("Load(alpha): %v", err)
	}

	failed, err := runner.ensureResourceDependencies(item, StartOptions{})
	if err != nil {
		t.Fatalf("ensureResourceDependencies: %v", err)
	}
	if len(failed) != 1 || failed[0] != "qdrant" {
		t.Fatalf("failed resources = %#v, want [qdrant]", failed)
	}
	if statusCalls < 2 {
		t.Fatalf("statusCalls = %d, want at least 2", statusCalls)
	}
}

func TestFileDependencySpecsIgnoresNonDependencyTopLevelFields(t *testing.T) {
	packageJSON := filepath.Join(t.TempDir(), "package.json")
	testpackage.WriteNodePackageManifest(t, packageJSON, testpackage.NodePackageManifest{
		Name: "fixture",
		Scripts: map[string]string{
			"build": "vite build",
		},
		Dependencies: map[string]string{
			"@local/pkg-a": "file:../pkg-a",
			"react":        "^18.0.0",
		},
		DevDependencies: map[string]string{
			"@local/pkg-b": "file:../pkg-b",
		},
		OptionalDependencies: map[string]string{
			"@local/pkg-c": "file:../pkg-c",
		},
	})

	specs, err := fileDependencySpecs(packageJSON)
	if err != nil {
		t.Fatalf("fileDependencySpecs: %v", err)
	}

	want := []string{"file:../pkg-a", "file:../pkg-b", "file:../pkg-c"}
	if len(specs) != len(want) {
		t.Fatalf("spec count = %d, want %d (%v)", len(specs), len(want), specs)
	}
	for i, spec := range specs {
		if spec != want[i] {
			t.Fatalf("spec[%d] = %q, want %q", i, spec, want[i])
		}
	}
}

func TestEnsureScenarioDatabaseUsesPostgresResourceLibs(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testresource.WritePortRegistry(t, root, nil)

	binDir := t.TempDir()
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	psqlScript := `#!/usr/bin/env bash
set -euo pipefail
state_file="` + filepath.Join(root, "bootstrap-state.txt") + `"
db_file="` + filepath.Join(root, "db-created.txt") + `"
sql=""
file=""
for ((i=1; i<=$#; i++)); do
  if [[ "${!i}" == "-c" ]]; then
    next=$((i+1))
    sql="${!next}"
  fi
  if [[ "${!i}" == "-f" ]]; then
    next=$((i+1))
    file="${!next}"
  fi
done
printf '%s\n' "$*" >> "` + filepath.Join(root, "psql.log") + `"
if [[ -n "$sql" && "$sql" == *"SELECT 1 FROM pg_database"* ]]; then
  if [[ -f "$db_file" ]]; then
    printf '1\n'
  fi
  exit 0
fi
if [[ -n "$sql" && "$sql" == *"CREATE DATABASE"* ]]; then
  printf '%s\n' "$sql" >> "` + filepath.Join(root, "create.txt") + `"
  : > "$db_file"
  exit 0
fi
if [[ -n "$sql" && "$sql" == *"CREATE TABLE IF NOT EXISTS vrooli_bootstrap_artifacts"* ]]; then
  exit 0
fi
if [[ -n "$sql" && "$sql" == *"SELECT checksum FROM vrooli_bootstrap_artifacts"* ]]; then
  compact="$(printf '%s' "$sql" | tr '\n' ' ')"
  key="$(printf '%s' "$compact" | sed -n "s/.*scenario_slug = '\\([^']*\\)'.*artifact_kind = '\\([^']*\\)'.*artifact_name = '\\([^']*\\)'.*/\\1|\\2|\\3/p")"
  if [[ -n "$key" && -f "$state_file" ]]; then
    value="$(grep "^$key|" "$state_file" | tail -n 1 | cut -d'|' -f4 || true)"
    if [[ -n "$value" ]]; then
      printf '%s\n' "$value"
    fi
  fi
  exit 0
fi
if [[ -n "$sql" && "$sql" == *"INSERT INTO vrooli_bootstrap_artifacts"* ]]; then
  compact="$(printf '%s' "$sql" | tr '\n' ' ')"
  mapfile -t quoted < <(printf '%s' "$compact" | grep -o "'[^']*'" | sed "s/'//g")
  record=""
  if [[ ${#quoted[@]} -ge 4 ]]; then
    record="${quoted[0]}|${quoted[1]}|${quoted[2]}|${quoted[3]}"
  fi
  if [[ -n "$record" ]]; then
    key="$(printf '%s' "$record" | cut -d'|' -f1-3)"
    tmp="${state_file}.tmp"
    if [[ -f "$state_file" ]]; then
      grep -v "^$key|" "$state_file" > "$tmp" || true
    else
      : > "$tmp"
    fi
    printf '%s\n' "$record" >> "$tmp"
    mv "$tmp" "$state_file"
  fi
  exit 0
fi
if [[ -n "$file" ]]; then
  basename "$file" >> "` + filepath.Join(root, "files.txt") + `"
fi
`
	testkitgo.WriteExecutable(t, filepath.Join(binDir, "psql"), psqlScript)

	scenarioPath := filepath.Join(root, "scenarios", "alpha")
	if err := os.MkdirAll(filepath.Join(scenarioPath, "initialization", "postgres"), 0o755); err != nil {
		t.Fatalf("mkdir initialization/postgres: %v", err)
	}
	testkitgo.WriteFile(t, filepath.Join(scenarioPath, "initialization", "postgres", "schema.sql"), "create table if not exists test();\n")
	testkitgo.WriteFile(t, filepath.Join(scenarioPath, "initialization", "postgres", "migration_001.sql"), "-- migration\n")

	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	item := scenario.Scenario{
		Slug: "alpha",
		Path: scenarioPath,
	}
	if err := runner.ensureScenarioDatabase(item, map[string]string{"POSTGRES_DB": "alpha_db"}, io.Discard); err != nil {
		t.Fatalf("ensureScenarioDatabase: %v", err)
	}
	if err := runner.ensureScenarioDatabase(item, map[string]string{"POSTGRES_DB": "alpha_db"}, io.Discard); err != nil {
		t.Fatalf("ensureScenarioDatabase second run: %v", err)
	}

	createData, err := os.ReadFile(filepath.Join(root, "create.txt"))
	if err != nil {
		t.Fatalf("read create.txt: %v", err)
	}
	if got := string(createData); got != "CREATE DATABASE \"alpha_db\";\n" {
		t.Fatalf("create.txt = %q", got)
	}

	schemaData, err := os.ReadFile(filepath.Join(root, "files.txt"))
	if err != nil {
		t.Fatalf("read files.txt: %v", err)
	}
	if got := string(schemaData); got != "schema.sql\nmigration_001.sql\n" {
		t.Fatalf("files.txt = %q", got)
	}
}

func TestApplyPostgresArtifactReappliesSchemaWhenChecksumChanges(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testresource.WritePortRegistry(t, root, nil)
	binDir := t.TempDir()
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	psqlScript := `#!/usr/bin/env bash
set -euo pipefail
state_file="` + filepath.Join(root, "bootstrap-state.txt") + `"
sql=""
file=""
for ((i=1; i<=$#; i++)); do
  if [[ "${!i}" == "-c" ]]; then
    next=$((i+1))
    sql="${!next}"
  fi
  if [[ "${!i}" == "-f" ]]; then
    next=$((i+1))
    file="${!next}"
  fi
done
if [[ -n "$sql" && "$sql" == *"CREATE TABLE IF NOT EXISTS vrooli_bootstrap_artifacts"* ]]; then
  exit 0
fi
if [[ -n "$sql" && "$sql" == *"SELECT checksum FROM vrooli_bootstrap_artifacts"* ]]; then
  compact="$(printf '%s' "$sql" | tr '\n' ' ')"
  key="$(printf '%s' "$compact" | sed -n "s/.*scenario_slug = '\\([^']*\\)'.*artifact_kind = '\\([^']*\\)'.*artifact_name = '\\([^']*\\)'.*/\\1|\\2|\\3/p")"
  if [[ -n "$key" && -f "$state_file" ]]; then
    value="$(grep "^$key|" "$state_file" | tail -n 1 | cut -d'|' -f4 || true)"
    if [[ -n "$value" ]]; then
      printf '%s\n' "$value"
    fi
  fi
  exit 0
fi
if [[ -n "$sql" && "$sql" == *"INSERT INTO vrooli_bootstrap_artifacts"* ]]; then
  compact="$(printf '%s' "$sql" | tr '\n' ' ')"
  mapfile -t quoted < <(printf '%s' "$compact" | grep -o "'[^']*'" | sed "s/'//g")
  record=""
  if [[ ${#quoted[@]} -ge 4 ]]; then
    record="${quoted[0]}|${quoted[1]}|${quoted[2]}|${quoted[3]}"
  fi
  key="$(printf '%s' "$record" | cut -d'|' -f1-3)"
  tmp="${state_file}.tmp"
  if [[ -f "$state_file" ]]; then
    grep -v "^$key|" "$state_file" > "$tmp" || true
  else
    : > "$tmp"
  fi
  printf '%s\n' "$record" >> "$tmp"
  mv "$tmp" "$state_file"
  exit 0
fi
if [[ -n "$file" ]]; then
  basename "$file" >> "` + filepath.Join(root, "files.txt") + `"
fi
`
	testkitgo.WriteExecutable(t, filepath.Join(binDir, "psql"), psqlScript)

	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	if err := runner.ensurePostgresBootstrapRegistry(map[string]string{}, "alpha_db", io.Discard); err != nil {
		t.Fatalf("ensurePostgresBootstrapRegistry: %v", err)
	}

	schemaPath := filepath.Join(root, "schema.sql")
	testkitgo.WriteFile(t, schemaPath, "create table if not exists alpha();\n")
	if err := runner.applyPostgresArtifact("alpha", map[string]string{}, "alpha_db", bootstrapArtifactKindSchema, schemaPath, io.Discard); err != nil {
		t.Fatalf("apply schema first pass: %v", err)
	}

	testkitgo.WriteFile(t, schemaPath, "create table if not exists alpha();\ncreate table if not exists beta();\n")
	if err := runner.applyPostgresArtifact("alpha", map[string]string{}, "alpha_db", bootstrapArtifactKindSchema, schemaPath, io.Discard); err != nil {
		t.Fatalf("apply schema second pass: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "files.txt"))
	if err != nil {
		t.Fatalf("read files.txt: %v", err)
	}
	if got := string(data); got != "schema.sql\nschema.sql\n" {
		t.Fatalf("files.txt = %q", got)
	}
}

func TestApplyPostgresArtifactRejectsChangedMigrationChecksum(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testresource.WritePortRegistry(t, root, nil)
	binDir := t.TempDir()
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	psqlScript := `#!/usr/bin/env bash
set -euo pipefail
state_file="` + filepath.Join(root, "bootstrap-state.txt") + `"
sql=""
file=""
for ((i=1; i<=$#; i++)); do
  if [[ "${!i}" == "-c" ]]; then
    next=$((i+1))
    sql="${!next}"
  fi
  if [[ "${!i}" == "-f" ]]; then
    next=$((i+1))
    file="${!next}"
  fi
done
if [[ -n "$sql" && "$sql" == *"CREATE TABLE IF NOT EXISTS vrooli_bootstrap_artifacts"* ]]; then
  exit 0
fi
if [[ -n "$sql" && "$sql" == *"SELECT checksum FROM vrooli_bootstrap_artifacts"* ]]; then
  compact="$(printf '%s' "$sql" | tr '\n' ' ')"
  key="$(printf '%s' "$compact" | sed -n "s/.*scenario_slug = '\\([^']*\\)'.*artifact_kind = '\\([^']*\\)'.*artifact_name = '\\([^']*\\)'.*/\\1|\\2|\\3/p")"
  if [[ -n "$key" && -f "$state_file" ]]; then
    value="$(grep "^$key|" "$state_file" | tail -n 1 | cut -d'|' -f4 || true)"
    if [[ -n "$value" ]]; then
      printf '%s\n' "$value"
    fi
  fi
  exit 0
fi
if [[ -n "$sql" && "$sql" == *"INSERT INTO vrooli_bootstrap_artifacts"* ]]; then
  compact="$(printf '%s' "$sql" | tr '\n' ' ')"
  mapfile -t quoted < <(printf '%s' "$compact" | grep -o "'[^']*'" | sed "s/'//g")
  record=""
  if [[ ${#quoted[@]} -ge 4 ]]; then
    record="${quoted[0]}|${quoted[1]}|${quoted[2]}|${quoted[3]}"
  fi
  printf '%s\n' "$record" >> "$state_file"
  exit 0
fi
if [[ -n "$file" ]]; then
  basename "$file" >> "` + filepath.Join(root, "files.txt") + `"
fi
`
	testkitgo.WriteExecutable(t, filepath.Join(binDir, "psql"), psqlScript)

	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	if err := runner.ensurePostgresBootstrapRegistry(map[string]string{}, "alpha_db", io.Discard); err != nil {
		t.Fatalf("ensurePostgresBootstrapRegistry: %v", err)
	}

	migrationPath := filepath.Join(root, "migration_001.sql")
	testkitgo.WriteFile(t, migrationPath, "-- first\n")
	if err := runner.applyPostgresArtifact("alpha", map[string]string{}, "alpha_db", bootstrapArtifactKindMigration, migrationPath, io.Discard); err != nil {
		t.Fatalf("apply migration first pass: %v", err)
	}

	testkitgo.WriteFile(t, migrationPath, "-- changed\n")
	err = runner.applyPostgresArtifact("alpha", map[string]string{}, "alpha_db", bootstrapArtifactKindMigration, migrationPath, io.Discard)
	if err == nil {
		t.Fatalf("expected changed migration checksum to fail")
	}
	if !strings.Contains(err.Error(), "different checksum") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunnerStartRejectsCircularDependencies(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	alpha := lifecycleFixtureManifest("alpha")
	alpha.Dependencies.Scenarios = map[string]scenario.Dependency{
		"beta": {Required: true},
	}
	writeLifecycleFixtureManifest(t, root, alpha)

	beta := lifecycleFixtureManifest("beta")
	beta.Dependencies.Scenarios = map[string]scenario.Dependency{
		"alpha": {Required: true},
	}
	writeLifecycleFixtureManifest(t, root, beta)

	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	_, err = runner.Start("alpha", StartOptions{})
	if err == nil {
		t.Fatalf("expected circular dependency to fail start")
	}
	if !strings.Contains(err.Error(), "circular scenario dependency detected") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnsureDependenciesUsesInjectedScenarioRecordReader(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testresource.WritePortRegistry(t, root, nil)

	testscenario.WriteScenarioService(t, root, "alpha", testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithDependencies(scenario.Dependencies{
			Scenarios: map[string]scenario.Dependency{
				"beta": {Required: true},
			},
		}),
	))
	testscenario.WriteScenarioService(t, root, "beta", testscenario.ScenarioServiceManifest("beta"))

	var readCalls []string
	runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.readScenarioRecords = func(gotHome, name string) ([]process.Record, error) {
			if gotHome != home {
				t.Fatalf("readScenarioRecords home = %q, want %q", gotHome, home)
			}
			readCalls = append(readCalls, name)
			return []process.Record{{
				PID:       os.Getpid(),
				PGID:      os.Getpid(),
				Scenario:  name,
				Step:      "develop",
				Status:    "running",
				StartedAt: time.Now().UTC(),
			}}, nil
		}
	})

	item, err := scenario.Load(root, "alpha", scenario.SandboxEnv{})
	if err != nil {
		t.Fatalf("Load(alpha): %v", err)
	}

	failed, err := runner.ensureDependencies(item, StartOptions{}, map[string]struct{}{}, []string{"alpha"})
	if err != nil {
		t.Fatalf("ensureDependencies: %v", err)
	}
	if len(failed) != 0 {
		t.Fatalf("failed dependencies = %#v", failed)
	}
	if got := strings.Join(readCalls, ","); got != "beta" {
		t.Fatalf("readScenarioRecords calls = %q, want beta", got)
	}
}

func TestRunnerStartBootstrapsTransitiveResourceDependencies(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testresource.WritePortRegistry(t, root, nil)

	testscenario.WriteScenarioService(t, root, "alpha", testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithDependencies(scenario.Dependencies{
			Scenarios: map[string]scenario.Dependency{
				"beta": {Required: true},
			},
		}),
	))
	testscenario.WriteScenarioService(t, root, "beta", testscenario.ScenarioServiceManifest(
		"beta",
		testscenario.WithDependencies(scenario.Dependencies{
			Resources: map[string]scenario.Dependency{
				"postgres": {Enabled: true, Required: true},
			},
		}),
	))

	var startedResources []string
	runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.resourceStatus = func(name string, fast bool) (resourcecontrol.Status, error) {
			for _, started := range startedResources {
				if started == name {
					healthy := true
					return resourcecontrol.Status{
						Resource:   resources.Resource{Name: name},
						Running:    true,
						Healthy:    &healthy,
						StatusCode: resourcecontrol.StatusCodeOK,
						Health:     "healthy",
					}, nil
				}
			}
			return resourcecontrol.Status{
				Resource:   resources.Resource{Name: name},
				Running:    false,
				StatusCode: resourcecontrol.StatusCodeUnavailable,
			}, nil
		}
		deps.runResource = func(name string, args []string, stdout, stderr io.Writer) error {
			startedResources = append(startedResources, name)
			return nil
		}
	})

	result, err := runner.Start("alpha", StartOptions{})
	if err != nil {
		t.Fatalf("Start(alpha): %v", err)
	}
	if result.Health != "running" {
		t.Fatalf("result.Health = %q, want running", result.Health)
	}
	if got := strings.Join(startedResources, ","); got != "postgres" {
		t.Fatalf("startedResources = %q, want postgres", got)
	}
}

func TestStepConditionsMetSupportsFilesystemEnvAndJSONChecks(t *testing.T) {
	root := "/fixture"
	probe := newFakeHostProbe()
	probe.addDir(root, time.Unix(10, 0))
	probe.addDir(filepath.Join(root, "data"), time.Unix(10, 0))
	probe.addFile(filepath.Join(root, "config.json"), time.Unix(10, 0), 0o644, []byte("{\"services\":[{\"name\":\"api\"}]}\n"))
	probe.lookPath["sh"] = "/bin/sh"
	probe.env["EXTERNAL_ONLY"] = "set"

	item := scenario.Scenario{
		Slug: "alpha",
		Path: root,
		Manifest: scenario.ServiceManifest{
			Dependencies: scenario.Dependencies{
				Resources: map[string]scenario.Dependency{
					"postgres": {Enabled: true},
				},
			},
		},
	}

	condition := &scenario.Condition{
		FileExists:      "config.json",
		DirectoryExists: "data",
		JSONPathExists:  "config.json:services.0.name",
		ResourceEnabled: "postgres",
		CommandExists:   "sh",
		BinaryExists:    "sh",
		EnvVarSet:       "EXTERNAL_ONLY",
	}

	ok, reason, err := stepConditionsMetWithDeps(item, condition, nil, probe.deps())
	if err != nil {
		t.Fatalf("stepConditionsMet: %v", err)
	}
	if !ok || reason != "" {
		t.Fatalf("expected condition to pass, ok=%v reason=%q", ok, reason)
	}

	ok, reason, err = stepConditionsMetWithDeps(item, &scenario.Condition{Always: "false"}, nil, probe.deps())
	if err != nil {
		t.Fatalf("stepConditionsMet(always=false): %v", err)
	}
	if ok || reason != "step disabled by always=false" {
		t.Fatalf("always=false => ok=%v reason=%q", ok, reason)
	}

	ok, reason, err = stepConditionsMetWithDeps(item, &scenario.Condition{FileNotExists: "config.json"}, nil, probe.deps())
	if err != nil {
		t.Fatalf("stepConditionsMet(file_not_exists): %v", err)
	}
	if ok || !strings.Contains(reason, "must not exist") {
		t.Fatalf("file_not_exists => ok=%v reason=%q", ok, reason)
	}

	found, err := jsonPathExistsWithDeps(filepath.Join(root, "config.json"), "config.json:services.0.name", probe.deps())
	if err != nil {
		t.Fatalf("jsonPathExists: %v", err)
	}
	if !found {
		t.Fatalf("expected JSON path to exist")
	}

	found, err = jsonPathExistsWithDeps(filepath.Join(root, "config.json"), "config.json:services.1.name", probe.deps())
	if err != nil {
		t.Fatalf("jsonPathExists missing path: %v", err)
	}
	if found {
		t.Fatalf("expected missing JSON path to return false")
	}
}

func TestStepConditionsMetRejectsDisabledResourceAndInvalidJSON(t *testing.T) {
	root := "/fixture"
	probe := newFakeHostProbe()
	probe.addDir(root, time.Unix(10, 0))
	probe.addFile(filepath.Join(root, "broken.json"), time.Unix(10, 0), 0o644, []byte("{broken"))

	item := scenario.Scenario{
		Slug: "alpha",
		Path: root,
		Manifest: scenario.ServiceManifest{
			Dependencies: scenario.Dependencies{
				Resources: map[string]scenario.Dependency{
					"postgres": {Enabled: false},
				},
			},
		},
	}

	ok, reason, err := stepConditionsMetWithDeps(item, &scenario.Condition{ResourceEnabled: "postgres"}, nil, probe.deps())
	if err != nil {
		t.Fatalf("stepConditionsMet(disabled resource): %v", err)
	}
	if ok || !strings.Contains(reason, `resource "postgres" is disabled`) {
		t.Fatalf("disabled resource => ok=%v reason=%q", ok, reason)
	}

	if _, _, err := stepConditionsMetWithDeps(item, &scenario.Condition{JSONPathExists: "broken.json:services.0.name"}, nil, probe.deps()); err == nil {
		t.Fatalf("expected invalid JSON path source to fail")
	}
}

func TestSetupNeededIgnoresCLIChecksForRuntimeFreshness(t *testing.T) {
	root := t.TempDir()
	runner := &Runner{}

	item := scenario.Scenario{
		Slug: "alpha",
		Path: filepath.Join(root, "scenarios", "alpha"),
		Manifest: scenario.ServiceManifest{
			Service: scenario.ServiceMetadata{Name: "alpha"},
			CLI: &scenario.CLIConfig{
				Enabled: true,
				Command: "fixture-cli",
				Adapter: scenario.CLIAdapterConfig{
					Kind:      "go_module",
					ModuleDir: "cli",
				},
			},
			Lifecycle: scenario.Lifecycle{
				Setup: scenario.Phase{
					Condition: &scenario.Condition{
						Checks: []scenario.ConditionCheck{
							{Type: "cli", Command: "fixture-cli"},
						},
					},
				},
			},
		},
	}

	needed, reasons, err := runner.SetupNeeded(item, false)
	if err != nil {
		t.Fatalf("SetupNeeded: %v", err)
	}
	if needed {
		t.Fatalf("expected runtime setup to ignore CLI freshness, reasons=%v", reasons)
	}
	if len(reasons) != 0 {
		t.Fatalf("expected no setup reasons, got %v", reasons)
	}
}

func TestUIBundleNeedsSetupTracksBundleFreshness(t *testing.T) {
	appRoot := "/app"
	probe := newFakeHostProbe()
	sourceDir := filepath.Join(appRoot, "ui", "src")
	bundlePath := filepath.Join(appRoot, "ui", "dist", "index.html")
	packageJSON := filepath.Join(appRoot, "ui", "package.json")
	sourcePath := filepath.Join(sourceDir, "main.tsx")
	older := time.Unix(100, 0)
	newer := time.Unix(200, 0)
	future := time.Unix(300, 0)
	probe.addDir(appRoot, older)
	probe.addDir(filepath.Join(appRoot, "ui"), older)
	probe.addDir(sourceDir, older)
	probe.addDir(filepath.Dir(bundlePath), newer)
	probe.addFile(sourcePath, older, 0o644, []byte("export default 'hi';\n"))
	probe.addFile(bundlePath, newer, 0o644, []byte("<html></html>\n"))
	probe.addFile(packageJSON, older, 0o644, []byte("{\"name\":\"fixture-ui\",\"dependencies\":{}}\n"))

	needed, reason, err := uiBundleNeedsSetupWithDeps(appRoot, scenario.ConditionCheck{}, probe.deps())
	if err != nil {
		t.Fatalf("uiBundleNeedsSetup fresh bundle: %v", err)
	}
	if needed {
		t.Fatalf("expected fresh bundle to satisfy setup, reason=%q", reason)
	}

	probe.addFile(sourcePath, future, 0o644, []byte("export default 'hi';\n"))

	needed, reason, err = uiBundleNeedsSetupWithDeps(appRoot, scenario.ConditionCheck{}, probe.deps())
	if err != nil {
		t.Fatalf("uiBundleNeedsSetup stale bundle: %v", err)
	}
	if !needed || !strings.Contains(reason, "UI bundle outdated") {
		t.Fatalf("stale bundle => needed=%v reason=%q", needed, reason)
	}
}

func TestEvaluateSetupCheckSupportsFilesystemStateChecks(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testresource.WritePortRegistry(t, root, nil)

	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	item := scenario.Scenario{Slug: "alpha", Path: root}

	needed, reason, err := runner.evaluateSetupCheck(item, scenario.ConditionCheck{Type: "resources", Populated: true})
	if err != nil {
		t.Fatalf("evaluateSetupCheck(resources missing): %v", err)
	}
	if !needed || reason != "Resources not populated" {
		t.Fatalf("resources missing => needed=%v reason=%q", needed, reason)
	}
	if err := os.MkdirAll(projectstate.SetupStateDir(root), 0o755); err != nil {
		t.Fatalf("mkdir setup state: %v", err)
	}
	testkitgo.WriteFile(t, projectstate.ResourcesPopulatedPath(root), "ok\n")
	needed, _, err = runner.evaluateSetupCheck(item, scenario.ConditionCheck{Type: "resources", Populated: true})
	if err != nil {
		t.Fatalf("evaluateSetupCheck(resources ready): %v", err)
	}
	if needed {
		t.Fatalf("expected populated resources marker to satisfy setup")
	}

	uiDir := filepath.Join(root, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatalf("mkdir ui: %v", err)
	}
	testpackage.WriteNodePackageManifest(t, filepath.Join(uiDir, "package.json"), testpackage.NodePackageManifest{
		Name: "fixture",
	})
	needed, reason, err = runner.evaluateSetupCheck(item, scenario.ConditionCheck{Type: "dependencies", Paths: []string{"ui/package.json"}})
	if err != nil {
		t.Fatalf("evaluateSetupCheck(dependencies missing): %v", err)
	}
	if !needed || reason != "Dependencies not installed" {
		t.Fatalf("dependencies missing => needed=%v reason=%q", needed, reason)
	}
	if err := os.MkdirAll(filepath.Join(uiDir, "node_modules"), 0o755); err != nil {
		t.Fatalf("mkdir node_modules: %v", err)
	}
	needed, _, err = runner.evaluateSetupCheck(item, scenario.ConditionCheck{Type: "dependencies", Paths: []string{"ui/package.json"}})
	if err != nil {
		t.Fatalf("evaluateSetupCheck(dependencies ready): %v", err)
	}
	if needed {
		t.Fatalf("expected node_modules to satisfy dependency check")
	}

	cacheDir := filepath.Join(root, "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}
	needed, reason, err = runner.evaluateSetupCheck(item, scenario.ConditionCheck{Type: "data", Path: "cache"})
	if err != nil {
		t.Fatalf("evaluateSetupCheck(data empty): %v", err)
	}
	if !needed || reason != "Data directory missing" {
		t.Fatalf("data empty => needed=%v reason=%q", needed, reason)
	}
	testkitgo.WriteFile(t, filepath.Join(cacheDir, "seed.txt"), "ok\n")
	needed, _, err = runner.evaluateSetupCheck(item, scenario.ConditionCheck{Type: "data", Path: "cache"})
	if err != nil {
		t.Fatalf("evaluateSetupCheck(data populated): %v", err)
	}
	if needed {
		t.Fatalf("expected populated data dir to satisfy setup")
	}

	needed, reason, err = runner.evaluateSetupCheck(item, scenario.ConditionCheck{Type: "files", Paths: []string{"config/app.yaml"}})
	if err != nil {
		t.Fatalf("evaluateSetupCheck(files missing): %v", err)
	}
	if !needed || reason != "Required files missing" {
		t.Fatalf("files missing => needed=%v reason=%q", needed, reason)
	}
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	testkitgo.WriteFile(t, filepath.Join(root, "config", "app.yaml"), "ok\n")
	needed, _, err = runner.evaluateSetupCheck(item, scenario.ConditionCheck{Type: "files", Paths: []string{"config/app.yaml"}})
	if err != nil {
		t.Fatalf("evaluateSetupCheck(files ready): %v", err)
	}
	if needed {
		t.Fatalf("expected present file to satisfy setup")
	}

	needed, reason, err = runner.evaluateSetupCheck(item, scenario.ConditionCheck{Type: "directories", Targets: []string{"runtime"}})
	if err != nil {
		t.Fatalf("evaluateSetupCheck(directories missing): %v", err)
	}
	if !needed || reason != "Missing directories" {
		t.Fatalf("directories missing => needed=%v reason=%q", needed, reason)
	}
	if err := os.MkdirAll(filepath.Join(root, "runtime"), 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	needed, _, err = runner.evaluateSetupCheck(item, scenario.ConditionCheck{Type: "directories", Targets: []string{"runtime"}})
	if err != nil {
		t.Fatalf("evaluateSetupCheck(directories ready): %v", err)
	}
	if needed {
		t.Fatalf("expected present directory to satisfy setup")
	}
}

func TestResourceAndDependencyChecksCoverMarkersAndToolchains(t *testing.T) {
	root := t.TempDir()

	if !resourcesNeedSetup(root, scenario.ConditionCheck{Resources: []string{"postgres", "redis"}}) {
		t.Fatalf("expected missing resource markers to require setup")
	}
	if err := os.MkdirAll(projectstate.SetupStateDir(root), 0o755); err != nil {
		t.Fatalf("mkdir setup state: %v", err)
	}
	testkitgo.WriteFile(t, projectstate.ResourcePopulatedPath(root, "postgres"), "ok\n")
	if !resourcesNeedSetup(root, scenario.ConditionCheck{Resources: []string{"postgres", "redis"}}) {
		t.Fatalf("expected missing redis marker to keep setup required")
	}
	testkitgo.WriteFile(t, projectstate.ResourcePopulatedPath(root, "redis"), "ok\n")
	if resourcesNeedSetup(root, scenario.ConditionCheck{Resources: []string{"postgres", "redis"}}) {
		t.Fatalf("expected all resource markers to satisfy setup")
	}

	goDir := filepath.Join(root, "go-worker")
	if err := os.MkdirAll(goDir, 0o755); err != nil {
		t.Fatalf("mkdir go worker: %v", err)
	}
	testkitgo.WriteFile(t, filepath.Join(goDir, "go.mod"), "module fixture\n")
	if !dependenciesNeedSetup(root, scenario.ConditionCheck{Paths: []string{"go-worker/go.mod"}}) {
		t.Fatalf("expected missing go.sum/vendor to require setup")
	}
	if err := os.MkdirAll(filepath.Join(goDir, "vendor"), 0o755); err != nil {
		t.Fatalf("mkdir vendor: %v", err)
	}
	if dependenciesNeedSetup(root, scenario.ConditionCheck{Paths: []string{"go-worker/go.mod"}}) {
		t.Fatalf("expected vendor fallback to satisfy Go dependency check")
	}

	pythonDir := filepath.Join(root, "python-worker")
	if err := os.MkdirAll(pythonDir, 0o755); err != nil {
		t.Fatalf("mkdir python worker: %v", err)
	}
	testkitgo.WriteFile(t, filepath.Join(pythonDir, "requirements.txt"), "pytest\n")
	if !dependenciesNeedSetup(root, scenario.ConditionCheck{Paths: []string{"python-worker/requirements.txt"}}) {
		t.Fatalf("expected missing Python virtualenv to require setup")
	}
	if err := os.MkdirAll(filepath.Join(pythonDir, ".venv"), 0o755); err != nil {
		t.Fatalf("mkdir .venv: %v", err)
	}
	if dependenciesNeedSetup(root, scenario.ConditionCheck{Paths: []string{"python-worker/requirements.txt"}}) {
		t.Fatalf("expected .venv to satisfy Python dependency check")
	}

	rustDir := filepath.Join(root, "rust-worker")
	if err := os.MkdirAll(rustDir, 0o755); err != nil {
		t.Fatalf("mkdir rust worker: %v", err)
	}
	testkitgo.WriteFile(t, filepath.Join(rustDir, "Cargo.toml"), "[package]\nname=\"fixture\"\nversion=\"0.1.0\"\n")
	if !dependenciesNeedSetup(root, scenario.ConditionCheck{Paths: []string{"rust-worker/Cargo.toml"}}) {
		t.Fatalf("expected missing Rust target dir to require setup")
	}
	if err := os.MkdirAll(filepath.Join(rustDir, "target"), 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if dependenciesNeedSetup(root, scenario.ConditionCheck{Paths: []string{"rust-worker/Cargo.toml"}}) {
		t.Fatalf("expected target dir to satisfy Rust dependency check")
	}

	if !dependenciesNeedSetup(root, scenario.ConditionCheck{Paths: []string{"config/missing.yaml"}}) {
		t.Fatalf("expected missing generic dependency path to require setup")
	}
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	testkitgo.WriteFile(t, filepath.Join(root, "config", "missing.yaml"), "ok\n")
	if dependenciesNeedSetup(root, scenario.ConditionCheck{Paths: []string{"config/missing.yaml"}}) {
		t.Fatalf("expected present generic dependency path to satisfy setup")
	}
}

func TestRunnerLoadScenarioSupportsCustomPathAndMissingScenario(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testresource.WritePortRegistry(t, root, nil)

	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	customPath := t.TempDir()
	servicePath := filepath.Join(customPath, ".vrooli", "service.json")
	testscenario.WriteScenarioServiceAtPath(t, customPath, scenario.ServiceManifest{
		Version: "1.0.0",
		Service: scenario.ServiceMetadata{
			DisplayName: "Custom fixture",
		},
	})

	item, err := runner.loadScenario("", customPath)
	if err != nil {
		t.Fatalf("loadScenario(custom): %v", err)
	}
	if item.Slug != filepath.Base(customPath) {
		t.Fatalf("slug = %q, want %q", item.Slug, filepath.Base(customPath))
	}
	if item.Path != customPath {
		t.Fatalf("path = %q, want %q", item.Path, customPath)
	}
	if item.ServicePath != servicePath {
		t.Fatalf("service path = %q, want %q", item.ServicePath, servicePath)
	}

	if _, err := runner.loadScenario("missing", ""); err == nil || !strings.Contains(err.Error(), `scenario "missing" not found`) {
		t.Fatalf("missing scenario error = %v", err)
	}
}

func TestWaitForHealthHonorsExplicitTimeoutAndDegradedState(t *testing.T) {
	runner := &Runner{}

	status, err := runner.WaitForHealth(scenario.Scenario{Slug: "alpha"}, nil)
	if err != nil {
		t.Fatalf("WaitForHealth(no checks): %v", err)
	}
	if status != "running" {
		t.Fatalf("status = %q, want running", status)
	}

	degradedItem := scenario.Scenario{
		Slug: "beta",
		Manifest: scenario.ServiceManifest{
			Ports: map[string]scenario.Port{
				"api": {EnvVar: "API_PORT"},
			},
			Lifecycle: scenario.Lifecycle{
				Health: &scenario.HealthConfig{
					Checks: []scenario.HealthCheck{
						{
							Name:     "api",
							Type:     "http",
							Target:   "http://127.0.0.1:${API_PORT}/health",
							Critical: false,
							Timeout:  25,
						},
					},
					Timeout:  25,
					Interval: 1,
				},
			},
		},
	}

	start := time.Now()
	status, err = runner.WaitForHealth(degradedItem, map[string]string{"API_PORT": "1"})
	if err != nil {
		t.Fatalf("WaitForHealth(degraded): %v", err)
	}
	if status != "degraded" {
		t.Fatalf("status = %q, want degraded", status)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("WaitForHealth ignored explicit timeout, elapsed=%s", elapsed)
	}

	unhealthyItem := scenario.Scenario{
		Slug: "gamma",
		Manifest: scenario.ServiceManifest{
			Lifecycle: scenario.Lifecycle{
				Health: &scenario.HealthConfig{
					Checks: []scenario.HealthCheck{
						{
							Name:     "api",
							Type:     "unsupported",
							Critical: true,
						},
					},
					Timeout:  25,
					Interval: 1,
				},
			},
		},
	}

	start = time.Now()
	status, err = runner.WaitForHealth(unhealthyItem, nil)
	if err == nil {
		t.Fatalf("expected unhealthy health checks to fail")
	}
	if status != "unhealthy" {
		t.Fatalf("status = %q, want unhealthy", status)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("WaitForHealth unhealthy timeout took too long: %s", elapsed)
	}
}

func TestWaitForHealthUsesInjectedClockAndSleep(t *testing.T) {
	base := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)
	now := base
	sleepCalls := 0
	runner := &Runner{
		deps: lifecycleDeps{
			now:   func() time.Time { return now },
			sleep: func(duration time.Duration) { sleepCalls++; now = now.Add(duration) },
		},
	}

	item := scenario.Scenario{
		Slug: "delta",
		Manifest: scenario.ServiceManifest{
			Lifecycle: scenario.Lifecycle{
				Health: &scenario.HealthConfig{
					Checks: []scenario.HealthCheck{{
						Name:     "api",
						Type:     "unsupported",
						Critical: true,
					}},
					StartupGracePeriod: 25,
					Timeout:            50,
					Interval:           10,
				},
			},
		},
	}

	status, err := runner.WaitForHealth(item, nil)
	if err == nil {
		t.Fatal("expected unhealthy health checks to fail")
	}
	if status != "unhealthy" {
		t.Fatalf("status = %q, want unhealthy", status)
	}
	if sleepCalls == 0 {
		t.Fatal("expected WaitForHealth to use injected sleep")
	}
	if !now.After(base) {
		t.Fatalf("expected injected clock to advance, now=%s base=%s", now, base)
	}
}

func TestKillOrphansOnPortsUsesInjectedListenerAndSignalSeams(t *testing.T) {
	listenCalls := 0
	var signaled []string
	runner := &Runner{
		deps: lifecycleDeps{
			listeningPIDs: func(port int) ([]int, error) {
				listenCalls++
				switch listenCalls {
				case 1:
					return []int{101, 202}, nil
				case 2:
					return []int{303}, nil
				default:
					return nil, nil
				}
			},
			signalPID: func(pid int, force bool) error {
				signaled = append(signaled, fmt.Sprintf("%d:%t", pid, force))
				return nil
			},
			sleep: func(time.Duration) {},
		},
	}

	if err := runner.killOrphansOnPorts(map[int]struct{}{18080: {}}); err != nil {
		t.Fatalf("killOrphansOnPorts: %v", err)
	}

	want := []string{"101:false", "202:false", "303:true"}
	if len(signaled) != len(want) {
		t.Fatalf("signaled = %v, want %v", signaled, want)
	}
	for i := range want {
		if signaled[i] != want[i] {
			t.Fatalf("signaled[%d] = %q, want %q", i, signaled[i], want[i])
		}
	}
}

// orphanScenarioItem builds a fixture with two fixed ports and the three
// shared listener PIDs used by the start-time cleanup tests below.
func orphanScenarioItem() scenario.Scenario {
	return scenario.Scenario{
		Slug: "alpha",
		Manifest: scenario.ServiceManifest{
			Ports: map[string]scenario.Port{
				"ui":  {EnvVar: "UI_PORT", Port: intPtr(36235)},
				"api": {EnvVar: "API_PORT", Port: intPtr(18800)},
			},
		},
	}
}

func orphanLifecycleDeps(signaled *[]string) lifecycleDeps {
	return lifecycleDeps{
		inspectPort: func(port int) (network.PortInspection, error) {
			switch port {
			case 36235:
				return network.PortInspection{
					Inspection: network.ListenerInspection{Available: true, Tool: "stub"},
					Listeners: []network.PortListener{
						{PID: 101},
						{PID: 202},
					},
				}, nil
			case 18800:
				return network.PortInspection{
					Inspection: network.ListenerInspection{Available: true, Tool: "stub"},
					Listeners:  []network.PortListener{{PID: 303}},
				}, nil
			default:
				return network.PortInspection{}, nil
			}
		},
		readProcessEnv: func(pid int) (map[string]string, error) {
			switch pid {
			case 101:
				return map[string]string{
					"VROOLI_LIFECYCLE_MANAGED": "true",
					"VROOLI_SCENARIO":          "alpha",
				}, nil
			case 202:
				return map[string]string{
					"VROOLI_LIFECYCLE_MANAGED": "true",
					"VROOLI_SCENARIO":          "beta",
				}, nil
			case 303:
				return map[string]string{}, nil
			default:
				return map[string]string{}, nil
			}
		},
		signalPID: func(pid int, force bool) error {
			*signaled = append(*signaled, fmt.Sprintf("%d:%t", pid, force))
			return nil
		},
		isPIDRunning: func(pid int) bool { return pid == 101 || pid == 303 },
		sleep:        func(time.Duration) {},
	}
}

// TestCleanupFixedPortOrphans_Aggressive covers the default behavior: kill
// same-scenario env-matched listeners AND env-less orphans on canonical
// ports, but never listeners owned by a different scenario.
func TestCleanupFixedPortOrphans_Aggressive(t *testing.T) {
	item := orphanScenarioItem()
	var signaled []string
	runner := &Runner{deps: orphanLifecycleDeps(&signaled)}

	if err := runner.cleanupFixedPortOrphans(item, nil); err != nil {
		t.Fatalf("cleanupFixedPortOrphans: %v", err)
	}

	// PID 101 (alpha-owned) AND PID 303 (env-less orphan) both killed.
	// PID 202 (beta-owned) is left alone to preserve cross-scenario safety.
	signaledSet := map[string]bool{}
	for _, s := range signaled {
		signaledSet[s] = true
	}
	if !signaledSet["101:false"] || !signaledSet["101:true"] {
		t.Errorf("env-matched PID 101 not signaled (SIGTERM+SIGKILL): %v", signaled)
	}
	if !signaledSet["303:false"] || !signaledSet["303:true"] {
		t.Errorf("env-less orphan PID 303 should be killed via aggressive fallback: %v", signaled)
	}
	if signaledSet["202:false"] || signaledSet["202:true"] {
		t.Errorf("beta-scenario PID 202 must not be killed: %v", signaled)
	}
}

// TestCleanupFixedPortOrphans_StrictPreservesOldBehavior proves that
// VROOLI_PORT_ORPHAN_STRICT=true reverts to the conservative "kill only
// env-matched listeners" behavior for debugging or manual inspection.
func TestCleanupFixedPortOrphans_StrictPreservesOldBehavior(t *testing.T) {
	t.Setenv("VROOLI_PORT_ORPHAN_STRICT", "true")

	item := orphanScenarioItem()
	var signaled []string
	runner := &Runner{deps: orphanLifecycleDeps(&signaled)}

	if err := runner.cleanupFixedPortOrphans(item, nil); err != nil {
		t.Fatalf("cleanupFixedPortOrphans: %v", err)
	}

	want := []string{"101:false", "101:true"}
	if len(signaled) != len(want) {
		t.Fatalf("signaled = %v, want %v", signaled, want)
	}
	for i := range want {
		if signaled[i] != want[i] {
			t.Fatalf("signaled[%d] = %q, want %q", i, signaled[i], want[i])
		}
	}
}

func TestCleanupFixedPortOrphansSkipsTrackedRuntimeOwner(t *testing.T) {
	item := scenario.Scenario{
		Slug: "alpha",
		Manifest: scenario.ServiceManifest{
			Ports: map[string]scenario.Port{
				"ui": {EnvVar: "UI_PORT", Port: intPtr(36235)},
			},
		},
	}

	runner := &Runner{
		deps: lifecycleDeps{
			inspectPort: func(port int) (network.PortInspection, error) {
				t.Fatal("inspectPort should not be called for tracked runtime owner")
				return network.PortInspection{}, nil
			},
		},
	}

	if err := runner.cleanupFixedPortOrphans(item, []process.Record{{PID: os.Getpid(), Port: 36235}}); err != nil {
		t.Fatalf("cleanupFixedPortOrphans: %v", err)
	}
}

func TestVerifyPortsReleased_HappyPath(t *testing.T) {
	calls := 0
	runner := &Runner{
		deps: lifecycleDeps{
			listeningPIDs: func(port int) ([]int, error) {
				calls++
				return nil, nil
			},
			sleep: func(time.Duration) {},
		},
	}
	if err := runner.verifyPortsReleased("alpha", map[int]struct{}{21234: {}}); err != nil {
		t.Fatalf("verifyPortsReleased: %v", err)
	}
	if calls != 1 {
		t.Errorf("listeningPIDs called %d times, want 1 (single poll, empty result)", calls)
	}
}

func TestVerifyPortsReleased_StillBoundFails(t *testing.T) {
	runner := &Runner{
		deps: lifecycleDeps{
			listeningPIDs: func(port int) ([]int, error) {
				return []int{99}, nil
			},
			sleep: func(time.Duration) {},
		},
	}
	err := runner.verifyPortsReleased("alpha", map[int]struct{}{21234: {}})
	if err == nil {
		t.Fatal("expected error when port stays bound")
	}
	if !strings.Contains(err.Error(), "still bound") {
		t.Errorf("error missing 'still bound': %v", err)
	}
}

func TestVerifyPortsReleased_EventuallyFreesSucceeds(t *testing.T) {
	polls := 0
	runner := &Runner{
		deps: lifecycleDeps{
			listeningPIDs: func(port int) ([]int, error) {
				polls++
				if polls < 3 {
					return []int{99}, nil
				}
				return nil, nil
			},
			sleep: func(time.Duration) {},
		},
	}
	if err := runner.verifyPortsReleased("alpha", map[int]struct{}{21234: {}}); err != nil {
		t.Fatalf("verifyPortsReleased: %v", err)
	}
	if polls < 3 {
		t.Errorf("expected at least 3 polls, got %d", polls)
	}
}

func TestVerifyPortsReleased_EmptyPortSetSkips(t *testing.T) {
	runner := &Runner{deps: lifecycleDeps{}}
	if err := runner.verifyPortsReleased("alpha", nil); err != nil {
		t.Errorf("empty portsToCheck should be no-op: %v", err)
	}
}

func TestConfirmFixedPortLocks_UpdatesLockToRealPID(t *testing.T) {
	root := t.TempDir()
	testresource.WritePortRegistry(t, root, nil)
	home := t.TempDir()
	manager, err := ports.NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	// Seed a lock with the lifecycle runner's PID (simulating what
	// allocatePortDefinition currently does).
	if err := manager.WriteLock(21234, "swarm-manager", os.Getpid()); err != nil {
		t.Fatalf("WriteLock: %v", err)
	}

	item := scenario.Scenario{
		Slug: "swarm-manager",
		Manifest: scenario.ServiceManifest{
			Ports: map[string]scenario.Port{
				"ui": {EnvVar: "UI_PORT", Port: intPtr(21234)},
			},
		},
	}

	runner := &Runner{
		Ports: manager,
		deps: lifecycleDeps{
			inspectPort: func(port int) (network.PortInspection, error) {
				if port != 21234 {
					return network.PortInspection{}, nil
				}
				return network.PortInspection{
					Inspection: network.ListenerInspection{Available: true, Tool: "stub"},
					Listeners:  []network.PortListener{{PID: 4242}},
				}, nil
			},
		},
	}
	runner.confirmFixedPortLocks(item)

	lock, exists, err := manager.ReadLock(21234)
	if err != nil || !exists {
		t.Fatalf("lock missing: %v exists=%v", err, exists)
	}
	if lock.PID != 4242 {
		t.Errorf("lock PID = %d, want 4242 (real listener)", lock.PID)
	}
}

func TestRuntimePortsAndStrictHealthUseRecordedMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	port := server.Listener.Addr().(*net.TCPAddr).Port
	item := scenario.Scenario{
		Slug: "alpha",
		Manifest: scenario.ServiceManifest{
			Ports: map[string]scenario.Port{
				"api": {EnvVar: "API_PORT"},
			},
			Lifecycle: scenario.Lifecycle{
				Health: &scenario.HealthConfig{
					Checks: []scenario.HealthCheck{
						{
							Name:     "api",
							Type:     "http",
							Target:   "http://127.0.0.1:${API_PORT}/",
							Critical: true,
							Timeout:  1000,
						},
					},
				},
			},
		},
	}

	runner := &Runner{}
	records := []process.Record{{
		PID:  os.Getpid(),
		Step: "start-api",
		Port: port,
	}}

	ports := runner.runtimePorts(item.Manifest, records)
	if ports["API_PORT"] != port {
		t.Fatalf("API_PORT = %d, want %d", ports["API_PORT"], port)
	}
	if !runner.isScenarioHealthyStrict(item, records) {
		t.Fatalf("expected live record and healthy endpoint to pass strict health")
	}
	if runner.isScenarioHealthyStrict(item, nil) {
		t.Fatalf("expected empty runtime to fail strict health")
	}
}

func TestExecutePhaseAppendsTestArgsAndWarnsOnStopFailure(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testresource.WritePortRegistry(t, root, nil)

	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	item := scenario.Scenario{
		Slug: "alpha",
		Path: root,
		Manifest: scenario.ServiceManifest{
			Lifecycle: scenario.Lifecycle{
				Test: scenario.Phase{
					Steps: []scenario.PhaseStep{
						{
							Name: "skip-me",
							Run:  "printf 'nope\\n' > skipped.txt",
							Condition: &scenario.Condition{
								Always: "false",
							},
						},
						{
							Name: "write-args",
							Run:  "printf '%s\\n' > args.txt",
						},
					},
				},
				Stop: scenario.Phase{
					Steps: []scenario.PhaseStep{
						{
							Name: "failing-stop",
							Run:  "exit 7",
						},
					},
				},
			},
		},
	}

	var testLog bytes.Buffer
	if err := runner.ExecutePhase(item, "test", nil, []string{"phase-a", "two words"}, &testLog); err != nil {
		t.Fatalf("ExecutePhase(test): %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "skipped.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected skipped step to avoid writing file, stat err=%v", err)
	}
	argsData, err := os.ReadFile(filepath.Join(root, "args.txt"))
	if err != nil {
		t.Fatalf("read args.txt: %v", err)
	}
	if got := string(argsData); got != "phase-a\ntwo words\n" {
		t.Fatalf("args.txt = %q", got)
	}
	if !strings.Contains(testLog.String(), "Skipping skip-me - step disabled by always=false") {
		t.Fatalf("expected skip log, got %q", testLog.String())
	}

	var stopLog bytes.Buffer
	if err := runner.ExecutePhase(item, "stop", nil, nil, &stopLog); err != nil {
		t.Fatalf("ExecutePhase(stop): %v", err)
	}
	if !strings.Contains(stopLog.String(), "[WARNING] Stop step completed with non-zero exit: failing-stop") {
		t.Fatalf("expected stop warning log, got %q", stopLog.String())
	}
}

func TestInjectTestGenieAutoStart(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "basic execute command",
			in:   "test-genie execute alpha --preset comprehensive",
			want: "test-genie --auto-start execute alpha --preset comprehensive",
		},
		{
			name: "preserves env assignments",
			in:   "TEST_MODE=1 test-genie execute alpha",
			want: "TEST_MODE=1 test-genie --auto-start execute alpha",
		},
		{
			name: "leaves explicit auto-start unchanged",
			in:   "test-genie --auto-start execute alpha",
			want: "test-genie --auto-start execute alpha",
		},
		{
			name: "ignores unrelated commands",
			in:   "echo test-genie execute alpha",
			want: "echo test-genie execute alpha",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := injectTestGenieAutoStart(tc.in); got != tc.want {
				t.Fatalf("injectTestGenieAutoStart(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestEvaluateSetupCheckRejectsUnknownCheckTypes(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testresource.WritePortRegistry(t, root, nil)

	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	needed, reason, err := runner.evaluateSetupCheck(scenario.Scenario{Slug: "alpha", Path: root}, scenario.ConditionCheck{Type: "custom"})
	if err == nil {
		t.Fatal("expected unsupported check type to fail")
	}
	if needed || reason != "" {
		t.Fatalf("needed/reason = %v/%q", needed, reason)
	}
	if !strings.Contains(err.Error(), `unsupported setup condition type "custom"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLocalReplacePathsSupportsInlineAndBlockForms(t *testing.T) {
	goModPath := filepath.Join(t.TempDir(), "go.mod")
	data := `module fixture

go 1.23.0

replace github.com/example/alpha => ../alpha

replace (
	github.com/example/beta => ../beta
	github.com/example/gamma => ../gamma
)
`
	testkitgo.WriteFile(t, goModPath, data)

	paths, err := localReplacePaths(goModPath)
	if err != nil {
		t.Fatalf("localReplacePaths: %v", err)
	}
	if got := strings.Join(paths, ","); got != "../alpha,../beta,../gamma" {
		t.Fatalf("paths = %q", got)
	}
}

func TestRunnerStartEnforcesScenarioHostRequirements(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixture(t, root, "alpha")

	var captured hostreqrun.Options
	calls := 0
	runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.enforceHostRequirements = func(opts hostreqrun.Options) (vrooliruntime.Report, error) {
			captured = opts
			calls++
			return vrooliruntime.Report{}, nil
		}
		// Short-circuit the actual phase execution by forcing a "healthy" early
		// return path is not possible here; instead keep the runResource/status
		// defaults — phases will try to execute the fixture's steps. Since this
		// test cares only about whether enforcement fires BEFORE phase work, we
		// override the phase runner via the condition: the fixture declares an
		// api binary build that runs bash. To avoid actually booting, we stub
		// resource status/runResource (no resource deps exist for alpha, so
		// these are effectively unused).
		deps.resourceStatus = func(name string, fast bool) (resourcecontrol.Status, error) {
			return resourcecontrol.Status{Resource: resources.Resource{Name: name}, Running: true, StatusCode: resourcecontrol.StatusCodeOK}, nil
		}
		deps.runResource = func(name string, args []string, stdout, stderr io.Writer) error {
			return nil
		}
	})

	// Trigger enforcement by invoking the scenario start path indirectly:
	// loadScenario + call the unexported enforcer. This keeps the test
	// hermetic — we do not need to bring up the full phase runtime, but we
	// still verify the scope passed to the hook.
	item, err := scenario.Load(root, "alpha", scenario.SandboxEnv{})
	if err != nil {
		t.Fatalf("Load(alpha): %v", err)
	}
	if err := runner.enforceScenarioHostRequirements(item); err != nil {
		t.Fatalf("enforceScenarioHostRequirements: %v", err)
	}
	if calls != 1 {
		t.Fatalf("enforce calls = %d, want 1", calls)
	}
	if captured.Scenarios != "" {
		t.Fatalf("Scenarios = %q, want empty when scenario path is provided", captured.Scenarios)
	}
	if len(captured.ScenarioPaths) != 1 || captured.ScenarioPaths[0] != item.Path {
		t.Fatalf("ScenarioPaths = %#v, want [%q]", captured.ScenarioPaths, item.Path)
	}
	if captured.Resources != "none" {
		t.Fatalf("Resources = %q, want none", captured.Resources)
	}
	if captured.Environment != "development" {
		t.Fatalf("Environment = %q, want development", captured.Environment)
	}
	if captured.When != "develop" {
		t.Fatalf("When = %q, want develop", captured.When)
	}
	if !captured.AutoInstall {
		t.Fatal("AutoInstall must be true")
	}
	if captured.Label != "scenario:alpha" {
		t.Fatalf("Label = %q, want scenario:alpha", captured.Label)
	}
}

func TestRunnerStartPropagatesHostRequirementErrors(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixture(t, root, "alpha")

	runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.enforceHostRequirements = func(_ hostreqrun.Options) (vrooliruntime.Report, error) {
			return vrooliruntime.Report{}, errors.New("docker missing")
		}
	})
	item, err := scenario.Load(root, "alpha", scenario.SandboxEnv{})
	if err != nil {
		t.Fatalf("Load(alpha): %v", err)
	}
	if err := runner.enforceScenarioHostRequirements(item); err == nil || !strings.Contains(err.Error(), "docker missing") {
		t.Fatalf("expected docker missing error, got %v", err)
	}
}

func TestRunnerEnforcesResourceHostRequirementsBeforeStart(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixtureManifest(t, root, testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithDependencies(scenario.Dependencies{
			Resources: map[string]scenario.Dependency{
				"postgres": {Enabled: true, Required: true},
			},
		}),
	))

	callOrder := []string{}
	var capturedResource hostreqrun.Options
	statusCalls := 0
	now := time.Unix(0, 0)
	runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.enforceHostRequirements = func(opts hostreqrun.Options) (vrooliruntime.Report, error) {
			callOrder = append(callOrder, "enforce:"+opts.Label)
			if opts.Resources == "postgres" {
				capturedResource = opts
			}
			return vrooliruntime.Report{}, nil
		}
		deps.resourceStatus = func(name string, fast bool) (resourcecontrol.Status, error) {
			statusCalls++
			// First call (before start) reports unavailable so start runs; the
			// wait-for-ready loop then sees a ready resource on subsequent polls.
			if statusCalls == 1 {
				return resourcecontrol.Status{Resource: resources.Resource{Name: name}, Running: false, StatusCode: resourcecontrol.StatusCodeUnavailable}, nil
			}
			return resourcecontrol.Status{Resource: resources.Resource{Name: name}, Running: true, StatusCode: resourcecontrol.StatusCodeOK}, nil
		}
		deps.runResource = func(name string, args []string, stdout, stderr io.Writer) error {
			callOrder = append(callOrder, "run:"+name+":"+strings.Join(args, ","))
			return nil
		}
		deps.now = func() time.Time { return now }
		deps.sleep = func(d time.Duration) { now = now.Add(d) }
	})

	item, err := scenario.Load(root, "alpha", scenario.SandboxEnv{})
	if err != nil {
		t.Fatalf("Load(alpha): %v", err)
	}

	_, err = runner.ensureResourceDependencies(item, StartOptions{})
	if err != nil {
		t.Fatalf("ensureResourceDependencies: %v", err)
	}

	if capturedResource.Resources != "postgres" {
		t.Fatalf("expected enforce call for postgres, got %+v", capturedResource)
	}
	if capturedResource.Environment != "development" {
		t.Fatalf("Environment = %q, want development", capturedResource.Environment)
	}
	if capturedResource.Scenarios != "none" {
		t.Fatalf("Scenarios = %q, want none", capturedResource.Scenarios)
	}
	if capturedResource.Label != "resource:postgres" {
		t.Fatalf("Label = %q, want resource:postgres", capturedResource.Label)
	}

	// Enforcement must happen BEFORE the run:postgres:start call.
	enforceIdx, runIdx := -1, -1
	for i, entry := range callOrder {
		if entry == "enforce:resource:postgres" && enforceIdx == -1 {
			enforceIdx = i
		}
		if entry == "run:postgres:start" && runIdx == -1 {
			runIdx = i
		}
	}
	if enforceIdx == -1 || runIdx == -1 {
		t.Fatalf("missing expected calls in order: %v", callOrder)
	}
	if enforceIdx >= runIdx {
		t.Fatalf("enforce (idx=%d) must precede run (idx=%d) in %v", enforceIdx, runIdx, callOrder)
	}
}

func TestRunnerHostRequirementEnforcementUsesRunnerEnvironment(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixture(t, root, "alpha")

	var captured hostreqrun.Options
	runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.enforceHostRequirements = func(opts hostreqrun.Options) (vrooliruntime.Report, error) {
			captured = opts
			return vrooliruntime.Report{}, nil
		}
	})
	runner.Environment = "production"

	item, err := scenario.Load(root, "alpha", scenario.SandboxEnv{})
	if err != nil {
		t.Fatalf("Load(alpha): %v", err)
	}
	if err := runner.enforceScenarioHostRequirements(item); err != nil {
		t.Fatalf("enforceScenarioHostRequirements: %v", err)
	}
	if captured.Environment != "production" {
		t.Fatalf("Environment = %q, want production", captured.Environment)
	}
}
