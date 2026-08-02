package securestore

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
)

// encryptedStore is the credential backend for a host with no native one. It
// implements the same Store interface as libsecret, Keychain, and Credential
// Manager, and passes the same conformance suite, because a host class whose
// credential path behaves differently from every other host class is a host
// class nobody can support.

// adapterEncryptedFile is the backend label. It is reported with the active key
// wrap appended once the store has been opened, so an operator reading `doctor`
// always learns which store holds their values and what protects the key.
const adapterEncryptedFile = "encrypted-file"

type encryptedStore struct {
	path string
	// providers are tried in order when opening the store. Order is strongest
	// first; any one of them opens it.
	providers []keyProvider

	mu sync.Mutex
	// dataKey is cached for the process lifetime once a wrap has opened it, so
	// resolving a manifest with several credentialed resources unwraps once
	// rather than once per resource. It is never written anywhere.
	dataKey []byte
	// openedBy and keyStore record which wrap actually opened the store, for
	// the operator diagnosis.
	openedBy string
	keyStore string
	// cache holds the open data key across CLI invocations in one login
	// session, so a passphrase-wrapped store is unlocked once rather than on
	// every command. A host-bound host never writes to it.
	cache unlockCache
}

// newEncryptedStore builds the adapter. It performs no I/O: whether the store
// exists and whether a wrap opens it are runtime properties discovered when a
// credential is actually needed, matching how every other adapter behaves.
func newEncryptedStore(path string, providers ...keyProvider) *encryptedStore {
	return &encryptedStore{path: path, providers: providers, cache: sessionUnlockCache()}
}

// AdapterName reports the backend and, once known, the wrap that opened it.
// Before the store has been opened there is nothing honest to append, and
// probing here would make a diagnostic label perform I/O.
func (store *encryptedStore) AdapterName() string {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.keyStore == "" {
		return adapterEncryptedFile
	}
	return adapterEncryptedFile + " (" + store.openedBy + "/" + store.keyStore + ")"
}

// ActiveWrap reports the provider and key store that opened this store, or
// empty strings when it has not been opened. Diagnose uses it to name the wrap
// without re-deriving it.
func (store *encryptedStore) ActiveWrap() (string, string) {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.openedBy, store.keyStore
}

// classifyFileError maps a sealed-file failure onto the three transport
// conditions every adapter owes its caller. The mapping is the whole reason
// this adapter can join the shared conformance suite unchanged.
func (store *encryptedStore) classifyFileError(err error) error {
	switch {
	case errors.Is(err, os.ErrNotExist):
		// Not initialized is genuinely "this host has no usable backend yet",
		// and the remedy is a command rather than a session repair.
		return fmt.Errorf("%w: no credential store on this host; run `vrooli credentials store init`", ErrAbsent)
	case errors.Is(err, errSealedVersion):
		return fmt.Errorf("%w: %v; upgrade Vrooli to read this store", ErrUnavailable, err)
	case errors.Is(err, errSealedCorrupt):
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	case errors.Is(err, os.ErrPermission):
		return fmt.Errorf("%w: the credential store file is not readable by this user: %v", ErrUnavailable, err)
	default:
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
}

// open loads the file and unwraps the data key, caching both for the process
// lifetime. Every read and write goes through it, so the three error conditions
// are decided in exactly one place.
func (store *encryptedStore) open() (*sealedFile, []byte, error) {
	file, err := readSealedFile(store.path)
	if err != nil {
		return nil, nil, store.classifyFileError(err)
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	return store.openLoadedLocked(file)
}

// unwrapDataKey tries each provider against the wrap it wrote. A provider that
// is simply unavailable on this host is not a failure of the store, so the
// reasons are collected and only reported together when none of them opens it.
func (store *encryptedStore) unwrapDataKey(file *sealedFile) ([]byte, string, string, error) {
	reasons := []string{}
	for _, provider := range store.providers {
		wrap, found := file.wrapFor(provider.Name())
		if !found {
			reasons = append(reasons, provider.Name()+": this store has no wrap for that provider")
			continue
		}
		dataKey, err := provider.Unwrap(wrap)
		if err != nil {
			reasons = append(reasons, provider.Name()+": "+conciseReason(err))
			continue
		}
		keyStore := wrap.KeyStore
		if keyStore == "" {
			keyStore = "unrecorded"
		}
		return dataKey, provider.Name(), keyStore, nil
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "no key provider is configured")
	}
	return nil, "", "", fmt.Errorf(
		"%w: no key wrap opened the credential store (%s); run `vrooli credentials store unlock`",
		ErrUnavailable, strings.Join(reasons, "; "))
}

func (store *encryptedStore) Get(service, key string) (string, error) {
	file, dataKey, err := store.open()
	if err != nil {
		return "", err
	}
	index, found := file.findEntry(service, key)
	if !found {
		return "", fmt.Errorf("%w: %s/%s", ErrNotFound, service, key)
	}
	value, err := openEntry(dataKey, service, key, file.Entries[index])
	if err != nil {
		// A tampered or unreadable entry is a broken backend, never a missing
		// value. Collapsing the two is what once let a host fault read as an
		// unset credential and abort a scenario start.
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return value, nil
}

func (store *encryptedStore) Put(service, key, value string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.mutate(func(file *sealedFile, dataKey []byte) (bool, error) {
		entry, err := sealEntry(dataKey, service, key, value)
		if err != nil {
			return false, err
		}
		file.putEntry(entry)
		return true, nil
	})
}

// Delete is idempotent. Removing what is already gone is the state the caller
// asked for, and every adapter must agree on that — the shared conformance
// suite caught libsecret disagreeing.
func (store *encryptedStore) Delete(service, key string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.mutate(func(file *sealedFile, _ []byte) (bool, error) {
		return file.deleteEntry(service, key), nil
	})
}

// mutate serializes read-modify-write so two concurrent writers cannot each
// load the file, change their own copy, and have the second rename erase the
// first. The caller holds store.mu; openLocked is the lock-free half of open.
func (store *encryptedStore) mutate(change func(*sealedFile, []byte) (bool, error)) error {
	file, dataKey, err := store.openLocked()
	if err != nil {
		return err
	}
	changed, err := change(file, dataKey)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}
	if err := writeSealedFile(store.path, file); err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return nil
}

// openLocked is open without the mutex, for callers that already hold it.
func (store *encryptedStore) openLocked() (*sealedFile, []byte, error) {
	file, err := readSealedFile(store.path)
	if err != nil {
		return nil, nil, store.classifyFileError(err)
	}
	return store.openLoadedLocked(file)
}

// openLoadedLocked turns a loaded file into an open data key. It is the single
// place the session cache is consulted and populated, so "unlocked once per
// session" cannot drift between the read and write paths.
func (store *encryptedStore) openLoadedLocked(file *sealedFile) (*sealedFile, []byte, error) {
	if store.dataKey != nil {
		return file, store.dataKey, nil
	}
	fingerprint := storeFingerprint(file)
	if dataKey, found := store.cache.Load(fingerprint); found {
		store.dataKey = dataKey
		store.openedBy, store.keyStore = cachedWrapFor(file)
		return file, dataKey, nil
	}

	dataKey, provider, keyStore, err := store.unwrapDataKey(file)
	if err != nil {
		return nil, nil, err
	}
	store.dataKey, store.openedBy, store.keyStore = dataKey, provider, keyStore
	// Only a passphrase unlock is worth caching. The host-bound wrap opens the
	// store with no human action, so caching there would write a data key into
	// the session directory to save an operator nothing at all.
	if provider == providerPassphrase {
		if err := store.cache.Save(fingerprint, dataKey); err != nil {
			return nil, nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
		}
	}
	return file, dataKey, nil
}

// cachedWrapFor names the wrap a cached key came from. A cached key is only
// ever written by the passphrase path, so that is what it reports; the key
// store is read back from the file rather than assumed.
func cachedWrapFor(file *sealedFile) (string, string) {
	if wrap, found := file.wrapFor(providerPassphrase); found {
		return providerPassphrase, wrap.KeyStore
	}
	return providerPassphrase, keyStorePassphrase
}

// initialized reports whether this host has a credential store file, without
// unlocking it. Listing what exists must not require the key.
func (store *encryptedStore) initialized() bool {
	_, err := os.Stat(store.path)
	return err == nil
}

// initialize creates the store: one random data key, wrapped independently by
// every provider that is actually available on this host. Storing several wraps
// side by side is what lets a host gain a TPM later without re-encrypting a
// single value, and what lets Vrooli pick the strongest option per host with no
// per-machine operator configuration.
//
// It refuses to overwrite an existing store. Silently replacing one would
// destroy every credential it holds, which is not a thing a command called
// "init" should be able to do by accident.
func (store *encryptedStore) initialize() ([]wrappedKey, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.initialized() {
		return nil, fmt.Errorf("a credential store already exists at %s; refusing to replace it", store.path)
	}

	dataKey, err := newDataKey()
	if err != nil {
		return nil, err
	}
	file := &sealedFile{Version: sealedFormatVersion}
	reasons := []string{}
	for _, provider := range store.providers {
		if _, err := provider.Available(); err != nil {
			reasons = append(reasons, provider.Name()+": "+conciseReason(err))
			continue
		}
		wrap, err := provider.Wrap(dataKey)
		if err != nil {
			reasons = append(reasons, provider.Name()+": "+conciseReason(err))
			continue
		}
		file.putWrap(wrap)
	}
	if len(file.Wraps) == 0 {
		return nil, fmt.Errorf("%w: no key provider can protect a data key on this host (%s)",
			ErrAbsent, strings.Join(reasons, "; "))
	}
	if err := writeSealedFile(store.path, file); err != nil {
		return nil, err
	}
	// Wraps are appended in provider order, which is strongest first, so the
	// first one is the one that will open this store from now on.
	store.dataKey = dataKey
	store.openedBy = file.Wraps[0].Provider
	store.keyStore = file.Wraps[0].KeyStore
	// An init that only produced a passphrase wrap is itself an unlock. Not
	// remembering it here would make the very next command re-prompt, which is
	// the behavior this design exists to avoid.
	if store.openedBy == providerPassphrase {
		if err := store.cache.Save(storeFingerprint(file), dataKey); err != nil {
			return nil, err
		}
	}
	return file.Wraps, nil
}

// addWrap adds or replaces one provider's wrap of the existing data key. It is
// how a host that gains a TPM starts using it: the data key does not change, so
// no stored value is re-encrypted and nothing can be lost in the process.
func (store *encryptedStore) addWrap(provider keyProvider) (wrappedKey, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	file, dataKey, err := store.openLocked()
	if err != nil {
		return wrappedKey{}, err
	}
	if _, err := provider.Available(); err != nil {
		return wrappedKey{}, err
	}
	wrap, err := provider.Wrap(dataKey)
	if err != nil {
		return wrappedKey{}, err
	}
	file.putWrap(wrap)
	if err := writeSealedFile(store.path, file); err != nil {
		return wrappedKey{}, err
	}
	return wrap, nil
}

// lock discards the cached data key. The next operation must unwrap again.
func (store *encryptedStore) lock() {
	store.mu.Lock()
	defer store.mu.Unlock()
	// Zero the key before dropping the reference so a heap dump taken after a
	// lock does not still contain it.
	for index := range store.dataKey {
		store.dataKey[index] = 0
	}
	store.dataKey = nil
	store.openedBy = ""
	store.keyStore = ""
	// The session cache is the whole reason a lock has to be an explicit
	// command: without clearing it, "locked" would last until the next process
	// read the key straight back out.
	_ = store.cache.Clear()
}
