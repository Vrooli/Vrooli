// Package scenarioroot resolves a scenario name (or explicit path) to an
// absolute scenario directory on disk. The runner domains (benchmark,
// lighthouse, capture) all need to locate a target scenario's source tree, so
// the resolution rule lives in one place: an explicit --path always wins; a
// bare scenario name resolves through the repo contract.
package scenarioroot

import (
	"fmt"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

// Resolve maps (scenario, path) to an absolute scenario root.
//
//   - When path is set it is made absolute and returned verbatim (the caller
//     pointed us at a directory directly — used by tests and ad-hoc audits).
//   - Otherwise scenario is resolved through the repo contract under repoRoot
//     (empty repoRoot resolves the repo root lazily).
func Resolve(repoRoot, scenario, path string) (string, error) {
	scenario = strings.TrimSpace(scenario)
	path = strings.TrimSpace(path)
	if path != "" {
		return filepath.Abs(path)
	}
	if scenario == "" {
		return "", fmt.Errorf("scenario or path is required")
	}
	root := strings.TrimSpace(repoRoot)
	if root == "" {
		resolved, err := repocontract.ResolveRepoRoot()
		if err != nil {
			return "", err
		}
		root = resolved
	}
	return repocontract.ResolveScenarioPath(root, scenario)
}
