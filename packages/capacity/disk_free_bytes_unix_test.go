//go:build !windows

package capacity

import (
	"testing"

	"golang.org/x/sys/unix"
)

func freeDiskBytesForTest(t *testing.T, path string) int64 {
	t.Helper()
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		t.Fatal(err)
	}
	return int64(stat.Bavail) * int64(stat.Bsize)
}
