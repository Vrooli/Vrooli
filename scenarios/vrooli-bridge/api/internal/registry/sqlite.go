package registry

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"vrooli-bridge/internal/clock"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface sqliteRepository depends on. Both
// *sql.DB (repository unit tests via testutil/db.NewSQLite) and
// *database.RoutedDB (production via main.go) satisfy it, so production
// participates in per-request routing without the test fixture wrapping its
// handle.
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

// Compile-time guarantee.
var _ Repository = (*sqliteRepository)(nil)

const (
	// RFC3339Nano sorts lexicographically in time order for a fixed zone, so a
	// string range/order comparison is a correct filter — matching the wire
	// format and the notes domain convention.
	nodeTimeFormat = time.RFC3339Nano

	insertNodeSQL = `
INSERT INTO nodes (id, name, os, arch, revision, endpoint, capabilities, scopes, pairing_correlation_id, created_at, updated_at, last_seen_at, revoked_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

	selectNodeColumns = `
SELECT id, name, os, arch, revision, endpoint, capabilities, scopes, pairing_correlation_id, created_at, updated_at, last_seen_at, revoked_at
FROM nodes
`

	selectNodeByIDSQL = selectNodeColumns + `WHERE id = ?`

	listNodesSQL = selectNodeColumns + `ORDER BY created_at DESC, id DESC`

	updateNodeSQL = `
UPDATE nodes
SET name = ?, endpoint = ?, capabilities = ?, scopes = ?, revision = ?, updated_at = ?
WHERE id = ?
`

	revokeNodeSQL = `
UPDATE nodes
SET revoked_at = ?, updated_at = ?
WHERE id = ? AND revoked_at = ''
`

	touchLastSeenSQL = `
UPDATE nodes
SET last_seen_at = ?
WHERE id = ?
`
)

func (s *sqliteRepository) Create(ctx context.Context, n Node) (Node, error) {
	if n.ID == "" {
		n.ID = uuid.NewString()
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = s.clock.Now().UTC()
	}
	if n.UpdatedAt.IsZero() {
		n.UpdatedAt = n.CreatedAt
	}

	caps, err := marshalStrings(n.Capabilities)
	if err != nil {
		return Node{}, fmt.Errorf("encode capabilities: %w", err)
	}
	scopes, err := marshalStrings(n.Scopes)
	if err != nil {
		return Node{}, fmt.Errorf("encode scopes: %w", err)
	}

	_, err = s.db.ExecContext(ctx, insertNodeSQL,
		n.ID, n.Name, n.OS, n.Arch, n.Revision, n.Endpoint, caps, scopes,
		n.PairingCorrelationID,
		n.CreatedAt.Format(nodeTimeFormat), n.UpdatedAt.Format(nodeTimeFormat),
		formatNullableTime(n.LastSeenAt), formatNullableTime(n.RevokedAt),
	)
	if err != nil {
		return Node{}, fmt.Errorf("insert node %q: %w", n.ID, err)
	}
	return n, nil
}

func (s *sqliteRepository) GetByPairingCorrelation(ctx context.Context, correlationID string) (Node, error) {
	row := s.db.QueryRowContext(ctx, selectNodeColumns+"WHERE pairing_correlation_id = ?", correlationID)
	n, err := scanNode(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Node{}, ErrNodeNotFound{ID: correlationID}
	}
	if err != nil {
		return Node{}, fmt.Errorf("get node by pairing correlation %q: %w", correlationID, err)
	}
	return n, nil
}

func (s *sqliteRepository) Get(ctx context.Context, id string) (Node, error) {
	row := s.db.QueryRowContext(ctx, selectNodeByIDSQL, id)
	n, err := scanNode(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Node{}, ErrNodeNotFound{ID: id}
	}
	if err != nil {
		return Node{}, fmt.Errorf("get node %q: %w", id, err)
	}
	return n, nil
}

func (s *sqliteRepository) List(ctx context.Context) ([]Node, error) {
	rows, err := s.db.QueryContext(ctx, listNodesSQL)
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}
	defer rows.Close()

	var nodes []Node
	for rows.Next() {
		n, err := scanNode(rows)
		if err != nil {
			return nil, fmt.Errorf("scan node: %w", err)
		}
		nodes = append(nodes, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate nodes: %w", err)
	}
	return nodes, nil
}

func (s *sqliteRepository) Update(ctx context.Context, n Node) (Node, error) {
	// Read-modify-write so the returned node carries the immutable fields
	// (os/arch/created_at) and so we return ErrNodeNotFound consistently.
	existing, err := s.Get(ctx, n.ID)
	if err != nil {
		return Node{}, err
	}

	existing.Name = n.Name
	existing.Endpoint = n.Endpoint
	existing.Capabilities = n.Capabilities
	existing.Scopes = n.Scopes
	existing.Revision = n.Revision
	existing.UpdatedAt = s.clock.Now().UTC()

	caps, err := marshalStrings(existing.Capabilities)
	if err != nil {
		return Node{}, fmt.Errorf("encode capabilities: %w", err)
	}
	scopes, err := marshalStrings(existing.Scopes)
	if err != nil {
		return Node{}, fmt.Errorf("encode scopes: %w", err)
	}

	if _, err := s.db.ExecContext(ctx, updateNodeSQL,
		existing.Name, existing.Endpoint, caps, scopes, existing.Revision,
		existing.UpdatedAt.Format(nodeTimeFormat), existing.ID,
	); err != nil {
		return Node{}, fmt.Errorf("update node %q: %w", n.ID, err)
	}
	return existing, nil
}

func (s *sqliteRepository) Revoke(ctx context.Context, id string) (Node, error) {
	existing, err := s.Get(ctx, id)
	if err != nil {
		return Node{}, err
	}
	if existing.Revoked() {
		// Idempotent: already revoked.
		return existing, nil
	}
	now := s.clock.Now().UTC()
	if _, err := s.db.ExecContext(ctx, revokeNodeSQL,
		now.Format(nodeTimeFormat), now.Format(nodeTimeFormat), id,
	); err != nil {
		return Node{}, fmt.Errorf("revoke node %q: %w", id, err)
	}
	existing.RevokedAt = now
	existing.UpdatedAt = now
	return existing, nil
}

func (s *sqliteRepository) TouchLastSeen(ctx context.Context, id string, t time.Time) error {
	if _, err := s.db.ExecContext(ctx, touchLastSeenSQL, t.UTC().Format(nodeTimeFormat), id); err != nil {
		return fmt.Errorf("touch last_seen for node %q: %w", id, err)
	}
	return nil
}

// rowScanner unifies *sql.Row and *sql.Rows under their common Scan surface.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanNode(s rowScanner) (Node, error) {
	var (
		n           Node
		capsRaw     string
		scopesRaw   string
		createdRaw  string
		updatedRaw  string
		lastSeenRaw string
		revokedRaw  string
	)
	if err := s.Scan(&n.ID, &n.Name, &n.OS, &n.Arch, &n.Revision, &n.Endpoint,
		&capsRaw, &scopesRaw, &n.PairingCorrelationID, &createdRaw, &updatedRaw, &lastSeenRaw, &revokedRaw); err != nil {
		return Node{}, err
	}

	caps, err := unmarshalStrings(capsRaw)
	if err != nil {
		return Node{}, fmt.Errorf("decode capabilities: %w", err)
	}
	scopes, err := unmarshalStrings(scopesRaw)
	if err != nil {
		return Node{}, fmt.Errorf("decode scopes: %w", err)
	}
	n.Capabilities = caps
	n.Scopes = scopes

	if n.CreatedAt, err = time.Parse(nodeTimeFormat, createdRaw); err != nil {
		return Node{}, fmt.Errorf("parse created_at %q: %w", createdRaw, err)
	}
	if n.UpdatedAt, err = time.Parse(nodeTimeFormat, updatedRaw); err != nil {
		return Node{}, fmt.Errorf("parse updated_at %q: %w", updatedRaw, err)
	}
	if n.LastSeenAt, err = parseNullableTime(lastSeenRaw); err != nil {
		return Node{}, fmt.Errorf("parse last_seen_at %q: %w", lastSeenRaw, err)
	}
	if n.RevokedAt, err = parseNullableTime(revokedRaw); err != nil {
		return Node{}, fmt.Errorf("parse revoked_at %q: %w", revokedRaw, err)
	}
	return n, nil
}

// marshalStrings encodes a string slice as a JSON array, normalising nil to
// "[]" so the column never holds NULL/"".
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

// formatNullableTime renders a zero time as "" (the column default) so absence
// is distinguishable from a real timestamp.
func formatNullableTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(nodeTimeFormat)
}

func parseNullableTime(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, nil
	}
	return time.Parse(nodeTimeFormat, raw)
}
