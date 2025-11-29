package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// ValidateAndNormalizeFolderPath validates a folder path and returns the absolute normalized path.
// Returns an error with appropriate message if validation fails.
// The folder path must be within the allowed root (VROOLI_ROOT or cwd).
func ValidateAndNormalizeFolderPath(folderPath string, log *logrus.Logger) (string, error) {
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
			if log != nil {
				log.WithError(cwdErr).Error("Failed to resolve VROOLI_ROOT for folder validation")
			}
			return "", fmt.Errorf("failed to resolve project root")
		}
		allowedRoot = cwd
	}

	// Normalize allowed root
	allowedRoot, err = filepath.Abs(allowedRoot)
	if err != nil {
		if log != nil {
			log.WithError(err).Error("Failed to normalize VROOLI_ROOT for folder validation")
		}
		return "", fmt.Errorf("failed to normalize project root")
	}

	// Check if path is within allowed root
	if !strings.HasPrefix(absPath, allowedRoot+string(os.PathSeparator)) && absPath != allowedRoot {
		return "", fmt.Errorf("folder path must be inside project root")
	}

	return absPath, nil
}

// EnsureDirectoryExists creates the directory if it doesn't exist.
// Returns an error if directory creation fails.
func EnsureDirectoryExists(path string, log *logrus.Logger) error {
	if err := os.MkdirAll(path, 0755); err != nil {
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
