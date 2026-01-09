// Package staleness provides timestamp-based staleness detection for Go API binaries.
// When an API starts, it can check if its source files have changed since compilation
// and automatically rebuild/restart itself if stale.
package staleness

import (
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// skipDirs contains directory names that should be skipped during file walks.
var skipDirs = map[string]bool{
	".git":         true,
	".vscode":      true,
	".idea":        true,
	"vendor":       true,
	"node_modules": true,
	"dist":         true,
	"build":        true,
	"tmp":          true,
	"data":         true,
	"testdata":     true,
	"coverage":     true,
}

// CheckNewerFiles walks a directory and returns true if any file matching the pattern
// has a modification time after the reference time. Returns the path of the first
// newer file found, or empty string if none found.
func CheckNewerFiles(dir, pattern string, refTime time.Time) (found bool, newerFile string) {
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors, continue walking
		}

		// Skip directories in the skip list
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file matches pattern
		matched, _ := filepath.Match(pattern, d.Name())
		if !matched {
			return nil
		}

		// Get file info for modification time
		info, err := d.Info()
		if err != nil {
			return nil
		}

		// Check if file is newer than reference time
		if info.ModTime().After(refTime) {
			found = true
			newerFile = path
			return filepath.SkipAll // Stop walking, we found one
		}

		return nil
	})

	return found, newerFile
}

// IsFileNewer checks if a single file is newer than the reference time.
func IsFileNewer(path string, refTime time.Time) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.ModTime().After(refTime)
}

// GetModTime returns the modification time of a file, or zero time on error.
func GetModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}
