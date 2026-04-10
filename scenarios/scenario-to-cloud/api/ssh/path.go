package ssh

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GetSSHDir returns the user's ~/.ssh directory.
func GetSSHDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(homeDir, ".ssh"), nil
}

// ValidateSSHPath ensures the path is within ~/.ssh/ to prevent path traversal.
func ValidateSSHPath(path string) error {
	// Handle ~ expansion
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		path = filepath.Join(homeDir, path[2:])
	}

	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Get ~/.ssh directory
	sshDir, err := GetSSHDir()
	if err != nil {
		return err
	}

	// Ensure path is within ~/.ssh
	if !strings.HasPrefix(absPath, sshDir+string(os.PathSeparator)) && absPath != sshDir {
		return fmt.Errorf("path must be within ~/.ssh")
	}

	// Check for path traversal
	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal not allowed")
	}

	return nil
}

// ValidateKeyFilename ensures the filename is safe.
func ValidateKeyFilename(filename string) error {
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}
	if strings.ContainsAny(filename, "/\\") {
		return fmt.Errorf("filename cannot contain path separators")
	}
	if strings.Contains(filename, "..") {
		return fmt.Errorf("filename cannot contain '..'")
	}
	if len(filename) > 64 {
		return fmt.Errorf("filename too long (max 64 characters)")
	}
	// Must start with alphanumeric or underscore
	if len(filename) > 0 {
		c := filename[0]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return fmt.Errorf("filename must start with alphanumeric character or underscore")
		}
	}
	return nil
}

// ExpandPath expands ~ to home directory.
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(homeDir, path[2:])
	}
	return path
}
