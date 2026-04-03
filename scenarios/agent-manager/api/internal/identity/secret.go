package identity

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
)

const (
	secretFileName = "identity-secret.key"
	secretSize     = 32
)

// LoadOrCreateSecret loads the HMAC secret from secretPath, or creates one if
// it does not exist. Returns an error if the file is unreadable, corrupted
// (wrong size), or the directory is not writable.
func LoadOrCreateSecret(dataDir string) ([]byte, error) {
	path := filepath.Join(dataDir, secretFileName)

	data, err := os.ReadFile(path)
	if err == nil {
		if len(data) != secretSize {
			return nil, fmt.Errorf("identity secret at %s has invalid size %d (expected %d)", path, len(data), secretSize)
		}
		return data, nil
	}

	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("cannot read identity secret at %s: %w", path, err)
	}

	// Generate a new secret.
	secret := make([]byte, secretSize)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("failed to generate identity secret: %w", err)
	}

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("cannot create data directory %s: %w", dataDir, err)
	}

	if err := os.WriteFile(path, secret, 0o600); err != nil {
		return nil, fmt.Errorf("cannot write identity secret to %s: %w", path, err)
	}

	return secret, nil
}
