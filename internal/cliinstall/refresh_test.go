package cliinstall

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
	scenariotest "github.com/vrooli/vrooli/internal/scenario/scenariotest"
)

type refreshTestInstaller struct {
	calls int
}

func (i *refreshTestInstaller) Install(_ context.Context, item InstallableCLI, installDir string) error {
	i.calls++
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(installDir, item.BinaryName), []byte("fixture executable"), 0o755)
}

func TestRefreshCLIRequiresManagerDiscoveredCanonicalExecutable(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	fixture.WriteScenarioStub(t, "alpha")
	writeGoScenarioCLIManifest(t, fixture.Root, "alpha")
	scenariotest.WriteScenarioCLIGoMod(t, fixture.Root, "alpha", "alpha/cli")

	manager := mustManager(t, fixture.Root, fixture.Home)
	installer := &refreshTestInstaller{}
	manager.Installer = installer
	item, err := manager.DiscoverScenarioCLI("alpha")
	if err != nil {
		t.Fatalf("discover scenario CLI: %v", err)
	}
	canonical := manager.InstalledBinaryPath(item)

	if err := manager.RefreshCLI(context.Background(), KindScenario, "alpha", filepath.Join(fixture.Home, "bin", "unrelated")); err == nil {
		t.Fatal("RefreshCLI accepted an unrelated running executable path")
	}
	if installer.calls != 0 {
		t.Fatalf("installer calls after path mismatch = %d, want 0", installer.calls)
	}

	if err := manager.RefreshCLI(context.Background(), KindScenario, "alpha", canonical); err != nil {
		t.Fatalf("RefreshCLI with canonical target: %v", err)
	}
	if installer.calls != 1 {
		t.Fatalf("installer calls = %d, want one manager-discovered install", installer.calls)
	}
}
