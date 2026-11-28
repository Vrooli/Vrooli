//go:build !darwin && !windows

package resources

import "errors"

func nativeOpenRGBMachineRoot() (string, error) {
	return "", errors.New("native machine OpenRGB storage is defined only for Darwin and Windows")
}
