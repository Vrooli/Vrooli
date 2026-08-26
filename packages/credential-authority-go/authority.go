// Package credentialauthority provides the public runtime boundary for
// scenario consumers of Vrooli's credential authority. The implementation
// remains in internal/credentialauthority; scenarios do not select a backend or read a
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

	internalcredentialauthority "github.com/vrooli/vrooli/internal/credentialauthority"
	"github.com/vrooli/vrooli/internal/resources/securestore"
)

type (
	Identity         = internalcredentialauthority.Identity
	Status           = internalcredentialauthority.Status
	KeyringReport    = securestore.KeyringReport
	RecoveryEntry    = internalcredentialauthority.RecoveryEntry
	RecoveryManifest = internalcredentialauthority.RecoveryManifest
	RecoveryReceipt  = internalcredentialauthority.RecoveryReceipt
)

var ParseIdentity = internalcredentialauthority.ParseIdentity

var (
	InspectRecovery      = internalcredentialauthority.InspectRecovery
	ReadRecoveryReceipt  = internalcredentialauthority.ReadRecoveryReceipt
	WriteRecoveryReceipt = internalcredentialauthority.WriteRecoveryReceipt
)

// The failure taxonomy, re-exported because a consumer that cannot tell these
// apart cannot behave correctly: "not configured" is an operator omission to
// report, while a provider fault means the answer is unknown and the caller
// must not act as though the value is absent. Collapsing them is the original
// defect this whole seam exists to prevent.
var (
	// ErrUnconfigured: the store works and holds no value for this identity.
	ErrUnconfigured = internalcredentialauthority.ErrUnconfigured
	// ErrProviderUnavailable: the store exists but cannot be reached now.
	ErrProviderUnavailable = internalcredentialauthority.ErrProviderUnavailable
	// ErrProviderAbsent: this host has no credential backend at all.
	ErrProviderAbsent = internalcredentialauthority.ErrProviderAbsent
)

type Authority struct {
	inner *internalcredentialauthority.Authority
}

func NewAuthority(store securestore.Store) (*Authority, error) {
	inner, err := internalcredentialauthority.NewAuthority(store)
	if err != nil {
		return nil, err
	}
	return &Authority{inner: inner}, nil
}

// Unavailable returns an authority whose backend reports that this host has no
// credential store.
//
// It exists because that condition cannot be produced on a working machine, and
// a consumer that cannot exercise it cannot prove it degrades correctly. Reads
// and writes fail with ErrProviderAbsent; metadata paths that do not need a
// value, such as reading a recovery receipt, still work.
func Unavailable(reason string) (*Authority, error) {
	return NewAuthority(securestore.Absent(reason))
}

func Default() (*Authority, error) {
	inner, err := internalcredentialauthority.DefaultAuthority()
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

// Recheck discards the cached availability verdict. A long-lived scenario
// process calls it once per request so a store that was unlocked after the
// process started is observed, rather than requiring a restart.
func (a *Authority) Recheck() { a.inner.Recheck() }

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
