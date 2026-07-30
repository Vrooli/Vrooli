// Package secrets owns the credential authority used by the control plane.
// It deliberately exposes values only to the runtime injection boundary.
package secrets

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/internal/resources/securestore"
)

const credentialService = "vrooli.credentials.v1"

var (
	ErrUnsupportedProvider = errors.New("credential provider is unsupported")
	ErrUnconfigured        = errors.New("credential is not configured")
)

// Identity is backend-neutral and stable across desktop, local, and hosted
// deployment tiers. It must never contain a Vault path or an environment name.
type Identity string

func ParseIdentity(raw string) (Identity, error) {
	value := strings.Trim(strings.ToLower(strings.TrimSpace(raw)), "/")
	if value == "" || strings.Contains(value, "//") {
		return "", fmt.Errorf("credential logical identity is required and must be namespaced")
	}
	parts := strings.Split(value, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("credential logical identity must be namespaced")
	}
	for _, part := range parts {
		if part == "" {
			return "", fmt.Errorf("credential logical identity contains an empty segment")
		}
		for _, r := range part {
			if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '-' && r != '_' && r != '.' {
				return "", fmt.Errorf("credential logical identity contains an invalid character")
			}
		}
	}
	return Identity(value), nil
}

type Status struct {
	Identity   Identity  `json:"identity"`
	Field      string    `json:"field"`
	Configured bool      `json:"configured"`
	Provider   string    `json:"provider"`
	CheckedAt  time.Time `json:"checked_at"`
}

// Authority is the only durable local credential writer. Its Store must be a
// probed native secure store; plaintext and process-environment fallbacks are
// intentionally absent.
type Authority struct {
	store securestore.Store
	mu    sync.Mutex
}

func NewAuthority(store securestore.Store) (*Authority, error) {
	if store == nil {
		return nil, fmt.Errorf("%w: no native credential store", ErrUnsupportedProvider)
	}
	if err := securestore.Probe(store); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnsupportedProvider, err)
	}
	return &Authority{store: store}, nil
}

func (a *Authority) Put(identity Identity, field, value string) error {
	if a == nil || a.store == nil {
		return ErrUnsupportedProvider
	}
	if _, err := ParseIdentity(string(identity)); err != nil {
		return err
	}
	field = strings.TrimSpace(field)
	if field == "" || strings.ContainsAny(field, "/\\") {
		return fmt.Errorf("credential field is required and cannot contain a path separator")
	}
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("credential value is required")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.store.Put(credentialService, string(identity)+":"+field, value)
}

// Inject resolves values only into the supplied ephemeral environment map.
// The caller owns process creation; this method never exports globally.
func (a *Authority) Inject(identity Identity, field, env string, target map[string]string) error {
	if a == nil || a.store == nil {
		return ErrUnsupportedProvider
	}
	if _, err := ParseIdentity(string(identity)); err != nil {
		return err
	}
	field = strings.TrimSpace(field)
	env = strings.TrimSpace(env)
	if field == "" || env == "" || target == nil {
		return fmt.Errorf("credential field, environment name, and target are required")
	}
	a.mu.Lock()
	value, err := a.store.Get(credentialService, string(identity)+":"+field)
	a.mu.Unlock()
	if err != nil || strings.TrimSpace(value) == "" {
		return ErrUnconfigured
	}
	target[env] = value
	return nil
}

func (a *Authority) Status(identity Identity, field string) Status {
	status := Status{Identity: identity, Field: strings.TrimSpace(field), Provider: "native-secure-store", CheckedAt: time.Now().UTC()}
	if a == nil || a.store == nil || status.Field == "" {
		return status
	}
	a.mu.Lock()
	value, err := a.store.Get(credentialService, string(identity)+":"+status.Field)
	a.mu.Unlock()
	status.Configured = err == nil && strings.TrimSpace(value) != ""
	return status
}

func (a *Authority) Delete(identity Identity, field string) error {
	if a == nil || a.store == nil {
		return ErrUnsupportedProvider
	}
	if _, err := ParseIdentity(string(identity)); err != nil {
		return err
	}
	return a.store.Delete(credentialService, string(identity)+":"+strings.TrimSpace(field))
}
