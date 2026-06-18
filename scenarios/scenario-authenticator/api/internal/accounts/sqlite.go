package accounts

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"scenario-authenticator/internal/clock"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface the repository depends on. Both
// *sql.DB (unit tests) and *database.RoutedDB (production) satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// timeFormat matches the notes domain so all timestamps round-trip identically.
const timeFormat = time.RFC3339Nano

type sqliteRepository struct {
	db    SQLExecutor
	clock clock.Clock
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db SQLExecutor, clk clock.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

var _ Repository = (*sqliteRepository)(nil)

func (s *sqliteRepository) Create(ctx context.Context, in CreateInput) (Account, error) {
	now := s.clock.Now().UTC()
	acc := Account{
		ID:            uuid.NewString(),
		RealmID:       in.RealmID,
		Email:         in.Email,
		Username:      in.Username,
		Roles:         in.Roles,
		EmailVerified: false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if len(acc.Roles) == 0 {
		acc.Roles = []string{"user"}
	}
	rolesJSON, err := json.Marshal(acc.Roles)
	if err != nil {
		return Account{}, fmt.Errorf("marshal roles: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO accounts (id, realm_id, email, username, password_hash, roles, email_verified,
                      failed_login_attempts, locked_until, created_at, updated_at, last_login)
VALUES (?, ?, ?, ?, ?, ?, 0, 0, '', ?, ?, '')`,
		acc.ID, acc.RealmID, acc.Email, acc.Username, in.PasswordHash, string(rolesJSON),
		acc.CreatedAt.Format(timeFormat), acc.UpdatedAt.Format(timeFormat),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return Account{}, ErrEmailTaken
		}
		return Account{}, fmt.Errorf("insert account: %w", err)
	}
	return acc, nil
}

const selectAccountCols = `id, realm_id, email, username, roles, email_verified,
       failed_login_attempts, locked_until, created_at, updated_at, last_login`

func (s *sqliteRepository) FindByEmail(ctx context.Context, realmID, email string) (Account, string, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT `+selectAccountCols+`, password_hash
FROM accounts WHERE realm_id = ? AND email = ?`, realmID, email)
	acc, hash, err := scanAccountWithHash(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Account{}, "", ErrAccountNotFound
	}
	if err != nil {
		return Account{}, "", fmt.Errorf("find account by email: %w", err)
	}
	return acc, hash, nil
}

func (s *sqliteRepository) FindByID(ctx context.Context, id string) (Account, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT `+selectAccountCols+` FROM accounts WHERE id = ?`, id)
	acc, err := scanAccount(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Account{}, ErrAccountNotFound
	}
	if err != nil {
		return Account{}, fmt.Errorf("find account by id: %w", err)
	}
	return acc, nil
}

func (s *sqliteRepository) SetLoginSuccess(ctx context.Context, id string, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE accounts
SET failed_login_attempts = 0, locked_until = '', last_login = ?, updated_at = ?
WHERE id = ?`, now.UTC().Format(timeFormat), now.UTC().Format(timeFormat), id)
	if err != nil {
		return fmt.Errorf("set login success: %w", err)
	}
	return nil
}

func (s *sqliteRepository) SetLoginFailure(ctx context.Context, id string, attempts int, lockedUntil time.Time) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE accounts
SET failed_login_attempts = ?, locked_until = ?, updated_at = ?
WHERE id = ?`, attempts, formatNullable(lockedUntil), s.clock.Now().UTC().Format(timeFormat), id)
	if err != nil {
		return fmt.Errorf("set login failure: %w", err)
	}
	return nil
}

func (s *sqliteRepository) RealmAudience(ctx context.Context, realmID string) (string, error) {
	var aud string
	err := s.db.QueryRowContext(ctx, `SELECT audience FROM realms WHERE id = ?`, realmID).Scan(&aud)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrRealmNotFound
	}
	if err != nil {
		return "", fmt.Errorf("realm audience: %w", err)
	}
	return aud, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAccount(sc rowScanner) (Account, error) {
	return scanAccountInto(sc, nil)
}

func scanAccountWithHash(sc rowScanner) (Account, string, error) {
	var hash string
	acc, err := scanAccountInto(sc, &hash)
	return acc, hash, err
}

// scanAccountInto scans the selectAccountCols columns into an Account; when
// hashDest is non-nil it expects a trailing password_hash column.
func scanAccountInto(sc rowScanner, hashDest *string) (Account, error) {
	var (
		acc          Account
		rolesJSON    string
		verifiedInt  int
		lockedRaw    string
		createdRaw   string
		updatedRaw   string
		lastLoginRaw string
	)
	dest := []any{
		&acc.ID, &acc.RealmID, &acc.Email, &acc.Username, &rolesJSON, &verifiedInt,
		&acc.FailedLoginAttempts, &lockedRaw, &createdRaw, &updatedRaw, &lastLoginRaw,
	}
	if hashDest != nil {
		dest = append(dest, hashDest)
	}
	if err := sc.Scan(dest...); err != nil {
		return Account{}, err
	}
	acc.EmailVerified = verifiedInt != 0
	if rolesJSON != "" {
		if err := json.Unmarshal([]byte(rolesJSON), &acc.Roles); err != nil {
			acc.Roles = []string{"user"}
		}
	}
	var err error
	if acc.CreatedAt, err = parseTime(createdRaw); err != nil {
		return Account{}, fmt.Errorf("parse created_at: %w", err)
	}
	if acc.UpdatedAt, err = parseTime(updatedRaw); err != nil {
		return Account{}, fmt.Errorf("parse updated_at: %w", err)
	}
	if acc.LockedUntil, err = parseNullableTime(lockedRaw); err != nil {
		return Account{}, fmt.Errorf("parse locked_until: %w", err)
	}
	if acc.LastLogin, err = parseNullableTime(lastLoginRaw); err != nil {
		return Account{}, fmt.Errorf("parse last_login: %w", err)
	}
	return acc, nil
}

func parseTime(raw string) (time.Time, error) {
	return time.Parse(timeFormat, raw)
}

func parseNullableTime(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, nil
	}
	return time.Parse(timeFormat, raw)
}

func formatNullable(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(timeFormat)
}

// isUniqueViolation reports whether err is a SQLite UNIQUE constraint failure.
// modernc.org/sqlite surfaces these as a message containing "UNIQUE constraint
// failed"; we match on the text to stay driver-agnostic.
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(strings.ToUpper(err.Error()), "UNIQUE CONSTRAINT")
}
