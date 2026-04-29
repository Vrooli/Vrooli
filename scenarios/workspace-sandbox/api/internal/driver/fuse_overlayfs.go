// Package driver provides filesystem drivers for sandbox isolation.
//
// fuse_overlayfs.go implements a driver using fuse-overlayfs for unprivileged
// operation with direct filesystem access.
//
// Key advantages over kernel overlayfs in user namespace:
//   - Direct filesystem access (merged directory visible to all processes)
//   - No need for user namespace wrapping
//   - Compatible with file managers, IDEs, and other tools
//
// Trade-offs:
//   - Slightly slower than kernel overlayfs
//   - Requires fuse-overlayfs to be installed
//   - Requires /dev/fuse access
package driver

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/google/uuid"

	"workspace-sandbox/internal/types"
)

// FuseOverlayfsDriver implements the Driver interface using fuse-overlayfs.
// This driver provides unprivileged overlayfs with direct filesystem access.
type FuseOverlayfsDriver struct {
	config Config
}

// NewFuseOverlayfsDriver creates a new fuse-overlayfs driver.
func NewFuseOverlayfsDriver(cfg Config) *FuseOverlayfsDriver {
	if cfg.BaseDir == "" {
		cfg.BaseDir = DefaultConfig().BaseDir
	}
	return &FuseOverlayfsDriver{config: cfg}
}

// Type returns the driver type.
func (d *FuseOverlayfsDriver) Type() DriverType {
	return DriverTypeFuseOverlayfs
}

// BaseDir returns the configured base directory for sandboxes.
func (d *FuseOverlayfsDriver) BaseDir() string {
	return d.config.BaseDir
}

// Version returns the driver version.
func (d *FuseOverlayfsDriver) Version() string {
	// Try to get fuse-overlayfs version
	cmd := exec.Command("fuse-overlayfs", "--version")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		// Look for the line containing "fuse-overlayfs" specifically
		// Output format:
		//   fusermount3 version: 3.14.0
		//   fuse-overlayfs: version 1.13-dev
		//   FUSE library version 3.14.0
		for _, line := range lines {
			if strings.HasPrefix(line, "fuse-overlayfs") {
				// Extract version from "fuse-overlayfs: version 1.13-dev"
				parts := strings.Split(line, "version")
				if len(parts) > 1 {
					return strings.TrimSpace(parts[1])
				}
			}
		}
		// Fallback to first line if fuse-overlayfs line not found
		if len(lines) > 0 {
			return strings.TrimSpace(lines[0])
		}
	}
	return "1.0"
}

// IsAvailable checks if fuse-overlayfs can be used on this system.
func (d *FuseOverlayfsDriver) IsAvailable(ctx context.Context) (bool, error) {
	// Check if fuse-overlayfs is installed
	if _, err := exec.LookPath("fuse-overlayfs"); err != nil {
		return false, fmt.Errorf("fuse-overlayfs not found in PATH: %w", err)
	}

	// Check if fusermount is available (for unmounting)
	if _, err := exec.LookPath("fusermount"); err != nil {
		if _, err := exec.LookPath("fusermount3"); err != nil {
			return false, fmt.Errorf("fusermount/fusermount3 not found: %w", err)
		}
	}

	// Check if /dev/fuse exists
	if _, err := os.Stat("/dev/fuse"); err != nil {
		return false, fmt.Errorf("/dev/fuse not available: %w", err)
	}

	return true, nil
}

// Mount creates the overlay mount using fuse-overlayfs.
//
// In addition to the project-root overlay (lower=ScopePath, merged at
// MergedDir), Mount also brings up a per-sandbox $HOME overlay:
// lower=host $HOME, upper=<sandboxDir>/home-upper, merged at
// <sandboxDir>/home-merged. The home overlay is the audit-of-change
// mechanism for HOME-relative writes (auth tokens, tool caches, etc.):
// reads pass through to the host home, writes land in the per-sandbox
// upper layer that's discarded at sandbox teardown. Without this, agent
// CLIs that read $HOME/<tool>/<config> wouldn't find their host state.
func (d *FuseOverlayfsDriver) Mount(ctx context.Context, s *types.Sandbox) (*MountPaths, error) {
	sandboxDir := filepath.Join(d.config.BaseDir, s.ID.String())

	paths := &MountPaths{
		LowerDir:  s.ScopePath,
		UpperDir:  filepath.Join(sandboxDir, "upper"),
		WorkDir:   filepath.Join(sandboxDir, "work"),
		MergedDir: filepath.Join(sandboxDir, "merged"),
	}

	// Create directories
	for _, dir := range []string{paths.UpperDir, paths.WorkDir, paths.MergedDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Build mount options
	// fuse-overlayfs format: lowerdir=...,upperdir=...,workdir=...
	mountOpts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s",
		paths.LowerDir, paths.UpperDir, paths.WorkDir)

	// Mount using fuse-overlayfs
	cmd := exec.CommandContext(ctx, "fuse-overlayfs", "-o", mountOpts, paths.MergedDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Cleanup on failure
		os.RemoveAll(sandboxDir)
		return nil, fmt.Errorf("fuse-overlayfs mount failed: %v (output: %s)", err, strings.TrimSpace(string(output)))
	}

	// Best-effort home overlay. A failure here (no $HOME, fuse-overlayfs
	// rejecting the home dir, etc.) leaves the project overlay in place
	// and the home merged dir empty — the bwrap layer treats an empty
	// HomeMergedDir as "skip the home bind", so the sandbox still works
	// for callers that don't require host home visibility.
	if hostHome := os.Getenv("HOME"); hostHome != "" {
		if homePaths, hErr := mountHomeOverlay(ctx, sandboxDir, hostHome); hErr == nil {
			paths.HomeLowerDir = homePaths.HomeLowerDir
			paths.HomeUpperDir = homePaths.HomeUpperDir
			paths.HomeWorkDir = homePaths.HomeWorkDir
			paths.HomeMergedDir = homePaths.HomeMergedDir
		} else {
			// Don't fail the entire Mount on home-overlay failure —
			// surface the error in the log but keep the sandbox usable.
			fmt.Fprintf(os.Stderr, "home-overlay mount failed (continuing without it): %v\n", hErr)
		}
	}

	return paths, nil
}

// mountHomeOverlay sets up the per-sandbox fuse-overlayfs over the
// host $HOME and returns the resulting paths. Caller is responsible for
// tearing it down via unmountHomeOverlay (typically in Driver.Unmount).
func mountHomeOverlay(ctx context.Context, sandboxDir, hostHome string) (*MountPaths, error) {
	paths := &MountPaths{
		HomeLowerDir:  hostHome,
		HomeUpperDir:  filepath.Join(sandboxDir, "home-upper"),
		HomeWorkDir:   filepath.Join(sandboxDir, "home-work"),
		HomeMergedDir: filepath.Join(sandboxDir, "home-merged"),
	}
	for _, dir := range []string{paths.HomeUpperDir, paths.HomeWorkDir, paths.HomeMergedDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create home overlay dir %s: %w", dir, err)
		}
	}
	mountOpts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s",
		paths.HomeLowerDir, paths.HomeUpperDir, paths.HomeWorkDir)
	cmd := exec.CommandContext(ctx, "fuse-overlayfs", "-o", mountOpts, paths.HomeMergedDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("fuse-overlayfs home mount failed: %v (output: %s)", err, strings.TrimSpace(string(output)))
	}
	return paths, nil
}

// unmountHomeOverlay tears down the home overlay if mounted. Best-
// effort: returns nil if the dir is missing or already unmounted.
func unmountHomeOverlay(ctx context.Context, sandboxDir string) error {
	homeMerged := filepath.Join(sandboxDir, "home-merged")
	if _, err := os.Stat(homeMerged); err != nil {
		return nil // no home overlay
	}
	if !isMountPoint(homeMerged) {
		return nil
	}
	fusermount := "fusermount"
	if _, err := exec.LookPath(fusermount); err != nil {
		fusermount = "fusermount3"
	}
	cmd := exec.CommandContext(ctx, fusermount, "-u", homeMerged)
	if _, err := cmd.CombinedOutput(); err == nil {
		return nil
	}
	cmd = exec.CommandContext(ctx, fusermount, "-u", "-z", homeMerged)
	output, err := cmd.CombinedOutput()
	if err != nil && isMountPoint(homeMerged) {
		return fmt.Errorf("home overlay unmount failed: %v (output: %s)", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// Unmount removes the fuse-overlayfs mounts (project + home).
func (d *FuseOverlayfsDriver) Unmount(ctx context.Context, s *types.Sandbox) error {
	// Tear down the home overlay first. Best-effort: a failure here
	// must not block the project-overlay unmount or the caller's
	// teardown, so we log it and continue.
	if s.MergedDir != "" {
		sandboxDir := filepath.Dir(s.MergedDir)
		if err := unmountHomeOverlay(ctx, sandboxDir); err != nil {
			fmt.Fprintf(os.Stderr, "home overlay unmount failed (continuing with project unmount): %v\n", err)
		}
	}

	if s.MergedDir == "" {
		return nil // Nothing to unmount
	}

	// Check if mounted first
	if !isMountPoint(s.MergedDir) {
		return nil // Already unmounted
	}

	// Use fusermount for FUSE unmounting
	fusermount := "fusermount"
	if _, err := exec.LookPath(fusermount); err != nil {
		fusermount = "fusermount3"
	}

	// Try normal unmount first
	cmd := exec.CommandContext(ctx, fusermount, "-u", s.MergedDir)
	_, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}

	// Try lazy unmount if normal fails
	cmd = exec.CommandContext(ctx, fusermount, "-u", "-z", s.MergedDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if already unmounted
		if !isMountPoint(s.MergedDir) {
			return nil
		}
		return fmt.Errorf("fusermount unmount failed: %v (output: %s)", err, strings.TrimSpace(string(output)))
	}

	return nil
}

// GetChangedFiles returns the list of files changed in the upper layer.
// Delegates to shared helper for overlayfs-based change detection.
func (d *FuseOverlayfsDriver) GetChangedFiles(ctx context.Context, s *types.Sandbox) ([]*types.FileChange, error) {
	return getOverlayChangedFiles(s)
}

// RemoveFromUpper removes a file from the upper layer.
// Delegates to shared helper with path traversal protection.
func (d *FuseOverlayfsDriver) RemoveFromUpper(ctx context.Context, s *types.Sandbox, filePath string) error {
	if s.UpperDir == "" {
		return fmt.Errorf("sandbox has no upper directory configured")
	}
	return removeFromUpperSecure(s.UpperDir, filePath)
}

// Cleanup removes all sandbox artifacts.
// Delegates to shared helper with driver-specific unmount.
func (d *FuseOverlayfsDriver) Cleanup(ctx context.Context, s *types.Sandbox) error {
	return cleanupSandboxDir(d.config.BaseDir, s.ID, func() error {
		return d.Unmount(ctx, s)
	})
}

// ListSandboxDirs walks BaseDir and returns the IDs of every UUID-named
// subdirectory. Used by the orphan reconciler to detect sandboxes the
// repository has lost track of.
func (d *FuseOverlayfsDriver) ListSandboxDirs(ctx context.Context) ([]uuid.UUID, error) {
	return listSandboxDirsInBase(d.config.BaseDir)
}

// CleanupOrphan releases a sandbox by ID alone. Mirrors Cleanup's
// shape but does not require a *types.Sandbox: the caller has nothing
// but a directory name from a filesystem walk. Both the project overlay
// (<sandboxDir>/merged) and the home overlay (<sandboxDir>/home-merged)
// are unmounted before the directory is removed.
//
// Idempotent: a missing dir, an already-unmounted overlay, or a
// fusermount error are all treated as best-effort. Returns the first
// hard rm-rf failure if the dir survives all attempts.
func (d *FuseOverlayfsDriver) CleanupOrphan(ctx context.Context, id uuid.UUID) error {
	sandboxDir := filepath.Join(d.config.BaseDir, id.String())
	if _, err := os.Stat(sandboxDir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat orphan sandbox dir %q: %w", sandboxDir, err)
	}

	mergedDir := filepath.Join(sandboxDir, "merged")
	homeMerged := filepath.Join(sandboxDir, "home-merged")

	// Unmount in the same order as Unmount(): home first, then project.
	// Best-effort — if either is already unmounted (or never mounted)
	// this is a no-op. We swallow errors because the rm -rf below is
	// the actual teardown; failed fusermount just leaves the FS in a
	// state RemoveAll cannot recover from, which we surface below.
	fusermount := "fusermount"
	if _, err := exec.LookPath(fusermount); err != nil {
		fusermount = "fusermount3"
	}
	for _, mountPath := range []string{homeMerged, mergedDir} {
		if !isMountPoint(mountPath) {
			continue
		}
		// Try a clean unmount, then a lazy unmount. Either way, ignore
		// the error: the subsequent RemoveAll surfaces any actual
		// "FS is wedged" condition.
		if out, err := exec.CommandContext(ctx, fusermount, "-u", mountPath).CombinedOutput(); err != nil {
			_, _ = exec.CommandContext(ctx, fusermount, "-u", "-z", mountPath).CombinedOutput()
			_ = out // discard; lazy unmount is the recovery path
		}
	}

	if err := os.RemoveAll(sandboxDir); err != nil {
		return fmt.Errorf("remove orphan sandbox dir %q: %w", sandboxDir, err)
	}
	return nil
}

// IsMounted checks if the sandbox is currently mounted.
// Delegates to shared helper.
func (d *FuseOverlayfsDriver) IsMounted(ctx context.Context, s *types.Sandbox) (bool, error) {
	if s.MergedDir == "" {
		return false, nil
	}
	return isMountPoint(s.MergedDir), nil
}

// VerifyMountIntegrity checks that the mount is healthy.
// Delegates to shared helper.
func (d *FuseOverlayfsDriver) VerifyMountIntegrity(ctx context.Context, s *types.Sandbox) error {
	return verifyOverlayMountIntegrity(s)
}

// --- Process Execution Methods ---

// Exec executes a command in the sandbox with process isolation via bubblewrap.
// When bwrap is available, provides namespace isolation (network, PID, filesystem view).
// Falls back to direct execution if bwrap is unavailable, with a warning logged.
func (d *FuseOverlayfsDriver) Exec(ctx context.Context, s *types.Sandbox, cfg BwrapConfig, cmd string, args ...string) (*ExecResult, error) {
	if s.MergedDir == "" {
		return nil, fmt.Errorf("sandbox merged directory not set")
	}

	// Try to use bwrap for process isolation
	bwrapPath, err := exec.LookPath("bwrap")
	if err == nil {
		return d.execWithBwrap(ctx, s, cfg, bwrapPath, cmd, args...)
	}

	// Fallback to direct execution (no isolation)
	return d.execDirect(ctx, s, cfg, cmd, args...)
}

// execWithBwrap runs a command with bubblewrap isolation.
func (d *FuseOverlayfsDriver) execWithBwrap(ctx context.Context, s *types.Sandbox, cfg BwrapConfig, bwrapPath, cmd string, args ...string) (*ExecResult, error) {
	// Build bwrap command arguments using shared function
	bwrapArgs := buildBwrapArgs(s, cfg)

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
	err := execCmd.Run()

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

// execDirect runs a command directly without isolation (fallback when bwrap unavailable).
func (d *FuseOverlayfsDriver) execDirect(ctx context.Context, s *types.Sandbox, cfg BwrapConfig, cmd string, args ...string) (*ExecResult, error) {
	execCmd := exec.CommandContext(ctx, cmd, args...)
	execCmd.Dir = s.MergedDir

	// Set environment
	env := os.Environ()
	for k, v := range cfg.Env {
		env = append(env, k+"="+v)
	}
	execCmd.Env = env

	// Capture output
	var stdout, stderr strings.Builder
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	err := execCmd.Run()

	result := &ExecResult{
		Stdout: []byte(stdout.String()),
		Stderr: []byte(stderr.String()),
	}

	if execCmd.Process != nil {
		result.PID = execCmd.Process.Pid
	}

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

// StartProcess starts a background process in the sandbox with process isolation.
// When bwrap is available, provides namespace isolation.
// Falls back to direct execution if bwrap is unavailable.
func (d *FuseOverlayfsDriver) StartProcess(ctx context.Context, s *types.Sandbox, cfg BwrapConfig, cmd string, args ...string) (int, error) {
	if s.MergedDir == "" {
		return 0, fmt.Errorf("sandbox merged directory not set")
	}

	// Try to use bwrap for process isolation
	bwrapPath, err := exec.LookPath("bwrap")
	if err == nil {
		return d.startProcessWithBwrap(ctx, s, cfg, bwrapPath, cmd, args...)
	}

	// Fallback to direct execution (no isolation)
	return d.startProcessDirect(ctx, s, cfg, cmd, args...)
}

// startProcessWithBwrap starts a background process with bubblewrap isolation.
func (d *FuseOverlayfsDriver) startProcessWithBwrap(ctx context.Context, s *types.Sandbox, cfg BwrapConfig, bwrapPath, cmd string, args ...string) (int, error) {
	// Build bwrap args using shared function
	bwrapArgs := buildBwrapArgs(s, cfg)
	bwrapArgs = append(bwrapArgs, cmd)
	bwrapArgs = append(bwrapArgs, args...)

	execCmd := exec.Command(bwrapPath, bwrapArgs...)

	for k, v := range cfg.Env {
		execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	execCmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	wireStartProcessIO(execCmd, cfg)

	if err := execCmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start process: %w", err)
	}

	spawnExitReaper(execCmd, cfg.OnExit)

	return execCmd.Process.Pid, nil
}

// startProcessDirect starts a background process without isolation.
func (d *FuseOverlayfsDriver) startProcessDirect(ctx context.Context, s *types.Sandbox, cfg BwrapConfig, cmd string, args ...string) (int, error) {
	execCmd := exec.Command(cmd, args...)
	execCmd.Dir = s.MergedDir

	env := os.Environ()
	for k, v := range cfg.Env {
		env = append(env, k+"="+v)
	}
	execCmd.Env = env

	execCmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	wireStartProcessIO(execCmd, cfg)

	if err := execCmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start process: %w", err)
	}

	spawnExitReaper(execCmd, cfg.OnExit)

	return execCmd.Process.Pid, nil
}

// IsFuseOverlayfsAvailable checks if fuse-overlayfs is available on this system.
func IsFuseOverlayfsAvailable() (bool, string, error) {
	// Check if fuse-overlayfs is installed
	path, err := exec.LookPath("fuse-overlayfs")
	if err != nil {
		return false, "", fmt.Errorf("fuse-overlayfs not found")
	}

	// Get version
	cmd := exec.Command(path, "--version")
	output, err := cmd.Output()
	if err != nil {
		return true, path, nil // Installed but can't get version
	}

	return true, strings.TrimSpace(string(output)), nil
}
