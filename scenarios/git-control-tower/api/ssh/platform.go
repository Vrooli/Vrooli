package ssh

import (
	"os"
	"path/filepath"
)

// Platform abstracts OS-specific operations for SSH key management.
// This interface provides seams for future Windows/macOS support.
type Platform interface {
	// GetSSHDir returns the path to the user's SSH directory (e.g., ~/.ssh).
	GetSSHDir() (string, error)
	// GetHomeDir returns the user's home directory.
	GetHomeDir() (string, error)
	// SSHKeygenPath returns the path to the ssh-keygen binary.
	SSHKeygenPath() string
	// SSHPath returns the path to the ssh binary.
	SSHPath() string
}

// LinuxPlatform implements Platform for Linux systems.
type LinuxPlatform struct{}

// GetSSHDir returns ~/.ssh on Linux.
func (p *LinuxPlatform) GetSSHDir() (string, error) {
	homeDir, err := p.GetHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".ssh"), nil
}

// GetHomeDir returns the user's home directory.
func (p *LinuxPlatform) GetHomeDir() (string, error) {
	return os.UserHomeDir()
}

// SSHKeygenPath returns the standard path to ssh-keygen on Linux.
func (p *LinuxPlatform) SSHKeygenPath() string {
	return "ssh-keygen"
}

// SSHPath returns the standard path to ssh on Linux.
func (p *LinuxPlatform) SSHPath() string {
	return "ssh"
}

// DefaultPlatform returns the platform implementation for the current OS.
func DefaultPlatform() Platform {
	return &LinuxPlatform{}
}
