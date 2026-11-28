package capacity

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
)

// Only operation claims whose owner id matches processBoundOwnerPattern are
// released on proof of owner death. Such a claim is held by exactly one local
// process for the duration of one command: its owner id embeds the holder's
// PID (and start token), the holder is always on this host, and nothing else
// can renew it. Other operation claims may name PIDs from another namespace
// (sandboxes, containers) where a /proc probe proves nothing, so they keep
// waiting for their TTL.
var processBoundOwnerPattern = regexp.MustCompile(`^cli-install:pid-(\d+)(?:-(\d+))?:`)

// ProcessBoundRunID is the run component of a process-bound owner id. The
// start token makes the identity exact: a reused PID has a different one.
func ProcessBoundRunID(pid int) string {
	if start, err := processStartIdentity(pid); err == nil && start != "" {
		return fmt.Sprintf("pid-%d-%s", pid, start)
	}
	return fmt.Sprintf("pid-%d", pid)
}

// processBoundOwnerGone reports whether the process named by a process-bound
// owner id has definitely exited. Owner ids of any other shape are never gone.
func processBoundOwnerGone(ownerID string) bool {
	match := processBoundOwnerPattern.FindStringSubmatch(ownerID)
	if match == nil {
		return false
	}
	pid, err := strconv.Atoi(match[1])
	if err != nil {
		return false
	}
	return processGone(pid, match[2])
}

// expireDeadOwnerClaimsTx moves active process-bound operation claims whose
// owner process has exited to expired. A killed or timed-out command never
// reaches its release, and until this sweep its claim kept charging the
// ledger for the whole 15 minute TTL. The caller holds the write transaction.
func (s *SQLiteStore) expireDeadOwnerClaimsTx(ctx context.Context, tx *sql.Tx, expired *[]CapacityClaim) error {
	rows, err := tx.QueryContext(ctx, claimSelectSQL+`
WHERE status IN (?, ?, ?) AND owner_kind = ? AND owner_id LIKE ?
ORDER BY created_at ASC`, StatusReserved, StatusGranted, StatusDegraded, OwnerKindOp, "cli-install:pid-%")
	if err != nil {
		return fmt.Errorf("list process-bound capacity claims: %w", err)
	}
	candidates, scanErr := scanClaims(rows)
	if closeErr := rows.Close(); closeErr != nil && scanErr == nil {
		scanErr = closeErr
	}
	if scanErr != nil {
		return scanErr
	}
	for _, claim := range candidates {
		if !processBoundOwnerGone(claim.OwnerID) {
			continue
		}
		now := s.now()
		result, err := tx.ExecContext(ctx, `
UPDATE capacity_claims
SET status = ?, generation = generation + 1, updated_at = ?
WHERE claim_id = ? AND generation = ? AND status IN (?, ?, ?)`, StatusExpired, formatTime(now), claim.ClaimID, claim.Generation, StatusReserved, StatusGranted, StatusDegraded)
		if err != nil {
			return fmt.Errorf("expire dead-owner capacity claim %s: %w", claim.ClaimID, err)
		}
		if n, raErr := result.RowsAffected(); raErr != nil || n == 0 {
			continue
		}
		claim.Status = StatusExpired
		claim.Generation++
		claim.UpdatedAt = now
		*expired = append(*expired, claim)
	}
	return nil
}
