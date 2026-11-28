package resources

import (
	"path/filepath"
	"testing"

	runtimestorage "github.com/vrooli/vrooli/internal/resources/runtime/storage"
)

func invokingStorageFixture(home string, sudoed bool, env map[string]string) (*runtimestorage.Resolver, error) {
	return resourceStorageResolverForInvoker(home, sudoed, runtimestorage.ResolverConfig{RuntimeOS: "linux", EnvGet: func(key string) string { return env[key] }, UserConfigDir: func() (string, error) {
		if value := env["XDG_CONFIG_HOME"]; value != "" {
			return value, nil
		}
		return filepath.Join(env["HOME"], ".config"), nil
	}, UserCacheDir: func() (string, error) {
		if value := env["XDG_CACHE_HOME"]; value != "" {
			return value, nil
		}
		return filepath.Join(env["HOME"], ".cache"), nil
	}})
}

func TestResourceStorageSudoDefaultsAllFollowInvokingHome(t *testing.T) {
	home := filepath.Join(t.TempDir(), "operator")
	resolver, err := invokingStorageFixture(home, true, map[string]string{"HOME": "/root", "XDG_CONFIG_HOME": "/root/.config", "XDG_CACHE_HOME": "/root/.cache", "XDG_DATA_HOME": "/root/.local/share", "XDG_STATE_HOME": "/root/.local/state"})
	if err != nil {
		t.Fatal(err)
	}
	paths, err := resolver.Resolve(runtimestorage.Options{ResourceID: "openrgb"})
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]string{"config": filepath.Join(home, ".config"), "cache": filepath.Join(home, ".cache"), "data": filepath.Join(home, ".local", "share"), "state": filepath.Join(home, ".local", "state"), "logs": filepath.Join(home, ".local", "state", "logs")}
	got := map[string]string{"config": paths.ConfigDir, "cache": paths.CacheDir, "data": paths.DataDir, "state": paths.StateDir, "logs": paths.LogsDir}
	for class, root := range expected {
		if want := filepath.Join(root, "vrooli", "resources", "openrgb"); got[class] != want {
			t.Errorf("sudo %s=%q want invoking-user path %q", class, got[class], want)
		}
	}
}

func TestResourceStorageOrdinaryAndExplicitRootSemanticsRemain(t *testing.T) {
	home := filepath.Join(t.TempDir(), "operator")
	ordinary, err := invokingStorageFixture(home, false, map[string]string{"HOME": home, "XDG_CONFIG_HOME": "/custom/config", "XDG_CACHE_HOME": "/custom/cache", "XDG_DATA_HOME": "/custom/data", "XDG_STATE_HOME": "/custom/state"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := ordinary.Resolve(runtimestorage.Options{ResourceID: "openrgb"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ConfigDir != "/custom/config/vrooli/resources/openrgb" || got.CacheDir != "/custom/cache/vrooli/resources/openrgb" || got.DataDir != "/custom/data/vrooli/resources/openrgb" || got.StateDir != "/custom/state/vrooli/resources/openrgb" {
		t.Fatalf("ordinary resolver changed: %+v", got)
	}
	override := filepath.Join(t.TempDir(), "explicit-resource-root")
	for _, sudoed := range []bool{false, true} {
		resolver, err := invokingStorageFixture(home, sudoed, map[string]string{"HOME": "/root", "VROOLI_RESOURCE_STORAGE_ROOT": override})
		if err != nil {
			t.Fatal(err)
		}
		paths, err := resolver.Resolve(runtimestorage.Options{ResourceID: "openrgb"})
		if err != nil {
			t.Fatal(err)
		}
		got := map[string]string{"config": paths.ConfigDir, "data": paths.DataDir, "cache": paths.CacheDir, "state": paths.StateDir, "logs": paths.LogsDir}
		for class, path := range got {
			if want := filepath.Join(override, class, "vrooli", "resources", "openrgb"); path != want {
				t.Errorf("explicit root sudo=%v %s=%q want %q", sudoed, class, path, want)
			}
		}
	}
}

func TestResourceStorageExplicitClassRootsRemainCallerControlled(t *testing.T) {
	home := filepath.Join(t.TempDir(), "operator")
	for _, sudoed := range []bool{false, true} {
		resolver, err := invokingStorageFixture(home, sudoed, map[string]string{"HOME": "/root", "VROOLI_RESOURCE_CONFIG_ROOT": "/explicit/config", "VROOLI_RESOURCE_CACHE_ROOT": "/explicit/cache"})
		if err != nil {
			t.Fatal(err)
		}
		got, err := resolver.Resolve(runtimestorage.Options{ResourceID: "openrgb"})
		if err != nil {
			t.Fatal(err)
		}
		if got.ConfigDir != "/explicit/config/vrooli/resources/openrgb" || got.CacheDir != "/explicit/cache/vrooli/resources/openrgb" {
			t.Fatalf("explicit class roots sudo=%v changed: %+v", sudoed, got)
		}
	}
}
