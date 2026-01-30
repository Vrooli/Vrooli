package ssh

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidateSSHPath ensures the path is within ~/.ssh/ to prevent path traversal.
func ValidateSSHPath(platform Platform, path string) error {
	// Handle ~ expansion
	if strings.HasPrefix(path, "~/") {
		homeDir, err := platform.GetHomeDir()
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
	sshDir, err := platform.GetSSHDir()
	if err != nil {
		return err
	}

	// Ensure path is within ~/.ssh
	// Check that absPath starts with sshDir followed by a separator, or is exactly sshDir
	if !strings.HasPrefix(absPath, sshDir+string(filepath.Separator)) && absPath != sshDir {
		return fmt.Errorf("path must be within ~/.ssh")
	}

	// Check for path traversal
	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal not allowed")
	}

	return nil
}

// ValidateKeyFilename ensures the filename is safe for use as an SSH key name.
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
	// Check for null bytes and other problematic characters
	for _, c := range filename {
		if c == 0 || c == '\n' || c == '\r' {
			return fmt.Errorf("filename contains invalid characters")
		}
	}
	return nil
}

// IsProtectedFile returns true if the filename is a protected SSH file that should not be deleted.
func IsProtectedFile(filename string) bool {
	protected := map[string]bool{
		"authorized_keys":  true,
		"authorized_keys2": true,
		"known_hosts":      true,
		"known_hosts.old":  true,
		"config":           true,
		"environment":      true,
		"rc":               true,
		"ssh_config":       true,
	}
	return protected[filename]
}

// ExpandPath expands ~ to the home directory.
func ExpandPath(platform Platform, path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := platform.GetHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(homeDir, path[2:])
	}
	return path
}
