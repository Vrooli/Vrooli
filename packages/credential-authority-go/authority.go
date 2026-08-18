// Package credentialauthority provides the public runtime boundary for
// scenario consumers of Vrooli's credential authority. The implementation
// remains in internal/secrets; scenarios do not select a backend or read a
// plaintext file themselves.
//
// This is the Go binding. The directory carries a -go suffix because a
// scenario may be written in any language, and each language gets its own
// binding over the same authority rather than its own storage.
//
// Reach for this package whenever Vrooli itself authors the consuming code.
// The `env` field on a credential descriptor exists for the other case — a
// process Vrooli does not author, such as a database container or a
// third-party CLI, which can only receive a value through its environment.
// Resolving here instead keeps the value out of the process environment,
// where it would be readable at /proc/<pid>/environ and inherited by every
// subprocess the scenario spawns.
package credentialauthority

import (
	"fmt"
	"path/filepath"

	"github.com/vrooli/vrooli/internal/resources/securestore"
	"github.com/vrooli/vrooli/internal/secrets"
)

type (
	Identity         = secrets.Identity
	Status           = secrets.Status
	KeyringReport    = securestore.KeyringReport
	RecoveryEntry    = secrets.RecoveryEntry
	RecoveryManifest = secrets.RecoveryManifest
	RecoveryReceipt  = secrets.RecoveryReceipt
)

var ParseIdentity = secrets.ParseIdentity

var (
	InspectRecovery      = secrets.InspectRecovery
	ReadRecoveryReceipt  = secrets.ReadRecoveryReceipt
	WriteRecoveryReceipt = secrets.WriteRecoveryReceipt
)

// The failure taxonomy, re-exported because a consumer that cannot tell these
// apart cannot behave correctly: "not configured" is an operator omission to
// report, while a provider fault means the answer is unknown and the caller
// must not act as though the value is absent. Collapsing them is the original
// defect this whole seam exists to prevent.
var (
	// ErrUnconfigured: the store works and holds no value for this identity.
	ErrUnconfigured = secrets.ErrUnconfigured
	// ErrProviderUnavailable: the store exists but cannot be reached now.
	ErrProviderUnavailable = secrets.ErrProviderUnavailable
	// ErrProviderAbsent: this host has no credential backend at all.
	ErrProviderAbsent = secrets.ErrProviderAbsent
)

type Authority struct{ inner *secrets.Authority }

func NewAuthority(store securestore.Store) (*Authority, error) {
	inner, err := secrets.NewAuthority(store)
	if err != nil {
		return nil, err
	}
	return &Authority{inner: inner}, nil
}

func Default() (*Authority, error) {
	inner, err := secrets.DefaultAuthority()
	if err != nil {
		return nil, err
	}
	return &Authority{inner: inner}, nil
}

func (a *Authority) Resolve(identity Identity, field string) (string, error) {
	return a.inner.Resolve(identity, field)
}

func (a *Authority) Put(identity Identity, field, value string) error {
	return a.inner.Put(identity, field, value)
}

func (a *Authority) Delete(identity Identity, field string) error {
	return a.inner.Delete(identity, field)
}

func (a *Authority) Status(identity Identity, field string) Status {
	return a.inner.Status(identity, field)
}

func (a *Authority) Provider() string { return a.inner.Provider() }

func (a *Authority) Availability() error { return a.inner.Availability() }

func (a *Authority) KeyringInspect(path string) (KeyringReport, error) {
	path, err := firstKeyringPath(path)
	if err != nil {
		return KeyringReport{}, err
	}
	return securestore.InspectKeyringFile(path)
}

func (a *Authority) KeyringRepair(path string) (KeyringReport, error) {
	path, err := firstKeyringPath(path)
	if err != nil {
		return KeyringReport{}, err
	}
	return securestore.RepairKeyringFile(path)
}

func firstKeyringPath(path string) (string, error) {
	if path != "" {
		return path, nil
	}
	dir, err := securestore.DefaultKeyringDir()
	if err != nil {
		return "", err
	}
	matches, err := filepath.Glob(filepath.Join(dir, "*.keyring"))
	if err != nil {
		return "", fmt.Errorf("list keyring files: %w", err)
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no keyring files found")
	}
	return matches[0], nil
}

func (a *Authority) ExportRecovery(entries []RecoveryEntry, passphrase string) ([]byte, error) {
	return a.inner.ExportRecovery(entries, passphrase)
}

func (a *Authority) RestoreRecovery(bundle []byte, passphrase string) error {
	return a.inner.RestoreRecovery(bundle, passphrase)
}
