package cliinstall

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
)

type stubInstaller struct {
	calls []installCall
}

type installCall struct {
	item       InstallableCLI
	installDir string
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
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", "alpha", "cli", "go.mod"), "module alpha/cli\n")
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", "beta", "cli", "go.mod"), "module beta/cli\n")

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

func TestDiscoverEnabledResourceCLIs(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	fixture.WriteResourceStub(t, "postgres")
	fixture.WriteResourceStub(t, "redis")
	fixture.WriteResourceStub(t, "ollama")
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "resources", "postgres", "cli", "go.mod"), "module resource-postgres/cli\n")
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "resources", "ollama", "cli", "go.mod"), "module resource-ollama/cli\n")
	testkitgo.WriteJSON(t, filepath.Join(fixture.Root, ".vrooli", "service.json"), map[string]any{
		"dependencies": map[string]any{
			"resources": map[string]any{
				"postgres": map[string]any{"enabled": true},
				"redis":    map[string]any{"enabled": true},
				"ollama":   map[string]any{"enabled": false},
			},
		},
	})

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
	fixture.WriteResourceStub(t, "postgres")
	fixture.WriteResourceStub(t, "redis")
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "resources", "postgres", "cli", "go.mod"), "module resource-postgres/cli\n")
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "resources", "redis", "cli", "go.mod"), "module resource-redis/cli\n")

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

func TestDiscoverScenarioCLIsReturnsUnexpectedErrors(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	fixture.WriteScenarioStub(t, "alpha")
	if err := os.MkdirAll(filepath.Join(fixture.Root, "scenarios", "alpha", "cli", "go.mod"), 0o755); err != nil {
		t.Fatalf("mkdir invalid go.mod path: %v", err)
	}

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
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", "alpha", "cli", "go.mod"), "module alpha/cli\n")
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", "beta", "cli", "go.mod"), "module beta/cli\n")

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
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", "alpha", "cli", "go.mod"), "module alpha/cli\n")

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

func TestEnsureResourceCLISkipsWhenInstalled(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	fixture.WriteResourceStub(t, "postgres")
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "resources", "postgres", "cli", "go.mod"), "module resource-postgres/cli\n")

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
	fixture.WriteResourceStub(t, "postgres")
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "resources", "postgres", "cli", "go.mod"), "module resource-postgres/cli\n")

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
	fixture.WriteResourceStub(t, "postgres")
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "resources", "postgres", "cli", "go.mod"), "module resource-postgres/cli\n")

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
