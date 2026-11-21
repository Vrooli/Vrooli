package desktophelper

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
)

// ProvisionConfig serializes owner provisioning and installs only public helper
// configuration. The resolver receives an existing pin as credential-loss
// evidence. Private signing material must never cross this callback boundary.
func ProvisionConfig(path string, requested Config, resolvePublic func(previousPin string) (string, error)) (Config, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || path == requested.XAuthorityFile || path == requested.GrantStatusFile || path == requested.StateDirectory || path == requested.AccessibilitySocket || resolvePublic == nil || requested.PublicKey != "" {
		return Config{}, ErrBootstrap
	}
	requested.PublicKey = base64.StdEncoding.EncodeToString(make([]byte, 32))
	if err := validateConfig(requested); err != nil {
		return Config{}, err
	}
	requested.PublicKey = ""
	if err := privateDirectory(filepath.Dir(path)); err != nil {
		return Config{}, err
	}
	unlock, err := lockPrivateFile(path + ".lock")
	if err != nil {
		return Config{}, err
	}
	defer unlock()
	previous := ""
	if _, err = os.Lstat(path); err == nil {
		existing, err := LoadConfig(path)
		if err != nil {
			return Config{}, err
		}
		previous = existing.PublicKey
		existing.PublicKey = ""
		if existing != requested {
			// Only an explicit protected accessibility binding may be changed.
			// Preserve the existing signing pin and all destination identity.
			comparable := requested
			comparable.AccessibilitySocket = existing.AccessibilitySocket
			comparable.AccessibilityBusID = existing.AccessibilityBusID
			if comparable != existing {
				return Config{}, ErrBootstrap
			}
			if err := privateDirectory(existing.StateDirectory); err != nil {
				return Config{}, err
			}
			releaseHelper, err := lockDirectory(existing.StateDirectory)
			if err != nil {
				return Config{}, ErrBootstrap
			}
			defer releaseHelper()
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return Config{}, err
	}
	public, err := resolvePublic(previous)
	if err != nil {
		return Config{}, err
	}
	if previous != "" && public != previous {
		return Config{}, ErrBootstrap
	}
	requested.PublicKey = public
	if err = validateConfig(requested); err != nil {
		return Config{}, err
	}
	if err = writePrivateJSON(path, requested); err != nil {
		return Config{}, err
	}
	return requested, nil
}
