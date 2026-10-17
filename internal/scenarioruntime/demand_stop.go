package scenarioruntime

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"

	platform "github.com/vrooli/platform-go"
)

// The intent survives controller crashes. File-lock ownership does not: the
// kernel releases it when its owner exits, without an unsafe timed takeover.
const demandStopSchemaSQL = `
CREATE TABLE IF NOT EXISTS runtime_demand_stops (
  instance_id TEXT PRIMARY KEY,
  generation INTEGER NOT NULL,
  signaling_started INTEGER NOT NULL DEFAULT 0 CHECK (signaling_started IN (0, 1)),
  created_at TEXT NOT NULL,
  FOREIGN KEY(instance_id) REFERENCES runtime_instances(instance_id) ON DELETE CASCADE
);`

// AcquireDemandStopLock serializes local teardown across registry handles and
// processes. The lock file must remain in place: unlinking it permits two owners
// to lock different inodes. Canonicalization makes symlink aliases share a lock.
func (s *SQLiteStore) AcquireDemandStopLock(ctx context.Context) (func(), error) {
	path, err := filepath.EvalSymlinks(s.path)
	if err != nil {
		return nil, fmt.Errorf("resolve demand stop lock authority: %w", err)
	}
	return platform.AcquireFileLockContext(ctx, path+".demand-stop.lock")
}

// PendingDemandStops excludes ordinary operator stops and legacy stopping rows
// without an intent. Guessing ownership from free-form stop reasons is unsafe.
func (s *SQLiteStore) PendingDemandStops(ctx context.Context) ([]Instance, error) {
	rows, err := s.db.QueryContext(ctx, instanceSelectSQL+`
 WHERE status = ? AND supervision_policy = ? AND EXISTS (
 SELECT 1 FROM runtime_demand_stops d WHERE d.instance_id = runtime_instances.instance_id
 AND d.generation = runtime_instances.generation)
 ORDER BY updated_at, instance_id`, StatusStopping, SupervisionPolicyDemand)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInstances(rows)
}

// DemandStopStarted must be read again under the stop lock, not inferred from
// a candidate snapshot taken before another controller completed its attempt.
func (s *SQLiteStore) DemandStopStarted(ctx context.Context, instanceID string, generation int64) (bool, error) {
	var started bool
	err := s.db.QueryRowContext(ctx, `SELECT d.signaling_started FROM runtime_demand_stops d
 JOIN runtime_instances i ON i.instance_id = d.instance_id AND i.generation = d.generation
 WHERE d.instance_id = ? AND d.generation = ? AND i.status = ? AND i.supervision_policy = ?`,
		instanceID, generation, StatusStopping, SupervisionPolicyDemand).Scan(&started)
	if err == sql.ErrNoRows {
		return false, ErrNotFound
	}
	return started, err
}

// BeginDemandStop commits the irreversible boundary before the first signal.
// A retry must never restore running after this point, even if its own preflight
// fails. The caller holds AcquireDemandStopLock through finalization.
func (s *SQLiteStore) BeginDemandStop(ctx context.Context, instanceID string, generation int64) error {
	return s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		instance, err := getInstanceTx(ctx, tx, instanceID)
		if err != nil {
			return err
		}
		if instance.Generation != generation || instance.Status != StatusStopping || instance.SupervisionPolicy != SupervisionPolicyDemand {
			return ErrNotFound
		}
		var started bool
		if err := tx.QueryRowContext(ctx, `SELECT signaling_started FROM runtime_demand_stops WHERE instance_id = ? AND generation = ?`, instanceID, generation).Scan(&started); err != nil {
			return err
		}
		if started {
			return nil
		}
		if _, err := tx.ExecContext(ctx, `UPDATE runtime_demand_stops SET signaling_started = 1 WHERE instance_id = ? AND generation = ?`, instanceID, generation); err != nil {
			return err
		}
		return s.auditDemandInstanceTx(ctx, tx, "stop_signaling", instance)
	})
}

// CompleteDemandStop releases ports and marks the generation stopped atomically.
// Failed persistence leaves both claims and the durable intent available to a
// later owner; a stale generation cannot release a newer generation's ports.
func (s *SQLiteStore) CompleteDemandStop(ctx context.Context, instanceID string, generation int64) error {
	return s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		current, err := getInstanceTx(ctx, tx, instanceID)
		if err != nil {
			return err
		}
		if current.Generation != generation {
			return ErrStaleGeneration
		}
		var started bool
		if err := tx.QueryRowContext(ctx, `SELECT signaling_started FROM runtime_demand_stops WHERE instance_id = ? AND generation = ?`, instanceID, generation).Scan(&started); err != nil {
			return err
		}
		if current.Status != StatusStopping || current.SupervisionPolicy != SupervisionPolicyDemand || !started {
			return fmt.Errorf("demand stop has not reached its finalization boundary")
		}
		now := formatTime(s.now())
		if _, err := tx.ExecContext(ctx, `UPDATE runtime_port_claims SET status = ?, updated_at = ? WHERE instance_id = ? AND status IN (?, ?)`, ClaimStatusReleased, now, instanceID, ClaimStatusReserved, ClaimStatusBound); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE runtime_instances SET status = ?, updated_at = ?, stopped_at = ?, stop_reason = ? WHERE instance_id = ? AND generation = ?`, StatusStopped, now, now, demandStopReason, instanceID, generation); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM runtime_demand_stops WHERE instance_id = ? AND generation = ?`, instanceID, generation); err != nil {
			return err
		}
		return s.auditDemandInstanceTx(ctx, tx, "stopped", current)
	})
}

// ErrDemandStopOwned prevents generic reapers and lifecycle cleanup from
// bypassing a pending demand teardown's ownership and port-release proof.
var ErrDemandStopOwned = errors.New("runtime is owned by pending demand teardown")

func requireNoDemandStopTx(ctx context.Context, tx *sql.Tx, instanceID string) error {
	var owned bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS (
SELECT 1 FROM runtime_demand_stops d JOIN runtime_instances i
ON i.instance_id = d.instance_id AND i.generation = d.generation
WHERE i.instance_id = ? AND i.status = ? AND i.supervision_policy = ?)`, instanceID, StatusStopping, SupervisionPolicyDemand).Scan(&owned); err != nil {
		return err
	}
	if owned {
		return ErrDemandStopOwned
	}
	return nil
}
