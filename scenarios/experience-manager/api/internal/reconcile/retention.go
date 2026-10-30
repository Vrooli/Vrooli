package reconcile

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Evidence retention.
//
// Every reconcile pass appends one row per claim × state × viewport, and a row
// can carry a full accessibility-tree JSON (hundreds of KB). Readers only ask
// for the newest rows of a claim (ListEvidence orders by checked_at DESC with a
// limit), so superseded history is never served. Retention removes a row only
// when it is older than MinAge AND at least KeepPerClaim newer rows exist for
// the same full claim key (scenario, document kind, page, component, state,
// claim, viewport). The newest verdicts of every claim are therefore kept
// forever, however old they are.

// EvidenceRetentionPolicy bounds superseded evidence history.
type EvidenceRetentionPolicy struct {
	KeepPerClaim int
	MinAge       time.Duration
	// Batch bounds one delete transaction so the single API connection is
	// released between batches. Rows can carry hundreds of KB of overflow
	// pages, so batches stay small to keep each transaction well under the
	// API's 5s health-check database ping.
	Batch int
	// Pause yields the connection between batches so request traffic and
	// health checks are served during a large backlog.
	Pause time.Duration
}

// DefaultEvidenceRetentionPolicy keeps the newest three verdicts per claim key
// and every row younger than fourteen days.
func DefaultEvidenceRetentionPolicy() EvidenceRetentionPolicy {
	return EvidenceRetentionPolicy{KeepPerClaim: 3, MinAge: 14 * 24 * time.Hour, Batch: 20, Pause: 250 * time.Millisecond}
}

// EvidenceRetentionResult summarizes one retention cycle.
type EvidenceRetentionResult struct {
	CycleAt     time.Time               `json:"cycle_at"`
	Policy      EvidenceRetentionPolicy `json:"policy"`
	Cutoff      time.Time               `json:"cutoff"`
	Candidates  int                     `json:"candidates"`
	DeletedRows int64                   `json:"deleted_rows"`
	// Kept counts candidates the write-side re-check refused (a newer row
	// disappeared or the candidate was not actually superseded).
	Kept int `json:"kept"`
	// ReleasedBytes are free pages returned to the filesystem after batches;
	// always 0 until the file is in auto_vacuum=INCREMENTAL mode.
	ReleasedBytes int64 `json:"released_bytes"`
}

const supersededCandidatesSQL = `
SELECT id FROM (
  SELECT e.id AS id, e.checked_at AS checked_at,
    ROW_NUMBER() OVER (
      PARTITION BY e.scenario, e.document_kind, e.page_id, e.component_id, e.state_id, e.claim_id,
        COALESCE(v.viewport_id, ''), COALESCE(v.viewport_width, 0), COALESCE(v.viewport_height, 0)
      ORDER BY e.checked_at DESC, e.id DESC) AS rn
  FROM reconcile_evidence e
  LEFT JOIN reconcile_evidence_viewports v ON v.evidence_id = e.id)
WHERE rn > ? AND checked_at < ?`

// supersededDeleteSQL re-proves supersession at write time, so a stale or
// inconsistent candidate read can never delete a claim's newest verdicts.
const supersededDeleteSQL = `
DELETE FROM reconcile_evidence
WHERE id = ? AND checked_at < ? AND (
  SELECT COUNT(*) FROM (
    SELECT 1 FROM reconcile_evidence n
    LEFT JOIN reconcile_evidence_viewports nv ON nv.evidence_id = n.id
    WHERE n.scenario = reconcile_evidence.scenario
      AND n.page_id = reconcile_evidence.page_id
      AND n.state_id = reconcile_evidence.state_id
      AND n.claim_id = reconcile_evidence.claim_id
      AND n.document_kind = reconcile_evidence.document_kind
      AND n.component_id = reconcile_evidence.component_id
      AND COALESCE(nv.viewport_id, '') = COALESCE((SELECT viewport_id FROM reconcile_evidence_viewports WHERE evidence_id = reconcile_evidence.id), '')
      AND COALESCE(nv.viewport_width, 0) = COALESCE((SELECT viewport_width FROM reconcile_evidence_viewports WHERE evidence_id = reconcile_evidence.id), 0)
      AND COALESCE(nv.viewport_height, 0) = COALESCE((SELECT viewport_height FROM reconcile_evidence_viewports WHERE evidence_id = reconcile_evidence.id), 0)
      AND (n.checked_at > reconcile_evidence.checked_at
        OR (n.checked_at = reconcile_evidence.checked_at AND n.id > reconcile_evidence.id))
    LIMIT ?)
) >= ?`

// PruneSupersededEvidence finds candidates through reader (which may be a
// separate read-only connection, so a long scan never holds the API's single
// writer connection) and deletes them through writer in bounded batches,
// re-checking supersession for every row.
func PruneSupersededEvidence(ctx context.Context, reader, writer SQLExecutor, policy EvidenceRetentionPolicy, now time.Time) (EvidenceRetentionResult, error) {
	if policy.KeepPerClaim < 1 {
		return EvidenceRetentionResult{}, fmt.Errorf("evidence retention: KeepPerClaim must be at least 1")
	}
	if policy.Batch <= 0 {
		policy.Batch = 20
	}
	cutoff := now.UTC().Add(-policy.MinAge)
	cutoffText := cutoff.Format(evidenceTimeFormat)
	result := EvidenceRetentionResult{CycleAt: now.UTC(), Policy: policy, Cutoff: cutoff}

	ids, err := supersededCandidates(ctx, reader, policy.KeepPerClaim, cutoffText)
	if err != nil {
		return result, err
	}
	result.Candidates = len(ids)

	for start := 0; start < len(ids); start += policy.Batch {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		end := start + policy.Batch
		if end > len(ids) {
			end = len(ids)
		}
		deleted, err := deleteSupersededBatch(ctx, writer, ids[start:end], cutoffText, policy.KeepPerClaim)
		if err != nil {
			return result, err
		}
		result.DeletedRows += deleted
		result.Kept += (end - start) - int(deleted)
		// Once the file is incremental, return this batch's pages at once so
		// the freelist never accumulates a backlog.
		if deleted > 0 {
			freed, err := ReleaseFreePages(ctx, writer, ReleasePageBatch)
			if err != nil {
				return result, err
			}
			result.ReleasedBytes += freed
		}
		if policy.Pause > 0 && end < len(ids) {
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-time.After(policy.Pause):
			}
		}
	}
	return result, nil
}

func supersededCandidates(ctx context.Context, reader SQLExecutor, keep int, cutoff string) ([]string, error) {
	rows, err := reader.QueryContext(ctx, supersededCandidatesSQL, keep, cutoff)
	if err != nil {
		return nil, fmt.Errorf("scan superseded evidence: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan superseded evidence id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate superseded evidence: %w", err)
	}
	return ids, nil
}

func deleteSupersededBatch(ctx context.Context, writer SQLExecutor, ids []string, cutoff string, keep int) (int64, error) {
	tx, err := writer.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin evidence retention batch: %w", err)
	}
	var deleted int64
	var removed []string
	for _, id := range ids {
		res, err := tx.ExecContext(ctx, supersededDeleteSQL, id, cutoff, keep, keep)
		if err != nil {
			_ = tx.Rollback()
			return 0, fmt.Errorf("delete superseded evidence %s: %w", id, err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			_ = tx.Rollback()
			return 0, fmt.Errorf("delete superseded evidence rows affected: %w", err)
		}
		if n > 0 {
			deleted += n
			removed = append(removed, id)
		}
	}
	// The viewport row is a 1:1 child; remove it explicitly rather than rely on
	// the connection's foreign_keys setting.
	if len(removed) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(removed)), ",")
		args := make([]any, len(removed))
		for i, id := range removed {
			args[i] = id
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM reconcile_evidence_viewports WHERE evidence_id IN ("+placeholders+")", args...); err != nil {
			_ = tx.Rollback()
			return 0, fmt.Errorf("delete superseded evidence viewports: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit evidence retention batch: %w", err)
	}
	return deleted, nil
}

// ReleasePageBatch bounds one incremental_vacuum transaction (4 MiB at the
// default page size).
const ReleasePageBatch = 1024

// ReleaseFreePages hands pages freed by retention back to the filesystem in
// bounded incremental_vacuum(batchPages) transactions, so the single API
// connection is released between batches. On a file still in
// auto_vacuum=NONE mode it returns at once having released nothing.
func ReleaseFreePages(ctx context.Context, writer SQLExecutor, batchPages int64) (int64, error) {
	if batchPages <= 0 {
		batchPages = ReleasePageBatch
	}
	var mode, pageSize int64
	if err := writer.QueryRowContext(ctx, "PRAGMA auto_vacuum").Scan(&mode); err != nil {
		return 0, fmt.Errorf("read auto_vacuum: %w", err)
	}
	if mode != 2 {
		return 0, nil
	}
	if err := writer.QueryRowContext(ctx, "PRAGMA page_size").Scan(&pageSize); err != nil {
		return 0, err
	}
	var released int64
	for {
		if err := ctx.Err(); err != nil {
			return released, err
		}
		var before, after int64
		if err := writer.QueryRowContext(ctx, "PRAGMA freelist_count").Scan(&before); err != nil {
			return released, err
		}
		if before == 0 {
			return released, nil
		}
		if err := incrementalVacuum(ctx, writer, batchPages); err != nil {
			return released, err
		}
		if err := writer.QueryRowContext(ctx, "PRAGMA freelist_count").Scan(&after); err != nil {
			return released, err
		}
		if after >= before {
			return released, nil
		}
		released += (before - after) * pageSize
	}
}

func incrementalVacuum(ctx context.Context, writer SQLExecutor, pages int64) error {
	rows, err := writer.QueryContext(ctx, fmt.Sprintf("PRAGMA incremental_vacuum(%d)", pages))
	if err != nil {
		return fmt.Errorf("incremental vacuum: %w", err)
	}
	defer rows.Close()
	for rows.Next() { //nolint:revive // draining runs the pragma to completion
	}
	return rows.Err()
}
