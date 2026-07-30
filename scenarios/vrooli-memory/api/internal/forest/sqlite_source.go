package forest

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// SQLiteCandidateSource projects only root nodes whose current facet policy
// permits compaction. Policy remains owned by facets; forest only applies it.
type SQLiteCandidateSource struct{ db *sql.DB }

func NewSQLiteCandidateSource(db *sql.DB) *SQLiteCandidateSource {
	return &SQLiteCandidateSource{db: db}
}

func (s *SQLiteCandidateSource) CompactionCandidates(ctx context.Context) ([]Candidate, error) {
	rows, err := s.db.QueryContext(ctx, `
WITH latest_assignment AS (
  SELECT fa.entry_id, fa.facet_id FROM facet_assignments fa
  WHERE fa.id=(SELECT newer.id FROM facet_assignments newer WHERE newer.entry_id=fa.entry_id ORDER BY newer.assigned_at DESC,newer.id DESC LIMIT 1)
)
SELECT e.id,la.facet_id,e.body,COALESCE((SELECT em.vector_json FROM facet_texts ft JOIN embeddings em ON em.facet_text_id=ft.id WHERE ft.entry_id=e.id ORDER BY ft.id LIMIT 1),'[]'),e.created_at,'entry',0,0
FROM entries e JOIN latest_assignment la ON la.entry_id=e.id JOIN facet_policies fp ON fp.facet_id=la.facet_id
WHERE fp.compaction_eligible=1 AND NOT EXISTS(SELECT 1 FROM pins p WHERE p.entry_id=e.id) AND NOT EXISTS(SELECT 1 FROM tree_edges te WHERE te.child_id=e.id AND te.child_kind='entry')
UNION ALL
SELECT sm.id,sm.facet_id,sm.body,sm.vector_json,sm.created_at,'summary',sm.depth,sm.generation
FROM summaries sm JOIN facet_policies fp ON fp.facet_id=sm.facet_id
WHERE fp.compaction_eligible=1 AND NOT EXISTS(SELECT 1 FROM tree_edges te WHERE te.child_id=sm.id AND te.child_kind='summary')
ORDER BY 5,1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Candidate
	for rows.Next() {
		var c Candidate
		var raw, created string
		if err := rows.Scan(&c.ID, &c.FacetID, &c.Body, &raw, &created, &c.Kind, &c.Depth, &c.Generation); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(raw), &c.Vector); err != nil {
			return nil, fmt.Errorf("decode compaction vector: %w", err)
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		out = append(out, c)
	}
	return out, rows.Err()
}

var _ CandidateSource = (*SQLiteCandidateSource)(nil)
