//go:build windows

package protogen

import (
	"path/filepath"
	"strings"
)

// sameFilesystem reports whether a and b are on one volume, which is exactly
// when a rename between them can succeed.
func sameFilesystem(a, b string) (bool, error) {
	left, err := filepath.Abs(a)
	if err != nil {
		return false, err
	}
	right, err := filepath.Abs(b)
	if err != nil {
		return false, err
	}
	return strings.EqualFold(filepath.VolumeName(left), filepath.VolumeName(right)), nil
}
