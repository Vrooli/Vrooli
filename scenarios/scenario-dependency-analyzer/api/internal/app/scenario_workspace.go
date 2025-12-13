package app

import (
	"os"
	"path/filepath"

	appconfig "scenario-dependency-analyzer/internal/config"
	types "scenario-dependency-analyzer/internal/types"
)

// scenarioWorkspace makes the scenarios directory, service configs, and
// existence checks explicit so services share a single view of the workspace.
type scenarioWorkspace struct {
	root string
}

// newScenarioWorkspace builds a workspace rooted at the configured scenarios directory.
// Falls back to environment-loaded config to remain compatible with legacy callers.
func newScenarioWorkspace(cfg appconfig.Config) *scenarioWorkspace {
	root := cfg.ScenariosDir
	if root == "" {
		root = appconfig.Load().ScenariosDir
	}
	return &scenarioWorkspace{root: root}
}

func (w *scenarioWorkspace) pathFor(name string) string {
	return filepath.Join(w.root, name)
}

func (w *scenarioWorkspace) loadConfig(name string) (*types.ServiceConfig, error) {
	return appconfig.LoadServiceConfig(w.pathFor(name))
}

func (w *scenarioWorkspace) listScenarioNames() ([]string, error) {
	entries, err := os.ReadDir(w.root)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if w.hasServiceConfig(entry.Name()) {
			names = append(names, entry.Name())
		}
	}
	return names, nil
}

func (w *scenarioWorkspace) hasServiceConfig(name string) bool {
	servicePath := filepath.Join(w.root, name, ".vrooli", "service.json")
	if _, err := os.Stat(servicePath); err != nil {
		return false
	}
	return true
}
