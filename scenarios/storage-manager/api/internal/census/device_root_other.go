//go:build !(aix || android || darwin || dragonfly || freebsd || hurd || illumos || linux || netbsd || openbsd || solaris)

package census

import (
	"fmt"
	"os"
	"path/filepath"
)

func deviceRootForPath(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("inspect runtime path %s: %w", path, err)
	}
	volume := filepath.VolumeName(path)
	if volume != "" {
		return volume + string(filepath.Separator), nil
	}
	return string(filepath.Separator), nil
}
