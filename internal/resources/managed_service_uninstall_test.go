package resources

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/binaryfetch"
)

func TestUninstallManagedServiceRemovesOnlyOwnedArtifacts(t *testing.T) {
	f := newRetentionFixture(t, "v1", "active")
	profiles := filepath.Join(f.root, "home", ".config", "OpenRGB")
	if err := os.MkdirAll(profiles, 0o700); err != nil {
		t.Fatal(err)
	}
	profile := filepath.Join(profiles, "Off.orp")
	if err := os.WriteFile(profile, []byte("operator profile"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := f.controller.LoadManifest(filepath.Join(f.root, "resources", f.name, "resource.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := uninstallManagedServiceArtifact(context.Background(), f.controller, manifest); err != nil {
		t.Fatalf("uninstall managed artifacts: %v", err)
	}
	if _, err := os.Stat(filepath.Join(f.artifacts, f.name)); !os.IsNotExist(err) {
		t.Fatalf("managed artifacts remain after uninstall, stat error=%v", err)
	}
	if data, err := os.ReadFile(profile); err != nil || string(data) != "operator profile" {
		t.Fatalf("operator profile changed or removed: data=%q err=%v", data, err)
	}
}

func TestUninstallManagedServicePreservesAdoptedHostTool(t *testing.T) {
	f := newRetentionFixture(t, "active")
	manifest, err := f.controller.LoadManifest(filepath.Join(f.root, "resources", f.name, "resource.json"))
	if err != nil {
		t.Fatal(err)
	}
	manifest.ManagedService.Artifact.Verification = "host-tool"
	manifest.ManagedService.Acquisition = &binaryfetch.Acquisition{
		Kind: "none",
		Targets: []binaryfetch.AcquisitionTarget{{
			Executable: "fixture-openrgb",
		}},
	}
	if err := uninstallManagedServiceArtifact(context.Background(), f.controller, manifest); err != nil {
		t.Fatalf("uninstall adopted host tool: %v", err)
	}
	if !f.present("active") {
		t.Fatal("host-tool uninstall removed a separately managed artifact")
	}
}

func TestUninstallManagedServiceRefusesArtifactsInUse(t *testing.T) {
	f := newRetentionFixture(t, "active")
	manifest, err := f.controller.LoadManifest(filepath.Join(f.root, "resources", f.name, "resource.json"))
	if err != nil {
		t.Fatal(err)
	}
	previous := artifactVersionsInUse
	artifactVersionsInUse = func(string) map[string]bool { return map[string]bool{"active": true} }
	t.Cleanup(func() { artifactVersionsInUse = previous })
	err = uninstallManagedServiceArtifact(context.Background(), f.controller, manifest)
	if err == nil || !strings.Contains(err.Error(), "while a process still uses it") {
		t.Fatalf("uninstall in-use artifact error = %v", err)
	}
	if !f.present("active") {
		t.Fatal("uninstall removed an artifact still in use")
	}
}
