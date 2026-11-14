package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

// getVrooliRoot returns the active repo root using only contract resolution.
func getVrooliRoot() string {
	for _, key := range []string{"VROOLI_SOURCE_ROOT", "VROOLI_ROOT"} {
		if root := strings.TrimSpace(os.Getenv(key)); root != "" {
			if resolved, err := repocontract.FindRepoRootFromPath(root); err == nil {
				return resolved
			}
		}
	}
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		return ""
	}
	return root
}

func resolveScenarioRoot(scenario string) string {
	repoRoot := getVrooliRoot()
	scenario = strings.TrimSpace(scenario)
	if repoRoot == "" || scenario == "" {
		return ""
	}
	resolved, err := repocontract.ResolveScenarioPath(repoRoot, scenario)
	if err != nil {
		return ""
	}
	return resolved
}

func resolveTopLevelDir(key string) string {
	repoRoot := getVrooliRoot()
	if repoRoot == "" {
		return ""
	}
	contract, err := repocontract.LoadDefault(repoRoot)
	if err != nil {
		return ""
	}
	resolved, err := contract.TopLevelDir(repoRoot, key)
	if err != nil {
		return ""
	}
	return resolved
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
