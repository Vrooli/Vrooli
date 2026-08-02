package storage

import (
	"path/filepath"
	"testing"
)

func TestResolveOwnerStoragePathUsesScenarioAndPortableBases(t *testing.T) {
	root := t.TempDir()
	scenario := OwnerManifest{
		Kind:         OwnerScenario,
		ManifestPath: filepath.Join(root, "scenarios", "demo", ".vrooli", "service.json"),
	}
	path, err := ResolveOwnerStoragePath(root, scenario, StorageEntry{
		Name: "database",
		Path: PortablePath{Value: "data/demo.sqlite"},
	}, PlatformLinux, PlatformSeams{})
	if err != nil {
		t.Fatalf("scenario path: %v", err)
	}
	if want := filepath.Join(root, "scenarios", "demo", "data", "demo.sqlite"); path != want {
		t.Fatalf("scenario path = %q, want %q", path, want)
	}

	resource := OwnerManifest{Kind: OwnerResource, ManifestPath: filepath.Join(root, "resources", "demo", "resource.json")}
	path, err = ResolveOwnerStoragePath(root, resource, StorageEntry{
		Name: "config",
		Path: PortablePath{Value: "$USER_CONFIG_DIR/vrooli/demo"},
	}, PlatformLinux, PlatformSeams{
		UserConfigDir: func() (string, error) { return filepath.Join(root, "config"), nil },
	})
	if err != nil {
		t.Fatalf("resource path: %v", err)
	}
	if want := filepath.Join(root, "config", "vrooli", "demo"); path != want {
		t.Fatalf("resource path = %q, want %q", path, want)
	}
}
