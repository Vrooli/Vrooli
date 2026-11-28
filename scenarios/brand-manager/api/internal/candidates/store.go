package candidates

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/database"
)

// Store is the persistence surface for candidates.
type Store interface {
	Create(ctx context.Context, c Candidate) (Candidate, error)
	Get(ctx context.Context, id string) (Candidate, error)
	List(ctx context.Context, brandID string, status Status, limit, offset int) ([]Candidate, error)
	TransitionStatus(ctx context.Context, id string, expected, to Status, note string) (Candidate, error)
	Pick(ctx context.Context, id string) (Candidate, error)
	// FindByAsset returns the brand's oldest candidate for an asset, if any.
	FindByAsset(ctx context.Context, brandID, assetID string) (Candidate, bool, error)
}

// ErrNotFound is returned when no candidate matches.
type ErrNotFound struct{ ID string }

func (e ErrNotFound) Error() string { return fmt.Sprintf("candidates: %q not found", e.ID) }

type sqliteStore struct {
	db    database.SQLTxExecutor
	clock func() time.Time
}

// NewSQLiteStore constructs the production store.
func NewSQLiteStore(db database.SQLTxExecutor, clock func() time.Time) Store {
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
	if !ValidStatus(c.Status) || c.Status != StatusProposed {
		return Candidate{}, ErrInvalidStatus{Status: c.Status}
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

func (s *sqliteStore) TransitionStatus(ctx context.Context, id string, expected, status Status, note string) (Candidate, error) {
	if !CanTransitionStatus(expected, status) {
		return Candidate{}, ErrInvalidStatusTransition{From: expected, To: status}
	}
	res, err := s.db.ExecContext(ctx, `UPDATE logo_candidates SET status=?, note=CASE WHEN ?='' THEN note ELSE ? END WHERE id=? AND status=?`,
		string(status), note, note, id, string(expected))
	if err != nil {
		return Candidate{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		current, getErr := s.Get(ctx, id)
		if getErr != nil {
			return Candidate{}, getErr
		}
		return Candidate{}, ErrStatusConflict{ID: current.ID, Expected: expected}
	}
	return s.Get(ctx, id)
}

func (s *sqliteStore) Pick(ctx context.Context, id string) (Candidate, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Candidate{}, err
	}
	defer func() { _ = tx.Rollback() }()
	candidate, err := scanCandidate(tx.QueryRowContext(ctx, `SELECT `+candidateColumns+` FROM logo_candidates WHERE id = ?`, id))
	if err != nil {
		return Candidate{}, err
	}
	if candidate.Status != StatusPicked {
		if !CanTransitionStatus(candidate.Status, StatusPicked) {
			return Candidate{}, ErrInvalidStatusTransition{From: candidate.Status, To: StatusPicked}
		}
		if !CanTransitionStatus(StatusPicked, StatusSuperseded) {
			return Candidate{}, ErrInvalidStatusTransition{From: StatusPicked, To: StatusSuperseded}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE logo_candidates SET status=? WHERE brand_id=? AND status=? AND id<>?`,
			string(StatusSuperseded), candidate.BrandID, string(StatusPicked), candidate.ID); err != nil {
			return Candidate{}, err
		}
		result, err := tx.ExecContext(ctx, `UPDATE logo_candidates SET status=? WHERE id=? AND status=?`,
			string(StatusPicked), id, string(candidate.Status))
		if err != nil {
			return Candidate{}, err
		}
		if rows, _ := result.RowsAffected(); rows != 1 {
			return Candidate{}, ErrStatusConflict{ID: id, Expected: candidate.Status}
		}
	}
	selected, err := scanCandidate(tx.QueryRowContext(ctx, `SELECT `+candidateColumns+` FROM logo_candidates WHERE id = ?`, id))
	if err != nil {
		return Candidate{}, err
	}
	if err := tx.Commit(); err != nil {
		return Candidate{}, err
	}
	return selected, nil
}

func (s *sqliteStore) FindByAsset(ctx context.Context, brandID, assetID string) (Candidate, bool, error) {
	c, err := scanCandidate(s.db.QueryRowContext(ctx,
		`SELECT `+candidateColumns+` FROM logo_candidates WHERE brand_id = ? AND asset_id = ? ORDER BY created_at ASC, id ASC LIMIT 1`,
		brandID, assetID))
	if err != nil {
		var notFound ErrNotFound
		if errors.As(err, &notFound) || errors.Is(err, sql.ErrNoRows) {
			return Candidate{}, false, nil
		}
		return Candidate{}, false, err
	}
	return c, true, nil
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
	if !ValidStatus(c.Status) {
		return Candidate{}, ErrInvalidStatus{Status: c.Status}
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return c, nil
}
