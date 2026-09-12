//go:build !windows

package checks

import "syscall"

// Statfs returns real filesystem statistics using statfs(2).
func (r *RealFileSystemReader) Statfs(path string) (*StatfsResult, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return nil, err
	}
	return &StatfsResult{
		Blocks: uint64(stat.Blocks),
		Bfree:  uint64(stat.Bfree),
		Bavail: uint64(stat.Bavail),
		Files:  uint64(stat.Files),
		Ffree:  uint64(stat.Ffree),
		Bsize:  int64(stat.Bsize),
	}, nil
}
