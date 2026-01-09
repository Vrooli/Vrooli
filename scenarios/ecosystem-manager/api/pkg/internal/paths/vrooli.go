package paths

import (
	"os"
	"path/filepath"
)

// DetectVrooliRoot finds the root of the Vrooli workspace by searching for the .vrooli directory.
// It first checks the VROOLI_ROOT environment variable, then walks up from the current directory.
// Returns "." if no root is found.
func DetectVrooliRoot() string {
	if root := os.Getenv("VROOLI_ROOT"); root != "" {
		return root
	}

	if wd, err := os.Getwd(); err == nil {
		dir := wd
		for dir != string(filepath.Separator) && dir != "." {
			if _, statErr := os.Stat(filepath.Join(dir, ".vrooli")); statErr == nil {
				return dir
			}
			dir = filepath.Dir(dir)
		}
	}

	return "."
}
