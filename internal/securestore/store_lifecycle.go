package securestore

import (
	"errors"
	"fmt"
	"strings"
)

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
	if err := RepairCredentialStoreOwnership(); err != nil {
		return StoreStatus{}, err
	}
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
	return changePassphraseStore(encrypted, current, next)
}

// ErrPassphraseWrapExists refuses AddPassphraseWrap on a store that already
// has a passphrase wrap: replacing that one proves the current passphrase
// first, which is ChangePassphraseStore's job.
var ErrPassphraseWrapExists = errors.New("credential store already has a passphrase wrap; change it with `vrooli credentials store change-passphrase`")

// ErrNoPassphraseWrap reports a store that opens only through unattended
// wraps, so there is no passphrase to verify.
var ErrNoPassphraseWrap = errors.New("credential store has no passphrase wrap")

// AddPassphraseWrap gives a store that opens only through an unattended wrap
// (TPM, host key, Keychain, DPAPI) a passphrase wrap as well, so whoever holds
// the passphrase can still open it after that binding is lost. A Bridge
// control plane uses it to escrow a recovery passphrase for a node it
// manages. The store is opened through its existing wraps; the data key and
// every stored value are unchanged, and no existing wrap is touched.
func AddPassphraseWrap(next string) (PassphraseCheck, error) {
	next = strings.TrimSpace(next)
	if next == "" {
		return PassphraseCheck{}, fmt.Errorf("a new credential store passphrase is required")
	}
	encrypted, _, err := encryptedStoreForAdmin()
	if err != nil {
		return PassphraseCheck{}, err
	}
	return addPassphraseWrap(encrypted, next)
}

func addPassphraseWrap(encrypted *encryptedStore, next string) (PassphraseCheck, error) {
	if !encrypted.initialized() {
		return PassphraseCheck{}, fmt.Errorf("%w: no credential store on this host; run `vrooli credentials store init`", ErrAbsent)
	}
	file, err := readSealedFile(encrypted.path)
	if err != nil {
		return PassphraseCheck{}, encrypted.classifyFileError(err)
	}
	if _, found := file.wrapFor(providerPassphrase); found {
		return PassphraseCheck{}, ErrPassphraseWrapExists
	}
	wrap, err := encrypted.addWrap(passphraseProvider{passphrase: next})
	// Zero the opened data key, but leave the session unlock cache alone:
	// adding a recovery wrap is not a lock, and clearing it would end an
	// operator's unlocked session as a side effect.
	encrypted.forgetDataKey()
	if err != nil {
		return PassphraseCheck{}, err
	}
	generation, err := passphraseWrapGeneration(wrap)
	if err != nil {
		return PassphraseCheck{}, err
	}
	return PassphraseCheck{Valid: true, Generation: generation}, nil
}

// forgetDataKey zeroes this process's copy of the data key without touching
// the session unlock cache (compare lock).
func (store *encryptedStore) forgetDataKey() {
	store.mu.Lock()
	defer store.mu.Unlock()
	for index := range store.dataKey {
		store.dataKey[index] = 0
	}
	store.dataKey = nil
	store.openedBy = ""
	store.keyStore = ""
}

// PassphraseCheck is the non-secret answer to "does this passphrase open the
// store's passphrase wrap", with the wrap's rotation counter.
type PassphraseCheck struct {
	Valid      bool   `json:"valid"`
	Generation uint64 `json:"generation,omitempty"`
}

// VerifyPassphrase reports whether passphrase opens the store's own passphrase
// wrap. It consults no other wrap, no unlock cache, and no node unlock socket,
// so a working TPM or host-bound wrap cannot make a wrong passphrase look
// right, and it changes nothing on disk or in any cache.
func VerifyPassphrase(passphrase string) (PassphraseCheck, error) {
	passphrase = strings.TrimSpace(passphrase)
	if passphrase == "" {
		return PassphraseCheck{}, fmt.Errorf("a credential store passphrase is required")
	}
	encrypted, _, err := encryptedStoreForAdmin()
	if err != nil {
		return PassphraseCheck{}, err
	}
	return verifyPassphrase(encrypted, passphrase)
}

func verifyPassphrase(encrypted *encryptedStore, passphrase string) (PassphraseCheck, error) {
	if !encrypted.initialized() {
		return PassphraseCheck{}, fmt.Errorf("%w: no credential store on this host; run `vrooli credentials store init`", ErrAbsent)
	}
	file, err := readSealedFile(encrypted.path)
	if err != nil {
		return PassphraseCheck{}, encrypted.classifyFileError(err)
	}
	wrap, found := file.wrapFor(providerPassphrase)
	if !found {
		return PassphraseCheck{}, ErrNoPassphraseWrap
	}
	generation, err := passphraseWrapGeneration(wrap)
	if err != nil {
		return PassphraseCheck{}, err
	}
	dataKey, err := (passphraseProvider{passphrase: passphrase}).Unwrap(wrap)
	for index := range dataKey {
		dataKey[index] = 0
	}
	if errors.Is(err, errWrongPassphrase) {
		return PassphraseCheck{Valid: false, Generation: generation}, nil
	}
	if err != nil {
		return PassphraseCheck{}, err
	}
	return PassphraseCheck{Valid: true, Generation: generation}, nil
}

func changePassphraseStore(encrypted *encryptedStore, current, next string) error {
	// Validate the supplied current passphrase against its own wrap. The
	// normal store chain may also have a host-bound wrap; using it here would let
	// a typo pass and would violate the command's promise that the current
	// passphrase is required before rotation.
	currentStore := newEncryptedStore(encrypted.path, passphraseProvider{passphrase: current})
	currentStore.cache = noUnlockCache{}
	if _, _, err := currentStore.open(); err != nil {
		currentStore.lock()
		return err
	}
	currentStore.lock()
	file, err := readSealedFile(encrypted.path)
	if err != nil {
		return err
	}
	currentWrap, found := file.wrapFor(providerPassphrase)
	if !found {
		return fmt.Errorf("credential store has no passphrase wrap")
	}
	currentGeneration, err := passphraseWrapGeneration(currentWrap)
	if err != nil {
		return err
	}
	SetPassphrase(current)
	if _, err := encrypted.addWrap(passphraseProvider{passphrase: next, generation: currentGeneration + 1}); err != nil {
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
