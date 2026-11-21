package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

func resolveWorkspaceSandboxScenarioDir() (string, error) {
	for _, key := range []string{"VROOLI_SOURCE_ROOT", "VROOLI_ROOT"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			root, err := repocontract.FindRepoRoot(value)
			if err == nil {
				path, err := repocontract.ResolveScenarioPath(root, "workspace-sandbox")
				if err != nil {
					return "", fmt.Errorf("resolve workspace-sandbox scenario path: %w", err)
				}
				return filepath.Clean(path), nil
			}
		}
	}

	root, err := repocontract.FindRepoRootFromCWD()
	if err != nil {
		return "", fmt.Errorf("resolve repo root: %w", err)
	}
	path, err := repocontract.ResolveScenarioPath(root, "workspace-sandbox")
	if err != nil {
		return "", fmt.Errorf("resolve workspace-sandbox scenario path: %w", err)
	}
	return filepath.Clean(path), nil
}
