package scenarioruntime

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const demandLeaseExpiredReason = "demand lease deadline elapsed"

const demandStopReason = "no active demand lease"

// DemandIdleGrace is a sliding warm interval, not accumulated credit. Repeated
// use keeps a transient capability warm without adding hours of future idle
// runtime. Caller ownership still ends immediately on release.
const DemandIdleGrace = 10 * time.Minute

// RetainExplicitInstance makes explicit start precedence durable. Demand use
// never demotes an operator-owned instance, and an explicit start cannot claim
// success against a generation already being stopped.
func (s *SQLiteStore) RetainExplicitInstance(ctx context.Context, instanceID string, generation int64) (Instance, error) {
	var out Instance
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		current, err := getInstanceTx(ctx, tx, instanceID)
		if err != nil {
			return err
		}
		if current.Generation != generation {
			return ErrStaleGeneration
		}
		if current.Status != StatusRunning {
			return fmt.Errorf("%w: explicit retention requires a running instance", ErrDemandLeaseConflict)
		}
		if current.SupervisionPolicy == SupervisionPolicyDemand {
			if _, err := tx.ExecContext(ctx, `UPDATE runtime_instances SET supervision_policy = ?, updated_at = ? WHERE instance_id = ? AND generation = ? AND status = ?`, SupervisionPolicyManaged, formatTime(s.now()), instanceID, generation, StatusRunning); err != nil {
				return err
			}
			if err := s.auditDemandInstanceTx(ctx, tx, "retained_explicitly", current); err != nil {
				return err
			}
		}
		out, err = getInstanceTx(ctx, tx, instanceID)
		return err
	})
	return out, err
}

const demandLeaseSelectSQL = `
SELECT lease_id, scenario, variant, consumer_id, kind, request_id,
  created_at, last_renewed_at, expires_at, status, stop_reason, metadata_json
FROM runtime_demand_leases`

// NormalizeDemandLeaseTTL supplies the default hold duration for callers that
// omit a TTL. Callers that provide a positive duration must still pass the
// maximum-duration validation performed by acquire and renew.
func NormalizeDemandLeaseTTL(ttl time.Duration) time.Duration {
	if ttl <= 0 {
		return DefaultDemandLeaseTTL
	}
	return ttl
}

func validateDemandLeaseTTL(ttl time.Duration) error {
	if ttl <= 0 {
		return nil
	}
	if ttl > MaxDemandLeaseTTL {
		return fmt.Errorf("demand lease ttl must be at most %s", MaxDemandLeaseTTL)
	}
	return nil
}

func normalizeDemandLease(lease DemandLease) (DemandLease, error) {
	lease.LeaseID = strings.TrimSpace(lease.LeaseID)
	lease.Scenario = strings.TrimSpace(lease.Scenario)
	lease.Variant = InstanceKey{Scenario: lease.Scenario, Variant: lease.Variant}.Normalize().Variant
	lease.ConsumerID = strings.TrimSpace(lease.ConsumerID)
	lease.Kind = strings.TrimSpace(lease.Kind)
	lease.RequestID = strings.TrimSpace(lease.RequestID)
	if lease.LeaseID == "" || lease.Scenario == "" || lease.ConsumerID == "" || lease.Kind == "" {
		return DemandLease{}, fmt.Errorf("demand lease requires lease_id, scenario, consumer_id and kind")
	}
	switch lease.Kind {
	case DemandLeaseExplicit, DemandLeaseCore, DemandLeaseDependency, DemandLeaseProgram, DemandLeaseJob:
	default:
		return DemandLease{}, fmt.Errorf("demand lease kind %q is not supported", lease.Kind)
	}
	if len(lease.LeaseID) > 200 || len(lease.Scenario) > 200 || len(lease.Variant) > 200 || len(lease.ConsumerID) > 200 || len(lease.Kind) > 64 || len(lease.RequestID) > 200 {
		return DemandLease{}, fmt.Errorf("demand lease identity is too long")
	}
	if len(lease.MetadataJSON) > 16*1024 {
		return DemandLease{}, fmt.Errorf("demand lease metadata exceeds 16 KiB")
	}
	return lease, nil
}

// AcquireDemandLease creates or refreshes one stable consumer hold. Reusing a
// lease ID with a different identity is rejected rather than silently moving
// demand between scenarios.
func (s *SQLiteStore) AcquireDemandLease(ctx context.Context, lease DemandLease, ttl time.Duration) (DemandLease, error) {
	lease, err := normalizeDemandLease(lease)
	if err != nil {
		return DemandLease{}, err
	}
	if err := validateDemandLeaseTTL(ttl); err != nil {
		return DemandLease{}, err
	}
	now := s.now()
	deadline := now.Add(NormalizeDemandLeaseTTL(ttl))
	lease.Status = DemandLeaseActive
	lease.CreatedAt = now
	lease.LastRenewedAt = now
	lease.ExpiresAt = deadline
	err = s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		if err := requireDemandTargetAvailable(ctx, tx, lease.Scenario, lease.Variant); err != nil {
			return err
		}
		existing, err := getDemandLeaseTx(ctx, tx, lease.LeaseID)
		if err == nil {
			if existing.Scenario != lease.Scenario || existing.Variant != lease.Variant || existing.ConsumerID != lease.ConsumerID || existing.Kind != lease.Kind {
				return ErrDemandLeaseConflict
			}
			lease.CreatedAt = existing.CreatedAt
			_, err = tx.ExecContext(ctx, `UPDATE runtime_demand_leases
SET request_id = ?, last_renewed_at = ?, expires_at = ?, status = ?, stop_reason = '', metadata_json = ?
WHERE lease_id = ?`, lease.RequestID, formatTime(now), formatTime(deadline), DemandLeaseActive, lease.MetadataJSON, lease.LeaseID)
			if err != nil {
				return fmt.Errorf("renew demand lease during acquire: %w", err)
			}
			return s.auditDemandLeaseTx(ctx, tx, "acquired", lease)
		}
		if err != ErrNotFound {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO runtime_demand_leases
(lease_id, scenario, variant, consumer_id, kind, request_id, created_at, last_renewed_at, expires_at, status, stop_reason, metadata_json)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '', ?)`,
			lease.LeaseID, lease.Scenario, lease.Variant, lease.ConsumerID, lease.Kind, lease.RequestID,
			formatTime(now), formatTime(now), formatTime(deadline), DemandLeaseActive, lease.MetadataJSON)
		if err != nil {
			return fmt.Errorf("create demand lease: %w", err)
		}
		return s.auditDemandLeaseTx(ctx, tx, "acquired", lease)
	})
	if err != nil {
		return DemandLease{}, err
	}
	return s.getDemandLease(ctx, lease.LeaseID)
}

// RenewDemandLease extends an active hold. An expired hold cannot be revived
// through renewal; callers must acquire a new bounded request explicitly.
func (s *SQLiteStore) RenewDemandLease(ctx context.Context, leaseID string, ttl time.Duration) (DemandLease, error) {
	leaseID = strings.TrimSpace(leaseID)
	if leaseID == "" {
		return DemandLease{}, fmt.Errorf("renew demand lease: lease_id is required")
	}
	if err := validateDemandLeaseTTL(ttl); err != nil {
		return DemandLease{}, err
	}
	now := s.now()
	deadline := now.Add(NormalizeDemandLeaseTTL(ttl))
	var out DemandLease
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		current, err := getDemandLeaseTx(ctx, tx, leaseID)
		if err != nil {
			return err
		}
		if err := requireDemandTargetAvailable(ctx, tx, current.Scenario, current.Variant); err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `UPDATE runtime_demand_leases
SET last_renewed_at = ?, expires_at = ?
WHERE lease_id = ? AND status = ? AND expires_at > ?`,
			formatTime(now), formatTime(deadline), leaseID, DemandLeaseActive, formatTime(now))
		if err != nil {
			return fmt.Errorf("renew demand lease: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			current, getErr := getDemandLeaseTx(ctx, tx, leaseID)
			if getErr != nil {
				return getErr
			}
			if current.Status == DemandLeaseExpired {
				return ErrDemandLeaseExpired
			}
			if current.Status == DemandLeaseActive && !current.ExpiresAt.After(now) {
				_, _ = tx.ExecContext(ctx, `UPDATE runtime_demand_leases SET status = ?, stop_reason = ? WHERE lease_id = ? AND status = ?`, DemandLeaseExpired, demandLeaseExpiredReason, leaseID, DemandLeaseActive)
				return ErrDemandLeaseExpired
			}
			return ErrDemandLeaseConflict
		}
		out, err = getDemandLeaseTx(ctx, tx, leaseID)
		if err != nil {
			return err
		}
		return s.auditDemandLeaseTx(ctx, tx, "renewed", out)
	})
	if err != nil {
		return DemandLease{}, err
	}
	return out, nil
}

// Acquisition and stop claims serialize through the same registry. Once a
// stop has won, a new hold cannot promise a live target until that transition
// completes (or is restored). Check every stopping instance for this variant,
// including explicit stops, rather than reviving a process being terminated.
func requireDemandTargetAvailable(ctx context.Context, tx *sql.Tx, scenario, variant string) error {
	var stopping bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS (
SELECT 1 FROM runtime_instances WHERE scenario = ? AND variant = ? AND status = ?
)`, scenario, variant, StatusStopping).Scan(&stopping); err != nil {
		return fmt.Errorf("check demand target transition: %w", err)
	}
	if stopping {
		return fmt.Errorf("%w: %s/%s is stopping", ErrDemandLeaseConflict, scenario, variant)
	}
	return nil
}

// ReleaseDemandLease ends an active hold. Releasing an already terminal row is
// idempotent, while a just-expired row returns ErrDemandLeaseExpired.
func (s *SQLiteStore) ReleaseDemandLease(ctx context.Context, leaseID, reason string) (DemandLease, error) {
	leaseID = strings.TrimSpace(leaseID)
	if leaseID == "" {
		return DemandLease{}, fmt.Errorf("release demand lease: lease_id is required")
	}
	now := s.now()
	var out DemandLease
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE runtime_demand_leases
SET status = ?, stop_reason = ?
WHERE lease_id = ? AND status = ? AND expires_at > ?`,
			DemandLeaseReleased, strings.TrimSpace(reason), leaseID, DemandLeaseActive, formatTime(now))
		if err != nil {
			return fmt.Errorf("release demand lease: %w", err)
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			current, getErr := getDemandLeaseTx(ctx, tx, leaseID)
			if getErr != nil {
				return getErr
			}
			if current.Status == DemandLeaseActive && !current.ExpiresAt.After(now) {
				_, _ = tx.ExecContext(ctx, `UPDATE runtime_demand_leases SET status = ?, stop_reason = ? WHERE lease_id = ? AND status = ?`, DemandLeaseExpired, demandLeaseExpiredReason, leaseID, DemandLeaseActive)
				return ErrDemandLeaseExpired
			}
			out = current
			return nil
		}
		out, err = getDemandLeaseTx(ctx, tx, leaseID)
		if err == nil && (out.Kind == DemandLeaseProgram ||
			(out.Kind == DemandLeaseDependency && !IsDependencyConsumerID(out.ConsumerID))) {
			_, err = tx.ExecContext(ctx, `INSERT INTO runtime_demand_idle_windows
(scenario, variant, last_used_at, idle_until) VALUES (?, ?, ?, ?)
ON CONFLICT(scenario, variant) DO UPDATE SET last_used_at = excluded.last_used_at, idle_until = excluded.idle_until
WHERE excluded.idle_until > runtime_demand_idle_windows.idle_until`,
				out.Scenario, out.Variant, formatTime(now), formatTime(now.Add(DemandIdleGrace)))
		}
		if err != nil {
			return err
		}
		return s.auditDemandLeaseTx(ctx, tx, "released", out)
	})
	return out, err
}

// ExpireDemandLeases transitions all active holds at or before at to expired
// in one transaction and returns the rows that changed.
func (s *SQLiteStore) ExpireDemandLeases(ctx context.Context, at time.Time) ([]DemandLease, error) {
	at = at.UTC()
	var expired []DemandLease
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, demandLeaseSelectSQL+` WHERE status = ? AND expires_at <= ? ORDER BY expires_at ASC`, DemandLeaseActive, formatTime(at))
		if err != nil {
			return fmt.Errorf("list expired demand leases: %w", err)
		}
		expired, err = scanDemandLeases(rows)
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE runtime_demand_leases SET status = ?, stop_reason = ? WHERE status = ? AND expires_at <= ?`, DemandLeaseExpired, demandLeaseExpiredReason, DemandLeaseActive, formatTime(at))
		if err != nil {
			return err
		}
		for _, lease := range expired {
			if err := s.auditDemandLeaseTx(ctx, tx, "expired", lease); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for i := range expired {
		expired[i].Status = DemandLeaseExpired
		expired[i].StopReason = demandLeaseExpiredReason
	}
	return expired, nil
}

// ListDemandLeases returns durable demand evidence ordered by expiry.
func (s *SQLiteStore) ListDemandLeases(ctx context.Context, filter DemandLeaseFilter) ([]DemandLease, error) {
	query := demandLeaseSelectSQL
	args := []any{}
	clauses := []string{}
	if filter.Scenario != "" {
		clauses = append(clauses, "scenario = ?")
		args = append(args, filter.Scenario)
	}
	if filter.Variant != "" {
		clauses = append(clauses, "variant = ?")
		args = append(args, filter.Variant)
	}
	if filter.ConsumerID != "" {
		clauses = append(clauses, "consumer_id = ?")
		args = append(args, filter.ConsumerID)
	}
	if len(filter.Statuses) > 0 {
		clauses = append(clauses, "status IN ("+placeholders(len(filter.Statuses))+")")
		args = append(args, stringsToAny(filter.Statuses)...)
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY expires_at ASC"
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list demand leases: %w", err)
	}
	defer rows.Close()
	return scanDemandLeases(rows)
}

// HasActiveDemand answers whether any unexpired active hold exists for a
// scenario variant. It does not mutate expired rows; a sweeper owns archival.
func (s *SQLiteStore) HasActiveDemand(ctx context.Context, scenario, variant string) (bool, error) {
	scenario = strings.TrimSpace(scenario)
	variant = InstanceKey{Scenario: scenario, Variant: variant}.Normalize().Variant
	if scenario == "" {
		return false, fmt.Errorf("has active demand: scenario is required")
	}
	var exists int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM runtime_demand_leases WHERE scenario = ? AND variant = ? AND status = ? AND expires_at > ? LIMIT 1`, scenario, variant, DemandLeaseActive, formatTime(s.now())).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return exists == 1, nil
}

// ClaimDemandStopCandidates atomically transitions running demand-managed
// instances with no unexpired consumer hold to stopping. Starting instances
// remain under the heartbeat/stale-start reconciler so an in-flight lifecycle
// operation is never interrupted by a demand sweep. Keeping the eligibility check
// and state transition in one registry transaction closes the expiry/acquire
// race that would otherwise let a consumer acquire a lease while maintenance
// was deciding whether it was safe to stop a process.
func (s *SQLiteStore) ClaimDemandStopCandidates(ctx context.Context, at time.Time, reason string) ([]Instance, error) {
	if at.IsZero() {
		at = s.now()
	}
	at = at.UTC()
	if strings.TrimSpace(reason) == "" {
		reason = demandStopReason
	}
	var out []Instance
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		out = nil
		rows, err := tx.QueryContext(ctx, instanceSelectSQL+`
 WHERE supervision_policy = ?
   AND status = ?
   AND NOT EXISTS (
     SELECT 1 FROM runtime_demand_idle_windows w
      WHERE w.scenario = runtime_instances.scenario AND w.variant = runtime_instances.variant AND w.idle_until > ?
   )
   AND NOT EXISTS (
     SELECT 1 FROM runtime_demand_leases d
      WHERE d.scenario = runtime_instances.scenario
        AND d.variant = runtime_instances.variant
        AND d.status = ?
        AND d.expires_at > ?
   )
 ORDER BY updated_at ASC, instance_id ASC`,
			SupervisionPolicyDemand, StatusRunning, formatTime(at),
			DemandLeaseActive, formatTime(at))
		if err != nil {
			return fmt.Errorf("list demand stop candidates: %w", err)
		}
		candidates, scanErr := scanInstances(rows)
		closeErr := rows.Close()
		if scanErr != nil {
			return fmt.Errorf("scan demand stop candidates: %w", scanErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close demand stop candidates: %w", closeErr)
		}
		for _, candidate := range candidates {
			result, updateErr := tx.ExecContext(ctx, `
UPDATE runtime_instances
SET status = ?, stop_reason = ?, updated_at = ?
WHERE instance_id = ? AND generation = ?
  AND supervision_policy = ?
  AND status = ?
  AND NOT EXISTS (
    SELECT 1 FROM runtime_demand_idle_windows w
     WHERE w.scenario = runtime_instances.scenario AND w.variant = runtime_instances.variant AND w.idle_until > ?
  )
  AND NOT EXISTS (
    SELECT 1 FROM runtime_demand_leases d
     WHERE d.scenario = runtime_instances.scenario
       AND d.variant = runtime_instances.variant
       AND d.status = ?
       AND d.expires_at > ?
  )`,
				StatusStopping, reason, formatTime(at), candidate.InstanceID, candidate.Generation,
				SupervisionPolicyDemand, StatusRunning, formatTime(at),
				DemandLeaseActive, formatTime(at))
			if updateErr != nil {
				return fmt.Errorf("claim demand stop candidate %s: %w", candidate.InstanceID, updateErr)
			}
			affected, affectedErr := result.RowsAffected()
			if affectedErr != nil {
				return fmt.Errorf("inspect demand stop candidate %s: %w", candidate.InstanceID, affectedErr)
			}
			if affected == 0 {
				continue
			}
			claimed, getErr := getInstanceTx(ctx, tx, candidate.InstanceID)
			if getErr != nil {
				return fmt.Errorf("read claimed demand stop candidate %s: %w", candidate.InstanceID, getErr)
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO runtime_demand_stops
(instance_id, generation, signaling_started, created_at) VALUES (?, ?, 0, ?)
ON CONFLICT(instance_id) DO UPDATE SET generation = excluded.generation, signaling_started = 0, created_at = excluded.created_at`,
				claimed.InstanceID, claimed.Generation, formatTime(at)); err != nil {
				return fmt.Errorf("persist demand stop intent: %w", err)
			}
			if err := s.auditDemandInstanceTx(ctx, tx, "stop_claimed", claimed); err != nil {
				return err
			}
			out = append(out, claimed)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RestoreDemandStopCandidate clears a transient stopping claim after a
// preflight could not prove process identity. It is deliberately scoped to
// demand-managed rows and is idempotent for a candidate already reclaimed by a
// concurrent stop path.
func (s *SQLiteStore) RestoreDemandStopCandidate(ctx context.Context, instanceID string, generation int64, phase string) (Instance, error) {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return Instance{}, fmt.Errorf("restore demand stop candidate: instance_id is required")
	}
	now := s.now()
	var out Instance
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		var irreversible bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM runtime_demand_stops WHERE instance_id = ? AND generation = ? AND signaling_started = 1)`, instanceID, generation).Scan(&irreversible); err != nil {
			return err
		}
		if irreversible {
			return fmt.Errorf("%w: demand termination has begun; running cannot be restored", ErrDemandLeaseConflict)
		}
		result, err := tx.ExecContext(ctx, `
UPDATE runtime_instances
SET status = ?, phase = ?, stop_reason = '', updated_at = ?
WHERE instance_id = ? AND generation = ?
  AND supervision_policy = ? AND status = ?`,
			StatusRunning, phase, formatTime(now), instanceID, generation,
			SupervisionPolicyDemand, StatusStopping)
		if err != nil {
			return fmt.Errorf("restore demand stop candidate %s: %w", instanceID, err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("inspect restored demand stop candidate %s: %w", instanceID, err)
		}
		if affected == 0 {
			out, err = getInstanceTx(ctx, tx, instanceID)
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM runtime_demand_stops WHERE instance_id = ? AND generation = ?`, instanceID, generation); err != nil {
			return err
		}
		out, err = getInstanceTx(ctx, tx, instanceID)
		if err != nil {
			return err
		}
		return s.auditDemandInstanceTx(ctx, tx, "stop_restored", out)
	})
	if err != nil {
		return Instance{}, err
	}
	return out, nil
}

func (s *SQLiteStore) getDemandLease(ctx context.Context, leaseID string) (DemandLease, error) {
	return scanDemandLease(s.db.QueryRowContext(ctx, demandLeaseSelectSQL+` WHERE lease_id = ?`, leaseID))
}

type demandLeaseRow interface{ Scan(...any) error }

func getDemandLeaseTx(ctx context.Context, tx *sql.Tx, leaseID string) (DemandLease, error) {
	return scanDemandLease(tx.QueryRowContext(ctx, demandLeaseSelectSQL+` WHERE lease_id = ?`, leaseID))
}

func scanDemandLease(row demandLeaseRow) (DemandLease, error) {
	var out DemandLease
	var created, renewed, expires string
	if err := row.Scan(&out.LeaseID, &out.Scenario, &out.Variant, &out.ConsumerID, &out.Kind, &out.RequestID, &created, &renewed, &expires, &out.Status, &out.StopReason, &out.MetadataJSON); err != nil {
		if err == sql.ErrNoRows {
			return DemandLease{}, ErrNotFound
		}
		return DemandLease{}, err
	}
	var err error
	if out.CreatedAt, err = parseRequiredTime(created); err != nil {
		return DemandLease{}, err
	}
	if out.LastRenewedAt, err = parseRequiredTime(renewed); err != nil {
		return DemandLease{}, err
	}
	if out.ExpiresAt, err = parseRequiredTime(expires); err != nil {
		return DemandLease{}, err
	}
	return out, nil
}

func scanDemandLeases(rows *sql.Rows) ([]DemandLease, error) {
	defer rows.Close()
	var out []DemandLease
	for rows.Next() {
		lease, err := scanDemandLease(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, lease)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func stringsToAny(values []string) []any {
	args := make([]any, len(values))
	for i, value := range values {
		args[i] = value
	}
	return args
}
