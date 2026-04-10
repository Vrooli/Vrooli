package storage

import (
	"path/filepath"
	"testing"
)

func TestResolveWindowsDefaults(t *testing.T) {
	t.Parallel()

	r := mustResolver(t, ResolverConfig{
		AppID:     "vrooli",
		RuntimeOS: "windows",
		EnvGet: mapEnv(map[string]string{
			"LOCALAPPDATA": `C:\Users\alice\AppData\Local`,
		}),
		UserHomeDir:   func() (string, error) { return `C:\Users\alice`, nil },
		UserConfigDir: func() (string, error) { return `C:\Users\alice\AppData\Roaming`, nil },
		UserCacheDir:  func() (string, error) { return `C:\Users\alice\AppData\Local\Cache`, nil },
	})

	paths, err := r.Resolve(Options{ScenarioID: "demo"})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if paths.ConfigDir != filepath.Join(`C:\Users\alice\AppData\Roaming`, "vrooli", "demo") {
		t.Fatalf("ConfigDir = %q", paths.ConfigDir)
	}
	if paths.DataDir != filepath.Join(`C:\Users\alice\AppData\Roaming`, "vrooli", "demo") {
		t.Fatalf("DataDir = %q", paths.DataDir)
	}
}

func TestResolveMacDefaults(t *testing.T) {
	t.Parallel()

	r := mustResolver(t, ResolverConfig{
		AppID:         "vrooli",
		RuntimeOS:     "darwin",
		EnvGet:        mapEnv(nil),
		UserHomeDir:   func() (string, error) { return "/Users/alice", nil },
		UserConfigDir: func() (string, error) { return "/Users/alice/Library/Application Support", nil },
		UserCacheDir:  func() (string, error) { return "/Users/alice/Library/Caches", nil },
	})

	paths, err := r.Resolve(Options{ScenarioID: "demo"})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if paths.LogsDir != filepath.Join("/Users/alice/Library/Logs", "vrooli", "demo") {
		t.Fatalf("LogsDir = %q", paths.LogsDir)
	}
	if paths.StateDir != filepath.Join("/Users/alice/Library/Application Support/State", "vrooli", "demo") {
		t.Fatalf("StateDir = %q", paths.StateDir)
	}
}

func TestResolveStorageRootOverride(t *testing.T) {
	t.Parallel()

	r := mustResolver(t, ResolverConfig{
		AppID:     "vrooli",
		RuntimeOS: "linux",
		EnvGet: mapEnv(map[string]string{
			envStorageRoot: "/srv/vrooli",
		}),
	})

	paths, err := r.Resolve(Options{ScenarioID: "demo"})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if paths.CacheDir != filepath.Join("/srv/vrooli", "cache", "vrooli", "demo") {
		t.Fatalf("CacheDir = %q", paths.CacheDir)
	}
}

func TestForClassUnknown(t *testing.T) {
	t.Parallel()

	_, err := (Paths{}).ForClass(Class("unknown"))
	if err == nil {
		t.Fatalf("expected class error")
	}
}
