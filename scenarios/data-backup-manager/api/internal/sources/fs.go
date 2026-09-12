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
func (c *filesystemCapturer) Restore(ctx context.Context, spec RestoreSpec) error {
	source, err := filepath.EvalSymlinks(spec.ArtifactPath)
	if err != nil {
		return err
	}
	if within(source, spec.Target) || within(spec.Target, source) {
		return fmt.Errorf("restore source and target must be disjoint")
	}
	if err := requireEmptyDestination(spec.Target); err != nil {
		return err
	}
	if err := os.MkdirAll(spec.Target, 0o750); err != nil {
		return fmt.Errorf("filesystem restore: mkdir target: %w", err)
	}
	if _, err := copyTreeContext(ctx, spec.ArtifactPath, spec.Target); err != nil {
		return fmt.Errorf("filesystem restore: walk %q: %w", spec.ArtifactPath, err)
	}
	return nil
}

// copyTree recursively copies src into dst, preserving directories, regular
// files, and symlinks (as symlinks). When src is a single file or symlink, it is
// placed under dst as dst/<basename>. Returns total bytes of copied regular-file
// content.
func copyTree(src, dst string) (int64, error) { return copyTreeContext(context.Background(), src, dst) }

func copyTreeContext(ctx context.Context, src, dst string) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	info, err := os.Lstat(src)
	if err != nil {
		return 0, err
	}
	sourceRoot, start := src, "."
	if !info.IsDir() {
		sourceRoot, start = filepath.Dir(src), filepath.Base(src)
	}
	in, err := os.OpenRoot(sourceRoot)
	if err != nil {
		return 0, err
	}
	defer in.Close()
	if err = os.MkdirAll(dst, 0700); err != nil {
		return 0, err
	}
	out, err := os.OpenRoot(dst)
	if err != nil {
		return 0, err
	}
	defer out.Close()
	var total int64
	type dirMeta struct {
		path string
		info os.FileInfo
	}
	var dirs []dirMeta
	err = fs.WalkDir(in.FS(), start, func(path string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if e = ctx.Err(); e != nil {
			return e
		}
		meta, e := in.Lstat(path)
		if e != nil {
			return e
		}
		switch {
		case meta.IsDir():
			if path != "." {
				if e = out.Mkdir(path, 0700); e != nil {
					return e
				}
			}
			dirs = append(dirs, dirMeta{path, meta})
			return nil
		case meta.Mode()&os.ModeSymlink != 0:
			link, e := in.Readlink(path)
			if e != nil {
				return e
			}
			return out.Symlink(link, path)
		case meta.Mode().IsRegular():
			f, e := in.Open(path)
			if e != nil {
				return e
			}
			opened, e := f.Stat()
			if e != nil {
				f.Close()
				return e
			}
			if !os.SameFile(meta, opened) {
				f.Close()
				return fmt.Errorf("source changed during copy")
			}
			dest, e := out.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if e != nil {
				f.Close()
				return e
			}
			n, e := io.Copy(dest, &contextReader{ctx: ctx, r: f})
			f.Close()
			total += n
			if e == nil {
				e = dest.Chmod(meta.Mode().Perm())
			}
			if e == nil {
				e = dest.Sync()
			}
			ce := dest.Close()
			if e != nil {
				return e
			}
			if ce != nil {
				return ce
			}
			return out.Chtimes(path, meta.ModTime(), meta.ModTime())
		default:
			return fmt.Errorf("unsupported filesystem entry %q", path)
		}
	})
	if err != nil {
		return total, err
	}
	for i := len(dirs) - 1; i >= 0; i-- {
		d := dirs[i]
		f, e := out.Open(d.path)
		if e != nil {
			return total, e
		}
		e = f.Chmod(d.info.Mode().Perm())
		f.Close()
		if e != nil {
			return total, e
		}
		if e = out.Chtimes(d.path, d.info.ModTime(), d.info.ModTime()); e != nil {
			return total, e
		}
	}
	return total, nil
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
	info, err := in.Stat()
	if err != nil {
		return 0, err
	}
	if !info.Mode().IsRegular() {
		return 0, fmt.Errorf("source is not a regular file: %q", src)
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return 0, fmt.Errorf("copyFile create %q: %w", dst, err)
	}
	n, err := io.Copy(out, in)
	if err == nil {
		err = out.Chmod(info.Mode().Perm())
	}
	if err == nil {
		err = out.Sync()
	}
	closeErr := out.Close()
	if err != nil {
		return n, err
	}
	if closeErr != nil {
		return n, closeErr
	}
	if err = os.Chtimes(dst, info.ModTime(), info.ModTime()); err != nil {
		return n, err
	}
	return n, nil
}

// Recheck at the effect boundary, including parent symlinks. Admission-time
// inspection is insufficient for queued jobs. Existing data is never removed.
func requireEmptyDestination(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("restore destination must be absolute")
	}
	for p := filepath.Clean(path); ; p = filepath.Dir(p) {
		info, err := os.Lstat(p)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if err == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
			return fmt.Errorf("restore destination contains a non-directory or symlink: %q", p)
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	entries, err := os.ReadDir(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return fmt.Errorf("restore destination is not empty")
	}
	return nil
}
