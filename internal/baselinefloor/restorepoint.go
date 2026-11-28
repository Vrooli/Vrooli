package baselinefloor

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vrooli/vrooli/internal/tuning"
)

// Capture takes a restore point: it copies the scenario working tree (srcDir)
// into restorePointDest using the copy ladder. A restore point is a code-level
// undo, so it holds source and configuration only: build artifacts, build
// outputs and runtime data (databases, logs, caches, .vrooli runtime state)
// are excluded. Data rollback belongs to data-backup-manager, not here. This
// is the git-free undo the engagement records; `baseline abandon` and promote
// auto-rollback Restore from it, and an engaged serving instance builds from
// a copy of it.
//
// Capture is idempotent: an existing restore point is overlaid (re-running a
// capture refreshes it). The reflink fast path is requested — it is instant and
// edit-safe where supported, and transparently falls back to a deep copy on
// ext4/Windows. Callers pass opts.Ignore (an IgnoreMatcher over the
// repository's .gitignore files plus declared outputs) to honor the repository
// rules; nil opts use DefaultExcludes + the floor patterns + reflink.
func Capture(srcDir, restorePointDest string, opts *CopyOptions) (CopyStats, error) {
	if err := ensureDir(srcDir); err != nil {
		return CopyStats{}, fmt.Errorf("baselinefloor: capture src: %w", err)
	}
	co := resolveCopyOptions(opts)
	if err := os.MkdirAll(filepath.Dir(restorePointDest), tuning.PermGroupDir); err != nil {
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
// Callers pass opts.Ignore built for the destination so that a restore point
// captured before runtime data was excluded never overlays a stale database
// or log onto the working tree.
func Restore(restorePointSrc, destDir string, opts *CopyOptions) (CopyStats, error) {
	if err := ensureDir(restorePointSrc); err != nil {
		return CopyStats{}, fmt.Errorf("baselinefloor: restore point: %w", err)
	}
	co := resolveCopyOptions(opts)
	if err := os.MkdirAll(destDir, tuning.PermGroupDir); err != nil {
		return CopyStats{}, fmt.Errorf("baselinefloor: restore mkdir %q: %w", destDir, err)
	}
	stats, err := CopyTree(restorePointSrc, destDir, co)
	if err != nil {
		return stats, fmt.Errorf("baselinefloor: restore %q→%q: %w", restorePointSrc, destDir, err)
	}
	return stats, nil
}

// floorIgnorer applies the floor's always-on exclusion patterns.
var floorIgnorer = NewIgnoreMatcher(IgnoreSpec{Patterns: floorIgnorePatterns})

// resolveCopyOptions applies the restore-point defaults (the base-name exclude
// set, the floor patterns, and the reflink fast path) when the caller passes
// nil, and fills a nil Exclude or Ignore on an otherwise-supplied value.
func resolveCopyOptions(opts *CopyOptions) CopyOptions {
	if opts == nil {
		return CopyOptions{Exclude: defaultExcludes, Ignore: floorIgnorer, Reflink: true}
	}
	co := *opts
	if co.Exclude == nil {
		co.Exclude = defaultExcludes
	}
	if co.Ignore == nil {
		co.Ignore = floorIgnorer
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
