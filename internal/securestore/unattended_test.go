package securestore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// useTestUnattendedProviders substitutes the wraps that count as unattended, so
// a test can decide what this host supports.
func useTestUnattendedProviders(t *testing.T, providers ...keyProvider) {
	t.Helper()
	previous := unattendedProviders
	unattendedProviders = func() []keyProvider { return providers }
	t.Cleanup(func() { unattendedProviders = previous })
}

func newPassphraseOnlyStore(t *testing.T, passphrase passphraseProvider) (string, *encryptedStore) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "credentials.enc.json")
	store := newEncryptedStore(path, passphrase)
	if _, err := store.initialize(); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if err := store.Put("svc", "key", "value-written-before-any-unattended-wrap"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	return path, store
}

// This is the property the whole feature rests on: an operator supplies a
// passphrase once, and the host is left able to open the store without one.
// The version that shipped before added the wrap on exactly one code path,
// which the two paths an operator actually reaches did not take, so the store
// stayed passphrase-only and the passphrase was retyped at every boot.
func TestEnsureUnattendedWrapConvergesAPassphraseOnlyStore(t *testing.T) {
	passphrase := passphraseProvider{passphrase: "the operator passphrase"}
	path, _ := newPassphraseOnlyStore(t, passphrase)
	fake := newFakeSystemdCreds(hostBoundTPM2Mode)
	useTestUnattendedProviders(t, hostBoundProvider{run: fake.run})

	status, err := ensureUnattendedWrap(newEncryptedStore(path, passphrase))
	if err != nil {
		t.Fatalf("ensureUnattendedWrap: %v", err)
	}
	if !status.Enabled || !status.Added {
		t.Fatalf("status = %+v, want an added, enabled wrap", status)
	}
	if status.Provider != providerHostBound || status.KeyStore != keyStoreTPM2 {
		t.Fatalf("status = %+v, want the TPM-protected host-bound wrap", status)
	}

	// The passphrase is now genuinely unnecessary: a store handed only the
	// unattended provider opens a value written before that wrap existed.
	unattended := newEncryptedStore(path, hostBoundProvider{run: fake.run})
	value, err := unattended.Get("svc", "key")
	if err != nil {
		t.Fatalf("Get with no passphrase available: %v", err)
	}
	if value != "value-written-before-any-unattended-wrap" {
		t.Fatalf("value = %q; adding a wrap must not re-encrypt or lose anything", value)
	}
	// And the passphrase still opens it, because it is the recovery path that
	// makes the store portable to a host with a different TPM.
	if _, err := newEncryptedStore(path, passphrase).Get("svc", "key"); err != nil {
		t.Fatalf("the passphrase wrap stopped working after convergence: %v", err)
	}
}

// A second run must not churn the file. Convergence is something setup does on
// every invocation, so it has to be a no-op once it has been reached.
func TestEnsureUnattendedWrapIsIdempotent(t *testing.T) {
	passphrase := passphraseProvider{passphrase: "the operator passphrase"}
	path, _ := newPassphraseOnlyStore(t, passphrase)
	fake := newFakeSystemdCreds(hostBoundTPM2Mode)
	useTestUnattendedProviders(t, hostBoundProvider{run: fake.run})

	if _, err := ensureUnattendedWrap(newEncryptedStore(path, passphrase)); err != nil {
		t.Fatalf("first convergence: %v", err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read store: %v", err)
	}

	status, err := ensureUnattendedWrap(newEncryptedStore(path, passphrase))
	if err != nil {
		t.Fatalf("second convergence: %v", err)
	}
	if !status.Enabled || status.Added || status.Repaired {
		t.Fatalf("status = %+v, want an already-enabled wrap reported without a change", status)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("re-read store: %v", err)
	}
	if string(before) != string(after) {
		t.Fatalf("a second convergence rewrote the store file")
	}
}

// A wrap that has stopped opening is the dangerous state, because it still
// appears in the file: the host reports itself unattended right up until the
// reboot that strands it. This is what a cleared TPM, a deleted Keychain item,
// or a firmware update that invalidates a PCR-bound policy looks like.
func TestEnsureUnattendedWrapRepairsAWrapThatNoLongerOpens(t *testing.T) {
	passphrase := passphraseProvider{passphrase: "the operator passphrase"}
	path, _ := newPassphraseOnlyStore(t, passphrase)
	fake := newFakeSystemdCreds(hostBoundTPM2Mode)
	useTestUnattendedProviders(t, hostBoundProvider{run: fake.run})

	if _, err := ensureUnattendedWrap(newEncryptedStore(path, passphrase)); err != nil {
		t.Fatalf("initial convergence: %v", err)
	}

	// Everything this key was sealed to is gone, but the wrap record remains.
	fake.blobs = map[string][]byte{}

	file, err := readSealedFile(path)
	if err != nil {
		t.Fatalf("read sealed file: %v", err)
	}
	if inspect := inspectUnattendedWrap(file); inspect.Enabled {
		t.Fatalf("a wrap that no longer opens was reported as unattended: %+v", inspect)
	}

	status, err := ensureUnattendedWrap(newEncryptedStore(path, passphrase))
	if err != nil {
		t.Fatalf("repair: %v", err)
	}
	if !status.Enabled || !status.Repaired || status.Added {
		t.Fatalf("status = %+v, want the broken wrap reported as repaired", status)
	}
	if _, err := newEncryptedStore(path, hostBoundProvider{run: fake.run}).Get("svc", "key"); err != nil {
		t.Fatalf("the repaired wrap does not open the store: %v", err)
	}
}

// A host with no unattended option is degraded, not broken. Failing here would
// break installation on exactly the hardware the passphrase wrap exists to
// serve, so the outcome has to be a reported reason.
func TestEnsureUnattendedWrapReportsABlockedHostWithoutFailing(t *testing.T) {
	passphrase := passphraseProvider{passphrase: "the operator passphrase"}
	path, _ := newPassphraseOnlyStore(t, passphrase)
	useTestUnattendedProviders(t, hostBoundProvider{run: newFakeSystemdCreds().run})

	status, err := ensureUnattendedWrap(newEncryptedStore(path, passphrase))
	if err != nil {
		t.Fatalf("ensureUnattendedWrap on a host with no unattended provider = %v, want a reported status", err)
	}
	if status.Enabled {
		t.Fatalf("status = %+v, want a host with no working provider reported as attended", status)
	}
	if !strings.Contains(status.Blocked, providerHostBound) {
		t.Fatalf("Blocked = %q, want it to name the provider that could not be used", status.Blocked)
	}
	// The store still works; only the reboot story is affected.
	if _, err := newEncryptedStore(path, passphrase).Get("svc", "key"); err != nil {
		t.Fatalf("a blocked convergence damaged the store: %v", err)
	}
}

// DescribeStore is what `credentials store status` and setup both read, so the
// unattended answer it reports has to be the proved one rather than an
// inference from the wrap list.
func TestDescribeStoreReportsTheVerifiedUnattendedAnswer(t *testing.T) {
	passphrase := passphraseProvider{passphrase: "the operator passphrase"}
	path, _ := newPassphraseOnlyStore(t, passphrase)
	useTestStorePath(t, path)
	SetPassphrase("the operator passphrase")
	t.Cleanup(func() { SetPassphrase("") })

	fake := newFakeSystemdCreds(hostBoundTPM2Mode)
	useTestUnattendedProviders(t, hostBoundProvider{run: fake.run})

	before, err := DescribeStore()
	if err != nil {
		t.Fatalf("DescribeStore: %v", err)
	}
	if before.Unattended.Enabled {
		t.Fatalf("a passphrase-only store reported unattended: %+v", before.Unattended)
	}

	if _, err := ensureUnattendedWrap(newEncryptedStore(path, passphrase)); err != nil {
		t.Fatalf("convergence: %v", err)
	}

	after, err := DescribeStore()
	if err != nil {
		t.Fatalf("DescribeStore after convergence: %v", err)
	}
	if !after.Unattended.Enabled || after.Unattended.Provider != providerHostBound {
		t.Fatalf("Unattended = %+v, want the host-bound wrap reported after convergence", after.Unattended)
	}
}

// Converging on unattended access must not cost the operator the session they
// are converging from. Adding a wrap leaves the data key untouched, so a
// session that had already unlocked the store stays unlocked — otherwise setup
// would ask for the passphrase again to finish the run that just removed the
// need for it.
func TestConvergenceKeepsAnExistingSessionUnlock(t *testing.T) {
	passphrase := passphraseProvider{passphrase: "the operator passphrase"}
	path, _ := newPassphraseOnlyStore(t, passphrase)
	fake := newFakeSystemdCreds(hostBoundTPM2Mode)
	useTestUnattendedProviders(t, hostBoundProvider{run: fake.run})

	cachePath := filepath.Join(t.TempDir(), "credential-unlock")
	session := fileUnlockCache{path: cachePath}

	// A session that has already unlocked: the key is cached, and the process
	// itself holds no passphrase.
	unlocking := newEncryptedStore(path, passphrase)
	unlocking.cache = session
	if _, _, err := unlocking.open(); err != nil {
		t.Fatalf("initial unlock: %v", err)
	}
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("the unlock was not cached: %v", err)
	}

	if _, err := ensureUnattendedWrap(newEncryptedStore(path, passphrase)); err != nil {
		t.Fatalf("convergence: %v", err)
	}

	// The same session, with no passphrase of its own and no access to the new
	// wrap, must still open the store from its cache.
	resumed := newEncryptedStore(path, passphraseProvider{})
	resumed.cache = session
	if _, err := resumed.Get("svc", "key"); err != nil {
		t.Fatalf("convergence invalidated a live session unlock: %v", err)
	}
}

// A changed passphrase still has to invalidate the cache, or a rotation would
// leave the old key usable for the rest of the session.
func TestChangingThePassphraseInvalidatesTheSessionUnlock(t *testing.T) {
	passphrase := passphraseProvider{passphrase: "the operator passphrase"}
	path, _ := newPassphraseOnlyStore(t, passphrase)

	before, err := readSealedFile(path)
	if err != nil {
		t.Fatalf("read sealed file: %v", err)
	}
	if err := changePassphraseStore(newEncryptedStore(path, passphrase), "the operator passphrase", "a new operator passphrase"); err != nil {
		t.Fatalf("changePassphraseStore: %v", err)
	}
	after, err := readSealedFile(path)
	if err != nil {
		t.Fatalf("re-read sealed file: %v", err)
	}
	if storeFingerprint(before) == storeFingerprint(after) {
		t.Fatalf("the store fingerprint survived a passphrase change, so a cached key would outlive the passphrase that made it")
	}
}

// The grant setup makes with usermod is invisible to every process that was
// already running, including the onboarding scenario where the operator types
// the passphrase. Picking it up at the TPM call is what lets that process add
// the wrap that removes the need for the passphrase it just received.
func TestWithDeviceGroupOnlyWrapsAnOutstandingGrant(t *testing.T) {
	name, args := withDeviceGroup("systemd-creds", []string{"encrypt", "--with-key=tpm2", "-", "-"})
	pending := PendingGroupGrant()
	if pending == "" {
		if name != "systemd-creds" || strings.Join(args, " ") != "encrypt --with-key=tpm2 - -" {
			t.Fatalf("command was rewritten with no grant outstanding: %s %v", name, args)
		}
		return
	}
	if !strings.HasSuffix(name, "sg") {
		// A host without `sg` is left alone rather than handed a command it
		// cannot run; the diagnosis then tells the operator to start a new
		// session instead.
		if name != "systemd-creds" {
			t.Fatalf("command was rewritten to %q, want either sg or the original", name)
		}
		return
	}
	if len(args) != 3 || args[0] != pending || args[1] != "-c" {
		t.Fatalf("sg invocation = %v, want the pending group, -c, and one quoted command", args)
	}
	for _, part := range []string{"'systemd-creds'", "'encrypt'", "'--with-key=tpm2'"} {
		if !strings.Contains(args[2], part) {
			t.Fatalf("sg command %q does not carry %s as a quoted argument", args[2], part)
		}
	}
}

// The quoting is what keeps an argument with a quote in it from becoming two
// arguments, or worse, a second command.
func TestShellQuoteArgumentSurvivesAQuoteInTheValue(t *testing.T) {
	if got, want := shellQuoteArgument(`it's`), `'it'\''s'`; got != want {
		t.Fatalf("shellQuoteArgument(`it's`) = %s, want %s", got, want)
	}
}
