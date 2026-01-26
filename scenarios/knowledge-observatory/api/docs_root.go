package main

import (
	"os"
	"path/filepath"
	"strings"
)

func resolveScenariosRoot() string {
	if root := strings.TrimSpace(os.Getenv("VROOLI_SCENARIOS_ROOT")); root != "" {
		if isDir(root) {
			return root
		}
	}
	if repo := strings.TrimSpace(os.Getenv("VROOLI_REPO_ROOT")); repo != "" {
		candidate := filepath.Join(repo, "scenarios")
		if isDir(candidate) {
			return candidate
		}
	}
	if inferred := inferScenariosRoot(); inferred != "" {
		return inferred
	}
	return ""
}

func inferScenariosRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	current := wd
	for i := 0; i < 8; i++ {
		if filepath.Base(current) == "scenarios" && isDir(current) {
			return current
		}
		candidate := filepath.Join(current, "scenarios")
		if isDir(candidate) {
			return candidate
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return ""
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
