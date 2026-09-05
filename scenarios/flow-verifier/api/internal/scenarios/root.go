// Package scenarios is the API's source of truth for what "a scenario"
// means in flow-verifier: a directory under <vrooli-root>/scenarios/
// containing a .vrooli/service.json descriptor. The flows domain stays
// scoped to a single root; this domain answers "which scenarios exist
// and how many flows does each one contain".
package scenarios

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// EnvVrooliRoot is the override agents and tests use to pin the
// detected Vrooli root regardless of CWD.
const EnvVrooliRoot = "FLOW_VERIFIER_VROOLI_ROOT"

// ResolveVrooliRoot returns the absolute path of the Vrooli repository
// root. Resolution order:
//
//  1. Environment override (EnvVrooliRoot) — used by tests and by
//     deployments that want to pin discovery to a specific path.
//  2. Walk up from startDir until a directory is found containing
//     both `scenarios/` and `.vrooli/` — the markers every Vrooli
//     checkout carries at its root.
//
// Returns an error (not a guess) when neither path succeeds: a silent
// fallback is what produced the original "empty inventory" bug.
func ResolveVrooliRoot(startDir string, lookupEnv func(string) string) (string, error) {
	if lookupEnv == nil {
		lookupEnv = os.Getenv
	}
	if override := lookupEnv(EnvVrooliRoot); override != "" {
		abs, err := filepath.Abs(override)
		if err != nil {
			return "", fmt.Errorf("scenarios: invalid %s=%q: %w", EnvVrooliRoot, override, err)
		}
		if !isVrooliRoot(abs) {
			return "", fmt.Errorf("scenarios: %s=%q is not a Vrooli root (missing scenarios/ or .vrooli/)", EnvVrooliRoot, abs)
		}
		return abs, nil
	}
	start, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	dir := start
	for {
		if isVrooliRoot(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("scenarios: could not locate Vrooli root from %q (no ancestor contains scenarios/ + .vrooli/); set %s to pin it", start, EnvVrooliRoot)
		}
		dir = parent
	}
}

func isVrooliRoot(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "scenarios")); err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(dir, ".vrooli")); err != nil {
		return false
	}
	return true
}

// ErrScenarioNotFound is returned by Service.Detail when the requested
// scenario id does not exist under the Vrooli root.
var ErrScenarioNotFound = errors.New("scenario not found")
