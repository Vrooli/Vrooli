package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

func resolveScenarioFiles(scenarioPath string, files []string) ([]string, error) {
	root, err := filepath.Abs(filepath.Clean(scenarioPath))
	if err != nil {
		return nil, fmt.Errorf("resolve scenario root: %w", err)
	}

	absPaths := make([]string, 0, len(files))
	for _, relPath := range files {
		cleanRel := filepath.Clean(relPath)
		if cleanRel == "." || filepath.IsAbs(cleanRel) || strings.HasPrefix(cleanRel, ".."+string(filepath.Separator)) || cleanRel == ".." {
			return nil, fmt.Errorf("file path %q escapes scenario root", relPath)
		}

		candidate := filepath.Join(root, cleanRel)
		relToRoot, err := filepath.Rel(root, candidate)
		if err != nil {
			return nil, fmt.Errorf("resolve file path %q: %w", relPath, err)
		}
		if relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) || filepath.IsAbs(relToRoot) {
			return nil, fmt.Errorf("file path %q escapes scenario root", relPath)
		}

		absPaths = append(absPaths, candidate)
	}

	return absPaths, nil
}
