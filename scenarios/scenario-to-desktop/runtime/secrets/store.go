// Package secrets provides secret management for the bundle runtime.
package secrets

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"

	"scenario-to-desktop-runtime/infra"
	"scenario-to-desktop-runtime/manifest"
)

// Store abstracts secret storage for testing.
type Store interface {
	// Load reads secrets from persistent storage.
	Load() (map[string]string, error)
	// Persist saves secrets to persistent storage.
	Persist(secrets map[string]string) error
	// Get returns a copy of the current secrets.
	Get() map[string]string
	// Set updates the internal secrets.
	Set(secrets map[string]string)
	// MissingRequired returns IDs of required secrets that are missing.
	MissingRequired() []string
	// MissingRequiredFrom checks a secrets map for missing required values.
	MissingRequiredFrom(secrets map[string]string) []string
	// FindSecret looks up a secret definition by ID.
	FindSecret(id string) *manifest.Secret
	// Merge combines new secrets with existing ones.
	Merge(newSecrets map[string]string) map[string]string
}

// Manager implements Store for managing secrets.
type Manager struct {
	manifest    *manifest.Manifest
	fs          infra.FileSystem
	secretsPath string

	mu      sync.RWMutex
	secrets map[string]string
}

// NewManager creates a new Manager with the given dependencies.
func NewManager(m *manifest.Manifest, fs infra.FileSystem, secretsPath string) *Manager {
	return &Manager{
		manifest:    m,
		fs:          fs,
		secretsPath: secretsPath,
		secrets:     make(map[string]string),
	}
}

// Load reads secrets from persistent storage.
// Returns an empty map if the file doesn't exist.
func (sm *Manager) Load() (map[string]string, error) {
	out := map[string]string{}
	data, err := sm.fs.ReadFile(sm.secretsPath)
	if err != nil {
		// Check if file doesn't exist
		if _, statErr := sm.fs.Stat(sm.secretsPath); statErr != nil {
			return out, nil
		}
		return nil, err
	}

	// Try new format first: {"secrets": {...}}
	var wrapper struct {
		Secrets map[string]string `json:"secrets"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil || wrapper.Secrets == nil {
		// Fall back to legacy flat format.
		var legacy map[string]string
		if err2 := json.Unmarshal(data, &legacy); err2 == nil {
			return legacy, nil
		}
		return nil, err
	}
	return wrapper.Secrets, nil
}

// Persist saves secrets to persistent storage.
// The file is created with 0600 permissions for security.
func (sm *Manager) Persist(secrets map[string]string) error {
	if err := sm.fs.MkdirAll(filepath.Dir(sm.secretsPath), 0o700); err != nil {
		return err
	}

	payload := struct {
		Secrets map[string]string `json:"secrets"`
	}{
		Secrets: secrets,
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return sm.fs.WriteFile(sm.secretsPath, data, 0o600)
}

// Get returns a copy of the current secrets.
func (sm *Manager) Get() map[string]string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	out := make(map[string]string, len(sm.secrets))
	for k, v := range sm.secrets {
		out[k] = v
	}
	return out
}

// Set updates the internal secrets map (thread-safe).
func (sm *Manager) Set(secrets map[string]string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.secrets = make(map[string]string, len(secrets))
	for k, v := range secrets {
		sm.secrets[k] = v
	}
}

// MissingRequired returns IDs of required secrets that are missing.
func (sm *Manager) MissingRequired() []string {
	return sm.MissingRequiredFrom(sm.Get())
}

// MissingRequiredFrom checks a secrets map for missing required values.
func (sm *Manager) MissingRequiredFrom(secrets map[string]string) []string {
	var missing []string
	for _, sec := range sm.manifest.Secrets {
		required := true
		if sec.Required != nil {
			required = *sec.Required
		}
		if !required {
			continue
		}
		val := strings.TrimSpace(secrets[sec.ID])
		if val == "" {
			missing = append(missing, sec.ID)
		}
	}
	return missing
}

// FindSecret looks up a secret definition by ID.
func (sm *Manager) FindSecret(id string) *manifest.Secret {
	for i := range sm.manifest.Secrets {
		if sm.manifest.Secrets[i].ID == id {
			return &sm.manifest.Secrets[i]
		}
	}
	return nil
}

// Merge combines new secrets with existing ones.
func (sm *Manager) Merge(newSecrets map[string]string) map[string]string {
	merged := sm.Get()
	for k, v := range newSecrets {
		merged[k] = v
	}
	return merged
}

// Ensure Manager implements Store.
var _ Store = (*Manager)(nil)
