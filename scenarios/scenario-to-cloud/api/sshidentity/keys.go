package sshidentity

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ReadPublicKeyAndFingerprint returns the .pub content and fingerprint for a private key path.
func ReadPublicKeyAndFingerprint(privateKeyPath string) (publicKey, fingerprint string, err error) {
	keyPath := expandHome(strings.TrimSpace(privateKeyPath))
	if keyPath == "" {
		return "", "", os.ErrNotExist
	}
	pubPath := keyPath + ".pub"
	content, readErr := os.ReadFile(pubPath)
	if readErr != nil {
		return "", "", readErr
	}
	publicKey = strings.TrimSpace(string(content))
	if publicKey == "" {
		return "", "", os.ErrInvalid
	}

	out, runErr := exec.Command("ssh-keygen", "-lf", pubPath).Output()
	if runErr != nil {
		return publicKey, "", nil
	}
	parts := strings.Fields(string(out))
	if len(parts) > 1 {
		fingerprint = strings.TrimSpace(parts[1])
	}
	return publicKey, fingerprint, nil
}

func expandHome(path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				return home
			}
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return path
}
