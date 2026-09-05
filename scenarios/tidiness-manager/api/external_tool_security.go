package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

var allowedExternalCommands = map[string]struct{}{
	"dupl":    {},
	"go":      {},
	"gocyclo": {},
	"make":    {},
	"npx":     {},
}

// commandExists only resolves the small set of tools Tidiness Manager is allowed
// to invoke. Call sites still pass fixed executable names and bounded arguments.
func commandExists(cmd string) bool {
	if _, ok := allowedExternalCommands[cmd]; !ok {
		return false
	}
	_, err := exec.LookPath(cmd)
	return err == nil
}

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
