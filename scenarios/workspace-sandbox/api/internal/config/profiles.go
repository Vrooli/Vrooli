// Package config provides profile storage for isolation configurations.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// IsolationProfile defines a named isolation configuration.
// Profiles control what resources are accessible inside the sandbox.
type IsolationProfile struct {
	// ID is the unique identifier for this profile.
	ID string `json:"id"`

	// Name is the human-readable display name.
	Name string `json:"name"`

	// Description explains what this profile is for.
	Description string `json:"description"`

	// Builtin indicates this is a system-defined profile that cannot be deleted.
	Builtin bool `json:"builtin"`

	// NetworkAccess controls network visibility.
	// Options: "none", "localhost", "full"
	NetworkAccess string `json:"networkAccess"`

	// ReadOnlyBinds maps host paths to sandbox paths (read-only).
	// Use $HOME, $USER, $VROOLI_ROOT as placeholders.
	ReadOnlyBinds map[string]string `json:"readOnlyBinds"`

	// ReadWriteBinds maps host paths to sandbox paths (read-write).
	ReadWriteBinds map[string]string `json:"readWriteBinds"`

	// Environment variables to set inside the sandbox.
	// Use $VAR syntax to reference host environment variables.
	Environment map[string]string `json:"environment"`

	// Hostname to set inside the sandbox.
	Hostname string `json:"hostname"`

	// Future extensibility (currently unused, reserved for later)
	// SharePID bool `json:"sharePID,omitempty"`
	// AllowDevices bool `json:"allowDevices,omitempty"`
	// SeccompProfile string `json:"seccompProfile,omitempty"`
}

// ProfileStore manages isolation profile storage and retrieval.
type ProfileStore interface {
	// List returns all available profiles (builtin + custom).
	List() ([]IsolationProfile, error)

	// Get returns a profile by ID.
	Get(id string) (*IsolationProfile, error)

	// Save creates or updates a custom profile.
	// Returns error if trying to modify a builtin profile.
	Save(profile IsolationProfile) error

	// Delete removes a custom profile.
	// Returns error if trying to delete a builtin profile.
	Delete(id string) error
}

// FileProfileStore implements ProfileStore using a JSON file.
type FileProfileStore struct {
	path   string
	mu     sync.RWMutex
	cache  []IsolationProfile
	loaded bool
}

// NewFileProfileStore creates a profile store backed by a JSON file.
// basePath should be the scenario directory (e.g., scenarios/workspace-sandbox).
func NewFileProfileStore(basePath string) *FileProfileStore {
	return &FileProfileStore{
		path: filepath.Join(basePath, ".vrooli", "workspace-sandbox-profiles.json"),
	}
}

// DefaultProfiles returns the built-in isolation profiles.
func DefaultProfiles() []IsolationProfile {
	return []IsolationProfile{
		{
			ID:            "full",
			Name:          "Full Isolation",
			Description:   "Maximum isolation - only /workspace and basic system paths accessible. No network access.",
			Builtin:       true,
			NetworkAccess: "none",
			ReadOnlyBinds: map[string]string{
				"/usr":             "/usr",
				"/lib":             "/lib",
				"/lib64":           "/lib64",
				"/bin":             "/bin",
				"/etc/resolv.conf": "/etc/resolv.conf",
				"/etc/hosts":       "/etc/hosts",
				"/etc/passwd":      "/etc/passwd",
				"/etc/group":       "/etc/group",
			},
			ReadWriteBinds: map[string]string{},
			Environment: map[string]string{
				"PATH":  "/usr/local/bin:/usr/bin:/bin",
				"HOME":  "/tmp",
				"SHELL": "/bin/sh",
			},
			Hostname: "sandbox",
		},
		{
			ID:            "vrooli-aware",
			Name:          "Vrooli-Aware",
			Description:   "Access to Vrooli CLIs, configs, and localhost network for API communication.",
			Builtin:       true,
			NetworkAccess: "localhost",
			ReadOnlyBinds: map[string]string{
				"/usr":                 "/usr",
				"/lib":                 "/lib",
				"/lib64":               "/lib64",
				"/bin":                 "/bin",
				"/etc/resolv.conf":     "/etc/resolv.conf",
				"/etc/hosts":           "/etc/hosts",
				"/etc/passwd":          "/etc/passwd",
				"/etc/group":           "/etc/group",
				"$HOME/.local/bin":     "/usr/local/bin",
				"$HOME/.config/vrooli": "$HOME/.config/vrooli",
				"$VROOLI_ROOT":         "/vrooli",
			},
			ReadWriteBinds: map[string]string{},
			Environment: map[string]string{
				"PATH":        "/usr/local/bin:/usr/bin:/bin",
				"HOME":        "/tmp",
				"SHELL":       "/bin/sh",
				"VROOLI_ROOT": "/vrooli",
				"VROOLI_ENV":  "$VROOLI_ENV",
			},
			Hostname: "sandbox",
		},
	}
}

// List returns all profiles (builtin merged with custom).
func (s *FileProfileStore) List() ([]IsolationProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	// Start with builtin profiles
	defaults := DefaultProfiles()
	result := make([]IsolationProfile, len(defaults))
	copy(result, defaults)

	// Track builtin IDs for deduplication
	builtinIDs := make(map[string]bool)
	for _, p := range defaults {
		builtinIDs[p.ID] = true
	}

	// Add custom profiles that aren't overriding builtins
	for _, p := range s.cache {
		if !builtinIDs[p.ID] {
			result = append(result, p)
		}
	}

	return result, nil
}

// Get returns a profile by ID.
func (s *FileProfileStore) Get(id string) (*IsolationProfile, error) {
	profiles, err := s.List()
	if err != nil {
		return nil, err
	}

	for i := range profiles {
		if profiles[i].ID == id {
			return &profiles[i], nil
		}
	}

	return nil, fmt.Errorf("profile not found: %s", id)
}

// Save creates or updates a custom profile.
func (s *FileProfileStore) Save(profile IsolationProfile) error {
	// Check if trying to modify builtin
	for _, b := range DefaultProfiles() {
		if b.ID == profile.ID && b.Builtin {
			return fmt.Errorf("cannot modify builtin profile: %s", profile.ID)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureLoaded(); err != nil {
		return err
	}

	// Update or append
	found := false
	for i, p := range s.cache {
		if p.ID == profile.ID {
			s.cache[i] = profile
			found = true
			break
		}
	}
	if !found {
		s.cache = append(s.cache, profile)
	}

	return s.persist()
}

// Delete removes a custom profile.
func (s *FileProfileStore) Delete(id string) error {
	// Check if trying to delete builtin
	for _, b := range DefaultProfiles() {
		if b.ID == id {
			return fmt.Errorf("cannot delete builtin profile: %s", id)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureLoaded(); err != nil {
		return err
	}

	found := false
	newCache := make([]IsolationProfile, 0, len(s.cache))
	for _, p := range s.cache {
		if p.ID != id {
			newCache = append(newCache, p)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("profile not found: %s", id)
	}

	s.cache = newCache
	return s.persist()
}

// ensureLoaded loads the cache from disk if not already loaded.
// Must be called with mu held.
func (s *FileProfileStore) ensureLoaded() error {
	if s.loaded {
		return nil
	}

	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		s.cache = []IsolationProfile{}
		s.loaded = true
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read profiles: %w", err)
	}

	var profiles []IsolationProfile
	if err := json.Unmarshal(data, &profiles); err != nil {
		return fmt.Errorf("failed to parse profiles: %w", err)
	}

	s.cache = profiles
	s.loaded = true
	return nil
}

// persist writes the cache to disk.
// Must be called with mu held.
func (s *FileProfileStore) persist() error {
	// Ensure directory exists
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create profiles directory: %w", err)
	}

	data, err := json.MarshalIndent(s.cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal profiles: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write profiles: %w", err)
	}

	return nil
}

// Reload clears the cache and reloads from disk.
// Useful after external modifications to the profiles file.
func (s *FileProfileStore) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.loaded = false
	s.cache = nil
	return s.ensureLoaded()
}
