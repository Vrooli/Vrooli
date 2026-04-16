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
	"sort"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/scenario"
	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
	testresource "github.com/vrooli/vrooli/packages/testkit-go/resourcefixture"
	testscenario "github.com/vrooli/vrooli/packages/testkit-go/scenariofixture"
)

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
			Install: []scenario.CLIInstallStep{{
				Kind: "command",
				Run:  "cd cli && ./install.sh",
			}},
		}),
	))
}

func writeShellScenarioCLIManifest(t *testing.T, root, name string) {
	t.Helper()
	testscenario.WriteScenarioService(t, root, name, testscenario.ScenarioServiceManifest(
		name,
		testscenario.WithCLI(&scenario.CLIConfig{
			Enabled: true,
			Command: name,
			Adapter: scenario.CLIAdapterConfig{
				Kind:          "shell_script",
				ScriptPath:    filepath.ToSlash(filepath.Join("cli", name)),
				InstallScript: "cli/install.sh",
			},
			Install: []scenario.CLIInstallStep{{
				Kind: "command",
				Run:  "cd cli && ./install.sh",
			}},
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

func writeShellResourceCLIManifest(t *testing.T, root, name string) {
	t.Helper()
	manifest := testresource.ResourceManifest(name)
	manifest.CLI = &scenario.CLIConfig{
		Enabled: true,
		Command: "resource-" + name,
		Adapter: scenario.CLIAdapterConfig{
			Kind:          "shell_script",
			ScriptPath:    filepath.ToSlash(filepath.Join("cli", "resource-"+name)),
			InstallScript: "cli/install.sh",
		},
		Install: []scenario.CLIInstallStep{{
			Kind: "command",
			Run:  "cd cli && ./install.sh",
		}},
	}
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

	manager := NewManager(fixture.Root, fixture.Home)
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
}

func TestDiscoverScenarioCLIsIncludesShellScriptAdapter(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	fixture.WriteScenarioStub(t, "alpha")
	writeShellScenarioCLIManifest(t, fixture.Root, "alpha")
	testkitgo.WriteRelativeExecutable(t, fixture.Root, filepath.Join("scenarios", "alpha", "cli", "alpha"), "#!/usr/bin/env bash\nexit 0\n")
	testkitgo.WriteRelativeExecutable(t, fixture.Root, filepath.Join("scenarios", "alpha", "cli", "install.sh"), "#!/usr/bin/env bash\nexit 0\n")

	manager := NewManager(fixture.Root, fixture.Home)
	item, err := manager.DiscoverScenarioCLI("alpha")
	if err != nil {
		t.Fatalf("DiscoverScenarioCLI: %v", err)
	}
	if item.CLI == nil || item.CLI.Adapter.Kind != "shell_script" {
		t.Fatalf("expected shell_script adapter, got %+v", item)
	}
	if item.ScenarioPath != filepath.Join(fixture.Root, "scenarios", "alpha") {
		t.Fatalf("scenario path = %q", item.ScenarioPath)
	}
}

func TestDiscoverScenarioCLIDoesNotInferGoModuleFromLayoutWithoutManifest(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	fixture.WriteScenarioStub(t, "alpha")
	testscenario.WriteScenarioService(t, fixture.Root, "alpha", testscenario.ScenarioServiceManifest("alpha"))
	testscenario.WriteScenarioCLIGoMod(t, fixture.Root, "alpha", "alpha/cli")
	testkitgo.WriteRelativeFile(t, fixture.Root, filepath.Join("scenarios", "alpha", "cli", "main.go"), "package main\nfunc main() {}\n")

	manager := NewManager(fixture.Root, fixture.Home)
	_, err := manager.DiscoverScenarioCLI("alpha")
	if err == nil {
		t.Fatal("expected missing CLI manifest error, got nil")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected not-exist error when layout exists without manifest, got %v", err)
	}
}

func TestDiscoverScenarioCLIDoesNotInferShellScriptFromLayoutWithoutManifest(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	fixture.WriteScenarioStub(t, "alpha")
	testscenario.WriteScenarioService(t, fixture.Root, "alpha", testscenario.ScenarioServiceManifest("alpha"))
	testkitgo.WriteRelativeExecutable(t, fixture.Root, filepath.Join("scenarios", "alpha", "cli", "alpha"), "#!/usr/bin/env bash\nexit 0\n")
	testkitgo.WriteRelativeExecutable(t, fixture.Root, filepath.Join("scenarios", "alpha", "cli", "install.sh"), "#!/usr/bin/env bash\nexit 0\n")

	manager := NewManager(fixture.Root, fixture.Home)
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

	manager := NewManager(fixture.Root, fixture.Home)
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
}

func TestDiscoverResourceCLIsReturnsAllInstallableModules(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	writeGoResourceCLIManifest(t, fixture.Root, "postgres")
	writeGoResourceCLIManifest(t, fixture.Root, "redis")
	testresource.WriteResourceCLIGoMod(t, fixture.Root, "postgres", "resource-postgres/cli")
	testresource.WriteResourceCLIGoMod(t, fixture.Root, "redis", "resource-redis/cli")

	manager := NewManager(fixture.Root, fixture.Home)
	items, err := manager.DiscoverResourceCLIs()
	if err != nil {
		t.Fatalf("DiscoverResourceCLIs: %v", err)
	}

	got := []string{items[0].BinaryName, items[1].BinaryName}
	if !reflect.DeepEqual(got, []string{"resource-postgres", "resource-redis"}) {
		t.Fatalf("resource CLI binaries = %v, want %v", got, []string{"resource-postgres", "resource-redis"})
	}
}

func TestDiscoverResourceCLIIncludesShellScriptAdapter(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	writeShellResourceCLIManifest(t, fixture.Root, "postgres")
	testkitgo.WriteRelativeExecutable(t, fixture.Root, filepath.Join("resources", "postgres", "cli", "resource-postgres"), "#!/usr/bin/env bash\nexit 0\n")
	testkitgo.WriteRelativeExecutable(t, fixture.Root, filepath.Join("resources", "postgres", "cli", "install.sh"), "#!/usr/bin/env bash\nexit 0\n")

	manager := NewManager(fixture.Root, fixture.Home)
	item, err := manager.DiscoverResourceCLI("postgres")
	if err != nil {
		t.Fatalf("DiscoverResourceCLI: %v", err)
	}
	if item.CLI == nil || item.CLI.Adapter.Kind != "shell_script" {
		t.Fatalf("expected shell_script adapter, got %+v", item)
	}
	if item.ResourcePath != filepath.Join(fixture.Root, "resources", "postgres") {
		t.Fatalf("resource path = %q", item.ResourcePath)
	}
}

func TestDiscoverResourceCLIDoesNotInferGoModuleFromLayoutWithoutEnabledManifest(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	writeDisabledResourceCLIManifest(t, fixture.Root, "postgres")
	testresource.WriteResourceCLIGoMod(t, fixture.Root, "postgres", "resource-postgres/cli")
	testkitgo.WriteRelativeFile(t, fixture.Root, filepath.Join("resources", "postgres", "cli", "main.go"), "package main\nfunc main() {}\n")

	manager := NewManager(fixture.Root, fixture.Home)
	_, err := manager.DiscoverResourceCLI("postgres")
	if err == nil {
		t.Fatal("expected missing CLI manifest error, got nil")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected not-exist error when layout exists without enabled manifest, got %v", err)
	}
}

func TestDiscoverResourceCLIDoesNotInferShellScriptFromLayoutWithoutEnabledManifest(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	writeDisabledResourceCLIManifest(t, fixture.Root, "postgres")
	testkitgo.WriteRelativeExecutable(t, fixture.Root, filepath.Join("resources", "postgres", "cli", "resource-postgres"), "#!/usr/bin/env bash\nexit 0\n")
	testkitgo.WriteRelativeExecutable(t, fixture.Root, filepath.Join("resources", "postgres", "cli", "install.sh"), "#!/usr/bin/env bash\nexit 0\n")

	manager := NewManager(fixture.Root, fixture.Home)
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

	manager := NewManager(fixture.Root, fixture.Home)
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
	manager := NewManager(fixture.Root, fixture.Home)
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
	manager := NewManager(fixture.Root, fixture.Home)
	manager.Installer = installer

	if err := manager.EnsureScenarioCLI("alpha"); err != nil {
		t.Fatalf("EnsureScenarioCLI: %v", err)
	}
	if len(installer.calls) != 1 {
		t.Fatalf("install calls = %d, want 1", len(installer.calls))
	}
}

func TestEnsureScenarioCLIShellScriptInstallsWhenMissing(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	fixture.WriteScenarioStub(t, "alpha")
	writeShellScenarioCLIManifest(t, fixture.Root, "alpha")
	testkitgo.WriteRelativeExecutable(t, fixture.Root, filepath.Join("scenarios", "alpha", "cli", "alpha"), "#!/usr/bin/env bash\nexit 0\n")
	testkitgo.WriteRelativeExecutable(t, fixture.Root, filepath.Join("scenarios", "alpha", "cli", "install.sh"), "#!/usr/bin/env bash\nexit 0\n")

	installer := &stubInstaller{}
	manager := NewManager(fixture.Root, fixture.Home)
	manager.Installer = installer

	if err := manager.EnsureScenarioCLI("alpha"); err != nil {
		t.Fatalf("EnsureScenarioCLI: %v", err)
	}
	if len(installer.calls) != 1 {
		t.Fatalf("install calls = %d, want 1", len(installer.calls))
	}
	if installer.calls[0].item.CLI == nil || installer.calls[0].item.CLI.Adapter.Kind != "shell_script" {
		t.Fatalf("expected shell_script install item, got %+v", installer.calls[0].item)
	}
}

func TestInspectScenarioCLIInstallLocationReportsPathMismatch(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	fixture.WriteScenarioStub(t, "alpha")
	writeGoScenarioCLIManifest(t, fixture.Root, "alpha")
	testscenario.WriteScenarioCLIGoMod(t, fixture.Root, "alpha", "alpha/cli")

	manager := NewManager(fixture.Root, fixture.Home)
	canonical := filepath.Join(fixture.Home, ".vrooli", "bin", "alpha")
	if err := os.MkdirAll(filepath.Dir(canonical), 0o755); err != nil {
		t.Fatalf("mkdir install dir: %v", err)
	}
	if err := os.WriteFile(canonical, []byte("installed"), 0o755); err != nil {
		t.Fatalf("write canonical binary: %v", err)
	}

	other := testkitgo.WriteRelativeExecutable(t, fixture.Home, filepath.Join(".local", "bin", "alpha"), "#!/usr/bin/env bash\nexit 0\n")
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

	manager := NewManager(fixture.Root, fixture.Home)
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
	manager := NewManager(fixture.Root, fixture.Home)
	manager.Installer = installer

	binaryPath := filepath.Join(fixture.Home, ".vrooli", "bin", "resource-postgres")
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		t.Fatalf("mkdir install dir: %v", err)
	}
	if err := os.WriteFile(binaryPath, []byte("installed"), 0o755); err != nil {
		t.Fatalf("write installed binary: %v", err)
	}
	fingerprint, err := computeTestFingerprint(filepath.Join(fixture.Root, "resources", "postgres", "cli"), "resource-postgres")
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

func TestEnsureResourceCLIInstallsWhenMetadataMissing(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	writeGoResourceCLIManifest(t, fixture.Root, "postgres")
	testresource.WriteResourceCLIGoMod(t, fixture.Root, "postgres", "resource-postgres/cli")

	installer := &stubInstaller{}
	manager := NewManager(fixture.Root, fixture.Home)
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
	manager := NewManager(fixture.Root, fixture.Home)
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

func TestEnsureResourceCLIShellScriptInstallsWhenMissing(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	writeShellResourceCLIManifest(t, fixture.Root, "postgres")
	testkitgo.WriteRelativeExecutable(t, fixture.Root, filepath.Join("resources", "postgres", "cli", "resource-postgres"), "#!/usr/bin/env bash\nexit 0\n")
	testkitgo.WriteRelativeExecutable(t, fixture.Root, filepath.Join("resources", "postgres", "cli", "install.sh"), "#!/usr/bin/env bash\nexit 0\n")

	installer := &stubInstaller{}
	manager := NewManager(fixture.Root, fixture.Home)
	manager.Installer = installer

	if err := manager.EnsureResourceCLI("postgres"); err != nil {
		t.Fatalf("EnsureResourceCLI: %v", err)
	}
	if len(installer.calls) != 1 {
		t.Fatalf("install calls = %d, want 1", len(installer.calls))
	}
	if installer.calls[0].item.CLI == nil || installer.calls[0].item.CLI.Adapter.Kind != "shell_script" {
		t.Fatalf("expected shell_script install item, got %+v", installer.calls[0].item)
	}
}

func TestRepoEnabledResourceCLIsAreAllDiscoverable(t *testing.T) {
	root := testkitgo.ProjectRoot(t)
	manager := NewManager(root, t.TempDir())

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
	manager := NewManager(root, t.TempDir())
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

func computeTestFingerprint(modulePath, binaryName string) (string, error) {
	return computeFingerprint(modulePath, binaryName)
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
