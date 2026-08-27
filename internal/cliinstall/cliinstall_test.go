package cliinstall

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/artifactlease"
	"github.com/vrooli/vrooli/internal/artifactledger"
	"github.com/vrooli/vrooli/internal/shell/shelltest"

	"github.com/vrooli/cli-core/cliutil"
	platform "github.com/vrooli/platform-go"
	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
	testresource "github.com/vrooli/vrooli/internal/resources/resourcestest"
	"github.com/vrooli/vrooli/internal/scenario"
	testscenario "github.com/vrooli/vrooli/internal/scenario/scenariotest"
)

// mustManager constructs a Manager and fails the test on error. NewManager now
// resolves the install dir from the runtime_home contract, which can error.
func mustManager(t *testing.T, root, home string) *Manager {
	t.Helper()
	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager(%q, %q): %v", root, home, err)
	}
	return manager
}

func TestAtomicInstallWaitsForRootRebuildLock(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	destination := filepath.Join(root, "vrooli")
	if err := os.WriteFile(source, []byte("new binary"), 0o755); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.WriteFile(destination, []byte("old binary"), 0o755); err != nil {
		t.Fatalf("write destination: %v", err)
	}

	lockFile, err := os.OpenFile(destination+".lock", os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatalf("open install lock: %v", err)
	}
	release, err := platform.LockFile(lockFile, false)
	if err != nil {
		_ = lockFile.Close()
		t.Fatalf("lock install lock: %v", err)
	}
	locked := true
	t.Cleanup(func() {
		if locked {
			release()
			_ = lockFile.Close()
		}
	})

	result := make(chan error, 1)
	go func() { result <- AtomicInstall(source, destination) }()

	select {
	case err := <-result:
		t.Fatalf("AtomicInstall ignored the shared install lock: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	release()
	locked = false
	if err := lockFile.Close(); err != nil {
		t.Fatalf("close install lock: %v", err)
	}
	if err := <-result; err != nil {
		t.Fatalf("AtomicInstall after lock release: %v", err)
	}
}

// computeScenarioCLIFingerprint is a test-only helper that exercises the
// canonical FreshnessSpec path end-to-end.
func computeScenarioCLIFingerprint(item InstallableCLI) (string, error) {
	spec, err := item.FreshnessSpec()
	if err != nil {
		return "", err
	}
	return cliutil.ComputeFreshnessFingerprint(spec)
}

func computeResourceCLIFingerprint(item InstallableCLI) (string, error) {
	return computeScenarioCLIFingerprint(item)
}

type stubInstaller struct {
	calls []installCall
}

type installCall struct {
	item       InstallableCLI
	installDir string
}

func writeGoScenarioCLIManifest(t *testing.T, root, name string) {
	t.Helper()
	testscenario.WriteScenarioService(t, root, name, testscenario.ScenarioServiceManifest(
		name,
		testscenario.WithCLI(&scenario.CLIConfig{
			Enabled: true,
			Command: name,
			Adapter: scenario.CLIAdapterConfig{
				Kind:      "go_module",
				ModuleDir: "cli",
			},
			SourceBuild: &scenario.CLISourceBuildConfig{Kind: "go_module"},
			Invoke:      scenario.CLIInvokeConfig{Kind: "installed_command", Command: name},
			Freshness:   &scenario.CLIFreshnessCheck{Inputs: []string{"cli/**", ".vrooli/service.json"}},
		}),
	))
}

func writeGoResourceCLIManifest(t *testing.T, root, name string) {
	t.Helper()
	testresource.WriteResourceManifest(t, root, name, testresource.ResourceManifest(
		name,
		testresource.WithResourceDriver("external-cli"),
		testresource.WithResourceBinary("bash"),
	))
}

func writeDisabledResourceCLIManifest(t *testing.T, root, name string) {
	t.Helper()
	manifest := testresource.ResourceManifest(name)
	manifest.CLI = &scenario.CLIConfig{Enabled: false}
	testresource.WriteResourceManifest(t, root, name, manifest)
}

func (s *stubInstaller) Install(_ context.Context, item InstallableCLI, installDir string) error {
	s.calls = append(s.calls, installCall{item: item, installDir: installDir})
	return nil
}

func TestDiscoverScenarioCLIs(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	fixture.WriteScenarioStub(t, "alpha")
	fixture.WriteScenarioStub(t, "beta")
	fixture.WriteScenarioStub(t, "gamma")
	writeGoScenarioCLIManifest(t, fixture.Root, "alpha")
	writeGoScenarioCLIManifest(t, fixture.Root, "beta")
	testscenario.WriteScenarioCLIGoMod(t, fixture.Root, "alpha", "alpha/cli")
	testscenario.WriteScenarioCLIGoMod(t, fixture.Root, "beta", "beta/cli")

	manager := mustManager(t, fixture.Root, fixture.Home)
	items, err := manager.DiscoverScenarioCLIs()
	if err != nil {
		t.Fatalf("DiscoverScenarioCLIs: %v", err)
	}

	got := []string{items[0].Name, items[1].Name}
	if !reflect.DeepEqual(got, []string{"alpha", "beta"}) {
		t.Fatalf("scenario discovery = %v, want %v", got, []string{"alpha", "beta"})
	}
	if items[0].BinaryName != "alpha" {
		t.Fatalf("scenario binary name = %q, want %q", items[0].BinaryName, "alpha")
	}
	if items[0].ManifestPath != filepath.Join(fixture.Root, "scenarios", "alpha", ".vrooli", "service.json") {
		t.Fatalf("scenario manifest path = %q", items[0].ManifestPath)
	}
}

func TestDiscoverScenarioCLIReportIncludesFailuresWithoutAborting(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	fixture.WriteScenarioStub(t, "alpha")
	fixture.WriteScenarioStub(t, "broken")
	writeGoScenarioCLIManifest(t, fixture.Root, "alpha")
	testscenario.WriteScenarioCLIGoMod(t, fixture.Root, "alpha", "alpha/cli")
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", "broken", ".vrooli", "service.json"), `{
  "service": {"name": "broken"},
  "cli": {
    "enabled": true,
    "command": "broken",
    "adapter": {"kind": "go_module", "module_path": "cli"}
  }
}`)

	report, err := mustManager(t, fixture.Root, fixture.Home).DiscoverScenarioCLIReport()
	if err != nil {
		t.Fatalf("DiscoverScenarioCLIReport: %v", err)
	}
	if len(report.Items) != 1 || report.Items[0].Name != "alpha" {
		t.Fatalf("items = %#v", report.Items)
	}
	if len(report.Failures) != 1 || report.Failures[0].Name != "broken" {
		t.Fatalf("failures = %#v", report.Failures)
	}
}

func TestDiscoverScenarioCLIDoesNotInferShellScriptFromLayoutWithoutManifest(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	fixture.WriteScenarioStub(t, "alpha")
	testscenario.WriteScenarioService(t, fixture.Root, "alpha", testscenario.ScenarioServiceManifest("alpha"))
	testscenario.WriteScenarioCLIGoMod(t, fixture.Root, "alpha", "alpha/cli")

	manager := mustManager(t, fixture.Root, fixture.Home)
	_, err := manager.DiscoverScenarioCLI("alpha")
	if err == nil {
		t.Fatal("expected missing CLI manifest error, got nil")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected not-exist error when shell layout exists without manifest, got %v", err)
	}
}

func TestDiscoverEnabledResourceCLIs(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	writeGoResourceCLIManifest(t, fixture.Root, "postgres")
	writeDisabledResourceCLIManifest(t, fixture.Root, "redis")
	writeGoResourceCLIManifest(t, fixture.Root, "ollama")
	testresource.WriteResourceCLIGoMod(t, fixture.Root, "postgres", "resource-postgres/cli")
	testresource.WriteResourceCLIGoMod(t, fixture.Root, "ollama", "resource-ollama/cli")
	testscenario.WriteProjectService(t, fixture.Root, testscenario.ProjectServiceManifest(
		testscenario.WithDependencies(mapProjectResources(
			map[string]bool{
				"postgres": true,
				"redis":    true,
				"ollama":   false,
			},
		)),
	))

	manager := mustManager(t, fixture.Root, fixture.Home)
	items, err := manager.DiscoverEnabledResourceCLIs()
	if err != nil {
		t.Fatalf("DiscoverEnabledResourceCLIs: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("enabled resource CLI count = %d, want 1", len(items))
	}
	if got := items[0].BinaryName; got != "resource-postgres" {
		t.Fatalf("resource binary name = %q, want %q", got, "resource-postgres")
	}
	if got := items[0].ManifestPath; got != filepath.Join(fixture.Root, "resources", "postgres", "resource.json") {
		t.Fatalf("resource manifest path = %q", got)
	}
}

func TestDiscoverResourceCLIsReturnsAllInstallableModules(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	writeGoResourceCLIManifest(t, fixture.Root, "postgres")
	writeGoResourceCLIManifest(t, fixture.Root, "redis")
	testresource.WriteResourceCLIGoMod(t, fixture.Root, "postgres", "resource-postgres/cli")
	testresource.WriteResourceCLIGoMod(t, fixture.Root, "redis", "resource-redis/cli")

	manager := mustManager(t, fixture.Root, fixture.Home)
	items, err := manager.DiscoverResourceCLIs()
	if err != nil {
		t.Fatalf("DiscoverResourceCLIs: %v", err)
	}

	got := []string{items[0].BinaryName, items[1].BinaryName}
	if !reflect.DeepEqual(got, []string{"resource-postgres", "resource-redis"}) {
		t.Fatalf("resource CLI binaries = %v, want %v", got, []string{"resource-postgres", "resource-redis"})
	}
}

func TestDiscoverResourceCLIDoesNotInferShellScriptFromLayoutWithoutEnabledManifest(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	writeDisabledResourceCLIManifest(t, fixture.Root, "postgres")
	testkitgo.WriteRelativeExecutable(t, fixture.Root, filepath.Join("resources", "postgres", "cli", "resource-postgres"), shelltest.BashShebang()+"exit 0\n")
	testkitgo.WriteRelativeExecutable(t, fixture.Root, filepath.Join("resources", "postgres", "cli", "install.sh"), shelltest.BashShebang()+"exit 0\n")

	manager := mustManager(t, fixture.Root, fixture.Home)
	_, err := manager.DiscoverResourceCLI("postgres")
	if err == nil {
		t.Fatal("expected missing CLI manifest error, got nil")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected not-exist error when shell layout exists without enabled manifest, got %v", err)
	}
}

func TestDiscoverScenarioCLIsReturnsUnexpectedErrors(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	fixture.WriteScenarioStub(t, "alpha")
	if err := os.MkdirAll(filepath.Join(fixture.Root, "scenarios", "alpha", "cli", "go.mod"), 0o755); err != nil {
		t.Fatalf("mkdir invalid go.mod path: %v", err)
	}
	writeGoScenarioCLIManifest(t, fixture.Root, "alpha")

	manager := mustManager(t, fixture.Root, fixture.Home)
	_, err := manager.DiscoverScenarioCLIs()
	if err == nil {
		t.Fatal("expected discovery error, got nil")
	}
	if !strings.Contains(err.Error(), "is a directory") {
		t.Fatalf("unexpected error = %v", err)
	}
}

func TestInstallAllScenarioCLIsInvokesInstaller(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	fixture.WriteScenarioStub(t, "alpha")
	fixture.WriteScenarioStub(t, "beta")
	writeGoScenarioCLIManifest(t, fixture.Root, "alpha")
	writeGoScenarioCLIManifest(t, fixture.Root, "beta")
	testscenario.WriteScenarioCLIGoMod(t, fixture.Root, "alpha", "alpha/cli")
	testscenario.WriteScenarioCLIGoMod(t, fixture.Root, "beta", "beta/cli")

	installer := &stubInstaller{}
	manager := mustManager(t, fixture.Root, fixture.Home)
	manager.Installer = installer

	if err := manager.InstallAllScenarioCLIs(); err != nil {
		t.Fatalf("InstallAllScenarioCLIs: %v", err)
	}
	if len(installer.calls) != 2 {
		t.Fatalf("install calls = %d, want 2", len(installer.calls))
	}
	if installer.calls[0].installDir != filepath.Join(fixture.Home, ".vrooli", "bin") {
		t.Fatalf("install dir = %q, want %q", installer.calls[0].installDir, filepath.Join(fixture.Home, ".vrooli", "bin"))
	}
}

func TestEnsureScenarioCLIInstallsWhenMissing(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	fixture.WriteScenarioStub(t, "alpha")
	writeGoScenarioCLIManifest(t, fixture.Root, "alpha")
	testscenario.WriteScenarioCLIGoMod(t, fixture.Root, "alpha", "alpha/cli")

	installer := &stubInstaller{}
	manager := mustManager(t, fixture.Root, fixture.Home)
	manager.Installer = installer

	if err := manager.EnsureScenarioCLI("alpha"); err != nil {
		t.Fatalf("EnsureScenarioCLI: %v", err)
	}
	if len(installer.calls) != 1 {
		t.Fatalf("install calls = %d, want 1", len(installer.calls))
	}
}

func TestInspectScenarioCLIInstallLocationReportsPathMismatch(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	fixture.WriteScenarioStub(t, "alpha")
	writeGoScenarioCLIManifest(t, fixture.Root, "alpha")
	testscenario.WriteScenarioCLIGoMod(t, fixture.Root, "alpha", "alpha/cli")

	manager := mustManager(t, fixture.Root, fixture.Home)
	canonical := filepath.Join(fixture.Home, ".vrooli", "bin", "alpha")
	if err := os.MkdirAll(filepath.Dir(canonical), 0o755); err != nil {
		t.Fatalf("mkdir install dir: %v", err)
	}
	if err := os.WriteFile(canonical, []byte("installed"), 0o755); err != nil {
		t.Fatalf("write canonical binary: %v", err)
	}

	other := testkitgo.WriteRelativeExecutable(t, fixture.Home, filepath.Join(".local", "bin", "alpha"), shelltest.BashShebang()+"exit 0\n")
	status, err := manager.InspectScenarioCLIInstallLocation("alpha", func(string) (string, error) {
		return other, nil
	})
	if err != nil {
		t.Fatalf("InspectScenarioCLIInstallLocation: %v", err)
	}
	if !status.CanonicalExists {
		t.Fatal("expected canonical install to exist")
	}
	if !status.PathMismatch() {
		t.Fatalf("expected path mismatch, got %+v", status)
	}
	if status.CanonicalPath != canonical {
		t.Fatalf("canonical path = %q, want %q", status.CanonicalPath, canonical)
	}
}

func TestInspectScenarioCLIInstallLocationHandlesCommandNotFound(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	fixture.WriteScenarioStub(t, "alpha")
	writeGoScenarioCLIManifest(t, fixture.Root, "alpha")
	testscenario.WriteScenarioCLIGoMod(t, fixture.Root, "alpha", "alpha/cli")

	manager := mustManager(t, fixture.Root, fixture.Home)
	status, err := manager.InspectScenarioCLIInstallLocation("alpha", func(string) (string, error) {
		return "", exec.ErrNotFound
	})
	if err != nil {
		t.Fatalf("InspectScenarioCLIInstallLocation: %v", err)
	}
	if status.Resolved {
		t.Fatalf("expected unresolved status, got %+v", status)
	}
	if status.PathMismatch() {
		t.Fatalf("unexpected path mismatch for unresolved command: %+v", status)
	}
}

func TestEnsureResourceCLISkipsWhenInstalled(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	writeGoResourceCLIManifest(t, fixture.Root, "postgres")
	testresource.WriteResourceCLIGoMod(t, fixture.Root, "postgres", "resource-postgres/cli")

	installer := &stubInstaller{}
	manager := mustManager(t, fixture.Root, fixture.Home)
	manager.Installer = installer

	binaryPath := filepath.Join(fixture.Home, ".vrooli", "bin", "resource-postgres")
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		t.Fatalf("mkdir install dir: %v", err)
	}
	if err := os.WriteFile(binaryPath, validNativeExecutableHeader(), 0o755); err != nil {
		t.Fatalf("write installed binary: %v", err)
	}
	item, err := manager.DiscoverResourceCLI("postgres")
	if err != nil {
		t.Fatalf("DiscoverResourceCLI: %v", err)
	}
	fingerprint, err := computeResourceCLIFingerprint(item)
	if err != nil {
		t.Fatalf("compute fingerprint: %v", err)
	}
	writeInstallMetadataFixture(t, binaryPath+".build.meta", InstallMetadata{
		Fingerprint: fingerprint,
	})

	if err := manager.EnsureResourceCLI("postgres"); err != nil {
		t.Fatalf("EnsureResourceCLI: %v", err)
	}
	if len(installer.calls) != 0 {
		t.Fatalf("install calls = %d, want 0", len(installer.calls))
	}
}

func TestEnsureResourceCLIReinstallsCorruptCurrentBinary(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	writeGoResourceCLIManifest(t, fixture.Root, "postgres")
	testresource.WriteResourceCLIGoMod(t, fixture.Root, "postgres", "resource-postgres/cli")

	installer := &stubInstaller{}
	manager := mustManager(t, fixture.Root, fixture.Home)
	manager.Installer = installer

	binaryPath := filepath.Join(fixture.Home, ".vrooli", "bin", "resource-postgres")
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		t.Fatalf("mkdir install dir: %v", err)
	}
	if err := os.WriteFile(binaryPath, []byte{0, 0, 0, 0, 0, 0}, 0o755); err != nil {
		t.Fatalf("write corrupt installed binary: %v", err)
	}
	item, err := manager.DiscoverResourceCLI("postgres")
	if err != nil {
		t.Fatalf("DiscoverResourceCLI: %v", err)
	}
	fingerprint, err := computeResourceCLIFingerprint(item)
	if err != nil {
		t.Fatalf("compute fingerprint: %v", err)
	}
	writeInstallMetadataFixture(t, binaryPath+".build.meta", InstallMetadata{
		Fingerprint: fingerprint,
	})

	if err := manager.EnsureResourceCLI("postgres"); err != nil {
		t.Fatalf("EnsureResourceCLI: %v", err)
	}
	if len(installer.calls) != 1 {
		t.Fatalf("install calls = %d, want 1", len(installer.calls))
	}
}

func TestEnsureResourceCLIInstallsWhenMetadataMissing(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	writeGoResourceCLIManifest(t, fixture.Root, "postgres")
	testresource.WriteResourceCLIGoMod(t, fixture.Root, "postgres", "resource-postgres/cli")

	installer := &stubInstaller{}
	manager := mustManager(t, fixture.Root, fixture.Home)
	manager.Installer = installer

	binaryPath := filepath.Join(fixture.Home, ".vrooli", "bin", "resource-postgres")
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		t.Fatalf("mkdir install dir: %v", err)
	}
	if err := os.WriteFile(binaryPath, []byte("installed"), 0o755); err != nil {
		t.Fatalf("write installed binary: %v", err)
	}

	if err := manager.EnsureResourceCLI("postgres"); err != nil {
		t.Fatalf("EnsureResourceCLI: %v", err)
	}
	if len(installer.calls) != 1 {
		t.Fatalf("install calls = %d, want 1", len(installer.calls))
	}
}

func TestEnsureResourceCLIReinstallsWhenFingerprintStale(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	writeGoResourceCLIManifest(t, fixture.Root, "postgres")
	testresource.WriteResourceCLIGoMod(t, fixture.Root, "postgres", "resource-postgres/cli")

	installer := &stubInstaller{}
	manager := mustManager(t, fixture.Root, fixture.Home)
	manager.Installer = installer

	binaryPath := filepath.Join(fixture.Home, ".vrooli", "bin", "resource-postgres")
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		t.Fatalf("mkdir install dir: %v", err)
	}
	if err := os.WriteFile(binaryPath, []byte("installed"), 0o755); err != nil {
		t.Fatalf("write installed binary: %v", err)
	}
	writeInstallMetadataFixture(t, binaryPath+".build.meta", InstallMetadata{
		Fingerprint: "stale-fingerprint",
	})

	if err := manager.EnsureResourceCLI("postgres"); err != nil {
		t.Fatalf("EnsureResourceCLI: %v", err)
	}
	if len(installer.calls) != 1 {
		t.Fatalf("install calls = %d, want 1", len(installer.calls))
	}
}

func TestRepoEnabledResourceCLIsAreAllDiscoverable(t *testing.T) {
	root := testkitgo.ProjectRoot(t)
	manager := mustManager(t, root, t.TempDir())

	items, err := manager.DiscoverEnabledResourceCLIs()
	if err != nil {
		t.Fatalf("DiscoverEnabledResourceCLIs: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, ".vrooli", "service.json"))
	if err != nil {
		t.Fatalf("read service.json: %v", err)
	}
	var payload struct {
		Dependencies struct {
			Resources map[string]struct {
				Enabled bool `json:"enabled"`
			} `json:"resources"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal service.json: %v", err)
	}

	want := make([]string, 0, len(payload.Dependencies.Resources))
	for name, entry := range payload.Dependencies.Resources {
		if entry.Enabled {
			want = append(want, name)
		}
	}
	sort.Strings(want)

	got := make([]string, 0, len(items))
	for _, item := range items {
		got = append(got, item.Name)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("enabled resource CLI discovery = %v, want %v", got, want)
	}
}

func TestInstallResourceCLIWithGoInstallerCreatesBinaryAndMetadata(t *testing.T) {
	root := testkitgo.ProjectRoot(t)
	manager := mustManager(t, root, t.TempDir())
	if err := manager.InstallResourceCLI("postgres"); err != nil {
		t.Fatalf("InstallResourceCLI: %v", err)
	}

	item, err := manager.DiscoverResourceCLI("postgres")
	if err != nil {
		t.Fatalf("DiscoverResourceCLI: %v", err)
	}
	if _, err := os.Stat(manager.InstalledBinaryPath(item)); err != nil {
		t.Fatalf("installed binary missing: %v", err)
	}
	if _, err := os.Stat(manager.InstalledManifestPath(item)); err != nil {
		t.Fatalf("installed manifest missing: %v", err)
	}
	meta, ok, err := manager.readInstallMetadata(item)
	if err != nil {
		t.Fatalf("readInstallMetadata: %v", err)
	}
	if !ok {
		t.Fatal("expected install metadata")
	}
	if strings.TrimSpace(meta.Fingerprint) == "" {
		t.Fatalf("metadata fingerprint = %q", meta.Fingerprint)
	}
}

func TestComputeScenarioCLIFingerprintIncludesManifestForGoModule(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	fixture.WriteScenarioStub(t, "alpha")
	testscenario.WriteScenarioCLIGoMod(t, fixture.Root, "alpha", "alpha/cli")
	testscenario.WriteScenarioService(t, fixture.Root, "alpha", testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithCLI(&scenario.CLIConfig{
			Enabled: true,
			Command: "alpha",
			Adapter: scenario.CLIAdapterConfig{
				Kind:      "go_module",
				ModuleDir: "cli",
			},
			Freshness: &scenario.CLIFreshnessCheck{
				Inputs: []string{"cli/**", ".vrooli/service.json"},
			},
		}),
	))

	item, err := mustManager(t, fixture.Root, t.TempDir()).DiscoverScenarioCLI("alpha")
	if err != nil {
		t.Fatalf("DiscoverScenarioCLI: %v", err)
	}
	before, err := computeScenarioCLIFingerprint(item)
	if err != nil {
		t.Fatalf("computeScenarioCLIFingerprint before: %v", err)
	}

	servicePath := filepath.Join(fixture.Root, "scenarios", "alpha", ".vrooli", "service.json")
	data, err := os.ReadFile(servicePath)
	if err != nil {
		t.Fatalf("read service manifest: %v", err)
	}
	data = []byte(strings.Replace(string(data), "\"alpha\"", "\"alpha-updated\"", 1))
	if err := os.WriteFile(servicePath, data, 0o644); err != nil {
		t.Fatalf("write service manifest: %v", err)
	}

	after, err := computeScenarioCLIFingerprint(item)
	if err != nil {
		t.Fatalf("computeScenarioCLIFingerprint after: %v", err)
	}
	if before == after {
		t.Fatal("expected service.json change to affect go-module CLI fingerprint")
	}
}

func TestComputeResourceCLIFingerprintIncludesManifestForGoModule(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	fixture.WriteResourceStub(t, "alpha")
	testresource.WriteResourceCLIGoMod(t, fixture.Root, "alpha", "alpha/cli")
	manifest := testresource.ResourceManifest("alpha",
		testresource.WithResourceDriver("external-cli"),
		testresource.WithResourceBinary("alpha"),
	)
	manifest.CLI = &scenario.CLIConfig{
		Enabled: true,
		Command: "resource-alpha",
		Adapter: scenario.CLIAdapterConfig{
			Kind:      "go_module",
			ModuleDir: "cli",
		},
		SourceBuild: &scenario.CLISourceBuildConfig{Kind: "go_module"},
		Invoke:      scenario.CLIInvokeConfig{Kind: "installed_command", Command: "resource-alpha"},
		Freshness: &scenario.CLIFreshnessCheck{
			Inputs: []string{"cli/**", "resource.json"},
		},
	}
	testresource.WriteResourceManifest(t, fixture.Root, "alpha", manifest)

	item, err := mustManager(t, fixture.Root, t.TempDir()).DiscoverResourceCLI("alpha")
	if err != nil {
		t.Fatalf("DiscoverResourceCLI: %v", err)
	}
	before, err := computeResourceCLIFingerprint(item)
	if err != nil {
		t.Fatalf("computeResourceCLIFingerprint before: %v", err)
	}

	manifestPath := filepath.Join(fixture.Root, "resources", "alpha", "resource.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read resource manifest: %v", err)
	}
	data = []byte(strings.Replace(string(data), "\"alpha\"", "\"alpha-updated\"", 1))
	if err := os.WriteFile(manifestPath, data, 0o644); err != nil {
		t.Fatalf("write resource manifest: %v", err)
	}

	after, err := computeResourceCLIFingerprint(item)
	if err != nil {
		t.Fatalf("computeResourceCLIFingerprint after: %v", err)
	}
	if before == after {
		t.Fatal("expected resource.json change to affect go-module CLI fingerprint")
	}
}

func TestInstalledScenarioCLINames(t *testing.T) {
	if runtime.GOOS == "windows" {
		testkitgo.SkipPlatform(t, "exec-bit filtering is unix-only; skipping on windows")
	}
	home := t.TempDir()
	binDir := filepath.Join(home, ".vrooli", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}

	writeExec := func(name string, mode os.FileMode) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(binDir, name), []byte(shelltest.POSIXShebang()+""), mode); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	// Real scenario CLIs (executable).
	writeExec("prompt-manager", 0o755)
	writeExec("agent-manager", 0o755)
	// Root vrooli binaries — must be excluded.
	writeExec("vrooli", 0o755)
	writeExec("vrooli-api", 0o755)
	writeExec("vrooli-buildmeta", 0o755)
	writeExec("vrooli-ports-migrate", 0o755)
	// Sidecar files — must be excluded.
	if err := os.WriteFile(filepath.Join(binDir, "prompt-manager.build.meta"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("write build meta: %v", err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "prompt-manager.manifest.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	// Non-executable file — must be excluded.
	if err := os.WriteFile(filepath.Join(binDir, "readme.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	// Subdirectory — must be excluded.
	if err := os.MkdirAll(filepath.Join(binDir, "subdir"), 0o755); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}

	manager := mustManager(t, t.TempDir(), home)
	got, err := manager.InstalledScenarioCLINames()
	if err != nil {
		t.Fatalf("InstalledScenarioCLINames: %v", err)
	}
	want := []string{"agent-manager", "prompt-manager"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestInstalledScenarioCLINamesMissingDir(t *testing.T) {
	manager := mustManager(t, t.TempDir(), t.TempDir())
	got, err := manager.InstalledScenarioCLINames()
	if err != nil {
		t.Fatalf("expected nil error for missing dir, got %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil names for missing dir, got %v", got)
	}
}

func TestRemoveScenarioCLIRemovesTripleIdempotently(t *testing.T) {
	manager := mustManager(t, t.TempDir(), t.TempDir())
	if err := os.MkdirAll(manager.InstallDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(manager.InstallDir(), "rcl-fixture-positive")
	for _, path := range []string{binary, binary + ".build.meta", binary + ".manifest.json"} {
		if err := os.WriteFile(path, []byte("fixture"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	removed, err := manager.RemoveScenarioCLI("rcl-fixture-positive")
	if err != nil || removed != 3 {
		t.Fatalf("RemoveScenarioCLI = removed %d, err %v; want 3", removed, err)
	}
	removed, err = manager.RemoveScenarioCLI("rcl-fixture-positive")
	if err != nil || removed != 0 {
		t.Fatalf("idempotent RemoveScenarioCLI = removed %d, err %v; want 0", removed, err)
	}
}

func TestRemoveScenarioCLISkipsLockedArtifact(t *testing.T) {
	manager := mustManager(t, t.TempDir(), t.TempDir())
	if err := os.MkdirAll(manager.InstallDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(manager.InstallDir(), "locked-fixture")
	for _, path := range []string{binary, binary + ".build.meta", binary + ".manifest.json"} {
		if err := os.WriteFile(path, []byte("fixture"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	manager.removeFile = func(path string) error {
		if path == binary {
			return fs.ErrPermission
		}
		return os.Remove(path)
	}

	report, err := manager.RemoveScenarioCLIReport("locked-fixture")
	if err != nil {
		t.Fatalf("RemoveScenarioCLIReport error = %v", err)
	}
	if report.Removed != 2 || !reflect.DeepEqual(report.Skipped, []string{binary}) {
		t.Fatalf("report = %+v, want two removed and %q skipped", report, binary)
	}
	if _, err := os.Stat(binary); err != nil {
		t.Fatalf("locked binary should remain for operator remediation: %v", err)
	}
}

func writeInstallMetadataFixture(t *testing.T, path string, meta InstallMetadata) {
	t.Helper()
	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
}

func validNativeExecutableHeader() []byte {
	switch runtime.GOOS {
	case "linux", "freebsd", "openbsd", "netbsd":
		return []byte("\x7fELF")
	case "darwin":
		return []byte("\xfe\xed\xfa\xcf")
	case "windows":
		return []byte("MZ\x00\x00")
	default:
		return []byte("\x7fELF")
	}
}

func mapProjectResources(enabled map[string]bool) scenario.Dependencies {
	return scenario.Dependencies{Resources: mapResourceDependencies(enabled)}
}

func mapResourceDependencies(enabled map[string]bool) map[string]scenario.Dependency {
	deps := make(map[string]scenario.Dependency, len(enabled))
	for name, ok := range enabled {
		deps[name] = scenario.Dependency{Enabled: ok}
	}
	return deps
}

// [REQ:VROOLI-ARTIFACT-ATTRIBUTION]
// Scenario CLI triples were disappearing with no record of who removed them or
// why. Every removal this path performs must now leave a receipt naming the
// artifact, the code path, and the rule that authorized it.
func TestRemoveScenarioCLIWritesAReceiptForEveryArtifact(t *testing.T) {
	home := t.TempDir()
	manager := mustManager(t, t.TempDir(), home)
	if err := os.MkdirAll(manager.InstallDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(manager.InstallDir(), "rcl-fixture-positive")
	for _, path := range []string{binary, binary + ".build.meta", binary + ".manifest.json"} {
		if err := os.WriteFile(path, []byte("fixture"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := manager.RemoveScenarioCLI("rcl-fixture-positive"); err != nil {
		t.Fatalf("RemoveScenarioCLI: %v", err)
	}

	ledger, err := artifactledger.New(home)
	if err != nil {
		t.Fatalf("open ledger: %v", err)
	}
	receipts, err := ledger.Read()
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}

	removed := map[string]bool{}
	for _, receipt := range receipts {
		if receipt.Outcome != artifactledger.OutcomeRemoved {
			continue
		}
		removed[receipt.Kind] = true
		if receipt.Component != "cliinstall.RemoveScenarioCLIReport" {
			t.Fatalf("receipt does not name the removing code path: %+v", receipt)
		}
		if receipt.Predicate == "" {
			t.Fatalf("receipt records a deletion without the rule that authorized it: %+v", receipt)
		}
		if receipt.Identity.Node == "" || receipt.Identity.PID == 0 {
			t.Fatalf("receipt carries no observed identity: %+v", receipt.Identity)
		}
	}
	for _, kind := range []string{"binary", "build-metadata", "manifest"} {
		if !removed[kind] {
			t.Fatalf("no removal receipt for the %s artifact; receipts=%+v", kind, receipts)
		}
	}
}

// A Manager assembled without a ledger must refuse to delete rather than
// perform an unattributable removal.
func TestRemoveScenarioCLIRefusesWithoutALedger(t *testing.T) {
	manager := mustManager(t, t.TempDir(), t.TempDir())
	if err := os.MkdirAll(manager.InstallDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(manager.InstallDir(), "unattributable")
	if err := os.WriteFile(binary, []byte("fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	manager.ledger = nil

	if _, err := manager.RemoveScenarioCLI("unattributable"); err == nil {
		t.Fatal("removal proceeded with no ledger configured")
	}
	if _, err := os.Stat(binary); err != nil {
		t.Fatalf("the artifact was removed despite the refusal: %v", err)
	}
}

// [REQ:VROOLI-ARTIFACT-LEASE]
// An installed CLI must carry an ownership record, or nothing downstream can
// tell a live artifact from an abandoned one.
func TestRecordInstalledCLIClaimsOwnership(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	manager := mustManager(t, root, home)
	if err := os.MkdirAll(manager.InstallDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	item := InstallableCLI{Kind: KindScenario, Name: "fixture-cli", BinaryName: "fixture-cli", ModulePath: filepath.Join(root, "scenarios", "fixture-cli", "cli")}
	binary := manager.InstalledBinaryPath(item)
	if err := os.WriteFile(binary, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := manager.recordInstalledCLI(item); err != nil {
		t.Fatalf("recordInstalledCLI: %v", err)
	}
	manager.noteOwnership(item, binary, true)

	lease, found, err := artifactlease.Load(binary)
	if err != nil || !found {
		t.Fatalf("no ownership record after install: %v found=%v", err, found)
	}
	if lease.Generation != 1 {
		t.Fatalf("generation = %d, want 1", lease.Generation)
	}
	if lease.OwnerModule != item.ModulePath {
		t.Fatalf("owner module = %q, want %q", lease.OwnerModule, item.ModulePath)
	}
	if lease.Owner.Node == "" {
		t.Fatalf("observed owner identity is empty: %+v", lease.Owner)
	}
}

// Removing the triple must remove its ownership record too, or a later install
// inherits a stale generation and a stale recorded absence.
func TestRemoveScenarioCLIClearsTheOwnershipRecord(t *testing.T) {
	home := t.TempDir()
	manager := mustManager(t, t.TempDir(), home)
	if err := os.MkdirAll(manager.InstallDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(manager.InstallDir(), "fixture-cli")
	for _, path := range []string{binary, binary + ".build.meta", binary + ".manifest.json"} {
		if err := os.WriteFile(path, []byte("fixture"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := artifactlease.Claim(binary, artifactlease.Owner{Node: "n"}, "/gone", time.Hour, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	if _, err := manager.RemoveScenarioCLI("fixture-cli"); err != nil {
		t.Fatalf("RemoveScenarioCLI: %v", err)
	}

	if _, err := os.Stat(artifactlease.Path(binary)); !os.IsNotExist(err) {
		t.Fatalf("the ownership record survived removal of the artifact it describes: %v", err)
	}
}

// [REQ:VROOLI-ARTIFACT-LEASE]
// The generation exists to detect that an artifact was replaced. A freshness
// check that finds the binary already current is not a replacement, and
// counting it as one made the generation climb from 3 to 8 in under an hour on
// an artifact nothing had rebuilt -- destroying the only signal it carries.
func TestFreshnessCheckRenewsWithoutClaimingANewGeneration(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	manager := mustManager(t, root, home)
	if err := os.MkdirAll(manager.InstallDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	item := InstallableCLI{Kind: KindScenario, Name: "fixture-cli", BinaryName: "fixture-cli", ModulePath: filepath.Join(root, "scenarios", "fixture-cli", "cli")}
	binary := manager.InstalledBinaryPath(item)
	if err := os.WriteFile(binary, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	manager.noteOwnership(item, binary, true)
	claimed, _, err := artifactlease.Load(binary)
	if err != nil {
		t.Fatal(err)
	}

	for range 5 {
		manager.noteOwnership(item, binary, false)
	}

	renewed, _, err := artifactlease.Load(binary)
	if err != nil {
		t.Fatal(err)
	}
	if renewed.Generation != claimed.Generation {
		t.Fatalf("generation moved from %d to %d across freshness checks that replaced nothing",
			claimed.Generation, renewed.Generation)
	}
}

// Rewriting a lease on every invocation is pointless disk churn, and it buries
// real artifact activity in the filesystem traces used to diagnose this system.
func TestRenewalSkipsWritingWhenNothingWouldChange(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	manager := mustManager(t, root, home)
	if err := os.MkdirAll(manager.InstallDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	item := InstallableCLI{Kind: KindScenario, Name: "fixture-cli", BinaryName: "fixture-cli", ModulePath: filepath.Join(root, "scenarios", "fixture-cli", "cli")}
	binary := manager.InstalledBinaryPath(item)
	if err := os.WriteFile(binary, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	manager.noteOwnership(item, binary, true)

	before, err := os.Stat(artifactlease.Path(binary))
	if err != nil {
		t.Fatal(err)
	}
	manager.noteOwnership(item, binary, false)
	after, err := os.Stat(artifactlease.Path(binary))
	if err != nil {
		t.Fatal(err)
	}

	if !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("a renewal rewrote a lease that was still fresh and unmarked")
	}
}
