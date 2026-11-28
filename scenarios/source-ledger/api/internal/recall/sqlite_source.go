package recall

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/vrooli/api-core/database"

	"source-ledger/internal/policy"
	vectorcodec "source-ledger/internal/vector"
)

// SQLiteSource projects the persistent journal and pin state into recall
// candidates. Forest summaries join this same source in the forest phase.
type sqliteReader interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type SQLiteSource struct {
	db          sqliteReader
	generations *vectorcodec.SQLiteGenerationServing
	vectors     vectorSnapshot
}

func NewSQLiteSource(db sqliteReader, generations ...*vectorcodec.SQLiteGenerationServing) *SQLiteSource {
	source := &SQLiteSource{db: db}
	if len(generations) > 0 {
		source.generations = generations[0]
	}
	return source
}

func (s *SQLiteSource) ActiveRecipeID(ctx context.Context) (string, bool, error) {
	if s.generations == nil {
		return "", false, nil
	}
	return s.generations.ActiveRecipeID(ctx)
}

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
    SELECT fa.entry_id, fa.facet_id, ROW_NUMBER() OVER (PARTITION BY fa.entry_id ORDER BY fa.assigned_at DESC, fa.id DESC) AS rn
    FROM facet_assignments fa
    JOIN entries scoped_entry ON scoped_entry.id=fa.entry_id AND scoped_entry.scope=?
  ) WHERE rn=1
), eligible_roots AS (
  SELECT e.created_at
  FROM entries e
  JOIN latest_assignment la ON la.entry_id=e.id
  JOIN facet_policies fp ON fp.scope=? AND fp.facet_id=la.facet_id AND fp.compaction_eligible=1
  WHERE e.scope=?
    AND NOT EXISTS(SELECT 1 FROM journal_observation_context oc WHERE oc.entry_id=e.id AND oc.routing_hint='defect-account')
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
		args = []any{scope, scope, scope, cutoff, cutoff, scope, scope, cutoff, scope}
	}
	err := s.db.QueryRowContext(ctx, query, args...).
		Scan(&result.UnsummarizedLeafCount, &result.OldestUnsummarizedLeafAt, &result.LastSummaryAt)
	return result, err
}

func (s *SQLiteSource) Nodes(ctx context.Context) ([]Node, error) {
	nodes, err := s.TextNodes(ctx)
	if err != nil {
		return nil, err
	}
	vectors, err := s.NodeVectors(ctx)
	if err != nil {
		return nil, err
	}
	setNodeVectors(nodes, vectors)
	return nodes, nil
}

func (s *SQLiteSource) NodeVectors(ctx context.Context) (map[string][][]float64, error) {
	scope := string(policy.ScopeFromContext(ctx))
	load := func() (map[string][][]float64, error) {
		return vectorcodec.EntryVectors(ctx, s.db, s.generations, scope)
	}
	if s.generations == nil || database.IsTestMode(ctx) {
		return load()
	}
	return s.vectors.get(ctx, scope, s.db, load)
}

func setNodeVectors(nodes []Node, vectors map[string][][]float64) {
	for i := range nodes {
		nodes[i].Vectors = vectors[nodes[i].EntryID]
	}
}

// TextNodes projects the scoped canopy without reading the embedding corpus.
func (s *SQLiteSource) TextNodes(ctx context.Context) ([]Node, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	scope := policy.ScopeFromContext(ctx)
	rows, err := s.db.QueryContext(ctx, `
WITH RECURSIVE scoped_facets AS MATERIALIZED (
  -- The latest-assignment index makes one bounded lookup per entry cheaper
  -- than ranking the entire assignment history. Materialize only IDs/facets.
  SELECT e.id,COALESCE((
    SELECT fa.facet_id FROM facet_assignments fa WHERE fa.entry_id=e.id
    ORDER BY fa.assigned_at DESC,fa.id DESC LIMIT 1
  ),e.facet_id) AS facet
  FROM entries e WHERE e.scope=?
	), superseded_entries(id) AS (
	  SELECT DISTINCT e.id
	  FROM entries e JOIN marks m ON m.entry_id=e.id
	  WHERE e.scope=? AND m.kind='superseded'
	), tainted_summaries(id) AS (
	  SELECT te.parent_id
	  FROM tree_edges te
	  JOIN superseded_entries se ON te.child_kind='entry' AND te.child_id=se.id
	  JOIN summaries parent ON parent.id=te.parent_id AND parent.scope=?
	  UNION
	  SELECT te.parent_id
	  FROM tree_edges te
	  JOIN tainted_summaries child ON te.child_kind='summary' AND te.child_id=child.id
	  JOIN summaries parent ON parent.id=te.parent_id AND parent.scope=?
	)
SELECT e.id,
       COALESCE((SELECT CASE WHEN EXISTS(SELECT 1 FROM tainted_summaries ts WHERE ts.id=te.parent_id) THEN '' ELSE te.parent_id END FROM tree_edges te WHERE te.child_id=e.id AND te.child_kind='entry' ORDER BY te.parent_id LIMIT 1),''),
       e.id,sf.facet,e.body,e.created_at,
       EXISTS(SELECT 1 FROM pins p WHERE p.entry_id=e.id AND (p.review_at IS NULL OR p.review_at>?)),
       NOT EXISTS(SELECT 1 FROM tree_edges WHERE child_id=e.id AND child_kind='entry')
         OR EXISTS(SELECT 1 FROM tree_edges te JOIN tainted_summaries ts ON ts.id=te.parent_id WHERE te.child_id=e.id AND te.child_kind='entry'),
       0,
       0,
       0
FROM scoped_facets sf JOIN entries e ON e.id=sf.id
WHERE NOT EXISTS(SELECT 1 FROM superseded_entries se WHERE se.id=e.id)
  AND NOT (EXISTS(SELECT 1 FROM facet_policies fp WHERE fp.scope=e.scope AND fp.facet_id=sf.facet AND fp.retention_policy=?) AND EXISTS(SELECT 1 FROM marks m WHERE m.entry_id=e.id AND m.kind='resolved'))
UNION ALL
SELECT s.id,
       COALESCE((SELECT CASE WHEN EXISTS(SELECT 1 FROM tainted_summaries ts WHERE ts.id=te.parent_id) THEN '' ELSE te.parent_id END FROM tree_edges te WHERE te.child_id=s.id AND te.child_kind='summary' ORDER BY te.parent_id LIMIT 1),''),
       s.id,s.facet_id,s.body,s.created_at,
       0,
       NOT EXISTS(SELECT 1 FROM tree_edges WHERE child_id=s.id AND child_kind='summary')
         OR EXISTS(SELECT 1 FROM tree_edges te JOIN tainted_summaries ts ON ts.id=te.parent_id WHERE te.child_id=s.id AND te.child_kind='summary'),
       s.depth,
       (SELECT COUNT(*) FROM tree_edges children WHERE children.parent_id=s.id),
       1
FROM summaries s
WHERE s.scope=? AND NOT EXISTS(SELECT 1 FROM tainted_summaries ts WHERE ts.id=s.id)`, scope, scope, scope, scope, now, policy.ExpireOnResolution, scope)
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
	if err := rows.Close(); err != nil {
		return nil, err
	}
	accounts, publications, err := s.currentAccounts(ctx)
	if err != nil {
		return nil, err
	}
	return projectAccounts(out, accounts, publications), nil
}

func projectAccounts(nodes []Node, accounts map[string]accountEvidence, publications map[string]bool) []Node {
	current := nodes[:0]
	for _, n := range nodes {
		if account, ok := accounts[n.EntryID]; ok {
			n.applyAccount(account)
		} else if publications[n.EntryID] {
			continue
		}
		current = append(current, n)
	}
	return current
}

// CategoryNodes only reads eligible originals and the current case accounts.
// Category memberships inherited by general canopy summaries are useful for
// topical recall but would bury observations in a category-only listing.
func (s *SQLiteSource) CategoryNodes(ctx context.Context, matches map[string][]string) ([]Node, error) {
	accounts, publications, err := s.currentAccounts(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(matches)+len(accounts))
	for id := range matches {
		ids = append(ids, id)
	}
	for id := range accounts {
		if _, exists := matches[id]; !exists {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	candidates, err := json.Marshal(ids)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
WITH candidates AS (
  SELECT e.id,e.scope,e.body,e.created_at,COALESCE((
    SELECT fa.facet_id FROM facet_assignments fa WHERE fa.entry_id=e.id
    ORDER BY fa.assigned_at DESC,fa.id DESC LIMIT 1
  ),e.facet_id) AS facet
  FROM entries e
  WHERE e.scope=? AND e.id IN (SELECT value FROM json_each(?))
  AND NOT EXISTS(SELECT 1 FROM marks m WHERE m.entry_id=e.id AND m.kind='superseded')
)
SELECT id,body,created_at,facet
FROM candidates c
WHERE NOT (
  EXISTS(SELECT 1 FROM facet_policies fp WHERE fp.scope=c.scope AND fp.facet_id=c.facet AND fp.retention_policy=?)
  AND EXISTS(SELECT 1 FROM marks m WHERE m.entry_id=c.id AND m.kind='resolved')
)
ORDER BY created_at,id`, policy.ScopeFromContext(ctx), string(candidates), policy.ExpireOnResolution)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	nodes := make([]Node, 0, len(ids))
	for rows.Next() {
		var n Node
		var created string
		if err := rows.Scan(&n.ID, &n.Text, &created, &n.FacetID); err != nil {
			return nil, err
		}
		n.EntryID = n.ID
		n.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		nodes = append(nodes, n)
	}
	return projectAccounts(nodes, accounts, publications), rows.Err()
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
	accounts, publications, err := s.currentAccounts(ctx)
	if err != nil {
		return nil, err
	}
	hidden := make(map[string]bool, len(publications))
	for id := range publications {
		if _, current := accounts[id]; !current {
			hidden[id] = true
		}
	}
	for _, account := range accounts {
		for _, id := range account.ReconciledIDs {
			if _, current := accounts[id]; !current {
				hidden[id] = true
			}
		}
	}
	hiddenIDs := make([]string, 0, len(hidden))
	for id := range hidden {
		hiddenIDs = append(hiddenIDs, id)
	}
	hiddenJSON, err := json.Marshal(hiddenIDs)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
WITH RECURSIVE latest_assignment AS (
  -- One pass with a window function. The correlated form this replaced ran a
  -- per-row subquery over facet_assignments and cost 0.33s on a 3.2k-row table,
  -- which every wake and recall paid before touching a single memory.
  SELECT entry_id, facet_id FROM (
    SELECT fa.entry_id, fa.facet_id, ROW_NUMBER() OVER (PARTITION BY fa.entry_id ORDER BY fa.assigned_at DESC, fa.id DESC) AS rn
    FROM facet_assignments fa
    JOIN entries scoped_entry ON scoped_entry.id=fa.entry_id AND scoped_entry.scope=?
  ) WHERE rn=1
	), superseded_entries(id) AS (
	  SELECT DISTINCT e.id
	  FROM entries e JOIN marks m ON m.entry_id=e.id
	  WHERE e.scope=? AND m.kind='superseded'
	  UNION SELECT value FROM json_each(?)
	  -- Old imports have a fresh append timestamp. Keep them searchable in
	  -- recall without displacing recent working context at session start.
	  UNION SELECT id FROM entries
	  WHERE scope=? AND kind GLOB 'legacy-bug-*' AND import_key GLOB 'request:legacy-scenario-qa-bugs/v1/*'
	), tainted_summaries(id) AS (
	  SELECT te.parent_id
	  FROM tree_edges te
	  JOIN superseded_entries se ON te.child_kind='entry' AND te.child_id=se.id
	  JOIN summaries parent ON parent.id=te.parent_id AND parent.scope=?
	  UNION
	  SELECT te.parent_id
	  FROM tree_edges te
	  JOIN tainted_summaries child ON te.child_kind='summary' AND te.child_id=child.id
	  JOIN summaries parent ON parent.id=te.parent_id AND parent.scope=?
	)
SELECT id,entry_id,facet_id,body,created_at,pinned,depth,span,summary FROM (
SELECT e.id AS id,e.id AS entry_id,COALESCE(la.facet_id,e.facet_id) AS facet_id,e.body AS body,e.created_at AS created_at,
       EXISTS(SELECT 1 FROM pins p WHERE p.entry_id=e.id AND (p.review_at IS NULL OR p.review_at>?)) AS pinned,
       0 AS depth,0 AS span,0 AS summary,
       ROW_NUMBER() OVER (PARTITION BY COALESCE(la.facet_id,e.facet_id) ORDER BY e.created_at DESC, e.id) AS rank
FROM entries e LEFT JOIN latest_assignment la ON la.entry_id=e.id
WHERE e.scope=? AND NOT EXISTS(SELECT 1 FROM superseded_entries se WHERE se.id=e.id)
  AND (NOT EXISTS(SELECT 1 FROM tree_edges WHERE child_id=e.id AND child_kind='entry')
       OR EXISTS(SELECT 1 FROM pins p WHERE p.entry_id=e.id AND (p.review_at IS NULL OR p.review_at>?))
       OR EXISTS(SELECT 1 FROM tree_edges te JOIN tainted_summaries ts ON ts.id=te.parent_id WHERE te.child_kind='entry' AND te.child_id=e.id))
  AND NOT (EXISTS(SELECT 1 FROM facet_policies fp WHERE fp.scope=? AND fp.facet_id=COALESCE(la.facet_id,e.facet_id) AND fp.retention_policy=?)
           AND EXISTS(SELECT 1 FROM marks m WHERE m.entry_id=e.id AND m.kind='resolved'))
UNION ALL
SELECT s.id,s.id,s.facet_id,s.body,s.created_at,0,s.depth,
       (SELECT COUNT(*) FROM tree_edges children WHERE children.parent_id=s.id),1,
       ROW_NUMBER() OVER (PARTITION BY s.facet_id ORDER BY s.created_at DESC, s.id)
FROM summaries s
WHERE s.scope=? AND NOT EXISTS(SELECT 1 FROM tainted_summaries ts WHERE ts.id=s.id)
  AND (NOT EXISTS(SELECT 1 FROM tree_edges WHERE child_id=s.id AND child_kind='summary')
       OR EXISTS(SELECT 1 FROM tree_edges te JOIN tainted_summaries ts ON ts.id=te.parent_id WHERE te.child_kind='summary' AND te.child_id=s.id))
) WHERE pinned=1 OR rank<=?
ORDER BY created_at DESC,id`, scope, scope, string(hiddenJSON), scope, scope, scope, now, scope, now, scope, policy.ExpireOnResolution, scope, perFacetLimit)
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
		if account, ok := accounts[n.EntryID]; ok {
			n.applyAccount(account)
		}
		out = append(out, n)
	}
	return out, rows.Err()
}
