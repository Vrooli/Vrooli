package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

// getVrooliRoot returns the active repo root, preferring canonical contract
// resolution while preserving explicit non-contract overrides as a fallback.
func getVrooliRoot() string {
	for _, key := range []string{"VROOLI_SOURCE_ROOT", "VROOLI_ROOT"} {
		if root := strings.TrimSpace(os.Getenv(key)); root != "" {
			if resolved, ok := canonicalRepoRootFromOverride(root); ok {
				return resolved
			}
			return filepath.Clean(root)
		}
	}
	if root, err := repocontract.ResolveRepoRoot(); err == nil {
		return root
	}
	if cwd, err := os.Getwd(); err == nil {
		return filepath.Clean(cwd)
	}
	return "."
}

func canonicalRepoRootFromOverride(path string) (string, bool) {
	current := filepath.Clean(strings.TrimSpace(path))
	if current == "" || current == "." {
		return "", false
	}
	for depth := 0; depth < 25; depth++ {
		if resolved, err := repocontract.FindRepoRoot(current); err == nil {
			return resolved, true
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return "", false
}

func resolveScenarioRoot(scenario string) string {
	repoRoot := getVrooliRoot()
	scenario = strings.TrimSpace(scenario)
	if repoRoot == "" || scenario == "" {
		return ""
	}
	if resolved, err := repocontract.ResolveScenarioPath(repoRoot, scenario); err == nil {
		return resolved
	}
	return filepath.Join(repoRoot, "scenarios", scenario)
}

func resolveTopLevelDir(key string) string {
	repoRoot := getVrooliRoot()
	if repoRoot == "" {
		return ""
	}
	contract, err := repocontract.LoadDefault(repoRoot)
	if err == nil {
		if resolved, resolveErr := contract.TopLevelDir(repoRoot, key); resolveErr == nil {
			return resolved
		}
	}
	return filepath.Join(repoRoot, key)
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil && logger != nil {
		logger.Error("failed to write JSON response: %v", err)
	}
}

func stringPtr(s string) *string {
	return &s
}
