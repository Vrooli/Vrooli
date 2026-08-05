package recall

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	vectorcodec "vrooli-memory/internal/vector"
)

// SQLiteSource projects the persistent journal and pin state into recall
// candidates. Forest summaries join this same source in the forest phase.
type SQLiteSource struct{ db *sql.DB }

func NewSQLiteSource(db *sql.DB) *SQLiteSource { return &SQLiteSource{db: db} }
func (s *SQLiteSource) Nodes(ctx context.Context) ([]Node, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	rows, err := s.db.QueryContext(ctx, `
SELECT e.id,
       COALESCE((SELECT parent_id FROM tree_edges WHERE child_id=e.id AND child_kind='entry' ORDER BY parent_id LIMIT 1),''),
       e.id,e.facet_id,e.body,e.created_at,
       EXISTS(SELECT 1 FROM pins p WHERE p.entry_id=e.id AND (p.review_at IS NULL OR p.review_at>?)),
       NOT EXISTS(SELECT 1 FROM tree_edges WHERE child_id=e.id AND child_kind='entry'),
       COALESCE((SELECT em.vector_blob FROM facet_texts ft JOIN embeddings em ON em.facet_text_id=ft.id WHERE ft.entry_id=e.id ORDER BY ft.id LIMIT 1),X''),
       COALESCE((SELECT em.vector_json FROM facet_texts ft JOIN embeddings em ON em.facet_text_id=ft.id WHERE ft.entry_id=e.id ORDER BY ft.id LIMIT 1),'[]'),
       0,
       0,
       0
FROM entries e
WHERE NOT (EXISTS(SELECT 1 FROM facet_policies fp WHERE fp.facet_id=e.facet_id AND fp.retention_policy='expire-on-resolution') AND EXISTS(SELECT 1 FROM marks m WHERE m.entry_id=e.id AND m.kind='resolved'))
UNION ALL
SELECT s.id,
       COALESCE((SELECT parent_id FROM tree_edges WHERE child_id=s.id AND child_kind='summary' ORDER BY parent_id LIMIT 1),''),
       s.id,s.facet_id,s.body,s.created_at,
       0,
       NOT EXISTS(SELECT 1 FROM tree_edges WHERE child_id=s.id AND child_kind='summary'),
       s.vector_blob,
       s.vector_json,
       s.depth,
       (SELECT COUNT(*) FROM tree_edges children WHERE children.parent_id=s.id),
       1
FROM summaries s
ORDER BY 6,1`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Node
	for rows.Next() {
		var n Node
		var created, raw string
		var blob []byte
		if err = rows.Scan(&n.ID, &n.ParentID, &n.EntryID, &n.FacetID, &n.Text, &created, &n.Pinned, &n.Frontier, &blob, &raw, &n.Depth, &n.Span, &n.Summary); err != nil {
			return nil, err
		}
		n.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		if len(blob) > 0 {
			n.Vector, err = vectorcodec.Decode(blob)
		} else if raw != "" {
			err = json.Unmarshal([]byte(raw), &n.Vector)
		}
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

var _ Source = (*SQLiteSource)(nil)
