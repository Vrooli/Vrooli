package credentialauthority

import (
	"context"
	"path/filepath"
	"testing"
)

// A credential issued after the source was built is found by the next read, and
// only at its own address.
func TestServiceCredentialFindsACredentialIssuedAfterItWasBuilt(t *testing.T) {
	authority, err := OpenEncryptedFile(filepath.Join(t.TempDir(), "authority.json"), "test-passphrase")
	if err != nil {
		t.Fatal(err)
	}
	open := func() (*Authority, error) { return authority, nil }
	const address = "scenario-authenticator/service-principal/storage-manager-ask-reader/notification-hub-ask-reader"
	read := ServiceCredentialFrom(open, address, "credential")
	ctx := context.Background()

	if value, err := read(ctx); err == nil || value != "" {
		t.Fatalf("before issuance: value=%q err=%v, want an error", value, err)
	}
	identity, err := ParseIdentity(address)
	if err != nil {
		t.Fatal(err)
	}
	if err := authority.Put(identity, "credential", "issued-secret"); err != nil {
		t.Fatal(err)
	}
	if value, err := read(ctx); err != nil || value != "issued-secret" {
		t.Fatalf("after issuance: value=%q err=%v", value, err)
	}
	other := ServiceCredentialFrom(open, "scenario-authenticator/service-principal/agent-manager-ask-reader/notification-hub-ask-reader", "credential")
	if _, err := other(ctx); err == nil {
		t.Fatal("another principal's credential must not be found")
	}
	if _, err := ServiceCredentialFrom(open, "not an address", "credential")(ctx); err == nil {
		t.Fatal("a malformed address is an error")
	}
}
