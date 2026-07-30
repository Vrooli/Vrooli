package secrets

import (
	"bytes"
	"testing"
)

func TestRecoveryRoundTripDoesNotExposeValueInBundleMetadata(t *testing.T) {
	source, err := NewAuthority(&authorityStore{})
	if err != nil {
		t.Fatal(err)
	}
	id, _ := ParseIdentity("vrooli/openrouter")
	const value = "recovery-fixture-value"
	if err := source.Put(id, "api-key", value); err != nil {
		t.Fatal(err)
	}
	bundle, err := source.ExportRecovery([]RecoveryEntry{{Identity: id, Field: "api-key"}}, "passphrase")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(bundle, []byte(value)) {
		t.Fatal("recovery bundle exposes plaintext value")
	}
	target, err := NewAuthority(&authorityStore{})
	if err != nil {
		t.Fatal(err)
	}
	if err := target.RestoreRecovery(bundle, "passphrase"); err != nil {
		t.Fatal(err)
	}
	got, err := target.read(id, "api-key")
	if err != nil || got != value {
		t.Fatalf("restore = %q, %v", got, err)
	}
	if err := target.RestoreRecovery(bundle, "wrong"); err == nil {
		t.Fatal("wrong passphrase restored bundle")
	}
}
