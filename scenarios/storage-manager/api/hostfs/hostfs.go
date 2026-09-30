// Package hostfs is the production implementation of the cleanup.FileSystem
// seam: the one place in storage-manager that reads and removes real files.
//
// It lives outside internal/ because internal/cleanup/no_real_cleanup_test.go
// forbids os.Remove/os.RemoveAll/exec.Command anywhere beneath internal/. That
// guard encodes a real architectural rule — the planning and policy core stays
// pure and testable against fakes, and every genuine side effect is confined to
// a thin, auditable adapter at the edge. This is that adapter.
//
// Before this package existed the seam had no production implementation at all;
// the only assignment to BuiltInDeps.FileSystem in the scenario was a fake in a
// test. Every file provider therefore reported "filesystem seam unavailable"
// forever, which is why 70 GB of temp files accumulated while storage-manager
// reported itself healthy.
package hostfs

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	coreRetention "github.com/vrooli/api-core/retention"
	"storage-manager/internal/cleanup"
)

// FS reads and removes host filesystem entries on behalf of the file providers.
type FS struct {
	restrictToOwner bool
}

// Options configures an FS.
type Options struct {
	// AllowForeignOwnership disables the current-user ownership restriction.
	//
	// The default (false, i.e. restricted) is the safe one and should almost
	// never be changed. On a shared /tmp the directory is world-writable with
	// the sticky bit set, so it holds entries belonging to other users and to
	// system services; on the host that motivated this package, 39 of the
	// top-level entries were root-owned service sockets and private mounts.
	// Attempting to remove those is both futile and dangerous, so they are
	// filtered out before they can ever reach a preview.
	AllowForeignOwnership bool
}

// New builds a production filesystem seam.
func New(opts Options) *FS {
	return &FS{restrictToOwner: !opts.AllowForeignOwnership}
}

// Compile-time guarantee that FS satisfies the seam.
var _ cleanup.FileSystem = (*FS)(nil)

// Stat reports a single entry.
func (f *FS) Stat(ctx context.Context, path string) (cleanup.FileInfo, error) {
	if err := ctx.Err(); err != nil {
		return cleanup.FileInfo{}, err
	}
	// Lstat, not Stat: a symlink must be reported as the link itself. Reporting
	// the target's size would attribute bytes to a path whose removal frees
	// only the link, and would let a link into a foreign tree masquerade as an
	// entry inside a configured root.
	info, err := os.Lstat(path)
	if err != nil {
		return cleanup.FileInfo{}, err
	}
	return toFileInfo(filepath.Clean(path), info), nil
}

// ReadFile reads provider metadata through the filesystem seam.
func (f *FS) ReadFile(ctx context.Context, path string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

// ReadDir lists the immediate children of a directory.
//
// Like Walk it is forgiving about individual entries: one child that vanished
// between the directory listing and its stat does not invalidate the rest of
// the listing, which in a temp directory is a routine race rather than a fault.
// Symlinks are excluded for the same reason Walk excludes them — a link is
// never a unit of reclaimable space, and following one would let a path inside
// a configured root stand in for a tree outside it.
func (f *FS) ReadDir(ctx context.Context, path string) ([]cleanup.FileInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	path = filepath.Clean(path)

	entries, err := os.ReadDir(path)
	if err != nil {
		if isBenignWalkError(err) {
			return nil, nil
		}
		return nil, err
	}

	out := make([]cleanup.FileInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.Type()&fs.ModeSymlink != 0 {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			if isBenignWalkError(err) {
				continue
			}
			return nil, err
		}
		if f.restrictToOwner && !ownedByCurrentUser(info) {
			continue
		}
		out = append(out, toFileInfo(filepath.Join(path, entry.Name()), info))
	}
	return out, nil
}

// Walk visits every entry beneath root.
//
// Walk is deliberately forgiving. It runs against directories it does not own
// and cannot fully read — /tmp on the incident host held 24,882 top-level
// entries across many users — so an unreadable subtree is an ordinary
// condition, not a failure. Permission errors skip the offending entry and the
// walk continues; anything else is returned, because a caller that silently
// swallowed, say, an I/O error would under-report reclaimable space and hide a
// failing disk.
func (f *FS) Walk(ctx context.Context, root string, visit func(cleanup.FileInfo) error) error {
	root = filepath.Clean(root)

	return coreRetention.WalkDirectory(ctx, root, func(path string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			if isBenignWalkError(walkErr) {
				// An unreadable directory is skipped whole; an unreadable file
				// is simply passed over.
				if entry != nil && entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			return walkErr
		}

		// Never descend through a symlink. WalkDir already refuses to follow
		// them, but reporting one as a candidate would let a link inside a
		// configured root point anywhere on the host, and removing the link is
		// never what a byte-reclamation pass intends.
		if entry.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			// The entry vanished between being listed and being stat'd. In a
			// temp directory that is completely routine, not an error.
			if isBenignWalkError(err) {
				return nil
			}
			return err
		}

		if path != root && f.restrictToOwner && !ownedByCurrentUser(info) {
			// Another user's entry. Skipping the whole subtree rather than the
			// single entry avoids walking into trees we could never remove.
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		return visit(toFileInfo(path, info))
	})
}

// RemoveAll deletes an entry and everything beneath it.
//
// The ownership re-check is not redundant with the one in Walk. Preview and
// apply are separate operations separated by operator approval time, so the
// filesystem can change in between; re-checking at the moment of deletion means
// the decision is made against what is on disk now, not what was on disk when
// the plan was built.
func (f *FS) RemoveAll(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path = filepath.Clean(path)

	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// Already gone. Removal is idempotent by nature and a concurrent
			// cleanup reaching the same entry first is a success, not a fault.
			return nil
		}
		return err
	}
	if f.restrictToOwner && !ownedByCurrentUser(info) {
		return fmt.Errorf("refusing to remove %s: owned by another user", path)
	}

	return coreRetention.RemovePath(ctx, path)
}

func (f *FS) MkdirAll(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return os.MkdirAll(filepath.Clean(path), 0o700)
}

func (f *FS) Rename(ctx context.Context, oldPath, newPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	oldPath, newPath = filepath.Clean(oldPath), filepath.Clean(newPath)
	info, err := os.Lstat(oldPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	if f.restrictToOwner && !ownedByCurrentUser(info) {
		return fmt.Errorf("refusing to rename %s: owned by another user", oldPath)
	}
	return os.Rename(oldPath, newPath)
}

// toFileInfo converts a stdlib FileInfo into the seam's transport type.
func toFileInfo(path string, info fs.FileInfo) cleanup.FileInfo {
	return cleanup.FileInfo{
		Path:    path,
		Size:    info.Size(),
		Mode:    info.Mode(),
		ModTime: info.ModTime(),
		IsDir:   info.IsDir(),
	}
}

// isBenignWalkError reports whether an error encountered mid-walk should be
// skipped rather than aborting.
//
// Permission and non-existence are both expected while traversing a shared
// temp directory: entries appear and vanish constantly, and other users' trees
// are unreadable by design. ErrNotExist additionally covers a dangling symlink
// or an entry removed between listing and stat.
func isBenignWalkError(err error) bool {
	return errors.Is(err, fs.ErrPermission) || errors.Is(err, fs.ErrNotExist)
}

// WithinRoot reports whether path is root or lies beneath it.
//
// Exported so callers composing roots can apply the same containment rule the
// providers use, rather than re-deriving prefix logic that is easy to get
// subtly wrong (a bare strings.HasPrefix would treat /tmpfoo as inside /tmp).
func WithinRoot(root, path string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	return path == root || strings.HasPrefix(path, root+string(filepath.Separator))
}
