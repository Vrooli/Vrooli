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
	"github.com/vrooli/vrooli/internal/secrets"
)

type (
	Identity = secrets.Identity
	Status   = secrets.Status
)

var ParseIdentity = secrets.ParseIdentity

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
