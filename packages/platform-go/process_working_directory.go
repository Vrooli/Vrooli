package platform

import (
	"errors"
	"path/filepath"
	"strings"
)

// ProcessWorkingDirectoryMatches compares underlying directory identities,
// rather than Windows casing or trailing-separator spelling. The caller must
// bind this observation to an expected process incarnation before and after.
func ProcessWorkingDirectoryMatches(pid int, expected string) (bool, error) {
	if pid <= 0 || !filepath.IsAbs(expected) || strings.ContainsRune(expected, 0) {
		return false, errors.New("platform: canonical expected process directory required")
	}
	return processWorkingDirectoryMatches(pid, expected)
}
