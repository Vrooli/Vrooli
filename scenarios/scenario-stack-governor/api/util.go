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
	_, err := os.Stat(path)
	return err == nil
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

