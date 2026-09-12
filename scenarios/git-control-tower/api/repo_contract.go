package main

import (
	"fmt"
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

func resolveScopePath(repoRoot, scopeType, scopeName string) (string, error) {
	contract, err := repocontract.LoadDefault(repoRoot)
	if err != nil {
		return "", err
	}

	switch scopeType {
	case "scenario":
		rel, err := resolveScenarioPathRelative(repoRoot, scopeName)
		if err != nil {
			return "", err
		}
		return ensureTrailingSlash(rel), nil
	case "resource":
		return ensureTrailingSlash(filepath.ToSlash(filepath.Join(contract.Layout().ResourceDir, scopeName))), nil
	case "package":
		return ensureTrailingSlash(filepath.ToSlash(filepath.Join(contract.Layout().PackageDir, scopeName))), nil
	default:
		return "", fmt.Errorf("unsupported scope type %q", scopeType)
	}
}

func ensureTrailingSlash(path string) string {
	if path == "" || path[len(path)-1] == '/' {
		return path
	}
	return path + "/"
}
