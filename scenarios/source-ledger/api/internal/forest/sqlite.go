package forest

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"source-ledger/internal/policy"
	vectorcodec "source-ledger/internal/vector"
)

type SQLiteRepository struct{ db *sql.DB }

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository { return &SQLiteRepository{db: db} }
func (r *SQLiteRepository) CreateSummary(ctx context.Context, s Summary, edges []Edge) (Summary, error) {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	s.Scope = string(policy.ScopeFromContext(ctx))
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
	if _, err = tx.ExecContext(ctx, `INSERT INTO summaries (id,scope,body,facet_id,vector_json,vector_blob,depth,generation,created_at) VALUES (?,?,?,?,?,?,?,?,?)`, s.ID, s.Scope, s.Body, s.FacetID, "", vectorcodec.Encode(s.Vector), s.Depth, s.Generation, s.CreatedAt.Format(time.RFC3339Nano)); err != nil {
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

func (r *SQLiteRepository) Nodes(ctx context.Context, limit int) ([]Node, error) {
	scope := policy.ScopeFromContext(ctx)
	rows, err := r.db.QueryContext(ctx, `
WITH latest_assignment AS (
  SELECT entry_id,facet_id FROM (
    SELECT entry_id,facet_id,ROW_NUMBER() OVER (PARTITION BY entry_id ORDER BY assigned_at DESC,id DESC) AS rn
    FROM facet_assignments
  ) WHERE rn=1
)
SELECT e.id,e.id,COALESCE(la.facet_id,e.facet_id),e.body,X'', '',e.created_at,0,0,0
FROM entries e LEFT JOIN latest_assignment la ON la.entry_id=e.id
WHERE e.scope=? AND NOT EXISTS(SELECT 1 FROM tree_edges WHERE child_id=e.id AND child_kind='entry')
UNION ALL
SELECT s.id,s.id,s.facet_id,s.body,s.vector_blob,s.vector_json,s.created_at,s.depth,s.generation,1
FROM summaries s WHERE s.scope=? AND NOT EXISTS(SELECT 1 FROM tree_edges WHERE child_id=s.id AND child_kind='summary')
ORDER BY 7,1`+nodeLimitClause(limit), append([]any{scope, scope}, limitArg(limit)...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Node
	for rows.Next() {
		var n Node
		var raw, created string
		var blob []byte
		if err := rows.Scan(&n.ID, &n.EntryID, &n.FacetID, &n.Body, &blob, &raw, &created, &n.Depth, &n.Generation, &n.Summary); err != nil {
			return nil, err
		}
		_ = blob
		_ = raw
		n.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	vectors, err := r.nodeVectors(ctx, out)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].Vectors = vectors[out[i].ID]
	}
	return out, nil
}

func nodeLimitClause(limit int) string {
	if limit > 0 {
		return " LIMIT ?"
	}
	return ""
}

func limitArg(limit int) []any {
	if limit > 0 {
		return []any{limit}
	}
	return nil
}

func (r *SQLiteRepository) nodeVectors(ctx context.Context, nodes []Node) (map[string][][]float64, error) {
	ids := make([]string, 0, len(nodes))
	for _, node := range nodes {
		ids = append(ids, node.ID)
	}
	if len(ids) == 0 {
		return map[string][][]float64{}, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT ft.entry_id,em.vector_blob,em.vector_json
	FROM facet_texts ft JOIN embeddings em ON em.facet_text_id=ft.id JOIN entries e ON e.id=ft.entry_id
WHERE e.scope=? AND ft.entry_id IN (`+placeholders+`)
ORDER BY ft.entry_id,ft.id`, append([]any{policy.ScopeFromContext(ctx)}, args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][][]float64{}
	for rows.Next() {
		var id, raw string
		var blob []byte
		if err := rows.Scan(&id, &blob, &raw); err != nil {
			return nil, err
		}
		vector, err := decodeVector(blob, raw)
		if err != nil {
			return nil, fmt.Errorf("decode node vector: %w", err)
		}
		if len(vector) > 0 {
			out[id] = append(out[id], vector)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows, err = r.db.QueryContext(ctx, `SELECT id,vector_blob,vector_json FROM summaries WHERE scope=? AND id IN (`+placeholders+`) ORDER BY id`, append([]any{policy.ScopeFromContext(ctx)}, args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, raw string
		var blob []byte
		if err := rows.Scan(&id, &blob, &raw); err != nil {
			return nil, err
		}
		vector, err := decodeVector(blob, raw)
		if err != nil {
			return nil, fmt.Errorf("decode summary node vector: %w", err)
		}
		if len(vector) > 0 {
			out[id] = append(out[id], vector)
		}
	}
	return out, rows.Err()
}

func decodeVector(blob []byte, raw string) ([]float64, error) {
	if len(blob) > 0 {
		return vectorcodec.Decode(blob)
	}
	if raw == "" || raw == "[]" {
		return nil, nil
	}
	var vector []float64
	if err := json.Unmarshal([]byte(raw), &vector); err != nil {
		return nil, err
	}
	return vector, nil
}

func (r *SQLiteRepository) Frontier(ctx context.Context) ([]Summary, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT s.id,s.body,s.facet_id,s.vector_blob,s.vector_json,s.depth,s.generation,s.created_at FROM summaries s LEFT JOIN tree_edges e ON e.child_id=s.id AND e.child_kind='summary' WHERE s.scope=? AND e.parent_id IS NULL ORDER BY s.created_at,s.id`, policy.ScopeFromContext(ctx))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Summary
	for rows.Next() {
		var s Summary
		var created, raw string
		var blob []byte
		if err := rows.Scan(&s.ID, &s.Body, &s.FacetID, &blob, &raw, &s.Depth, &s.Generation, &created); err != nil {
			return nil, err
		}
		var err error
		if len(blob) > 0 {
			s.Vector, err = vectorcodec.Decode(blob)
		} else if raw != "" {
			err = json.Unmarshal([]byte(raw), &s.Vector)
		}
		if err != nil {
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
	scope := policy.ScopeFromContext(ctx)
	if _, err = tx.ExecContext(ctx, `DELETE FROM tree_edges WHERE parent_id IN (SELECT id FROM summaries WHERE scope=?)`, scope); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM summaries WHERE scope=?`, scope); err != nil {
		return err
	}
	return tx.Commit()
}
