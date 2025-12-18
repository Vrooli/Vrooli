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
	"time"

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

	return paths, nil
}

// Unmount removes the fuse-overlayfs mount.
func (d *FuseOverlayfsDriver) Unmount(ctx context.Context, s *types.Sandbox) error {
	if s.MergedDir == "" {
		return nil // Nothing to unmount
	}

	// Check if mounted first
	if !d.isMounted(s.MergedDir) {
		return nil // Already unmounted
	}

	// Use fusermount for FUSE unmounting
	fusermount := "fusermount"
	if _, err := exec.LookPath(fusermount); err != nil {
		fusermount = "fusermount3"
	}

	// Try normal unmount first
	cmd := exec.CommandContext(ctx, fusermount, "-u", s.MergedDir)
	output, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}

	// Try lazy unmount if normal fails
	cmd = exec.CommandContext(ctx, fusermount, "-u", "-z", s.MergedDir)
	output, err = cmd.CombinedOutput()
	if err != nil {
		// Check if already unmounted
		if !d.isMounted(s.MergedDir) {
			return nil
		}
		return fmt.Errorf("fusermount unmount failed: %v (output: %s)", err, strings.TrimSpace(string(output)))
	}

	return nil
}

// isMounted checks if a path is a mount point.
func (d *FuseOverlayfsDriver) isMounted(path string) bool {
	cmd := exec.Command("mountpoint", "-q", path)
	return cmd.Run() == nil
}

// GetChangedFiles returns the list of files changed in the upper layer.
func (d *FuseOverlayfsDriver) GetChangedFiles(ctx context.Context, s *types.Sandbox) ([]*types.FileChange, error) {
	if s.UpperDir == "" {
		return nil, fmt.Errorf("sandbox upper directory not set")
	}

	var changes []*types.FileChange

	err := filepath.Walk(s.UpperDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory
		if path == s.UpperDir {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(s.UpperDir, path)
		if err != nil {
			return err
		}

		// Skip overlayfs internal files
		if strings.HasPrefix(relPath, ".overlay") {
			return nil
		}

		// Determine change type
		changeType := d.detectChangeType(s, relPath, info)

		change := &types.FileChange{
			ID:             StableFileID(s.ID, relPath),
			SandboxID:      s.ID,
			FilePath:       relPath,
			ChangeType:     changeType,
			FileSize:       info.Size(),
			FileMode:       int(info.Mode()),
			DetectedAt:     time.Now(),
			ApprovalStatus: types.ApprovalPending,
		}

		changes = append(changes, change)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk upper directory: %w", err)
	}

	return changes, nil
}

// detectChangeType determines if a file was added, modified, or deleted.
func (d *FuseOverlayfsDriver) detectChangeType(s *types.Sandbox, relPath string, upperInfo os.FileInfo) types.ChangeType {
	lowerPath := filepath.Join(s.LowerDir, relPath)
	lowerInfo, err := os.Stat(lowerPath)

	if os.IsNotExist(err) {
		return types.ChangeTypeAdded
	}

	if err != nil {
		return types.ChangeTypeModified
	}

	// Check for whiteout (deletion marker)
	if d.isWhiteout(filepath.Join(s.UpperDir, relPath)) {
		return types.ChangeTypeDeleted
	}

	// Compare sizes and modes
	if lowerInfo.Size() != upperInfo.Size() || lowerInfo.Mode() != upperInfo.Mode() {
		return types.ChangeTypeModified
	}

	return types.ChangeTypeModified
}

// isWhiteout checks if a file is an overlayfs whiteout marker.
func (d *FuseOverlayfsDriver) isWhiteout(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}

	// fuse-overlayfs uses character devices with major 0, minor 0 for whiteouts
	if info.Mode()&os.ModeCharDevice != 0 {
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			return stat.Rdev == 0
		}
	}

	return false
}

// RemoveFromUpper removes a file from the upper layer.
func (d *FuseOverlayfsDriver) RemoveFromUpper(ctx context.Context, s *types.Sandbox, filePath string) error {
	if s.UpperDir == "" {
		return fmt.Errorf("sandbox has no upper directory configured")
	}

	// Security: ensure filePath is relative and doesn't escape
	cleanPath := filepath.Clean(filePath)
	if filepath.IsAbs(cleanPath) {
		cleanPath = strings.TrimPrefix(cleanPath, "/")
	}
	if strings.HasPrefix(cleanPath, "..") {
		return fmt.Errorf("path traversal not allowed: %s", filePath)
	}

	fullPath := filepath.Join(s.UpperDir, cleanPath)

	// Verify fullPath is under UpperDir
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}
	absUpperDir, err := filepath.Abs(s.UpperDir)
	if err != nil {
		return fmt.Errorf("failed to resolve upper dir: %w", err)
	}
	if !strings.HasPrefix(absFullPath, absUpperDir) {
		return fmt.Errorf("path escapes upper directory: %s", filePath)
	}

	// Remove the file
	err = os.Remove(fullPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove file: %w", err)
	}

	// Clean up empty parent directories
	dir := filepath.Dir(fullPath)
	for dir != absUpperDir && dir != "." && dir != "/" {
		if rmErr := os.Remove(dir); rmErr != nil {
			break
		}
		dir = filepath.Dir(dir)
	}

	return nil
}

// Cleanup removes all sandbox artifacts.
func (d *FuseOverlayfsDriver) Cleanup(ctx context.Context, s *types.Sandbox) error {
	// Unmount first
	if err := d.Unmount(ctx, s); err != nil {
		fmt.Printf("warning: unmount failed during cleanup: %v\n", err)
	}

	// Remove sandbox directory
	sandboxDir := filepath.Join(d.config.BaseDir, s.ID.String())
	if err := os.RemoveAll(sandboxDir); err != nil {
		return fmt.Errorf("failed to remove sandbox directory: %w", err)
	}

	return nil
}

// IsMounted checks if the sandbox is currently mounted.
func (d *FuseOverlayfsDriver) IsMounted(ctx context.Context, s *types.Sandbox) (bool, error) {
	if s.MergedDir == "" {
		return false, nil
	}
	return d.isMounted(s.MergedDir), nil
}

// VerifyMountIntegrity checks that the mount is healthy.
func (d *FuseOverlayfsDriver) VerifyMountIntegrity(ctx context.Context, s *types.Sandbox) error {
	if s.MergedDir == "" {
		return fmt.Errorf("merged directory path is empty")
	}

	info, err := os.Stat(s.MergedDir)
	if os.IsNotExist(err) {
		return fmt.Errorf("merged directory does not exist: %s", s.MergedDir)
	}
	if err != nil {
		return fmt.Errorf("cannot stat merged directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("merged path is not a directory: %s", s.MergedDir)
	}

	// Verify it's mounted
	if !d.isMounted(s.MergedDir) {
		return fmt.Errorf("merged directory is not mounted: %s", s.MergedDir)
	}

	// Check accessibility
	entries, err := os.ReadDir(s.MergedDir)
	if err != nil {
		return fmt.Errorf("merged directory is not accessible: %w", err)
	}
	_ = entries

	// Check upper dir
	if s.UpperDir != "" {
		upperInfo, err := os.Stat(s.UpperDir)
		if err != nil {
			return fmt.Errorf("upper directory check failed: %w", err)
		}
		if !upperInfo.IsDir() {
			return fmt.Errorf("upper path is not a directory: %s", s.UpperDir)
		}
	}

	return nil
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

	// Redirect output to log writer if provided
	if cfg.LogWriter != nil {
		execCmd.Stdout = cfg.LogWriter
		execCmd.Stderr = cfg.LogWriter
	}

	// Start the process
	if err := execCmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start process: %w", err)
	}

	return execCmd.Process.Pid, nil
}

// startProcessDirect starts a background process without isolation.
func (d *FuseOverlayfsDriver) startProcessDirect(ctx context.Context, s *types.Sandbox, cfg BwrapConfig, cmd string, args ...string) (int, error) {
	execCmd := exec.Command(cmd, args...)
	execCmd.Dir = s.MergedDir

	// Set environment
	env := os.Environ()
	for k, v := range cfg.Env {
		env = append(env, k+"="+v)
	}
	execCmd.Env = env

	// Set up process group for cleanup
	execCmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Redirect output to log writer if provided
	if cfg.LogWriter != nil {
		execCmd.Stdout = cfg.LogWriter
		execCmd.Stderr = cfg.LogWriter
	}

	if err := execCmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start process: %w", err)
	}

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
