// Package driver provides sandbox driver interfaces and implementations.
//
// helpers.go contains shared helper functions for overlayfs-based drivers.
// These functions are used by every OverlayDriver flavor (kernel and FUSE)
// to avoid code duplication while maintaining proper separation of concerns.
//
// All mount/unmount operations route through fsmount.Mounter so syscalls
// stay confined to the canonical seam (Round 4 Phase 7).
package driver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"workspace-sandbox/internal/fsmount"
	"workspace-sandbox/internal/types"
)

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
// All syscalls go through `m`. `backend` selects kernel vs userspace.
//
// DOC: home-overlay seam (project overlay is its sibling). See
// docs/internal/SEAMS.md.
func mountProjectOverlay(ctx context.Context, m fsmount.Mounter, backend fsmount.Backend, sandboxDir, scopePath string) (*MountPaths, error) {
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
	if err := m.Mount(ctx, fsmount.MountOpts{
		Backend: backend,
		Lower:   paths.LowerDir,
		Upper:   paths.UpperDir,
		Work:    paths.WorkDir,
		Merged:  paths.MergedDir,
	}); err != nil {
		os.RemoveAll(sandboxDir)
		return nil, fmt.Errorf("project overlay mount: %w", err)
	}
	if err := verifyMounted(m, paths.MergedDir); err != nil {
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
func mountHomeOverlay(ctx context.Context, m fsmount.Mounter, backend fsmount.Backend, homeOverlayBaseDir string, sandboxID uuid.UUID, hostHome string) (lower, upper, work, merged string, err error) {
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
	if mountErr := m.Mount(ctx, fsmount.MountOpts{
		Backend: backend,
		Lower:   hostHome,
		Upper:   upper,
		Work:    work,
		Merged:  merged,
	}); mountErr != nil {
		os.RemoveAll(parent)
		return "", "", "", "", types.NewHomeOverlayUnavailableError(fmt.Errorf("home overlay mount: %w", mountErr))
	}
	if verifyErr := verifyMounted(m, merged); verifyErr != nil {
		// Roll back the mount we just made: a successful mount that
		// doesn't actually appear as a mount point means a stale daemon
		// or kernel state slipped through; tear it down so we don't
		// leave a half-mount behind.
		_ = m.Unmount(ctx, merged, false)
		os.RemoveAll(parent)
		return "", "", "", "", types.NewHomeOverlayUnavailableError(fmt.Errorf("home overlay verify: %w", verifyErr))
	}
	return hostHome, upper, work, merged, nil
}

// verifyMounted is the post-mount sanity check. After Mount returns
// nil, we still need to confirm the mount actually exists in the kernel
// (defensive against fuse-overlayfs daemon crashes that fork-and-die),
// and that the resulting merged dir is readable + writable.
func verifyMounted(m fsmount.Mounter, merged string) error {
	if !m.IsMountPoint(merged) {
		return fmt.Errorf("post-mount verify: %s is not a mount point (mount returned nil but no kernel mount appeared)", merged)
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
func unmountHomeOverlay(ctx context.Context, m fsmount.Mounter, homeOverlayBaseDir string, sandboxID uuid.UUID) {
	if homeOverlayBaseDir == "" {
		return
	}
	parent := homeOverlayDir(homeOverlayBaseDir, sandboxID)
	homeMerged := filepath.Join(parent, "home-merged")
	if !m.IsMountPoint(homeMerged) {
		return
	}
	if err := m.Unmount(ctx, homeMerged, false); err != nil {
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
// Mount State Helpers
// =============================================================================
//
// IsMountPoint(path) used to live here as a package-level function calling
// `mountpoint -q`. It moved into fsmount.Mounter.IsMountPoint as part of
// Round 4 Phase 7 so the seam owns every mount-state observation.

// verifyOverlayMountIntegrity performs comprehensive health checks on an overlay mount.
// Returns nil if healthy, error describing the problem otherwise.
//
// Checks performed:
//   - Mount point exists and is a directory
//   - Mount is actually mounted (not just a directory)
//   - Mount is accessible (can list contents)
//   - Upper dir exists and is writable
func verifyOverlayMountIntegrity(m fsmount.Mounter, s *types.Sandbox) error {
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
	if !m.IsMountPoint(s.MergedDir) {
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
	dir := filepath.Dir(fullPath)
	for dir != absUpperDir && dir != "." && dir != "/" {
		if rmErr := os.Remove(dir); rmErr != nil {
			break
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
func cleanupSandboxDirAll(ctx context.Context, m fsmount.Mounter, baseDir, homeOverlayBaseDir string, sandboxID uuid.UUID) error {
	// Always attempt to release the home overlay first; even if the
	// project sandbox dir was already removed, a stale daemon may still
	// hold the home merge.
	unmountHomeOverlay(ctx, m, homeOverlayBaseDir, sandboxID)
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
	if m.IsMountPoint(projectMerged) {
		if err := m.Unmount(ctx, projectMerged, false); err != nil {
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
// belong to the driver's own bookkeeping, e.g., the driver preference
// file). A missing baseDir is not an error and returns an
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
			continue
		}
		out = append(out, id)
	}
	return out, nil
}
