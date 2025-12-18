// Package driver provides sandbox driver interfaces and implementations.
package driver

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"workspace-sandbox/internal/types"
)

// IsolationLevel defines the level of isolation for sandbox execution.
type IsolationLevel string

const (
	// IsolationFull provides maximum isolation - only /workspace and basic system paths.
	IsolationFull IsolationLevel = "full"

	// IsolationVrooliAware allows access to Vrooli CLIs and localhost network.
	IsolationVrooliAware IsolationLevel = "vrooli-aware"
)

// ResourceLimits configures process resource constraints via prlimit.
// Zero values mean unlimited (no limit applied).
type ResourceLimits struct {
	// MemoryLimitMB sets the maximum address space in megabytes.
	// Maps to prlimit --as flag.
	MemoryLimitMB int

	// CPUTimeSec sets the maximum CPU time in seconds.
	// Maps to prlimit --cpu flag.
	CPUTimeSec int

	// MaxProcesses sets the maximum number of child processes.
	// Maps to prlimit --nproc flag.
	MaxProcesses int

	// MaxOpenFiles sets the maximum number of open file descriptors.
	// Maps to prlimit --nofile flag.
	MaxOpenFiles int

	// TimeoutSec sets the wall-clock timeout in seconds.
	// This is handled via context timeout, not prlimit.
	TimeoutSec int
}

// HasLimits returns true if any resource limits are set.
func (r ResourceLimits) HasLimits() bool {
	return r.MemoryLimitMB > 0 || r.CPUTimeSec > 0 || r.MaxProcesses > 0 || r.MaxOpenFiles > 0
}

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

	// IsolationLevel controls the isolation preset.
	// Default: IsolationFull
	IsolationLevel IsolationLevel

	// ResourceLimits configures process resource constraints.
	ResourceLimits ResourceLimits

	// LogWriter is an optional writer for capturing process stdout/stderr.
	// If provided, both stdout and stderr are written to this writer.
	// Used for background processes (StartProcess) to capture output.
	LogWriter io.WriteCloser
}

// DefaultBwrapConfig returns a secure default configuration.
func DefaultBwrapConfig() BwrapConfig {
	return BwrapConfig{
		AllowNetwork:   false,
		AllowDevices:   false,
		SharePID:       false,
		Hostname:       "sandbox",
		IsolationLevel: IsolationFull,
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
//
// Resource limits (memory, CPU, processes) are enforced via prlimit if configured.
// Wall-clock timeout is enforced via context timeout.
func (d *OverlayfsDriver) Exec(ctx context.Context, s *types.Sandbox, cfg BwrapConfig, cmd string, args ...string) (*ExecResult, error) {
	// Verify sandbox is in a valid state
	if s.MergedDir == "" {
		return nil, fmt.Errorf("sandbox is not mounted (merged directory empty)")
	}

	// Build the full command (potentially wrapped with prlimit)
	executable, execArgs := BuildExecCommand(s, cfg, cmd, args...)

	// Check if the executable is available
	execPath, err := exec.LookPath(executable)
	if err != nil {
		if executable == "prlimit" {
			return nil, fmt.Errorf("prlimit not found: %w. Resource limits require prlimit (part of util-linux)", err)
		}
		return nil, fmt.Errorf("bubblewrap (bwrap) not found: %w. Install with: apt-get install bubblewrap", err)
	}

	// Apply wall-clock timeout if configured
	execCtx := ctx
	var cancel context.CancelFunc
	if cfg.ResourceLimits.TimeoutSec > 0 {
		timeout := time.Duration(cfg.ResourceLimits.TimeoutSec) * time.Second
		execCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	// Create the command
	execCmd := exec.CommandContext(execCtx, execPath, execArgs...)

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
		} else if execCtx.Err() == context.DeadlineExceeded {
			// Timeout - return a special exit code
			result.ExitCode = 124 // Standard timeout exit code
			result.Error = fmt.Errorf("process timed out after %d seconds", cfg.ResourceLimits.TimeoutSec)
		} else {
			result.ExitCode = -1
			result.Error = err
		}
	}

	return result, nil
}

// buildPrlimitArgs constructs prlimit arguments for resource limiting.
// Returns nil if no limits are configured.
func buildPrlimitArgs(limits ResourceLimits) []string {
	if !limits.HasLimits() {
		return nil
	}

	args := []string{}

	if limits.MemoryLimitMB > 0 {
		// --as sets address space (virtual memory) limit in bytes
		bytes := int64(limits.MemoryLimitMB) * 1024 * 1024
		args = append(args, fmt.Sprintf("--as=%d", bytes))
	}

	if limits.CPUTimeSec > 0 {
		// --cpu sets CPU time limit in seconds
		args = append(args, fmt.Sprintf("--cpu=%d", limits.CPUTimeSec))
	}

	if limits.MaxProcesses > 0 {
		// --nproc sets max number of processes
		args = append(args, fmt.Sprintf("--nproc=%d", limits.MaxProcesses))
	}

	if limits.MaxOpenFiles > 0 {
		// --nofile sets max number of open file descriptors
		args = append(args, fmt.Sprintf("--nofile=%d", limits.MaxOpenFiles))
	}

	// Add separator before the wrapped command
	args = append(args, "--")

	return args
}

// buildBwrapArgs constructs the bubblewrap argument list.
// This is a package-level function so it can be shared across drivers.
func buildBwrapArgs(s *types.Sandbox, cfg BwrapConfig) []string {
	args := []string{}

	// Unshare namespaces for isolation
	args = append(args, "--unshare-user")
	args = append(args, "--unshare-ipc")
	args = append(args, "--unshare-uts")
	args = append(args, "--unshare-cgroup")

	if !cfg.SharePID {
		args = append(args, "--unshare-pid")
	}

	// Network isolation depends on isolation level
	if cfg.IsolationLevel == IsolationVrooliAware {
		// Vrooli-aware: don't unshare network (allow localhost)
		// Note: AllowNetwork=true also allows network
	} else if !cfg.AllowNetwork {
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

	// Vrooli-aware isolation: add access to CLIs and configs
	if cfg.IsolationLevel == IsolationVrooliAware {
		addVrooliAwareBinds(&args)
	}

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

// addVrooliAwareBinds adds bind mounts for Vrooli-aware isolation.
// This includes access to Vrooli CLIs and configuration.
func addVrooliAwareBinds(args *[]string) {
	home := os.Getenv("HOME")
	if home == "" {
		return
	}

	// Bind ~/.local/bin read-only (scenario CLIs)
	localBin := filepath.Join(home, ".local", "bin")
	if _, err := os.Stat(localBin); err == nil {
		*args = append(*args, "--ro-bind", localBin, "/usr/local/bin")
	}

	// Bind ~/.config/vrooli read-only (CLI configurations)
	vrooliConfig := filepath.Join(home, ".config", "vrooli")
	if _, err := os.Stat(vrooliConfig); err == nil {
		// Create the target path structure inside sandbox
		user := os.Getenv("USER")
		if user == "" {
			user = "user"
		}
		targetConfig := filepath.Join("/home", user, ".config", "vrooli")
		*args = append(*args, "--ro-bind", vrooliConfig, targetConfig)
	}

	// Also bind VROOLI_ROOT if set (read-only for reference)
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot != "" {
		if _, err := os.Stat(vrooliRoot); err == nil {
			*args = append(*args, "--ro-bind", vrooliRoot, "/vrooli")
		}
	}
}

// GetVrooliEnvVars returns environment variables for Vrooli-aware isolation.
// These variables help agents interact with Vrooli CLIs and services.
func GetVrooliEnvVars() map[string]string {
	vars := make(map[string]string)

	// VROOLI_ROOT - points to /vrooli inside sandbox (we bind-mount the real root there)
	if vrooliRoot := os.Getenv("VROOLI_ROOT"); vrooliRoot != "" {
		vars["VROOLI_ROOT"] = "/vrooli"
	}

	// Pass through common Vrooli-related environment variables
	envsToCopy := []string{
		"VROOLI_ENV",              // Environment (development, production, etc.)
		"VROOLI_LOG_LEVEL",        // Logging level
		"API_MANAGER_URL",         // API manager endpoint
		"SCENARIO_REGISTRY_URL",   // Scenario registry endpoint
		"RESOURCE_REGISTRY_URL",   // Resource registry endpoint
		"XDG_CONFIG_HOME",         // Config directory standard
		"XDG_DATA_HOME",           // Data directory standard
	}

	for _, env := range envsToCopy {
		if val := os.Getenv(env); val != "" {
			vars[env] = val
		}
	}

	return vars
}

// ApplyVrooliAwareConfig augments a BwrapConfig for Vrooli-aware isolation.
// This sets up the environment variables needed for agents to interact with
// Vrooli CLIs and services running on localhost.
func ApplyVrooliAwareConfig(cfg *BwrapConfig) {
	// Set isolation level
	cfg.IsolationLevel = IsolationVrooliAware

	// Add Vrooli environment variables
	vrooliEnv := GetVrooliEnvVars()
	for k, v := range vrooliEnv {
		cfg.Env[k] = v
	}

	// Vrooli-aware always allows network (for localhost API access)
	cfg.AllowNetwork = true
}

// BuildExecCommand builds the full command line for executing in a sandbox.
// Returns (executable, args) where executable is either "prlimit" or "bwrap".
func BuildExecCommand(s *types.Sandbox, cfg BwrapConfig, cmd string, cmdArgs ...string) (string, []string) {
	bwrapArgs := buildBwrapArgs(s, cfg)
	bwrapArgs = append(bwrapArgs, cmd)
	bwrapArgs = append(bwrapArgs, cmdArgs...)

	// Check if we need prlimit wrapping
	prlimitArgs := buildPrlimitArgs(cfg.ResourceLimits)
	if prlimitArgs == nil {
		// No resource limits - run bwrap directly
		return "bwrap", bwrapArgs
	}

	// Wrap with prlimit: prlimit [limits] -- bwrap [bwrap-args]
	fullArgs := prlimitArgs
	fullArgs = append(fullArgs, "bwrap")
	fullArgs = append(fullArgs, bwrapArgs...)

	return "prlimit", fullArgs
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
//
// Resource limits are applied via prlimit if configured.
// Note: TimeoutSec is not enforced for background processes since they're detached.
// Use process tracking and manual kill for cleanup.
//
// If cfg.LogWriter is provided, stdout and stderr are redirected to it.
// The caller is responsible for closing the LogWriter when the process exits.
func (d *OverlayfsDriver) StartProcess(ctx context.Context, s *types.Sandbox, cfg BwrapConfig, cmd string, args ...string) (int, error) {
	if s.MergedDir == "" {
		return 0, fmt.Errorf("sandbox is not mounted (merged directory empty)")
	}

	// Build the full command (potentially wrapped with prlimit)
	executable, execArgs := BuildExecCommand(s, cfg, cmd, args...)

	// Check if the executable is available
	execPath, err := exec.LookPath(executable)
	if err != nil {
		if executable == "prlimit" {
			return 0, fmt.Errorf("prlimit not found: %w. Resource limits require prlimit (part of util-linux)", err)
		}
		return 0, fmt.Errorf("bubblewrap not found: %w", err)
	}

	// Create command but don't wait for it
	execCmd := exec.Command(execPath, execArgs...)

	// Set environment
	for k, v := range cfg.Env {
		execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Set up process group so we can kill all children
	execCmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Redirect output to log writer if provided
	if cfg.LogWriter != nil {
		execCmd.Stdout = cfg.LogWriter
		execCmd.Stderr = cfg.LogWriter
	}

	// Start the process
	if err := execCmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start process: %w", err)
	}

	// Don't wait - let it run in background
	// The caller is responsible for tracking and cleanup

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
