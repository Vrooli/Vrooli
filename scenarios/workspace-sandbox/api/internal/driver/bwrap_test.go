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

	driver := NewOverlayfsDriver(DefaultConfig())
	sandbox := &types.Sandbox{
		ID:        uuid.New(),
		ScopePath: "/tmp/test",
		LowerDir:  "/tmp/lower",
		UpperDir:  "/tmp/upper",
		WorkDir:   "/tmp/work",
		MergedDir: "/tmp/merged",
	}

	cfg := DefaultBwrapConfig()
	args := driver.buildBwrapArgs(sandbox, cfg)

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

	driver := NewOverlayfsDriver(DefaultConfig())
	sandbox := &types.Sandbox{
		ID:        uuid.New(),
		MergedDir: "/tmp/test",
		LowerDir:  "/tmp/lower",
	}

	// Network disabled (default)
	cfg := DefaultBwrapConfig()
	args := driver.buildBwrapArgs(sandbox, cfg)
	if !contains(args, "--unshare-net") {
		t.Error("expected --unshare-net when AllowNetwork=false")
	}

	// Network enabled
	cfg.AllowNetwork = true
	args = driver.buildBwrapArgs(sandbox, cfg)
	if contains(args, "--unshare-net") {
		t.Error("did not expect --unshare-net when AllowNetwork=true")
	}
}

// [REQ:REQ-P0-004] Test PID namespace isolation toggle
func TestBwrapPIDNamespaceConfig(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("bwrap tests require Linux")
	}

	driver := NewOverlayfsDriver(DefaultConfig())
	sandbox := &types.Sandbox{
		ID:        uuid.New(),
		MergedDir: "/tmp/test",
		LowerDir:  "/tmp/lower",
	}

	// PID isolated (default)
	cfg := DefaultBwrapConfig()
	args := driver.buildBwrapArgs(sandbox, cfg)
	if !contains(args, "--unshare-pid") {
		t.Error("expected --unshare-pid when SharePID=false")
	}

	// PID shared
	cfg.SharePID = true
	args = driver.buildBwrapArgs(sandbox, cfg)
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
