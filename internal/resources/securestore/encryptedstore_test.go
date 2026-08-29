//nolint:goconst // test data deliberately reuses stable storage fixtures.
package securestore

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestEncryptedStore returns a factory over one initialized store file, so
// every handle the conformance suite builds talks to the same backend the way
// separate CLI invocations do.
func newTestEncryptedStore(t *testing.T, passphrase string) (func() Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "state", "credentials.enc.json")
	newStore := func() Store {
		return newEncryptedStore(path, passphraseProvider{passphrase: passphrase})
	}
	if _, err := newEncryptedStore(path, passphraseProvider{passphrase: passphrase}).initialize(); err != nil {
		t.Fatalf("initialize encrypted store: %v", err)
	}
	return newStore, path
}

// TestEncryptedStoreConformance is the gate for this adapter. It runs the same
// suite every native adapter runs, unmodified: a suite change would mean the
// abstraction leaked, not that the suite needs adjusting.
func TestEncryptedStoreConformance(t *testing.T) {
	newStore, _ := newTestEncryptedStore(t, "conformance passphrase")
	runConformance(t, newStore)
}

func TestEncryptedStoreReportsAbsentBeforeInitialization(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credentials.enc.json")
	store := newEncryptedStore(path, passphraseProvider{passphrase: "p"})

	if store.initialized() {
		t.Fatalf("an uninitialized host reported an initialized store")
	}
	_, err := store.Get("svc", "key")
	if !errors.Is(err, ErrAbsent) {
		t.Fatalf("Get before init = %v, want ErrAbsent", err)
	}
	if errors.Is(err, ErrNotFound) {
		t.Fatalf("an uninitialized store reported a missing value: %v", err)
	}
	if !strings.Contains(err.Error(), "store init") {
		t.Fatalf("the absent explanation does not name the remedy: %v", err)
	}
	if err := store.Put("svc", "key", "value"); !errors.Is(err, ErrAbsent) {
		t.Fatalf("Put before init = %v, want ErrAbsent", err)
	}
	if err := store.Delete("svc", "key"); !errors.Is(err, ErrAbsent) {
		t.Fatalf("Delete before init = %v, want ErrAbsent", err)
	}
}

func TestEncryptedStoreReportsUnavailableWhenNoWrapOpens(t *testing.T) {
	_, path := newTestEncryptedStore(t, "the right passphrase")
	store := newEncryptedStore(path, passphraseProvider{passphrase: "the wrong passphrase"})

	_, err := store.Get("svc", "key")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Get with no working wrap = %v, want ErrUnavailable", err)
	}
	if errors.Is(err, ErrAbsent) || errors.Is(err, ErrNotFound) {
		t.Fatalf("a locked store was reported as absent or empty: %v", err)
	}
	if !strings.Contains(err.Error(), "store unlock") {
		t.Fatalf("the unavailable explanation does not name the remedy: %v", err)
	}
}

// TestEncryptedStoreFailsClosedOnADamagedFile is the corruption case at the
// adapter boundary: a flipped byte must never surface as "you never set this".
func TestEncryptedStoreFailsClosedOnADamagedFile(t *testing.T) {
	newStore, path := newTestEncryptedStore(t, "passphrase")
	if err := newStore().Put("svc", "key", "the-value"); err != nil {
		t.Fatalf("Put: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read store: %v", err)
	}
	// Flip a byte inside the last entry's ciphertext field. The JSON stays
	// well-formed, so only the AEAD can catch this.
	marker := `"ciphertext": "`
	index := strings.LastIndex(string(data), marker) + len(marker)
	damaged := []byte(string(data))
	if damaged[index] == 'A' {
		damaged[index] = 'B'
	} else {
		damaged[index] = 'A'
	}
	if err := os.WriteFile(path, damaged, sealedFilePerm); err != nil {
		t.Fatalf("write damaged store: %v", err)
	}

	value, err := newStore().Get("svc", "key")
	if err == nil {
		t.Fatalf("a damaged entry read back %d bytes", len(value))
	}
	if errors.Is(err, ErrNotFound) {
		t.Fatalf("a damaged entry was reported as a missing value: %v", err)
	}
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("damaged-entry error = %v, want ErrUnavailable", err)
	}
	if strings.Contains(err.Error(), "the-value") {
		t.Fatalf("the error message leaked the credential value: %v", err)
	}
}

// TestEncryptedStoreKeepsValuesOutOfEveryReportableString is the leak check at
// this seam. Everything this adapter can be asked to print — the adapter name,
// every error it produces — is searched for the value.
func TestEncryptedStoreKeepsValuesOutOfEveryReportableString(t *testing.T) {
	const secret = "sk-live-DO-NOT-LEAK-1234567890"
	newStore, path := newTestEncryptedStore(t, "passphrase")
	store := newStore()
	if err := store.Put("svc", "key", secret); err != nil {
		t.Fatalf("Put: %v", err)
	}

	reportable := []string{AdapterName(store)}
	if _, err := store.Get("svc", "absent"); err != nil {
		reportable = append(reportable, err.Error())
	}
	if err := store.Put("", "", ""); err != nil {
		reportable = append(reportable, err.Error())
	}
	locked := newEncryptedStore(path, passphraseProvider{passphrase: "wrong"})
	if _, err := locked.Get("svc", "key"); err != nil {
		reportable = append(reportable, err.Error())
	}
	for _, text := range reportable {
		if strings.Contains(text, secret) {
			t.Fatalf("a reportable string carried the credential value: %q", text)
		}
	}

	// The stored file must not contain it either, which is the whole claim the
	// amended storage rule makes.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read store: %v", err)
	}
	if strings.Contains(string(data), secret) {
		t.Fatalf("the credential value is recoverable from the store file")
	}
}

func TestEncryptedStoreAdapterNameReportsTheActiveWrap(t *testing.T) {
	newStore, _ := newTestEncryptedStore(t, "passphrase")
	store := newStore()

	// Before anything opens the store there is nothing honest to report beyond
	// the backend, and naming a wrap would mean guessing.
	if got := AdapterName(store); got != adapterEncryptedFile {
		t.Fatalf("AdapterName before opening = %q, want %q", got, adapterEncryptedFile)
	}
	if _, err := store.Get("svc", "nothing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get: %v", err)
	}
	got := AdapterName(store)
	if !strings.Contains(got, adapterEncryptedFile) || !strings.Contains(got, providerPassphrase) || !strings.Contains(got, keyStorePassphrase) {
		t.Fatalf("AdapterName after opening = %q, want it to name the backend and the active wrap", got)
	}
	provider, keyStore := store.(*encryptedStore).ActiveWrap()
	if provider != providerPassphrase || keyStore != keyStorePassphrase {
		t.Fatalf("ActiveWrap = %q/%q", provider, keyStore)
	}
}

func TestEncryptedStoreRefusesToReplaceAnExistingStore(t *testing.T) {
	newStore, path := newTestEncryptedStore(t, "passphrase")
	if err := newStore().Put("svc", "key", "must-survive"); err != nil {
		t.Fatalf("Put: %v", err)
	}

	second := newEncryptedStore(path, passphraseProvider{passphrase: "a different passphrase"})
	if _, err := second.initialize(); err == nil {
		t.Fatalf("initialize replaced an existing store, destroying every value in it")
	}
	value, err := newStore().Get("svc", "key")
	if err != nil || value != "must-survive" {
		t.Fatalf("the existing store did not survive a second init: %q, %v", value, err)
	}
}

func TestEncryptedStoreInitializeRequiresAWorkingProvider(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credentials.enc.json")
	store := newEncryptedStore(path,
		hostBoundProvider{run: newFakeSystemdCreds().run},
		passphraseProvider{passphrase: ""},
	)
	_, err := store.initialize()
	if !errors.Is(err, ErrAbsent) {
		t.Fatalf("initialize with no usable provider = %v, want ErrAbsent", err)
	}
	if _, statErr := os.Stat(path); statErr == nil {
		t.Fatalf("a failed init left a store file behind")
	}
}

// TestEncryptedStoreAddsAWrapWithoutReEncrypting is the property that keeps a
// host which gains a TPM from becoming a migration event.
func TestEncryptedStoreAddsAWrapWithoutReEncrypting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credentials.enc.json")
	passphrase := passphraseProvider{passphrase: "the operator passphrase"}
	fake := newFakeSystemdCreds("tpm2")
	hostBound := hostBoundProvider{run: fake.run}

	store := newEncryptedStore(path, passphrase)
	if _, err := store.initialize(); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if err := store.Put("svc", "key", "written-before-the-tpm"); err != nil {
		t.Fatalf("Put: %v", err)
	}

	wrap, err := newEncryptedStore(path, passphrase).addWrap(hostBound)
	if err != nil {
		t.Fatalf("addWrap: %v", err)
	}
	if wrap.Provider != providerHostBound || wrap.KeyStore != keyStoreTPM2 {
		t.Fatalf("added wrap = %q/%q", wrap.Provider, wrap.KeyStore)
	}

	// The host-bound wrap alone now opens every value written before it existed.
	tpmOnly := newEncryptedStore(path, hostBound)
	value, err := tpmOnly.Get("svc", "key")
	if err != nil {
		t.Fatalf("Get through the new wrap: %v", err)
	}
	if value != "written-before-the-tpm" {
		t.Fatalf("value through the new wrap = %q", value)
	}
	if _, keyStore := tpmOnly.ActiveWrap(); keyStore != keyStoreTPM2 {
		t.Fatalf("the store opened by the TPM wrap reports key store %q", keyStore)
	}
	// And the passphrase still opens it too: adding a wrap removes nothing.
	if value, err := newEncryptedStore(path, passphrase).Get("svc", "key"); err != nil || value != "written-before-the-tpm" {
		t.Fatalf("the original wrap stopped working after a second was added: %q, %v", value, err)
	}
}

// TestEncryptedStoreLockDiscardsTheCachedKey covers the guarantee the lock
// command rests on: after locking, the next read must go back to a wrap.
func TestEncryptedStoreLockDiscardsTheCachedKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credentials.enc.json")
	store := newEncryptedStore(path, passphraseProvider{passphrase: "passphrase"})
	if _, err := store.initialize(); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if err := store.Put("svc", "key", "value"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if _, keyStore := store.ActiveWrap(); keyStore == "" {
		t.Fatalf("the store reports no active wrap after a successful write")
	}

	store.lock()
	if provider, keyStore := store.ActiveWrap(); provider != "" || keyStore != "" {
		t.Fatalf("lock left an active wrap: %q/%q", provider, keyStore)
	}
	if got := AdapterName(store); got != adapterEncryptedFile {
		t.Fatalf("AdapterName after lock = %q, want the bare backend", got)
	}
	// The provider is still configured, so the store reopens rather than
	// breaking — locking is not the same as losing the key.
	if value, err := store.Get("svc", "key"); err != nil || value != "value" {
		t.Fatalf("the store did not reopen after a lock: %q, %v", value, err)
	}
}
