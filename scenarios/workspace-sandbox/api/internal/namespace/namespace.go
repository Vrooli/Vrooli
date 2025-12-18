// Package namespace provides utilities for working with Linux user namespaces
// to enable unprivileged overlayfs mounts.
//
// # Background
//
// Linux kernel 5.11+ supports overlayfs mounts from within unprivileged user
// namespaces. This allows workspace-sandbox to use native kernel overlayfs
// without requiring root privileges or special capabilities.
//
// # How It Works
//
// 1. At startup, we check if user namespaces are available
// 2. If available, we re-exec the process via `unshare --user --mount --map-root-user`
// 3. Inside the user namespace, we appear as UID 0 and can mount overlayfs
// 4. The mount persists for the lifetime of the process
// 5. Network, IPC, etc. are NOT namespaced - only user and mount namespaces
//
// # Fallback Chain
//
// If user namespaces aren't available (older kernel, container restrictions),
// we fall back to:
// 1. fuse-overlayfs (if installed)
// 2. Copy driver (always works, but slower)
package namespace

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// Environment variable set when we're running inside the user namespace
const InUserNamespaceEnv = "WORKSPACE_SANDBOX_IN_USERNS"

// Environment variable to disable user namespace (for debugging/fallback)
const DisableUserNamespaceEnv = "WORKSPACE_SANDBOX_DISABLE_USERNS"

// Status represents the current namespace status
type Status struct {
	// InUserNamespace is true if we're currently in a user namespace
	InUserNamespace bool

	// CanCreateUserNamespace is true if we can create new user namespaces
	CanCreateUserNamespace bool

	// CanMountOverlayfs is true if overlayfs mounts work (tested empirically)
	CanMountOverlayfs bool

	// KernelVersion is the running kernel version
	KernelVersion string

	// Reason explains why certain features aren't available
	Reason string
}

// Check returns the current namespace status
func Check() Status {
	status := Status{
		InUserNamespace: os.Getenv(InUserNamespaceEnv) == "1",
		KernelVersion:   getKernelVersion(),
	}

	// If explicitly disabled, don't check further
	if os.Getenv(DisableUserNamespaceEnv) == "1" {
		status.Reason = "user namespace disabled via " + DisableUserNamespaceEnv
		return status
	}

	// Check if we can create user namespaces
	status.CanCreateUserNamespace = canCreateUserNamespace()
	if !status.CanCreateUserNamespace {
		status.Reason = "cannot create user namespaces (kernel config or security policy)"
		return status
	}

	// If we're already in a user namespace, test if overlayfs works
	if status.InUserNamespace {
		status.CanMountOverlayfs = testOverlayfsMount()
		if !status.CanMountOverlayfs {
			status.Reason = "overlayfs mount failed inside user namespace"
		}
		return status
	}

	// We can create namespaces but haven't entered one yet
	status.Reason = "not yet in user namespace"
	return status
}

// EnterUserNamespace re-execs the current process inside a user namespace.
// This function only returns if re-exec fails; on success, it replaces the process.
//
// Returns nil if we're already in a user namespace or if namespaces are disabled.
// Returns an error if re-exec fails.
func EnterUserNamespace() error {
	// Already in user namespace?
	if os.Getenv(InUserNamespaceEnv) == "1" {
		return nil
	}

	// Explicitly disabled?
	if os.Getenv(DisableUserNamespaceEnv) == "1" {
		return nil
	}

	// Check if we can create user namespaces
	if !canCreateUserNamespace() {
		return errors.New("user namespaces not available")
	}

	// Find unshare binary
	unsharePath, err := exec.LookPath("unshare")
	if err != nil {
		return fmt.Errorf("unshare command not found: %w", err)
	}

	// Get current executable path
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine executable path: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("cannot resolve executable symlinks: %w", err)
	}

	// Build argument list for unshare
	// --user: create user namespace
	// --mount: create mount namespace (required for overlayfs)
	// --map-root-user: map current UID/GID to root inside namespace
	// --propagation private: prevent mount propagation issues
	args := []string{
		"unshare",
		"--user",
		"--mount",
		"--map-root-user",
		"--propagation", "private",
		exePath,
	}
	args = append(args, os.Args[1:]...)

	// Set environment to indicate we're in a user namespace
	env := os.Environ()
	env = append(env, InUserNamespaceEnv+"=1")

	// Re-exec via unshare
	// Use syscall.Exec for true process replacement (no child process)
	return syscall.Exec(unsharePath, args, env)
}

// MustEnterUserNamespace is like EnterUserNamespace but logs and continues
// on failure instead of returning an error. Use this when fallback is acceptable.
func MustEnterUserNamespace(logger func(format string, args ...interface{})) {
	err := EnterUserNamespace()
	if err != nil {
		logger("user namespace not available, will use fallback driver: %v", err)
	}
}

// canCreateUserNamespace checks if the current process can create user namespaces
func canCreateUserNamespace() bool {
	// Check the sysctl that controls unprivileged user namespaces
	data, err := os.ReadFile("/proc/sys/kernel/unprivileged_userns_clone")
	if err == nil {
		val := strings.TrimSpace(string(data))
		if val == "0" {
			return false
		}
	}
	// If the file doesn't exist, user namespaces are likely enabled by default

	// Check if unshare is available
	_, err = exec.LookPath("unshare")
	if err != nil {
		return false
	}

	// Try to actually create a user namespace (most reliable test)
	cmd := exec.Command("unshare", "--user", "true")
	err = cmd.Run()
	return err == nil
}

// testOverlayfsMount tests if we can actually mount overlayfs
func testOverlayfsMount() bool {
	// Create temporary directories for the test
	tmpDir, err := os.MkdirTemp("", "overlay-test-")
	if err != nil {
		return false
	}
	defer os.RemoveAll(tmpDir)

	lower := filepath.Join(tmpDir, "lower")
	upper := filepath.Join(tmpDir, "upper")
	work := filepath.Join(tmpDir, "work")
	merged := filepath.Join(tmpDir, "merged")

	for _, dir := range []string{lower, upper, work, merged} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return false
		}
	}

	// Try to mount overlayfs
	opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lower, upper, work)
	err = syscall.Mount("overlay", merged, "overlay", 0, opts)
	if err != nil {
		return false
	}

	// Clean up - unmount
	syscall.Unmount(merged, 0)
	return true
}

// getKernelVersion returns the kernel version string
func getKernelVersion() string {
	var uname syscall.Utsname
	if err := syscall.Uname(&uname); err != nil {
		return "unknown"
	}

	// Convert [65]int8 to string
	var buf []byte
	for _, c := range uname.Release {
		if c == 0 {
			break
		}
		buf = append(buf, byte(c))
	}
	return string(buf)
}

// IsKernelAtLeast checks if the kernel version is at least the specified version
func IsKernelAtLeast(major, minor int) bool {
	version := getKernelVersion()
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return false
	}

	kernelMajor, err := strconv.Atoi(parts[0])
	if err != nil {
		return false
	}
	kernelMinor, err := strconv.Atoi(strings.Split(parts[1], "-")[0])
	if err != nil {
		return false
	}

	if kernelMajor > major {
		return true
	}
	if kernelMajor == major && kernelMinor >= minor {
		return true
	}
	return false
}

// RequiredKernelVersion is the minimum kernel version for unprivileged overlayfs
const RequiredKernelMajor = 5
const RequiredKernelMinor = 11 // 5.13 recommended for SELinux support
