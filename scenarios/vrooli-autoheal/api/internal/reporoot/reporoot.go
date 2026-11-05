package reporoot

import (
	"os"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

// Resolve returns the canonical repo root when it can be derived, while still
// honoring explicit overrides even when repo-contract discovery is unavailable.
func Resolve(getenv func(string) string) string {
	// Prefer the explicit runtime override over inherited source-root context.
	for _, key := range []string{"VROOLI_ROOT", "VROOLI_SOURCE_ROOT"} {
		if root := strings.TrimSpace(getenv(key)); root != "" {
			if resolved, ok := CanonicalizeOverride(root); ok {
				return resolved
			}
			return filepath.Clean(root)
		}
	}

	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		return ""
	}
	return root
}

// CanonicalizeOverride attempts to map any descendant path back to the repo
// root using repo-contract discovery.
func CanonicalizeOverride(path string) (string, bool) {
	current := filepath.Clean(strings.TrimSpace(path))
	if current == "" || current == "." {
		return "", false
	}
	for depth := 0; depth < 25; depth++ {
		if resolved, err := repocontract.FindRepoRootFromPath(current); err == nil {
			return resolved, true
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return "", false
}

func ResolveFromOS() string {
	return Resolve(os.Getenv)
}
