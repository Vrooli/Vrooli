//go:build aix || android || darwin || dragonfly || freebsd || hurd || illumos || linux || netbsd || openbsd || solaris

package census

import (
	"fmt"
	"path/filepath"
)

// deviceRootForPath returns the mount point containing path. Walking parents
// by st_dev avoids assuming that a user's home lives on /, which is common on
// managed hosts and desktop installations.
func deviceRootForPath(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	meta, err := inspectPath(path)
	if err != nil {
		return "", fmt.Errorf("inspect runtime path %s: %w", path, err)
	}
	if !meta.identity.valid {
		return filepath.VolumeName(path) + string(filepath.Separator), nil
	}
	current := filepath.Clean(path)
	for {
		parent := filepath.Dir(current)
		if parent == current {
			return current, nil
		}
		parentMeta, statErr := inspectPath(parent)
		if statErr != nil || !parentMeta.identity.valid || parentMeta.device != meta.device {
			return current, nil
		}
		current = parent
	}
}
