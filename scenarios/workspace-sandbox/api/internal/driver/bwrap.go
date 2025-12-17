// Package driver provides sandbox driver interfaces and implementations.
package driver

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"workspace-sandbox/internal/types"
)

// BwrapConfig configures bubblewrap execution parameters.
type BwrapConfig struct {
	// AllowNetwork permits network access within the sandbox.
	// Default: false (more secure)
	AllowNetwork bool

	// AllowDevices permits device access.
	// Default: false (more secure)
	AllowDevices bool

	// ReadOnlyPaths are additional paths to bind read-only.
	ReadOnlyPaths []string

	// ReadWritePaths are additional paths to bind read-write.
	ReadWritePaths []string

	// Environment variables to set in the sandbox.
	Env map[string]string

	// WorkingDir sets the initial working directory.
	// If empty, defaults to the sandbox merged directory.
	WorkingDir string

	// SharePID shares the PID namespace with the host.
	// Default: false (isolated PID namespace)
	SharePID bool

	// Hostname sets the hostname inside the sandbox.
	Hostname string
}

// DefaultBwrapConfig returns a secure default configuration.
func DefaultBwrapConfig() BwrapConfig {
	return BwrapConfig{
		AllowNetwork: false,
		AllowDevices: false,
		SharePID:     false,
		Hostname:     "sandbox",
		Env: map[string]string{
			"PATH":  "/usr/local/bin:/usr/bin:/bin",
			"HOME":  "/tmp",
			"SHELL": "/bin/sh",
		},
	}
}

// ExecResult contains the result of executing a command in the sandbox.
type ExecResult struct {
	// ExitCode is the process exit code.
	ExitCode int

	// Stdout contains captured stdout output.
	Stdout []byte

	// Stderr contains captured stderr output.
	Stderr []byte

	// PID is the process ID (for tracking purposes).
	PID int

	// Error contains any execution error.
	Error error
}

// Exec executes a command inside the sandbox using bubblewrap.
//
// This method provides process isolation by running the command in a constrained
// filesystem view where:
//   - The sandbox merged directory is mounted as the root
//   - The canonical repo directories are read-only
//   - The sandbox upper layer is writable
//   - Network and device access are restricted by default
//
// The command runs in its own process namespace (unless SharePID is set),
// providing isolation from host processes.
func (d *OverlayfsDriver) Exec(ctx context.Context, s *types.Sandbox, cfg BwrapConfig, cmd string, args ...string) (*ExecResult, error) {
	// Verify sandbox is in a valid state
	if s.MergedDir == "" {
		return nil, fmt.Errorf("sandbox is not mounted (merged directory empty)")
	}

	// Check if bwrap is available
	bwrapPath, err := exec.LookPath("bwrap")
	if err != nil {
		return nil, fmt.Errorf("bubblewrap (bwrap) not found: %w. Install with: apt-get install bubblewrap", err)
	}

	// Build bwrap command arguments
	bwrapArgs := d.buildBwrapArgs(s, cfg)

	// Add the command to execute
	bwrapArgs = append(bwrapArgs, cmd)
	bwrapArgs = append(bwrapArgs, args...)

	// Create the command
	execCmd := exec.CommandContext(ctx, bwrapPath, bwrapArgs...)

	// Set up environment
	for k, v := range cfg.Env {
		execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Capture output
	var stdout, stderr strings.Builder
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	// Execute
	err = execCmd.Run()

	result := &ExecResult{
		Stdout: []byte(stdout.String()),
		Stderr: []byte(stderr.String()),
	}

	if execCmd.Process != nil {
		result.PID = execCmd.Process.Pid
	}

	// Determine exit code
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
			result.Error = err
		}
	}

	return result, nil
}

// buildBwrapArgs constructs the bubblewrap argument list.
func (d *OverlayfsDriver) buildBwrapArgs(s *types.Sandbox, cfg BwrapConfig) []string {
	args := []string{}

	// Unshare namespaces for isolation
	args = append(args, "--unshare-user")
	args = append(args, "--unshare-ipc")
	args = append(args, "--unshare-uts")
	args = append(args, "--unshare-cgroup")

	if !cfg.SharePID {
		args = append(args, "--unshare-pid")
	}

	if !cfg.AllowNetwork {
		args = append(args, "--unshare-net")
	}

	// Die with parent - ensures cleanup
	args = append(args, "--die-with-parent")

	// Set hostname
	if cfg.Hostname != "" {
		args = append(args, "--hostname", cfg.Hostname)
	}

	// Bind the sandbox merged directory as root
	// This is where the agent/tool will see the combined filesystem
	args = append(args, "--bind", s.MergedDir, "/workspace")

	// Make the project root read-only to prevent direct writes
	// The merged directory includes overlay, so writes go to upper layer
	args = append(args, "--ro-bind", s.LowerDir, "/workspace-readonly")

	// Essential system directories (read-only)
	args = append(args, "--ro-bind", "/usr", "/usr")
	args = append(args, "--ro-bind", "/lib", "/lib")
	if _, err := os.Stat("/lib64"); err == nil {
		args = append(args, "--ro-bind", "/lib64", "/lib64")
	}
	args = append(args, "--ro-bind", "/bin", "/bin")

	// Optional: /etc for basic system config (read-only)
	args = append(args, "--ro-bind", "/etc/resolv.conf", "/etc/resolv.conf")
	args = append(args, "--ro-bind", "/etc/hosts", "/etc/hosts")
	args = append(args, "--ro-bind", "/etc/passwd", "/etc/passwd")
	args = append(args, "--ro-bind", "/etc/group", "/etc/group")

	// Proc filesystem
	args = append(args, "--proc", "/proc")

	// Dev filesystem
	if cfg.AllowDevices {
		args = append(args, "--dev", "/dev")
	} else {
		args = append(args, "--dev", "/dev")
		// But restrict to minimal devices via symlinks
	}

	// Tmp directory (writable)
	args = append(args, "--tmpfs", "/tmp")

	// Additional read-only paths
	for _, path := range cfg.ReadOnlyPaths {
		if _, err := os.Stat(path); err == nil {
			args = append(args, "--ro-bind", path, path)
		}
	}

	// Additional read-write paths
	for _, path := range cfg.ReadWritePaths {
		if _, err := os.Stat(path); err == nil {
			args = append(args, "--bind", path, path)
		}
	}

	// Set working directory
	workDir := cfg.WorkingDir
	if workDir == "" {
		workDir = "/workspace"
	}
	args = append(args, "--chdir", workDir)

	// Add -- to separate bwrap args from command
	args = append(args, "--")

	return args
}

// IsBwrapAvailable checks if bubblewrap is installed and usable.
func IsBwrapAvailable(ctx context.Context) (bool, string, error) {
	bwrapPath, err := exec.LookPath("bwrap")
	if err != nil {
		return false, "", fmt.Errorf("bwrap not found in PATH: %w", err)
	}

	// Try to get version
	cmd := exec.CommandContext(ctx, bwrapPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return false, bwrapPath, fmt.Errorf("bwrap version check failed: %w", err)
	}

	return true, strings.TrimSpace(string(output)), nil
}

// StartProcess starts a long-running process in the sandbox and returns its PID.
// The process runs in the background and can be monitored/killed via the returned PID.
func (d *OverlayfsDriver) StartProcess(ctx context.Context, s *types.Sandbox, cfg BwrapConfig, cmd string, args ...string) (int, error) {
	if s.MergedDir == "" {
		return 0, fmt.Errorf("sandbox is not mounted (merged directory empty)")
	}

	bwrapPath, err := exec.LookPath("bwrap")
	if err != nil {
		return 0, fmt.Errorf("bubblewrap not found: %w", err)
	}

	// Build bwrap args
	bwrapArgs := d.buildBwrapArgs(s, cfg)
	bwrapArgs = append(bwrapArgs, cmd)
	bwrapArgs = append(bwrapArgs, args...)

	// Create command but don't wait for it
	execCmd := exec.Command(bwrapPath, bwrapArgs...)

	// Set environment
	for k, v := range cfg.Env {
		execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Set up process group so we can kill all children
	execCmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Start the process
	if err := execCmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start process: %w", err)
	}

	// Don't wait - let it run in background
	// The caller is responsible for tracking and cleaning up

	return execCmd.Process.Pid, nil
}

// KillProcessGroup kills a process and all its children by process group ID.
func KillProcessGroup(pid int) error {
	// Kill the process group (negative PID)
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		// Process may already be dead, try direct kill
		return syscall.Kill(pid, syscall.SIGKILL)
	}

	// Kill entire process group
	return syscall.Kill(-pgid, syscall.SIGKILL)
}

// IsProcessRunning checks if a process with the given PID is still running.
func IsProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, FindProcess always succeeds, so we need to send signal 0
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// WaitForProcess waits for a process to exit and returns its exit code.
func WaitForProcess(pid int) (int, error) {
	process, err := os.FindProcess(pid)
	if err != nil {
		return -1, fmt.Errorf("process not found: %w", err)
	}

	state, err := process.Wait()
	if err != nil {
		return -1, fmt.Errorf("wait failed: %w", err)
	}

	return state.ExitCode(), nil
}

// GetBwrapInfo returns information about the bwrap installation.
func GetBwrapInfo(ctx context.Context) (*BwrapInfo, error) {
	available, version, err := IsBwrapAvailable(ctx)
	if !available {
		return &BwrapInfo{
			Available: false,
			Error:     err.Error(),
		}, nil
	}

	// Check kernel features
	info := &BwrapInfo{
		Available: true,
		Version:   version,
		Path:      mustExecLookPath("bwrap"),
	}

	// Check for user namespace support
	if data, err := os.ReadFile("/proc/sys/kernel/unprivileged_userns_clone"); err == nil {
		info.UserNamespaceEnabled = strings.TrimSpace(string(data)) == "1"
	}

	// Check for overlayfs in namespaces
	info.OverlayfsInUserNS = checkOverlayfsUserNS()

	return info, nil
}

// BwrapInfo contains information about bubblewrap capabilities.
type BwrapInfo struct {
	Available            bool   `json:"available"`
	Version              string `json:"version,omitempty"`
	Path                 string `json:"path,omitempty"`
	UserNamespaceEnabled bool   `json:"userNamespaceEnabled"`
	OverlayfsInUserNS    bool   `json:"overlayfsInUserNS"`
	Error                string `json:"error,omitempty"`
}

func mustExecLookPath(name string) string {
	path, _ := exec.LookPath(name)
	return path
}

func checkOverlayfsUserNS() bool {
	// Check if kernel supports overlayfs in user namespaces
	// This is a best-effort check
	data, err := os.ReadFile("/proc/filesystems")
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "overlay")
}

// SafeGitWrapper returns the path to a safe-git wrapper script that explains
// blocked git commands during agent runs.
func SafeGitWrapper(sandboxID string) string {
	return filepath.Join("/tmp", fmt.Sprintf("safe-git-%s.sh", sandboxID))
}

// CreateSafeGitWrapper creates a wrapper script that intercepts dangerous git commands.
// This is guidance, not a security boundary.
func CreateSafeGitWrapper(sandboxID, sandboxPath string) error {
	wrapper := fmt.Sprintf(`#!/bin/bash
# Safe Git Wrapper for Sandbox %s
# This wrapper intercepts git commands that could modify history or lose work.
# It is NOT a security boundary - an adversarial process can bypass it.

BLOCKED_COMMANDS=(
    "stash"
    "reset"
    "checkout"
    "clean"
    "rebase"
    "merge"
    "push"
    "pull"
)

CMD="$1"

for blocked in "${BLOCKED_COMMANDS[@]}"; do
    if [[ "$CMD" == "$blocked" ]]; then
        echo "⚠️  GIT COMMAND BLOCKED: git $CMD"
        echo ""
        echo "This sandbox is designed for safe, isolated changes."
        echo "The command 'git $CMD' could modify the repository state in ways"
        echo "that conflict with the sandbox approval workflow."
        echo ""
        echo "Instead, use the sandbox workflow:"
        echo "  1. Make your changes in: %s"
        echo "  2. Review changes via: GET /api/v1/sandboxes/%s/diff"
        echo "  3. Approve or reject via the API/UI"
        echo ""
        echo "If you need this git command, exit the sandbox and run it on the host."
        exit 1
    fi
done

# Pass through allowed commands
exec /usr/bin/git "$@"
`, sandboxID, sandboxPath, sandboxID)

	wrapperPath := SafeGitWrapper(sandboxID)
	if err := os.WriteFile(wrapperPath, []byte(wrapper), 0o755); err != nil {
		return fmt.Errorf("failed to create safe-git wrapper: %w", err)
	}

	return nil
}

// RemoveSafeGitWrapper removes the wrapper script.
func RemoveSafeGitWrapper(sandboxID string) error {
	return os.Remove(SafeGitWrapper(sandboxID))
}
