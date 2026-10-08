//go:build !windows

package collectors

import "golang.org/x/sys/unix"

// statfsBytes reports filesystem capacity for the mount containing path.
//
// free and available are deliberately distinct: free counts every unallocated
// block including the superuser reserve, while available counts only what an
// unprivileged writer may consume. Callers must derive pressure from
// available — see diskUsedPercent.
func statfsBytes(path string) (total, free, available int64, err error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, 0, 0, err
	}
	blockSize := int64(stat.Bsize)
	if blockSize <= 0 {
		return 0, 0, 0, nil
	}
	return int64(stat.Blocks) * blockSize, int64(stat.Bfree) * blockSize, int64(stat.Bavail) * blockSize, nil
}
