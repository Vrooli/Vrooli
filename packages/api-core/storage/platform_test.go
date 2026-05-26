package storage

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

// runtimeHomeRoot resolves the expected runtime-home class root for `home`
// through the same authority the resolver uses, so assertions never hardcode the
// on-disk dir names (the contract owns them).
func runtimeHomeRoot(t *testing.T, home, key string) string {
	t.Helper()
	p, err := repocontract.RuntimeHomeEntryPath(home, key)
	if err != nil {
		t.Fatalf("RuntimeHomeEntryPath(%q, %q) error = %v", home, key, err)
	}
	return p
}

// TestResolveRuntimeHomeDefault (T-S1): with a home and no overrides, all five
// classes resolve under the operator runtime home, class-scoped to app/scenario.
func TestResolveRuntimeHomeDefault(t *testing.T) {
	t.Parallel()

	const home = "/abs/home/op"
	r := mustResolver(t, ResolverConfig{
		AppID:       "vrooli",
		EnvGet:      mapEnv(nil),
		UserHomeDir: func() (string, error) { return home, nil },
	})

	paths, err := r.Resolve(Options{ScenarioID: "swarm-manager"})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	cases := []struct {
		name string
		got  string
		key  string
	}{
		{"config", paths.ConfigDir, repocontract.HomeKeyConfig},
		{"data", paths.DataDir, repocontract.HomeKeyData},
		{"cache", paths.CacheDir, repocontract.HomeKeyCache},
		{"logs", paths.LogsDir, repocontract.HomeKeyLogs},
		{"state", paths.StateDir, repocontract.HomeKeyState},
	}
	for _, c := range cases {
		want := filepath.Join(runtimeHomeRoot(t, home, c.key), "vrooli", "swarm-manager")
		if c.got != want {
			t.Errorf("%s dir = %q, want %q", c.name, c.got, want)
		}
	}
}

// TestResolveNoXDGDefault (T-S2): regression lock for the drift removal. With XDG
// env vars set to sentinels and no Vrooli overrides, no resolved root contains
// the XDG sentinel — the XDG branch is gone.
func TestResolveNoXDGDefault(t *testing.T) {
	t.Parallel()

	const home = "/abs/home/op"
	r := mustResolver(t, ResolverConfig{
		AppID: "vrooli",
		EnvGet: mapEnv(map[string]string{
			"XDG_DATA_HOME":  "/xdg/share",
			"XDG_STATE_HOME": "/xdg/state",
			"XDG_CACHE_HOME": "/xdg/cache",
			"LOCALAPPDATA":   "/xdg/localappdata",
		}),
		// RuntimeOS/UserConfigDir/UserCacheDir set to sentinels to prove they
		// no longer steer the default.
		RuntimeOS:     "darwin",
		UserHomeDir:   func() (string, error) { return home, nil },
		UserConfigDir: func() (string, error) { return "/xdg/config", nil },
		UserCacheDir:  func() (string, error) { return "/xdg/cache", nil },
	})

	paths, err := r.Resolve(Options{ScenarioID: "demo"})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	root, err := repocontract.VrooliUserRoot(home)
	if err != nil {
		t.Fatalf("VrooliUserRoot() error = %v", err)
	}
	for _, p := range []string{paths.ConfigDir, paths.DataDir, paths.CacheDir, paths.LogsDir, paths.StateDir} {
		if strings.Contains(p, "/xdg/") || strings.Contains(p, "Library") || strings.Contains(p, "AppData") {
			t.Errorf("resolved root %q still contains an XDG/OS-specific segment", p)
		}
		if !strings.HasPrefix(p, root) {
			t.Errorf("resolved root %q not under runtime home %q", p, root)
		}
	}
}

// TestResolveStorageRootOverride (T-S3): overrides still win over the
// runtime-home default, at every precedence level.
func TestResolveStorageRootOverride(t *testing.T) {
	t.Parallel()

	const home = "/abs/home/op"

	// Global VROOLI_STORAGE_ROOT beats the runtime-home default.
	rGlobal := mustResolver(t, ResolverConfig{
		AppID:       "vrooli",
		EnvGet:      mapEnv(map[string]string{envStorageRoot: "/srv/vrooli"}),
		UserHomeDir: func() (string, error) { return home, nil },
	})
	paths, err := rGlobal.Resolve(Options{ScenarioID: "demo"})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if paths.CacheDir != filepath.Join("/srv/vrooli", "cache", "vrooli", "demo") {
		t.Fatalf("global override CacheDir = %q", paths.CacheDir)
	}

	// Per-class VROOLI_DATA_ROOT beats both default and global for its class.
	rClass := mustResolver(t, ResolverConfig{
		AppID: "vrooli",
		EnvGet: mapEnv(map[string]string{
			envStorageRoot: "/srv/vrooli",
			envDataRoot:    "/o/data",
		}),
		UserHomeDir: func() (string, error) { return home, nil },
	})
	// Per-class override only applies under the OS/runtime-home default path, not
	// when a global root override is set; mirror the resolver's documented
	// precedence by asserting global wins when both are present.
	pClass, err := rClass.Resolve(Options{ScenarioID: "demo"})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if pClass.DataDir != filepath.Join("/srv/vrooli", "data", "vrooli", "demo") {
		t.Fatalf("global beats per-class: DataDir = %q", pClass.DataDir)
	}

	// Options.RootOverride beats everything.
	pRO, err := rGlobal.Resolve(Options{ScenarioID: "demo", RootOverride: "/ro"})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if pRO.DataDir != filepath.Join("/ro", "data", "vrooli", "demo") {
		t.Fatalf("RootOverride DataDir = %q", pRO.DataDir)
	}
}

// TestResolvePerClassOverrideAboveDefault: with no global override, a per-class
// env override beats the runtime-home default for that class only.
func TestResolvePerClassOverrideAboveDefault(t *testing.T) {
	t.Parallel()

	const home = "/abs/home/op"
	r := mustResolver(t, ResolverConfig{
		AppID:       "vrooli",
		EnvGet:      mapEnv(map[string]string{envDataRoot: "/o/data"}),
		UserHomeDir: func() (string, error) { return home, nil },
	})
	paths, err := r.Resolve(Options{ScenarioID: "demo"})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if paths.DataDir != filepath.Join("/o/data", "vrooli", "demo") {
		t.Fatalf("per-class DataDir = %q", paths.DataDir)
	}
	// Other classes still resolve under the runtime home.
	want := filepath.Join(runtimeHomeRoot(t, home, repocontract.HomeKeyCache), "vrooli", "demo")
	if paths.CacheDir != want {
		t.Fatalf("CacheDir = %q, want %q", paths.CacheDir, want)
	}
}

// TestResolveContractFailureSurfaces (T-S4): when the runtime-home contract
// cannot be resolved, the default path returns an error rather than silently
// falling back to an XDG path. NOT parallel: it swaps the package seam.
func TestResolveContractFailureSurfaces(t *testing.T) {
	orig := runtimeHomeEntryPath
	runtimeHomeEntryPath = func(string, string) (string, error) {
		return "", errors.New("no contract discoverable")
	}
	defer func() { runtimeHomeEntryPath = orig }()

	r := mustResolver(t, ResolverConfig{
		AppID:       "vrooli",
		EnvGet:      mapEnv(nil),
		UserHomeDir: func() (string, error) { return "/abs/home/op", nil },
	})
	if _, err := r.Resolve(Options{ScenarioID: "demo"}); err == nil {
		t.Fatalf("expected contract-load failure to surface, got nil error")
	}

	// An override bypasses the contract entirely (the sanctioned escape hatch).
	if _, err := r.Resolve(Options{ScenarioID: "demo", RootOverride: "/ro"}); err != nil {
		t.Fatalf("override should bypass contract: %v", err)
	}
}

// TestResolveVPSProfileUnchanged (T-S5): the deliberate server FHS policy is
// untouched by the user-profile drift removal.
func TestResolveVPSProfileUnchanged(t *testing.T) {
	t.Parallel()

	r := mustResolver(t, ResolverConfig{
		AppID:   "vrooli",
		Profile: ProfileVPS,
		EnvGet:  mapEnv(nil),
	})
	paths, err := r.Resolve(Options{ScenarioID: "demo"})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	want := Paths{
		ConfigDir: filepath.Join("/etc", "vrooli", "demo"),
		DataDir:   filepath.Join("/var/lib", "vrooli", "demo"),
		CacheDir:  filepath.Join("/var/cache", "vrooli", "demo"),
		LogsDir:   filepath.Join("/var/log", "vrooli", "demo"),
		StateDir:  filepath.Join("/var/lib/vrooli-state", "vrooli", "demo"),
	}
	if paths != want {
		t.Fatalf("VPS paths = %+v, want %+v", paths, want)
	}
}

// TestResolveInvalidOverride (T-S6): a non-absolute override is rejected.
func TestResolveInvalidOverride(t *testing.T) {
	t.Parallel()

	r := mustResolver(t, ResolverConfig{
		AppID:  "vrooli",
		EnvGet: mapEnv(map[string]string{envStorageRoot: "relative/path"}),
	})
	_, err := r.Resolve(Options{ScenarioID: "demo"})
	if err == nil {
		t.Fatalf("expected invalid override error")
	}
	var sErr *Error
	if !errors.As(err, &sErr) || sErr.Kind != ErrInvalidInput {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
}

func TestForClassUnknown(t *testing.T) {
	t.Parallel()

	_, err := (Paths{}).ForClass(Class("unknown"))
	if err == nil {
		t.Fatalf("expected class error")
	}
}
