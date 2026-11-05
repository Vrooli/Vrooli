package domains

import (
	"os"
	"path/filepath"
)

// RepoScenarioLocator resolves a scenario name to <repoRoot>/scenarios/<name>.
// It is the production ScenarioLocator.
type RepoScenarioLocator struct {
	repoRoot string
}

// NewRepoScenarioLocator constructs a locator rooted at the repository
// root.
func NewRepoScenarioLocator(repoRoot string) *RepoScenarioLocator {
	return &RepoScenarioLocator{repoRoot: repoRoot}
}

var _ ScenarioLocator = (*RepoScenarioLocator)(nil)

// Locate returns the scenario's root directory, verifying it exists.
func (l *RepoScenarioLocator) Locate(scenario string) (string, error) {
	dir := filepath.Join(l.repoRoot, "scenarios", scenario)
	if scenario == "control-plane" {
		dir = l.repoRoot
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return "", ErrScenarioNotFound{Scenario: scenario}
	}
	return dir, nil
}
