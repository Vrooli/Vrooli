package securestore

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// The operator surface for the encrypted store. Every native adapter is managed
// by the operating system, so this is the only backend Vrooli has to create,
// unlock, and describe itself.

// WrapInfo is one key-encryption wrap as an operator sees it. It carries no key
// material — only which provider wrote the wrap and what protects it.
type WrapInfo struct {
	Provider string `json:"provider"`
	// KeyStore distinguishes a TPM-protected wrap from one protected by a host
	// key on the same disk. They are not equally strong and are never merged
	// into a single "encrypted" claim.
	KeyStore string `json:"key_store"`
}

// StoreStatus is what `vrooli credentials store status` prints. Listing the
// wraps does not require opening the store, which is exactly why service and
// key names are not sealed.
type StoreStatus struct {
	Path        string `json:"path"`
	Initialized bool   `json:"initialized"`
	// Unlocked is true when a wrap has opened the data key in this process.
	Unlocked       bool       `json:"unlocked"`
	ActiveWrap     string     `json:"active_wrap,omitempty"`
	ActiveKeyStore string     `json:"active_key_store,omitempty"`
	Wraps          []WrapInfo `json:"wraps"`
	// Entries is how many credentials the store holds. It is readable without
	// the key and never includes a value.
	Entries int `json:"entries"`
	// Active is true when this store is the backend the host is actually using,
	// rather than a store sitting beside a working native adapter.
	Active bool `json:"active"`
	// UnlockCache is where a passphrase unlock is remembered for the rest of
	// this login session, or empty on a host with no session-scoped memory —
	// where an unlock lasts exactly one command. It never holds a credential
	// value, only the key that opens them, and it is never on durable storage.
	UnlockCache string `json:"unlock_cache,omitempty"`
	// HostBoundBlocked names what stops the unattended host-bound wrap from
	// working here, or is empty when nothing does. It is reported because the
	// difference between "this host reboots unattended" and "this host needs a
	// passphrase typed at every boot" is invisible otherwise, and an operator
	// who assumes the wrong one finds out during an outage.
	HostBoundBlocked string `json:"host_bound_blocked,omitempty"`
}

// errNoEncryptedBackend means the process selected a backend that has no
// encrypted store behind it, which only happens under an invalid override.
var errNoEncryptedBackend = errors.New("this host has no encrypted credential store backend")

func encryptedStoreForAdmin() (*encryptedStore, Store, error) {
	store := Default()
	encrypted, ok := encryptedBackend(store)
	if !ok {
		return nil, store, fmt.Errorf("%w: %s", errNoEncryptedBackend, AdapterName(store))
	}
	return encrypted, store, nil
}

// DescribeStore reports the encrypted store on this host without unlocking it.
func DescribeStore() (StoreStatus, error) {
	encrypted, chain, err := encryptedStoreForAdmin()
	if err != nil {
		return StoreStatus{}, err
	}
	status := StoreStatus{
		Path:             encrypted.path,
		Active:           backendName(chain) == adapterEncryptedFile,
		UnlockCache:      encrypted.cache.Location(),
		HostBoundBlocked: hostBoundFix(),
	}
	provider, keyStore := encrypted.ActiveWrap()
	status.ActiveWrap, status.ActiveKeyStore = provider, keyStore
	status.Unlocked = provider != ""

	file, err := readSealedFile(encrypted.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return status, nil
		}
		return status, encrypted.classifyFileError(err)
	}
	status.Initialized = true
	status.Entries = len(file.Entries)
	for _, wrap := range file.Wraps {
		status.Wraps = append(status.Wraps, WrapInfo{Provider: wrap.Provider, KeyStore: wrap.KeyStore})
	}

	// "Unlocked" has to mean "a wrap opens this store without asking the
	// operator for anything new", not "this particular handle happens to have
	// opened it". Every command is a fresh process, so the handle-local answer
	// would report a working host-bound store as locked forever.
	if !status.Unlocked {
		if _, _, err := encrypted.open(); err == nil {
			provider, keyStore := encrypted.ActiveWrap()
			status.Unlocked, status.ActiveWrap, status.ActiveKeyStore = true, provider, keyStore
		}
	}
	// A store the host-bound wrap opens needs no unlock and therefore never
	// writes one. Naming a location it does not use would tell an operator to
	// go looking for a file that is not there.
	if status.ActiveWrap == providerHostBound {
		status.UnlockCache = ""
	}
	return status, nil
}

// InitializeStore creates the encrypted store on this host. The passphrase may
// be empty on a host whose host-bound wrap works, which is the case that lets a
// server reboot into a working state with no human at all.
func InitializeStore(passphrase string) (StoreStatus, error) {
	encrypted, _, err := encryptedStoreForAdmin()
	if err != nil {
		return StoreStatus{}, err
	}
	SetPassphrase(passphrase)
	if _, err := encrypted.initialize(); err != nil {
		return StoreStatus{}, err
	}
	return DescribeStore()
}

// UnlockStore opens the encrypted store with an operator passphrase and keeps
// the result available to later commands. It proves the passphrase before
// reporting success, so an operator never walks away believing a typo unlocked
// anything.
func UnlockStore(passphrase string) (StoreStatus, error) {
	encrypted, _, err := encryptedStoreForAdmin()
	if err != nil {
		return StoreStatus{}, err
	}
	if !encrypted.initialized() {
		return StoreStatus{}, fmt.Errorf("%w: no credential store on this host; run `vrooli credentials store init`", ErrAbsent)
	}
	SetPassphrase(passphrase)
	encrypted.lock()
	if _, _, err := encrypted.open(); err != nil {
		return StoreStatus{}, err
	}
	return DescribeStore()
}

// LockStore discards the open data key immediately, both in this process and
// for later ones.
func LockStore() error {
	encrypted, _, err := encryptedStoreForAdmin()
	if err != nil {
		return err
	}
	encrypted.lock()
	SetPassphrase("")
	return nil
}

// RewrapStore adds or refreshes the host-bound wrap of an existing store. It is
// how a host that gains a TPM starts using it: the data key does not change, so
// no stored value is re-encrypted and nothing can be lost.
func RewrapStore(passphrase string) (WrapInfo, error) {
	encrypted, _, err := encryptedStoreForAdmin()
	if err != nil {
		return WrapInfo{}, err
	}
	if passphrase != "" {
		SetPassphrase(passphrase)
	}
	wrap, err := encrypted.addWrap(newHostBoundProvider())
	if err != nil {
		return WrapInfo{}, err
	}
	return WrapInfo{Provider: wrap.Provider, KeyStore: wrap.KeyStore}, nil
}

// ChangePassphraseStore replaces the passphrase wrap around the existing data
// key. It first opens the store with the current passphrase, so a wrong
// current value leaves the file untouched. Stored credential entries are not
// read or re-encrypted.
func ChangePassphraseStore(current, next string) error {
	current = strings.TrimSpace(current)
	next = strings.TrimSpace(next)
	if current == "" || next == "" {
		return fmt.Errorf("current and new credential store passphrases are required")
	}
	encrypted, _, err := encryptedStoreForAdmin()
	if err != nil {
		return err
	}
	// Validate the supplied current passphrase against its own wrap. The
	// normal store chain may also have a host-bound wrap; using it here would
	// let a typo pass and would violate the command's promise that the current
	// passphrase is required before rotation.
	currentStore := newEncryptedStore(encrypted.path, passphraseProvider{passphrase: current})
	currentStore.cache = noUnlockCache{}
	if _, _, err := currentStore.open(); err != nil {
		currentStore.lock()
		return err
	}
	currentStore.lock()
	SetPassphrase(current)
	if _, err := encrypted.addWrap(passphraseProvider{passphrase: next}); err != nil {
		encrypted.lock()
		SetPassphrase("")
		return err
	}
	// The old cache fingerprint must not survive the wrap replacement. Locking
	// also zeroes the in-process data key before the command returns.
	encrypted.lock()
	SetPassphrase("")
	return nil
}
