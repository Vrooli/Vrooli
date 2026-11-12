//go:build windows

package resources

import (
	"golang.org/x/sys/windows"
	"path/filepath"
)

func nativeOpenRGBMachineRoot() (string, error) {
	root, err := windows.KnownFolderPath(windows.FOLDERID_ProgramData, 0)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "Vrooli", "resources", "openrgb"), nil
}
