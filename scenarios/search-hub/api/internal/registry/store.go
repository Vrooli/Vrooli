package registry

import (
	"context"
	"database/sql"
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
	Upsert(ctx context.Context, d *registryv1.ProviderDescriptor) (created bool, err error)

	// List returns descriptors matching filter, ordered by provider_id.
	List(ctx context.Context, filter ListFilter) ([]*registryv1.ProviderDescriptor, error)

	// Get returns the descriptor for id or ErrProviderNotFound.
	Get(ctx context.Context, id string) (*registryv1.ProviderDescriptor, error)

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

func (s *sqliteStore) Upsert(ctx context.Context, d *registryv1.ProviderDescriptor) (bool, error) {
	Normalize(d)
	if err := Validate(d); err != nil {
		return false, err
	}

	blob, err := marshalOpts.Marshal(d)
	if err != nil {
		return false, fmt.Errorf("marshal descriptor: %w", err)
	}

	now := s.clock.Now().UTC().Format(providerTimeFormat)

	// Determine insert vs update so we can report `created` honestly. SQLite is
	// a single writer, so the read-then-write race is benign for this registry.
	var existingCreatedAt string
	err = s.db.QueryRowContext(ctx, `SELECT created_at FROM providers WHERE provider_id = ?`, d.ProviderId).
		Scan(&existingCreatedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		_, insErr := s.db.ExecContext(ctx, `
INSERT INTO providers (provider_id, provider_group, bucket, type, state, scope, descriptor, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			d.ProviderId, d.ProviderGroup, int32(d.Bucket), d.Type, int32(d.State), int32(d.Scope),
			string(blob), now, now)
		if insErr != nil {
			return false, fmt.Errorf("insert provider: %w", insErr)
		}
		return true, nil
	case err != nil:
		return false, fmt.Errorf("probe provider: %w", err)
	default:
		_, updErr := s.db.ExecContext(ctx, `
UPDATE providers
SET provider_group = ?, bucket = ?, type = ?, state = ?, scope = ?, descriptor = ?, updated_at = ?
WHERE provider_id = ?`,
			d.ProviderGroup, int32(d.Bucket), d.Type, int32(d.State), int32(d.Scope),
			string(blob), now, d.ProviderId)
		if updErr != nil {
			return false, fmt.Errorf("update provider: %w", updErr)
		}
		return false, nil
	}
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
