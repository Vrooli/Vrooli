package recall

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"source-ledger/internal/policy"
	vectorcodec "source-ledger/internal/vector"
)

// SQLiteSource projects the persistent journal and pin state into recall
// candidates. Forest summaries join this same source in the forest phase.
type SQLiteSource struct{ db *sql.DB }

func NewSQLiteSource(db *sql.DB) *SQLiteSource { return &SQLiteSource{db: db} }

// Liveness reports the portion of a configured scope's compaction-eligible
// forest that has not yet been absorbed into a forest parent. It uses the same
// policy, age, pin, root, and vector gates as compaction so frontier_target is
// an observable target rather than a comparison against durable records that
// compaction is forbidden to summarize. Policy-less fixtures retain the legacy
// raw-leaf behavior for backwards-compatible health observations.
func (s *SQLiteSource) Liveness(ctx context.Context) (CompactionLiveness, error) {
	scope := policy.ScopeFromContext(ctx)
	var result CompactionLiveness
	var policyCount int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM facet_policies WHERE scope=?`, scope).Scan(&policyCount); err != nil {
		return result, err
	}
	query := `
SELECT COUNT(*), COALESCE(MIN(e.created_at), ''),
       COALESCE((SELECT MAX(created_at) FROM summaries WHERE scope=?), '')
FROM entries e
WHERE e.scope=?

  AND NOT EXISTS (SELECT 1 FROM tree_edges WHERE child_id=e.id AND child_kind='entry')`
	args := []any{scope, scope}
	if policyCount > 0 {
		cutoff := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339Nano)
		query = `
WITH latest_assignment AS (
  SELECT entry_id, facet_id FROM (
    SELECT entry_id, facet_id, ROW_NUMBER() OVER (PARTITION BY entry_id ORDER BY assigned_at DESC, id DESC) AS rn
    FROM facet_assignments
  ) WHERE rn=1
), eligible_roots AS (
  SELECT e.created_at
  FROM entries e
  JOIN latest_assignment la ON la.entry_id=e.id
  JOIN facet_policies fp ON fp.scope=? AND fp.facet_id=la.facet_id AND fp.compaction_eligible=1
  WHERE e.scope=?
    AND e.created_at<=?
    AND NOT EXISTS (SELECT 1 FROM tree_edges WHERE child_id=e.id AND child_kind='entry')
    AND NOT EXISTS (SELECT 1 FROM pins p WHERE p.entry_id=e.id AND (p.review_at IS NULL OR p.review_at>?))
    AND EXISTS (
      SELECT 1 FROM facet_texts ft JOIN embeddings em ON em.facet_text_id=ft.id
      WHERE ft.entry_id=e.id AND (length(em.vector_blob)>0 OR (em.vector_json<>'' AND em.vector_json<>'[]'))
    )
  UNION ALL
  SELECT sm.created_at
  FROM summaries sm
  JOIN facet_policies fp ON fp.scope=? AND fp.facet_id=sm.facet_id AND fp.compaction_eligible=1
  WHERE sm.scope=?
    AND sm.created_at<=?
    AND NOT EXISTS (SELECT 1 FROM tree_edges WHERE child_id=sm.id AND child_kind='summary')
    AND (length(sm.vector_blob)>0 OR (sm.vector_json<>'' AND sm.vector_json<>'[]'))
)
SELECT COUNT(*), COALESCE(MIN(created_at), ''),
       COALESCE((SELECT MAX(created_at) FROM summaries WHERE scope=?), '')
FROM eligible_roots`
		args = []any{scope, scope, cutoff, cutoff, scope, scope, cutoff, scope}
	}
	err := s.db.QueryRowContext(ctx, query, args...).
		Scan(&result.UnsummarizedLeafCount, &result.OldestUnsummarizedLeafAt, &result.LastSummaryAt)
	return result, err
}

func (s *SQLiteSource) Nodes(ctx context.Context) ([]Node, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	scope := policy.ScopeFromContext(ctx)
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
SELECT e.id,
       COALESCE((SELECT parent_id FROM tree_edges WHERE child_id=e.id AND child_kind='entry' ORDER BY parent_id LIMIT 1),''),
       e.id,COALESCE(la.facet_id,e.facet_id),e.body,e.created_at,
       EXISTS(SELECT 1 FROM pins p WHERE p.entry_id=e.id AND (p.review_at IS NULL OR p.review_at>?)),
       NOT EXISTS(SELECT 1 FROM tree_edges WHERE child_id=e.id AND child_kind='entry'),
       0,
       0,
       0
-- The authority for an entry's facet is its latest assignment, which is how
-- compaction already reads it. entries.facet_id records only what the write
-- path guessed at append time and is never revised, so reading it here made
-- corrected and late-classified memories invisible to wake.
FROM entries e LEFT JOIN latest_assignment la ON la.entry_id=e.id
WHERE e.scope=? AND NOT (EXISTS(SELECT 1 FROM facet_policies fp WHERE fp.scope=? AND fp.facet_id=COALESCE(la.facet_id,e.facet_id) AND fp.retention_policy=?) AND EXISTS(SELECT 1 FROM marks m WHERE m.entry_id=e.id AND m.kind='resolved'))
UNION ALL
SELECT s.id,
       COALESCE((SELECT parent_id FROM tree_edges WHERE child_id=s.id AND child_kind='summary' ORDER BY parent_id LIMIT 1),''),
       s.id,s.facet_id,s.body,s.created_at,
       0,
       NOT EXISTS(SELECT 1 FROM tree_edges WHERE child_id=s.id AND child_kind='summary'),
       s.depth,
       (SELECT COUNT(*) FROM tree_edges children WHERE children.parent_id=s.id),
       1
FROM summaries s
WHERE s.scope=?
ORDER BY 6,1`, now, scope, scope, policy.ExpireOnResolution, scope)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Node
	for rows.Next() {
		var n Node
		var created string
		if err = rows.Scan(&n.ID, &n.ParentID, &n.EntryID, &n.FacetID, &n.Text, &created, &n.Pinned, &n.Frontier, &n.Depth, &n.Span, &n.Summary); err != nil {
			return nil, err
		}
		n.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	vectors, err := s.vectorsByNode(ctx)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].Vectors = vectors[out[i].ID]
	}
	return out, nil
}

// vectorsByNode loads every derived facet embedding in one pass. Fetching them
// per node through correlated subqueries returned only the first space, which
// left two thirds of the embedding corpus written and never read.
func (s *SQLiteSource) vectorsByNode(ctx context.Context) (map[string][][]float64, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT ft.entry_id, em.vector_blob, em.vector_json
FROM facet_texts ft JOIN embeddings em ON em.facet_text_id=ft.id JOIN entries e ON e.id=ft.entry_id
WHERE e.scope=?
UNION ALL
SELECT sm.id, sm.vector_blob, sm.vector_json FROM summaries sm WHERE sm.scope=?
ORDER BY 1`, policy.ScopeFromContext(ctx), policy.ScopeFromContext(ctx))
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
			return nil, err
		}
		out[id] = append(out[id], vector)
	}
	return out, rows.Err()
}

var _ Source = (*SQLiteSource)(nil)

// AmbientNodes returns only what wake renders: pinned entries and unabsorbed
// frontier nodes, with no embeddings loaded at all. Wake scores no similarity,
// so its cost now tracks the frontier it renders rather than the whole corpus.
func (s *SQLiteSource) AmbientNodes(ctx context.Context, perFacetLimit int) ([]Node, error) {
	if perFacetLimit <= 0 {
		perFacetLimit = 1
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	scope := policy.ScopeFromContext(ctx)
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
SELECT id,entry_id,facet_id,body,created_at,pinned,depth,span,summary FROM (
SELECT e.id AS id,e.id AS entry_id,COALESCE(la.facet_id,e.facet_id) AS facet_id,e.body AS body,e.created_at AS created_at,
       EXISTS(SELECT 1 FROM pins p WHERE p.entry_id=e.id AND (p.review_at IS NULL OR p.review_at>?)) AS pinned,
       0 AS depth,0 AS span,0 AS summary,
       ROW_NUMBER() OVER (PARTITION BY COALESCE(la.facet_id,e.facet_id) ORDER BY e.created_at DESC, e.id) AS rank
FROM entries e LEFT JOIN latest_assignment la ON la.entry_id=e.id
WHERE e.scope=? AND (NOT EXISTS(SELECT 1 FROM tree_edges WHERE child_id=e.id AND child_kind='entry')
       OR EXISTS(SELECT 1 FROM pins p WHERE p.entry_id=e.id AND (p.review_at IS NULL OR p.review_at>?)))
  AND NOT (EXISTS(SELECT 1 FROM facet_policies fp WHERE fp.scope=? AND fp.facet_id=COALESCE(la.facet_id,e.facet_id) AND fp.retention_policy=?)
           AND EXISTS(SELECT 1 FROM marks m WHERE m.entry_id=e.id AND m.kind='resolved'))
UNION ALL
SELECT s.id,s.id,s.facet_id,s.body,s.created_at,0,s.depth,
       (SELECT COUNT(*) FROM tree_edges children WHERE children.parent_id=s.id),1,
       ROW_NUMBER() OVER (PARTITION BY s.facet_id ORDER BY s.created_at DESC, s.id)
FROM summaries s
WHERE s.scope=? AND NOT EXISTS(SELECT 1 FROM tree_edges WHERE child_id=s.id AND child_kind='summary')
) WHERE pinned=1 OR rank<=?
ORDER BY created_at DESC,id`, now, scope, now, scope, policy.ExpireOnResolution, scope, perFacetLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Node
	for rows.Next() {
		var n Node
		var created string
		if err := rows.Scan(&n.ID, &n.EntryID, &n.FacetID, &n.Text, &created, &n.Pinned, &n.Depth, &n.Span, &n.Summary); err != nil {
			return nil, err
		}
		n.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		// Every row this query returns is either pinned or unabsorbed.
		n.Frontier = !n.Pinned
		out = append(out, n)
	}
	return out, rows.Err()
}
