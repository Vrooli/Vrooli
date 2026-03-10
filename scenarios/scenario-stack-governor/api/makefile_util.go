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

// canonicalTargetSet returns the 16 canonical Makefile targets that every scenario must define.
func canonicalTargetSet() map[string]struct{} {
	targets := []string{
		"help", "start", "stop", "test", "logs", "status",
		"clean", "build", "fmt", "fmt-go", "fmt-ui",
		"lint", "lint-go", "lint-ui", "check", "dev",
	}
	set := make(map[string]struct{}, len(targets))
	for _, t := range targets {
		set[t] = struct{}{}
	}
	return set
}
