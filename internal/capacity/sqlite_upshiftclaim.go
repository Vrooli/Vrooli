package capacity

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const (
	// gcTerminalChunkRows bounds one GC transaction, and with it how long the
	// SQLite writer lock is held (~1 ms per deleted row).
	gcTerminalChunkRows = 200
	// gcTerminalBudget bounds one GCTerminalClaims call; the backlog drains
	// across maintenance passes instead of in one long stall.
	gcTerminalBudget = 5 * time.Second
)

// UpshiftClaim steps a claim UP to a larger profile rung (the symmetric
// counterpart of DegradeClaim): it raises amount_bytes and restores status to
// granted (the claim is no longer running below its preferred size), bumping the
// generation under the optimistic-concurrency guard. The label is the adopter's
// concern; the ledger records the resulting amount.
func (s *SQLiteStore) UpshiftClaim(ctx context.Context, claimID string, generation int64, step string, amountBytes int64) (CapacityClaim, error) {
	if strings.TrimSpace(claimID) == "" {
		return CapacityClaim{}, fmt.Errorf("%w: claim_id is required", ErrInvalidClaim)
	}
	if amountBytes < 0 {
		return CapacityClaim{}, fmt.Errorf("%w: upshift amount must be non-negative", ErrInvalidClaim)
	}
	now := s.now()
	var out CapacityClaim
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		unresolved, fenceErr := hasUnresolvedResidencyTransitionTx(ctx, tx, claimID, s.now())
		if fenceErr != nil {
			return fenceErr
		}
		if unresolved {
			return ErrResidencyTransitionOpen
		}
		result, execErr := tx.ExecContext(ctx, `
UPDATE capacity_claims
SET status = ?, amount_bytes = ?, generation = generation + 1, updated_at = ?
WHERE claim_id = ? AND generation = ? AND status IN (?, ?, ?)`,
			StatusGranted, amountBytes, formatTime(now),
			claimID, generation, StatusReserved, StatusGranted, StatusDegraded)
		if execErr != nil {
			return fmt.Errorf("upshift capacity claim: %w", execErr)
		}
		_ = step // the label is the adopter's concern; the ledger records the resulting amount.
		return finishMutation(ctx, tx, result, claimID, &out)
	})
	if err != nil {
		return CapacityClaim{}, err
	}
	return out, nil
}

// ReleaseClaim terminates a claim cleanly (by claim ID, no generation guard —
// the owner is done with it).
func (s *SQLiteStore) ReleaseClaim(ctx context.Context, claimID string) (CapacityClaim, error) {
	return s.terminate(ctx, claimID, StatusReleased)
}

// PreemptClaim records preemption intent only for the planned idle generation.
// It does not stop the owner or establish that physical memory was released.
func (s *SQLiteStore) PreemptClaim(ctx context.Context, claimID string, generation int64, reason string) (CapacityClaim, error) {
	_ = reason
	var out CapacityClaim
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		unresolved, fenceErr := hasUnresolvedResidencyTransitionTx(ctx, tx, claimID, s.now())
		if fenceErr != nil {
			return fenceErr
		}
		if unresolved {
			return ErrResidencyTransitionOpen
		}
		result, err := tx.ExecContext(ctx, `UPDATE capacity_claims SET status = ?, generation = generation + 1, updated_at = ?
WHERE claim_id = ? AND generation = ? AND status IN (?, ?, ?) AND activity_state = ? AND protected = 0`,
			StatusPreempted, formatTime(s.now()), claimID, generation, StatusReserved, StatusGranted, StatusDegraded, ActivityIdle)
		if err != nil {
			return fmt.Errorf("preempt idle capacity claim: %w", err)
		}
		return finishMutation(ctx, tx, result, claimID, &out)
	})
	return out, err
}

func (s *SQLiteStore) terminate(ctx context.Context, claimID, status string) (CapacityClaim, error) {
	if strings.TrimSpace(claimID) == "" {
		return CapacityClaim{}, fmt.Errorf("%w: claim_id is required", ErrInvalidClaim)
	}
	now := s.now()
	var out CapacityClaim
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		unresolved, fenceErr := hasUnresolvedResidencyTransitionTx(ctx, tx, claimID, s.now())
		if fenceErr != nil {
			return fenceErr
		}
		if unresolved {
			return ErrResidencyTransitionOpen
		}
		result, execErr := tx.ExecContext(ctx, `
UPDATE capacity_claims
SET status = ?, generation = generation + 1, updated_at = ?
WHERE claim_id = ? AND status IN (?, ?, ?)`,
			status, formatTime(now), claimID, StatusReserved, StatusGranted, StatusDegraded)
		if execErr != nil {
			return fmt.Errorf("terminate capacity claim: %w", execErr)
		}
		affected, raErr := result.RowsAffected()
		if raErr != nil {
			return fmt.Errorf("inspect capacity claim termination: %w", raErr)
		}
		if affected == 0 {
			// Already terminal or absent — surface NotFound vs idempotent return.
			existing, getErr := getClaimTx(ctx, tx, claimID)
			if getErr != nil {
				return getErr
			}
			out = existing
			return nil
		}
		out, execErr = getClaimTx(ctx, tx, claimID)
		return execErr
	})
	if err != nil {
		return CapacityClaim{}, err
	}
	return out, nil
}

// ExpireStaleClaims sweeps active claims whose heartbeat deadline has elapsed to
// status=expired, returning the expired set. Process-bound operation claims
// (see dead_owner_claims.go) whose owner process has exited are expired
// immediately rather than at their deadline.
func (s *SQLiteStore) ExpireStaleClaims(ctx context.Context, at time.Time, proofs AbsenceProofs) ([]CapacityClaim, error) {
	var expired []CapacityClaim
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		if err := acquireClaimWriteAuthority(ctx, tx); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, claimSelectSQL+`
WHERE status IN (?, ?, ?) AND heartbeat_deadline_at IS NOT NULL AND heartbeat_deadline_at <= ?
ORDER BY created_at ASC`, StatusReserved, StatusGranted, StatusDegraded, formatTime(at.UTC()))
		if err != nil {
			return fmt.Errorf("list stale capacity claims: %w", err)
		}
		candidates, scanErr := scanClaims(rows)
		if closeErr := rows.Close(); closeErr != nil && scanErr == nil {
			scanErr = closeErr
		}
		if scanErr != nil {
			return scanErr
		}
		expired = nil
		for _, claim := range candidates {
			// The scan or earlier rows can age an absence witness. Validate at this
			// row's mutation boundary while retaining exclusive SQLite write authority.
			now := s.now()
			unresolved, fenceErr := hasUnresolvedResidencyTransitionTx(ctx, tx, claim.ClaimID, now)
			if fenceErr != nil {
				return fenceErr
			}
			if unresolved {
				continue
			}
			if claim.ResourceKind == ResourceKindVRAM && claim.OwnerKind != OwnerKindOp && !proofs[claim.ClaimID].validFor(claim, now) {
				continue
			}
			if _, err := tx.ExecContext(ctx, `
UPDATE capacity_claims
SET status = ?, generation = generation + 1, updated_at = ?
WHERE claim_id = ? AND status IN (?, ?, ?)`, StatusExpired, formatTime(now), claim.ClaimID, StatusReserved, StatusGranted, StatusDegraded); err != nil {
				return fmt.Errorf("expire capacity claim %s: %w", claim.ClaimID, err)
			}
			claim.Status = StatusExpired
			claim.Generation++
			claim.UpdatedAt = now
			expired = append(expired, claim)
		}
		if err := s.expireDeadOwnerClaimsTx(ctx, tx, &expired); err != nil {
			return err
		}
		// Disk leases lapse on the same heartbeat contract as claims, so the
		// periodic lease-expiry pass settles them too and an unvisited ledger
		// does not keep counting dead reservations.
		_, _, err = settleLapsedDiskLeasesTx(ctx, tx, s.now())
		return err
	})
	if err != nil {
		return nil, err
	}
	return expired, nil
}

// GCTerminalClaims prunes terminal (released/expired/preempted) claims whose
// updated_at is older than olderThan, returning how many rows were deleted and
// the sum of their last-recorded amount_bytes (informational). Active claims
// (reserved/granted/degraded) are NEVER pruned — only history is collected. GC
// is always safe regardless of enforce mode (it never frees live capacity, it
// only trims dead rows).
func (s *SQLiteStore) GCTerminalClaims(ctx context.Context, olderThan time.Time) (GCResult, error) {
	cutoff := formatTime(olderThan.UTC())
	var total GCResult
	budget := time.Now().Add(gcTerminalBudget)
	for {
		chunk, err := s.gcTerminalChunk(ctx, cutoff)
		total.Count += chunk.Count
		total.Bytes += chunk.Bytes
		if err != nil {
			return total, err
		}
		// A short chunk means nothing eligible is left. Otherwise keep going
		// until the budget is spent; the next pass resumes where this one stopped.
		if chunk.Count < gcTerminalChunkRows || time.Now().After(budget) || ctx.Err() != nil {
			return total, nil
		}
	}
}

// gcTerminalChunk deletes one bounded batch of prunable terminal claims in its
// own transaction.
//
// A single DELETE over every eligible row used to run here. With the ledger at
// tens of thousands of terminal rows it held the SQLite writer lock for ~45 s
// (every deleted claim makes SQLite scan the unindexed claim_id column of
// capacity_residency_transition_events to check its foreign key) and then
// failed on the first claim a residency transition still referenced, rolling
// the whole thing back. It never pruned anything, was retried on every
// maintenance pass, and each attempt stalled every other writer in the fleet.
// Claims with residency history are now skipped (that history is the audit
// proof for the claim and must outlive it), and rows go in batches small
// enough that the writer lock is released between them.
func (s *SQLiteStore) gcTerminalChunk(ctx context.Context, cutoff string) (GCResult, error) {
	var res GCResult
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		res = GCResult{}
		rows, err := tx.QueryContext(ctx, `
SELECT claim_id, amount_bytes
FROM capacity_claims
WHERE status NOT IN (?, ?, ?) AND updated_at < ?
  AND claim_id NOT IN (SELECT claim_id FROM capacity_residency_transitions)
  AND claim_id NOT IN (SELECT claim_id FROM capacity_residency_transition_events)
ORDER BY updated_at ASC
LIMIT ?`, StatusReserved, StatusGranted, StatusDegraded, cutoff, gcTerminalChunkRows)
		if err != nil {
			return fmt.Errorf("select terminal capacity claims: %w", err)
		}
		var ids []any
		for rows.Next() {
			var id string
			var bytes int64
			if scanErr := rows.Scan(&id, &bytes); scanErr != nil {
				_ = rows.Close()
				return fmt.Errorf("scan terminal capacity claim: %w", scanErr)
			}
			ids = append(ids, id)
			res.Bytes += bytes
		}
		if err := rows.Close(); err != nil {
			return err
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("select terminal capacity claims: %w", err)
		}
		if len(ids) == 0 {
			return nil
		}
		marks := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
		if _, execErr := tx.ExecContext(ctx, `DELETE FROM capacity_claims WHERE claim_id IN (`+marks+`)`, ids...); execErr != nil {
			return fmt.Errorf("prune terminal capacity claims: %w", execErr)
		}
		res.Count = len(ids)
		return nil
	})
	if err != nil {
		return GCResult{}, err
	}
	return res, nil
}

// RecordObserved persists a usage-sampling result on an active claim (§Phase 2):
// the latest observed bytes, the decaying peak, and the sample time. It is pure
// telemetry — it does NOT bump the generation (so it never invalidates a pending
// activity report or arbitration decision) and only writes active claims (a
// terminal claim is never resurrected). Observed usage NEVER feeds Decide
// (contract C1).
func (s *SQLiteStore) RecordObserved(ctx context.Context, claimID string, observed, peak int64, at time.Time, absence *AbsenceProof) (CapacityClaim, error) {
	if strings.TrimSpace(claimID) == "" {
		return CapacityClaim{}, fmt.Errorf("%w: claim_id is required", ErrInvalidClaim)
	}
	var out CapacityClaim
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		if observed == 0 {
			if err := acquireClaimWriteAuthority(ctx, tx); err != nil {
				return err
			}
			current, err := getClaimTx(ctx, tx, claimID)
			if err != nil {
				return err
			}
			if absence == nil || !absence.validFor(current, s.now()) {
				return nil
			}
		}
		result, execErr := tx.ExecContext(ctx, `
UPDATE capacity_claims
SET observed_bytes = ?, observed_peak_bytes = ?, observed_at = ?
WHERE claim_id = ? AND status IN (?, ?, ?)`,
			observed, peak, formatTime(at.UTC()), claimID, StatusReserved, StatusGranted, StatusDegraded)
		if execErr != nil {
			return fmt.Errorf("record observed usage: %w", execErr)
		}
		affected, raErr := result.RowsAffected()
		if raErr != nil {
			return fmt.Errorf("inspect observed-usage update: %w", raErr)
		}
		if affected == 0 {
			// Claim terminal or absent — telemetry on a dead row is a silent no-op.
			return nil
		}
		out, execErr = getClaimTx(ctx, tx, claimID)
		return execErr
	})
	if err != nil {
		return CapacityClaim{}, err
	}
	return out, nil
}

func (s *SQLiteStore) RecordFootprint(ctx context.Context, observation FootprintObservation) (Footprint, error) {
	if strings.TrimSpace(observation.Resource) == "" || strings.TrimSpace(observation.Rung) == "" {
		return Footprint{}, fmt.Errorf("%w: footprint resource and rung are required", ErrInvalidClaim)
	}
	if observation.Bytes < 0 {
		return Footprint{}, fmt.Errorf("%w: footprint bytes must be non-negative", ErrInvalidClaim)
	}
	if observation.Source != FootprintSourceManifestDefault && observation.Source != FootprintSourceMeasured {
		return Footprint{}, fmt.Errorf("%w: footprint source %q is invalid", ErrInvalidClaim, observation.Source)
	}
	at := observation.ObservedAt.UTC()
	if at.IsZero() {
		at = s.now()
	}
	var out Footprint
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
INSERT INTO capacity_footprints(resource, rung, tunables_key, gpu_index, peak_bytes, samples, first_seen_at, last_seen_at, source)
VALUES (?, ?, ?, ?, ?, CASE WHEN ? = 'measured' THEN 1 ELSE 0 END, ?, ?, ?)
ON CONFLICT(resource, rung, tunables_key, gpu_index) DO UPDATE SET
  peak_bytes = MAX(capacity_footprints.peak_bytes, excluded.peak_bytes),
  samples = capacity_footprints.samples + CASE WHEN excluded.source = 'measured' THEN 1 ELSE 0 END,
  last_seen_at = excluded.last_seen_at,
  source = CASE WHEN excluded.peak_bytes > capacity_footprints.peak_bytes THEN excluded.source
    WHEN excluded.peak_bytes = capacity_footprints.peak_bytes AND excluded.source = 'measured' THEN 'measured'
    ELSE capacity_footprints.source END`,
			observation.Resource, observation.Rung, observation.TunablesKey, observation.GPUIndex,
			observation.Bytes, observation.Source, formatTime(at), formatTime(at), observation.Source)
		if err != nil {
			return fmt.Errorf("record capacity footprint: %w", err)
		}
		return scanFootprint(tx.QueryRowContext(ctx, `
SELECT resource, rung, tunables_key, gpu_index, peak_bytes, samples, first_seen_at, last_seen_at, source
FROM capacity_footprints WHERE resource = ? AND rung = ? AND tunables_key = ? AND gpu_index = ?`,
			observation.Resource, observation.Rung, observation.TunablesKey, observation.GPUIndex), &out)
	})
	return out, err
}
