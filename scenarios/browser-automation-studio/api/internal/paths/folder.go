package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	repocontract "github.com/vrooli/repo-contract-go"
)

// ResolveScenarioDir attempts to locate the absolute scenario root directory (scenarios/browser-automation-studio)
// using repo-contract-backed repository discovery. Falls back to a best-effort path under cwd.
func ResolveScenarioDir(log *logrus.Logger) string {
	if root, err := repocontract.FindRepoRootFromEnvOrCWD(); err == nil {
		if scenarioDir, resolveErr := repocontract.ResolveScenarioPath(root, scenarioRoot); resolveErr == nil {
			return scenarioDir
		} else if log != nil {
			log.WithError(resolveErr).Warn("Failed to resolve scenario path from repo contract; falling back to cwd-derived path")
		}
	} else if log != nil {
		log.WithError(err).Warn("Failed to resolve repo root from repo contract; falling back to cwd-derived path")
	}

	cwd, err := os.Getwd()
	if err != nil {
		if log != nil {
			log.WithError(err).Warn("Failed to resolve working directory; using relative scenario path")
		}
		return filepath.Join("scenarios", scenarioRoot)
	}

	absCwd, err := filepath.Abs(cwd)
	if err == nil {
		for dir := absCwd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
			parent := filepath.Dir(dir)
			if filepath.Base(dir) == scenarioRoot && filepath.Base(parent) == "scenarios" {
				return dir
			}
		}
	}

	root := filepath.Join(absCwd, "scenarios", scenarioRoot)
	if abs, err := filepath.Abs(root); err == nil {
		return abs
	}
	return root
}

// ResolveProjectsRoot returns the absolute directory where project folders should live.
func ResolveProjectsRoot(log *logrus.Logger) string {
	return filepath.Join(ResolveScenarioDir(log), "data", "projects")
}

// ResolveDemoProjectFolder returns the canonical filesystem folder for the seed/demo project.
func ResolveDemoProjectFolder(log *logrus.Logger) string {
	return filepath.Join(ResolveProjectsRoot(log), "demo")
}

// ValidateAndNormalizeFolderPath validates a folder path and returns the absolute normalized path.
// Returns an error with appropriate message if validation fails.
func ValidateAndNormalizeFolderPath(folderPath string, log *logrus.Logger) (string, error) {
	trimmed := strings.TrimSpace(folderPath)
	if trimmed == "" {
		return "", fmt.Errorf("invalid path")
	}

	// Get absolute path
	absPath, err := filepath.Abs(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid path")
	}

	return absPath, nil
}

// EnsureDirectoryExists creates the directory if it doesn't exist.
// Returns an error if directory creation fails.
func EnsureDirectoryExists(path string, log *logrus.Logger) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		if log != nil {
			log.WithError(err).WithField("folder_path", path).Error("Failed to create project directory")
		}
		return fmt.Errorf("failed to create directory")
	}
	return nil
}

// ValidateAndPrepareFolderPath combines validation and directory creation.
// Returns the absolute normalized path or an error.
func ValidateAndPrepareFolderPath(folderPath string, log *logrus.Logger) (string, error) {
	absPath, err := ValidateAndNormalizeFolderPath(folderPath, log)
	if err != nil {
		return "", err
	}

	if err := EnsureDirectoryExists(absPath, log); err != nil {
		return "", err
	}

	return absPath, nil
}
