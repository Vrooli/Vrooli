//go:build unix

package diskpolicy

import "golang.org/x/sys/unix"

// observe reports the filesystem's size and the bytes an unprivileged caller can
// still use.
func observe(path string) (total, free int64, err error) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return 0, 0, err
	}
	return int64(st.Blocks) * int64(st.Bsize), int64(st.Bavail) * int64(st.Bsize), nil //nolint:gosec // sizes are non-negative
}
