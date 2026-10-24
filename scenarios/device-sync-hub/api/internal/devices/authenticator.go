package devices

import (
	"context"
	"errors"
	"strings"
)

// ErrUntrustedDevice is returned when a presented device token resolves to a
// device that exists but is not TRUSTED (PENDING awaiting approval, or
// REVOKED). Distinct from "unknown token" only internally — the handler edge
// maps both to the same unauthenticated response so a caller cannot probe which
// tokens exist.
var ErrUntrustedDevice = errors.New("device is not trusted")

// Authenticator resolves a raw hub device token to the TRUSTED device that owns
// it. It is the trust-enforcement seam the transfer and realtime layers depend
// on: every device-token-authed call resolves the caller here before acting.
//
// Declared in the devices package because device trust is this domain's
// responsibility — the token is hub-issued (not an authenticator JWT) and only
// this domain knows how it is hashed and what TRUSTED means.
type Authenticator interface {
	// Authenticate resolves rawToken to its device. It returns:
	//   - the TRUSTED device, nil          on success;
	//   - zero, ErrUntrustedDevice         when the token is unknown, blank, or
	//                                       resolves to a non-TRUSTED device;
	//   - zero, wrapped error              on a storage failure.
	// Unknown and untrusted are deliberately indistinguishable to the caller.
	Authenticate(ctx context.Context, rawToken string) (Device, error)
}

// tokenAuthenticator is the production Authenticator over the Repository.
type tokenAuthenticator struct {
	repo Repository
}

// NewAuthenticator constructs the production Authenticator. It shares the same
// Repository the service uses, so it sees committed device-trust transitions
// immediately (a revoke flips trust_state and the very next Authenticate fails).
func NewAuthenticator(repo Repository) Authenticator {
	return &tokenAuthenticator{repo: repo}
}

// Compile-time guarantee.
var _ Authenticator = (*tokenAuthenticator)(nil)

func (a *tokenAuthenticator) Authenticate(ctx context.Context, rawToken string) (Device, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return Device{}, ErrUntrustedDevice
	}
	d, err := a.repo.DeviceByToken(ctx, hashSecret(rawToken))
	if err != nil {
		var notFound ErrDeviceNotFound
		if errors.As(err, &notFound) {
			// Unknown token — same outcome as untrusted (no probing oracle).
			return Device{}, ErrUntrustedDevice
		}
		return Device{}, err
	}
	if d.TrustState != TrustTrusted {
		return Device{}, ErrUntrustedDevice
	}
	return d, nil
}
