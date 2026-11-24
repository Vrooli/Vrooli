package main

import (
	"fmt"
	"strings"
)

// validateGitRepoDeps checks the shared prerequisites for Git-backed services.
func validateGitRepoDeps(git GitRunner, repoDir string) (string, error) {
	if git == nil {
		return "", fmt.Errorf("git runner is required")
	}
	repoDir = strings.TrimSpace(repoDir)
	if repoDir == "" {
		return "", fmt.Errorf("repo dir is required")
	}
	return repoDir, nil
}
