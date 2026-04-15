package lifecycle

import (
	"bytes"
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

	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/projectstate"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/scenario"
	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
	testpackage "github.com/vrooli/vrooli/packages/testkit-go/packagefixture"
	testresource "github.com/vrooli/vrooli/packages/testkit-go/resourcefixture"
	testscenario "github.com/vrooli/vrooli/packages/testkit-go/scenariofixture"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=9 | LAST: 2026-04-13

func newLifecycleRunnerForTest(t *testing.T, root, home string, mutate func(*lifecycleDeps), logger ...*slog.Logger) *Runner {
	t.Helper()
	deps := defaultLifecycleDeps()
	if mutate != nil {
		mutate(&deps)
	}
	runner, err := newRunnerWithDeps(root, home, io.Discard, io.Discard, deps, logger...)
	if err != nil {
		t.Fatalf("newRunnerWithDeps: %v", err)
	}
	return runner
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
	return fakeFileInfo{name: d.name, node: d.node}, nil
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

	result, err := runner.ExecutePhaseDetailed(item, "setup", map[string]string{}, nil, io.Discard)
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
	testresource.WriteResourceManifest(t, root, "postgres", manifestpkg.ResourceManifest{
		Name:            "postgres",
		Driver:          "docker-service",
		PortabilityTier: "full",
		Runtime: manifestpkg.ResourceRuntime{
			Image:         "postgres:16-alpine",
			ContainerName: "vrooli-postgres-main",
		},
	})
	binDir := t.TempDir()
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	dockerScript := `#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >> "` + filepath.Join(root, "psql.log") + `"
if [[ "$*" == *"SELECT 1 FROM pg_database"* ]]; then
  exit 0
fi
for ((i=1; i<=$#; i++)); do
  if [[ "${!i}" == "-c" ]]; then
    next=$((i+1))
    printf '%s\n' "${!next}" >> "` + filepath.Join(root, "create.txt") + `"
  fi
done
if [[ "$*" == *" -i "* ]]; then
  cat >> "` + filepath.Join(root, "files.txt") + `"
  printf '\n--EOF--\n' >> "` + filepath.Join(root, "files.txt") + `"
fi
`
	testkitgo.WriteExecutable(t, filepath.Join(binDir, "docker"), dockerScript)

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
	if got := string(schemaData); got != "create table if not exists test();\n\n--EOF--\n-- migration\n\n--EOF--\n" {
		t.Fatalf("files.txt = %q", got)
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

func TestCLINeedsSetupDetectsMissingAndStaleBinary(t *testing.T) {
	appRoot := "/app"
	probe := newFakeHostProbe()
	probe.addDir(appRoot, time.Unix(10, 0))
	item := scenario.Scenario{
		Slug: "alpha",
		Path: appRoot,
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
		},
	}

	needed, reason, err := cliNeedsSetupWithDeps(item, scenario.ConditionCheck{Command: "fixture-cli"}, probe.deps())
	if err != nil {
		t.Fatalf("cliNeedsSetup missing binary: %v", err)
	}
	if !needed || reason != "CLI not installed: fixture-cli" {
		t.Fatalf("missing binary => needed=%v reason=%q", needed, reason)
	}

	cliSourceDir := filepath.Join(appRoot, "cli")
	sourcePath := filepath.Join(cliSourceDir, "main.go")
	cliPath := "/bin/fixture-cli"
	old := time.Unix(100, 0)
	now := time.Unix(200, 0)
	future := time.Unix(300, 0)
	probe.addDir(cliSourceDir, old)
	probe.addFile(sourcePath, old, 0o644, []byte("package main\n"))
	probe.addFile(cliPath, now, 0o755, []byte("#!/usr/bin/env bash\nexit 0\n"))
	probe.lookPath["fixture-cli"] = cliPath

	needed, reason, err = cliNeedsSetupWithDeps(item, scenario.ConditionCheck{Command: "fixture-cli"}, probe.deps())
	if err != nil {
		t.Fatalf("cliNeedsSetup fresh binary: %v", err)
	}
	if needed {
		t.Fatalf("expected fresh CLI binary to satisfy setup, reason=%q", reason)
	}

	probe.addFile(sourcePath, future, 0o644, []byte("package main\n"))

	needed, reason, err = cliNeedsSetupWithDeps(item, scenario.ConditionCheck{Command: "fixture-cli"}, probe.deps())
	if err != nil {
		t.Fatalf("cliNeedsSetup stale binary: %v", err)
	}
	if !needed || reason != "CLI not installed: fixture-cli" {
		t.Fatalf("stale binary => needed=%v reason=%q", needed, reason)
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
