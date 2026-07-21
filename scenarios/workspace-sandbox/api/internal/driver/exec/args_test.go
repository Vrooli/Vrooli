package exec

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/google/uuid"

	"workspace-sandbox/internal/types"
)

func TestBuildBwrapArgs(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("bwrap tests require Linux")
	}

	sandbox := &types.Sandbox{
		ID:        uuid.New(),
		ScopePath: "/tmp/test",
		LowerDir:  "/tmp/lower",
		UpperDir:  "/tmp/upper",
		WorkDir:   "/tmp/work",
		MergedDir: "/tmp/merged",
	}

	cfg := DefaultBwrapConfig()
	args := BuildBwrapArgs(sandbox, cfg)

	for _, want := range []string{
		"--unshare-user",
		"--unshare-net", // network disabled by default
		"--die-with-parent",
		"--",
	} {
		if !contains(args, want) {
			t.Errorf("expected %s in args, got: %v", want, args)
		}
	}
	// workspace bind
	if !hasBind(args, "/tmp/merged", "/workspace") {
		t.Errorf("expected --bind /tmp/merged /workspace; got: %v", args)
	}
}

func TestBwrapNetworkConfig(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("bwrap tests require Linux")
	}

	sandbox := &types.Sandbox{
		ID:        uuid.New(),
		MergedDir: "/tmp/test",
		LowerDir:  "/tmp/lower",
	}

	cfg := DefaultBwrapConfig()
	args := BuildBwrapArgs(sandbox, cfg)
	if !contains(args, "--unshare-net") {
		t.Error("expected --unshare-net when AllowNetwork=false")
	}

	cfg.AllowNetwork = true
	args = BuildBwrapArgs(sandbox, cfg)
	if contains(args, "--unshare-net") {
		t.Error("did not expect --unshare-net when AllowNetwork=true")
	}
}

func TestBwrapPIDNamespaceConfig(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("bwrap tests require Linux")
	}

	sandbox := &types.Sandbox{
		ID:        uuid.New(),
		MergedDir: "/tmp/test",
		LowerDir:  "/tmp/lower",
	}

	cfg := DefaultBwrapConfig()
	args := BuildBwrapArgs(sandbox, cfg)
	if !contains(args, "--unshare-pid") {
		t.Error("expected --unshare-pid when SharePID=false")
	}

	cfg.SharePID = true
	args = BuildBwrapArgs(sandbox, cfg)
	if contains(args, "--unshare-pid") {
		t.Error("did not expect --unshare-pid when SharePID=true")
	}
}

func TestResourceLimitsHasLimits(t *testing.T) {
	tests := []struct {
		name     string
		limits   ResourceLimits
		expected bool
	}{
		{"all zero", ResourceLimits{}, false},
		{"memory set", ResourceLimits{MemoryLimitMB: 512}, true},
		{"cpu time set", ResourceLimits{CPUTimeSec: 60}, true},
		{"max processes set", ResourceLimits{MaxProcesses: 100}, true},
		{"max files set", ResourceLimits{MaxOpenFiles: 1024}, true},
		{"timeout only - not counted as prlimit", ResourceLimits{TimeoutSec: 300}, false},
		{"multiple limits", ResourceLimits{MemoryLimitMB: 512, CPUTimeSec: 60, MaxProcesses: 100}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.limits.HasLimits(); got != tt.expected {
				t.Errorf("HasLimits() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBuildPrlimitArgs(t *testing.T) {
	tests := []struct {
		name     string
		limits   ResourceLimits
		wantNil  bool
		wantArgs []string
	}{
		{"no limits returns nil", ResourceLimits{}, true, nil},
		{"memory limit", ResourceLimits{MemoryLimitMB: 512}, false, []string{"--as=536870912", "--"}},
		{"cpu time limit", ResourceLimits{CPUTimeSec: 60}, false, []string{"--cpu=60", "--"}},
		{"max processes", ResourceLimits{MaxProcesses: 100}, false, []string{"--nproc=100", "--"}},
		{"max open files", ResourceLimits{MaxOpenFiles: 1024}, false, []string{"--nofile=1024", "--"}},
		{"combined limits", ResourceLimits{MemoryLimitMB: 256, CPUTimeSec: 30, MaxProcesses: 50}, false, []string{"--as=268435456", "--cpu=30", "--nproc=50", "--"}},
		{"timeout not in prlimit args", ResourceLimits{TimeoutSec: 300}, true, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := BuildPrlimitArgs(tt.limits)
			if tt.wantNil {
				if args != nil {
					t.Errorf("expected nil, got %v", args)
				}
				return
			}
			if args == nil {
				t.Fatal("expected non-nil args")
			}
			for _, want := range tt.wantArgs {
				if !contains(args, want) {
					t.Errorf("expected %q in args, got: %v", want, args)
				}
			}
		})
	}
}

func TestBuildExecCommand(t *testing.T) {
	sandbox := &types.Sandbox{
		ID:        uuid.New(),
		ScopePath: "/tmp/test",
		LowerDir:  "/tmp/lower",
		UpperDir:  "/tmp/upper",
		WorkDir:   "/tmp/work",
		MergedDir: "/tmp/merged",
	}

	tests := []struct {
		name        string
		cfg         BwrapConfig
		cmd         string
		args        []string
		wantExe     string
		wantContain []string
	}{
		{
			name:        "no limits - bwrap directly",
			cfg:         DefaultBwrapConfig(),
			cmd:         "ls",
			args:        []string{"-la"},
			wantExe:     "bwrap",
			wantContain: []string{"--unshare-user", "--bind", "/tmp/merged", "/workspace", "--", "ls", "-la"},
		},
		{
			name: "with memory limit - prlimit wrapper",
			cfg: BwrapConfig{
				Hostname:       "sandbox",
				Env:            map[string]string{"PATH": "/usr/bin"},
				ResourceLimits: ResourceLimits{MemoryLimitMB: 512},
			},
			cmd:         "my-agent",
			args:        []string{"--task", "fix"},
			wantExe:     "prlimit",
			wantContain: []string{"--as=536870912", "--", "bwrap", "--unshare-user", "my-agent", "--task", "fix"},
		},
		{
			name: "with multiple limits",
			cfg: BwrapConfig{
				Hostname:       "sandbox",
				Env:            map[string]string{},
				ResourceLimits: ResourceLimits{MemoryLimitMB: 256, CPUTimeSec: 60},
			},
			cmd:         "build",
			args:        nil,
			wantExe:     "prlimit",
			wantContain: []string{"--as=268435456", "--cpu=60", "bwrap", "build"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exe, args := BuildExecCommand(sandbox, tt.cfg, tt.cmd, tt.args...)
			if exe != tt.wantExe {
				t.Errorf("executable = %q, want %q", exe, tt.wantExe)
			}
			for _, want := range tt.wantContain {
				if !contains(args, want) {
					t.Errorf("expected %q in args, got: %v", want, args)
				}
			}
		})
	}
}

// TestBuildBwrapArgs_BindsHomeOverlayAtHostPath guards the home-overlay
// contract: when Sandbox.HomeMergedDir is populated, BuildBwrapArgs binds
// it at cfg.HostHome inside the namespace so agent CLIs find their host
// config via the overlay's lower layer.
func TestBuildBwrapArgs_BindsHomeOverlayAtHostPath(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("bwrap tests require Linux")
	}

	fakeHome := t.TempDir()
	sandboxDir := t.TempDir()
	homeMerged := filepath.Join(sandboxDir, "home-merged")
	if err := os.MkdirAll(homeMerged, 0o755); err != nil {
		t.Fatalf("mkdir home-merged: %v", err)
	}

	sandbox := &types.Sandbox{
		ID:            uuid.New(),
		MergedDir:     filepath.Join(sandboxDir, "merged"),
		LowerDir:      "/tmp/lower",
		HomeMergedDir: homeMerged,
	}
	cfg := DefaultBwrapConfig()
	cfg.HostHome = fakeHome
	args := BuildBwrapArgs(sandbox, cfg)

	if !hasBind(args, homeMerged, fakeHome) {
		t.Errorf("expected --bind %s %s in args (home overlay must bind at host $HOME); got: %v",
			homeMerged, fakeHome, args)
	}
}

// TestBuildBwrapArgs_NoHomeBindWhenHomeMergedDirEmpty verifies the
// fallback path: a sandbox without a home overlay still produces valid
// bwrap args, just without the home bind.
func TestBuildBwrapArgs_NoHomeBindWhenHomeMergedDirEmpty(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("bwrap tests require Linux")
	}

	fakeHome := t.TempDir()

	sandbox := &types.Sandbox{
		ID:            uuid.New(),
		MergedDir:     "/tmp/test",
		LowerDir:      "/tmp/lower",
		HomeMergedDir: "",
	}
	cfg := DefaultBwrapConfig()
	cfg.HostHome = fakeHome
	args := BuildBwrapArgs(sandbox, cfg)

	for i := 0; i+2 < len(args); i++ {
		if args[i] == "--bind" && args[i+2] == fakeHome {
			t.Errorf("unexpected --bind <src> %s without HomeMergedDir set: src=%s", fakeHome, args[i+1])
		}
	}
}

// TestBuildBwrapArgs_PureNoEnvReads ensures BuildBwrapArgs ignores the
// process environment — every input must arrive via cfg. Sets HOME and
// MIRROR_PROJECT_ROOT in the process env, then asserts the argv shape
// matches the equivalent run with cfg-supplied values only.
func TestBuildBwrapArgs_PureNoEnvReads(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("bwrap tests require Linux")
	}

	t.Setenv("HOME", "/process/env/home")
	t.Setenv("WORKSPACE_SANDBOX_MIRROR_PROJECT_ROOT", "1")
	t.Setenv("VROOLI_ROOT", "/process/env/vrooli")

	sandbox := &types.Sandbox{
		ID:        uuid.New(),
		MergedDir: "/tmp/merged",
		LowerDir:  "/tmp/lower",
		// HomeMergedDir empty → no home bind
		// ProjectRoot empty → no mirror
	}

	cfg := DefaultBwrapConfig()
	// HostHome and MirrorProjectRoot deliberately left unset on cfg.
	args := BuildBwrapArgs(sandbox, cfg)

	// Process env had HOME=/process/env/home but cfg.HostHome is empty,
	// so no bind to that path should appear.
	for i := 0; i+2 < len(args); i++ {
		if args[i] == "--bind" && args[i+2] == "/process/env/home" {
			t.Errorf("BuildBwrapArgs leaked $HOME from process env: %v", args)
		}
	}
}

// TestBuildBwrapArgs_Golden pins the full argv output for representative
// (profile × home-overlay × mirror) combinations. Adding a new isolation
// behavior without updating this test is the failure mode we want.
func TestBuildBwrapArgs_Golden(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("bwrap tests require Linux")
	}

	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	sandbox := &types.Sandbox{
		ID:        id,
		MergedDir: "/sb/merged",
		LowerDir:  "/sb/lower",
	}

	cases := []struct {
		name string
		s    *types.Sandbox
		cfg  BwrapConfig
		want []string
	}{
		{
			name: "default-no-network",
			s:    sandbox,
			cfg:  BwrapConfig{Hostname: "sandbox"},
			want: []string{
				"--unshare-user", "--unshare-ipc", "--unshare-uts", "--unshare-cgroup",
				"--unshare-pid",
				"--unshare-net",
				"--die-with-parent",
				"--hostname", "sandbox",
				"--bind", "/sb/merged", "/workspace",
				"--ro-bind", "/sb/lower", "/workspace-readonly",
				"--proc", "/proc",
				"--dev", "/dev",
				"--tmpfs", "/tmp",
				"--chdir", "/workspace",
				"--",
			},
		},
		{
			name: "allow-network-no-pid-unshare-when-shared",
			s:    sandbox,
			cfg:  BwrapConfig{Hostname: "sandbox", AllowNetwork: true, SharePID: true},
			want: []string{
				"--unshare-user", "--unshare-ipc", "--unshare-uts", "--unshare-cgroup",
				// no --unshare-pid (SharePID=true), no --unshare-net (AllowNetwork=true)
				"--die-with-parent",
				"--hostname", "sandbox",
				"--bind", "/sb/merged", "/workspace",
				"--ro-bind", "/sb/lower", "/workspace-readonly",
				"--proc", "/proc",
				"--dev", "/dev",
				"--tmpfs", "/tmp",
				"--chdir", "/workspace",
				"--",
			},
		},
		{
			name: "with-home-overlay",
			s: &types.Sandbox{
				ID:            id,
				MergedDir:     "/sb/merged",
				LowerDir:      "/sb/lower",
				HomeMergedDir: "/sb/home-merged",
			},
			cfg: BwrapConfig{Hostname: "sandbox", HostHome: "/h/u"},
			want: []string{
				"--unshare-user", "--unshare-ipc", "--unshare-uts", "--unshare-cgroup",
				"--unshare-pid",
				"--unshare-net",
				"--die-with-parent",
				"--hostname", "sandbox",
				"--bind", "/sb/merged", "/workspace",
				"--dir", "/h",
				"--dir", "/h/u",
				"--bind", "/sb/home-merged", "/h/u",
				// Merged dir re-bound at its own host path AFTER the home
				// overlay bind so a leaked host-side merged path resolves to
				// the real workspace, not the home overlay's empty shadow.
				"--dir", "/sb",
				"--dir", "/sb/merged",
				"--bind", "/sb/merged", "/sb/merged",
				"--ro-bind", "/sb/lower", "/workspace-readonly",
				"--proc", "/proc",
				"--dev", "/dev",
				"--tmpfs", "/tmp",
				"--chdir", "/workspace",
				"--",
			},
		},
		{
			// Mask paths are emitted as trailing tmpfs over-binds; entries
			// that would cover the workspace, merged dir, or lower layer are
			// skipped, and the list is sorted for argv determinism.
			name: "with-mask-paths",
			s: &types.Sandbox{
				ID:            id,
				MergedDir:     "/sb/merged",
				LowerDir:      "/sb/lower",
				HomeMergedDir: "/sb/home-merged",
			},
			cfg: BwrapConfig{
				Hostname: "sandbox",
				HostHome: "/h/u",
				MaskPaths: []string{
					"/h/u/other-checkouts",
					"/h/u/.codex-worktrees",
					"/workspace",           // would cover workspace → skipped
					"/sb/merged/sub",       // under merged dir → skipped
					"/workspace-readonly",  // lower layer → skipped
					"relative/not-allowed", // not absolute → skipped
				},
			},
			want: []string{
				"--unshare-user", "--unshare-ipc", "--unshare-uts", "--unshare-cgroup",
				"--unshare-pid",
				"--unshare-net",
				"--die-with-parent",
				"--hostname", "sandbox",
				"--bind", "/sb/merged", "/workspace",
				"--dir", "/h",
				"--dir", "/h/u",
				"--bind", "/sb/home-merged", "/h/u",
				"--dir", "/sb",
				"--dir", "/sb/merged",
				"--bind", "/sb/merged", "/sb/merged",
				"--ro-bind", "/sb/lower", "/workspace-readonly",
				"--tmpfs", "/h/u/.codex-worktrees",
				"--tmpfs", "/h/u/other-checkouts",
				"--proc", "/proc",
				"--dev", "/dev",
				"--tmpfs", "/tmp",
				"--chdir", "/workspace",
				"--",
			},
		},
		{
			name: "with-mirror-project-root",
			s: &types.Sandbox{
				ID:          id,
				MergedDir:   "/sb/merged",
				LowerDir:    "/sb/lower",
				ProjectRoot: "/proj/repo",
			},
			cfg: BwrapConfig{Hostname: "sandbox", MirrorProjectRoot: true},
			want: []string{
				"--unshare-user", "--unshare-ipc", "--unshare-uts", "--unshare-cgroup",
				"--unshare-pid",
				"--unshare-net",
				"--die-with-parent",
				"--hostname", "sandbox",
				"--bind", "/sb/merged", "/workspace",
				"--dir", "/proj",
				"--dir", "/proj/repo",
				"--bind", "/sb/merged", "/proj/repo",
				"--ro-bind", "/sb/lower", "/workspace-readonly",
				"--proc", "/proc",
				"--dev", "/dev",
				"--tmpfs", "/tmp",
				"--chdir", "/workspace",
				"--",
			},
		},
		{
			name: "with-binds-sorted",
			s:    sandbox,
			cfg: BwrapConfig{
				Hostname: "sandbox",
				ReadOnlyBinds: map[string]string{
					"/usr": "/usr",
					"/bin": "/bin",
				},
				ReadWriteBinds: map[string]string{
					"/data": "/data",
				},
			},
			want: []string{
				"--unshare-user", "--unshare-ipc", "--unshare-uts", "--unshare-cgroup",
				"--unshare-pid",
				"--unshare-net",
				"--die-with-parent",
				"--hostname", "sandbox",
				"--bind", "/sb/merged", "/workspace",
				"--ro-bind", "/sb/lower", "/workspace-readonly",
				// sorted-by-source: /bin before /usr
				"--ro-bind", "/bin", "/bin",
				"--ro-bind", "/usr", "/usr",
				"--bind", "/data", "/data",
				"--proc", "/proc",
				"--dev", "/dev",
				"--tmpfs", "/tmp",
				"--chdir", "/workspace",
				"--",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildBwrapArgs(tc.s, tc.cfg)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("BuildBwrapArgs golden mismatch.\n got: %v\nwant: %v", got, tc.want)
			}
		})
	}
}

func TestDefaultBwrapConfig(t *testing.T) {
	cfg := DefaultBwrapConfig()
	if cfg.AllowNetwork {
		t.Error("default config should have AllowNetwork=false")
	}
	if cfg.AllowDevices {
		t.Error("default config should have AllowDevices=false")
	}
	if cfg.SharePID {
		t.Error("default config should have SharePID=false")
	}
	if cfg.Hostname != "sandbox" {
		t.Errorf("default hostname should be 'sandbox', got '%s'", cfg.Hostname)
	}
	if cfg.Env == nil {
		t.Error("default Env must be a non-nil map")
	}
	if cfg.ReadOnlyBinds == nil {
		t.Error("default ReadOnlyBinds must be a non-nil map")
	}
	if cfg.ReadWriteBinds == nil {
		t.Error("default ReadWriteBinds must be a non-nil map")
	}
}

func TestApplyIsolationProfile_RejectsNil(t *testing.T) {
	cfg := DefaultBwrapConfig()
	if err := ApplyIsolationProfile(&cfg, nil); !errors.Is(err, ErrIsolationProfileRequired) {
		t.Errorf("ApplyIsolationProfile(nil) = %v, want ErrIsolationProfileRequired", err)
	}
}

func TestApplyIsolationProfile_AppliesNetworkAndHostname(t *testing.T) {
	cfg := DefaultBwrapConfig()
	p := &IsolationProfile{
		ID:            "p",
		NetworkAccess: "localhost",
		Hostname:      "custom",
	}
	if err := ApplyIsolationProfile(&cfg, p); err != nil {
		t.Fatalf("ApplyIsolationProfile: %v", err)
	}
	if !cfg.AllowNetwork {
		t.Error("expected AllowNetwork=true for NetworkAccess=localhost")
	}
	if cfg.Hostname != "custom" {
		t.Errorf("expected Hostname=custom, got %q", cfg.Hostname)
	}
}

func TestApplyIsolationProfile_MaskPathsExpandWithoutStat(t *testing.T) {
	cfg := DefaultBwrapConfig()
	cfg.HostHome = "/h/u"
	p := &IsolationProfile{
		MaskPaths: []string{
			"$HOME/.codex-worktrees",                  // expands; must survive even if absent on host
			"$UNKNOWN_PLACEHOLDER/x",                  // unresolvable → dropped
			"relative/path",                           // not absolute → dropped
			"/abs/other-checkouts/../other-checkouts", // cleaned
		},
	}
	if err := ApplyIsolationProfile(&cfg, p); err != nil {
		t.Fatalf("ApplyIsolationProfile: %v", err)
	}
	want := []string{"/h/u/.codex-worktrees", "/abs/other-checkouts"}
	if len(cfg.MaskPaths) != len(want) {
		t.Fatalf("MaskPaths = %v; want %v", cfg.MaskPaths, want)
	}
	for i := range want {
		if cfg.MaskPaths[i] != want[i] {
			t.Errorf("MaskPaths[%d] = %q; want %q", i, cfg.MaskPaths[i], want[i])
		}
	}

	// Empty $HOME must drop $HOME-based masks rather than emit a bogus
	// root-anchored path.
	cfgNoHome := DefaultBwrapConfig()
	cfgNoHome.HostHome = ""
	t.Setenv("HOME", "")
	if err := ApplyIsolationProfile(&cfgNoHome, &IsolationProfile{MaskPaths: []string{"$HOME/.codex-worktrees"}}); err != nil {
		t.Fatalf("ApplyIsolationProfile: %v", err)
	}
	if len(cfgNoHome.MaskPaths) != 0 {
		t.Errorf("empty $HOME: MaskPaths = %v; want empty", cfgNoHome.MaskPaths)
	}
}

func TestApplyIsolationProfile_BindsAreMapped(t *testing.T) {
	srcDir := t.TempDir()
	cfg := DefaultBwrapConfig()
	p := &IsolationProfile{
		ReadOnlyBinds:  map[string]string{srcDir: "/inside-ro"},
		ReadWriteBinds: map[string]string{srcDir: "/inside-rw"},
	}
	if err := ApplyIsolationProfile(&cfg, p); err != nil {
		t.Fatalf("ApplyIsolationProfile: %v", err)
	}
	if cfg.ReadOnlyBinds[srcDir] != "/inside-ro" {
		t.Errorf("expected ReadOnlyBinds[%s] = /inside-ro, got %q", srcDir, cfg.ReadOnlyBinds[srcDir])
	}
	if cfg.ReadWriteBinds[srcDir] != "/inside-rw" {
		t.Errorf("expected ReadWriteBinds[%s] = /inside-rw, got %q", srcDir, cfg.ReadWriteBinds[srcDir])
	}
}

func TestApplyIsolationProfile_SkipsMissingSource(t *testing.T) {
	cfg := DefaultBwrapConfig()
	p := &IsolationProfile{
		ReadOnlyBinds: map[string]string{"/this/path/should/not/exist/zzqyx": "/inside"},
	}
	if err := ApplyIsolationProfile(&cfg, p); err != nil {
		t.Fatalf("ApplyIsolationProfile: %v", err)
	}
	if len(cfg.ReadOnlyBinds) != 0 {
		t.Errorf("expected missing-source bind to be skipped, got: %v", cfg.ReadOnlyBinds)
	}
}

func TestApplyIsolationProfile_ExpandsHomePlaceholder(t *testing.T) {
	homeDir := t.TempDir()
	cfg := DefaultBwrapConfig()
	cfg.HostHome = homeDir
	p := &IsolationProfile{
		ReadOnlyBinds: map[string]string{"$HOME": "/inside-home"},
	}
	if err := ApplyIsolationProfile(&cfg, p); err != nil {
		t.Fatalf("ApplyIsolationProfile: %v", err)
	}
	if cfg.ReadOnlyBinds[homeDir] != "/inside-home" {
		t.Errorf("expected $HOME placeholder expanded to %s, got: %v", homeDir, cfg.ReadOnlyBinds)
	}
}

func TestApplyIsolationProfile_MergesVrooliAwarePath(t *testing.T) {
	homeDir := t.TempDir()
	cfg := DefaultBwrapConfig()
	cfg.HostHome = homeDir
	cfg.Env["PATH"] = "/workspace:/custom/bin"
	p := &IsolationProfile{
		ID: "vrooli-aware",
		Environment: map[string]string{
			"PATH": "$HOME/.vrooli/bin:$HOME/go/bin:$HOME/.local/bin:/usr/local/bin:/usr/bin:/bin",
			"HOME": "$HOME",
		},
	}

	if err := ApplyIsolationProfile(&cfg, p); err != nil {
		t.Fatalf("ApplyIsolationProfile: %v", err)
	}

	want := []string{
		filepath.Join(homeDir, ".vrooli/bin"),
		filepath.Join(homeDir, "go/bin"),
		filepath.Join(homeDir, ".local/bin"),
		"/usr/local/bin",
		"/usr/bin",
		"/bin",
		"/workspace",
		"/custom/bin",
	}
	if got := strings.Split(cfg.Env["PATH"], ":"); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("PATH entries:\n got %q\nwant %q", got, want)
	}
	if got := cfg.Env["HOME"]; got != homeDir {
		t.Fatalf("HOME = %q, want %q", got, homeDir)
	}
}

// TestApplyIsolationProfile_RelativeHostHomeNotExpanded pins the guard
// that a non-absolute HostHome (e.g. HOME=.home inherited from a sandboxed
// parent) never expands into a workspace-relative $HOME. With
// `--chdir <workspace>` such a HOME would otherwise materialize
// `<workspace>/.home` from every $HOME-relative write. The bind side
// already refuses a relative HostHome (see args.go); the env side must
// match — $HOME expands to empty rather than `.home`.
func TestApplyIsolationProfile_RelativeHostHomeNotExpanded(t *testing.T) {
	cfg := DefaultBwrapConfig()
	cfg.HostHome = ".home"
	p := &IsolationProfile{
		ID: "vrooli-aware",
		Environment: map[string]string{
			"HOME": "$HOME",
			"PATH": "$HOME/.vrooli/bin:/usr/bin",
		},
	}

	if err := ApplyIsolationProfile(&cfg, p); err != nil {
		t.Fatalf("ApplyIsolationProfile: %v", err)
	}

	if got := cfg.Env["HOME"]; strings.Contains(got, ".home") {
		t.Fatalf("HOME = %q; relative host home must not expand into a workspace-relative path", got)
	}
	for _, entry := range strings.Split(cfg.Env["PATH"], ":") {
		if strings.HasPrefix(entry, ".home") || entry == ".home/.vrooli/bin" {
			t.Fatalf("PATH entry %q leaked relative host home", entry)
		}
	}
}

func TestApplyIsolationProfile_DeduplicatesPathEntries(t *testing.T) {
	homeDir := t.TempDir()
	cfg := DefaultBwrapConfig()
	cfg.HostHome = homeDir
	cfg.Env["PATH"] = filepath.Join(homeDir, ".vrooli/bin") + ":/usr/bin:/extra/bin"
	p := &IsolationProfile{
		ID: "vrooli-aware",
		Environment: map[string]string{
			"PATH": "$HOME/.vrooli/bin:$HOME/go/bin:/usr/bin:/bin",
		},
	}

	if err := ApplyIsolationProfile(&cfg, p); err != nil {
		t.Fatalf("ApplyIsolationProfile: %v", err)
	}

	want := []string{
		filepath.Join(homeDir, ".vrooli/bin"),
		filepath.Join(homeDir, "go/bin"),
		"/usr/bin",
		"/bin",
		"/extra/bin",
	}
	if got := strings.Split(cfg.Env["PATH"], ":"); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("PATH entries:\n got %q\nwant %q", got, want)
	}
}

// --- helpers ---

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func hasBind(args []string, wantSrc, wantDst string) bool {
	for i := 0; i+2 < len(args); i++ {
		if args[i] == "--bind" && args[i+1] == wantSrc && args[i+2] == wantDst {
			return true
		}
	}
	return false
}
