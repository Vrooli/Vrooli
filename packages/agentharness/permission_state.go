package agentharness

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// PermissionStateSchemaVersion is shared by every permissions adapter. The
// sidecar is intentionally provider-neutral so drift detection does not need
// to understand JSON, TOML, or the native provider schema.
const PermissionStateSchemaVersion = 2

// PermissionState records the last successful write to a native permission
// file. ManagedBash is used by adapters whose native format cannot safely
// carry an ownership marker (currently OpenCode).
type PermissionState struct {
	SchemaVersion int       `json:"schemaVersion"`
	Fingerprint   string    `json:"fingerprint"`
	ManagedBash   []string  `json:"managedBash,omitempty"`
	WrittenByVer  string    `json:"writtenByVersion"`
	WrittenAt     time.Time `json:"writtenAt"`
	SettingsPath  string    `json:"settingsPath"`
	Scope         string    `json:"scope,omitempty"`
}

// LoadPermissionState reads a sidecar. A missing sidecar is not an error.
func LoadPermissionState(path string) (*PermissionState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read state: %w", err)
	}
	var state PermissionState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}
	return &state, nil
}

// WritePermissionState persists a provider-neutral sidecar after a native
// adapter has successfully written its settings file.
func WritePermissionState(path string, policy PermissionPolicy, fingerprint, writtenByVersion, scope string) error {
	managed := append([]string{}, policy.BashDeny...)
	managed = append(managed, policy.BashAsk...)
	managed = append(managed, policy.BashAllow...)
	sort.Strings(managed)
	state := PermissionState{
		SchemaVersion: PermissionStateSchemaVersion,
		Fingerprint:   fingerprint,
		ManagedBash:   managed,
		WrittenByVer:  writtenByVersion,
		WrittenAt:     time.Now().UTC(),
		SettingsPath:  policy.SettingsPath,
		Scope:         scope,
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir state dir: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

// PermissionStateSettingsPath is kept separate from the state writer because
// adapters may use a provider-specific filename for admin or project scope.
func PermissionStateSettingsPath(settingsPath string) string {
	return filepath.Join(filepath.Dir(settingsPath), ".vrooli-permissions-state.json")
}
