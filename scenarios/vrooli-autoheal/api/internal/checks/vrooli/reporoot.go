package vrooli

import (
	"os"
	"path/filepath"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/reporoot"
)

// resolveVrooliRoot locates the Vrooli repo root for checks that shell into
// repo-level tooling, canonicalizing overrides that point inside the repo.
func resolveVrooliRoot() string {
	if root := reporoot.Resolve(os.Getenv); root != "" {
		return root
	}
	for _, path := range []string{"/opt/vrooli", "/usr/local/vrooli"} {
		if _, err := os.Stat(path); err == nil {
			if resolved, ok := reporoot.CanonicalizeOverride(path); ok {
				return resolved
			}
			return filepath.Clean(path)
		}
	}
	return ""
}
