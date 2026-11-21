// Package scenariopath locates the react-component-library scenario root from
// wherever a CLI command runs: a subdirectory, the repository root, or a
// sandbox overlay. It is the one resolver; catalog, asset and workspaces all
// use it so "where is the scenario" cannot drift between commands.
package scenariopath

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vrooli/cli-core/cliutil"
)

// ScenarioName is the directory name under scenarios/.
const ScenarioName = "react-component-library"

// Find walks up from the working directory, the declared scenario root and the
// shared resolver's answer. It returns the first directory that has both the
// API module and the catalog config.
func Find() (string, error) {
	starts := make([]string, 0, 4)
	if cwd, err := os.Getwd(); err == nil {
		starts = append(starts, cwd)
	}
	for _, env := range []string{"VROOLI_SCENARIO_ROOT", "SCENARIO_PATH", "VROOLI_SOURCE_ROOT"} {
		if value := filepath.Clean(os.Getenv(env)); value != "" && value != "." {
			starts = append(starts, value)
		}
	}
	if resolved := cliutil.ResolveScenarioPath(ScenarioName); resolved != "" {
		starts = append(starts, resolved)
	}
	for _, start := range starts {
		for current := filepath.Clean(start); ; current = filepath.Dir(current) {
			if IsRoot(current) {
				return current, nil
			}
			parent := filepath.Dir(current)
			if parent == current {
				break
			}
		}
	}
	return "", fmt.Errorf("cannot locate the %s scenario root", ScenarioName)
}

// IsRoot reports whether a directory is the scenario root. It checks the two
// files every command in this scenario depends on.
func IsRoot(path string) bool {
	_, apiErr := os.Stat(filepath.Join(path, "api", "go.mod"))
	_, catalogErr := os.Stat(filepath.Join(path, "catalog", "config.json"))
	return apiErr == nil && catalogErr == nil
}
