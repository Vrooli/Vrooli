// Package driver provides filesystem drivers for sandbox isolation.
//
// copy.go implements a cross-platform fallback driver that works on any OS
// by using file copies instead of overlayfs.
//
// [OT-P2-004] Cross-Platform Driver Interface
package driver

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	execStd "os/exec"
	"path/filepath"
	"strings"
	"time"

	"workspace-sandbox/internal/types"
)

// CopyDriver implements the Driver interface using file copies.
// This is a cross-platform fallback driver that works on any OS.
//
// # How It Works
//
// Instead of overlayfs, CopyDriver:
//  1. Copies the scope directory to an "original" snapshot directory
//  2. Creates a "workspace" directory where actual work happens
//  3. On diff, compares workspace against original to detect changes
//  4. On approval, copies changed files back to the canonical repo
//
// # Trade-offs vs OverlayfsDriver
//
// Pros:
//   - Works on any OS (macOS, Windows, Linux without overlayfs)
//   - No kernel dependencies or special privileges needed
//   - Simple implementation, easy to debug
//
// Cons:
//   - Higher disk usage (full copy of scope directory)
//   - Slower sandbox creation for large directories
//   - No true copy-on-write semantics
//
// # Recommended Use Cases
//
//   - Development on macOS/Windows
//   - Small to medium scope directories
//   - When overlayfs is unavailable
type CopyDriver struct {
	config Config
}

// NewCopyDriver creates a new copy-based fallback driver.
func NewCopyDriver(cfg Config) *CopyDriver {
	if cfg.BaseDir == "" {
		cfg.BaseDir = DefaultConfig().BaseDir
	}
	return &CopyDriver{config: cfg}
}

// Type returns the driver type.
func (d *CopyDriver) Type() DriverType {
	return DriverTypeCopy
}

// Version returns the driver version.
func (d *CopyDriver) Version() string {
	return "1.0"
}

// IsAvailable checks if the copy driver can operate on this system.
// The copy driver is always available as it uses only standard file operations.
func (d *CopyDriver) IsAvailable(ctx context.Context) (bool, error) {
	// Copy driver is always available
	return true, nil
}

// Mount creates the sandbox workspace by copying the scope directory.
//
// Directory structure:
//
//	{baseDir}/{sandboxID}/
//	  original/   - snapshot of scope at creation time (read-only reference)
//	  workspace/  - working directory where changes are made
//
// Note: We don't have a separate merged dir like overlayfs.
// The workspace IS the merged view.
func (d *CopyDriver) Mount(ctx context.Context, s *types.Sandbox) (*MountPaths, error) {
	sandboxDir := filepath.Join(d.config.BaseDir, s.ID.String())

	paths := &MountPaths{
		LowerDir:  filepath.Join(sandboxDir, "original"),  // Snapshot of original
		UpperDir:  filepath.Join(sandboxDir, "workspace"), // Working directory
		WorkDir:   filepath.Join(sandboxDir, "meta"),      // Metadata/temp storage
		MergedDir: filepath.Join(sandboxDir, "workspace"), // Same as UpperDir for copy driver
	}

	// Create directories
	for _, dir := range []string{paths.LowerDir, paths.UpperDir, paths.WorkDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Copy scope directory to both original and workspace
	// Original serves as the baseline for diff generation
	// Workspace is where actual work happens
	if err := copyDirectory(s.ScopePath, paths.LowerDir); err != nil {
		os.RemoveAll(sandboxDir)
		return nil, fmt.Errorf("failed to copy scope to original: %w", err)
	}

	if err := copyDirectory(s.ScopePath, paths.UpperDir); err != nil {
		os.RemoveAll(sandboxDir)
		return nil, fmt.Errorf("failed to copy scope to workspace: %w", err)
	}

	return paths, nil
}

// Unmount is a no-op for the copy driver since there's no actual mount.
func (d *CopyDriver) Unmount(ctx context.Context, s *types.Sandbox) error {
	// No mount to unmount
	return nil
}

// GetChangedFiles returns the list of files that differ between workspace and original.
func (d *CopyDriver) GetChangedFiles(ctx context.Context, s *types.Sandbox) ([]*types.FileChange, error) {
	if s.UpperDir == "" {
		return nil, fmt.Errorf("sandbox workspace directory not set")
	}

	originalDir := s.LowerDir
	workspaceDir := s.UpperDir

	var changes []*types.FileChange

	// Track files we've seen in workspace
	workspaceFiles := make(map[string]bool)

	// Walk workspace to find added and modified files
	err := filepath.Walk(workspaceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory
		if path == workspaceDir {
			return nil
		}

		relPath, err := filepath.Rel(workspaceDir, path)
		if err != nil {
			return err
		}

		// Skip hidden files and directories that might be metadata
		if strings.HasPrefix(relPath, ".") {
			return nil
		}

		workspaceFiles[relPath] = true

		// Check if file exists in original
		originalPath := filepath.Join(originalDir, relPath)
		originalInfo, originalErr := os.Stat(originalPath)

		var changeType types.ChangeType

		if os.IsNotExist(originalErr) {
			// File doesn't exist in original - it's added
			changeType = types.ChangeTypeAdded
		} else if originalErr != nil {
			// Error accessing original - treat as modified
			changeType = types.ChangeTypeModified
		} else if info.IsDir() && originalInfo.IsDir() {
			// Both are directories - skip
			return nil
		} else if filesAreDifferent(originalPath, path, originalInfo, info) {
			// Files are different - modified
			changeType = types.ChangeTypeModified
		} else {
			// Files are the same - no change
			return nil
		}

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
		return nil, fmt.Errorf("failed to walk workspace directory: %w", err)
	}

	// Walk original to find deleted files
	err = filepath.Walk(originalDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == originalDir {
			return nil
		}

		relPath, err := filepath.Rel(originalDir, path)
		if err != nil {
			return err
		}

		// Skip hidden files
		if strings.HasPrefix(relPath, ".") {
			return nil
		}

		// If we already saw this in workspace, it's not deleted
		if workspaceFiles[relPath] {
			return nil
		}

		// File exists in original but not in workspace - deleted
		if !info.IsDir() {
			change := &types.FileChange{
				ID:             StableFileID(s.ID, relPath),
				SandboxID:      s.ID,
				FilePath:       relPath,
				ChangeType:     types.ChangeTypeDeleted,
				FileSize:       info.Size(),
				FileMode:       int(info.Mode()),
				DetectedAt:     time.Now(),
				ApprovalStatus: types.ApprovalPending,
			}
			changes = append(changes, change)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk original directory: %w", err)
	}

	return changes, nil
}

// RemoveFromUpper removes a file from the workspace after partial approval.
func (d *CopyDriver) RemoveFromUpper(ctx context.Context, s *types.Sandbox, filePath string) error {
	if s.UpperDir == "" {
		return fmt.Errorf("sandbox has no workspace directory configured")
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

	// Verify path is under workspace
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}
	absUpperDir, err := filepath.Abs(s.UpperDir)
	if err != nil {
		return fmt.Errorf("failed to resolve workspace dir: %w", err)
	}
	if !strings.HasPrefix(absFullPath, absUpperDir) {
		return fmt.Errorf("path escapes workspace directory: %s", filePath)
	}

	// Remove the file
	err = os.Remove(fullPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove file from workspace: %w", err)
	}

	return nil
}

// Cleanup removes all sandbox artifacts.
func (d *CopyDriver) Cleanup(ctx context.Context, s *types.Sandbox) error {
	sandboxDir := filepath.Join(d.config.BaseDir, s.ID.String())
	if err := os.RemoveAll(sandboxDir); err != nil {
		return fmt.Errorf("failed to remove sandbox directory: %w", err)
	}
	return nil
}

// IsMounted always returns true for copy driver since there's no actual mount.
func (d *CopyDriver) IsMounted(ctx context.Context, s *types.Sandbox) (bool, error) {
	// Check if the workspace directory exists
	if s.MergedDir == "" {
		return false, nil
	}
	_, err := os.Stat(s.MergedDir)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// VerifyMountIntegrity checks that the workspace directories are intact.
func (d *CopyDriver) VerifyMountIntegrity(ctx context.Context, s *types.Sandbox) error {
	if s.MergedDir == "" {
		return fmt.Errorf("workspace directory path is empty")
	}

	info, err := os.Stat(s.MergedDir)
	if os.IsNotExist(err) {
		return fmt.Errorf("workspace directory does not exist: %s", s.MergedDir)
	}
	if err != nil {
		return fmt.Errorf("cannot stat workspace directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("workspace path is not a directory: %s", s.MergedDir)
	}

	// Check original dir exists too
	if s.LowerDir != "" {
		info, err = os.Stat(s.LowerDir)
		if os.IsNotExist(err) {
			return fmt.Errorf("original directory does not exist: %s", s.LowerDir)
		}
		if err != nil {
			return fmt.Errorf("cannot stat original directory: %w", err)
		}
		if !info.IsDir() {
			return fmt.Errorf("original path is not a directory: %s", s.LowerDir)
		}
	}

	return nil
}

// --- Process Execution Methods (OT-P0-003) ---

// Exec executes a command in the sandbox workspace.
//
// Unlike OverlayfsDriver, the CopyDriver doesn't use bubblewrap for isolation.
// It simply runs the command in the workspace directory.
//
// Note: This provides no additional isolation - the process has full access
// to the filesystem. For production use with untrusted code, use OverlayfsDriver
// on Linux.
func (d *CopyDriver) Exec(ctx context.Context, s *types.Sandbox, cfg BwrapConfig, cmd string, args ...string) (*ExecResult, error) {
	if s.MergedDir == "" {
		return nil, fmt.Errorf("sandbox workspace directory not set")
	}

	// For copy driver, we just run the command directly in the workspace
	// No bwrap isolation available
	execCmd := execStd.CommandContext(ctx, cmd, args...)
	execCmd.Dir = s.MergedDir

	// Set environment
	env := os.Environ()
	for k, v := range cfg.Env {
		env = append(env, k+"="+v)
	}
	execCmd.Env = env

	stdout, err := execCmd.Output()
	exitCode := 0
	var stderr []byte

	if err != nil {
		if exitErr, ok := err.(*execStd.ExitError); ok {
			exitCode = exitErr.ExitCode()
			stderr = exitErr.Stderr
		} else {
			return nil, fmt.Errorf("failed to execute command: %w", err)
		}
	}

	return &ExecResult{
		ExitCode: exitCode,
		Stdout:   stdout,
		Stderr:   stderr,
		PID:      0, // Not tracked for simple exec
	}, nil
}

// StartProcess starts a background process in the sandbox workspace.
//
// Unlike OverlayfsDriver, this doesn't use bubblewrap for isolation.
// The process runs directly in the workspace directory.
//
// If cfg.LogWriter is provided, stdout and stderr are redirected to it.
// The caller is responsible for closing the LogWriter when the process exits.
func (d *CopyDriver) StartProcess(ctx context.Context, s *types.Sandbox, cfg BwrapConfig, cmd string, args ...string) (int, error) {
	if s.MergedDir == "" {
		return 0, fmt.Errorf("sandbox workspace directory not set")
	}

	// For copy driver, we just start the command directly
	execCmd := execStd.Command(cmd, args...)
	execCmd.Dir = s.MergedDir

	// Set environment
	env := os.Environ()
	for k, v := range cfg.Env {
		env = append(env, k+"="+v)
	}
	execCmd.Env = env

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


// --- Helper Functions ---

// copyDirectory recursively copies a directory tree.
func copyDirectory(src, dst string) error {
	return filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath, info.Mode())
	})
}

// copyFile copies a single file.
func copyFile(src, dst string, mode fs.FileMode) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// filesAreDifferent checks if two files have different content.
func filesAreDifferent(path1, path2 string, info1, info2 fs.FileInfo) bool {
	// Different sizes means different content
	if info1.Size() != info2.Size() {
		return true
	}

	// Different modes means different
	if info1.Mode() != info2.Mode() {
		return true
	}

	// For small files, compare content directly
	if info1.Size() < 64*1024 { // 64KB threshold
		content1, err1 := os.ReadFile(path1)
		content2, err2 := os.ReadFile(path2)
		if err1 != nil || err2 != nil {
			return true
		}
		return string(content1) != string(content2)
	}

	// For larger files, modification time is a reasonable heuristic
	// (though not perfect - content could differ even with same mtime)
	return info1.ModTime() != info2.ModTime()
}
