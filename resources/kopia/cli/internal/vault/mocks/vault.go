// Package mocks provides the in-memory test double for the vault.Vault seam.
package mocks

import (
	"context"
	"resource-kopia/cli/internal/vault"
	"strings"
	"sync"
)

// FakeVault is an in-memory Vault for unit tests. It records puts so tests can
// assert that a generated passphrase was stored (and never an empty one).
type FakeVault struct {
	mu    sync.Mutex
	store map[string]string
	// GetErr / PutErr, when set, are returned to simulate vault being down.
	GetErr    error
	PutErr    error
	DeleteErr error
	// Gets / Puts record access for assertions.
	Gets    []string
	Puts    []string
	Deletes []string
}

var _ vault.Vault = (*FakeVault)(nil)

// NewFakeVault returns an empty FakeVault.
func NewFakeVault() *FakeVault {
	return &FakeVault{store: map[string]string{}}
}

func key(path, k string) string { return path + "::" + k }

// Seed pre-populates a secret (e.g. an existing passphrase or S3 creds).
func (f *FakeVault) Seed(path, k, value string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.store == nil {
		f.store = map[string]string{}
	}
	f.store[key(path, k)] = value
}

// GetSecret implements vault.Vault.
func (f *FakeVault) GetSecret(_ context.Context, path, k string) (string, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Gets = append(f.Gets, key(path, k))
	if f.GetErr != nil {
		return "", false, f.GetErr
	}
	v, ok := f.store[key(path, k)]
	return v, ok, nil
}

// PutSecret implements vault.Vault.
func (f *FakeVault) PutSecret(_ context.Context, path, k, value string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.PutErr != nil {
		return f.PutErr
	}
	if f.store == nil {
		f.store = map[string]string{}
	}
	f.store[key(path, k)] = value
	f.Puts = append(f.Puts, key(path, k))
	return nil
}

// DeleteSecret implements vault.Vault.
func (f *FakeVault) DeleteSecret(_ context.Context, path string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.DeleteErr != nil {
		return f.DeleteErr
	}
	for k := range f.store {
		if strings.HasPrefix(k, path+"::") {
			delete(f.store, k)
		}
	}
	f.Deletes = append(f.Deletes, path)
	return nil
}

// Value returns a stored secret for assertions.
func (f *FakeVault) Value(path, k string) (string, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.store[key(path, k)]
	return v, ok
}
