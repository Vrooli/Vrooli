package administration

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"strings"
	"time"
)

const ScopeMetricsRead = "metrics:read"

var ErrReaderTokenRejected = errors.New("reader token rejected")

type SQLStore interface {
	Exec(string, ...any) (sql.Result, error)
	QueryRow(string, ...any) *sql.Row
	Query(string, ...any) (*sql.Rows, error)
}

func (s *ReaderTokens) Revoke(ctx context.Context, id string) error {
	_ = ctx
	_, err := s.db.Exec(`UPDATE reader_tokens SET revoked_at=$2 WHERE id=$1 AND revoked_at IS NULL`, strings.TrimSpace(id), s.now().UTC())
	return err
}
func (s *ReaderTokens) List(ctx context.Context) ([]ReaderToken, error) {
	_ = ctx
	rows, err := s.db.Query(`SELECT id,label,scope,prefix,source,created_by,created_at,last_used_at,revoked_at FROM reader_tokens ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ReaderToken
	for rows.Next() {
		var t ReaderToken
		if err := rows.Scan(&t.ID, &t.Label, &t.Scope, &t.Prefix, &t.Source, &t.CreatedBy, &t.CreatedAt, &t.LastUsedAt, &t.RevokedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ReaderTokens is the persistence owner. Only SHA-256 digests are stored.
type ReaderTokens struct {
	db  SQLStore
	now func() time.Time
}

func NewReaderTokens(db SQLStore) *ReaderTokens { return &ReaderTokens{db: db, now: time.Now} }

func MintReaderToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return "lpbs_rt_" + base64.RawURLEncoding.EncodeToString(raw), nil
}

type IssuedReaderToken struct {
	ID, Label, Scope, Prefix, Token string
	CreatedAt                       time.Time
}
type ReaderToken struct {
	ID, Label, Scope, Prefix, Source string
	CreatedBy                        *int64
	CreatedAt                        time.Time
	LastUsedAt, RevokedAt            *time.Time
}

func (s *ReaderTokens) Issue(ctx context.Context, label, scope string) (*IssuedReaderToken, error) {
	_ = ctx
	label = strings.TrimSpace(label)
	if label == "" {
		return nil, fmt.Errorf("label is required")
	}
	if scope == "" {
		scope = ScopeMetricsRead
	}
	if scope != ScopeMetricsRead {
		return nil, fmt.Errorf("unsupported scope %q", scope)
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, err
	}
	token := "lpbs_rt_" + base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(token))
	id := uuid.New()
	now := s.now().UTC()
	if _, err := s.db.Exec(`INSERT INTO reader_tokens (id,label,scope,token_sha256,prefix,source,created_at) VALUES ($1,$2,$3,$4,$5,'operator',$6)`, id.String(), label, scope, hex.EncodeToString(sum[:]), token[:16], now); err != nil {
		return nil, fmt.Errorf("store reader token: %w", err)
	}
	return &IssuedReaderToken{ID: id.String(), Label: label, Scope: scope, Prefix: token[:16], Token: token, CreatedAt: now}, nil
}
func (s *ReaderTokens) Authenticate(ctx context.Context, token, scope string) error {
	_ = ctx
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	var revoked *time.Time
	var id string
	var foundScope string
	if err := s.db.QueryRow(`SELECT id,scope,revoked_at FROM reader_tokens WHERE token_sha256=$1`, hex.EncodeToString(sum[:])).Scan(&id, &foundScope, &revoked); err != nil || revoked != nil || foundScope != scope {
		return ErrReaderTokenRejected
	}
	_, _ = s.db.Exec(`UPDATE reader_tokens SET last_used_at=$2 WHERE id=$1 AND (last_used_at IS NULL OR last_used_at < $2 - INTERVAL '1 minute')`, id, s.now().UTC())
	return nil
}

func (s *ReaderTokens) UpsertAuthority(token string) error {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	_, err := s.db.Exec(`UPDATE reader_tokens SET revoked_at=NOW() WHERE source='authority' AND scope=$1 AND token_sha256 <> $2`, ScopeMetricsRead, hex.EncodeToString(sum[:]))
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO reader_tokens (id,label,scope,token_sha256,prefix,source) VALUES ($1,'command-center-local',$2,$3,$4,'authority') ON CONFLICT (token_sha256) DO UPDATE SET revoked_at=NULL`, uuid.New().String(), ScopeMetricsRead, hex.EncodeToString(sum[:]), token[:16])
	return err
}
