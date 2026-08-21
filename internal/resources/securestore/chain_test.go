package securestore

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// usePassphraseOnlyProviders pins a store under test to the passphrase wrap.
// Without it these tests assert a different thing on a machine with a reachable
// TPM than on one without: the store would be built with an unattended wrap as
// well, and "the passphrase no longer opens it" stops being observable because
// the other wrap opens it regardless — which is the correct behaviour and the
// wrong thing for a test about the passphrase wrap to measure.
func usePassphraseOnlyProviders(t *testing.T) {
	t.Helper()
	previous := defaultKeyProviders
	defaultKeyProviders = func() []keyProvider {
		return []keyProvider{passphraseProvider{source: passphraseSource}}
	}
	t.Cleanup(func() { defaultKeyProviders = previous })
}

// useTestStorePath points the whole chain at a temporary store file and puts
// the real resolver back afterwards, so one test cannot leave the package
// aimed at a path the next one did not choose.
func useTestStorePath(t *testing.T, path string) {
	t.Helper()
	previous := credentialStorePath
	credentialStorePath = func() (string, error) { return path, nil }
	t.Cleanup(func() { credentialStorePath = previous })
}

// spyStore records every operation performed on it. The split-brain guard is a
// claim about what is *not* touched, so counting calls is the only way to
// assert it.
type spyStore struct {
	name  string
	calls int
	err   error
}

func (spy *spyStore) AdapterName() string { return spy.name }

func (spy *spyStore) Put(string, string, string) error {
	spy.calls++
	return spy.err
}

func (spy *spyStore) Get(string, string) (string, error) {
	spy.calls++
	if spy.err != nil {
		return "", spy.err
	}
	return "", fmt.Errorf("%w: nothing stored", ErrNotFound)
}

func (spy *spyStore) Delete(string, string) error {
	spy.calls++
	return spy.err
}

// TestChainSelectsTheEncryptedStoreOnlyWhenTheNativeOneIsAbsent walks the three
// conditions a native adapter can be in. It is the whole selection contract in
// one table.
func TestChainSelectsTheEncryptedStoreOnlyWhenTheNativeOneIsAbsent(t *testing.T) {
	for _, testCase := range []struct {
		name         string
		nativeErr    error
		wantFallback bool
	}{
		{name: "native available", nativeErr: nil, wantFallback: false},
		{name: "native unavailable", nativeErr: fmt.Errorf("%w: keyring session unreachable", ErrUnavailable), wantFallback: false},
		{name: "native absent", nativeErr: fmt.Errorf("%w: no adapter on this host", ErrAbsent), wantFallback: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			native := &spyStore{name: "native", err: testCase.nativeErr}
			fallback := &spyStore{name: adapterEncryptedFile}
			chain := &chainStore{native: native, fallback: fallback}

			_, _ = chain.Get("svc", "key")
			_ = chain.Put("svc", "key", "value")
			_ = chain.Delete("svc", "key")

			if testCase.wantFallback {
				if fallback.calls == 0 {
					t.Fatalf("the encrypted store was never used on an absent native backend")
				}
				if got := chain.AdapterName(); got != adapterEncryptedFile {
					t.Fatalf("AdapterName = %q, want the encrypted store", got)
				}
				return
			}
			if fallback.calls != 0 {
				t.Fatalf("the encrypted store was used %d times while the native backend was %s",
					fallback.calls, testCase.name)
			}
			if got := chain.AdapterName(); got != "native" {
				t.Fatalf("AdapterName = %q, want the native adapter", got)
			}
		})
	}
}

// TestChainNeverOpensTheEncryptedStoreWhileTheNativeOneIsMerelyUnreachable is
// the split-brain guard stated on its own, because it is the load-bearing rule
// of the whole design and the one a future refactor is most likely to break. A
// chain that falls back on any error looks correct and silently splits
// credentials across two stores according to session health: a value written
// while the keyring was up disappears when it goes down, and one written while
// it was down disappears when it recovers.
func TestChainNeverOpensTheEncryptedStoreWhileTheNativeOneIsMerelyUnreachable(t *testing.T) {
	native := &spyStore{name: "libsecret", err: fmt.Errorf("%w: Could not connect: Permission denied", ErrUnavailable)}
	// A real encrypted store over a path that does not exist: if the chain ever
	// reaches it, the error changes shape and the test says so.
	fallback := newEncryptedStore(filepath.Join(t.TempDir(), "never-touched.enc.json"),
		passphraseProvider{passphrase: "unused"})
	chain := &chainStore{native: native, fallback: fallback}

	for _, operation := range []func() error{
		func() error { _, err := chain.Get("svc", "key"); return err },
		func() error { return chain.Put("svc", "key", "value") },
		func() error { return chain.Delete("svc", "key") },
	} {
		err := operation()
		if !errors.Is(err, ErrUnavailable) {
			t.Fatalf("operation error = %v, want the native ErrUnavailable to stand", err)
		}
		if errors.Is(err, ErrAbsent) {
			t.Fatalf("an unreachable native store was reported as absent, which would invite a fallback: %v", err)
		}
	}
	if fallback.initialized() {
		t.Fatalf("the encrypted store file was created while the native backend was merely unreachable")
	}
	if provider, keyStore := fallback.ActiveWrap(); provider != "" || keyStore != "" {
		t.Fatalf("the encrypted store was opened during a native outage: %q/%q", provider, keyStore)
	}
}

// TestChainDecidesOncePerProcess keeps the authority from moving under a caller
// mid-run, which would split credentials just as surely as falling back on the
// wrong error.
func TestChainDecidesOncePerProcess(t *testing.T) {
	native := &spyStore{name: "native", err: fmt.Errorf("%w: no adapter", ErrAbsent)}
	fallback := &spyStore{name: adapterEncryptedFile}
	chain := &chainStore{native: native, fallback: fallback}

	if _, err := chain.Get("svc", "key"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("first Get = %v, want the fallback's answer", err)
	}
	// The native adapter recovers. The already-decided chain must not move.
	native.err = nil
	if _, _ = chain.Get("svc", "key"); fallback.calls < 2 {
		t.Fatalf("the chain moved off the encrypted store after the native one recovered")
	}
}

func TestBackendOverrideSelectsTheEncryptedStoreOnAHostThatHasANativeOne(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credentials.enc.json")
	useTestStorePath(t, path)

	t.Setenv(BackendOverrideEnv, BackendEncryptedFile)
	if got := backendName(Default()); got != adapterEncryptedFile {
		t.Fatalf("override backend = %q, want the encrypted store", got)
	}

	t.Setenv(BackendOverrideEnv, BackendNative)
	if got := backendName(Default()); got == adapterEncryptedFile {
		t.Fatalf("the native override selected the encrypted store")
	}

	// A typo must not silently pick a backend. Choosing the wrong store is how
	// an operator ends up with two of them.
	t.Setenv(BackendOverrideEnv, "keychain-please")
	store := Default()
	if _, err := store.Get("svc", "key"); !errors.Is(err, ErrAbsent) {
		t.Fatalf("an unrecognized override = %v, want ErrAbsent", err)
	} else if !strings.Contains(err.Error(), BackendOverrideEnv) {
		t.Fatalf("the refusal does not name the variable that caused it: %v", err)
	}
}

// TestDiagnoseNamesTheBackendAndTheKeyWrap is the operator-facing half of this
// phase: doctor must always answer "which store holds my values, and what
// protects the key".
func TestDiagnoseNamesTheBackendAndTheKeyWrap(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credentials.enc.json")
	useTestStorePath(t, path)
	usePassphraseOnlyProviders(t)
	t.Setenv(BackendOverrideEnv, BackendEncryptedFile)

	// Before initialization the diagnosis must be absent and must name the
	// command that fixes it — not a package to install on a host that will
	// never have a desktop.
	diagnosis := Diagnose()
	if diagnosis.Condition != "absent" {
		t.Fatalf("uninitialized store condition = %q, want absent", diagnosis.Condition)
	}
	if diagnosis.Backend != adapterEncryptedFile {
		t.Fatalf("backend = %q, want %q", diagnosis.Backend, adapterEncryptedFile)
	}
	if !strings.Contains(diagnosis.Fix, "credentials store init") {
		t.Fatalf("fix = %q, want it to name the initialization command", diagnosis.Fix)
	}
	if diagnosis.Unlocked {
		t.Fatalf("an uninitialized store reported itself unlocked")
	}

	SetPassphrase("the operator passphrase")
	t.Cleanup(func() { SetPassphrase("") })
	if _, err := InitializeStore("the operator passphrase"); err != nil {
		t.Fatalf("InitializeStore: %v", err)
	}

	diagnosis = Diagnose()
	if diagnosis.Condition != "available" || !diagnosis.Available {
		t.Fatalf("initialized store condition = %q (%v)", diagnosis.Condition, diagnosis.Explanation)
	}
	if diagnosis.Backend != adapterEncryptedFile {
		t.Fatalf("backend = %q", diagnosis.Backend)
	}
	if diagnosis.KeyWrap != providerPassphrase || diagnosis.KeyStore != keyStorePassphrase {
		t.Fatalf("key wrap = %q/%q, want the passphrase wrap", diagnosis.KeyWrap, diagnosis.KeyStore)
	}
	if !diagnosis.Unlocked {
		t.Fatalf("a store opened by a wrap reported itself locked")
	}
	if !strings.Contains(diagnosis.Adapter, adapterEncryptedFile) {
		t.Fatalf("adapter = %q, want it to name the encrypted backend", diagnosis.Adapter)
	}
}

// TestDescribeStoreListsWrapsWithoutUnlocking is why service and key names stay
// in cleartext: an operator must be able to see what a host holds, and which
// wraps could open it, without the key.
func TestDescribeStoreListsWrapsWithoutUnlocking(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credentials.enc.json")
	useTestStorePath(t, path)
	usePassphraseOnlyProviders(t)
	t.Setenv(BackendOverrideEnv, BackendEncryptedFile)

	if _, err := InitializeStore("passphrase"); err != nil {
		t.Fatalf("InitializeStore: %v", err)
	}
	SetPassphrase("")

	status, err := DescribeStore()
	if err != nil {
		t.Fatalf("DescribeStore: %v", err)
	}
	if !status.Initialized || status.Path != path {
		t.Fatalf("status = %+v", status)
	}
	if len(status.Wraps) != 1 || status.Wraps[0].Provider != providerPassphrase {
		t.Fatalf("wraps = %+v, want the passphrase wrap listed without unlocking", status.Wraps)
	}
	if !status.Active {
		t.Fatalf("the overridden backend did not report itself as the authority")
	}
}

func TestChangePassphraseReplacesOnlyTheWrap(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credentials.enc.json")
	useTestStorePath(t, path)
	usePassphraseOnlyProviders(t)
	t.Setenv(BackendOverrideEnv, BackendEncryptedFile)
	if _, err := InitializeStore("old-passphrase"); err != nil {
		t.Fatal(err)
	}
	SetPassphrase("old-passphrase")
	store := Default()
	if err := store.Put("vrooli/test", "marker", "marker-value"); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := ChangePassphraseStore("wrong-passphrase", "new-passphrase"); err == nil {
		t.Fatal("wrong current passphrase unexpectedly changed the store")
	}
	afterWrong, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, afterWrong) {
		t.Fatal("wrong current passphrase changed the store file")
	}
	if err := ChangePassphraseStore("old-passphrase", "new-passphrase"); err != nil {
		t.Fatal(err)
	}
	SetPassphrase("old-passphrase")
	if _, err := Default().Get("vrooli/test", "marker"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("old passphrase read = %v, want unavailable", err)
	}
	SetPassphrase("new-passphrase")
	value, err := Default().Get("vrooli/test", "marker")
	if err != nil || value != "marker-value" {
		t.Fatalf("new passphrase read = %q, %v", value, err)
	}
}
