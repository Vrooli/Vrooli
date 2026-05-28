package permissions

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// StateSchemaVersion bumps when the on-disk State shape changes
// incompatibly. drift-check refuses to compare across versions.
const StateSchemaVersion = 1

// State records the adapter's last-write summary. It lives next to
// opencode.json (typically ~/.config/opencode/.vrooli-permissions-state.json)
// and is authoritative for drift detection.
type State struct {
	SchemaVersion int       `json:"schemaVersion"`
	Fingerprint   string    `json:"fingerprint"`
	WrittenByVer  string    `json:"writtenByVersion"`
	WrittenAt     time.Time `json:"writtenAt"`
	SettingsPath  string    `json:"settingsPath"`
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

// WriteState persists the state record. Call this from CLI verbs after
// a successful Save; the adapter itself stays I/O-minimal.
func (a *Adapter) WriteState(p Policy, writtenByVersion string) error {
	s := State{
		SchemaVersion: StateSchemaVersion,
		Fingerprint:   Fingerprint(p),
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
