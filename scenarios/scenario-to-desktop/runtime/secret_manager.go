package bundleruntime

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"

	"scenario-to-desktop-runtime/manifest"
)

// SecretManager implements SecretStore for managing secrets.
type SecretManager struct {
	manifest    *manifest.Manifest
	fs          FileSystem
	secretsPath string

	mu      sync.RWMutex
	secrets map[string]string
}

// NewSecretManager creates a new SecretManager with the given dependencies.
func NewSecretManager(m *manifest.Manifest, fs FileSystem, secretsPath string) *SecretManager {
	return &SecretManager{
		manifest:    m,
		fs:          fs,
		secretsPath: secretsPath,
		secrets:     make(map[string]string),
	}
}

// Load reads secrets from persistent storage.
// Returns an empty map if the file doesn't exist.
func (sm *SecretManager) Load() (map[string]string, error) {
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
func (sm *SecretManager) Persist(secrets map[string]string) error {
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
func (sm *SecretManager) Get() map[string]string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	out := make(map[string]string, len(sm.secrets))
	for k, v := range sm.secrets {
		out[k] = v
	}
	return out
}

// Set updates the internal secrets map (thread-safe).
func (sm *SecretManager) Set(secrets map[string]string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.secrets = make(map[string]string, len(secrets))
	for k, v := range secrets {
		sm.secrets[k] = v
	}
}

// MissingRequired returns IDs of required secrets that are missing.
func (sm *SecretManager) MissingRequired() []string {
	return sm.MissingRequiredFrom(sm.Get())
}

// MissingRequiredFrom checks a secrets map for missing required values.
func (sm *SecretManager) MissingRequiredFrom(secrets map[string]string) []string {
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
func (sm *SecretManager) FindSecret(id string) *manifest.Secret {
	for i := range sm.manifest.Secrets {
		if sm.manifest.Secrets[i].ID == id {
			return &sm.manifest.Secrets[i]
		}
	}
	return nil
}

// Merge combines new secrets with existing ones and updates the store.
func (sm *SecretManager) Merge(newSecrets map[string]string) map[string]string {
	merged := sm.Get()
	for k, v := range newSecrets {
		merged[k] = v
	}
	return merged
}

// Ensure SecretManager implements SecretStore.
var _ SecretStore = (*SecretManager)(nil)
