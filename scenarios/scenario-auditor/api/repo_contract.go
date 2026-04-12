package main

import (
	"fmt"
	"path/filepath"

	repocontract "github.com/vrooli/repo-contract-go"
)

func resolveRepoRoot() (string, error) {
	root, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		return "", fmt.Errorf("resolve repo root: %w", err)
	}
	return root, nil
}

func resolveScenariosRoot() (string, error) {
	root, err := resolveRepoRoot()
	if err != nil {
		return "", err
	}
	contract, err := repocontract.LoadDefault(root)
	if err != nil {
		return "", fmt.Errorf("load repo contract: %w", err)
	}
	return filepath.Join(root, filepath.FromSlash(contract.Layout().ScenarioDir)), nil
}

func resolveContractScenarioPath(name string) (string, error) {
	root, err := resolveRepoRoot()
	if err != nil {
		return "", err
	}
	path, err := repocontract.ResolveScenarioPath(root, name)
	if err != nil {
		return "", fmt.Errorf("resolve scenario path: %w", err)
	}
	return path, nil
}

func resolveScenarioAuditorRoot() (string, error) {
	return resolveContractScenarioPath("scenario-auditor")
}

func relativeToRepoRoot(path string) string {
	root, err := resolveRepoRoot()
	if err != nil {
		return filepath.ToSlash(path)
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func resolveScenarioAuditorDataDir() (string, error) {
	root, err := resolveRepoRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ".vrooli", "data", "scenario-auditor"), nil
}
