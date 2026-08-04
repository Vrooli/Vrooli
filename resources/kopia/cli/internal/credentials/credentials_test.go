package credentials

import (
	"errors"
	"strings"
	"testing"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	kopiaregistry "github.com/vrooli/vrooli/packages/kopiaregistry-go"
)

type fakeStore struct {
	value  string
	putErr error
}

func (f *fakeStore) Put(credentialauthority.Identity, string, string) error {
	if f.putErr != nil {
		return f.putErr
	}
	return nil
}

func (f *fakeStore) Resolve(credentialauthority.Identity, string) (string, error) {
	if f.value == "" {
		return "", credentialauthority.ErrUnconfigured
	}
	return f.value, nil
}
func (f *fakeStore) Delete(credentialauthority.Identity, string) error { return nil }

func TestGeneratePassphraseStrength(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		value, err := GeneratePassphrase()
		if err != nil || strings.TrimSpace(value) == "" || len(value) < 32 {
			t.Fatalf("GeneratePassphrase() = %q, %v", value, err)
		}
		if seen[value] {
			t.Fatalf("passphrase collision: %q", value)
		}
		seen[value] = true
	}
}

func TestValidateStoredPassphraseRequiresMatchingReadback(t *testing.T) {
	identity, err := kopiaregistry.PassphraseIdentity("nightly")
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateStoredPassphrase(&fakeStore{value: "passphrase"}, identity, "passphrase"); err != nil {
		t.Fatal(err)
	}
	for _, store := range []Store{&fakeStore{value: "other"}, &fakeStore{putErr: errors.New("locked")}} {
		if err := ValidateStoredPassphrase(store, identity, "passphrase"); err == nil {
			t.Fatal("ValidateStoredPassphrase accepted a failed readback")
		}
	}
}
