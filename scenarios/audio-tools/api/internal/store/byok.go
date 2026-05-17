package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// BYOKCredential is a stored credential row. The Cipher field holds the
// AES-GCM ciphertext; callers must round-trip through byokstore.Encryptor
// to extract the plaintext secret.
type BYOKCredential struct {
	ProviderID  string
	Capability  string // stt | tts | summarize
	SecretKind  string // "api_key"
	Cipher      []byte
	Fingerprint string
	CreatedAt   time.Time
	LastUsedAt  *time.Time
}

// BYOKStore persists encrypted credentials.
type BYOKStore struct{ db *sql.DB }

func NewBYOKStore(db *sql.DB) *BYOKStore { return &BYOKStore{db: db} }

// Upsert inserts or replaces the credential for (provider, capability).
func (s *BYOKStore) Upsert(ctx context.Context, c BYOKCredential) error {
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}
	if c.SecretKind == "" {
		c.SecretKind = "api_key"
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO byok_credentials(provider_id, capability, secret_kind, secret_cipher, fingerprint, created_at, last_used_at)
		VALUES (?,?,?,?,?,?,NULL)
		ON CONFLICT(provider_id, capability) DO UPDATE SET
			secret_kind=excluded.secret_kind,
			secret_cipher=excluded.secret_cipher,
			fingerprint=excluded.fingerprint,
			created_at=excluded.created_at,
			last_used_at=NULL
	`, c.ProviderID, c.Capability, c.SecretKind, c.Cipher, c.Fingerprint, c.CreatedAt.Format(time.RFC3339))
	return err
}

// Delete removes the credential for (provider, capability). Returns
// (false, nil) if no row matched.
func (s *BYOKStore) Delete(ctx context.Context, providerID, capability string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM byok_credentials WHERE provider_id=? AND capability=?`,
		providerID, capability)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// Get fetches the stored credential for (provider, capability).
// Returns (cred, true, nil) on hit, (_, false, nil) on miss.
func (s *BYOKStore) Get(ctx context.Context, providerID, capability string) (BYOKCredential, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT provider_id, capability, secret_kind, secret_cipher, fingerprint, created_at, last_used_at
		FROM byok_credentials WHERE provider_id=? AND capability=?`,
		providerID, capability)
	c, err := scanCredential(row)
	if errors.Is(err, sql.ErrNoRows) {
		return BYOKCredential{}, false, nil
	}
	if err != nil {
		return BYOKCredential{}, false, err
	}
	return c, true, nil
}

// List returns all stored credentials in stable order (provider, capability).
func (s *BYOKStore) List(ctx context.Context) ([]BYOKCredential, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT provider_id, capability, secret_kind, secret_cipher, fingerprint, created_at, last_used_at
		FROM byok_credentials ORDER BY provider_id, capability`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BYOKCredential
	for rows.Next() {
		c, err := scanCredential(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// MarkUsed updates last_used_at for (provider, capability) without
// touching the cipher.
func (s *BYOKStore) MarkUsed(ctx context.Context, providerID, capability string, t time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE byok_credentials SET last_used_at=? WHERE provider_id=? AND capability=?`,
		t.UTC().Format(time.RFC3339), providerID, capability)
	return err
}

// scanRow narrows database/sql.Row and Rows to a common Scan surface so
// we can share the parsing code.
type scanRow interface {
	Scan(dest ...any) error
}

func scanCredential(r scanRow) (BYOKCredential, error) {
	var (
		c          BYOKCredential
		createdAt  string
		lastUsedAt sql.NullString
	)
	if err := r.Scan(&c.ProviderID, &c.Capability, &c.SecretKind, &c.Cipher, &c.Fingerprint, &createdAt, &lastUsedAt); err != nil {
		return BYOKCredential{}, err
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if lastUsedAt.Valid {
		t, _ := time.Parse(time.RFC3339, lastUsedAt.String)
		c.LastUsedAt = &t
	}
	return c, nil
}
