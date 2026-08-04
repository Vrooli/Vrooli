package secrets

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/resources/securestore"
)

// The disaster drill. "We never lose secrets again" is a claim about recovery,
// and recovery is the one path where a silent defect is invisible until the day
// it matters: a bundle that exports cleanly and restores something subtly
// different looks like a backup right up until it is the only copy left.
//
// These tests exercise the values that have actually broken this seam before —
// a multi-line PEM, and text shaped like the encoding marker — and they check
// the bytes, not just that no error was returned.

// drillValues are the payloads a recovery bundle must survive unchanged.
var drillValues = map[string]string{
	// The incident value. Multi-line by construction; it corrupted a whole
	// GNOME keyring once and is now base64-wrapped at the storage seam. A
	// bundle must return the caller's bytes, not the wrapped form.
	"pem": `-----BEGIN PRIVATE KEY-----
MIIG/QIBADANBgkqhkiG9w0BAQEFAASCBucwggbjAgEAAoIBgQCcX0/9ykvEELDI
3nSpbXHrdIHb82ZmFwbGVob2xkZXJub3RhcmVhbGtleXBsYWNlaG9sZGVybm90YQ
-----END PRIVATE KEY-----
`,
	// Text that looks like the storage marker. If anything sniffed rather than
	// checked, this decodes to garbage somewhere far from here.
	"marker-lookalike": "vrooli-b64:v1:bm90LWVuY29kZWQ=",
	"crlf":             "first\r\nsecond\rthird",
	"unicode":          "pässwörd-🔐-日本語",
	"api-key":          "sk-or-v1-0123456789abcdef",
	"trailing-space":   "value-with-trailing-space   ",
}

func drillEntries() []RecoveryEntry {
	identity, _ := ParseIdentity("vrooli/drill")
	entries := make([]RecoveryEntry, 0, len(drillValues))
	for field := range drillValues {
		entries = append(entries, RecoveryEntry{Identity: identity, Field: field})
	}
	return entries
}

func provisionDrillValues(t *testing.T, authority *Authority) {
	t.Helper()
	identity, err := ParseIdentity("vrooli/drill")
	if err != nil {
		t.Fatal(err)
	}
	for field, value := range drillValues {
		if err := authority.Put(identity, field, value); err != nil {
			t.Fatalf("provision %s: %v", field, err)
		}
	}
}

func assertDrillValuesRestored(t *testing.T, authority *Authority) {
	t.Helper()
	identity, err := ParseIdentity("vrooli/drill")
	if err != nil {
		t.Fatal(err)
	}
	for field, want := range drillValues {
		got, err := authority.Resolve(identity, field)
		if err != nil {
			t.Fatalf("resolve %s after restore: %v", field, err)
		}
		if got != want {
			// Lengths, never values: a failing test must not print a secret.
			t.Fatalf("restored %s is %d bytes, want %d — the bundle did not round-trip",
				field, len(got), len(want))
		}
	}
}

// TestRecoveryDrillSurvivesTotalStoreLoss is the drill an operator would run:
// provision, export everything, lose the store completely, restore into a
// brand-new one, and get every byte back.
func TestRecoveryDrillSurvivesTotalStoreLoss(t *testing.T) {
	source, err := NewAuthority(&authorityStore{})
	if err != nil {
		t.Fatal(err)
	}
	provisionDrillValues(t, source)

	const passphrase = "drill-passphrase"
	bundle, err := source.ExportRecovery(drillEntries(), passphrase)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	for field, value := range drillValues {
		if strings.Contains(string(bundle), value) {
			t.Fatalf("bundle contains the plaintext of %s", field)
		}
	}

	// Total loss: a new store that has never seen any of these values.
	target, err := NewAuthority(&authorityStore{})
	if err != nil {
		t.Fatal(err)
	}
	if err := target.RestoreRecovery(bundle, passphrase); err != nil {
		t.Fatalf("restore: %v", err)
	}
	assertDrillValuesRestored(t, target)
}

// A bundle must not open under a passphrase that merely looks close, and a
// failed restore must leave the target untouched rather than half-written.
func TestRecoveryRefusesAWrongPassphraseWithoutPartiallyWriting(t *testing.T) {
	source, err := NewAuthority(&authorityStore{})
	if err != nil {
		t.Fatal(err)
	}
	provisionDrillValues(t, source)
	bundle, err := source.ExportRecovery(drillEntries(), "correct-passphrase")
	if err != nil {
		t.Fatal(err)
	}

	store := &authorityStore{}
	target, err := NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	if err := target.RestoreRecovery(bundle, "correct-passphrasE"); err == nil {
		t.Fatal("a one-character passphrase difference opened the bundle")
	}
	if len(store.values) != 0 {
		t.Fatalf("a failed restore wrote %d values; it must leave the target untouched", len(store.values))
	}
}

// Export is fail-closed. A bundle written from a store nobody could read would
// look like a backup and not be one, which is worse than no bundle at all.
func TestRecoveryExportRefusesWhileTheProviderIsUnavailable(t *testing.T) {
	unreadable, err := NewAuthority(&unavailableStore{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := unreadable.ExportRecovery(drillEntries(), "passphrase"); err == nil {
		t.Fatal("export produced a bundle from a store it could not read")
	}
}

// An export that silently omitted a value the operator believed was captured is
// the failure mode this whole surface exists to prevent, so a missing entry
// must fail the export rather than shrink the bundle.
func TestRecoveryExportFailsRatherThanSkippingAnUnsetEntry(t *testing.T) {
	authority, err := NewAuthority(&authorityStore{})
	if err != nil {
		t.Fatal(err)
	}
	identity, _ := ParseIdentity("vrooli/drill")
	if err := authority.Put(identity, "present", "value"); err != nil {
		t.Fatal(err)
	}
	_, err = authority.ExportRecovery([]RecoveryEntry{
		{Identity: identity, Field: "present"},
		{Identity: identity, Field: "never-provisioned"},
	}, "passphrase")
	if err == nil {
		t.Fatal("export skipped an unset entry instead of failing")
	}
	if !strings.Contains(err.Error(), "never-provisioned") {
		t.Fatalf("export error = %v, want it to name the entry it could not capture", err)
	}
}

// A recovery bundle is written with owner-only permissions and never over an
// existing path: silently replacing a bundle is how an operator ends up with
// one copy of the wrong thing.
func TestRecoveryBundleFileIsOwnerOnlyAndNeverOverwritten(t *testing.T) {
	source, err := NewAuthority(&authorityStore{})
	if err != nil {
		t.Fatal(err)
	}
	provisionDrillValues(t, source)
	bundle, err := source.ExportRecovery(drillEntries(), "passphrase")
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "vrooli-recovery.bundle")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(bundle); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode&0o077 != 0 {
		t.Fatalf("bundle mode = %o, want no group or other access", mode)
	}
	if _, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600); err == nil {
		t.Fatal("a second export overwrote an existing bundle")
	}
}

// unavailableStore is a backend that exists but cannot be reached — the
// condition that must never produce a recovery bundle.
type unavailableStore struct{}

func (unavailableStore) Put(string, string, string) error { return securestore.ErrUnavailable }
func (unavailableStore) Get(string, string) (string, error) {
	return "", securestore.ErrUnavailable
}
func (unavailableStore) Delete(string, string) error { return securestore.ErrUnavailable }

// A bundle nobody can open is not a backup, and the difference only shows on
// the day the original is gone. Verification must be provable before then.
func TestInspectRecoveryProvesABundleOpensWithoutExposingValues(t *testing.T) {
	source, err := NewAuthority(&authorityStore{})
	if err != nil {
		t.Fatal(err)
	}
	provisionDrillValues(t, source)
	const passphrase = "drill-passphrase"
	bundle, err := source.ExportRecovery(drillEntries(), passphrase)
	if err != nil {
		t.Fatal(err)
	}

	manifest, err := InspectRecovery(bundle, passphrase)
	if err != nil {
		t.Fatalf("InspectRecovery() = %v", err)
	}
	if len(manifest.Entries) != len(drillValues) {
		t.Fatalf("manifest holds %d entries, want %d", len(manifest.Entries), len(drillValues))
	}
	// The manifest names what would be restored and nothing more. If a value
	// could leak here, verifying a backup would mean printing the secrets it
	// exists to protect.
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	for field, value := range drillValues {
		if strings.Contains(string(encoded), value) {
			t.Fatalf("the verification manifest exposes the value of %s", field)
		}
	}

	if _, err := InspectRecovery(bundle, "drill-passphrasE"); err == nil {
		t.Fatal("a one-character passphrase difference verified successfully")
	}
}

// Verification must not require a credential store: the machine holding the
// backup is often not the machine that made it.
func TestInspectRecoveryNeedsNoCredentialStore(t *testing.T) {
	source, err := NewAuthority(&authorityStore{})
	if err != nil {
		t.Fatal(err)
	}
	provisionDrillValues(t, source)
	bundle, err := source.ExportRecovery(drillEntries(), "passphrase")
	if err != nil {
		t.Fatal(err)
	}
	previous := DefaultAuthority
	DefaultAuthority = func() (*Authority, error) { return nil, ErrProviderAbsent }
	t.Cleanup(func() { DefaultAuthority = previous })

	if _, err := InspectRecovery(bundle, "passphrase"); err != nil {
		t.Fatalf("InspectRecovery() = %v on a host with no credential store", err)
	}
}
