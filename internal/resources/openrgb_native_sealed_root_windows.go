//go:build windows

package resources

import (
	"golang.org/x/sys/windows"
	"path/filepath"
)

func nativeOpenRGBSealedRoot() (string, error) {
	root, err := windows.KnownFolderPath(windows.FOLDERID_ProgramFiles, 0)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "Vrooli", "OpenRGB"), nil
}
