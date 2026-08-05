package forest

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	vectorcodec "vrooli-memory/internal/vector"
)

// SQLiteCandidateSource projects only root nodes whose current facet policy
// permits compaction. Policy remains owned by facets; forest only applies it.
type SQLiteCandidateSource struct{ db *sql.DB }

func NewSQLiteCandidateSource(db *sql.DB) *SQLiteCandidateSource {
	return &SQLiteCandidateSource{db: db}
}

func (s *SQLiteCandidateSource) CompactionCandidates(ctx context.Context) ([]Candidate, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	rows, err := s.db.QueryContext(ctx, `
WITH latest_assignment AS (
  -- One pass with a window function. The correlated form this replaced ran a
  -- per-row subquery over facet_assignments and cost 0.33s on a 3.2k-row table,
  -- which every wake and recall paid before touching a single memory.
  SELECT entry_id, facet_id FROM (
    SELECT entry_id, facet_id, ROW_NUMBER() OVER (PARTITION BY entry_id ORDER BY assigned_at DESC, id DESC) AS rn
    FROM facet_assignments
  ) WHERE rn=1
)
SELECT e.id,la.facet_id,e.body,fp.retention_policy,fp.compaction_eligible,e.created_at,'entry',0,0
FROM entries e JOIN latest_assignment la ON la.entry_id=e.id JOIN facet_policies fp ON fp.facet_id=la.facet_id
WHERE fp.compaction_eligible=1 AND NOT EXISTS(SELECT 1 FROM pins p WHERE p.entry_id=e.id AND (p.review_at IS NULL OR p.review_at>?)) AND NOT EXISTS(SELECT 1 FROM tree_edges te WHERE te.child_id=e.id AND te.child_kind='entry')
UNION ALL
SELECT sm.id,sm.facet_id,sm.body,fp.retention_policy,fp.compaction_eligible,sm.created_at,'summary',sm.depth,sm.generation
FROM summaries sm JOIN facet_policies fp ON fp.facet_id=sm.facet_id
WHERE fp.compaction_eligible=1 AND NOT EXISTS(SELECT 1 FROM tree_edges te WHERE te.child_id=sm.id AND te.child_kind='summary')
ORDER BY created_at,id`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Candidate
	for rows.Next() {
		var c Candidate
		var created string
		if err := rows.Scan(&c.ID, &c.FacetID, &c.Body, &c.RetentionPolicy, &c.Compactable, &created, &c.Kind, &c.Depth, &c.Generation); err != nil {
			return nil, err
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	vectors, err := s.vectorsByCandidate(ctx)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].Vectors = vectors[out[i].ID]
	}
	return out, nil
}

// vectorsByCandidate loads every derived facet embedding in one pass so pair
// scoring can compare each space against its counterpart. The previous
// per-candidate subquery returned only the first space, so clustering could
// group by exactly one notion of relatedness.
func (s *SQLiteCandidateSource) vectorsByCandidate(ctx context.Context) (map[string][][]float64, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT ft.entry_id, em.vector_blob, em.vector_json
FROM facet_texts ft JOIN embeddings em ON em.facet_text_id=ft.id
UNION ALL
SELECT sm.id, sm.vector_blob, sm.vector_json FROM summaries sm
ORDER BY 1`)
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
		var vector []float64
		switch {
		case len(blob) > 0:
			vector, err = vectorcodec.Decode(blob)
		case raw != "" && raw != "[]":
			err = json.Unmarshal([]byte(raw), &vector)
		default:
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("decode compaction vector: %w", err)
		}
		out[id] = append(out[id], vector)
	}
	return out, rows.Err()
}

var _ CandidateSource = (*SQLiteCandidateSource)(nil)
