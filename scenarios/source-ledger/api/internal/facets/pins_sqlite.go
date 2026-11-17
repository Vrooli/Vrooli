package facets

// Pins keep an entry out of compaction until its review date; proposals and
// candidates come from retention policy and recall counts.

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"source-ledger/internal/policy"

	"github.com/google/uuid"
)

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
	budget, err := r.budgetPinProposals(ctx)
	if err != nil {
		return nil, err
	}
	return append(out, budget...), nil
}

// budgetPinProposals lists the open pin-budget merge proposals.
func (r *SQLiteRepository) budgetPinProposals(ctx context.Context) ([]PinProposal, error) {
	var out []PinProposal
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
