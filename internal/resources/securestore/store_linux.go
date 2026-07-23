//go:build linux

package securestore

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Default uses libsecret's Secret Service command client. If no desktop
// keyring is available it fails closed, so a headless Linux target stays
// conditional instead of leaving Vault recovery material on disk.
func Default() Store {
	if _, err := exec.LookPath("secret-tool"); err != nil {
		return Unavailable("secret-tool (libsecret) is not installed")
	}
	return secretToolStore{}
}

type secretToolStore struct{}

func (secretToolStore) Put(service, key, value string) error {
	cmd := exec.Command("secret-tool", "store", "--label=Vrooli managed resource", "service", service, "key", key)
	cmd.Stdin = strings.NewReader(value)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("store secure resource material: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (secretToolStore) Get(service, key string) (string, error) {
	output, err := exec.Command("secret-tool", "lookup", "service", service, "key", key).Output()
	if err != nil {
		return "", fmt.Errorf("read secure resource material: %w", err)
	}
	return strings.TrimSuffix(string(output), "\n"), nil
}

func (secretToolStore) Delete(service, key string) error {
	cmd := exec.Command("secret-tool", "clear", "service", service, "key", key)
	if output, err := cmd.CombinedOutput(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return fmt.Errorf("%w: secret-tool is not installed", ErrUnavailable)
		}
		return fmt.Errorf("delete secure resource material: %w: %s", err, bytes.TrimSpace(output))
	}
	return nil
}
