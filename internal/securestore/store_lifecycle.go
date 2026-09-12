package securestore

import (
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
