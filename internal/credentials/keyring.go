// Package credentials owns the control-plane keyring capability. The secure
// store remains the format-specific implementation; this package is the seam
// shared by CLI and scenario clients so neither grows a private repair copy.
package credentials

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/resources/securestore"
)

type KeyringReport = securestore.KeyringReport

func DefaultKeyringPath(path string) (string, error) {
	if strings.TrimSpace(path) != "" {
		return path, nil
	}
	dir, err := securestore.DefaultKeyringDir()
	if err != nil {
		return "", err
	}
	matches, err := filepath.Glob(filepath.Join(dir, "*.keyring"))
	if err != nil {
		return "", fmt.Errorf("list keyring files: %w", err)
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no keyring files found")
	}
	sort.Strings(matches)
	return matches[0], nil
}

func Inspect(path string) (KeyringReport, error) {
	path, err := DefaultKeyringPath(path)
	if err != nil {
		return KeyringReport{}, err
	}
	return securestore.InspectKeyringFile(path)
}

func Repair(path string) (KeyringReport, error) {
	path, err := DefaultKeyringPath(path)
	if err != nil {
		return KeyringReport{}, err
	}
	return securestore.RepairKeyringFile(path)
}

func IsPasswordless(path string) (bool, error) {
	path, err := DefaultKeyringPath(path)
	if err != nil {
		return false, err
	}
	return securestore.IsPasswordlessKeyring(path)
}

func Unlock(ctx context.Context, input io.Reader) error {
	return securestore.UnlockLoginKeyring(ctx, input)
}
