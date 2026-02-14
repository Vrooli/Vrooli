package pathutil

import (
	"os"
	"path/filepath"
	"strings"
)

// DOC: docs/internal/SECURITY-POSTURE.md
// DOC: docs/internal/SEAMS.md

// ResolveScenarioRoot resolves the absolute scenario root path.
// Priority:
// 1) SCENARIO_ROOT (if set and absolute/relative resolvable)
// 2) VROOLI_ROOT/scenarios/{scenario}
// 3) Walk up from cwd searching for scenarios/{scenario}
// 4) Fallback to cwd/scenarios/{scenario}
func ResolveScenarioRoot(scenario string) string {
	name := strings.TrimSpace(scenario)
	if name == "" {
		name = "swarm-manager"
	}

	if explicit := strings.TrimSpace(os.Getenv("SCENARIO_ROOT")); explicit != "" {
		if abs, err := filepath.Abs(explicit); err == nil {
			return abs
		}
	}

	if vrooliRoot := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); vrooliRoot != "" {
		candidate := filepath.Join(vrooliRoot, "scenarios", name)
		if abs, err := filepath.Abs(candidate); err == nil {
			return abs
		}
	}

	if cwd, err := os.Getwd(); err == nil {
		dir := cwd
		for {
			candidate := filepath.Join(dir, "scenarios", name)
			if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
				if abs, absErr := filepath.Abs(candidate); absErr == nil {
					return abs
				}
				return candidate
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}

		fallback := filepath.Join(cwd, "scenarios", name)
		if abs, absErr := filepath.Abs(fallback); absErr == nil {
			return abs
		}
		return fallback
	}

	return filepath.Join("scenarios", name)
}

// ResolveScenariosDir resolves the absolute scenarios directory root.
func ResolveScenariosDir() string {
	return filepath.Dir(ResolveScenarioRoot("swarm-manager"))
}
