package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func trimEnv(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// isScenarioDir returns true if the directory name looks like a real scenario.
// Directories starting with '_' or '.' are excluded (e.g. _artifacts, .git).
func isScenarioDir(name string) bool {
	return name != "" && name[0] != '_' && name[0] != '.'
}

func scenarioRootFromCWD() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for i := 0; i < 8; i++ {
		if fileExists(filepath.Join(dir, ".vrooli", "service.json")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("scenario root not found from cwd")
}

func configPathForScenario(scenarioRoot string) string {
	return filepath.Join(scenarioRoot, "config", "rules.json")
}
