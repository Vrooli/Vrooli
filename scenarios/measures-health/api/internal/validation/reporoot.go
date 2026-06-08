package validation

import (
	"os"
	"path/filepath"
	"strings"
)

// ResolveRepoRoot finds the Vrooli monorepo root: the nearest ancestor of the
// process working directory that contains both a scenarios/ and a packages/
// directory. MEASURES_HEALTH_REPO_ROOT (or VROOLI_REPO_ROOT) overrides the
// search. It degrades to "." when nothing matches so the validator constructs;
// reads then fail per-call rather than crashing boot.
func ResolveRepoRoot() string {
	for _, env := range []string{"MEASURES_HEALTH_REPO_ROOT", "VROOLI_REPO_ROOT"} {
		if v := strings.TrimSpace(os.Getenv(env)); v != "" {
			return v
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	dir := cwd
	for {
		if isRepoRoot(dir) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return cwd
		}
		dir = parent
	}
}

func isRepoRoot(dir string) bool {
	for _, marker := range []string{"scenarios", "packages"} {
		if fi, err := os.Stat(filepath.Join(dir, marker)); err != nil || !fi.IsDir() {
			return false
		}
	}
	return true
}
