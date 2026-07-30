package forest

import (
	"context"
	"database/sql"
	"encoding/json"
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
	vector, err := json.Marshal(s.Vector)
	if err != nil {
		return Summary{}, fmt.Errorf("encode summary vector: %w", err)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Summary{}, err
	}
	defer func() { _ = tx.Rollback() }()
	for _, e := range edges {
		if e.ParentID != s.ID {
			e.ParentID = s.ID
		}
		if e.ChildKind != "entry" && e.ChildKind != "summary" {
			return Summary{}, fmt.Errorf("unknown child kind %q", e.ChildKind)
		}
		var parents int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM tree_edges WHERE child_id=? AND child_kind=?`, e.ChildID, e.ChildKind).Scan(&parents); err != nil {
			return Summary{}, err
		}
		if parents != 0 {
			return Summary{}, fmt.Errorf("child %s/%s is no longer on frontier", e.ChildKind, e.ChildID)
		}
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO summaries (id,body,facet_id,vector_json,depth,generation,created_at) VALUES (?,?,?,?,?,?,?)`, s.ID, s.Body, s.FacetID, string(vector), s.Depth, s.Generation, s.CreatedAt.Format(time.RFC3339Nano)); err != nil {
		return Summary{}, err
	}
	for _, e := range edges {
		e.ParentID = s.ID
		if _, err = tx.ExecContext(ctx, `INSERT INTO tree_edges (parent_id,child_id,child_kind) VALUES (?,?,?)`, e.ParentID, e.ChildID, e.ChildKind); err != nil {
			return Summary{}, err
		}
	}
	if err = tx.Commit(); err != nil {
		return Summary{}, err
	}
	return s, nil
}

func (r *SQLiteRepository) Nodes(ctx context.Context) ([]Node, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT e.id,e.id,e.facet_id,e.body,COALESCE((SELECT em.vector_json FROM facet_texts ft JOIN embeddings em ON em.facet_text_id=ft.id WHERE ft.entry_id=e.id ORDER BY ft.id LIMIT 1),'[]'),0,0,e.created_at,0
FROM entries e WHERE NOT EXISTS(SELECT 1 FROM tree_edges WHERE child_id=e.id AND child_kind='entry')
UNION ALL
SELECT s.id,s.id,s.facet_id,s.body,s.vector_json,s.depth,s.generation,s.created_at,1
FROM summaries s WHERE NOT EXISTS(SELECT 1 FROM tree_edges WHERE child_id=s.id AND child_kind='summary')
ORDER BY 8,1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Node
	for rows.Next() {
		var n Node
		var raw, created string
		if err := rows.Scan(&n.ID, &n.EntryID, &n.FacetID, &n.Body, &raw, &n.Depth, &n.Generation, &created, &n.Summary); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(raw), &n.Vector); err != nil {
			return nil, fmt.Errorf("decode node vector: %w", err)
		}
		n.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) Frontier(ctx context.Context) ([]Summary, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT s.id,s.body,s.facet_id,s.vector_json,s.depth,s.generation,s.created_at FROM summaries s LEFT JOIN tree_edges e ON e.child_id=s.id AND e.child_kind='summary' WHERE e.parent_id IS NULL ORDER BY s.created_at,s.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Summary
	for rows.Next() {
		var s Summary
		var created, vector string
		if err := rows.Scan(&s.ID, &s.Body, &s.FacetID, &vector, &s.Depth, &s.Generation, &created); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(vector), &s.Vector); err != nil {
			return nil, fmt.Errorf("decode summary vector: %w", err)
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
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `DELETE FROM tree_edges`); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM summaries`); err != nil {
		return err
	}
	return tx.Commit()
}
