//go:build !darwin && !windows

package resources

import "errors"

func openRGBNativeOperatorPrincipal() (string, error) {
	return "", errors.New("native OpenRGB owner descriptor is defined only for Darwin and Windows")
}
