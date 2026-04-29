package exec

import (
	"os"
	"path/filepath"
	"runtime"
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

	tests := []struct {
		name     string
		expected string
		find     func([]string) bool
	}{
		{"unshare-user", "--unshare-user", func(args []string) bool { return contains(args, "--unshare-user") }},
		{"unshare-net (network disabled)", "--unshare-net", func(args []string) bool { return contains(args, "--unshare-net") }},
		{"die-with-parent", "--die-with-parent", func(args []string) bool { return contains(args, "--die-with-parent") }},
		{"workspace bind", "--bind /tmp/merged", func(args []string) bool {
			for i, arg := range args {
				if arg == "--bind" && i+2 < len(args) && args[i+1] == "/tmp/merged" {
					return true
				}
			}
			return false
		}},
		{"separator", "--", func(args []string) bool { return contains(args, "--") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.find(args) {
				t.Errorf("expected %s in args, got: %v", tt.expected, args)
			}
		})
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
			name:    "no limits - bwrap directly",
			cfg:     DefaultBwrapConfig(),
			cmd:     "ls",
			args:    []string{"-la"},
			wantExe: "bwrap",
			wantContain: []string{"--unshare-user", "--bind", "/tmp/merged", "/workspace", "--", "ls", "-la"},
		},
		{
			name: "with memory limit - prlimit wrapper",
			cfg: BwrapConfig{
				IsolationLevel: IsolationFull,
				Hostname:       "sandbox",
				Env:            map[string]string{"PATH": "/usr/bin"},
				ResourceLimits: ResourceLimits{MemoryLimitMB: 512},
			},
			cmd:     "my-agent",
			args:    []string{"--task", "fix"},
			wantExe: "prlimit",
			wantContain: []string{"--as=536870912", "--", "bwrap", "--unshare-user", "my-agent", "--task", "fix"},
		},
		{
			name: "with multiple limits",
			cfg: BwrapConfig{
				Hostname:       "sandbox",
				Env:            map[string]string{},
				ResourceLimits: ResourceLimits{MemoryLimitMB: 256, CPUTimeSec: 60},
			},
			cmd:     "build",
			args:    nil,
			wantExe: "prlimit",
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

func TestVrooliAwareIsolation(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("bwrap tests require Linux")
	}

	sandbox := &types.Sandbox{
		ID:        uuid.New(),
		MergedDir: "/tmp/test",
		LowerDir:  "/tmp/lower",
	}

	cfgFull := DefaultBwrapConfig()
	cfgFull.IsolationLevel = IsolationFull
	argsFull := BuildBwrapArgs(sandbox, cfgFull)
	if !contains(argsFull, "--unshare-net") {
		t.Error("full isolation should include --unshare-net")
	}

	cfgVrooli := DefaultBwrapConfig()
	cfgVrooli.IsolationLevel = IsolationVrooliAware
	argsVrooli := BuildBwrapArgs(sandbox, cfgVrooli)
	if contains(argsVrooli, "--unshare-net") {
		t.Error("vrooli-aware isolation should NOT include --unshare-net")
	}
}

// TestBuildBwrapArgs_BindsHomeOverlayAtHostPath guards the home-overlay
// contract: when Sandbox.HomeMergedDir is populated, BuildBwrapArgs binds
// it at the host $HOME path inside the namespace so agent CLIs find their
// host config via the overlay's lower layer.
func TestBuildBwrapArgs_BindsHomeOverlayAtHostPath(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("bwrap tests require Linux")
	}

	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)

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
	cfg.IsolationLevel = IsolationVrooliAware
	args := BuildBwrapArgs(sandbox, cfg)

	if !hasBind(args, homeMerged, fakeHome) {
		t.Errorf("expected --bind %s %s in vrooli-aware args (home overlay must bind at host $HOME); got: %v",
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
	t.Setenv("HOME", fakeHome)

	sandbox := &types.Sandbox{
		ID:            uuid.New(),
		MergedDir:     "/tmp/test",
		LowerDir:      "/tmp/lower",
		HomeMergedDir: "",
	}
	cfg := DefaultBwrapConfig()
	cfg.IsolationLevel = IsolationVrooliAware
	args := BuildBwrapArgs(sandbox, cfg)

	for i := 0; i+2 < len(args); i++ {
		if args[i] == "--bind" && args[i+2] == fakeHome {
			t.Errorf("unexpected --bind <src> %s without HomeMergedDir set: src=%s", fakeHome, args[i+1])
		}
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
	if cfg.Env["PATH"] == "" {
		t.Error("default config should set PATH")
	}
}

func TestDefaultBwrapConfigIncludesIsolationLevel(t *testing.T) {
	cfg := DefaultBwrapConfig()
	if cfg.IsolationLevel != IsolationFull {
		t.Errorf("default IsolationLevel = %q, want %q", cfg.IsolationLevel, IsolationFull)
	}
}

func TestGetVrooliEnvVars(t *testing.T) {
	origVrooliRoot := os.Getenv("VROOLI_ROOT")
	origVrooliEnv := os.Getenv("VROOLI_ENV")
	origApiManager := os.Getenv("API_MANAGER_URL")
	defer func() {
		restoreEnv("VROOLI_ROOT", origVrooliRoot)
		restoreEnv("VROOLI_ENV", origVrooliEnv)
		restoreEnv("API_MANAGER_URL", origApiManager)
	}()

	os.Unsetenv("VROOLI_ROOT")
	os.Unsetenv("VROOLI_ENV")
	os.Unsetenv("API_MANAGER_URL")

	vars := GetVrooliEnvVars()
	if _, ok := vars["VROOLI_ROOT"]; ok {
		t.Error("VROOLI_ROOT should not be set when environment variable is empty")
	}

	os.Setenv("VROOLI_ROOT", "/home/user/Vrooli")
	vars = GetVrooliEnvVars()
	if vars["VROOLI_ROOT"] != "/vrooli" {
		t.Errorf("VROOLI_ROOT = %q, want %q", vars["VROOLI_ROOT"], "/vrooli")
	}

	os.Setenv("VROOLI_ENV", "development")
	os.Setenv("API_MANAGER_URL", "http://localhost:8110")
	vars = GetVrooliEnvVars()
	if vars["VROOLI_ENV"] != "development" {
		t.Errorf("VROOLI_ENV = %q, want %q", vars["VROOLI_ENV"], "development")
	}
	if vars["API_MANAGER_URL"] != "http://localhost:8110" {
		t.Errorf("API_MANAGER_URL = %q, want %q", vars["API_MANAGER_URL"], "http://localhost:8110")
	}
}

func TestApplyVrooliAwareConfig(t *testing.T) {
	origVrooliRoot := os.Getenv("VROOLI_ROOT")
	defer restoreEnv("VROOLI_ROOT", origVrooliRoot)
	os.Setenv("VROOLI_ROOT", "/home/user/Vrooli")

	cfg := DefaultBwrapConfig()
	if cfg.IsolationLevel != IsolationFull {
		t.Fatalf("default IsolationLevel = %q, want %q", cfg.IsolationLevel, IsolationFull)
	}
	if cfg.AllowNetwork {
		t.Fatal("default AllowNetwork = true, want false")
	}

	ApplyVrooliAwareConfig(&cfg)

	if cfg.IsolationLevel != IsolationVrooliAware {
		t.Errorf("IsolationLevel = %q, want %q", cfg.IsolationLevel, IsolationVrooliAware)
	}
	if !cfg.AllowNetwork {
		t.Error("AllowNetwork = false, want true for Vrooli-aware isolation")
	}
	if cfg.Env["VROOLI_ROOT"] != "/vrooli" {
		t.Errorf("Env[VROOLI_ROOT] = %q, want %q", cfg.Env["VROOLI_ROOT"], "/vrooli")
	}
}

func TestApplyVrooliAwareConfigPreservesExistingEnv(t *testing.T) {
	cfg := DefaultBwrapConfig()
	cfg.Env["CUSTOM_VAR"] = "custom_value"

	ApplyVrooliAwareConfig(&cfg)

	if cfg.Env["CUSTOM_VAR"] != "custom_value" {
		t.Errorf("Env[CUSTOM_VAR] = %q, want %q", cfg.Env["CUSTOM_VAR"], "custom_value")
	}
	if cfg.Env["PATH"] == "" {
		t.Error("Env[PATH] should not be empty")
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

func restoreEnv(key, value string) {
	if value == "" {
		os.Unsetenv(key)
	} else {
		os.Setenv(key, value)
	}
}
