//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package capacity

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func diskFilesystem(dir string) (string, string, int64, error) {
	if dir == "" {
		return "", "", 0, fmt.Errorf("disk reservation path is required")
	}
	canonical, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", "", 0, fmt.Errorf("resolve disk reservation path: %w", err)
	}
	info, err := os.Stat(canonical)
	if err != nil {
		return "", "", 0, fmt.Errorf("stat disk reservation path: %w", err)
	}
	if !info.IsDir() {
		return "", "", 0, fmt.Errorf("disk reservation path must be a directory")
	}
	var st unix.Stat_t
	if err := unix.Stat(canonical, &st); err != nil {
		return "", "", 0, fmt.Errorf("identify disk reservation filesystem: %w", err)
	}
	var fs unix.Statfs_t
	if err := unix.Statfs(canonical, &fs); err != nil {
		return "", "", 0, fmt.Errorf("measure disk reservation filesystem: %w", err)
	}
	if fs.Bsize <= 0 || fs.Bavail > uint64(math.MaxInt64)/uint64(fs.Bsize) {
		return "", "", 0, fmt.Errorf("filesystem reports invalid available capacity")
	}
	return canonical, fmt.Sprintf("dev:%d", st.Dev), int64(fs.Bavail) * int64(fs.Bsize), nil
}
