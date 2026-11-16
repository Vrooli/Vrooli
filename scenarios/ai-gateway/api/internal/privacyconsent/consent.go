// Package privacyconsent owns short-lived, revocable authorization to send
// CONFIDENTIAL material to one selected remote source. Receipts are bearer
// secrets: only their digest is persisted and callers never choose a subject.
package privacyconsent

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/vrooli/api-core/identity"
)

var ErrInvalidGrant = errors.New("remote confidential consent is missing, invalid, expired, revoked, or bound to another source")

const (
	MaxLifetime = 90 * 24 * time.Hour
	MinLifetime = time.Minute
)

type Grant struct {
	ID        string     `json:"id"`
	Source    string     `json:"source"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

// Store is the only authority for grant creation, revocation, and checking.
// `user` must come from a verified identity principal, never request JSON.
type Store struct {
	db  *sql.DB
	now func() time.Time
}

func New(db *sql.DB, clock func() time.Time) (*Store, error) {
	if db == nil {
		return nil, errors.New("consent database is required")
	}
	if clock == nil {
		clock = time.Now
	}
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS ai_remote_consent_grants (
		grant_id TEXT PRIMARY KEY,
		receipt_digest TEXT NOT NULL UNIQUE,
		user_subject TEXT NOT NULL,
		remote_source TEXT NOT NULL,
		created_at TEXT NOT NULL,
		expires_at TEXT NOT NULL,
		revoked_at TEXT NOT NULL DEFAULT ''
	)`)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS ai_remote_consent_user_source ON ai_remote_consent_grants(user_subject, remote_source, revoked_at, expires_at)`)
	if err != nil {
		return nil, err
	}
	return &Store{db: db, now: clock}, nil
}

func normalize(user, source string) (string, string, error) {
	user = strings.TrimSpace(user)
	source = strings.ToLower(strings.TrimSpace(source))
	if user == "" || len(user) > 512 {
		return "", "", errors.New("verified user subject is required")
	}
	switch source {
	case "own-key", "vrooli-credits":
	default:
		return "", "", errors.New("consent source must be own-key or vrooli-credits")
	}
	return user, source, nil
}

func (s *Store) Grant(ctx context.Context, user, source string, lifetime time.Duration) (string, Grant, error) {
	user, source, err := normalize(user, source)
	if err != nil {
		return "", Grant{}, err
	}
	if lifetime < MinLifetime || lifetime > MaxLifetime {
		return "", Grant{}, errors.New("consent lifetime must be between one minute and ninety days")
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", Grant{}, err
	}
	receipt := base64.RawURLEncoding.EncodeToString(secret)
	digest := sha256.Sum256([]byte(receipt))
	now := s.now().UTC()
	id := make([]byte, 18)
	if _, err := rand.Read(id); err != nil {
		return "", Grant{}, err
	}
	grant := Grant{ID: base64.RawURLEncoding.EncodeToString(id), Source: source, CreatedAt: now, ExpiresAt: now.Add(lifetime)}
	_, err = s.db.ExecContext(ctx, `INSERT INTO ai_remote_consent_grants(grant_id, receipt_digest, user_subject, remote_source, created_at, expires_at) VALUES (?, ?, ?, ?, ?, ?)`, grant.ID, encodeDigest(digest[:]), user, source, now.Format(time.RFC3339Nano), grant.ExpiresAt.Format(time.RFC3339Nano))
	if err != nil {
		return "", Grant{}, err
	}
	return receipt, grant, nil
}

func (s *Store) Verify(ctx context.Context, receipt, source string) error {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok || !principal.IsHuman() || strings.TrimSpace(principal.Subject) == "" {
		return ErrInvalidGrant
	}
	if len(receipt) < 32 || len(receipt) > 128 {
		return ErrInvalidGrant
	}
	source = strings.ToLower(strings.TrimSpace(source))
	if source != "own-key" && source != "vrooli-credits" {
		return ErrInvalidGrant
	}
	digest := sha256.Sum256([]byte(receipt))
	var dbSource, expires, revoked string
	err := s.db.QueryRowContext(ctx, `SELECT remote_source, expires_at, revoked_at FROM ai_remote_consent_grants WHERE receipt_digest=? AND user_subject=?`, encodeDigest(digest[:]), principal.Subject).Scan(&dbSource, &expires, &revoked)
	if err != nil || dbSource != source || revoked != "" {
		return ErrInvalidGrant
	}
	expiry, err := time.Parse(time.RFC3339Nano, expires)
	if err != nil || !s.now().Before(expiry) {
		return ErrInvalidGrant
	}
	return nil
}

func (s *Store) Revoke(ctx context.Context, user, id string) error {
	user = strings.TrimSpace(user)
	id = strings.TrimSpace(id)
	if user == "" || id == "" {
		return errors.New("verified user subject and grant id are required")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE ai_remote_consent_grants SET revoked_at=? WHERE grant_id=? AND user_subject=? AND revoked_at=''`, s.now().UTC().Format(time.RFC3339Nano), id, user)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) List(ctx context.Context, user string) ([]Grant, error) {
	user = strings.TrimSpace(user)
	if user == "" {
		return nil, errors.New("verified user subject is required")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT grant_id, remote_source, created_at, expires_at, revoked_at FROM ai_remote_consent_grants WHERE user_subject=? ORDER BY created_at DESC`, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Grant
	for rows.Next() {
		var g Grant
		var created, expires, revoked string
		if err := rows.Scan(&g.ID, &g.Source, &created, &expires, &revoked); err != nil {
			return nil, err
		}
		g.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return nil, err
		}
		g.ExpiresAt, err = time.Parse(time.RFC3339Nano, expires)
		if err != nil {
			return nil, err
		}
		if revoked != "" {
			var revokedAt time.Time
			revokedAt, err = time.Parse(time.RFC3339Nano, revoked)
			if err != nil {
				return nil, err
			}
			g.RevokedAt = &revokedAt
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func encodeDigest(digest []byte) string { return base64.RawURLEncoding.EncodeToString(digest) }
