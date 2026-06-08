package registry

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"search-hub/internal/clock"

	"google.golang.org/protobuf/encoding/protojson"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

// SQLExecutor is the narrow database surface the store depends on. Declared at
// the consumer per seam-discovery: both *sql.DB (used by store tests via
// testutil/db.NewSQLite) and *database.RoutedDB (used in production by
// main.go) satisfy it, so production wiring participates in per-request
// routing without forcing the test fixture to wrap its handle.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Store is the persistence seam the registry handler depends on. Production
// wires the SQLite-backed implementation; handler unit tests wire a fake.
type Store interface {
	// Upsert validates and persists d (upsert keyed by provider_id). Returns
	// created=true when a new leaf was inserted, false when an existing leaf
	// was updated. Returns ErrInvalidDescriptor on validation failure.
	//
	// presentedToken is the control token the caller cached from a prior
	// registration (empty on a first registration or after the provider lost its
	// in-memory copy on restart). On INSERT the store MINTS a fresh token and
	// ignores presentedToken; on UPDATE it requires presentedToken to be empty or
	// to match the stored token (else ErrTokenMismatch). The authoritative token
	// for the provider is always returned so the caller can (re)cache it.
	Upsert(ctx context.Context, d *registryv1.ProviderDescriptor, presentedToken string) (created bool, controlToken string, err error)

	// List returns descriptors matching filter, ordered by provider_id.
	List(ctx context.Context, filter ListFilter) ([]*registryv1.ProviderDescriptor, error)

	// Get returns the descriptor for id or ErrProviderNotFound.
	Get(ctx context.Context, id string) (*registryv1.ProviderDescriptor, error)

	// Token returns the stored control token for id (empty string when the
	// provider exists but has no token yet). Returns ErrProviderNotFound when no
	// such provider is registered. search-hub uses it to present the token when
	// it calls a provider's token-gated verbs (override/reindex/config-write).
	Token(ctx context.Context, id string) (string, error)

	// Delete removes the leaf with the given provider_id. Returns removed=true
	// when a row was deleted, false when no such leaf existed (idempotent).
	Delete(ctx context.Context, id string) (removed bool, err error)
}

// sqliteStore is the production Store impl. Unexported so callers depend on
// the Store interface and substitute the fake without reaching inside.
type sqliteStore struct {
	db    SQLExecutor
	clock clock.Clock
}

// NewSQLiteStore constructs the production Store. db is the connection pool
// opened in main.go (*database.RoutedDB in production, *sql.DB in unit tests);
// clk supplies created_at/updated_at timestamps so tests can advance time
// deterministically.
func NewSQLiteStore(db SQLExecutor, clk clock.Clock) Store {
	return &sqliteStore{db: db, clock: clk}
}

// Compile-time guarantee.
var _ Store = (*sqliteStore)(nil)

const providerTimeFormat = time.RFC3339Nano

// marshalOpts keeps blobs compact and deterministic (EmitUnpopulated false so
// zero fields don't bloat the row; proto field names for readability when a
// human inspects the column).
var marshalOpts = protojson.MarshalOptions{UseProtoNames: true}

func (s *sqliteStore) Upsert(ctx context.Context, d *registryv1.ProviderDescriptor, presentedToken string) (bool, string, error) {
	Normalize(d)
	if err := Validate(d); err != nil {
		return false, "", err
	}

	blob, err := marshalOpts.Marshal(d)
	if err != nil {
		return false, "", fmt.Errorf("marshal descriptor: %w", err)
	}

	now := s.clock.Now().UTC().Format(providerTimeFormat)

	// Determine insert vs update so we can report `created` honestly and so the
	// token is minted exactly once (on insert) and echoed thereafter. SQLite is a
	// single writer, so the read-then-write race is benign for this registry.
	var storedToken string
	err = s.db.QueryRowContext(ctx, `SELECT control_token FROM providers WHERE provider_id = ?`, d.ProviderId).
		Scan(&storedToken)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		token := newControlToken()
		_, insErr := s.db.ExecContext(ctx, `
INSERT INTO providers (provider_id, provider_group, bucket, type, state, scope, descriptor, control_token, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			d.ProviderId, d.ProviderGroup, int32(d.Bucket), d.Type, int32(d.State), int32(d.Scope),
			string(blob), token, now, now)
		if insErr != nil {
			return false, "", fmt.Errorf("insert provider: %w", insErr)
		}
		return true, token, nil
	case err != nil:
		return false, "", fmt.Errorf("probe provider: %w", err)
	default:
		// Ownership proof: a non-empty presented token must match the stored one.
		// An empty presented token is allowed (the provider re-registers on every
		// boot but holds the token only in memory) and simply receives the echo.
		if presentedToken != "" && presentedToken != storedToken {
			return false, "", ErrTokenMismatch{ProviderID: d.ProviderId}
		}
		// Heal a legacy row that predates the token column (empty stored token) by
		// minting one now, so every registered provider ends up with a stable token.
		if storedToken == "" {
			storedToken = newControlToken()
		}
		_, updErr := s.db.ExecContext(ctx, `
UPDATE providers
SET provider_group = ?, bucket = ?, type = ?, state = ?, scope = ?, descriptor = ?, control_token = ?, updated_at = ?
WHERE provider_id = ?`,
			d.ProviderGroup, int32(d.Bucket), d.Type, int32(d.State), int32(d.Scope),
			string(blob), storedToken, now, d.ProviderId)
		if updErr != nil {
			return false, "", fmt.Errorf("update provider: %w", updErr)
		}
		return false, storedToken, nil
	}
}

// Token returns the stored control token for id (empty when the provider has
// none yet), or ErrProviderNotFound when no such provider is registered.
func (s *sqliteStore) Token(ctx context.Context, id string) (string, error) {
	var token string
	err := s.db.QueryRowContext(ctx, `SELECT control_token FROM providers WHERE provider_id = ?`, id).Scan(&token)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrProviderNotFound{ProviderID: id}
	}
	if err != nil {
		return "", fmt.Errorf("get provider token: %w", err)
	}
	return token, nil
}

// newControlToken returns a random 128-bit hex token (stdlib only — no external
// uuid dep, matching the aisearch reindex job-id generator's posture). A
// crypto/rand failure is fatal to security, so it panics rather than minting a
// predictable token.
func newControlToken() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("registry: crypto/rand failed minting control token: %v", err))
	}
	return hex.EncodeToString(b[:])
}

func (s *sqliteStore) List(ctx context.Context, filter ListFilter) ([]*registryv1.ProviderDescriptor, error) {
	var (
		clauses []string
		args    []any
	)
	if filter.Bucket != 0 {
		clauses = append(clauses, "bucket = ?")
		args = append(args, filter.Bucket)
	}
	if strings.TrimSpace(filter.Type) != "" {
		clauses = append(clauses, "type = ?")
		args = append(args, filter.Type)
	}
	if filter.State != 0 {
		clauses = append(clauses, "state = ?")
		args = append(args, filter.State)
	}

	query := "SELECT descriptor FROM providers"
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY provider_id ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query providers: %w", err)
	}
	defer rows.Close()

	out := make([]*registryv1.ProviderDescriptor, 0)
	for rows.Next() {
		d, scanErr := scanDescriptor(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate providers: %w", err)
	}
	return out, nil
}

func (s *sqliteStore) Get(ctx context.Context, id string) (*registryv1.ProviderDescriptor, error) {
	var blob string
	err := s.db.QueryRowContext(ctx, `SELECT descriptor FROM providers WHERE provider_id = ?`, id).Scan(&blob)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProviderNotFound{ProviderID: id}
	}
	if err != nil {
		return nil, fmt.Errorf("get provider: %w", err)
	}
	return unmarshalDescriptor(blob)
}

func (s *sqliteStore) Delete(ctx context.Context, id string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM providers WHERE provider_id = ?`, id)
	if err != nil {
		return false, fmt.Errorf("delete provider: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("delete provider rows affected: %w", err)
	}
	return n > 0, nil
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanDescriptor(s scanner) (*registryv1.ProviderDescriptor, error) {
	var blob string
	if err := s.Scan(&blob); err != nil {
		return nil, fmt.Errorf("scan provider: %w", err)
	}
	return unmarshalDescriptor(blob)
}

func unmarshalDescriptor(blob string) (*registryv1.ProviderDescriptor, error) {
	d := &registryv1.ProviderDescriptor{}
	if err := protojson.Unmarshal([]byte(blob), d); err != nil {
		return nil, fmt.Errorf("unmarshal descriptor: %w", err)
	}
	return d, nil
}
