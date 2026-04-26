package main

import (
	"path/filepath"
	"sort"
	"strings"
)

// locateMakefiles returns Makefile paths for the given scenario (or all scenarios when scenarioName is empty).
func locateMakefiles(repoRoot, scenarioName string) ([]string, error) {
	pattern := filepath.Join(repoRoot, "scenarios", "*", "Makefile")
	cleaned := strings.TrimSpace(scenarioName)
	if cleaned != "" {
		pattern = filepath.Join(repoRoot, "scenarios", cleaned, "Makefile")
	}
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	return matches, nil
}

// canonicalTargetSet returns all canonical Makefile targets: the 16 required
// targets plus known aliases. The fixer uses this set to decide which targets
// are "custom" (and should be preserved) vs part of the canonical template.
func canonicalTargetSet() map[string]struct{} {
	targets := append(requiredTargets(), canonicalAliases()...)
	set := make(map[string]struct{}, len(targets))
	for _, t := range targets {
		set[t] = struct{}{}
	}
	return set
}

// requiredTargets returns the 17 targets every scenario Makefile must define.
// This is the single source of truth — structureValidatePhony reads from it.
func requiredTargets() []string {
	return []string{
		"help", "start", "stop", "restart", "test", "logs", "status",
		"clean", "build", "fmt", "fmt-go", "fmt-ui",
		"lint", "lint-go", "lint-ui", "check", "dev",
	}
}

// canonicalAliases returns well-known alias targets that are part of the
// canonical template but are NOT required (they won't be flagged if missing).
func canonicalAliases() []string {
	return []string{"run"}
}
