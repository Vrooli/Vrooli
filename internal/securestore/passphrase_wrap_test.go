package securestore

import (
	"errors"
	"path/filepath"
	"testing"
)

// newUnattendedOnlyStore is the store `vrooli setup` leaves on a TPM host: it
// opens through the host-bound wrap alone and has no passphrase wrap, so
// losing the TPM binding would lose every value in it.
func newUnattendedOnlyStore(t *testing.T) (string, hostBoundProvider) {
	t.Helper()
	fake := newFakeSystemdCreds(hostBoundTPM2Mode)
	unattended := hostBoundProvider{run: fake.run}
	useTestUnattendedProviders(t, unattended)
	path := filepath.Join(t.TempDir(), "credentials.enc.json")
	store := newEncryptedStore(path, unattended)
	if _, err := store.initialize(); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if err := store.Put("svc", "key", "value-sealed-by-the-tpm-only"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	return path, unattended
}

func passphraseOnly(path, passphrase string) *encryptedStore {
	store := newEncryptedStore(path, passphraseProvider{passphrase: passphrase})
	store.cache = noUnlockCache{}
	return store
}

// The escrowed passphrase has to open the store on its own, without the TPM:
// that is the recovery it exists for.
func TestAddPassphraseWrapMakesATPMOnlyStoreRecoverableByPassphrase(t *testing.T) {
	path, unattended := newUnattendedOnlyStore(t)

	check, err := addPassphraseWrap(newEncryptedStore(path, unattended), "escrowed on the control plane")
	if err != nil {
		t.Fatalf("addPassphraseWrap: %v", err)
	}
	if !check.Valid || check.Generation != 1 {
		t.Fatalf("check = %+v, want a valid first-generation wrap", check)
	}
	value, err := passphraseOnly(path, "escrowed on the control plane").Get("svc", "key")
	if err != nil {
		t.Fatalf("the added passphrase wrap does not open the store: %v", err)
	}
	if value != "value-sealed-by-the-tpm-only" {
		t.Fatalf("value = %q; adding a wrap must not re-encrypt or lose anything", value)
	}
	if _, err := newEncryptedStore(path, unattended).Get("svc", "key"); err != nil {
		t.Fatalf("the unattended wrap stopped opening the store: %v", err)
	}
}

// A second add would silently replace a passphrase someone may already hold.
func TestAddPassphraseWrapRefusesAStoreThatAlreadyHasOne(t *testing.T) {
	path, unattended := newUnattendedOnlyStore(t)
	if _, err := addPassphraseWrap(newEncryptedStore(path, unattended), "first"); err != nil {
		t.Fatalf("addPassphraseWrap: %v", err)
	}
	if _, err := addPassphraseWrap(newEncryptedStore(path, unattended), "second"); !errors.Is(err, ErrPassphraseWrapExists) {
		t.Fatalf("second add error = %v, want ErrPassphraseWrapExists", err)
	}
	if _, err := passphraseOnly(path, "first").Get("svc", "key"); err != nil {
		t.Fatalf("the refused add changed the existing passphrase wrap: %v", err)
	}
}

// A working TPM wrap must not make a wrong passphrase verify: the control
// plane decides which escrowed passphrase a node holds from this answer.
func TestVerifyPassphraseChecksOnlyThePassphraseWrap(t *testing.T) {
	path, unattended := newUnattendedOnlyStore(t)
	if _, err := verifyPassphrase(newEncryptedStore(path, unattended), "anything"); !errors.Is(err, ErrNoPassphraseWrap) {
		t.Fatalf("verify before a passphrase wrap exists: err = %v, want ErrNoPassphraseWrap", err)
	}
	if _, err := addPassphraseWrap(newEncryptedStore(path, unattended), "the escrowed one"); err != nil {
		t.Fatalf("addPassphraseWrap: %v", err)
	}

	right, err := verifyPassphrase(newEncryptedStore(path, unattended), "the escrowed one")
	if err != nil || !right.Valid {
		t.Fatalf("right passphrase: check = %+v, err = %v", right, err)
	}
	wrong, err := verifyPassphrase(newEncryptedStore(path, unattended), "a stale one")
	if err != nil {
		t.Fatalf("a wrong passphrase is an answer, not an error: %v", err)
	}
	if wrong.Valid {
		t.Fatalf("a wrong passphrase verified through the TPM wrap")
	}
}

// Rotation replaces the wrap and bumps its generation; the old passphrase
// stops verifying and the new one starts.
func TestVerifyPassphraseFollowsAChangedPassphrase(t *testing.T) {
	path, unattended := newUnattendedOnlyStore(t)
	if _, err := addPassphraseWrap(newEncryptedStore(path, unattended), "old"); err != nil {
		t.Fatalf("addPassphraseWrap: %v", err)
	}
	if err := changePassphraseStore(newEncryptedStore(path, unattended), "old", "new"); err != nil {
		t.Fatalf("changePassphraseStore: %v", err)
	}
	t.Cleanup(func() { SetPassphrase("") })
	old, err := verifyPassphrase(newEncryptedStore(path, unattended), "old")
	if err != nil || old.Valid {
		t.Fatalf("old passphrase after rotation: check = %+v, err = %v", old, err)
	}
	next, err := verifyPassphrase(newEncryptedStore(path, unattended), "new")
	if err != nil || !next.Valid || next.Generation != 2 {
		t.Fatalf("new passphrase after rotation: check = %+v, err = %v", next, err)
	}
}
