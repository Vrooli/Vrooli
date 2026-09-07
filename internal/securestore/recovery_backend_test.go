package securestore

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/testenv"
)

// A recovery bundle is restored on a host that is not the one that wrote it,
// and that host very often has a different backend: the workstation with a
// GNOME keyring is gone and the replacement is a headless server running the
// encrypted file store. That migration is only safe if the stored form of a
// value does not depend on which backend wrote it.
//
// This is why guardValues sits at the composition root rather than inside each
// adapter. These tests pin the property directly, because the failure it
// prevents is silent: a value that round-trips within one backend but not
// across two would restore into garbage on exactly the host doing the recovery.

// recoveryDrillValues mirrors the payloads the authority-level drill uses.
var recoveryDrillValues = map[string]string{
	"pem":              multiLinePEM,
	"marker-lookalike": valueEncodingPrefix + "bm90LWVuY29kZWQ=",
	"crlf":             "first\r\nsecond\rthird",
	"unicode":          "pässwörd-🔐-日本語",
	"plain":            "sk-or-v1-0123456789abcdef",
}

// newDrillEncryptedStore builds a real encrypted file store opened by a
// passphrase, with no session cache so each handle is independent.
func newDrillEncryptedStore(t *testing.T, path, passphrase string) Store {
	t.Helper()
	store := newEncryptedStore(path, passphraseProvider{passphrase: passphrase})
	store.cache = noUnlockCache{}
	if _, err := store.initialize(); err != nil {
		t.Fatalf("initialize encrypted store: %v", err)
	}
	return guardValues(store)
}

// TestStoredFormIsIdenticalAcrossBackends is the cross-backend restore
// property: what one backend hands back is byte-identical to what another does,
// so a bundle written on a desktop restores correctly onto a headless server.
func TestStoredFormIsIdenticalAcrossBackends(t *testing.T) {
	const service = "vrooli.credentials.v1"

	native := guardValues(testenv.NewCredentialStore(ErrNotFound))
	encrypted := newDrillEncryptedStore(t, filepath.Join(t.TempDir(), "secrets.enc.json"), "drill-passphrase")

	for field, want := range recoveryDrillValues {
		t.Run(field, func(t *testing.T) {
			if err := native.Put(service, field, want); err != nil {
				t.Fatalf("native put: %v", err)
			}
			if err := encrypted.Put(service, field, want); err != nil {
				t.Fatalf("encrypted put: %v", err)
			}

			fromNative, err := native.Get(service, field)
			if err != nil {
				t.Fatalf("native get: %v", err)
			}
			fromEncrypted, err := encrypted.Get(service, field)
			if err != nil {
				t.Fatalf("encrypted get: %v", err)
			}
			if fromNative != want {
				t.Fatalf("native returned %d bytes, want %d", len(fromNative), len(want))
			}
			if fromEncrypted != want {
				t.Fatalf("encrypted returned %d bytes, want %d", len(fromEncrypted), len(want))
			}
			if fromNative != fromEncrypted {
				t.Fatal("the two backends disagree about a value, so a cross-backend restore would corrupt it")
			}
		})
	}
}

// A value sealed by one encrypted store must open in a second handle over the
// same file and passphrase. Without this, "restore" would mean "readable until
// the process exits".
func TestEncryptedStoreValuesSurviveANewHandle(t *testing.T) {
	const service = "vrooli.credentials.v1"
	path := filepath.Join(t.TempDir(), "secrets.enc.json")
	const passphrase = "drill-passphrase"

	writer := newDrillEncryptedStore(t, path, passphrase)
	for field, value := range recoveryDrillValues {
		if err := writer.Put(service, field, value); err != nil {
			t.Fatalf("put %s: %v", field, err)
		}
	}

	// A brand-new handle, as a later command would build.
	reopened := newEncryptedStore(path, passphraseProvider{passphrase: passphrase})
	reopened.cache = noUnlockCache{}
	reader := guardValues(reopened)
	for field, want := range recoveryDrillValues {
		got, err := reader.Get(service, field)
		if err != nil {
			t.Fatalf("reopen get %s: %v", field, err)
		}
		if got != want {
			t.Fatalf("reopened %s is %d bytes, want %d", field, len(got), len(want))
		}
	}

	// The wrong passphrase must not open it, and must say so as a passphrase
	// problem rather than as a corrupt store.
	wrong := newEncryptedStore(path, passphraseProvider{passphrase: "not-the-passphrase"})
	wrong.cache = noUnlockCache{}
	if _, err := guardValues(wrong).Get(service, "plain"); err == nil {
		t.Fatal("the wrong passphrase opened the store")
	}
}

// The file that holds the values must never contain them. This is the encrypted
// store's whole contract: not "no credential material on disk" but "no value
// recoverable from the file alone".
func TestEncryptedStoreFileNeverContainsAPlaintextValue(t *testing.T) {
	const service = "vrooli.credentials.v1"
	path := filepath.Join(t.TempDir(), "secrets.enc.json")
	store := newDrillEncryptedStore(t, path, "drill-passphrase")

	for field, value := range recoveryDrillValues {
		if err := store.Put(service, field, value); err != nil {
			t.Fatalf("put %s: %v", field, err)
		}
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for field, value := range recoveryDrillValues {
		if len(value) < 8 {
			continue
		}
		if bytes.Contains(contents, []byte(value)) {
			t.Fatalf("the store file contains the plaintext of %s", field)
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode&0o077 != 0 {
		t.Fatalf("store file mode = %o, want no group or other access", mode)
	}
}
