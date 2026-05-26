package sources

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// filesystemCapturer captures an arbitrary directory tree by recursively
// copying it into the stage directory. No external resource CLI is needed —
// this is pure file I/O.
type filesystemCapturer struct{}

// Compile-time guarantee.
var _ Capturer = (*filesystemCapturer)(nil)

func newFilesystemCapturer() *filesystemCapturer { return &filesystemCapturer{} }

func (c *filesystemCapturer) Kind() SourceKind { return KindFilesystem }

// Capture recursively copies the directory tree at spec.Locator into
// <spec.StageDir>/fs. The returned Artifact.Path points to that copy;
// Artifact.Bytes is the total on-disk size of all copied files.
func (c *filesystemCapturer) Capture(_ context.Context, spec CaptureSpec) (Artifact, error) {
	dst := filepath.Join(spec.StageDir, "fs")
	if err := os.MkdirAll(dst, 0o750); err != nil {
		return Artifact{}, fmt.Errorf("filesystem capture: mkdir stage: %w", err)
	}

	var total int64
	err := filepath.WalkDir(spec.Locator, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(spec.Locator, path)
		if relErr != nil {
			return relErr
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o750)
		}
		n, copyErr := copyFile(path, target)
		if copyErr != nil {
			return copyErr
		}
		total += n
		return nil
	})
	if err != nil {
		return Artifact{}, fmt.Errorf("filesystem capture: walk %q: %w", spec.Locator, err)
	}
	return Artifact{Path: dst, Bytes: total}, nil
}

// Restore recursively copies the artifact directory at spec.ArtifactPath into
// spec.Target, recreating the original tree byte-for-byte.
func (c *filesystemCapturer) Restore(_ context.Context, spec RestoreSpec) error {
	if err := os.MkdirAll(spec.Target, 0o750); err != nil {
		return fmt.Errorf("filesystem restore: mkdir target: %w", err)
	}
	err := filepath.WalkDir(spec.ArtifactPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(spec.ArtifactPath, path)
		if relErr != nil {
			return relErr
		}
		target := filepath.Join(spec.Target, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o750)
		}
		_, copyErr := copyFile(path, target)
		return copyErr
	})
	if err != nil {
		return fmt.Errorf("filesystem restore: walk %q: %w", spec.ArtifactPath, err)
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
