// Package mocks provides an in-memory credential-authority store for
// resource-kopia tests. It intentionally has no external secret-service seam.
package mocks

import (
	"sync"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	kopiaregistry "github.com/vrooli/vrooli/packages/kopiaregistry-go"

	"github.com/vrooli/vrooli/resources/kopia/cli/internal/credentials"
)

// FakeStore is an in-memory credential authority. It records failures so
// callers can prove that provider errors remain distinct from missing values.
type FakeStore struct {
	mu        sync.Mutex
	values    map[string]string
	GetErr    error
	PutErr    error
	DeleteErr error
}

var _ credentials.Store = (*FakeStore)(nil)

func NewFakeStore() *FakeStore {
	return &FakeStore{values: map[string]string{}}
}

func key(identity credentialauthority.Identity, field string) string {
	return string(identity) + ":" + field
}

func (f *FakeStore) Seed(identity credentialauthority.Identity, field, value string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[key(identity, field)] = value
}

func (f *FakeStore) SeedPassphrase(repo, value string) {
	identity, err := kopiaregistry.PassphraseIdentity(repo)
	if err != nil {
		return
	}
	f.Seed(identity, kopiaregistry.PassphraseField, value)
}

func (f *FakeStore) SeedS3(repo string, creds credentials.S3Credentials) {
	identity, err := kopiaregistry.PassphraseIdentity(repo)
	if err != nil {
		return
	}
	f.Seed(identity, credentials.S3AccessKeyIDField, creds.AccessKeyID)
	f.Seed(identity, credentials.S3SecretAccessKeyField, creds.SecretAccessKey)
}

func (f *FakeStore) Put(identity credentialauthority.Identity, field, value string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.PutErr != nil {
		return f.PutErr
	}
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[key(identity, field)] = value
	return nil
}

func (f *FakeStore) Resolve(identity credentialauthority.Identity, field string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.GetErr != nil {
		return "", f.GetErr
	}
	value, ok := f.values[key(identity, field)]
	if !ok {
		return "", credentialauthority.ErrUnconfigured
	}
	return value, nil
}

func (f *FakeStore) Delete(identity credentialauthority.Identity, field string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.DeleteErr != nil {
		return f.DeleteErr
	}
	delete(f.values, key(identity, field))
	return nil
}
