//go:build !(aix || android || darwin || dragonfly || freebsd || hurd || illumos || linux || netbsd || openbsd || solaris)

package census

import (
	"fmt"
	"path/filepath"
)

func deviceRootForPath(path string) (string, error) {
	return deviceRootForPathWith(hostFileSystem{}, path)
}

// DeviceRoot returns the canonical accounting root for a scan request. Device
// scoped reports use this same root as their durable snapshot key, so status,
// history, scheduled scans, and forced scans all address one observation.
func DeviceRoot(path string) (string, error) {
	return deviceRootForPath(path)
}

func deviceRootForPathWith(filesystem FileSystem, path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if _, err := filesystem.Stat(path); err != nil {
		return "", fmt.Errorf("inspect runtime path %s: %w", path, err)
	}
	volume := filepath.VolumeName(path)
	if volume != "" {
		return volume + string(filepath.Separator), nil
	}
	return string(filepath.Separator), nil
}
