package securestore

import (
	"fmt"
	"strings"
)

// UnattendedStatus answers whether this host opens its credential store with
// no human action, and what had to change to make that true.
type UnattendedStatus struct {
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider,omitempty"`
	KeyStore string `json:"key_store,omitempty"`
	Added    bool   `json:"added,omitempty"`
	Repaired bool   `json:"repaired,omitempty"`
	Blocked  string `json:"blocked,omitempty"`
}

// unattendedProviders are the wraps that need no human at boot, strongest
// first. The passphrase provider remains recovery-only and is excluded.
var unattendedProviders = func() []keyProvider {
	providers := defaultKeyProviders()
	unattended := make([]keyProvider, 0, len(providers))
	for _, provider := range providers {
		if provider.Name() != providerPassphrase {
			unattended = append(unattended, provider)
		}
	}
	return unattended
}

func inspectUnattendedWrap(file *sealedFile) UnattendedStatus {
	var reasons []string
	for _, provider := range unattendedProviders() {
		wrap, found := file.wrapFor(provider.Name())
		if !found {
			if _, err := provider.Available(); err != nil {
				reasons = append(reasons, provider.Name()+": "+conciseReason(err))
				continue
			}
			reasons = append(reasons, provider.Name()+": this host supports it but the store has no such wrap yet")
			continue
		}
		if _, err := provider.Unwrap(wrap); err != nil {
			reasons = append(reasons, provider.Name()+": the store has this wrap and it no longer opens: "+conciseReason(err))
			continue
		}
		return UnattendedStatus{Enabled: true, Provider: wrap.Provider, KeyStore: wrap.KeyStore}
	}
	return UnattendedStatus{Blocked: strings.Join(reasons, "; ")}
}

// EnsureUnattendedWrap is the single decision seam for whether a reboot needs
// a human. It never regenerates the data key or re-encrypts stored values.
func EnsureUnattendedWrap(passphrase string) (UnattendedStatus, error) {
	encrypted, _, err := encryptedStoreForAdmin()
	if err != nil {
		return UnattendedStatus{}, err
	}
	if !encrypted.initialized() {
		return UnattendedStatus{}, fmt.Errorf("%w: no credential store on this host; run `vrooli credentials store init`", ErrAbsent)
	}
	if passphrase != "" {
		SetPassphrase(passphrase)
	}
	return ensureUnattendedWrap(encrypted)
}

func ensureUnattendedWrap(encrypted *encryptedStore) (UnattendedStatus, error) {
	file, err := readSealedFile(encrypted.path)
	if err != nil {
		return UnattendedStatus{}, encrypted.classifyFileError(err)
	}
	if status := inspectUnattendedWrap(file); status.Enabled {
		return status, nil
	}

	var reasons []string
	for _, provider := range unattendedProviders() {
		if _, err := provider.Available(); err != nil {
			reasons = append(reasons, provider.Name()+": "+conciseReason(err))
			continue
		}
		_, replacing := file.wrapFor(provider.Name())
		wrap, addErr := encrypted.addWrap(provider)
		if addErr != nil {
			reasons = append(reasons, provider.Name()+": "+conciseReason(addErr))
			continue
		}
		return UnattendedStatus{Enabled: true, Provider: wrap.Provider, KeyStore: wrap.KeyStore, Added: !replacing, Repaired: replacing}, nil
	}
	return UnattendedStatus{Blocked: strings.Join(reasons, "; ")}, nil
}

// RewrapStore adds or refreshes an unattended wrap around the existing data
// key. Stored credential values are not read or re-encrypted.
func RewrapStore(passphrase string) (WrapInfo, error) {
	status, err := EnsureUnattendedWrap(passphrase)
	if err != nil {
		return WrapInfo{}, err
	}
	if !status.Enabled {
		return WrapInfo{}, fmt.Errorf("%w: no unattended key wrap can protect this store (%s)", errKeyProviderUnavailable, status.Blocked)
	}
	return WrapInfo{Provider: status.Provider, KeyStore: status.KeyStore}, nil
}
