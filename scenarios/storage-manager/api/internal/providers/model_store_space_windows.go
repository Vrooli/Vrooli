//go:build windows

package providers

import "errors"

// MeasureFilesystemSpace is unsupported on Windows; no model store runs there.
func MeasureFilesystemSpace(string) (FilesystemSpace, error) {
	return FilesystemSpace{}, errors.New("model store filesystem measurement is unsupported on windows")
}
