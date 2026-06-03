package sources

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// filesystemCapturer captures an arbitrary filesystem locator — a directory
// tree OR a single file — by recursively copying it into the stage directory.
// No external resource CLI is needed; this is pure file I/O.
type filesystemCapturer struct{}

// Compile-time guarantee.
var _ Capturer = (*filesystemCapturer)(nil)

func newFilesystemCapturer() *filesystemCapturer { return &filesystemCapturer{} }

func (c *filesystemCapturer) Kind() SourceKind { return KindFilesystem }

// Capture produces a snapshottable artifact for a filesystem locator. It is
// split by locator shape to avoid the redundant staging copy that dominated
// run I/O on large trees:
//
//   - Directory locator → snapshot IN PLACE. kopia reads the live tree
//     directly, so Artifact.Path IS the source path and nothing is copied to
//     the stage dir. kopia stores symlinks as symlinks and never follows them
//     (verified empirically), so an in-place snapshot of ~/.vrooli/state does
//     NOT dereference its codex session links to ~/.codex/auth.json — a
//     deliberately-excluded credential. On restore, kopia restores the
//     snapshot-root's children into the scratch dir, the exact shape a staged
//     copy produced, so the restore path is unchanged.
//   - Single-file / single-symlink locator → STAGE under <StageDir>/fs/<basename>.
//     A bare single-file kopia snapshot does not restore cleanly into a
//     directory, so wrapping it in a directory preserves the original filename
//     across the capture→snapshot→restore round trip. These locators are tiny
//     (config.toml, history.jsonl, secrets.json), so the staging copy is cheap.
//
// In both cases symlinks are preserved, never followed: copyTree (staging) and
// kopia (in-place) each treat a symlink as a link, so the excluded-credential
// guarantee holds on both paths.
func (c *filesystemCapturer) Capture(_ context.Context, spec CaptureSpec) (Artifact, error) {
	info, err := os.Lstat(spec.Locator)
	if err != nil {
		return Artifact{}, fmt.Errorf("filesystem capture: stat %q: %w", spec.Locator, err)
	}
	if info.IsDir() {
		total, sizeErr := treeSize(spec.Locator)
		if sizeErr != nil {
			return Artifact{}, fmt.Errorf("filesystem capture: size %q: %w", spec.Locator, sizeErr)
		}
		return Artifact{Path: spec.Locator, Bytes: total}, nil
	}

	dst := filepath.Join(spec.StageDir, "fs")
	if err := os.MkdirAll(dst, 0o750); err != nil {
		return Artifact{}, fmt.Errorf("filesystem capture: mkdir stage: %w", err)
	}
	total, err := copyTree(spec.Locator, dst)
	if err != nil {
		return Artifact{}, fmt.Errorf("filesystem capture: stage %q: %w", spec.Locator, err)
	}
	return Artifact{Path: dst, Bytes: total}, nil
}

// treeSize sums the logical size of the regular files under root without
// following symlinks (WalkDir does not follow them; a symlink is not a regular
// file, so it is never traversed into). Used as the in-place capture's
// Artifact.Bytes for the storage-cap check — a conservative logical total, not
// the deduped on-disk size.
func treeSize(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			info, infoErr := d.Info()
			if infoErr != nil {
				return infoErr
			}
			total += info.Size()
		}
		return nil
	})
	return total, err
}

// Restore recursively copies the staged artifact at spec.ArtifactPath into
// spec.Target, recreating the original tree (files, directories, and symlinks).
func (c *filesystemCapturer) Restore(_ context.Context, spec RestoreSpec) error {
	if err := os.MkdirAll(spec.Target, 0o750); err != nil {
		return fmt.Errorf("filesystem restore: mkdir target: %w", err)
	}
	if _, err := copyTree(spec.ArtifactPath, spec.Target); err != nil {
		return fmt.Errorf("filesystem restore: walk %q: %w", spec.ArtifactPath, err)
	}
	return nil
}

// copyTree recursively copies src into dst, preserving directories, regular
// files, and symlinks (as symlinks). When src is a single file or symlink, it is
// placed under dst as dst/<basename>. Returns total bytes of copied regular-file
// content.
func copyTree(src, dst string) (int64, error) {
	var total int64
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(src, path)
		if relErr != nil {
			return relErr
		}
		// A single-file or single-symlink locator: WalkDir visits exactly the
		// locator itself with rel ".". Stage it under its basename so the
		// artifact stays a directory tree (dst/<name>) rather than clobbering
		// dst, which is already a directory.
		if rel == "." && !d.IsDir() {
			rel = filepath.Base(path)
		}
		target := filepath.Join(dst, rel)
		switch {
		case d.IsDir():
			return os.MkdirAll(target, 0o750)
		case d.Type()&fs.ModeSymlink != 0:
			return copySymlink(path, target)
		default:
			n, copyErr := copyFile(path, target)
			if copyErr != nil {
				return copyErr
			}
			total += n
			return nil
		}
	})
	return total, err
}

// copySymlink recreates the symlink at src as a symlink at dst, copying the link
// target verbatim. The target is never dereferenced.
func copySymlink(src, dst string) error {
	link, err := os.Readlink(src)
	if err != nil {
		return fmt.Errorf("copySymlink readlink %q: %w", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return fmt.Errorf("copySymlink mkdir %q: %w", filepath.Dir(dst), err)
	}
	// Make the create idempotent across re-runs / re-restores.
	if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("copySymlink clear %q: %w", dst, err)
	}
	if err := os.Symlink(link, dst); err != nil {
		return fmt.Errorf("copySymlink symlink %q→%q: %w", dst, link, err)
	}
	return nil
}

// copyFile copies src to dst, creating (or truncating) dst. Returns bytes written.
func copyFile(src, dst string) (int64, error) {
	in, err := os.Open(src)
	if err != nil {
		return 0, fmt.Errorf("copyFile open %q: %w", src, err)
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return 0, fmt.Errorf("copyFile mkdir %q: %w", filepath.Dir(dst), err)
	}
	out, err := os.Create(dst)
	if err != nil {
		return 0, fmt.Errorf("copyFile create %q: %w", dst, err)
	}
	defer out.Close()

	n, err := io.Copy(out, in)
	if err != nil {
		return 0, fmt.Errorf("copyFile copy %q→%q: %w", src, dst, err)
	}
	return n, nil
}
