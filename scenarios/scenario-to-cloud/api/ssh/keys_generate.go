package ssh

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// GenerateKey generates a new SSH key pair.
func GenerateKey(req GenerateKeyRequest) (KeyInfo, error) {
	// Validate key type
	if req.Type != KeyTypeEd25519 && req.Type != KeyTypeRSA {
		return KeyInfo{}, fmt.Errorf("key type must be 'ed25519' or 'rsa'")
	}

	// Set defaults
	if req.Type == KeyTypeRSA && req.Bits == 0 {
		req.Bits = 4096
	}
	if req.Filename == "" {
		if req.Type == KeyTypeEd25519 {
			req.Filename = "id_ed25519"
		} else {
			req.Filename = "id_rsa"
		}
	}

	// Validate filename
	if err := ValidateKeyFilename(req.Filename); err != nil {
		return KeyInfo{}, err
	}

	// Determine output path
	sshDir, err := GetSSHDir()
	if err != nil {
		return KeyInfo{}, err
	}

	// Ensure ~/.ssh exists with proper permissions
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		return KeyInfo{}, fmt.Errorf("cannot create ~/.ssh directory: %w", err)
	}

	keyPath := filepath.Join(sshDir, req.Filename)

	// Check if file already exists
	if _, err := os.Stat(keyPath); err == nil {
		return KeyInfo{}, fmt.Errorf("key already exists: %s", keyPath)
	}

	// Build ssh-keygen command
	args := []string{"-t", string(req.Type)}
	if req.Type == KeyTypeRSA && req.Bits > 0 {
		args = append(args, "-b", fmt.Sprintf("%d", req.Bits))
	}
	args = append(args, "-f", keyPath)
	if req.Comment != "" {
		args = append(args, "-C", req.Comment)
	} else {
		args = append(args, "-C", "generated-by-vrooli")
	}
	// Passphrase (empty string means no passphrase)
	args = append(args, "-N", req.Password)

	cmd := exec.Command("ssh-keygen", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return KeyInfo{}, fmt.Errorf("ssh-keygen failed: %s", stderr.String())
	}

	// Read the generated key info
	keyInfo, err := parseKeyFile(keyPath)
	if err != nil {
		return KeyInfo{}, fmt.Errorf("failed to read generated key: %w", err)
	}

	return keyInfo, nil
}
