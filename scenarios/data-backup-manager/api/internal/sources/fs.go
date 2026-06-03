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

// Capture copies the tree at spec.Locator into <spec.StageDir>/fs. The artifact
// is always a directory: a directory locator's children land directly under it;
// a single-file (or single-symlink) locator lands as fs/<basename>. The returned
// Artifact.Path points to that copy; Artifact.Bytes is the total bytes copied.
//
// Symlinks are preserved as symlinks, never followed. Following them would both
// break on links to directories (copy_file_range refuses a directory) and, worse,
// dereference links that point at deliberately-excluded files — e.g. the codex
// session trees under ~/.vrooli/state symlink to ~/.codex/auth.json, a sensitive
// credential coverage intentionally leaves unregistered.
func (c *filesystemCapturer) Capture(_ context.Context, spec CaptureSpec) (Artifact, error) {
	dst := filepath.Join(spec.StageDir, "fs")
	if err := os.MkdirAll(dst, 0o750); err != nil {
		return Artifact{}, fmt.Errorf("filesystem capture: mkdir stage: %w", err)
	}
	total, err := copyTree(spec.Locator, dst)
	if err != nil {
		return Artifact{}, fmt.Errorf("filesystem capture: walk %q: %w", spec.Locator, err)
	}
	return Artifact{Path: dst, Bytes: total}, nil
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
