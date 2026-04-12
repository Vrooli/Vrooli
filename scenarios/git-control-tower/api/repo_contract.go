package main

import (
	"path/filepath"

	repocontract "github.com/vrooli/repo-contract-go"
)

func resolveScenarioPath(repoRoot, scenario string) (string, error) {
	return repocontract.ResolveScenarioPath(repoRoot, scenario)
}

func resolveScenarioPathRelative(repoRoot, scenario string) (string, error) {
	path, err := resolveScenarioPath(repoRoot, scenario)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(repoRoot, path)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}
