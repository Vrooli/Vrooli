//go:build windows

package resources

import (
	"path/filepath"

	"golang.org/x/sys/windows"
)

func nativeOpenRGBAdoptionPaths(string) ([]string, error) {
	root, err := windows.KnownFolderPath(windows.FOLDERID_ProgramFiles, 0)
	if err != nil {
		return nil, err
	}
	return []string{filepath.Join(root, "OpenRGB", "OpenRGB.exe")}, nil
}
