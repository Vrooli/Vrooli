//go:build !windows

package health

import "syscall"

// diskFreeBytes returns the bytes available to a non-privileged user on the
// filesystem containing path, via statfs. Works on linux and darwin without
// cgo; the field types differ across those platforms (Bsize is int64 on linux,
// uint32 on darwin), so both are widened through uint64 explicitly. The caller
// (Sample) clamps the unsigned result into the proto's int64.
func diskFreeBytes(path string) (uint64, error) {
	free, _, err := diskUsage(path)
	return free, err
}

// diskUsage returns the bytes available to a non-privileged user and the
// volume's total size.
func diskUsage(path string) (free, total uint64, err error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, 0, err
	}
	bsize := uint64(st.Bsize) // #nosec G115 -- statfs exposes non-negative filesystem counts; widening preserves the byte calculation.
	return uint64(st.Bavail) * bsize, uint64(st.Blocks) * bsize, nil
}
