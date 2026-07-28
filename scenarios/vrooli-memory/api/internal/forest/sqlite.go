package forest

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type SQLiteRepository struct{ db *sql.DB }

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository { return &SQLiteRepository{db: db} }
func (r *SQLiteRepository) CreateSummary(ctx context.Context, s Summary, edges []Edge) (Summary, error) {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now().UTC()
	}
	if s.Generation == 0 {
		s.Generation = 1
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Summary{}, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO summaries (id,body,facet_id,depth,generation,created_at) VALUES (?,?,?,?,?,?)`, s.ID, s.Body, s.FacetID, s.Depth, s.Generation, s.CreatedAt.Format(time.RFC3339Nano)); err != nil {
		return Summary{}, err
	}
	for _, e := range edges {
		if e.ParentID != s.ID {
			return Summary{}, fmt.Errorf("edge parent %q does not match summary %q", e.ParentID, s.ID)
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO tree_edges (parent_id,child_id,child_kind) VALUES (?,?,?)`, e.ParentID, e.ChildID, e.ChildKind); err != nil {
			return Summary{}, err
		}
	}
	if err = tx.Commit(); err != nil {
		return Summary{}, err
	}
	return s, nil
}
func (r *SQLiteRepository) Frontier(ctx context.Context) ([]Summary, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT s.id,s.body,s.facet_id,s.depth,s.generation,s.created_at FROM summaries s LEFT JOIN tree_edges e ON e.child_id=s.id AND e.child_kind='summary' WHERE e.parent_id IS NULL ORDER BY s.created_at,s.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Summary
	for rows.Next() {
		var s Summary
		var created string
		if err := rows.Scan(&s.ID, &s.Body, &s.FacetID, &s.Depth, &s.Generation, &created); err != nil {
			return nil, err
		}
		s.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		out = append(out, s)
	}
	return out, rows.Err()
}
func (r *SQLiteRepository) Rebuild(ctx context.Context) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `DELETE FROM tree_edges`); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM summaries`); err != nil {
		return err
	}
	return tx.Commit()
}
