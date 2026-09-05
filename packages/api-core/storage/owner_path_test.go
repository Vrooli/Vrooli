package storage

import (
	"errors"
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

func TestResolveOwnerStoragePathHonorsDeclaredPlatforms(t *testing.T) {
	owner := OwnerManifest{Kind: OwnerTool, ID: "kdump-tools", Platforms: []Platform{PlatformLinux}}
	entry := StorageEntry{Name: "crash_dumps", Path: PortablePath{Value: "/var/crash"}}
	for _, platform := range []Platform{PlatformMacOS, PlatformWindows} {
		_, err := ResolveOwnerStoragePath("/repo", owner, entry, platform, PlatformSeams{})
		var absent *NotApplicable
		if !errors.As(err, &absent) {
			t.Fatalf("%s error = %v, want NotApplicable", platform, err)
		}
	}
	path, err := ResolveOwnerStoragePath("/repo", owner, entry, PlatformLinux, PlatformSeams{})
	if err != nil || path != "/var/crash" {
		t.Fatalf("linux path = %q, err=%v", path, err)
	}
}

func TestEffectivePlatformsEntryNarrowsOwner(t *testing.T) {
	owner := OwnerManifest{Platforms: []Platform{PlatformLinux, PlatformMacOS, PlatformWindows}}
	entry := StorageEntry{Platforms: []Platform{PlatformLinux}}
	got := EffectivePlatforms(owner, entry)
	if len(got) != 1 || got[0] != PlatformLinux {
		t.Fatalf("effective platforms = %#v", got)
	}
}

func TestResolveOwnerStoragePathAgreesWithResolverForEveryOwnerKindAndClass(t *testing.T) {
	root := t.TempDir()
	seams := PlatformSeams{UserHomeDir: func() (string, error) { return filepath.Join(root, "home"), nil }}
	r, err := NewResolver(ResolverConfig{UserHomeDir: seams.UserHomeDir})
	if err != nil {
		t.Fatal(err)
	}
	classes := []Class{ClassConfig, ClassData, ClassCache, ClassLogs, ClassState}
	kinds := []OwnerKind{OwnerScenario, OwnerResource, OwnerTool, OwnerSafeguard}
	for _, kind := range kinds {
		for _, class := range classes {
			owner := OwnerManifest{Kind: kind, ID: string(kind) + "-fixture", ManifestPath: filepath.Join(root, string(kind), "fixture.json")}
			entry := StorageEntry{Name: string(class), Class: class, Subpath: "nested"}
			want, err := r.ResolveOwner(root, owner, entry, PlatformLinux, seams)
			if err != nil {
				t.Fatalf("resolver %s/%s: %v", kind, class, err)
			}
			got, err := ResolveOwnerStoragePath(root, owner, entry, PlatformLinux, seams)
			if err != nil {
				t.Fatalf("compat resolver %s/%s: %v", kind, class, err)
			}
			if got != want {
				t.Errorf("%s/%s: compatibility path %q differs from resolver %q", kind, class, got, want)
			}
		}
	}
}

func TestResolverUsesLifecycleScenarioDataRootForCurrentScenario(t *testing.T) {
	t.Setenv("SCENARIO_NAME", "demo")
	t.Setenv("SCENARIO_DATA_DIR", "/tmp/vrooli-demo-data")
	owner := OwnerManifest{Kind: OwnerScenario, ID: "demo"}
	entry := StorageEntry{Class: ClassData, Subpath: "storage.db"}
	path, err := ResolveOwnerStoragePath("/repo", owner, entry, PlatformLinux, PlatformSeams{})
	if err != nil {
		t.Fatal(err)
	}
	if path != "/tmp/vrooli-demo-data/storage.db" {
		t.Fatalf("runtime data path = %q", path)
	}
}
