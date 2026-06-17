//go:build unix

package models

import "golang.org/x/sys/unix"

// diskAvail reports the free bytes available to an unprivileged caller at path's
// filesystem (statfs Bavail × block size). It is the production DiskSpaceFunc.
func diskAvail(path string) (int64, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return 0, err
	}
	return int64(st.Bavail) * int64(st.Bsize), nil //nolint:gosec // filesystem sizes are non-negative
}
