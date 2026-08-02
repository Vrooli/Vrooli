//go:build darwin

package hostpaths

import (
	"os"
	"path/filepath"
)

// trashRoots resolves the macOS user trash.
//
// macOS does not follow the freedesktop layout: the user trash is a flat
// ~/.Trash with no separate metadata directory, so there is no info/ pairing to
// preserve the way there is on linux.
//
// Per-volume trashes (/Volumes/<name>/.Trashes/$uid) are deliberately not
// resolved, matching the linux decision to stay within the home trash.
func trashRoots() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{filepath.Join(home, ".Trash")}
}
