package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vrooli/api-core/storage"
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
	return resolveContractScenarioPathFromRepoRoot(root, name)
}

func resolveContractScenarioPathFromRepoRoot(root, name string) (string, error) {
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
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", fmt.Errorf("create storage resolver: %w", err)
	}
	dataDir, err := storage.EnsureClassDir(
		resolver,
		storage.Options{ScenarioID: "scenario-auditor"},
		storage.ClassData,
		0o755,
	)
	if err != nil {
		return "", fmt.Errorf("ensure scenario-auditor data dir: %w", err)
	}
	legacy := filepath.Join(root, ".vrooli", "data", "scenario-auditor")
	if _, statErr := os.Stat(legacy); statErr == nil {
		if _, dstErr := os.Stat(dataDir); os.IsNotExist(dstErr) {
			if err := os.Rename(legacy, dataDir); err != nil {
				return "", fmt.Errorf("migrate legacy scenario-auditor data dir: %w", err)
			}
		}
	}
	return dataDir, nil
}
