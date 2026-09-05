package facets

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"source-ledger/internal/policy"

	"github.com/google/uuid"
)

const pinReviewInterval = 90 * 24 * time.Hour

type SQLiteRepository struct{ db *sql.DB }

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository { return &SQLiteRepository{db: db} }

var seeds = []Definition{
	{ID: "standing-rule", Label: "Standing rule", ClassificationGuidance: "A durable instruction, policy, invariant, or reusable guidance that should be followed across work.", RetentionPolicy: "pinned-or-review", ResidentBudget: 8},
	{ID: "environment-fact", Label: "Environment fact", ClassificationGuidance: "A stable fact about a host, runtime, tool, configuration, or operating environment.", RetentionPolicy: "retain", ResidentBudget: 4},
	{ID: "gotcha", Label: "Gotcha", ClassificationGuidance: "A recurring failure mode, trap, debugging lesson, or warning about what goes wrong.", RetentionPolicy: "retain", ResidentBudget: 4},
	{ID: "episode", Label: "Episode", ClassificationGuidance: "A completed project, implementation, validation, or historical work event with an outcome. Any work record with Trigger, Approach, Evidence, and Outcome fields is an episode, including partial, failed, or in-progress outcomes, even when it names a system or project.", RetentionPolicy: "compact", CompactionEligible: true, ResidentBudget: 12},
	{ID: "thread", Label: "Thread", ClassificationGuidance: "An active, unresolved line of work, investigation, or follow-up that is not complete.", RetentionPolicy: "expire-on-resolution", ResidentBudget: 4},
	{ID: "entity-record", Label: "Entity record", ClassificationGuidance: "A durable record describing the identity, ownership, or stable reference of a named system, scenario, person, or artifact; it is not a completed implementation work record.", RetentionPolicy: "retain", ResidentBudget: 4},
}

var retentionPolicies = map[string]struct{}{
	"retain":               {},
	"compact":              {},
	"expire-on-resolution": {},
	"pinned-or-review":     {},
}

func (r *SQLiteRepository) Seed(ctx context.Context) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, d := range seeds {
		if _, err = tx.ExecContext(ctx, `INSERT INTO facet_definitions(id,label,classification_guidance,created_at) VALUES(?,?,?,?) ON CONFLICT(id) DO UPDATE SET label=excluded.label,classification_guidance=excluded.classification_guidance`, d.ID, d.Label, d.ClassificationGuidance, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO facet_policies(facet_id,retention_policy,compaction_eligible,resident_budget) VALUES(?,?,?,?) ON CONFLICT(facet_id) DO UPDATE SET resident_budget=CASE WHEN facet_policies.resident_budget=0 THEN excluded.resident_budget ELSE facet_policies.resident_budget END`, d.ID, d.RetentionPolicy, d.CompactionEligible, d.ResidentBudget); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SeedExamples records a small, deterministic, corpus-derived example set for
// every facet. The examples are data owned by the facet tables, never prompt
// constants, and are inserted only when a given facet/entry pair is absent.
func (r *SQLiteRepository) SeedExamples(ctx context.Context) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM facet_examples WHERE scope='agent-memory'`); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
WITH latest AS (
  SELECT entry_id,facet_id,ROW_NUMBER() OVER (PARTITION BY entry_id ORDER BY assigned_at DESC,id DESC) rn
  FROM facet_assignments
), ranked AS (
  SELECT e.id,e.body,l.facet_id,ROW_NUMBER() OVER (PARTITION BY l.facet_id ORDER BY e.id) facet_rank
  FROM entries e JOIN latest l ON l.entry_id=e.id AND l.rn=1
  WHERE e.scope='agent-memory' AND trim(e.body)<>'' AND
    ((l.facet_id='episode' AND e.source_runtime='swarm-manager' AND e.kind='work-record')
      OR (l.facet_id<>'episode' AND e.kind<>'work-record'))
)
INSERT INTO facet_examples(id,scope,facet_id,entry_id,body,created_at)
SELECT 'corpus-example:'||facet_id||':'||id,'agent-memory',facet_id,id,body,?
FROM ranked WHERE facet_rank<=3`, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *SQLiteRepository) List(ctx context.Context) ([]Definition, error) {
	scope := policy.ScopeFromContext(ctx)
	rows, err := r.db.QueryContext(ctx, `SELECT d.id,d.label,d.classification_guidance,p.retention_policy,p.compaction_eligible,p.resident_budget FROM facet_definitions d JOIN facet_policies p ON p.facet_id=d.id AND p.scope=? WHERE d.scope=? ORDER BY d.id`, scope, scope)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Definition
	for rows.Next() {
		var d Definition
		if err = rows.Scan(&d.ID, &d.Label, &d.ClassificationGuidance, &d.RetentionPolicy, &d.CompactionEligible, &d.ResidentBudget); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i := range out {
		examples, err := r.examples(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].ClassificationExamples = examples
	}
	return out, nil
}

// SetPolicy changes only the mutable policy attached to an existing facet.
// Facet definitions and journal assignments remain historical data; changing
// a retention or residency rule never rewrites either one.
func (r *SQLiteRepository) SetPolicy(ctx context.Context, requested FacetPolicy) (Definition, error) {
	requested.ID = strings.TrimSpace(requested.ID)
	requested.RetentionPolicy = strings.TrimSpace(requested.RetentionPolicy)
	if requested.ID == "" {
		return Definition{}, errors.New("facet_id is required")
	}
	if _, ok := retentionPolicies[requested.RetentionPolicy]; !ok {
		return Definition{}, errors.New("retention_policy must be one of retain, compact, expire-on-resolution, or pinned-or-review")
	}
	if requested.ResidentBudget < 0 {
		return Definition{}, errors.New("resident_budget cannot be negative")
	}
	scope := policy.ScopeFromContext(ctx)
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Definition{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM facet_definitions WHERE id=? AND scope=?`, requested.ID, scope).Scan(&exists); err != nil {
		return Definition{}, err
	}
	if exists == 0 {
		return Definition{}, ErrUnknownFacet{ID: requested.ID}
	}
	result, err := tx.ExecContext(ctx, `UPDATE facet_policies SET retention_policy=?,compaction_eligible=?,resident_budget=? WHERE facet_id=? AND scope=?`, requested.RetentionPolicy, requested.CompactionEligible, requested.ResidentBudget, requested.ID, scope)
	if err != nil {
		return Definition{}, err
	}
	if affected, err := result.RowsAffected(); err != nil {
		return Definition{}, err
	} else if affected != 1 {
		return Definition{}, ErrUnknownFacet{ID: requested.ID}
	}
	if err := tx.Commit(); err != nil {
		return Definition{}, err
	}
	items, err := r.List(ctx)
	if err != nil {
		return Definition{}, err
	}
	for _, item := range items {
		if item.ID == requested.ID {
			return item, nil
		}
	}
	return Definition{}, ErrUnknownFacet{ID: requested.ID}
}

func (r *SQLiteRepository) examples(ctx context.Context, facetID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT body FROM facet_examples WHERE scope=? AND facet_id=? ORDER BY id LIMIT 3`, policy.ScopeFromContext(ctx), facetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var body string
		if err := rows.Scan(&body); err != nil {
			return nil, err
		}
		out = append(out, body)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) Validate(ctx context.Context, id string) error {
	if id == UnclassifiedFacet {
		return nil
	}
	var found string
	err := r.db.QueryRowContext(ctx, `SELECT id FROM facet_definitions WHERE scope=? AND id=?`, policy.ScopeFromContext(ctx), id).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUnknownFacet{ID: id}
	}
	return err
}

func (r *SQLiteRepository) CompactionEligible(ctx context.Context, entryID string) (bool, error) {
	var eligible, pinned bool
	err := r.db.QueryRowContext(ctx, `SELECT p.compaction_eligible, EXISTS(SELECT 1 FROM pins WHERE entry_id=? AND (review_at IS NULL OR review_at>?)) FROM facet_assignments a JOIN facet_policies p ON p.scope=? AND p.facet_id=a.facet_id JOIN entries e ON e.id=a.entry_id WHERE e.scope=? AND a.entry_id=? ORDER BY a.assigned_at DESC,a.id DESC LIMIT 1`, entryID, time.Now().UTC().Format(time.RFC3339Nano), policy.ScopeFromContext(ctx), policy.ScopeFromContext(ctx), entryID).Scan(&eligible, &pinned)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return eligible && !pinned, err
}

func (r *SQLiteRepository) SetPin(ctx context.Context, entryID string, pinned bool) error {
	if !pinned {
		_, err := r.db.ExecContext(ctx, `DELETE FROM pins WHERE entry_id=? AND EXISTS(SELECT 1 FROM entries WHERE id=? AND scope=?)`, entryID, entryID, policy.ScopeFromContext(ctx))
		return err
	}
	now := time.Now().UTC()
	nowText := now.Format(time.RFC3339Nano)
	// Lapsed pins fail safe: the journal entry remains intact and searchable,
	// but an expired review no longer consumes the standing-rule attention budget.
	if err := r.lapseExpiredPins(ctx, nowText); err != nil {
		return err
	}
	var budget int
	if err := r.db.QueryRowContext(ctx, `SELECT p.resident_budget FROM entries e JOIN facet_assignments a ON a.entry_id=e.id JOIN facet_policies p ON p.scope=? AND p.facet_id=a.facet_id WHERE e.scope=? AND e.id=? ORDER BY a.assigned_at DESC,a.id DESC LIMIT 1`, policy.ScopeFromContext(ctx), policy.ScopeFromContext(ctx), entryID).Scan(&budget); err != nil {
		return err
	}
	if budget > 0 {
		var count int
		if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pins p JOIN entries e ON e.id=p.entry_id JOIN facet_assignments a ON a.entry_id=e.id JOIN facet_policies fp ON fp.scope=? AND fp.facet_id=a.facet_id WHERE e.scope=? AND fp.retention_policy='pinned-or-review' AND a.id=(SELECT newer.id FROM facet_assignments newer WHERE newer.entry_id=a.entry_id ORDER BY newer.assigned_at DESC,newer.id DESC LIMIT 1) AND p.entry_id<>?`, policy.ScopeFromContext(ctx), policy.ScopeFromContext(ctx), entryID).Scan(&count); err != nil {
			return err
		}
		if count >= budget {
			proposalID := "pin-budget-" + string(policy.ScopeFromContext(ctx)) + "-" + entryID
			var weakest string
			_ = r.db.QueryRowContext(ctx, `SELECT p.entry_id FROM pins p JOIN entries e ON e.id=p.entry_id JOIN facet_assignments a ON a.entry_id=e.id JOIN facet_policies fp ON fp.scope=? AND fp.facet_id=a.facet_id WHERE e.scope=? AND fp.retention_policy='pinned-or-review' ORDER BY p.pinned_at ASC,p.entry_id ASC LIMIT 1`, policy.ScopeFromContext(ctx), policy.ScopeFromContext(ctx)).Scan(&weakest)
			ids, _ := json.Marshal([]string{entryID, weakest})
			if _, err := r.db.ExecContext(ctx, `INSERT INTO merge_proposals(id,scope,rationale,entry_ids_json) VALUES(?,?,?,?) ON CONFLICT(id) DO UPDATE SET rationale=excluded.rationale,entry_ids_json=excluded.entry_ids_json,resolved_at=NULL`, proposalID, policy.ScopeFromContext(ctx), "Pin budget is full; review the requested pin against the weakest current pin.", string(ids)); err != nil {
				return err
			}
			return ErrPinBudgetExceeded{ProposalID: proposalID}
		}
	}
	reviewAt := now.Add(pinReviewInterval).Format(time.RFC3339Nano)
	_, err := r.db.ExecContext(ctx, `INSERT INTO pins(entry_id,review_at,pinned_at) VALUES(?,?,?) ON CONFLICT(entry_id) DO UPDATE SET review_at=excluded.review_at`, entryID, reviewAt, nowText)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO pin_reviews(id,entry_id,due_at) VALUES(?,?,?)`, uuid.NewString(), entryID, reviewAt)
	return err
}

func (r *SQLiteRepository) lapseExpiredPins(ctx context.Context, nowText string) error {
	rows, err := r.db.QueryContext(ctx, `SELECT p.entry_id,p.review_at FROM pins p JOIN entries e ON e.id=p.entry_id WHERE e.scope=? AND p.review_at IS NOT NULL AND p.review_at<=?`, policy.ScopeFromContext(ctx), nowText)
	if err != nil {
		return err
	}
	defer rows.Close()
	var expired []struct{ entryID, dueAt string }
	for rows.Next() {
		var item struct{ entryID, dueAt string }
		if err := rows.Scan(&item.entryID, &item.dueAt); err != nil {
			return err
		}
		expired = append(expired, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, item := range expired {
		if _, err := r.db.ExecContext(ctx, `INSERT INTO pin_reviews(id,entry_id,due_at,resolved_at) VALUES(?,?,?,?)`, uuid.NewString(), item.entryID, item.dueAt, nowText); err != nil {
			return err
		}
	}
	_, err = r.db.ExecContext(ctx, `DELETE FROM pins WHERE review_at IS NOT NULL AND review_at<=? AND EXISTS(SELECT 1 FROM entries e WHERE e.id=pins.entry_id AND e.scope=?)`, nowText, policy.ScopeFromContext(ctx))
	return err
}

func (r *SQLiteRepository) Pinned(ctx context.Context, entryID string) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pins p JOIN entries e ON e.id=p.entry_id WHERE p.entry_id=? AND e.scope=? AND (p.review_at IS NULL OR p.review_at>?))`, entryID, policy.ScopeFromContext(ctx), time.Now().UTC().Format(time.RFC3339Nano)).Scan(&ok)
	return ok, err
}

func (r *SQLiteRepository) Assign(ctx context.Context, a Assignment) (Assignment, error) {
	if err := r.Validate(ctx, a.FacetID); err != nil {
		return Assignment{}, err
	}
	if strings.TrimSpace(a.ActorID) == "" {
		a.ActorID = "system:facets"
	}
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	if a.AssignedAt.IsZero() {
		a.AssignedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO facet_assignments(id,entry_id,facet_id,assigned_at,actor_id) VALUES(?,?,?,?,?)`, a.ID, a.EntryID, a.FacetID, a.AssignedAt.Format(time.RFC3339Nano), a.ActorID)
	return a, err
}

func (r *SQLiteRepository) Assignments(ctx context.Context, entryID string) ([]Assignment, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT a.id,a.entry_id,a.facet_id,a.assigned_at,a.actor_id FROM facet_assignments a JOIN entries e ON e.id=a.entry_id WHERE a.entry_id=? AND e.scope=? ORDER BY a.assigned_at,a.id`, entryID, policy.ScopeFromContext(ctx))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Assignment
	for rows.Next() {
		var a Assignment
		var raw string
		if err = rows.Scan(&a.ID, &a.EntryID, &a.FacetID, &raw, &a.ActorID); err != nil {
			return nil, err
		}
		a.AssignedAt, _ = time.Parse(time.RFC3339Nano, raw)
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) MarkSuperseded(ctx context.Context, entryID, replacementID string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO marks(id,entry_id,kind,replacement_entry_id,created_at) SELECT ?,?,?,?,? WHERE EXISTS(SELECT 1 FROM entries WHERE id=? AND scope=?)`, uuid.NewString(), entryID, "superseded", replacementID, time.Now().UTC().Format(time.RFC3339Nano), entryID, policy.ScopeFromContext(ctx))
	return err
}

func (r *SQLiteRepository) ResolveThread(ctx context.Context, entryID string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO marks(id,entry_id,kind,created_at) SELECT ?,?,?,? WHERE EXISTS(SELECT 1 FROM entries WHERE id=? AND scope=?)`, uuid.NewString(), entryID, "resolved", time.Now().UTC().Format(time.RFC3339Nano), entryID, policy.ScopeFromContext(ctx))
	return err
}

func (r *SQLiteRepository) ListPinProposals(ctx context.Context) ([]PinProposal, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	rows, err := r.db.QueryContext(ctx, `SELECT e.id,e.body FROM entries e JOIN facet_assignments a ON a.entry_id=e.id JOIN facet_policies p ON p.scope=? AND p.facet_id=a.facet_id WHERE e.scope=? AND a.id=(SELECT newer.id FROM facet_assignments newer WHERE newer.entry_id=e.id ORDER BY newer.assigned_at DESC,newer.id DESC LIMIT 1) AND p.retention_policy='pinned-or-review' AND NOT EXISTS(SELECT 1 FROM pins WHERE entry_id=e.id AND (review_at IS NULL OR review_at>?)) ORDER BY e.body,e.id`, policy.ScopeFromContext(ctx), policy.ScopeFromContext(ctx), now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	groups := map[string][]string{}
	for rows.Next() {
		var id, body string
		if err := rows.Scan(&id, &body); err != nil {
			return nil, err
		}
		key := strings.Join(strings.Fields(body), " ")
		groups[key] = append(groups[key], id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var out []PinProposal
	for body, ids := range groups {
		if len(ids) < 2 {
			continue
		}
		hash := sha256.Sum256([]byte(body))
		id := "pin-" + string(policy.ScopeFromContext(ctx)) + "-" + hex.EncodeToString(hash[:8])
		payload, _ := json.Marshal(ids)
		_, err := r.db.ExecContext(ctx, `INSERT INTO merge_proposals(id,scope,rationale,entry_ids_json) VALUES(?,?,?,?) ON CONFLICT(id) DO UPDATE SET entry_ids_json=excluded.entry_ids_json`, id, policy.ScopeFromContext(ctx), "Redundant standing-rule memories share the same normalized body.", string(payload))
		if err != nil {
			return nil, err
		}
		out = append(out, PinProposal{ID: id, EntryIDs: ids, Rationale: "Redundant standing-rule memories share the same normalized body."})
	}
	budgetRows, err := r.db.QueryContext(ctx, `SELECT id,entry_ids_json,rationale FROM merge_proposals WHERE scope=? AND resolved_at IS NULL AND id LIKE 'pin-budget-%' ORDER BY id`, policy.ScopeFromContext(ctx))
	if err != nil {
		return nil, err
	}
	defer budgetRows.Close()
	for budgetRows.Next() {
		var id, raw, rationale string
		if err := budgetRows.Scan(&id, &raw, &rationale); err != nil {
			return nil, err
		}
		var ids []string
		if err := json.Unmarshal([]byte(raw), &ids); err != nil {
			return nil, err
		}
		out = append(out, PinProposal{ID: id, EntryIDs: ids, Rationale: rationale})
	}
	if err := budgetRows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *SQLiteRepository) ResolvePinProposal(ctx context.Context, id string, accept bool) error {
	var raw string
	if err := r.db.QueryRowContext(ctx, `SELECT entry_ids_json FROM merge_proposals WHERE id=? AND scope=?`, id, policy.ScopeFromContext(ctx)).Scan(&raw); err != nil {
		return err
	}
	var ids []string
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return err
	}
	if accept && len(ids) > 0 {
		// Tradeoff proposals put the requested entry first and the entries
		// being displaced after it. Release the displaced pins before the
		// requested pin, otherwise a full budget rejects its own proposal.
		for _, entryID := range ids[1:] {
			if err := r.SetPin(ctx, entryID, false); err != nil {
				return err
			}
		}
		if err := r.SetPin(ctx, ids[0], true); err != nil {
			return err
		}
	}
	_, err := r.db.ExecContext(ctx, `UPDATE merge_proposals SET resolved_at=? WHERE id=? AND scope=?`, time.Now().UTC().Format(time.RFC3339Nano), id, policy.ScopeFromContext(ctx))
	return err
}

func (r *SQLiteRepository) RecordRecall(ctx context.Context, entryIDs []string) error {
	if len(entryIDs) == 0 {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, entryID := range entryIDs {
		if strings.TrimSpace(entryID) == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO facet_recall_stats(entry_id,recall_count,last_recalled_at) SELECT ?,?,? WHERE EXISTS(SELECT 1 FROM entries WHERE id=? AND scope=?) ON CONFLICT(entry_id) DO UPDATE SET recall_count=facet_recall_stats.recall_count+1,last_recalled_at=excluded.last_recalled_at`, entryID, 1, now, entryID, policy.ScopeFromContext(ctx)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *SQLiteRepository) ListPinCandidates(ctx context.Context, limit int) ([]PinCandidate, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `
WITH latest AS (
  SELECT entry_id,facet_id,ROW_NUMBER() OVER (PARTITION BY entry_id ORDER BY assigned_at DESC,id DESC) rn
  FROM facet_assignments
)
SELECT e.id,e.body,e.created_at,COALESCE(rs.recall_count,0),rs.last_recalled_at
FROM entries e JOIN latest l ON l.entry_id=e.id AND l.rn=1
JOIN facet_policies p ON p.scope=? AND p.facet_id=l.facet_id
LEFT JOIN facet_recall_stats rs ON rs.entry_id=e.id
WHERE e.scope=? AND p.retention_policy='pinned-or-review'
  AND NOT EXISTS(SELECT 1 FROM pins WHERE entry_id=e.id AND (review_at IS NULL OR review_at>?))
ORDER BY COALESCE(rs.recall_count,0) DESC, COALESCE(rs.last_recalled_at,'' ) DESC, e.created_at DESC, e.id
LIMIT ?`, policy.ScopeFromContext(ctx), policy.ScopeFromContext(ctx), time.Now().UTC().Format(time.RFC3339Nano), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PinCandidate
	for rows.Next() {
		var candidate PinCandidate
		var created, recalled sql.NullString
		if err := rows.Scan(&candidate.EntryID, &candidate.Body, &created, &candidate.RecallCount, &recalled); err != nil {
			return nil, err
		}
		candidate.CreatedAt, _ = time.Parse(time.RFC3339Nano, created.String)
		if recalled.Valid {
			parsed, err := time.Parse(time.RFC3339Nano, recalled.String)
			if err != nil {
				return nil, err
			}
			candidate.LastRecalledAt = &parsed
		}
		out = append(out, candidate)
	}
	return out, rows.Err()
}
