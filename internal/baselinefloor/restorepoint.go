package baselinefloor

import (
	"fmt"
	"os"
	"path/filepath"
)

// Capture takes a restore point: it copies the scenario working tree (srcDir)
// into restorePointDest using the copy ladder, excluding build artifacts. This
// is the git-free, code-level undo the engagement records; `baseline abandon`
// (live mode) and promote auto-rollback Restore from it.
//
// Capture is idempotent: an existing restore point is overlaid (re-running a
// capture refreshes it). The reflink fast path is requested — it is instant and
// edit-safe where supported, and transparently falls back to a deep copy on
// ext4/Windows. Callers that want a different exclude set pass opts.Exclude;
// nil opts use DefaultExcludes + reflink.
func Capture(srcDir, restorePointDest string, opts *CopyOptions) (CopyStats, error) {
	if err := ensureDir(srcDir); err != nil {
		return CopyStats{}, fmt.Errorf("baselinefloor: capture src: %w", err)
	}
	co := resolveCopyOptions(opts)
	if err := os.MkdirAll(filepath.Dir(restorePointDest), 0o750); err != nil {
		return CopyStats{}, fmt.Errorf("baselinefloor: capture mkdir %q: %w", restorePointDest, err)
	}
	stats, err := CopyTree(srcDir, restorePointDest, co)
	if err != nil {
		return stats, fmt.Errorf("baselinefloor: capture %q→%q: %w", srcDir, restorePointDest, err)
	}
	return stats, nil
}

// Restore overlays a previously captured restore point back onto destDir — the
// sanctioned, git-free undo (the only one; git stash/reset/checkout are banned).
//
// Restore is an OVERLAY, not a mirror: files in the restore point overwrite their
// counterparts in destDir, but a file created in destDir after the capture
// (e.g. a new source file the abandoned work added) is left in place. This is
// intentional and matches the `abandon` contract ("roll back to the restore
// point + park dirty work"); the engagement layer parks anything that diverges.
// Build artifacts were excluded from the capture and regenerate on restart.
func Restore(restorePointSrc, destDir string, opts *CopyOptions) (CopyStats, error) {
	if err := ensureDir(restorePointSrc); err != nil {
		return CopyStats{}, fmt.Errorf("baselinefloor: restore point: %w", err)
	}
	co := resolveCopyOptions(opts)
	if err := os.MkdirAll(destDir, 0o750); err != nil {
		return CopyStats{}, fmt.Errorf("baselinefloor: restore mkdir %q: %w", destDir, err)
	}
	stats, err := CopyTree(restorePointSrc, destDir, co)
	if err != nil {
		return stats, fmt.Errorf("baselinefloor: restore %q→%q: %w", restorePointSrc, destDir, err)
	}
	return stats, nil
}

// resolveCopyOptions applies the restore-point defaults (the .gitignore-aligned
// exclude set + reflink fast path) when the caller passes nil, and fills a nil
// Exclude map on an otherwise-supplied options value.
func resolveCopyOptions(opts *CopyOptions) CopyOptions {
	if opts == nil {
		return CopyOptions{Exclude: defaultExcludes, Reflink: true}
	}
	co := *opts
	if co.Exclude == nil {
		co.Exclude = defaultExcludes
	}
	return co
}

func ensureDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%q is not a directory", path)
	}
	return nil
}
