package accounts

import (
	"context"
	"errors"
	"time"
)

// Account is the domain shape of a stored account. Never carries the password
// hash — credential material is returned only by FindByEmail, separately, so it
// never rides a response payload by accident.
type Account struct {
	ID                  string
	RealmID             string
	Email               string
	Username            string
	Roles               []string
	Scopes              []string
	EmailVerified       bool
	FailedLoginAttempts int
	LockedUntil         time.Time // zero when not locked
	CreatedAt           time.Time
	UpdatedAt           time.Time
	LastLogin           time.Time // zero when never logged in
}

// Locked reports whether the account is locked at time now.
func (a Account) Locked(now time.Time) bool {
	return !a.LockedUntil.IsZero() && now.Before(a.LockedUntil)
}

// CreateInput is the data needed to persist a new account. PasswordHash is the
// argon2id PHC string; the repository never sees plaintext.
type CreateInput struct {
	RealmID      string
	Email        string
	Username     string
	PasswordHash string
	Roles        []string
}

// Sentinel errors the service maps to Connect codes.
var (
	// ErrAccountNotFound — no account matched the lookup.
	ErrAccountNotFound = errors.New("account not found")
	// ErrEmailTaken — an account with that email already exists in the realm.
	ErrEmailTaken = errors.New("email already registered")
	// ErrRealmNotFound — the named realm does not exist.
	ErrRealmNotFound = errors.New("realm not found")
)

// Repository is the persistence seam for accounts + realms. Production wires the
// SQLite impl; tests substitute a fake or a real temp-file SQLite handle.
type Repository interface {
	// Create persists a new account, returning ErrEmailTaken on a duplicate
	// (realm_id, email).
	Create(ctx context.Context, in CreateInput) (Account, error)
	// FindByEmail returns the account and its stored password hash, or
	// ErrAccountNotFound.
	FindByEmail(ctx context.Context, realmID, email string) (Account, string, error)
	// FindByID returns the account by id, or ErrAccountNotFound.
	FindByID(ctx context.Context, id string) (Account, error)
	// SetLoginSuccess resets the failure counter + lock and records last_login.
	SetLoginSuccess(ctx context.Context, id string, now time.Time) error
	// SetLoginFailure records the new failed-attempt count and lock expiry
	// (zero lockedUntil clears the lock).
	SetLoginFailure(ctx context.Context, id string, attempts int, lockedUntil time.Time) error
	// UpdatePasswordHash replaces the stored Argon2id password hash.
	UpdatePasswordHash(ctx context.Context, id, passwordHash string) error
	// RealmAudience returns the aud string for a realm, or ErrRealmNotFound.
	RealmAudience(ctx context.Context, realmID string) (string, error)
}
