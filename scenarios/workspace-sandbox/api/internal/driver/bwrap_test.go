package driver

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"workspace-sandbox/internal/types"
)

// [REQ:REQ-P0-004] Test bubblewrap configuration building
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
	args := buildBwrapArgs(sandbox, cfg)

	// Verify essential arguments are present
	tests := []struct {
		name     string
		expected string
		find     func([]string) bool
	}{
		{
			name:     "unshare-user",
			expected: "--unshare-user",
			find:     func(args []string) bool { return contains(args, "--unshare-user") },
		},
		{
			name:     "unshare-net (network disabled)",
			expected: "--unshare-net",
			find:     func(args []string) bool { return contains(args, "--unshare-net") },
		},
		{
			name:     "die-with-parent",
			expected: "--die-with-parent",
			find:     func(args []string) bool { return contains(args, "--die-with-parent") },
		},
		{
			name:     "workspace bind",
			expected: "--bind /tmp/merged",
			find: func(args []string) bool {
				for i, arg := range args {
					if arg == "--bind" && i+2 < len(args) && args[i+1] == "/tmp/merged" {
						return true
					}
				}
				return false
			},
		},
		{
			name:     "separator",
			expected: "--",
			find:     func(args []string) bool { return contains(args, "--") },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.find(args) {
				t.Errorf("expected %s in args, got: %v", tt.expected, args)
			}
		})
	}
}

// [REQ:REQ-P0-004] Test network isolation toggle
func TestBwrapNetworkConfig(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("bwrap tests require Linux")
	}

	sandbox := &types.Sandbox{
		ID:        uuid.New(),
		MergedDir: "/tmp/test",
		LowerDir:  "/tmp/lower",
	}

	// Network disabled (default)
	cfg := DefaultBwrapConfig()
	args := buildBwrapArgs(sandbox, cfg)
	if !contains(args, "--unshare-net") {
		t.Error("expected --unshare-net when AllowNetwork=false")
	}

	// Network enabled
	cfg.AllowNetwork = true
	args = buildBwrapArgs(sandbox, cfg)
	if contains(args, "--unshare-net") {
		t.Error("did not expect --unshare-net when AllowNetwork=true")
	}
}

// [REQ:REQ-P0-004] Test PID namespace isolation toggle
func TestBwrapPIDNamespaceConfig(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("bwrap tests require Linux")
	}

	sandbox := &types.Sandbox{
		ID:        uuid.New(),
		MergedDir: "/tmp/test",
		LowerDir:  "/tmp/lower",
	}

	// PID isolated (default)
	cfg := DefaultBwrapConfig()
	args := buildBwrapArgs(sandbox, cfg)
	if !contains(args, "--unshare-pid") {
		t.Error("expected --unshare-pid when SharePID=false")
	}

	// PID shared
	cfg.SharePID = true
	args = buildBwrapArgs(sandbox, cfg)
	if contains(args, "--unshare-pid") {
		t.Error("did not expect --unshare-pid when SharePID=true")
	}
}

// [REQ:REQ-P0-004] Test bwrap availability check
func TestIsBwrapAvailable(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("bwrap tests require Linux")
	}

	ctx := context.Background()
	available, version, err := IsBwrapAvailable(ctx)

	// Check if bwrap is installed
	_, lookErr := exec.LookPath("bwrap")
	if lookErr != nil {
		if available {
			t.Error("IsBwrapAvailable returned true but bwrap not in PATH")
		}
		t.Logf("bwrap not installed (expected in CI): %v", err)
		return
	}

	if !available {
		t.Errorf("IsBwrapAvailable returned false but bwrap is installed: %v", err)
	}

	if version == "" {
		t.Log("warning: bwrap version string is empty")
	} else {
		t.Logf("bwrap version: %s", version)
	}
}

// [REQ:REQ-P0-004] Test safe-git wrapper creation
func TestCreateSafeGitWrapper(t *testing.T) {
	sandboxID := uuid.New().String()
	sandboxPath := "/workspace"

	err := CreateSafeGitWrapper(sandboxID, sandboxPath)
	if err != nil {
		t.Fatalf("CreateSafeGitWrapper failed: %v", err)
	}

	// Verify file was created
	wrapperPath := SafeGitWrapper(sandboxID)
	info, err := os.Stat(wrapperPath)
	if err != nil {
		t.Fatalf("wrapper not created: %v", err)
	}

	// Verify executable
	if info.Mode().Perm()&0100 == 0 {
		t.Error("wrapper should be executable")
	}

	// Verify content contains blocked commands
	content, err := os.ReadFile(wrapperPath)
	if err != nil {
		t.Fatalf("failed to read wrapper: %v", err)
	}

	expectedStrings := []string{
		"stash",
		"reset",
		"checkout",
		"GIT COMMAND BLOCKED",
	}
	for _, expected := range expectedStrings {
		if !containsString(string(content), expected) {
			t.Errorf("wrapper should contain '%s'", expected)
		}
	}

	// Cleanup
	err = RemoveSafeGitWrapper(sandboxID)
	if err != nil {
		t.Errorf("RemoveSafeGitWrapper failed: %v", err)
	}

	// Verify removed
	_, err = os.Stat(wrapperPath)
	if !os.IsNotExist(err) {
		t.Error("wrapper should be removed")
	}
}

// [REQ:REQ-P0-004] Test process running check
func TestIsProcessRunning(t *testing.T) {
	// Current process should be running
	if !IsProcessRunning(os.Getpid()) {
		t.Error("current process should be running")
	}

	// Non-existent PID should not be running
	// Using a very high PID that's unlikely to exist
	if IsProcessRunning(999999) {
		t.Error("non-existent process should not be running")
	}
}

// [REQ:REQ-P0-004] Test default config values
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

// [REQ:REQ-P0-004] Test bwrap info retrieval
func TestGetBwrapInfo(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("bwrap info tests require Linux")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := GetBwrapInfo(ctx)
	if err != nil {
		t.Fatalf("GetBwrapInfo failed: %v", err)
	}

	// Info struct should always be returned
	if info == nil {
		t.Fatal("GetBwrapInfo returned nil info")
	}

	t.Logf("bwrap info: available=%v, version=%s, userNS=%v, overlayNS=%v",
		info.Available, info.Version, info.UserNamespaceEnabled, info.OverlayfsInUserNS)
}

// --- Resource Limits Tests (Phase 1) ---

// [REQ:OT-P2-008] Test resource limits struct HasLimits method
func TestResourceLimitsHasLimits(t *testing.T) {
	tests := []struct {
		name     string
		limits   ResourceLimits
		expected bool
	}{
		{
			name:     "all zero",
			limits:   ResourceLimits{},
			expected: false,
		},
		{
			name:     "memory set",
			limits:   ResourceLimits{MemoryLimitMB: 512},
			expected: true,
		},
		{
			name:     "cpu time set",
			limits:   ResourceLimits{CPUTimeSec: 60},
			expected: true,
		},
		{
			name:     "max processes set",
			limits:   ResourceLimits{MaxProcesses: 100},
			expected: true,
		},
		{
			name:     "max files set",
			limits:   ResourceLimits{MaxOpenFiles: 1024},
			expected: true,
		},
		{
			name:     "timeout only - not counted as prlimit",
			limits:   ResourceLimits{TimeoutSec: 300},
			expected: false, // TimeoutSec is handled via context, not prlimit
		},
		{
			name:     "multiple limits",
			limits:   ResourceLimits{MemoryLimitMB: 512, CPUTimeSec: 60, MaxProcesses: 100},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.limits.HasLimits(); got != tt.expected {
				t.Errorf("HasLimits() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// [REQ:OT-P2-008] Test prlimit argument building
func TestBuildPrlimitArgs(t *testing.T) {
	tests := []struct {
		name       string
		limits     ResourceLimits
		wantNil    bool
		wantArgs   []string
		dontWant   []string
	}{
		{
			name:    "no limits returns nil",
			limits:  ResourceLimits{},
			wantNil: true,
		},
		{
			name:    "memory limit",
			limits:  ResourceLimits{MemoryLimitMB: 512},
			wantNil: false,
			wantArgs: []string{
				"--as=536870912", // 512 * 1024 * 1024
				"--",
			},
		},
		{
			name:    "cpu time limit",
			limits:  ResourceLimits{CPUTimeSec: 60},
			wantNil: false,
			wantArgs: []string{
				"--cpu=60",
				"--",
			},
		},
		{
			name:    "max processes",
			limits:  ResourceLimits{MaxProcesses: 100},
			wantNil: false,
			wantArgs: []string{
				"--nproc=100",
				"--",
			},
		},
		{
			name:    "max open files",
			limits:  ResourceLimits{MaxOpenFiles: 1024},
			wantNil: false,
			wantArgs: []string{
				"--nofile=1024",
				"--",
			},
		},
		{
			name:    "combined limits",
			limits:  ResourceLimits{MemoryLimitMB: 256, CPUTimeSec: 30, MaxProcesses: 50},
			wantNil: false,
			wantArgs: []string{
				"--as=268435456", // 256 MB
				"--cpu=30",
				"--nproc=50",
				"--",
			},
		},
		{
			name:    "timeout not in prlimit args",
			limits:  ResourceLimits{TimeoutSec: 300}, // timeout handled via context
			wantNil: true,                            // no prlimit needed for timeout-only
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildPrlimitArgs(tt.limits)

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

			for _, dontWant := range tt.dontWant {
				if contains(args, dontWant) {
					t.Errorf("did not expect %q in args, got: %v", dontWant, args)
				}
			}
		})
	}
}

// [REQ:OT-P2-008] Test BuildExecCommand with and without resource limits
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
			wantContain: []string{
				"--unshare-user",
				"--bind", "/tmp/merged", "/workspace",
				"--",
				"ls", "-la",
			},
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
			wantContain: []string{
				"--as=536870912", // memory limit
				"--",
				"bwrap",
				"--unshare-user",
				"my-agent", "--task", "fix",
			},
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
			wantContain: []string{
				"--as=268435456",
				"--cpu=60",
				"bwrap",
				"build",
			},
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

// [REQ:OT-P2-008] Test Vrooli-aware isolation level affects network
func TestVrooliAwareIsolation(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("bwrap tests require Linux")
	}

	sandbox := &types.Sandbox{
		ID:        uuid.New(),
		MergedDir: "/tmp/test",
		LowerDir:  "/tmp/lower",
	}

	// Full isolation should unshare network
	cfgFull := DefaultBwrapConfig()
	cfgFull.IsolationLevel = IsolationFull
	argsFull := buildBwrapArgs(sandbox, cfgFull)

	if !contains(argsFull, "--unshare-net") {
		t.Error("full isolation should include --unshare-net")
	}

	// Vrooli-aware should NOT unshare network (allow localhost)
	cfgVrooli := DefaultBwrapConfig()
	cfgVrooli.IsolationLevel = IsolationVrooliAware
	argsVrooli := buildBwrapArgs(sandbox, cfgVrooli)

	if contains(argsVrooli, "--unshare-net") {
		t.Error("vrooli-aware isolation should NOT include --unshare-net")
	}
}

// [REQ:OT-P2-008] Test default config includes isolation level
func TestDefaultBwrapConfigIncludesIsolationLevel(t *testing.T) {
	cfg := DefaultBwrapConfig()

	if cfg.IsolationLevel != IsolationFull {
		t.Errorf("default IsolationLevel = %q, want %q", cfg.IsolationLevel, IsolationFull)
	}
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// [REQ:OT-P2-008] Test GetVrooliEnvVars returns expected environment variables
func TestGetVrooliEnvVars(t *testing.T) {
	// Save original environment
	origVrooliRoot := os.Getenv("VROOLI_ROOT")
	origVrooliEnv := os.Getenv("VROOLI_ENV")
	origApiManager := os.Getenv("API_MANAGER_URL")
	defer func() {
		restoreEnv("VROOLI_ROOT", origVrooliRoot)
		restoreEnv("VROOLI_ENV", origVrooliEnv)
		restoreEnv("API_MANAGER_URL", origApiManager)
	}()

	// Test with no environment variables set
	os.Unsetenv("VROOLI_ROOT")
	os.Unsetenv("VROOLI_ENV")
	os.Unsetenv("API_MANAGER_URL")

	vars := GetVrooliEnvVars()
	if _, ok := vars["VROOLI_ROOT"]; ok {
		t.Error("VROOLI_ROOT should not be set when environment variable is empty")
	}

	// Test with VROOLI_ROOT set
	os.Setenv("VROOLI_ROOT", "/home/user/Vrooli")
	vars = GetVrooliEnvVars()
	if vars["VROOLI_ROOT"] != "/vrooli" {
		t.Errorf("VROOLI_ROOT = %q, want %q", vars["VROOLI_ROOT"], "/vrooli")
	}

	// Test with additional environment variables
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

// [REQ:OT-P2-008] Test ApplyVrooliAwareConfig sets up config correctly
func TestApplyVrooliAwareConfig(t *testing.T) {
	// Save original environment
	origVrooliRoot := os.Getenv("VROOLI_ROOT")
	defer restoreEnv("VROOLI_ROOT", origVrooliRoot)

	os.Setenv("VROOLI_ROOT", "/home/user/Vrooli")

	cfg := DefaultBwrapConfig()

	// Verify default state
	if cfg.IsolationLevel != IsolationFull {
		t.Fatalf("default IsolationLevel = %q, want %q", cfg.IsolationLevel, IsolationFull)
	}
	if cfg.AllowNetwork {
		t.Fatal("default AllowNetwork = true, want false")
	}

	// Apply Vrooli-aware config
	ApplyVrooliAwareConfig(&cfg)

	// Verify changes
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

// [REQ:OT-P2-008] Test ApplyVrooliAwareConfig preserves existing env vars
func TestApplyVrooliAwareConfigPreservesExistingEnv(t *testing.T) {
	cfg := DefaultBwrapConfig()
	cfg.Env["CUSTOM_VAR"] = "custom_value"

	ApplyVrooliAwareConfig(&cfg)

	if cfg.Env["CUSTOM_VAR"] != "custom_value" {
		t.Errorf("Env[CUSTOM_VAR] = %q, want %q", cfg.Env["CUSTOM_VAR"], "custom_value")
	}
	// Default PATH should still be present
	if cfg.Env["PATH"] == "" {
		t.Error("Env[PATH] should not be empty")
	}
}

// Helper to restore environment variable
func restoreEnv(key, value string) {
	if value == "" {
		os.Unsetenv(key)
	} else {
		os.Setenv(key, value)
	}
}
