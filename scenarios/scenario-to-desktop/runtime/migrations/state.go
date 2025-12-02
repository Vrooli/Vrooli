// Package migrations provides migration tracking for the bundle runtime.
package migrations

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"scenario-to-desktop-runtime/infra"
	"scenario-to-desktop-runtime/manifest"
)

// State tracks applied migrations per service and the current app version.
type State struct {
	AppVersion string              `json:"app_version"`
	Applied    map[string][]string `json:"applied"` // service ID -> list of applied migration versions
}

// NewState creates an empty migrations state.
func NewState() State {
	return State{
		Applied: make(map[string][]string),
	}
}

// Tracker manages migration state persistence.
type Tracker struct {
	Path string
	FS   infra.FileSystem
}

// NewTracker creates a new migrations tracker.
func NewTracker(path string, fs infra.FileSystem) *Tracker {
	return &Tracker{
		Path: path,
		FS:   fs,
	}
}

// Load reads the migrations state from disk.
// Returns an empty state if the file doesn't exist.
func (t *Tracker) Load() (State, error) {
	state := NewState()
	data, err := t.FS.ReadFile(t.Path)
	if err != nil {
		// Check if file doesn't exist
		if _, statErr := t.FS.Stat(t.Path); statErr != nil {
			return state, nil
		}
		return state, err
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return state, err
	}
	if state.Applied == nil {
		state.Applied = make(map[string][]string)
	}
	return state, nil
}

// Persist saves the migrations state to disk.
func (t *Tracker) Persist(state State) error {
	if err := t.FS.MkdirAll(filepath.Dir(t.Path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return t.FS.WriteFile(t.Path, data, 0o600)
}

// Phase determines if this is a first install, upgrade, or current version.
func Phase(state State, currentVersion string) string {
	if state.AppVersion == "" {
		return "first_install"
	}
	if state.AppVersion != currentVersion {
		return "upgrade"
	}
	return "current"
}

// ShouldRun checks if a migration should run based on its run_on condition.
func ShouldRun(m manifest.Migration, phase string) bool {
	runOn := strings.TrimSpace(m.RunOn)
	if runOn == "" {
		runOn = "always"
	}
	switch runOn {
	case "always":
		return true
	case "first_install":
		return phase == "first_install"
	case "upgrade":
		return phase == "upgrade"
	default:
		return false
	}
}

// BuildAppliedSet creates a lookup set from applied version list.
func BuildAppliedSet(versions []string) map[string]bool {
	set := make(map[string]bool)
	for _, v := range versions {
		set[v] = true
	}
	return set
}

// MarkApplied records a migration as applied in the state.
func MarkApplied(state *State, serviceID, version string) {
	if state.Applied[serviceID] == nil {
		state.Applied[serviceID] = []string{}
	}
	state.Applied[serviceID] = append(state.Applied[serviceID], version)
}
