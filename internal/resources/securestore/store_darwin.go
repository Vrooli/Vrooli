//go:build darwin

package securestore

import (
	"fmt"
	"os/exec"
	"strings"
)

func Default() Store {
	if _, err := exec.LookPath("security"); err != nil {
		return Unavailable("macOS security tool is not installed")
	}
	return keychainStore{}
}

type keychainStore struct{}

func (keychainStore) Put(service, key, value string) error {
	output, err := exec.Command("security", "add-generic-password", "-U", "-s", service, "-a", key, "-w", value).CombinedOutput()
	if err != nil {
		return fmt.Errorf("store secure resource material: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}
func (keychainStore) Get(service, key string) (string, error) {
	output, err := exec.Command("security", "find-generic-password", "-w", "-s", service, "-a", key).Output()
	if err != nil {
		return "", fmt.Errorf("read secure resource material: %w", err)
	}
	return strings.TrimSuffix(string(output), "\n"), nil
}
func (keychainStore) Delete(service, key string) error {
	output, err := exec.Command("security", "delete-generic-password", "-s", service, "-a", key).CombinedOutput()
	if err != nil {
		return fmt.Errorf("delete secure resource material: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}
