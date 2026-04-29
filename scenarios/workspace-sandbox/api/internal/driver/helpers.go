// Package driver provides sandbox driver interfaces and implementations.
//
// helpers.go contains shared helper functions for overlayfs-based drivers.
// These functions are used by every OverlayDriver flavor (kernel and FUSE)
// to avoid code duplication while maintaining proper separation of concerns.
//
// The pattern follows bwrap.go - package-level functions that drivers call
// rather than embedded structs, which is more idiomatic Go.
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

	"github.com/google/uuid"

	"workspace-sandbox/internal/types"
)

// mountFunc performs the actual overlay mount given the option string and
// target path. Driver-specific (kernel overlayfs vs fuse-overlayfs).
type mountFunc func(ctx context.Context, target, opts string) error

// unmountFunc tears down a mount at target. Must be idempotent: returning
// nil for an already-unmounted target.
type unmountFunc func(ctx context.Context, target string) error

// =============================================================================
// Overlayfs Mount Helpers (shared template for the two overlayfs drivers)
// =============================================================================

// mountProjectOverlay mounts the per-sandbox project overlay rooted at
// sandboxDir (under config.BaseDir). The lower layer is `scopePath`.
// Failure is fatal and triggers cleanup of sandboxDir.
//
// Layout:
//
//	sandboxDir/
//	  upper/   (writable layer for project changes)
//	  work/    (overlayfs scratch)
//	  merged/  (the mount point)
//
// `mountFn` is the only driver-specific bit (kernel overlayfs syscall vs
// fuse-overlayfs binary).
//
// DOC: home-overlay seam (project overlay is its sibling). See
// docs/internal/SEAMS.md.
func mountProjectOverlay(ctx context.Context, sandboxDir, scopePath string, mountFn mountFunc) (*MountPaths, error) {
	paths := &MountPaths{
		LowerDir:  scopePath,
		UpperDir:  filepath.Join(sandboxDir, "upper"),
		WorkDir:   filepath.Join(sandboxDir, "work"),
		MergedDir: filepath.Join(sandboxDir, "merged"),
	}
	for _, dir := range []string{paths.UpperDir, paths.WorkDir, paths.MergedDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			os.RemoveAll(sandboxDir)
			return nil, fmt.Errorf("create overlay dir %s: %w", dir, err)
		}
	}
	opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s",
		paths.LowerDir, paths.UpperDir, paths.WorkDir)
	if err := mountFn(ctx, paths.MergedDir, opts); err != nil {
		os.RemoveAll(sandboxDir)
		return nil, fmt.Errorf("project overlay mount: %w", err)
	}
	if err := verifyMounted(paths.MergedDir); err != nil {
		_ = mountFn // unused fallback; we cannot unmount here without unmountFn
		os.RemoveAll(sandboxDir)
		return nil, fmt.Errorf("project overlay verify: %w", err)
	}
	return paths, nil
}

// homeOverlayDir returns the per-sandbox home-overlay parent directory
// (homeOverlayBaseDir/<sandboxID>). Used by both mountHomeOverlay and the
// orphan reconciler.
//
// DOC: home-overlay storage seam. See docs/internal/SEAMS.md.
func homeOverlayDir(homeOverlayBaseDir string, sandboxID uuid.UUID) string {
	return filepath.Join(homeOverlayBaseDir, sandboxID.String())
}

// mountHomeOverlay mounts a per-sandbox host-$HOME overlay rooted at
// homeOverlayBaseDir/<sandboxID>. The base directory MUST be outside
// $HOME (validated at config load time) — placing the upper layer inside
// $HOME (the lower layer) creates a self-referential overlayfs mount
// whose behavior is undefined per kernel docs.
//
// Returns a typed *types.HomeOverlayUnavailableError on failure. Callers
// (overlayfs.go, fuse_overlayfs.go) must NOT silently swallow this;
// they should log structured warning, set Sandbox.HomeOverlayState =
// HomeOverlayAbsent, and return success — the sandbox itself is still
// useful for profiles that don't require the overlay.
//
// Layout:
//
//	homeOverlayBaseDir/<sandboxID>/
//	  home-upper/    (writable layer for $HOME writes)
//	  home-work/     (overlayfs scratch)
//	  home-merged/   (the mount point bound at /home/<user> by bwrap)
//
// DOC: home-overlay seam. See docs/internal/SEAMS.md.
func mountHomeOverlay(ctx context.Context, homeOverlayBaseDir string, sandboxID uuid.UUID, hostHome string, mountFn mountFunc, unmountFn unmountFunc) (lower, upper, work, merged string, err error) {
	if homeOverlayBaseDir == "" {
		return "", "", "", "", types.NewHomeOverlayUnavailableError(fmt.Errorf("HomeOverlayBaseDir is empty; config.ResolveHomeOverlayBaseDir was not called"))
	}
	parent := homeOverlayDir(homeOverlayBaseDir, sandboxID)
	upper = filepath.Join(parent, "home-upper")
	work = filepath.Join(parent, "home-work")
	merged = filepath.Join(parent, "home-merged")
	for _, dir := range []string{upper, work, merged} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			os.RemoveAll(parent)
			return "", "", "", "", types.NewHomeOverlayUnavailableError(fmt.Errorf("create home overlay dir %s: %w", dir, err))
		}
	}
	opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", hostHome, upper, work)
	if err := mountFn(ctx, merged, opts); err != nil {
		os.RemoveAll(parent)
		return "", "", "", "", types.NewHomeOverlayUnavailableError(fmt.Errorf("home overlay mount: %w", err))
	}
	if err := verifyMounted(merged); err != nil {
		// Roll back the mount we just made: a successful mount-cmd that
		// doesn't actually appear as a mount point means a stale daemon
		// or kernel state slipped through; tear it down so we don't
		// leave a half-mount behind.
		if unmountFn != nil {
			_ = unmountFn(ctx, merged)
		}
		os.RemoveAll(parent)
		return "", "", "", "", types.NewHomeOverlayUnavailableError(fmt.Errorf("home overlay verify: %w", err))
	}
	return hostHome, upper, work, merged, nil
}

// verifyMounted is the post-mount sanity check. After mountFn returns
// nil, we still need to confirm the mount actually exists in the kernel
// (defensive against fuse-overlayfs daemon crashes that fork-and-die),
// and that the resulting merged dir is readable + writable.
func verifyMounted(merged string) error {
	if !isMountPoint(merged) {
		return fmt.Errorf("post-mount verify: %s is not a mount point (mountFn returned nil but no kernel mount appeared)", merged)
	}
	info, err := os.Stat(merged)
	if err != nil {
		return fmt.Errorf("post-mount stat %s: %w", merged, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("post-mount %s is not a directory", merged)
	}
	// Writable probe: create + delete a 0-byte file. This catches
	// read-only mounts that pass mountpoint(1) but would fail at runtime
	// for any agent CLI trying to write to e.g. ~/.config/claude/.
	probe := filepath.Join(merged, ".vrooli-mount-probe")
	f, err := os.OpenFile(probe, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("post-mount writable-probe (create) %s: %w", probe, err)
	}
	_ = f.Close()
	if err := os.Remove(probe); err != nil {
		return fmt.Errorf("post-mount writable-probe (remove) %s: %w", probe, err)
	}
	return nil
}

// unmountHomeOverlay tears down homeOverlayBaseDir/<sandboxID>/home-merged
// if it is mounted. Best-effort: errors are logged and discarded so
// callers can proceed with the project unmount and rm -rf.
//
// DOC: home-overlay storage seam. See docs/internal/SEAMS.md.
func unmountHomeOverlay(ctx context.Context, homeOverlayBaseDir string, sandboxID uuid.UUID, unmountFn unmountFunc) {
	if homeOverlayBaseDir == "" {
		return
	}
	parent := homeOverlayDir(homeOverlayBaseDir, sandboxID)
	homeMerged := filepath.Join(parent, "home-merged")
	if !isMountPoint(homeMerged) {
		return
	}
	if err := unmountFn(ctx, homeMerged); err != nil {
		fmt.Fprintf(os.Stderr, "home overlay unmount failed: %v\n", err)
	}
}

// removeHomeOverlayDir removes the per-sandbox home-overlay parent
// directory (homeOverlayBaseDir/<sandboxID>) after it has been
// unmounted. Idempotent.
func removeHomeOverlayDir(homeOverlayBaseDir string, sandboxID uuid.UUID) error {
	if homeOverlayBaseDir == "" {
		return nil
	}
	parent := homeOverlayDir(homeOverlayBaseDir, sandboxID)
	if _, err := os.Stat(parent); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat home overlay dir %q: %w", parent, err)
	}
	if err := os.RemoveAll(parent); err != nil {
		return fmt.Errorf("remove home overlay dir %q: %w", parent, err)
	}
	return nil
}

// listHomeOverlayDirs returns the IDs of all UUID-named subdirectories
// directly under homeOverlayBaseDir. Used by the orphan reconciler to
// detect home-overlay dirs whose project-side sandbox is gone.
func listHomeOverlayDirs(homeOverlayBaseDir string) ([]uuid.UUID, error) {
	if homeOverlayBaseDir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(homeOverlayBaseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read homeOverlayBaseDir %q: %w", homeOverlayBaseDir, err)
	}
	out := make([]uuid.UUID, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id, err := uuid.Parse(e.Name())
		if err != nil {
			continue
		}
		out = append(out, id)
	}
	return out, nil
}

// =============================================================================
// Overlayfs Change Detection Helpers
// =============================================================================

// getOverlayChangedFiles walks the upper directory and builds a list of FileChange
// objects representing files that have been added, modified, or deleted.
// Used by every OverlayDriver flavor (kernel and FUSE).
func getOverlayChangedFiles(s *types.Sandbox) ([]*types.FileChange, error) {
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

		// Skip .git directory - git internal files should never be included in diffs.
		// This prevents sandbox changes from including git objects, index, etc.
		if relPath == ".git" || strings.HasPrefix(relPath, ".git"+string(filepath.Separator)) {
			return nil
		}

		// Skip directories - they are structural containers created by overlayfs
		// when files inside them are modified. The actual file changes are what
		// we want to track. Directory additions are implied by file additions,
		// and directory deletions are tracked via whiteout markers.
		if info.IsDir() {
			return nil
		}

		baseName := filepath.Base(relPath)
		if baseName == ".wh..opq" {
			return nil
		}
		if strings.HasPrefix(baseName, ".wh.") {
			targetName := strings.TrimPrefix(baseName, ".wh.")
			if targetName == "" {
				return nil
			}
			if strings.HasPrefix(targetName, ".wh.") || targetName == ".wh..opq" {
				return nil
			}

			targetRel := targetName
			if dir := filepath.Dir(relPath); dir != "." {
				targetRel = filepath.Join(dir, targetName)
			}

			var fileSize int64
			var fileMode int
			if lowerInfo, statErr := os.Stat(filepath.Join(s.LowerDir, targetRel)); statErr == nil {
				fileSize = lowerInfo.Size()
				fileMode = int(lowerInfo.Mode())
			}

			change := &types.FileChange{
				ID:             StableFileID(s.ID, targetRel),
				SandboxID:      s.ID,
				FilePath:       targetRel,
				ChangeType:     types.ChangeTypeDeleted,
				FileSize:       fileSize,
				FileMode:       fileMode,
				DetectedAt:     time.Now(),
				ApprovalStatus: types.ApprovalPending,
			}

			changes = append(changes, change)
			return nil
		}

		// Determine change type
		changeType := detectOverlayChangeType(s, relPath, info)

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

// detectOverlayChangeType determines if a file was added, modified, or deleted
// by comparing it against the lower (original) layer.
func detectOverlayChangeType(s *types.Sandbox, relPath string, upperInfo os.FileInfo) types.ChangeType {
	// Check if file exists in lower layer
	lowerPath := filepath.Join(s.LowerDir, relPath)
	lowerInfo, err := os.Stat(lowerPath)

	if os.IsNotExist(err) {
		return types.ChangeTypeAdded
	}

	if err != nil {
		// Can't determine, assume modified
		return types.ChangeTypeModified
	}

	// Check for overlayfs whiteout (deletion marker)
	if isWhiteout(filepath.Join(s.UpperDir, relPath)) {
		return types.ChangeTypeDeleted
	}

	// Compare sizes and modes
	if lowerInfo.Size() != upperInfo.Size() || lowerInfo.Mode() != upperInfo.Mode() {
		return types.ChangeTypeModified
	}

	// TODO: Could add content hash comparison for more accuracy
	return types.ChangeTypeModified
}

// isWhiteout checks if a file is an overlayfs whiteout marker.
// Overlayfs whiteouts are character devices with major 0, minor 0.
func isWhiteout(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}

	// Overlayfs whiteouts are character devices with major 0, minor 0
	if info.Mode()&os.ModeCharDevice != 0 {
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			return stat.Rdev == 0
		}
	}

	return false
}

// =============================================================================
// Mount State Helpers
// =============================================================================

// isMountPoint checks if a path is currently a mount point using the
// mountpoint command. Returns false if the command fails or path is not mounted.
func isMountPoint(path string) bool {
	cmd := exec.Command("mountpoint", "-q", path)
	return cmd.Run() == nil
}

// verifyOverlayMountIntegrity performs comprehensive health checks on an overlay mount.
// Returns nil if healthy, error describing the problem otherwise.
//
// Checks performed:
//   - Mount point exists and is a directory
//   - Mount is actually mounted (not just a directory)
//   - Mount is accessible (can list contents)
//   - Upper dir exists and is writable
func verifyOverlayMountIntegrity(s *types.Sandbox) error {
	// Check merged dir exists
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

	// Verify it's actually mounted
	if !isMountPoint(s.MergedDir) {
		return fmt.Errorf("merged directory is not mounted (may be stale): %s", s.MergedDir)
	}

	// Check accessibility by attempting to list the directory
	entries, err := os.ReadDir(s.MergedDir)
	if err != nil {
		return fmt.Errorf("merged directory is not accessible: %w", err)
	}
	_ = entries // We just want to verify access, don't need the entries

	// Check upper dir for write capability
	if s.UpperDir != "" {
		upperInfo, err := os.Stat(s.UpperDir)
		if err != nil {
			return fmt.Errorf("upper directory check failed: %w", err)
		}
		if !upperInfo.IsDir() {
			return fmt.Errorf("upper path is not a directory: %s", s.UpperDir)
		}

		// Try to verify write access by checking directory is writable
		// Note: We don't actually write a test file to avoid side effects
		if upperInfo.Mode().Perm()&0o200 == 0 {
			return fmt.Errorf("upper directory appears not writable: %s", s.UpperDir)
		}
	}

	return nil
}

// =============================================================================
// Secure File Removal Helpers
// =============================================================================

// removeFromUpperSecure removes a file from the upper (writable) layer with
// path traversal protection. This is used after partial approval to clean up
// applied files while preserving unapproved changes.
// Returns nil if file doesn't exist (idempotent).
//
// Security measures:
//   - Cleans and validates the path
//   - Rejects absolute paths and path traversal attempts
//   - Verifies resolved path is under upperDir (defense in depth)
//   - Cleans up empty parent directories
func removeFromUpperSecure(upperDir, filePath string) error {
	// Security: ensure filePath is relative and doesn't escape the sandbox
	cleanPath := filepath.Clean(filePath)
	if filepath.IsAbs(cleanPath) {
		// Strip leading slash for relative path construction
		cleanPath = strings.TrimPrefix(cleanPath, "/")
	}
	if strings.HasPrefix(cleanPath, "..") {
		return fmt.Errorf("path traversal not allowed: %s", filePath)
	}

	fullPath := filepath.Join(upperDir, cleanPath)

	// Verify fullPath is actually under upperDir (defense in depth)
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}
	absUpperDir, err := filepath.Abs(upperDir)
	if err != nil {
		return fmt.Errorf("failed to resolve upper dir: %w", err)
	}
	if !strings.HasPrefix(absFullPath, absUpperDir) {
		return fmt.Errorf("path escapes upper directory: %s", filePath)
	}

	// Remove the file - ignore not-exist errors (idempotent)
	err = os.Remove(fullPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove file from upper layer: %w", err)
	}

	// Also remove empty parent directories up to upperDir
	// This prevents leftover empty directories after file removal
	dir := filepath.Dir(fullPath)
	for dir != absUpperDir && dir != "." && dir != "/" {
		// Try to remove - will fail if not empty, which is fine
		if rmErr := os.Remove(dir); rmErr != nil {
			break // Directory not empty or error, stop
		}
		dir = filepath.Dir(dir)
	}

	return nil
}

// =============================================================================
// Cleanup Helpers
// =============================================================================

// cleanupSandboxDirAll unmounts both the home overlay (if present, lives
// under homeOverlayBaseDir) and the project overlay (lives under
// baseDir), then removes both per-sandbox directories. Idempotent: a
// missing dir is a no-op; unmount errors are logged and the rm -rf
// proceeds (the rm error is the actual fatal signal).
//
// Used by both Cleanup (sandbox struct path) and CleanupOrphan
// (filesystem-only path) on every mount-backed driver. The home overlay
// dir lives outside baseDir (under homeOverlayBaseDir) since Phase B —
// the upper layer cannot live inside $HOME without breaking overlayfs.
func cleanupSandboxDirAll(ctx context.Context, baseDir, homeOverlayBaseDir string, sandboxID uuid.UUID, unmountFn unmountFunc) error {
	// Always attempt to release the home overlay first; even if the
	// project sandbox dir was already removed, a stale daemon may still
	// hold the home merge.
	unmountHomeOverlay(ctx, homeOverlayBaseDir, sandboxID, unmountFn)
	if err := removeHomeOverlayDir(homeOverlayBaseDir, sandboxID); err != nil {
		fmt.Fprintf(os.Stderr, "home overlay dir cleanup: %v\n", err)
	}

	sandboxDir := filepath.Join(baseDir, sandboxID.String())
	if _, err := os.Stat(sandboxDir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat sandbox dir %q: %w", sandboxDir, err)
	}

	projectMerged := filepath.Join(sandboxDir, "merged")
	if isMountPoint(projectMerged) {
		if err := unmountFn(ctx, projectMerged); err != nil {
			fmt.Fprintf(os.Stderr, "project overlay unmount failed: %v\n", err)
		}
	}

	if err := os.RemoveAll(sandboxDir); err != nil {
		return fmt.Errorf("remove sandbox dir %q: %w", sandboxDir, err)
	}
	return nil
}

// mergeUUIDLists deduplicates and unions two (list, error) results.
// Either error short-circuits.
func mergeUUIDLists(a []uuid.UUID, errA error, b []uuid.UUID, errB error) ([]uuid.UUID, error) {
	if errA != nil {
		return nil, errA
	}
	if errB != nil {
		return nil, errB
	}
	seen := make(map[uuid.UUID]bool, len(a)+len(b))
	out := make([]uuid.UUID, 0, len(a)+len(b))
	for _, id := range a {
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	for _, id := range b {
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out, nil
}

// listSandboxDirsInBase returns the IDs of all UUID-named subdirectories
// directly under baseDir. Non-UUID directories and stray files are
// silently skipped — callers must NOT treat them as sandboxes (they may
// belong to the driver's own bookkeeping, e.g., LoadDriverPreference's
// preference file). A missing baseDir is not an error and returns an
// empty slice, since "no dirs to enumerate" is the same as "no orphans".
//
// This helper is shared across drivers because the layout convention
// (one UUID-named dir per sandbox under BaseDir) is identical for
// every OverlayDriver flavor and CopyDriver.
func listSandboxDirsInBase(baseDir string) ([]uuid.UUID, error) {
	if baseDir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read baseDir %q: %w", baseDir, err)
	}
	out := make([]uuid.UUID, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id, err := uuid.Parse(e.Name())
		if err != nil {
			continue // not a sandbox dir; skip silently
		}
		out = append(out, id)
	}
	return out, nil
}
