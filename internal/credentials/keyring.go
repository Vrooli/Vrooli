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

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/resources/securestore"
)

const (
	keyringLocked = "locked"
)

type (
	KeyringReport = securestore.KeyringReport
	KeyringBackup = securestore.KeyringBackup
)

type KeyringVerdictState string

const (
	KeyringUnlocked     KeyringVerdictState = "unlocked"
	KeyringLocked       KeyringVerdictState = keyringLocked
	KeyringFileRejected KeyringVerdictState = "file_rejected"
	KeyringDaemonStale  KeyringVerdictState = "daemon_stale"
	KeyringAbsent       KeyringVerdictState = "absent"
)

type KeyringVerdict struct {
	State  KeyringVerdictState `json:"state"`
	Reason string              `json:"reason,omitempty"`
}

func DeriveKeyringVerdict(report KeyringReport, capability hostinventory.CredentialStoreCapability) KeyringVerdict {
	if report.Assessed && !report.Loadable {
		return KeyringVerdict{State: KeyringFileRejected, Reason: "the keyring file contains malformed entries"}
	}
	if report.StaleDaemon {
		return KeyringVerdict{State: KeyringDaemonStale, Reason: report.StaleDaemonDetail}
	}
	switch capability.State {
	case "ready":
		return KeyringVerdict{State: KeyringUnlocked, Reason: "Secret Service answered a login-collection read"}
	case keyringLocked:
		return KeyringVerdict{State: KeyringLocked, Reason: capability.Reason}
	case "empty", "unavailable", "unsupported":
		return KeyringVerdict{State: KeyringAbsent, Reason: capability.Reason}
	default:
		return KeyringVerdict{State: KeyringAbsent, Reason: capability.Reason}
	}
}

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

// RetireBackup removes one explicitly named, regular keyring backup. The
// control-plane CLI owns this mutation so operators never need a shell delete.
func RetireBackup(path string) error {
	return securestore.RetireKeyringBackup(path)
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
