//go:build !windows

package hostwatchdog

import "syscall"

func freeSpace(path string) (uint64, float64, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, 0, err
	}
	if st.Bsize == 0 {
		return 0, 0, syscall.EINVAL
	}
	available := uint64(st.Bavail) * uint64(st.Bsize)
	total := uint64(st.Blocks) * uint64(st.Bsize)
	used := total - uint64(st.Bfree)*uint64(st.Bsize)
	var percent float64
	if total > 0 {
		percent = float64(used) * 100 / float64(total)
	}
	return available, percent, nil
}
