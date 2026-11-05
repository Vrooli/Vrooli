//go:build linux

package baselinefloor

import (
	"errors"
	"io/fs"
	"os"

	"golang.org/x/sys/unix"
)

// cloneFile attempts a copy-on-write reflink of src→dst using the Linux FICLONE
// ioctl (supported on btrfs, xfs-reflink, zfs, ...). It is instant and edit-safe:
// the clone shares extents until one side is written, so a later in-place edit of
// the working tree does NOT mutate the restore point.
//
// Returns (true, nil) when the clone succeeded. When the filesystem does not
// support reflinks — the common case on ext4, which this host uses — it returns
// (false, nil) so the caller transparently falls back to a native-Go deep copy;
// the empty dst it created is removed first. A genuine error (read/permission)
// is returned as (false, err).
func cloneFile(dst, src string, mode fs.FileMode) (bool, error) {
	srcF, err := os.Open(src)
	if err != nil {
		return false, err
	}
	defer srcF.Close()

	dstF, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return false, err
	}

	cloneErr := unix.IoctlFileClone(int(dstF.Fd()), int(srcF.Fd()))
	closeErr := dstF.Close()
	if cloneErr != nil {
		// Remove the empty placeholder so the deep-copy fallback starts clean.
		_ = os.Remove(dst)
		if isUnsupportedClone(cloneErr) {
			return false, nil
		}
		return false, cloneErr
	}
	if closeErr != nil {
		_ = os.Remove(dst)
		return false, closeErr
	}
	if chmodErr := os.Chmod(dst, mode); chmodErr != nil {
		return false, chmodErr
	}
	return true, nil
}

// isUnsupportedClone reports whether a FICLONE error means "this filesystem /
// cross-device pair cannot reflink" (fall back) rather than a real failure.
func isUnsupportedClone(err error) bool {
	return errors.Is(err, unix.ENOTSUP) ||
		errors.Is(err, unix.EOPNOTSUPP) ||
		errors.Is(err, unix.EXDEV) || // src and dst on different filesystems
		errors.Is(err, unix.EINVAL) || // e.g. not a regular file / unaligned
		errors.Is(err, unix.ENOTTY) // ioctl not implemented by the fs driver
}
