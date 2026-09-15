package candidates

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Store is the persistence surface for candidates.
type Store interface {
	Create(ctx context.Context, c Candidate) (Candidate, error)
	Get(ctx context.Context, id string) (Candidate, error)
	List(ctx context.Context, brandID string, status Status, limit, offset int) ([]Candidate, error)
	UpdateStatus(ctx context.Context, id string, status Status, note string) (Candidate, error)
	SupersedePicked(ctx context.Context, brandID, exceptID string) error
}

// ErrNotFound is returned when no candidate matches.
type ErrNotFound struct{ ID string }

func (e ErrNotFound) Error() string { return fmt.Sprintf("candidates: %q not found", e.ID) }

// SQLExecutor is the narrow database surface the store depends on.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteStore struct {
	db    SQLExecutor
	clock func() time.Time
}

// NewSQLiteStore constructs the production store.
func NewSQLiteStore(db SQLExecutor, clock func() time.Time) Store {
	if clock == nil {
		clock = time.Now
	}
	return &sqliteStore{db: db, clock: clock}
}

const candidateColumns = `id, brand_id, asset_id, media_type, concept, prompt, role, model, seed, origin, parent_id, status, note, generation_ref, created_at`

func (s *sqliteStore) Create(ctx context.Context, c Candidate) (Candidate, error) {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = s.clock().UTC()
	}
	if c.Status == "" {
		c.Status = StatusProposed
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO logo_candidates (`+candidateColumns+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		c.ID, c.BrandID, c.AssetID, c.MediaType, c.Concept, c.Prompt, c.Role, c.Model, c.Seed,
		string(c.Origin), c.ParentID, string(c.Status), c.Note, c.GenerationRef, c.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Candidate{}, fmt.Errorf("insert candidate: %w", err)
	}
	return c, nil
}

func (s *sqliteStore) Get(ctx context.Context, id string) (Candidate, error) {
	return scanCandidate(s.db.QueryRowContext(ctx, `SELECT `+candidateColumns+` FROM logo_candidates WHERE id = ?`, id))
}

func (s *sqliteStore) List(ctx context.Context, brandID string, status Status, limit, offset int) ([]Candidate, error) {
	query := `SELECT ` + candidateColumns + ` FROM logo_candidates WHERE brand_id = ?`
	args := []any{brandID}
	if status != "" {
		query += ` AND status = ?`
		args = append(args, string(status))
	}
	query += ` ORDER BY created_at DESC, id DESC`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}
	if offset > 0 {
		query += ` OFFSET ?`
		args = append(args, offset)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Candidate
	for rows.Next() {
		c, err := scanCandidate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *sqliteStore) UpdateStatus(ctx context.Context, id string, status Status, note string) (Candidate, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE logo_candidates SET status=?, note=CASE WHEN ?='' THEN note ELSE ? END WHERE id=?`,
		string(status), note, note, id)
	if err != nil {
		return Candidate{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Candidate{}, ErrNotFound{ID: id}
	}
	return s.Get(ctx, id)
}

func (s *sqliteStore) SupersedePicked(ctx context.Context, brandID, exceptID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE logo_candidates SET status='SUPERSEDED' WHERE brand_id=? AND status='PICKED' AND id<>?`,
		brandID, exceptID)
	return err
}

type rowScanner interface{ Scan(dest ...any) error }

func scanCandidate(sc rowScanner) (Candidate, error) {
	var c Candidate
	var origin, status, created string
	err := sc.Scan(&c.ID, &c.BrandID, &c.AssetID, &c.MediaType, &c.Concept, &c.Prompt, &c.Role, &c.Model,
		&c.Seed, &origin, &c.ParentID, &status, &c.Note, &c.GenerationRef, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return Candidate{}, ErrNotFound{}
	}
	if err != nil {
		return Candidate{}, err
	}
	c.Origin = Origin(origin)
	c.Status = Status(status)
	c.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return c, nil
}
