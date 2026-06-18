package pairing

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"vrooli-bridge/internal/clock"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface the repository depends on; both
// *sql.DB (tests) and *database.RoutedDB (production) satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db    SQLExecutor
	clock clock.Clock
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db SQLExecutor, clk clock.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

var _ Repository = (*sqliteRepository)(nil)

// timeFormat sorts lexicographically in time order, matching the registry
// domain and the wire format.
const timeFormat = time.RFC3339Nano

func (s *sqliteRepository) CreateCode(ctx context.Context, c PairingCode) (PairingCode, error) {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = s.clock.Now().UTC()
	}
	scopes, err := marshalStrings(c.Scopes)
	if err != nil {
		return PairingCode{}, fmt.Errorf("encode scopes: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO pairing_codes (id, code_hash, name, scopes, created_at, expires_at, redeemed_at, redeemed_node_id)
		 VALUES (?, ?, ?, ?, ?, ?, '', '')`,
		c.ID, c.CodeHash, c.Name, scopes,
		c.CreatedAt.Format(timeFormat), c.ExpiresAt.UTC().Format(timeFormat),
	)
	if err != nil {
		return PairingCode{}, fmt.Errorf("insert pairing code: %w", err)
	}
	return c, nil
}

func (s *sqliteRepository) GetCodeByHash(ctx context.Context, codeHash string) (PairingCode, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, code_hash, name, scopes, created_at, expires_at, redeemed_at, redeemed_node_id
		 FROM pairing_codes WHERE code_hash = ?`, codeHash)
	c, err := scanCode(row)
	if errors.Is(err, sql.ErrNoRows) {
		return PairingCode{}, ErrCodeNotFound
	}
	if err != nil {
		return PairingCode{}, fmt.Errorf("get pairing code: %w", err)
	}
	return c, nil
}

func (s *sqliteRepository) BurnCode(ctx context.Context, id, nodeID string) error {
	now := s.clock.Now().UTC().Format(timeFormat)
	// The redeemed_at = '' guard makes this the single-use gate: a second
	// concurrent redeem affects zero rows and is rejected.
	res, err := s.db.ExecContext(ctx,
		`UPDATE pairing_codes SET redeemed_at = ?, redeemed_node_id = ? WHERE id = ? AND redeemed_at = ''`,
		now, nodeID, id)
	if err != nil {
		return fmt.Errorf("burn pairing code: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("burn pairing code rows: %w", err)
	}
	if n == 0 {
		return ErrCodeUsed
	}
	return nil
}

func (s *sqliteRepository) StoreCredential(ctx context.Context, c Credential) error {
	if c.CreatedAt.IsZero() {
		c.CreatedAt = s.clock.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO node_credentials (node_id, public_key, created_at, revoked_at)
		 VALUES (?, ?, ?, '')
		 ON CONFLICT(node_id) DO UPDATE SET public_key = excluded.public_key, created_at = excluded.created_at, revoked_at = ''`,
		c.NodeID, c.PublicKey, c.CreatedAt.Format(timeFormat))
	if err != nil {
		return fmt.Errorf("store credential: %w", err)
	}
	return nil
}

func (s *sqliteRepository) RevokeCredential(ctx context.Context, nodeID string) error {
	now := s.clock.Now().UTC().Format(timeFormat)
	_, err := s.db.ExecContext(ctx,
		`UPDATE node_credentials SET revoked_at = ? WHERE node_id = ? AND revoked_at = ''`, now, nodeID)
	if err != nil {
		return fmt.Errorf("revoke credential: %w", err)
	}
	return nil
}

func (s *sqliteRepository) ActivePublicKey(ctx context.Context, nodeID string) (ed25519.PublicKey, bool, error) {
	var pubB64, revokedRaw string
	err := s.db.QueryRowContext(ctx,
		`SELECT public_key, revoked_at FROM node_credentials WHERE node_id = ?`, nodeID).
		Scan(&pubB64, &revokedRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("lookup credential: %w", err)
	}
	if revokedRaw != "" {
		return nil, false, nil
	}
	pub, err := base64.StdEncoding.DecodeString(pubB64)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		// A malformed stored key cannot authenticate anything; treat as absent.
		return nil, false, nil
	}
	return ed25519.PublicKey(pub), true, nil
}

func (s *sqliteRepository) CreateRequest(ctx context.Context, r PairingRequest) (PairingRequest, error) {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = s.clock.Now().UTC()
	}
	if r.Status == "" {
		r.Status = RequestPending
	}
	caps, err := marshalStrings(r.Capabilities)
	if err != nil {
		return PairingRequest{}, fmt.Errorf("encode capabilities: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO pairing_requests (id, public_key, name, os, arch, endpoint, capabilities, status, node_id, created_at, decided_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, '', ?, '')`,
		r.ID, r.PublicKey, r.Name, r.OS, r.Arch, r.Endpoint, caps, string(r.Status),
		r.CreatedAt.Format(timeFormat))
	if err != nil {
		return PairingRequest{}, fmt.Errorf("insert pairing request: %w", err)
	}
	return r, nil
}

func (s *sqliteRepository) GetRequest(ctx context.Context, id string) (PairingRequest, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, public_key, name, os, arch, endpoint, capabilities, status, node_id, created_at, decided_at
		 FROM pairing_requests WHERE id = ?`, id)
	r, err := scanRequest(row)
	if errors.Is(err, sql.ErrNoRows) {
		return PairingRequest{}, ErrRequestNotFound
	}
	if err != nil {
		return PairingRequest{}, fmt.Errorf("get pairing request: %w", err)
	}
	return r, nil
}

func (s *sqliteRepository) DecideRequest(ctx context.Context, id string, status RequestStatus, nodeID string) error {
	now := s.clock.Now().UTC().Format(timeFormat)
	res, err := s.db.ExecContext(ctx,
		`UPDATE pairing_requests SET status = ?, node_id = ?, decided_at = ? WHERE id = ? AND status = 'pending'`,
		string(status), nodeID, now, id)
	if err != nil {
		return fmt.Errorf("decide pairing request: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("decide pairing request rows: %w", err)
	}
	if n == 0 {
		return ErrRequestDecided
	}
	return nil
}

func (s *sqliteRepository) ListRequests(ctx context.Context, includeDecided bool) ([]PairingRequest, error) {
	query := `SELECT id, public_key, name, os, arch, endpoint, capabilities, status, node_id, created_at, decided_at
		FROM pairing_requests `
	if !includeDecided {
		query += `WHERE status = 'pending' `
	}
	query += `ORDER BY created_at DESC, id DESC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list pairing requests: %w", err)
	}
	defer rows.Close()

	var out []PairingRequest
	for rows.Next() {
		r, err := scanRequest(rows)
		if err != nil {
			return nil, fmt.Errorf("scan pairing request: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pairing requests: %w", err)
	}
	return out, nil
}

// --- scan / encode helpers (local to pairing; mirror the registry domain) ---

type rowScanner interface {
	Scan(dest ...any) error
}

func scanCode(s rowScanner) (PairingCode, error) {
	var (
		c          PairingCode
		scopesRaw  string
		createdRaw string
		expiresRaw string
		redeemRaw  string
	)
	if err := s.Scan(&c.ID, &c.CodeHash, &c.Name, &scopesRaw, &createdRaw, &expiresRaw, &redeemRaw, &c.RedeemedNodeID); err != nil {
		return PairingCode{}, err
	}
	var err error
	if c.Scopes, err = unmarshalStrings(scopesRaw); err != nil {
		return PairingCode{}, fmt.Errorf("decode scopes: %w", err)
	}
	if c.CreatedAt, err = time.Parse(timeFormat, createdRaw); err != nil {
		return PairingCode{}, fmt.Errorf("parse created_at: %w", err)
	}
	if c.ExpiresAt, err = time.Parse(timeFormat, expiresRaw); err != nil {
		return PairingCode{}, fmt.Errorf("parse expires_at: %w", err)
	}
	if c.RedeemedAt, err = parseNullableTime(redeemRaw); err != nil {
		return PairingCode{}, fmt.Errorf("parse redeemed_at: %w", err)
	}
	return c, nil
}

func scanRequest(s rowScanner) (PairingRequest, error) {
	var (
		r          PairingRequest
		capsRaw    string
		statusRaw  string
		createdRaw string
		decidedRaw string
	)
	if err := s.Scan(&r.ID, &r.PublicKey, &r.Name, &r.OS, &r.Arch, &r.Endpoint,
		&capsRaw, &statusRaw, &r.NodeID, &createdRaw, &decidedRaw); err != nil {
		return PairingRequest{}, err
	}
	var err error
	if r.Capabilities, err = unmarshalStrings(capsRaw); err != nil {
		return PairingRequest{}, fmt.Errorf("decode capabilities: %w", err)
	}
	r.Status = RequestStatus(statusRaw)
	if r.CreatedAt, err = time.Parse(timeFormat, createdRaw); err != nil {
		return PairingRequest{}, fmt.Errorf("parse created_at: %w", err)
	}
	if r.DecidedAt, err = parseNullableTime(decidedRaw); err != nil {
		return PairingRequest{}, fmt.Errorf("parse decided_at: %w", err)
	}
	return r, nil
}

func marshalStrings(v []string) (string, error) {
	if v == nil {
		v = []string{}
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func unmarshalStrings(raw string) ([]string, error) {
	if raw == "" {
		return nil, nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func parseNullableTime(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, nil
	}
	return time.Parse(timeFormat, raw)
}
