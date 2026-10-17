package storage

import (
	"path/filepath"
	"testing"
)

func lifecycleTestOwner(root string) OwnerManifest {
	return OwnerManifest{
		Kind:         OwnerScenario,
		ID:           "demo",
		ManifestPath: filepath.Join(root, "scenarios", "demo", ".vrooli", "service.json"),
	}
}

func TestResolveOwnerStorageLocationsAddsTheLifecycleDataDirForClassData(t *testing.T) {
	t.Setenv("SCENARIO_DATA_DIR", "")
	root := t.TempDir()
	seams := PlatformSeams{UserHomeDir: func() (string, error) { return filepath.Join(root, "home"), nil }}
	entry := StorageEntry{Name: "database", Kind: "file", Class: ClassData, Subpath: "demo.db"}

	locations, err := ResolveOwnerStorageLocations(root, lifecycleTestOwner(root), entry, HostPlatform(), seams)
	if err != nil {
		t.Fatalf("locations: %v", err)
	}
	primary, err := ResolveOwnerStoragePath(root, lifecycleTestOwner(root), entry, HostPlatform(), seams)
	if err != nil {
		t.Fatalf("primary: %v", err)
	}
	want := filepath.Join(root, "scenarios", "demo", "data", "demo.db")
	if len(locations) != 2 || locations[0] != primary || locations[1] != want {
		t.Fatalf("locations = %#v, want [%s %s]", locations, primary, want)
	}
}

// TestResolveOwnerStorageLocationsKeepsOtherEntriesSingle proves the extra
// location is scoped to scenario class data: every other shape keeps exactly
// the one path pruners already act on.
func TestResolveOwnerStorageLocationsKeepsOtherEntriesSingle(t *testing.T) {
	t.Setenv("SCENARIO_DATA_DIR", "")
	root := t.TempDir()
	seams := PlatformSeams{UserHomeDir: func() (string, error) { return filepath.Join(root, "home"), nil }}
	other := HostPlatform()
	for _, candidate := range []Platform{PlatformLinux, PlatformMacOS, PlatformWindows} {
		if candidate != HostPlatform() {
			other = candidate
			break
		}
	}
	cases := []struct {
		name     string
		owner    OwnerManifest
		entry    StorageEntry
		platform Platform
	}{
		{"cache class", lifecycleTestOwner(root), StorageEntry{Name: "cache", Kind: "dir", Class: ClassCache}, HostPlatform()},
		{"state class", lifecycleTestOwner(root), StorageEntry{Name: "state", Kind: "dir", Class: ClassState}, HostPlatform()},
		{"pinned path", lifecycleTestOwner(root), StorageEntry{Name: "pinned", Kind: "dir", Path: PortablePath{Value: "data/pinned"}}, HostPlatform()},
		{"resource owner", OwnerManifest{Kind: OwnerResource, ID: "demo", ManifestPath: filepath.Join(root, "resources", "demo", "resource.json")}, StorageEntry{Name: "data", Kind: "dir", Class: ClassData}, HostPlatform()},
		{"foreign platform", lifecycleTestOwner(root), StorageEntry{Name: "data", Kind: "dir", Class: ClassData}, other},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.platform == HostPlatform() && tc.name == "foreign platform" {
				t.Skip("no foreign platform on this host")
			}
			locations, err := ResolveOwnerStorageLocations(root, tc.owner, tc.entry, tc.platform, seams)
			if err != nil {
				t.Fatalf("locations: %v", err)
			}
			if len(locations) != 1 {
				t.Fatalf("locations = %#v, want only the primary path", locations)
			}
		})
	}
}

// TestResolveOwnerStorageLocationsNeverChangesThePrimaryPath pins the safety
// property: the first location is exactly what every pruner already targets.
func TestResolveOwnerStorageLocationsNeverChangesThePrimaryPath(t *testing.T) {
	t.Setenv("SCENARIO_DATA_DIR", "")
	root := t.TempDir()
	seams := PlatformSeams{UserHomeDir: func() (string, error) { return filepath.Join(root, "home"), nil }}
	for _, entry := range []StorageEntry{
		{Name: "data", Kind: "dir", Class: ClassData},
		{Name: "wal", Kind: "file", Class: ClassData, Subpath: "demo.db-wal", Regenerable: true},
	} {
		primary, err := ResolveOwnerStoragePath(root, lifecycleTestOwner(root), entry, HostPlatform(), seams)
		if err != nil {
			t.Fatalf("primary: %v", err)
		}
		locations, err := ResolveOwnerStorageLocations(root, lifecycleTestOwner(root), entry, HostPlatform(), seams)
		if err != nil || locations[0] != primary {
			t.Fatalf("%s: locations[0] = %v (err %v), want primary %s", entry.Name, locations, err, primary)
		}
	}
}

func TestLifecycleDataDirMatchesTheLifecycleRule(t *testing.T) {
	if got := LifecycleDataDir("/repo/scenarios/demo"); got != "/repo/scenarios/demo/data" {
		t.Fatalf("LifecycleDataDir = %q", got)
	}
}
