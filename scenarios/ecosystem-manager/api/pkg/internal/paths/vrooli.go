package paths

import (
	"path/filepath"

	repocontract "github.com/vrooli/repo-contract-go"
)

// DetectVrooliRoot finds the canonical root of the Vrooli workspace.
// Returns "." only when repo-contract resolution fails.
func DetectVrooliRoot() string {
	if root, err := repocontract.ResolveRepoRoot(); err == nil {
		return root
	}
	return filepath.Clean(".")
}
