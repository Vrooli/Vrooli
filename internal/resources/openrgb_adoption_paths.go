package resources

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func resolveOpenRGBAdoptionPath(home, command string) (string, error) {
	if path, err := exec.LookPath(command); err == nil {
		return filepath.Abs(path)
	}
	candidates, err := nativeOpenRGBAdoptionPaths(home)
	if err != nil {
		return "", err
	}
	for _, path := range candidates {
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return "", err
		}
		if !info.Mode().IsRegular() {
			return "", errors.New("existing native OpenRGB path is not a regular file")
		}
		return path, nil
	}
	return "", errors.New("compatible native OpenRGB not found on PATH or in the standard application directory on " + runtime.GOOS)
}

func openRGBMacAdoptionPaths(home string) []string {
	return []string{"/Applications/OpenRGB.app/Contents/MacOS/OpenRGB", filepath.Join(home, "Applications", "OpenRGB.app", "Contents", "MacOS", "OpenRGB")}
}
