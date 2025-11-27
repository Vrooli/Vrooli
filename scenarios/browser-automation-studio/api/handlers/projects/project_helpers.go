package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// validateAndNormalizeFolderPath validates a folder path and returns the absolute normalized path.
// Returns an error with appropriate message if validation fails.
func validateAndNormalizeFolderPath(folderPath string, log *logrus.Logger) (string, error) {
	// Get absolute path
	absPath, err := filepath.Abs(folderPath)
	if err != nil {
		return "", fmt.Errorf("invalid path")
	}

	// Get allowed root directory
	allowedRoot := strings.TrimSpace(os.Getenv("VROOLI_ROOT"))
	if allowedRoot == "" {
		cwd, cwdErr := os.Getwd()
		if cwdErr != nil {
			log.WithError(cwdErr).Error("Failed to resolve VROOLI_ROOT for folder validation")
			return "", fmt.Errorf("failed to resolve project root")
		}
		allowedRoot = cwd
	}

	// Normalize allowed root
	allowedRoot, err = filepath.Abs(allowedRoot)
	if err != nil {
		log.WithError(err).Error("Failed to normalize VROOLI_ROOT for folder validation")
		return "", fmt.Errorf("failed to normalize project root")
	}

	// Check if path is within allowed root
	if !strings.HasPrefix(absPath, allowedRoot+string(os.PathSeparator)) && absPath != allowedRoot {
		return "", fmt.Errorf("folder path must be inside project root")
	}

	return absPath, nil
}

// ensureDirectoryExists creates the directory if it doesn't exist.
// Returns an error if directory creation fails.
func ensureDirectoryExists(path string, log *logrus.Logger) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		log.WithError(err).WithField("folder_path", path).Error("Failed to create project directory")
		return fmt.Errorf("failed to create directory")
	}
	return nil
}

// validateAndPrepareFolderPath combines validation and directory creation.
// Returns the absolute normalized path or an error.
func validateAndPrepareFolderPath(folderPath string, log *logrus.Logger) (string, error) {
	absPath, err := validateAndNormalizeFolderPath(folderPath, log)
	if err != nil {
		return "", err
	}

	if err := ensureDirectoryExists(absPath, log); err != nil {
		return "", err
	}

	return absPath, nil
}
