//go:build !windows

package platform

import "os"

func nativeCurrentProcessElevated() (bool, error) {
	return os.Geteuid() == 0, nil
}
