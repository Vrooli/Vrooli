//go:build unix

package retention

import (
	"fmt"
	"syscall"
)

// freeSpace reports the bytes available to an unprivileged process on the
// filesystem holding dir.
//
// Bavail is deliberate rather than Bfree: reserved blocks are not available to
// the process doing the compaction, and treating them as available is how a
// free-space guard passes and the write then fails part-way through.
func freeSpace(dir string) (int64, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(dir, &st); err != nil {
		return 0, fmt.Errorf("statfs %s: %w", dir, err)
	}
	return int64(st.Bavail) * int64(st.Bsize), nil
}
