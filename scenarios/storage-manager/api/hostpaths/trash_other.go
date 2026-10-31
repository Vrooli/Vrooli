//go:build !linux && !darwin && !windows

package hostpaths

import (
	"os"
	"path/filepath"
	"strings"
)

// trashRoots resolves the trash on the remaining unix-like platforms.
//
// The BSDs and illumos follow the freedesktop layout when a desktop environment
// is present and have no trash at all otherwise. The XDG resolution is
// therefore the right default, and hostpaths.existing filters the root out on
// the headless case where the directory does not exist. This file keeps the
// package building everywhere rather than restricting the scenario to the three
// first-class platforms.
func trashRoots() []string {
	base := ""
	if dataHome := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); dataHome != "" {
		base = filepath.Join(dataHome, "Trash")
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		base = filepath.Join(home, ".local", "share", "Trash")
	}
	// See trash_linux.go for why the subdirectories are the roots rather than
	// the Trash directory itself.
	return []string{
		filepath.Join(base, "files"),
		filepath.Join(base, "info"),
	}
}
