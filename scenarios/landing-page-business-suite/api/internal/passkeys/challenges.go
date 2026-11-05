package passkeys

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Challenge struct {
	ID, Purpose, Subject, BindingHash string
	Value                             string
	SessionData                       json.RawMessage
	ExpiresAt                         time.Time
}

// EncodeSessionData serializes the verifier state for a durable ceremony.
func EncodeSessionData(value any) (json.RawMessage, error) { return json.Marshal(value) }

func Hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// ChallengeWriter is deliberately narrow so ceremony code cannot update a
// challenge without going through the durable store's one-use contract.
type ChallengeWriter interface {
	Create(context.Context, Challenge) error
	Consume(context.Context, string, string, string) (Challenge, error)
	Purge(context.Context) error
}

type Repository struct {
	DB interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
		QueryRowContext(context.Context, string, ...any) *sql.Row
	}
}

func NewChallengeID() string { return uuid.NewString() }

func (r *Repository) Create(ctx context.Context, c Challenge) error {
	if r == nil || r.DB == nil {
		return fmt.Errorf("challenge store unavailable")
	}
	if c.ExpiresAt.IsZero() {
		c.ExpiresAt = time.Now().UTC().Add(5 * time.Minute)
	}
	if c.ID == "" {
		c.ID = NewChallengeID()
	}
	_, err := r.DB.ExecContext(ctx, `INSERT INTO webauthn_challenges (id, challenge_hash, purpose, subject, session_data, binding_hash, expires_at) VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7)`, c.ID, Hash(c.Value), c.Purpose, c.Subject, string(c.SessionData), c.BindingHash, c.ExpiresAt)
	return err
}

func (r *Repository) Purge(ctx context.Context) error {
	if r == nil || r.DB == nil {
		return fmt.Errorf("challenge store unavailable")
	}
	_, err := r.DB.ExecContext(ctx, `DELETE FROM webauthn_challenges WHERE expires_at < NOW() - INTERVAL '1 day' OR consumed_at IS NOT NULL AND consumed_at < NOW() - INTERVAL '1 day'`)
	return err
}

func (r *Repository) Consume(ctx context.Context, id, purpose, bindingHash string) (Challenge, error) {
	if r == nil || r.DB == nil {
		return Challenge{}, fmt.Errorf("challenge store unavailable")
	}
	var c Challenge
	var session []byte
	err := r.DB.QueryRowContext(ctx, `UPDATE webauthn_challenges SET consumed_at = NOW() WHERE id = $1 AND purpose = $2 AND binding_hash = $3 AND consumed_at IS NULL AND expires_at > NOW() RETURNING challenge_hash, subject, session_data, expires_at`, id, purpose, bindingHash).Scan(&c.ID, &c.Subject, &session, &c.ExpiresAt)
	if err != nil {
		return Challenge{}, err
	}
	c.Purpose, c.BindingHash, c.SessionData = purpose, bindingHash, session
	return c, nil
}

func NormalizeBinding(binding string) string { return strings.TrimSpace(binding) }
