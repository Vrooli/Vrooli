// Package secrets provides secret management for the bundle runtime.
package secrets

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	credentialauthority "github.com/vrooli/vrooli/internal/secrets"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/strutil"
)

// RecoveryStore is the optional recovery surface exposed by the desktop
// runtime. It is kept separate from Store so test-only in-memory stores do not
// need to pretend they can create durable recovery bundles.
type RecoveryStore interface {
	ExportRecovery(passphrase string) ([]byte, int, error)
	RestoreRecovery(bundle []byte, passphrase string) error
}

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
	// Validate checks secrets against manifest requirements (required, format).
	// Returns nil if all validations pass, or an error with details.
	Validate(secrets map[string]string) error
	// GenerateMissing generates values for per_install_generated secrets that don't exist.
	GenerateMissing(existingSecrets map[string]string) (map[string]string, error)
}

// Manager implements Store for managing secrets.
type Manager struct {
	manifest  *manifest.Manifest
	validator *Validator
	generator *Generator
	authority *credentialauthority.Authority
	identity  credentialauthority.Identity
	initErr   error

	mu      sync.RWMutex
	secrets map[string]string
}

func (sm *Manager) ExportRecovery(passphrase string) ([]byte, int, error) {
	if sm.authority == nil {
		return nil, 0, fmt.Errorf("desktop recovery unavailable: native credential authority is not configured")
	}
	entries := make([]credentialauthority.RecoveryEntry, 0, len(sm.manifest.Secrets))
	seen := make(map[string]bool)
	for _, definition := range sm.manifest.Secrets {
		identity, field, err := sm.addressOf(definition)
		if err != nil {
			return nil, 0, err
		}
		key := string(identity) + "\x00" + field
		if seen[key] {
			continue
		}
		seen[key] = true
		entries = append(entries, credentialauthority.RecoveryEntry{Identity: identity, Field: field})
	}
	bundle, err := sm.authority.ExportRecovery(entries, passphrase)
	if err != nil {
		return nil, 0, err
	}
	return bundle, len(entries), nil
}

func (sm *Manager) RestoreRecovery(bundle []byte, passphrase string) error {
	if sm.authority == nil {
		return fmt.Errorf("desktop recovery unavailable: native credential authority is not configured")
	}
	if err := sm.authority.RestoreRecovery(bundle, passphrase); err != nil {
		return err
	}
	loaded, err := sm.Load()
	if err != nil {
		return err
	}
	sm.Set(loaded)
	return nil
}

// NewNativeManager creates the production desktop credential store. Values are
// held only by the OS-native authority; the app-data directory is never a
// credential persistence location.
func NewNativeManager(m *manifest.Manifest) (*Manager, error) {
	if m == nil || len(m.Secrets) == 0 {
		return newManager(m), nil
	}
	authority, err := credentialauthority.DefaultAuthority()
	if err != nil {
		return nil, err
	}
	return NewNativeManagerWithAuthority(m, authority)
}

// NewNativeManagerWithAuthority supplies the already-probed authority used by
// a desktop runtime. It is useful to embedders that own native-store startup
// and to deterministic tests; it never accepts a file-backed substitute.
func NewNativeManagerWithAuthority(m *manifest.Manifest, authority *credentialauthority.Authority) (*Manager, error) {
	if authority == nil {
		return nil, fmt.Errorf("native credential authority is required")
	}
	identity, err := desktopIdentity(m)
	if err != nil {
		return nil, err
	}
	manager := newManager(m)
	manager.authority = authority
	manager.identity = identity
	return manager, nil
}

// NewUnavailableManager preserves a useful runtime error when the platform
// cannot provide a native authority. It intentionally does not fall back to a
// file, environment variable, or legacy secrets reader.
func NewUnavailableManager(m *manifest.Manifest, err error) *Manager {
	manager := newManager(m)
	manager.initErr = err
	return manager
}

// NewManager creates an in-memory manager for explicitly injected test or
// embedding flows. Desktop credentials must be supplied through
// NewNativeManager.
func NewManager(m *manifest.Manifest) *Manager {
	return newManager(m)
}

func newManager(m *manifest.Manifest) *Manager {
	return &Manager{
		manifest:  m,
		validator: NewValidator(m),
		generator: NewGenerator(),
		secrets:   make(map[string]string),
	}
}

// Load reads credentials from native authority or returns the explicitly
// injected in-memory values. It never reads a desktop secrets file.
func (sm *Manager) Load() (map[string]string, error) {
	out := map[string]string{}
	if sm.initErr != nil {
		return nil, fmt.Errorf("native credential authority unavailable: %w", sm.initErr)
	}
	if sm.authority != nil {
		for _, definition := range sm.manifest.Secrets {
			identity, field, err := sm.addressOf(definition)
			if err != nil {
				return nil, err
			}
			// The map key stays the secret's own ID: that is what the injector
			// and the prompt UI address it by. Only where the value is *stored*
			// has changed.
			if err := sm.authority.Inject(identity, field, definition.ID, out); err != nil {
				if errors.Is(err, credentialauthority.ErrUnconfigured) {
					continue
				}
				if errors.Is(err, credentialauthority.ErrProviderAbsent) {
					return nil, fmt.Errorf("native credential authority unavailable: %w; configure a credential backend or restore a recovery bundle before starting the desktop app", err)
				}
				return nil, fmt.Errorf("read desktop credential %s: %w", definition.ID, err)
			}
		}
		return out, nil
	}
	return sm.Get(), nil
}

// addressOf resolves where one bundle secret lives in the credential store.
//
// A secret that declares a logical_id resolves to the identity every other tier
// uses, so provisioning it once during onboarding is enough. One that does not
// falls back to the bundle's own namespace, which keeps an existing manifest
// working — but it is a bundle-local value that no Tier 1 install will find,
// and the generator fills the declaration in precisely to avoid that.
func (sm *Manager) addressOf(definition manifest.Secret) (credentialauthority.Identity, string, error) {
	field := definition.CredentialField()
	if field == "" {
		return "", "", fmt.Errorf("bundle credential has no id, field, or target name")
	}
	if declared := strings.TrimSpace(definition.LogicalID); declared != "" {
		identity, err := credentialauthority.ParseIdentity(declared)
		if err != nil {
			return "", "", fmt.Errorf("bundle credential %s declares an unusable logical_id: %w", definition.ID, err)
		}
		return identity, field, nil
	}
	return sm.identity, field, nil
}

// Persist writes native authority fields or updates injected in-memory values.
// It never creates a plaintext file.
func (sm *Manager) Persist(secrets map[string]string) error {
	if sm.initErr != nil {
		return fmt.Errorf("native credential authority unavailable: %w", sm.initErr)
	}
	if sm.authority != nil {
		for _, definition := range sm.manifest.Secrets {
			identity, field, err := sm.addressOf(definition)
			if err != nil {
				return err
			}
			value := strings.TrimSpace(secrets[definition.ID])
			if value == "" {
				if err := sm.authority.Delete(identity, field); err != nil {
					return fmt.Errorf("delete desktop credential %s: %w", definition.ID, err)
				}
				continue
			}
			if err := sm.authority.Put(identity, field, value); err != nil {
				return fmt.Errorf("store desktop credential %s: %w", definition.ID, err)
			}
		}
		sm.Set(secrets)
		return nil
	}
	sm.Set(secrets)
	return nil
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

// Validate checks secrets against manifest requirements (required, format).
// Returns nil if all validations pass, or an error with details.
func (sm *Manager) Validate(secrets map[string]string) error {
	errs := sm.validator.Validate(secrets)
	if len(errs) == 0 {
		return nil
	}
	return errs
}

// GenerateMissing generates values for per_install_generated secrets that don't exist.
func (sm *Manager) GenerateMissing(existingSecrets map[string]string) (map[string]string, error) {
	return sm.generator.GenerateSecrets(sm.manifest, existingSecrets)
}

// Ensure Manager implements Store.
var _ Store = (*Manager)(nil)

func desktopIdentity(m *manifest.Manifest) (credentialauthority.Identity, error) {
	name := "desktop-app"
	if m != nil {
		name = m.App.Name
	}
	name = strutil.SanitizeAppName(name)
	return credentialauthority.ParseIdentity("vrooli/desktop/" + name)
}
