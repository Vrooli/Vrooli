package pathutil

import (
	"path/filepath"
	"sort"
	"strings"
)

// ChangedPathScope groups changed repo paths into direct scenario changes and
// shared paths that cannot be mapped to a single scenario in v1.
type ChangedPathScope struct {
	DirectScenarioPaths map[string][]string
	SharedPaths         []string
}

// ScenarioFromRepoPath maps a repo-relative path to a scenario name when the
// path is under scenarios/<name>/...
func ScenarioFromRepoPath(path string) (string, bool) {
	normalized := filepath.ToSlash(strings.TrimSpace(path))
	if !strings.HasPrefix(normalized, "scenarios/") {
		return "", false
	}
	rest := strings.TrimPrefix(normalized, "scenarios/")
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		return "", false
	}
	return parts[0], true
}

// GroupChangedPaths partitions repo-relative changed paths into direct scenario
// paths and shared/ambiguous paths.
func GroupChangedPaths(paths []string) ChangedPathScope {
	scope := ChangedPathScope{
		DirectScenarioPaths: map[string][]string{},
		SharedPaths:         []string{},
	}
	seenShared := map[string]struct{}{}
	for _, path := range paths {
		trimmed := filepath.ToSlash(strings.TrimSpace(path))
		if trimmed == "" {
			continue
		}
		if scenarioName, ok := ScenarioFromRepoPath(trimmed); ok {
			scope.DirectScenarioPaths[scenarioName] = appendUnique(scope.DirectScenarioPaths[scenarioName], trimmed)
			continue
		}
		if _, exists := seenShared[trimmed]; exists {
			continue
		}
		seenShared[trimmed] = struct{}{}
		scope.SharedPaths = append(scope.SharedPaths, trimmed)
	}
	for scenarioName := range scope.DirectScenarioPaths {
		sort.Strings(scope.DirectScenarioPaths[scenarioName])
	}
	sort.Strings(scope.SharedPaths)
	return scope
}

// UniqueSortedStrings deduplicates and sorts a string slice.
func UniqueSortedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	sort.Strings(result)
	return result
}

func appendUnique(values []string, candidate string) []string {
	for _, existing := range values {
		if existing == candidate {
			return values
		}
	}
	return append(values, candidate)
}
