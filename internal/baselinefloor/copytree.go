package baselinefloor

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/vrooli/vrooli/internal/scenariostale"
)

// defaultExcludes is the restore-point copy ladder exclude set, aligned to the
// repo .gitignore. Every entry is a build artifact or VCS/runtime directory that
// is regenerated on restart (dist/generated/.build-fingerprint.json), recomputed
// by tooling (coverage), VCS-managed (.git), or operator runtime state that must
// NOT be snapshotted into a code restore point (.vrooli). Excluding them is the
// single biggest size/speed win (60–80% smaller on UI scenarios) and is safe
// because excluded == git-ignored == regenerable; excluding a stale dist is in
// fact MORE correct, since cli-core rebuilds from source on restart.
//
// Matched by base name at any depth, against both directories and files.
var defaultExcludes = map[string]struct{}{
	"node_modules":            {},
	"dist":                    {},
	"coverage":                {},
	".git":                    {},
	".vrooli":                 {},
	"generated":               {},
	scenariostale.SidecarFile: {}, // ".build-fingerprint.json"
	".turbo":                  {},
	".next":                   {},
	"__pycache__":             {},
}

// DefaultExcludes returns a fresh copy of the restore-point exclude set so a
// caller can extend it (e.g. add a scenario-specific artifact dir) without
// mutating the package default.
func DefaultExcludes() map[string]struct{} {
	out := make(map[string]struct{}, len(defaultExcludes))
	for k := range defaultExcludes {
		out[k] = struct{}{}
	}
	return out
}

// CopyOptions tunes a CopyTree run.
type CopyOptions struct {
	// Exclude names (directories or files, matched by base name at any depth)
	// skipped during the walk. Use DefaultExcludes() as a starting point. A nil
	// map copies everything.
	Exclude map[string]struct{}
	// Reflink enables the copy-on-write reflink fast path per regular file where
	// the OS + filesystem support it (Linux FICLONE; no-op elsewhere). On any
	// reflink failure the file falls back to a native-Go deep copy, so enabling
	// it is always safe. The deep copy is the portable floor.
	Reflink bool
}

// CopyStats reports what a CopyTree run did. ReflinkFiles + DeepCopyFiles ==
// regular files copied; the split shows how much the CoW fast path was used.
type CopyStats struct {
	Dirs          int
	Symlinks      int
	ReflinkFiles  int
	DeepCopyFiles int
	BytesCopied   int64
	Excluded      int
}

// CopyTree recursively copies src into dst.
//
// Contract (Baseline Modes §8 "Restore-point copy ladder"):
//   - Directories and file modes are preserved.
//   - Symlinks are recreated verbatim, never dereferenced (a symlink into an
//     excluded tree therefore stays a dangling-but-harmless link, not a copy of
//     the excluded content).
//   - Regular files use the reflink fast path when opts.Reflink is set and the
//     filesystem supports it, else a native-Go deep copy. NEVER a hardlink.
//   - Excluded base names are skipped (SkipDir for directories).
//   - dst is overlaid: existing files are overwritten, making re-copy idempotent.
//     Files present in dst but absent in src are left in place (callers that need
//     an exact mirror clean dst first).
func CopyTree(src, dst string, opts CopyOptions) (CopyStats, error) {
	var stats CopyStats

	info, err := os.Lstat(src)
	if err != nil {
		return stats, fmt.Errorf("copytree: stat src %q: %w", src, err)
	}
	if !info.IsDir() {
		return stats, fmt.Errorf("copytree: src %q is not a directory", src)
	}

	walkErr := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		base := d.Name()
		if path != src {
			if _, skip := opts.Exclude[base]; skip {
				stats.Excluded++
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		rel, relErr := filepath.Rel(src, path)
		if relErr != nil {
			return relErr
		}
		target := filepath.Join(dst, rel)

		switch {
		case d.IsDir():
			mode := fs.FileMode(0o750)
			if fi, statErr := d.Info(); statErr == nil {
				mode = fi.Mode().Perm()
			}
			if mkErr := os.MkdirAll(target, mode); mkErr != nil {
				return fmt.Errorf("copytree: mkdir %q: %w", target, mkErr)
			}
			stats.Dirs++
			return nil

		case d.Type()&fs.ModeSymlink != 0:
			if symErr := copySymlink(path, target); symErr != nil {
				return symErr
			}
			stats.Symlinks++
			return nil

		case d.Type().IsRegular():
			fi, statErr := d.Info()
			if statErr != nil {
				return fmt.Errorf("copytree: info %q: %w", path, statErr)
			}
			reflinked, n, copyErr := copyRegularFile(path, target, fi.Mode().Perm(), opts.Reflink)
			if copyErr != nil {
				return copyErr
			}
			if reflinked {
				stats.ReflinkFiles++
			} else {
				stats.DeepCopyFiles++
			}
			stats.BytesCopied += n
			return nil

		default:
			// Sockets, devices, named pipes: not meaningful in a source tree
			// restore point. Skip rather than fail the whole copy.
			return nil
		}
	})
	if walkErr != nil {
		return stats, fmt.Errorf("copytree %q→%q: %w", src, dst, walkErr)
	}
	return stats, nil
}

// copyRegularFile copies src→dst preserving mode. When reflink is requested it
// first attempts a copy-on-write clone; on any unsupported/failed clone it falls
// back to a native-Go deep copy. Returns whether the reflink path was used and
// the number of bytes written (0 for a successful reflink, which copies no bytes
// through userspace).
func copyRegularFile(src, dst string, mode fs.FileMode, reflink bool) (bool, int64, error) {
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return false, 0, fmt.Errorf("copytree: mkdir parent of %q: %w", dst, err)
	}
	if reflink {
		ok, err := cloneFile(dst, src, mode)
		if err != nil {
			return false, 0, fmt.Errorf("copytree: reflink %q→%q: %w", src, dst, err)
		}
		if ok {
			return true, 0, nil
		}
	}
	n, err := deepCopyFile(src, dst, mode)
	return false, n, err
}

// deepCopyFile is the portable copy floor: a streaming io.Copy that creates (or
// truncates) dst with the given mode. It never hardlinks.
func deepCopyFile(src, dst string, mode fs.FileMode) (int64, error) {
	in, err := os.Open(src)
	if err != nil {
		return 0, fmt.Errorf("copytree: open %q: %w", src, err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return 0, fmt.Errorf("copytree: create %q: %w", dst, err)
	}
	n, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return n, fmt.Errorf("copytree: copy %q→%q: %w", src, dst, copyErr)
	}
	if closeErr != nil {
		return n, fmt.Errorf("copytree: close %q: %w", dst, closeErr)
	}
	// Ensure the mode lands even if dst pre-existed with a different mode.
	if chmodErr := os.Chmod(dst, mode); chmodErr != nil {
		return n, fmt.Errorf("copytree: chmod %q: %w", dst, chmodErr)
	}
	return n, nil
}

// copySymlink recreates the symlink at src as a symlink at dst, copying the link
// target verbatim. The target is never dereferenced, and the create is
// idempotent across re-copies.
func copySymlink(src, dst string) error {
	link, err := os.Readlink(src)
	if err != nil {
		return fmt.Errorf("copytree: readlink %q: %w", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return fmt.Errorf("copytree: mkdir parent of %q: %w", dst, err)
	}
	if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("copytree: clear %q: %w", dst, err)
	}
	if err := os.Symlink(link, dst); err != nil {
		return fmt.Errorf("copytree: symlink %q→%q: %w", dst, link, err)
	}
	return nil
}
