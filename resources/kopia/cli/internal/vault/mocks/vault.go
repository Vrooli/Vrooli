// Package mocks provides the in-memory test double for the vault.Vault seam.
package mocks

import (
	"context"
	"strings"
	"sync"

	"resource-kopia/cli/internal/vault"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	kopiaregistry "github.com/vrooli/vrooli/packages/kopiaregistry-go"
)

// FakeVault is an in-memory Vault for unit tests. It records puts so tests can
// assert that a generated passphrase was stored (and never an empty one).
type FakeVault struct {
	mu          sync.Mutex
	store       map[string]string
	credentials map[string]string
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
	return &FakeVault{store: map[string]string{}, credentials: map[string]string{}}
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

// SeedPassphrase pre-populates the authority side of the fake for a repository.
func (f *FakeVault) SeedPassphrase(repo, value string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.credentials == nil {
		f.credentials = map[string]string{}
	}
	identity, err := kopiaregistry.PassphraseIdentity(repo)
	if err != nil {
		return
	}
	f.credentials[credentialKey(identity, kopiaregistry.PassphraseField)] = value
}

var _ interface {
	Put(credentialauthority.Identity, string, string) error
	Resolve(credentialauthority.Identity, string) (string, error)
	Delete(credentialauthority.Identity, string) error
} = (*FakeVault)(nil)

func credentialKey(identity credentialauthority.Identity, field string) string {
	return string(identity) + ":" + field
}

func (f *FakeVault) Put(identity credentialauthority.Identity, field, value string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.PutErr != nil {
		return f.PutErr
	}
	if f.credentials == nil {
		f.credentials = map[string]string{}
	}
	f.credentials[credentialKey(identity, field)] = value
	return nil
}

func (f *FakeVault) Resolve(identity credentialauthority.Identity, field string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.GetErr != nil {
		return "", f.GetErr
	}
	value, ok := f.credentials[credentialKey(identity, field)]
	if !ok {
		return "", credentialauthority.ErrUnconfigured
	}
	return value, nil
}

func (f *FakeVault) Delete(identity credentialauthority.Identity, field string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.DeleteErr != nil {
		return f.DeleteErr
	}
	delete(f.credentials, credentialKey(identity, field))
	return nil
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
