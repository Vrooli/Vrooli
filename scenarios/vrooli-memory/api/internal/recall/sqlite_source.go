package recall

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

// SQLiteSource projects the persistent journal and pin state into recall
// candidates. Forest summaries join this same source in the forest phase.
type SQLiteSource struct{ db *sql.DB }

func NewSQLiteSource(db *sql.DB) *SQLiteSource { return &SQLiteSource{db: db} }
func (s *SQLiteSource) Nodes(ctx context.Context) ([]Node, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT e.id,e.id,e.facet_id,e.body,e.created_at,EXISTS(SELECT 1 FROM pins p WHERE p.entry_id=e.id),COALESCE((SELECT em.vector_json FROM facet_texts ft JOIN embeddings em ON em.facet_text_id=ft.id WHERE ft.entry_id=e.id ORDER BY ft.id LIMIT 1),'[]') FROM entries e ORDER BY e.created_at,e.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Node
	for rows.Next() {
		var n Node
		var created, raw string
		if err = rows.Scan(&n.ID, &n.EntryID, &n.FacetID, &n.Text, &created, &n.Pinned, &raw); err != nil {
			return nil, err
		}
		n.Frontier = true
		n.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		if err = json.Unmarshal([]byte(raw), &n.Vector); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

var _ Source = (*SQLiteSource)(nil)
