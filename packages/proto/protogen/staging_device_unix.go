//go:build !windows

package protogen

import (
	"fmt"
	"os"
	"syscall"
)

// sameFilesystem reports whether a and b are on one filesystem, which is
// exactly when a rename between them can succeed.
func sameFilesystem(a, b string) (bool, error) {
	left, err := deviceOf(a)
	if err != nil {
		return false, err
	}
	right, err := deviceOf(b)
	if err != nil {
		return false, err
	}
	return left == right, nil
}

func deviceOf(path string) (uint64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("no device information for %s", path)
	}
	return uint64(stat.Dev), nil //nolint:unconvert // Dev is int32 on darwin.
}
