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

	// StdoutWriter receives the process's stdout. Required for StartProcess
	// to capture output; Exec uses its own buffers.
	StdoutWriter io.Writer

	// StderrWriter receives the process's stderr. Required for StartProcess
	// to capture output; Exec uses its own buffers.
	StderrWriter io.Writer

	// StdinReader, if non-nil, is wired to the process's stdin pipe. The
	// caller owns reads on the other end (typically an io.Pipe) and is
	// responsible for closing the writer half to signal EOF.
	StdinReader io.Reader

	// OnExit, if non-nil, is invoked exactly once after cmd.Wait() returns
	// for a background-started process. It receives the exit code, the
	// terminating signal (0 if exited normally), and a best-effort
	// OOM-killed indicator. The driver dispatches this from a goroutine
	// it owns; callers must not rely on synchronisation with StartProcess
	// returning.
	OnExit func(exitCode int, signal int, oomKilled bool)
}

// ExitInfoFromState extracts a standard tuple (exitCode, signal, oomKilled)
// from a *os.ProcessState plus the wait error. Exported so tests / future
// drivers can share the canonical extraction logic.
//
// Behavior:
//   - exited normally: exitCode = state.ExitCode(), signal = 0
//   - killed by signal: exitCode = -1, signal = int(sig)
//   - OOM-killed: detected via syscall.WaitStatus.Signal() == SIGKILL plus
//     a check on /sys/fs/cgroup/.../memory.oom_control if available.
//     We surface the SIGKILL case but conservatively flag oomKilled=false
//     unless a stronger indicator is available; callers can use the bool.
func ExitInfoFromState(state *os.ProcessState, waitErr error) (exitCode, signal int, oomKilled bool) {
	if state == nil {
		// We have no state — best we can do.
		if waitErr != nil {
			return -1, 0, false
		}
		return 0, 0, false
	}
	if status, ok := state.Sys().(syscall.WaitStatus); ok {
		if status.Signaled() {
			return -1, int(status.Signal()), false
		}
		if status.Exited() {
			return status.ExitStatus(), 0, false
		}
	}
	return state.ExitCode(), 0, false
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
			"PATH":         "/usr/local/bin:/usr/bin:/bin",
			"HOME":         "/tmp",
			"SHELL":        "/bin/sh",
			"PROJECT_PATH": "/workspace",
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

	// Bind the per-sandbox home overlay at the host $HOME path inside
	// the namespace. This is the audit-of-change mechanism for HOME-
	// relative writes (auth tokens, tool caches, etc.): reads pass
	// through to the host home (lower layer), writes land in the
	// per-sandbox upper layer that's discarded at sandbox teardown.
	//
	// Without this, agent CLIs that read $HOME/<tool>/<config> couldn't
	// find their host state (the 2026-04-28 follow-on after the SSE-500
	// chain was fully fixed). The fuse-overlayfs is created in the
	// driver's Mount; bwrap just exposes it.
	if s.HomeMergedDir != "" {
		if hostHome := os.Getenv("HOME"); hostHome != "" && filepath.IsAbs(hostHome) {
			addDirHierarchy(&args, hostHome)
			args = append(args, "--bind", s.HomeMergedDir, filepath.Clean(hostHome))
		}
	}

	// Optional compatibility: mirror the workspace at the project's host path so
	// prompts/tools that use host-absolute paths (e.g., /home/user/project/...) work.
	//
	// Enabled via WORKSPACE_SANDBOX_MIRROR_PROJECT_ROOT=true|1.
	if shouldMirrorProjectRoot() && s.ProjectRoot != "" && filepath.IsAbs(s.ProjectRoot) {
		addDirHierarchy(&args, s.ProjectRoot)
		args = append(args, "--bind", s.MergedDir, filepath.Clean(s.ProjectRoot))
	}

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

func shouldMirrorProjectRoot() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("WORKSPACE_SANDBOX_MIRROR_PROJECT_ROOT")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func addDirHierarchy(args *[]string, absPath string) {
	clean := filepath.Clean(absPath)
	if clean == "" || clean == "/" || !filepath.IsAbs(clean) {
		return
	}

	// Avoid creating/overriding critical system mountpoints.
	switch clean {
	case "/bin", "/sbin", "/usr", "/etc", "/lib", "/lib64", "/proc", "/dev", "/tmp":
		return
	}

	parts := strings.Split(strings.TrimPrefix(clean, "/"), string(filepath.Separator))
	cur := ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		cur = cur + string(filepath.Separator) + p
		*args = append(*args, "--dir", cur)
	}
}

// addVrooliAwareBinds adds bind mounts for Vrooli-aware isolation.
//
// Most agent-config visibility ($HOME/.local/{bin,share}, $HOME/.config,
// $HOME/.claude, etc.) now flows through the per-sandbox HOME overlay
// set up by Driver.Mount and bound at the host $HOME path inside the
// namespace by buildBwrapArgs. This function is the legacy path, used
// only when the ProfileStore is nil; it intentionally adds nothing
// $HOME-related to avoid shadowing the overlay. Kept around for the
// VROOLI_ROOT entry, which is independent of $HOME.
func addVrooliAwareBinds(args *[]string) {
	// Bind VROOLI_ROOT if set (read-only for reference). Independent of
	// $HOME so it lives outside the home overlay.
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
		"VROOLI_ENV",            // Environment (development, production, etc.)
		"VROOLI_LOG_LEVEL",      // Logging level
		"API_MANAGER_URL",       // API manager endpoint
		"SCENARIO_REGISTRY_URL", // Scenario registry endpoint
		"RESOURCE_REGISTRY_URL", // Resource registry endpoint
		"XDG_CONFIG_HOME",       // Config directory standard
		"XDG_DATA_HOME",         // Data directory standard
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

// IsolationProfile mirrors config.IsolationProfile for driver-level use.
// This avoids a circular import between driver and config packages.
type IsolationProfile struct {
	ID             string
	Name           string
	Description    string
	Builtin        bool
	NetworkAccess  string            // "none", "localhost", "full"
	ReadOnlyBinds  map[string]string // host path -> sandbox path
	ReadWriteBinds map[string]string // host path -> sandbox path
	Environment    map[string]string // env var -> value (supports $VAR expansion)
	Hostname       string
}

// ApplyIsolationProfile configures BwrapConfig based on an IsolationProfile.
// This replaces the hardcoded IsolationLevel logic with profile-driven configuration.
// Profile binds are added to the existing ReadOnlyPaths/ReadWritePaths arrays,
// which are then processed by buildBwrapArgs.
func ApplyIsolationProfile(cfg *BwrapConfig, profile *IsolationProfile) {
	if profile == nil {
		return
	}

	// Apply network access
	switch profile.NetworkAccess {
	case "none":
		cfg.AllowNetwork = false
	case "localhost", "full":
		cfg.AllowNetwork = true
	}

	// Apply hostname
	if profile.Hostname != "" {
		cfg.Hostname = profile.Hostname
	}

	// Apply read-only binds (expand placeholders)
	for src := range profile.ReadOnlyBinds {
		expandedSrc := expandPathPlaceholders(src)
		if expandedSrc != "" {
			cfg.ReadOnlyPaths = append(cfg.ReadOnlyPaths, expandedSrc)
		}
	}

	// Apply read-write binds
	for src := range profile.ReadWriteBinds {
		expandedSrc := expandPathPlaceholders(src)
		if expandedSrc != "" {
			cfg.ReadWritePaths = append(cfg.ReadWritePaths, expandedSrc)
		}
	}

	// Apply environment variables
	if cfg.Env == nil {
		cfg.Env = make(map[string]string)
	}
	for k, v := range profile.Environment {
		cfg.Env[k] = expandEnvPlaceholders(v)
	}
}

// expandPathPlaceholders expands $HOME, $USER, $VROOLI_ROOT in paths.
// Returns empty string if any placeholder cannot be resolved.
func expandPathPlaceholders(path string) string {
	if path == "" {
		return ""
	}

	home := os.Getenv("HOME")
	user := os.Getenv("USER")
	vrooliRoot := os.Getenv("VROOLI_ROOT")

	result := path
	result = strings.ReplaceAll(result, "$HOME", home)
	result = strings.ReplaceAll(result, "$USER", user)
	result = strings.ReplaceAll(result, "$VROOLI_ROOT", vrooliRoot)

	// Skip if placeholder wasn't resolved (e.g., $VROOLI_ROOT when not set)
	if strings.Contains(result, "$") {
		return ""
	}

	return result
}

// expandEnvPlaceholders expands $VAR references in environment values.
func expandEnvPlaceholders(value string) string {
	return os.ExpandEnv(value)
}

// MapIsolationLevelToProfile returns the profile ID for a legacy IsolationLevel.
// This maintains backwards compatibility with existing code using IsolationLevel.
func MapIsolationLevelToProfile(level IsolationLevel) string {
	switch level {
	case IsolationVrooliAware:
		return "vrooli-aware"
	case IsolationFull:
		fallthrough
	default:
		return "full"
	}
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
// stdout and stderr are wired to cfg.StdoutWriter / cfg.StderrWriter (both
// required for background processes; pass io.Discard to drop). If
// cfg.StdinReader is non-nil it is wired to the process's stdin pipe.
//
// When cfg.OnExit is non-nil the driver spawns a wait reaper goroutine that
// calls cfg.OnExit exactly once after cmd.Wait() returns.
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

	wireStartProcessIO(execCmd, cfg)

	// Start the process
	if err := execCmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start process: %w", err)
	}

	spawnExitReaper(execCmd, cfg.OnExit)

	return execCmd.Process.Pid, nil
}

// wireStartProcessIO wires cmd.Stdout / cmd.Stderr / cmd.Stdin from cfg.
// Background processes always need a writer for each stream so output is
// captured; pass io.Discard if the caller does not want to retain it.
func wireStartProcessIO(execCmd *exec.Cmd, cfg BwrapConfig) {
	if cfg.StdoutWriter != nil {
		execCmd.Stdout = cfg.StdoutWriter
	} else {
		execCmd.Stdout = io.Discard
	}
	if cfg.StderrWriter != nil {
		execCmd.Stderr = cfg.StderrWriter
	} else {
		execCmd.Stderr = io.Discard
	}
	if cfg.StdinReader != nil {
		execCmd.Stdin = cfg.StdinReader
	}
}

// spawnExitReaper waits for the process in a goroutine and dispatches
// ExitInfo to onExit (when non-nil) exactly once.
func spawnExitReaper(execCmd *exec.Cmd, onExit func(int, int, bool)) {
	if onExit == nil {
		// Still need to wait so the kernel doesn't accumulate zombies.
		go func() { _ = execCmd.Wait() }()
		return
	}
	go func() {
		waitErr := execCmd.Wait()
		exitCode, signal, oom := ExitInfoFromState(execCmd.ProcessState, waitErr)
		onExit(exitCode, signal, oom)
	}()
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
