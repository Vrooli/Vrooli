// Package securestore is the resource-host-private seam for operating-system
// credential storage. It intentionally has no dependency on api-core or
// cli-core, and broker state must never implement this interface.
package securestore

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
)

var ErrUnavailable = errors.New("operating-system secure storage is unavailable")

// Store persists bootstrap material in the platform credential facility. The
// value is never written to resource state, diagnostics, or application
// environment variables.
type Store interface {
	Put(service, key, value string) error
	Get(service, key string) (string, error)
	Delete(service, key string) error
}

type unavailableStore struct{ reason string }

func (s unavailableStore) Put(string, string, string) error {
	return fmt.Errorf("%w: %s", ErrUnavailable, s.reason)
}
func (s unavailableStore) Get(string, string) (string, error) {
	return "", fmt.Errorf("%w: %s", ErrUnavailable, s.reason)
}
func (s unavailableStore) Delete(string, string) error {
	return fmt.Errorf("%w: %s", ErrUnavailable, s.reason)
}

// Unavailable returns a fail-closed store for a target where the OS credential
// facility cannot be verified. Callers must surface this as a conditional
// target limitation; they must not replace it with a 0600 plaintext file.
func Unavailable(reason string) Store { return unavailableStore{reason: reason} }

// Probe proves that a credential backend can store, retrieve, and remove a
// throwaway value. Command discovery alone is not sufficient evidence that a
// Linux Secret Service session is available. Probe never reports the value it
// writes and always attempts cleanup before returning.
func Probe(store Store) error {
	if store == nil {
		return fmt.Errorf("%w: credential store is not configured", ErrUnavailable)
	}
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Errorf("generate secure-store probe key: %w", err)
	}
	key := "probe-" + hex.EncodeToString(bytes)
	const service = "vrooli.securestore.probe"
	const value = "ready"
	if err := store.Put(service, key, value); err != nil {
		return fmt.Errorf("%w: write probe: %v", ErrUnavailable, err)
	}
	defer store.Delete(service, key)
	got, err := store.Get(service, key)
	if err != nil {
		return fmt.Errorf("%w: read probe: %v", ErrUnavailable, err)
	}
	if got != value {
		return fmt.Errorf("%w: probe readback did not match", ErrUnavailable)
	}
	if err := store.Delete(service, key); err != nil {
		return fmt.Errorf("%w: delete probe: %v", ErrUnavailable, err)
	}
	return nil
}
