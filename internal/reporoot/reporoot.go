// Package reporoot resolves the canonical repository root for host-facing
// control-plane operations.
package reporoot

import (
	"os"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

// ResolveFromOS honors the explicit runtime override and otherwise delegates
// repository-root discovery to the shared repo-contract authority.
func ResolveFromOS() string {
	for _, key := range []string{"VROOLI_ROOT", "VROOLI_SOURCE_ROOT"} {
		if root := strings.TrimSpace(os.Getenv(key)); root != "" {
			if resolved, ok := canonicalize(root); ok {
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

func canonicalize(path string) (string, bool) {
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
