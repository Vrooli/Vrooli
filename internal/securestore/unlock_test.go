package securestore

import (
	"bytes"
	"encoding/base64"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMain keeps the whole package away from the developer's real login
// session. Without it, any test that initializes a passphrase-wrapped store
// would write an open data key into /run/user/<uid> and leave it there — and
// worse, a later test asserting that a wrong passphrase fails would pass for
// the wrong reason by reading the previous test's remembered key.
// The default is no cache at all, so a test that depends on one has to say so
// with useTestUnlockCache. That keeps a remembered key from one test silently
// satisfying another test's assertion that a wrong passphrase fails.
func TestMain(m *testing.M) {
	sessionUnlockCache = func() unlockCache { return noUnlockCache{} }
	os.Exit(m.Run())
}

// useTestUnlockCache points the session cache at a temporary directory that
// stands in for the login session's tmpfs, and puts the real one back
// afterwards.
func useTestUnlockCache(t *testing.T) fileUnlockCache {
	t.Helper()
	cache := fileUnlockCache{path: filepath.Join(t.TempDir(), unlockCacheDirName, unlockCacheFileName)}
	previous := sessionUnlockCache
	sessionUnlockCache = func() unlockCache { return cache }
	t.Cleanup(func() { sessionUnlockCache = previous })
	return cache
}

// TestOneUnlockServesLaterInvocations is the property the whole phase exists
// for. Each newEncryptedStore is a separate CLI invocation: re-prompting for a
// passphrase on each of them would make provisioning three credentials mean
// typing it three times.
func TestOneUnlockServesLaterInvocations(t *testing.T) {
	cache := useTestUnlockCache(t)
	path := filepath.Join(t.TempDir(), "state", "credentials.enc.json")

	// First invocation: the operator supplies the passphrase.
	unlocking := newEncryptedStore(path, passphraseProvider{passphrase: "the operator passphrase"})
	if _, err := unlocking.initialize(); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if err := unlocking.Put("svc", "key", "value"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if _, err := os.Stat(cache.path); err != nil {
		t.Fatalf("the passphrase unlock was not remembered: %v", err)
	}

	// Later invocations know no passphrase at all. Without the cache they
	// could not open the store; with it they must not need to.
	for attempt := 1; attempt <= 2; attempt++ {
		later := newEncryptedStore(path, passphraseProvider{passphrase: ""})
		value, err := later.Get("svc", "key")
		if err != nil {
			t.Fatalf("invocation %d could not read without a passphrase: %v", attempt, err)
		}
		if value != "value" {
			t.Fatalf("invocation %d read %q", attempt, value)
		}
	}
}

// TestLockTakesEffectImmediately is the other half: an operator who locks must
// not find the store still open one command later.
func TestLockTakesEffectImmediately(t *testing.T) {
	cache := useTestUnlockCache(t)
	path := filepath.Join(t.TempDir(), "state", "credentials.enc.json")

	unlocking := newEncryptedStore(path, passphraseProvider{passphrase: "the operator passphrase"})
	if _, err := unlocking.initialize(); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if err := unlocking.Put("svc", "key", "value"); err != nil {
		t.Fatalf("Put: %v", err)
	}

	newEncryptedStore(path, passphraseProvider{passphrase: ""}).lock()

	if _, err := os.Stat(cache.path); !os.IsNotExist(err) {
		t.Fatalf("lock left the remembered key behind: %v", err)
	}
	locked := newEncryptedStore(path, passphraseProvider{passphrase: ""})
	if _, err := locked.Get("svc", "key"); err == nil {
		t.Fatalf("the store was still open after a lock")
	}
	// The passphrase still works: locking discards an unlock, it does not
	// damage the store.
	if value, err := newEncryptedStore(path, passphraseProvider{passphrase: "the operator passphrase"}).Get("svc", "key"); err != nil || value != "value" {
		t.Fatalf("the passphrase stopped working after a lock: %q, %v", value, err)
	}
}

// TestNoCachedKeyReachesTheStateDirectory is the leak assertion the plan asks
// for: the remembered key lives in the session runtime directory and nowhere
// under the durable state directory that holds the store itself.
func TestNoCachedKeyReachesTheStateDirectory(t *testing.T) {
	cache := useTestUnlockCache(t)
	stateDir := t.TempDir()
	path := filepath.Join(stateDir, "credentials.enc.json")

	store := newEncryptedStore(path, passphraseProvider{passphrase: "the operator passphrase"})
	if _, err := store.initialize(); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if err := store.Put("svc", "key", "value"); err != nil {
		t.Fatalf("Put: %v", err)
	}

	file, err := readSealedFile(path)
	if err != nil {
		t.Fatalf("readSealedFile: %v", err)
	}
	dataKey, found := cache.Load(storeFingerprint(file))
	if !found {
		t.Fatalf("the unlock was not remembered, so this test proves nothing")
	}
	encoded := base64.StdEncoding.EncodeToString(dataKey)

	err = filepath.WalkDir(stateDir, func(walked string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, readErr := os.ReadFile(walked)
		if readErr != nil {
			return readErr
		}
		if bytes.Contains(data, dataKey) || strings.Contains(string(data), encoded) {
			t.Fatalf("the open data key was written into the state directory at %s", walked)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk state directory: %v", err)
	}

	// And the remembered key itself is owner-only.
	info, err := os.Stat(cache.path)
	if err != nil {
		t.Fatalf("stat remembered key: %v", err)
	}
	if got := info.Mode().Perm(); got != fs.FileMode(sealedFilePerm) {
		t.Fatalf("remembered key mode = %o, want %o", got, sealedFilePerm)
	}
}

// TestAHostBoundStoreNeverCaches is the plan's last step: the host-bound wrap
// needs no unlock, so caching there would write a data key into the session
// directory to save an operator nothing at all.
func TestAHostBoundStoreNeverCaches(t *testing.T) {
	cache := useTestUnlockCache(t)
	path := filepath.Join(t.TempDir(), "state", "credentials.enc.json")
	hostBound := hostBoundProvider{run: newFakeSystemdCreds("tpm2").run}

	store := newEncryptedStore(path, hostBound)
	if _, err := store.initialize(); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if err := store.Put("svc", "key", "value"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if value, err := newEncryptedStore(path, hostBound).Get("svc", "key"); err != nil || value != "value" {
		t.Fatalf("host-bound read: %q, %v", value, err)
	}

	if _, err := os.Stat(cache.path); !os.IsNotExist(err) {
		t.Fatalf("a host-bound store created an unlock cache entry at %s", cache.path)
	}
}

// TestARememberedKeyFromAnotherStoreIsIgnored covers re-initialization: a
// leftover entry must never open, or fail to open, the wrong file.
func TestARememberedKeyFromAnotherStoreIsIgnored(t *testing.T) {
	cache := useTestUnlockCache(t)
	dir := t.TempDir()

	first := newEncryptedStore(filepath.Join(dir, "first.enc.json"), passphraseProvider{passphrase: "first"})
	if _, err := first.initialize(); err != nil {
		t.Fatalf("initialize first: %v", err)
	}
	if err := first.Put("svc", "key", "first-value"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if _, err := os.Stat(cache.path); err != nil {
		t.Fatalf("first store did not remember its unlock: %v", err)
	}

	// A different store, and the leftover entry names the first one.
	secondPath := filepath.Join(dir, "second.enc.json")
	second := newEncryptedStore(secondPath, passphraseProvider{passphrase: "second"})
	if _, err := second.initialize(); err != nil {
		t.Fatalf("initialize second: %v", err)
	}
	// A fresh invocation with no passphrase must not open the second store off
	// the first one's remembered key.
	if _, err := newEncryptedStore(secondPath, passphraseProvider{passphrase: ""}).Get("svc", "key"); err == nil {
		t.Fatalf("a remembered key from another store opened this one")
	}
	// With the right passphrase it opens, and the stale entry is replaced.
	if _, err := newEncryptedStore(secondPath, passphraseProvider{passphrase: "second"}).Get("svc", "key"); err == nil {
		t.Fatalf("the second store answered for a key it never held")
	}
	file, err := readSealedFile(secondPath)
	if err != nil {
		t.Fatalf("readSealedFile: %v", err)
	}
	if _, found := cache.Load(storeFingerprint(file)); !found {
		t.Fatalf("the second store's unlock was not remembered after a correct passphrase")
	}
}

// TestNoSessionMeansNoCacheRatherThanADiskWrite covers the host with no
// session-scoped memory. The answer there is an unlock that lasts one process,
// never a data key placed on durable storage.
func TestNoSessionMeansNoCacheRatherThanADiskWrite(t *testing.T) {
	previous := sessionUnlockCache
	sessionUnlockCache = func() unlockCache { return noUnlockCache{} }
	t.Cleanup(func() { sessionUnlockCache = previous })

	path := filepath.Join(t.TempDir(), "state", "credentials.enc.json")
	store := newEncryptedStore(path, passphraseProvider{passphrase: "passphrase"})
	if _, err := store.initialize(); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if err := store.Put("svc", "key", "value"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if location := store.cache.Location(); location != "" {
		t.Fatalf("a host with no session reported an unlock location %q", location)
	}
	if _, err := newEncryptedStore(path, passphraseProvider{passphrase: ""}).Get("svc", "key"); err == nil {
		t.Fatalf("a store opened without a passphrase on a host that cannot remember one")
	}
}
