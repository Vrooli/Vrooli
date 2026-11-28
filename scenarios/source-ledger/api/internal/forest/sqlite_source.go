package forest

import (
	"context"
	"database/sql"
	"time"

	"source-ledger/internal/policy"
	vectorcodec "source-ledger/internal/vector"
)

// SQLiteCandidateSource projects only root nodes whose current facet policy
// permits compaction. Policy remains owned by facets; forest only applies it.
type SQLiteCandidateSource struct {
	db          *sql.DB
	generations *vectorcodec.SQLiteGenerationServing
}

func NewSQLiteCandidateSource(db *sql.DB, generations ...*vectorcodec.SQLiteGenerationServing) *SQLiteCandidateSource {
	source := &SQLiteCandidateSource{db: db}
	if len(generations) > 0 {
		source.generations = generations[0]
	}
	return source
}

func (s *SQLiteCandidateSource) CompactionCandidates(ctx context.Context) ([]Candidate, error) {
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
SELECT e.id,la.facet_id,e.body,fp.retention_policy,fp.compaction_eligible,e.created_at,'entry',0,0
FROM entries e JOIN latest_assignment la ON la.entry_id=e.id JOIN facet_policies fp ON fp.scope=? AND fp.facet_id=la.facet_id
WHERE e.scope=? AND fp.compaction_eligible=1 AND NOT EXISTS(SELECT 1 FROM journal_observation_context oc WHERE oc.entry_id=e.id AND oc.routing_hint='defect-account') AND NOT EXISTS(SELECT 1 FROM pins p WHERE p.entry_id=e.id AND (p.review_at IS NULL OR p.review_at>?)) AND NOT EXISTS(SELECT 1 FROM tree_edges te WHERE te.child_id=e.id AND te.child_kind='entry')
UNION ALL
SELECT sm.id,sm.facet_id,sm.body,fp.retention_policy,fp.compaction_eligible,sm.created_at,'summary',sm.depth,sm.generation
FROM summaries sm JOIN facet_policies fp ON fp.scope=? AND fp.facet_id=sm.facet_id
WHERE sm.scope=? AND fp.compaction_eligible=1 AND NOT EXISTS(SELECT 1 FROM tree_edges te WHERE te.child_id=sm.id AND te.child_kind='summary')
ORDER BY created_at,id`, scope, scope, now, scope, scope)
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
	// Every vector space of a candidate, so pair scoring compares each space
	// with its counterpart rather than one notion of relatedness.
	vectors, err := vectorcodec.EntryVectors(ctx, s.db, s.generations, string(policy.ScopeFromContext(ctx)))
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].Vectors = vectors[out[i].ID]
	}
	return out, nil
}

var _ CandidateSource = (*SQLiteCandidateSource)(nil)
