// Package config provides profile storage for isolation configurations.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/vrooli/api-core/storage"

	"workspace-sandbox/internal/types"
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

	// MaskPaths lists host paths hidden from the workload by mounting an
	// empty tmpfs over them (after all other binds, so deny beats allow).
	// Use $HOME, $USER, $VROOLI_ROOT as placeholders. Intended for host
	// state the home overlay would otherwise expose but no workload needs
	// — e.g. unrelated repository checkouts under $HOME.
	MaskPaths []string `json:"maskPaths,omitempty"`

	// HomeOverlayRequirement declares how strongly this profile depends
	// on the per-sandbox host-$HOME overlay being present:
	//
	//   - "not_needed":  profile ignores $HOME (e.g. HOME=/tmp).
	//   - "optional":    profile uses $HOME-relative paths when the
	//                    overlay is Present, falls back gracefully when
	//                    Absent. Callers record HOME_OVERLAY_FALLBACK
	//                    audit code instead of refusing.
	//   - "required":    profile cannot function without the overlay.
	//                    Handlers MUST refuse exec with HTTP 409
	//                    (HomeOverlayRequiredError) when the sandbox's
	//                    HomeOverlayState is anything other than
	//                    HomeOverlayPresent — failing fast at exec time
	//                    prevents the silent
	//                    "env: $HOME/.local/bin/agent: No such file"
	//                    at process spawn.
	//
	// DOC: home-overlay seam — profile-side requirement declaration.
	// See docs/internal/SEAMS.md.
	HomeOverlayRequirement types.HomeOverlayRequirement `json:"homeOverlayRequirement"`

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
func NewFileProfileStore(_ string) (*FileProfileStore, error) {
	path, err := resolveProfilesPath()
	if err != nil {
		return nil, err
	}
	return &FileProfileStore{path: path}, nil
}

// NewFileProfileStoreAtPath creates a profile store pinned to an explicit file path.
// This is primarily intended for tests.
func NewFileProfileStoreAtPath(path string) *FileProfileStore {
	return &FileProfileStore{path: path}
}

func resolveProfilesPath() (string, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", err
	}
	return resolver.Path(
		storage.Options{ScenarioID: "workspace-sandbox"},
		storage.ClassConfig,
		"profiles.json",
	)
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
			// HOME=/tmp; no host $HOME visibility expected.
			HomeOverlayRequirement: types.HomeOverlayNotNeeded,
			// /etc/ssl/certs is bound so TLS clients using the system
			// trust store (rustls-native-certs, OpenSSL, Go's crypto/x509
			// fallback) find the host CA bundle. Without it, Rust agents
			// like Codex emit "no native root CA certificates found" and
			// every HTTPS handshake fails. The dir is mostly symlinks
			// into /usr/share/ca-certificates (already covered by the
			// /usr bind) plus the generated ca-certificates.crt bundle.
			ReadOnlyBinds: map[string]string{
				"/usr":             "/usr",
				"/lib":             "/lib",
				"/lib64":           "/lib64",
				"/bin":             "/bin",
				"/etc/resolv.conf": "/etc/resolv.conf",
				"/etc/hosts":       "/etc/hosts",
				"/etc/passwd":      "/etc/passwd",
				"/etc/group":       "/etc/group",
				"/etc/ssl/certs":   "/etc/ssl/certs",
			},
			ReadWriteBinds: map[string]string{},
			Environment: map[string]string{
				"PATH": "/usr/local/bin:/usr/bin:/bin",
				"HOME": "/tmp",
				// SSL_CERT_FILE / SSL_CERT_DIR pin the trust store path
				// explicitly so TLS libraries that don't probe the
				// default locations still find the bundle.
				"SSL_CERT_FILE": "/etc/ssl/certs/ca-certificates.crt",
				"SSL_CERT_DIR":  "/etc/ssl/certs",
				"SHELL":         "/bin/sh",
			},
			Hostname:  "sandbox",
			MaskPaths: []string{"$HOME/.codex-worktrees"},
		},
		{
			ID:            "vrooli-aware",
			Name:          "Vrooli-Aware",
			Description:   "Access to Vrooli CLIs, configs, and localhost network for API communication.",
			Builtin:       true,
			NetworkAccess: "localhost",
			// PATH=$HOME/... and HOME=$HOME require the host-home overlay
			// to be present; the handler refuses exec otherwise.
			HomeOverlayRequirement: types.HomeOverlayRequired,
			// Most $HOME-relative state is provided by the per-sandbox
			// HOME overlay set up in driver.Mount. One path is
			// intentionally different: ~/.vrooli is mounted read-only
			// from the host after the HOME overlay bind so agents see
			// live runtime state, logs, and scenario databases instead
			// of a stale per-sandbox snapshot. The bind is read-only
			// because Vrooli runtime state is outside the tracked
			// workspace; host mutations must go through controlled
			// lifecycle/scenario APIs.
			// See `full` profile for why /etc/ssl/certs is bound — Codex
			// (rustls) and any system-OpenSSL caller need the host CA
			// trust store to verify HTTPS endpoints.
			ReadOnlyBinds: map[string]string{
				"/usr":             "/usr",
				"/lib":             "/lib",
				"/lib64":           "/lib64",
				"/bin":             "/bin",
				"/etc/resolv.conf": "/etc/resolv.conf",
				"/etc/hosts":       "/etc/hosts",
				"/etc/passwd":      "/etc/passwd",
				"/etc/group":       "/etc/group",
				"/etc/ssl/certs":   "/etc/ssl/certs",
				"$VROOLI_ROOT":     "/vrooli",
				"$HOME/.vrooli":    "$HOME/.vrooli",
			},
			ReadWriteBinds: map[string]string{},
			Environment: map[string]string{
				// PATH includes canonical Vrooli and Go tool locations
				// surfaced through the HOME overlay, then standard
				// system paths. This makes Vrooli agents independent of
				// the caller's interactive shell while keeping every
				// $HOME-relative write auditable through the sandbox.
				"PATH": "$HOME/.vrooli/bin:$HOME/go/bin:$HOME/.local/bin:/usr/local/bin:/usr/bin:/bin",
				// HOME points to the host home so $HOME-relative
				// lookups resolve to the overlay merged dir, not /tmp.
				"HOME": "$HOME",
				// SSL_CERT_FILE / SSL_CERT_DIR pin the trust store path
				// explicitly — see `full` profile comment.
				"SSL_CERT_FILE": "/etc/ssl/certs/ca-certificates.crt",
				"SSL_CERT_DIR":  "/etc/ssl/certs",
				"SHELL":         "/bin/sh",
				"VROOLI_ROOT":   "/vrooli",
				"VROOLI_ENV":    "$VROOLI_ENV",
			},
			Hostname: "sandbox",
			// Other repository checkouts under $HOME are never a workload's
			// business; the home overlay would otherwise expose them
			// read-visible (2026-07-20 escaped-agent incident).
			MaskPaths: []string{"$HOME/.codex-worktrees"},
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

	if profile.HomeOverlayRequirement == "" {
		profile.HomeOverlayRequirement = types.HomeOverlayNotNeeded
	}
	if !profile.HomeOverlayRequirement.IsValid() {
		return fmt.Errorf(
			"profile %q: invalid homeOverlayRequirement %q (want one of %q/%q/%q)",
			profile.ID, profile.HomeOverlayRequirement,
			types.HomeOverlayNotNeeded, types.HomeOverlayOptional, types.HomeOverlayRequired,
		)
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

	for i := range profiles {
		if profiles[i].HomeOverlayRequirement == "" {
			profiles[i].HomeOverlayRequirement = types.HomeOverlayNotNeeded
		}
		if !profiles[i].HomeOverlayRequirement.IsValid() {
			return fmt.Errorf(
				"profile %q: invalid homeOverlayRequirement %q (want one of %q/%q/%q)",
				profiles[i].ID, profiles[i].HomeOverlayRequirement,
				types.HomeOverlayNotNeeded, types.HomeOverlayOptional, types.HomeOverlayRequired,
			)
		}
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
