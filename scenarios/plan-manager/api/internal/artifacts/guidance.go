// Package artifacts owns the just-in-time guidance for durable plan evidence.
package artifacts

import (
	"os"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

// Instruction resolves the existing runtime-home contract, never a repository
// scratch directory. The same plan handle is used during authoring and execution.
func Instruction(handle string) string {
	home, err := os.UserHomeDir()
	if err == nil {
		root, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyPlanArtifacts)
		if err == nil && handle != "" && filepath.Base(handle) == handle && !strings.ContainsAny(handle, `/\\`) && handle != "." && handle != ".." {
			return "Store optional planning source, submission files, and execution evidence in " + filepath.Join(root, handle) + ". The canonical plan is published separately by Plan Manager. Reference source files directly unless an immutable copy is required; keep these planning artifacts outside scenario source directories."
		}
	}
	return "Planning artifact location unavailable; resolve the runtime-home plan_artifacts contract before saving supporting files. Keep planning artifacts outside scenario source directories."
}
