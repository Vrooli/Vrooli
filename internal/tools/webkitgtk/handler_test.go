package webkitgtk

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestInspectChecksDevelopmentModulesInsteadOfPkgConfigBinary(t *testing.T) {
	previous := checkDevelopmentModules
	t.Cleanup(func() { checkDevelopmentModules = previous })
	checkDevelopmentModules = func() bool { return false }

	manifest := hostreqkit.ToolManifest{
		Name: "webkitgtk", Commands: []string{"pkg-config"},
		DefaultPackage: "libwebkit2gtk-4.1-dev",
	}
	status := NewHandler(manifest).Inspect(hostreqkit.Host{OS: "linux", PackageManager: "apt-get"}, hostreqspec.ResolvedRequirement{
		Name: "webkitgtk", Kind: hostreqspec.KindTool, Required: true,
	})
	if status.Installed {
		t.Fatal("pkg-config executable was mistaken for installed WebKitGTK development headers")
	}
	if !status.InstallSupported || status.PackageName != "libwebkit2gtk-4.1-dev" {
		t.Fatalf("install declaration=%+v", status)
	}
}

func TestInspectReportsInstalledOnlyWhenEveryModuleResolves(t *testing.T) {
	previous := checkDevelopmentModules
	t.Cleanup(func() { checkDevelopmentModules = previous })
	checkDevelopmentModules = func() bool { return true }

	status := NewHandler(hostreqkit.ToolManifest{
		Name: "webkitgtk", Commands: []string{"pkg-config"}, VersionArgs: []string{"--version"},
		DefaultPackage: "libwebkit2gtk-4.1-dev",
	}).Inspect(hostreqkit.Host{OS: "linux", PackageManager: "apt-get"}, hostreqspec.ResolvedRequirement{
		Name: "webkitgtk", Kind: hostreqspec.KindTool, Required: true,
	})
	if !status.Installed || status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("available development modules were not adopted: %+v", status)
	}
}
