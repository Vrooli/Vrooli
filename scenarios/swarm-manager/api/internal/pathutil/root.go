package pathutil

import (
	"os"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

// DOC: docs/internal/SECURITY-POSTURE.md
// DOC: docs/internal/SEAMS.md

// ResolveScenarioRoot resolves the absolute scenario root path.
// Priority:
// 1) SCENARIO_ROOT when explicitly provided
// 2) repo-contract-backed repository discovery plus canonical scenario layout
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

	root, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err == nil {
		if path, resolveErr := repocontract.ResolveScenarioPath(root, name); resolveErr == nil {
			return path
		}
	}

	return ""
}

// ResolveScenariosDir resolves the absolute scenarios directory root.
func ResolveScenariosDir() string {
	root := ResolveScenarioRoot("swarm-manager")
	if root == "" {
		return ""
	}
	return filepath.Dir(root)
}

// ScenariosFromGlobs extracts deduplicated scenario names from acceptance glob
// patterns. Globs that start with "scenarios/<name>/..." yield the scenario
// name; all other patterns are skipped.
func ScenariosFromGlobs(globs []string) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, g := range globs {
		if !strings.HasPrefix(g, "scenarios/") {
			continue
		}
		rest := strings.TrimPrefix(g, "scenarios/")
		parts := strings.SplitN(rest, "/", 2)
		if len(parts) == 0 || parts[0] == "" {
			continue
		}
		name := parts[0]
		if _, ok := seen[name]; !ok {
			seen[name] = struct{}{}
			result = append(result, name)
		}
	}
	return result
}
