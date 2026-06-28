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

// State records the adapter's last-write summary. It lives next to the
// target config file (~/.grok/.vrooli-permissions-state.json for user
// scope, ~/.grok/.vrooli-permissions-state.admin.json for admin scope)
// and is authoritative for drift detection: if the live config
// fingerprint diverges, the user (or some other tool) hand-edited it.
type State struct {
	SchemaVersion int       `json:"schemaVersion"`
	Fingerprint   string    `json:"fingerprint"`
	WrittenByVer  string    `json:"writtenByVersion"`
	WrittenAt     time.Time `json:"writtenAt"`
	SettingsPath  string    `json:"settingsPath"`
	Scope         Scope     `json:"scope"`
}

// StatePath returns the conventional state-file path. Admin scope gets a
// distinct filename so user and admin state never collide.
func (a *Adapter) StatePath() string {
	name := ".vrooli-permissions-state.json"
	if a.Scope == ScopeAdmin {
		name = ".vrooli-permissions-state.admin.json"
	}
	return filepath.Join(filepath.Dir(a.SettingsPath), name)
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

// WriteState persists the state record. Call this from CLI verbs after a
// successful Save; the adapter itself stays I/O-minimal.
func (a *Adapter) WriteState(p Policy, writtenByVersion string) error {
	s := State{
		SchemaVersion: StateSchemaVersion,
		Fingerprint:   Fingerprint(p),
		WrittenByVer:  writtenByVersion,
		WrittenAt:     time.Now().UTC(),
		SettingsPath:  a.SettingsPath,
		Scope:         a.Scope,
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
