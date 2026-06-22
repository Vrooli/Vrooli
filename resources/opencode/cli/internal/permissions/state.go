package permissions

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

// StateSchemaVersion bumps when the on-disk State shape changes
// incompatibly. drift-check refuses to compare across versions. v2 added
// ManagedBash, making the sidecar (not an inline opencode.json key) the
// source of truth for which bash patterns the adapter owns.
const StateSchemaVersion = 2

// State records the adapter's last-write summary. It lives next to
// opencode.json (typically ~/.config/opencode/.vrooli-permissions-state.json)
// and is authoritative for both drift detection and which bash patterns are
// Vrooli-managed.
type State struct {
	SchemaVersion int    `json:"schemaVersion"`
	Fingerprint   string `json:"fingerprint"`
	// ManagedBash is the sorted list of permission.bash patterns this adapter
	// owns. It is the source of truth for the managed set — opencode.json
	// cannot carry it because opencode rejects unknown top-level keys.
	ManagedBash  []string  `json:"managedBash"`
	WrittenByVer string    `json:"writtenByVersion"`
	WrittenAt    time.Time `json:"writtenAt"`
	SettingsPath string    `json:"settingsPath"`
}

// StatePath returns the conventional state-file path next to
// SettingsPath.
func (a *Adapter) StatePath() string {
	return filepath.Join(filepath.Dir(a.SettingsPath), ".vrooli-permissions-state.json")
}

// LoadState reads the state file. Missing file resolves to (nil, nil).
func (a *Adapter) LoadState() (*State, error) {
	data, err := os.ReadFile(a.StatePath())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read state: %w", err)
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}
	return &s, nil
}

// WriteState persists the state record, including the managed-pattern list
// derived from the policy. Save calls this after writing opencode.json so the
// sidecar stays the authoritative record of which entries are managed.
func (a *Adapter) WriteState(p Policy, writtenByVersion string) error {
	managed := make([]string, 0, len(p.BashDeny)+len(p.BashAsk)+len(p.BashAllow))
	managed = append(managed, p.BashDeny...)
	managed = append(managed, p.BashAsk...)
	managed = append(managed, p.BashAllow...)
	sort.Strings(managed)
	s := State{
		SchemaVersion: StateSchemaVersion,
		Fingerprint:   Fingerprint(p),
		ManagedBash:   managed,
		WrittenByVer:  writtenByVersion,
		WrittenAt:     time.Now().UTC(),
		SettingsPath:  a.SettingsPath,
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(a.StatePath()), 0o755); err != nil {
		return fmt.Errorf("mkdir state dir: %w", err)
	}
	return os.WriteFile(a.StatePath(), data, 0o644)
}
