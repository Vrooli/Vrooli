// Package gitpaths owns the portable path rules shared by Git worktree capture
// and durable worktree-selection records.
package gitpaths

import (
	"fmt"
	"path"
	"strings"
	"unicode/utf8"
)

// ValidateRepositoryRelativePaths accepts canonical, unique, slash-separated
// Git paths that remain repository-relative across supported host platforms.
func ValidateRepositoryRelativePaths(paths []string) error {
	if len(paths) == 0 {
		return fmt.Errorf("at least one worktree path is required")
	}
	seen := make(map[string]struct{}, len(paths))
	for _, candidate := range paths {
		if !canonicalRepositoryPath(candidate) {
			return fmt.Errorf("worktree path %q is not a canonical repository-relative Git path", candidate)
		}
		if _, exists := seen[candidate]; exists {
			return fmt.Errorf("worktree path %q occurs more than once", candidate)
		}
		seen[candidate] = struct{}{}
	}
	return nil
}

func canonicalRepositoryPath(candidate string) bool {
	if candidate == "" || !utf8.ValidString(candidate) || strings.ContainsRune(candidate, '\x00') || strings.ContainsRune(candidate, '\\') {
		return false
	}
	if path.IsAbs(candidate) || candidate == "." || candidate == ".." || strings.HasPrefix(candidate, "../") || path.Clean(candidate) != candidate {
		return false
	}
	// Windows treats a leading drive designator as rooted or drive-relative
	// even when slash-based path cleaning considers it an ordinary segment.
	return !hasDrivePrefix(candidate)
}

func hasDrivePrefix(candidate string) bool {
	return len(candidate) >= 2 && ((candidate[0] >= 'a' && candidate[0] <= 'z') || (candidate[0] >= 'A' && candidate[0] <= 'Z')) && candidate[1] == ':'
}
