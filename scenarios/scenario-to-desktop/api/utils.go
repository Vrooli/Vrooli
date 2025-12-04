package main

import (
	"os"
	"path/filepath"
	"strings"
)

func contains(list []string, item string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}

func uniqueStrings(input []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(input))
	for _, value := range input {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func detectVrooliRoot() string {
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot != "" {
		return vrooliRoot
	}
	currentDir, err := os.Getwd()
	if err != nil {
		return "../../.."
	}
	return filepath.Clean(filepath.Join(currentDir, "../../.."))
}

// containsParentRef checks for attempts to traverse upward in a path.
func containsParentRef(path string) bool {
	if strings.Contains(path, "..") {
		return true
	}
	parts := strings.Split(path, string(filepath.Separator))
	for _, part := range parts {
		if part == ".." || part == "." {
			return true
		}
	}
	return false
}
