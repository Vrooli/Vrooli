package storage

import (
	"path/filepath"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

// TestResolveDefaultIsRuntimeHomeOnLinux proves the user-profile default resolves
// under the operator runtime home and is OS-agnostic: RuntimeOS=linux and XDG env
// sentinels do not steer it (the XDG branch was removed). Complements the
// platform_test.go T-S1/T-S2 coverage from the resolver entry point.
func TestResolveDefaultIsRuntimeHomeOnLinux(t *testing.T) {
	t.Parallel()

	const home = "/home/test"
	r := mustResolver(t, ResolverConfig{
		AppID:     "vrooli",
		Profile:   ProfileAuto,
		RuntimeOS: "linux",
		EnvGet: mapEnv(map[string]string{
			"XDG_DATA_HOME":  "/xdg/data",
			"XDG_STATE_HOME": "/xdg/state",
		}),
		UserHomeDir: func() (string, error) { return home, nil },
	})

	paths, err := r.Resolve(Options{ScenarioID: "landing-page-business-suite"})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	for cls, got := range map[string]string{
		repocontract.HomeKeyConfig: paths.ConfigDir,
		repocontract.HomeKeyData:   paths.DataDir,
		repocontract.HomeKeyCache:  paths.CacheDir,
		repocontract.HomeKeyLogs:   paths.LogsDir,
		repocontract.HomeKeyState:  paths.StateDir,
	} {
		root, err := repocontract.RuntimeHomeEntryPath(home, cls)
		if err != nil {
			t.Fatalf("RuntimeHomeEntryPath(%q) error = %v", cls, err)
		}
		want := filepath.Join(root, "vrooli", "landing-page-business-suite")
		if got != want {
			t.Errorf("%s dir = %q, want %q", cls, got, want)
		}
		if strings.Contains(got, "/xdg/") {
			t.Errorf("%s dir = %q still contains XDG segment", cls, got)
		}
	}
}

func TestResolveVPSProfile(t *testing.T) {
	t.Parallel()

	r := mustResolver(t, ResolverConfig{
		AppID:     "vrooli",
		Profile:   ProfileVPS,
		RuntimeOS: "linux",
		EnvGet:    mapEnv(nil),
	})

	paths, err := r.Resolve(Options{ScenarioID: "lpbs"})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if paths.ConfigDir != filepath.Join("/etc", "vrooli", "lpbs") {
		t.Fatalf("ConfigDir = %q", paths.ConfigDir)
	}
	if paths.DataDir != filepath.Join("/var/lib", "vrooli", "lpbs") {
		t.Fatalf("DataDir = %q", paths.DataDir)
	}
	if paths.CacheDir != filepath.Join("/var/cache", "vrooli", "lpbs") {
		t.Fatalf("CacheDir = %q", paths.CacheDir)
	}
	if paths.LogsDir != filepath.Join("/var/log", "vrooli", "lpbs") {
		t.Fatalf("LogsDir = %q", paths.LogsDir)
	}
	if paths.StateDir != filepath.Join("/var/lib/vrooli-state", "vrooli", "lpbs") {
		t.Fatalf("StateDir = %q", paths.StateDir)
	}
}

func TestResolveRootOverride(t *testing.T) {
	t.Parallel()

	r := mustResolver(t, ResolverConfig{AppID: "vrooli", EnvGet: mapEnv(nil)})
	paths, err := r.Resolve(Options{ScenarioID: "demo", RootOverride: "/tmp/vrooli-root"})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if paths.ConfigDir != filepath.Join("/tmp/vrooli-root", "config", "vrooli", "demo") {
		t.Fatalf("ConfigDir = %q", paths.ConfigDir)
	}
	if paths.DataDir != filepath.Join("/tmp/vrooli-root", "data", "vrooli", "demo") {
		t.Fatalf("DataDir = %q", paths.DataDir)
	}
}

func TestResolveEnvOverridePerClass(t *testing.T) {
	t.Parallel()

	r := mustResolver(t, ResolverConfig{
		AppID:     "vrooli",
		RuntimeOS: "linux",
		EnvGet: mapEnv(map[string]string{
			envDataRoot:  "/srv/data",
			envLogsRoot:  "/srv/logs",
			envStateRoot: "/srv/state",
		}),
		UserHomeDir:   func() (string, error) { return "/home/test", nil },
		UserConfigDir: func() (string, error) { return "/home/test/.config", nil },
		UserCacheDir:  func() (string, error) { return "/home/test/.cache", nil },
	})

	paths, err := r.Resolve(Options{ScenarioID: "demo"})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if paths.DataDir != filepath.Join("/srv/data", "vrooli", "demo") {
		t.Fatalf("DataDir = %q", paths.DataDir)
	}
	if paths.LogsDir != filepath.Join("/srv/logs", "vrooli", "demo") {
		t.Fatalf("LogsDir = %q", paths.LogsDir)
	}
	if paths.StateDir != filepath.Join("/srv/state", "vrooli", "demo") {
		t.Fatalf("StateDir = %q", paths.StateDir)
	}
}

func TestResolveRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	r := mustResolver(t, ResolverConfig{AppID: "vrooli", EnvGet: mapEnv(nil)})

	_, err := r.Resolve(Options{ScenarioID: "../bad"})
	if err == nil {
		t.Fatalf("expected error for invalid scenario id")
	}
	if !strings.Contains(err.Error(), "invalid scenario id") {
		t.Fatalf("error = %v", err)
	}

	_, err = r.Resolve(Options{ScenarioID: "ok", RootOverride: "relative/path"})
	if err == nil {
		t.Fatalf("expected error for relative root override")
	}
}

func TestPathGuardsTraversal(t *testing.T) {
	t.Parallel()

	r := mustResolver(t, ResolverConfig{AppID: "vrooli", EnvGet: mapEnv(nil)})

	_, err := r.Path(Options{ScenarioID: "demo", RootOverride: "/tmp/test-root"}, ClassData, "../../etc/passwd")
	if err == nil {
		t.Fatalf("expected traversal error")
	}

	_, err = r.Path(Options{ScenarioID: "demo", RootOverride: "/tmp/test-root"}, ClassData, "/etc/passwd")
	if err == nil {
		t.Fatalf("expected absolute path error")
	}

	p, err := r.Path(Options{ScenarioID: "demo", RootOverride: "/tmp/test-root"}, ClassData, "uploads/image.png")
	if err != nil {
		t.Fatalf("Path() error = %v", err)
	}
	if p != filepath.Join("/tmp/test-root", "data", "vrooli", "demo", "uploads", "image.png") {
		t.Fatalf("Path = %q", p)
	}
}

func TestNewResolverRejectsBadAppID(t *testing.T) {
	t.Parallel()

	_, err := NewResolver(ResolverConfig{AppID: "bad/app"})
	if err == nil {
		t.Fatalf("expected app id validation error")
	}
}

func mustResolver(t *testing.T, cfg ResolverConfig) *Resolver {
	t.Helper()
	r, err := NewResolver(cfg)
	if err != nil {
		t.Fatalf("NewResolver() error = %v", err)
	}
	return r
}

func mapEnv(values map[string]string) func(string) string {
	if values == nil {
		values = map[string]string{}
	}
	return func(key string) string {
		return values[key]
	}
}
