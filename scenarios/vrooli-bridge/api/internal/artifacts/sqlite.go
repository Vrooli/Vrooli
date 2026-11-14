package artifacts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/vrooli/api-core/schedule"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface sqliteRepository depends on. Both
// *sql.DB (repository unit tests) and *database.RoutedDB (production) satisfy
// it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db SQLExecutor, clk schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

type sqliteProducedRepository struct {
	db    SQLExecutor
	clock schedule.Clock
}

// NewSQLiteProducedRepository constructs the bounded produced-artifact store.
func NewSQLiteProducedRepository(db SQLExecutor, clk schedule.Clock) ProducedArtifactRepository {
	return &sqliteProducedRepository{db: db, clock: clk}
}

// Compile-time guarantee.
var (
	_ Repository                 = (*sqliteRepository)(nil)
	_ ProducedArtifactRepository = (*sqliteProducedRepository)(nil)
)

// timeFormat sorts lexicographically in time order for a fixed zone, so a string
// order comparison is a correct newest-first filter.
const timeFormat = time.RFC3339Nano

const (
	insertSQL = `
INSERT INTO distributions (id, node_id, name, source_ref, destination_path, status, delivery_ref, detail, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

	selectColumns = `
SELECT id, node_id, name, source_ref, destination_path, status, delivery_ref, detail, created_at, updated_at
FROM distributions
`

	selectByIDSQL = selectColumns + `WHERE id = ?`

	updateStatusSQL = `
UPDATE distributions SET status = ?, delivery_ref = ?, detail = ?, updated_at = ? WHERE id = ?
`
)

func (s *sqliteRepository) Create(ctx context.Context, d Distribution) (Distribution, error) {
	if d.ID == "" {
		d.ID = uuid.NewString()
	}
	now := s.clock.Now().UTC()
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}
	if d.UpdatedAt.IsZero() {
		d.UpdatedAt = now
	}
	if d.Status == StatusUnspecified {
		d.Status = StatusPending
	}
	if _, err := s.db.ExecContext(ctx, insertSQL,
		d.ID, d.NodeID, d.Name, d.SourceRef, d.DestinationPath, int(d.Status),
		d.DeliveryRef, d.Detail, d.CreatedAt.Format(timeFormat), d.UpdatedAt.Format(timeFormat),
	); err != nil {
		return Distribution{}, fmt.Errorf("insert distribution %q: %w", d.ID, err)
	}
	return d, nil
}

func (s *sqliteRepository) Get(ctx context.Context, id string) (Distribution, error) {
	row := s.db.QueryRowContext(ctx, selectByIDSQL, id)
	d, err := scan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Distribution{}, ErrDistributionNotFound{ID: id}
	}
	if err != nil {
		return Distribution{}, fmt.Errorf("get distribution %q: %w", id, err)
	}
	return d, nil
}

func (s *sqliteRepository) List(ctx context.Context, filter ListFilter) ([]Distribution, error) {
	query := selectColumns
	args := make([]any, 0, 2)
	if filter.NodeID != "" {
		query += `WHERE node_id = ? `
		args = append(args, filter.NodeID)
	}
	// rowid DESC breaks created_at ties by insertion order, so newest-first is
	// deterministic even when several distributions share a timestamp.
	query += `ORDER BY created_at DESC, rowid DESC`
	if filter.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list distributions: %w", err)
	}
	defer rows.Close()

	var out []Distribution
	for rows.Next() {
		d, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan distribution: %w", err)
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate distributions: %w", err)
	}
	return out, nil
}

func (s *sqliteRepository) UpdateStatus(ctx context.Context, id string, status DeliveryStatus, deliveryRef, detail string) (Distribution, error) {
	existing, err := s.Get(ctx, id)
	if err != nil {
		return Distribution{}, err
	}
	existing.Status = status
	if deliveryRef != "" {
		existing.DeliveryRef = deliveryRef
	}
	existing.Detail = detail
	existing.UpdatedAt = s.clock.Now().UTC()

	if _, err := s.db.ExecContext(ctx, updateStatusSQL,
		int(existing.Status), existing.DeliveryRef, existing.Detail, existing.UpdatedAt.Format(timeFormat), id,
	); err != nil {
		return Distribution{}, fmt.Errorf("update distribution %q: %w", id, err)
	}
	return existing, nil
}

// rowScanner unifies *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scan(sc rowScanner) (Distribution, error) {
	var (
		d          Distribution
		status     int
		createdRaw string
		updatedRaw string
	)
	if err := sc.Scan(&d.ID, &d.NodeID, &d.Name, &d.SourceRef, &d.DestinationPath, &status,
		&d.DeliveryRef, &d.Detail, &createdRaw, &updatedRaw); err != nil {
		return Distribution{}, err
	}
	d.Status = DeliveryStatus(status)
	var err error
	if d.CreatedAt, err = time.Parse(timeFormat, createdRaw); err != nil {
		return Distribution{}, fmt.Errorf("parse created_at %q: %w", createdRaw, err)
	}
	if updatedRaw != "" {
		if d.UpdatedAt, err = time.Parse(timeFormat, updatedRaw); err != nil {
			return Distribution{}, fmt.Errorf("parse updated_at %q: %w", updatedRaw, err)
		}
	}
	return d, nil
}

const (
	putProducedSQL = `
INSERT INTO produced_artifacts (run_id, name, media_type, data, size_bytes, artifact_ref, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(run_id, name) DO UPDATE SET media_type = excluded.media_type, data = excluded.data, size_bytes = excluded.size_bytes, artifact_ref = excluded.artifact_ref, created_at = excluded.created_at
`
	getProducedSQL = `
SELECT run_id, name, media_type, data, size_bytes, artifact_ref, created_at
FROM produced_artifacts WHERE run_id = ? AND name = ?
`
)

func (s *sqliteProducedRepository) Put(ctx context.Context, a ProducedArtifact) (ProducedArtifact, error) {
	if a.CreatedAt.IsZero() {
		a.CreatedAt = s.clock.Now().UTC()
	}
	if a.SizeBytes == 0 {
		a.SizeBytes = int64(len(a.Data))
	}
	if _, err := s.db.ExecContext(ctx, putProducedSQL, a.RunID, a.Name, a.MediaType, a.Data, a.SizeBytes, a.ArtifactRef, a.CreatedAt.Format(timeFormat)); err != nil {
		return ProducedArtifact{}, fmt.Errorf("store produced artifact %q/%q: %w", a.RunID, a.Name, err)
	}
	return a, nil
}

func (s *sqliteProducedRepository) Get(ctx context.Context, runID, name string) (ProducedArtifact, error) {
	var a ProducedArtifact
	var createdRaw string
	err := s.db.QueryRowContext(ctx, getProducedSQL, runID, name).Scan(
		&a.RunID, &a.Name, &a.MediaType, &a.Data, &a.SizeBytes, &a.ArtifactRef, &createdRaw,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ProducedArtifact{}, ErrProducedArtifactNotFound{RunID: runID, Name: name}
	}
	if err != nil {
		return ProducedArtifact{}, fmt.Errorf("get produced artifact %q/%q: %w", runID, name, err)
	}
	a.CreatedAt, err = time.Parse(timeFormat, createdRaw)
	if err != nil {
		return ProducedArtifact{}, fmt.Errorf("parse produced artifact created_at: %w", err)
	}
	return a, nil
}
