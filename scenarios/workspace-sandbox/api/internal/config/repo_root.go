package config

import (
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

// ResolveDefaultProjectRoot returns the canonical repo root used when callers
// do not explicitly provide a project root.
func ResolveDefaultProjectRoot() string {
	if projectRoot := envString("PROJECT_ROOT", ""); projectRoot != "" {
		return projectRoot
	}

	for _, key := range []string{"VROOLI_SOURCE_ROOT", "VROOLI_ROOT"} {
		if value := strings.TrimSpace(envString(key, "")); value != "" {
			if root, err := repocontract.FindRepoRoot(value); err == nil {
				return root
			}
		}
	}

	root, err := repocontract.FindRepoRootFromCWD()
	if err == nil {
		return root
	}
	return ""
}
