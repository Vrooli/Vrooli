package ssh

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// GenerateKey generates a new SSH key pair.
func GenerateKey(platform Platform, req GenerateKeyRequest) (KeyInfo, string, error) {
	if err := validateKeyType(req.Type); err != nil {
		return KeyInfo{}, "", err
	}

	applyGenerateDefaults(&req)

	if err := ValidateKeyFilename(req.Filename); err != nil {
		return KeyInfo{}, "", err
	}

	keyPath, err := prepareKeyPath(platform, req.Filename)
	if err != nil {
		return KeyInfo{}, "", err
	}

	if err := runSSHKeygen(platform, req, keyPath); err != nil {
		return KeyInfo{}, "", err
	}

	return readGeneratedKey(platform, keyPath)
}

// validateKeyType checks that the key type is supported.
func validateKeyType(kt KeyType) error {
	if kt != KeyTypeEd25519 && kt != KeyTypeRSA {
		return fmt.Errorf("key type must be 'ed25519' or 'rsa'")
	}
	return nil
}

// applyGenerateDefaults fills in default values for the generate request.
func applyGenerateDefaults(req *GenerateKeyRequest) {
	if req.Type == KeyTypeRSA && req.Bits == 0 {
		req.Bits = 4096
	}
	if req.Filename == "" {
		if req.Type == KeyTypeEd25519 {
			req.Filename = "github_ed25519"
		} else {
			req.Filename = "github_rsa"
		}
	}
}

// prepareKeyPath ensures the SSH directory exists and returns the full key path.
// Returns an error if the key file already exists.
func prepareKeyPath(platform Platform, filename string) (string, error) {
	sshDir, err := platform.GetSSHDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		return "", fmt.Errorf("cannot create ~/.ssh directory: %w", err)
	}
	keyPath := filepath.Join(sshDir, filename)
	if _, err := os.Stat(keyPath); err == nil {
		return "", fmt.Errorf("key already exists: %s", filename)
	}
	return keyPath, nil
}

// buildKeygenArgs constructs the ssh-keygen command arguments.
func buildKeygenArgs(req GenerateKeyRequest, keyPath string) []string {
	args := []string{"-t", string(req.Type)}
	if req.Type == KeyTypeRSA && req.Bits > 0 {
		args = append(args, "-b", fmt.Sprintf("%d", req.Bits))
	}
	args = append(args, "-f", keyPath)
	comment := "github-key"
	if req.Comment != "" {
		comment = req.Comment
	}
	args = append(args, "-C", comment, "-N", "")
	return args
}

// runSSHKeygen executes ssh-keygen with the given request parameters.
func runSSHKeygen(platform Platform, req GenerateKeyRequest, keyPath string) error {
	args := buildKeygenArgs(req, keyPath)
	cmd := exec.Command(platform.SSHKeygenPath(), args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ssh-keygen failed: %s", stderr.String())
	}
	return nil
}

// readGeneratedKey reads the key info and public key content after generation.
func readGeneratedKey(platform Platform, keyPath string) (KeyInfo, string, error) {
	keyInfo, err := parseKeyFile(platform, keyPath)
	if err != nil {
		return KeyInfo{}, "", fmt.Errorf("failed to read generated key: %w", err)
	}
	pubKeyContent, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		return keyInfo, "", nil
	}
	return keyInfo, string(pubKeyContent), nil
}
