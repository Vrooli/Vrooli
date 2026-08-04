package securestore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
)

const (
	backendSelectionVersion = 1
	backendSelectionFile    = "credential-backend.json"
)

// BackendSelection is the setup-time authority decision for this installation.
// It contains no credential material. Once written, transient native-store
// outages do not change the authority and cannot split credentials across two
// backends.
type BackendSelection struct {
	Version    int       `json:"version"`
	Backend    string    `json:"backend"`
	SelectedAt time.Time `json:"selected_at"`
	Reason     string    `json:"reason,omitempty"`
}

var backendSelectionPath = func() (string, error) {
	return config.VrooliPath(repocontract.HomeKeyState, backendSelectionFile)
}

// SelectedBackend returns the persisted setup decision. A missing file means
// this is an older installation that has not passed through setup's backend
// decision yet; callers may use the legacy discovery behavior for that one
// transition state.
func SelectedBackend() (string, bool, error) {
	path, err := backendSelectionPath()
	if err != nil {
		return "", false, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("read credential backend selection: %w", err)
	}
	var selection BackendSelection
	if err := json.Unmarshal(data, &selection); err != nil {
		return "", false, fmt.Errorf("decode credential backend selection: %w", err)
	}
	if selection.Version != backendSelectionVersion {
		return "", false, fmt.Errorf("credential backend selection version %d is unsupported", selection.Version)
	}
	if err := validateBackend(selection.Backend); err != nil {
		return "", false, err
	}
	return selection.Backend, true, nil
}

// SelectBackend records the single backend this installation uses. It is an
// explicit setup operation, not a fallback triggered by an individual read or
// write failure.
func SelectBackend(backend, reason string) error {
	if err := validateBackend(backend); err != nil {
		return err
	}
	path, err := backendSelectionPath()
	if err != nil {
		return err
	}
	selection := BackendSelection{
		Version:    backendSelectionVersion,
		Backend:    backend,
		SelectedAt: time.Now().UTC(),
		Reason:     strings.TrimSpace(reason),
	}
	data, err := json.MarshalIndent(selection, "", "  ")
	if err != nil {
		return fmt.Errorf("encode credential backend selection: %w", err)
	}
	if err := config.WriteOwnedFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write credential backend selection: %w", err)
	}
	return nil
}

func validateBackend(backend string) error {
	switch strings.TrimSpace(backend) {
	case BackendNative, BackendEncryptedFile:
		return nil
	default:
		return fmt.Errorf("unsupported credential backend %q; use %q or %q", backend, BackendNative, BackendEncryptedFile)
	}
}
