package mocks

import (
	"context"
	"fmt"

	"device-sync-hub/internal/auth"
	"device-sync-hub/internal/devices"
)

// FakeSecrets is a deterministic devices.Secrets for tests: it hands out
// predictable tokens/codes so a test can assert the exact raw value returned to
// a caller and the exact hash persisted. Each call increments a counter.
type FakeSecrets struct {
	TokenSeq int
	CodeSeq  int

	TokenErr error
	CodeErr  error
}

// Compile-time guarantee.
var _ devices.Secrets = (*FakeSecrets)(nil)

func (f *FakeSecrets) DeviceToken() (string, error) {
	if f.TokenErr != nil {
		return "", f.TokenErr
	}
	f.TokenSeq++
	return fmt.Sprintf("test-token-%d", f.TokenSeq), nil
}

func (f *FakeSecrets) PairingCode() (string, error) {
	if f.CodeErr != nil {
		return "", f.CodeErr
	}
	f.CodeSeq++
	return fmt.Sprintf("CODE%d-AAAAA", f.CodeSeq), nil
}

// FakeAuth is a devices-test double for auth.Validator. Only RevokeSession is
// exercised by the devices service (Validate belongs to the middleware tests),
// but both are implemented so the seam is satisfied.
type FakeAuth struct {
	RevokeErr  error
	RevokedIDs []string
}

// Compile-time guarantee.
var _ auth.Validator = (*FakeAuth)(nil)

func (f *FakeAuth) Validate(context.Context, string) (auth.Identity, error) {
	return auth.Identity{}, auth.ErrUnauthenticated
}

func (f *FakeAuth) RevokeSession(_ context.Context, sessionID string) error {
	f.RevokedIDs = append(f.RevokedIDs, sessionID)
	return f.RevokeErr
}
