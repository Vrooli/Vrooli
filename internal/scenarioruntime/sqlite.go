package scenarioruntime

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"

	// Importing modernc.org/sqlite registers the pure-Go SQLite driver.
	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

const defaultBindHost = "127.0.0.1"

const (
	runtimeRegistryTxRetryAttempts = 5
	runtimeRegistryTxRetryBase     = tuning.FastPersistenceRetryInterval
	runtimeRegistryTxRetryMax      = tuning.FastHealthPollInterval
)

type Config struct {
	HomeDir  string
	DBPath   string
	Clock    Clock
	ReadOnly bool
}

type SQLiteStore struct {
	db    *sql.DB
	clock Clock
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }

// DefaultDBPath resolves the runtime registry SQLite path from the runtime_home
// authority. When homeDir is empty it falls back to the sudo-aware resolver in
// internal/config (never bare os.UserHomeDir, which would point a sudo'd process
// at /root).
func DefaultDBPath(homeDir string) (string, error) {
	if strings.TrimSpace(homeDir) == "" {
		dir, err := config.HomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		homeDir = dir
	}
	return repocontract.RuntimeHomeEntryPath(homeDir, repocontract.HomeKeyRuntimeDB)
}

func NewSQLiteStore(ctx context.Context, cfg Config) (*SQLiteStore, error) {
	clk := cfg.Clock
	if clk == nil {
		clk = realClock{}
	}

	dbPath := cfg.DBPath
	if strings.TrimSpace(dbPath) == "" {
		resolved, err := DefaultDBPath(cfg.HomeDir)
		if err != nil {
			return nil, err
		}
		dbPath = resolved
	}
	readOnly := cfg.ReadOnly || strings.TrimSpace(os.Getenv("VROOLI_SANDBOX_MERGED")) != ""
	if !readOnly {
		if _, err := config.EnsureOwnedDir(filepath.Dir(dbPath)); err != nil {
			return nil, fmt.Errorf("prepare runtime registry directory: %w", err)
		}
	}

	dsn := buildDSN(dbPath)
	if readOnly {
		dsn = buildReadOnlyDSN(dbPath)
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open runtime registry sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)

	store := &SQLiteStore{db: db, clock: clk}
	if err := store.ensureSchema(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *SQLiteStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *SQLiteStore) CreateInstance(ctx context.Context, in Instance) (Instance, error) {
	if strings.TrimSpace(in.InstanceID) == "" {
		in.InstanceID = newID("inst")
	}
	if strings.TrimSpace(in.Scenario) == "" {
		return Instance{}, fmt.Errorf("create instance: scenario is required")
	}
	// Normalize the variant through the InstanceKey SSOT so an empty/whitespace
	// variant becomes "live" and casing is canonical — the per-(scenario,variant)
	// uniqueness and generation counter depend on it being normalized.
	in.Variant = InstanceKey{Scenario: in.Scenario, Variant: in.Variant}.Normalize().Variant
	now := s.now()
	if in.Status == "" {
		in.Status = StatusStarting
	}
	if in.OwnerKind == "" {
		in.OwnerKind = OwnerKindLifecycle
	}
	if in.SupervisionPolicy == "" {
		in.SupervisionPolicy = SupervisionPolicyManaged
	}
	if in.StartedAt.IsZero() {
		in.StartedAt = now
	}
	in.UpdatedAt = now
	in.SchemaVersion = SchemaVersion

	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		if in.Generation <= 0 {
			next, err := nextGeneration(ctx, tx, in.Scenario, in.Variant)
			if err != nil {
				return err
			}
			in.Generation = next
		}
		_, err := tx.ExecContext(ctx, `
INSERT INTO runtime_instances (
  instance_id, scenario, variant, generation, scope_path, sandbox_id, status, phase,
  started_at, updated_at, last_heartbeat_at, heartbeat_deadline_at, stopped_at,
  stop_reason, owner_kind, owner_pid, working_dir, host_boot_id, host_session_id,
  supervisor_id, supervised_at, last_reconciled_at, reconciliation_status,
  reconciliation_reason, supervision_policy, schema_version
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			in.InstanceID, in.Scenario, in.Variant, in.Generation, in.ScopePath, in.SandboxID, in.Status, in.Phase,
			formatTime(in.StartedAt), formatTime(in.UpdatedAt), formatOptionalTime(in.LastHeartbeatAt),
			formatOptionalTime(in.HeartbeatDeadlineAt), formatOptionalTime(in.StoppedAt),
			in.StopReason, in.OwnerKind, optionalIntValue(in.OwnerPID), in.WorkingDir, in.HostBootID, in.HostSessionID,
			in.SupervisorID, formatOptionalTime(in.SupervisedAt), formatOptionalTime(in.LastReconciledAt),
			in.ReconciliationStatus, in.ReconciliationReason, in.SupervisionPolicy, in.SchemaVersion)
		if err != nil {
			return fmt.Errorf("insert runtime instance: %w", err)
		}
		return nil
	})
	if err != nil {
		return Instance{}, err
	}
	return in, nil
}

func (s *SQLiteStore) UpdateInstanceStatus(ctx context.Context, instanceID string, generation int64, status string, phase string) (Instance, error) {
	now := s.now()
	var out Instance
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
UPDATE runtime_instances
SET status = ?, phase = ?, updated_at = ?
WHERE instance_id = ? AND generation = ?`,
			status, phase, formatTime(now), instanceID, generation)
		if err != nil {
			return fmt.Errorf("update runtime instance: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("inspect runtime instance update: %w", err)
		}
		if affected == 0 {
			return ErrStaleGeneration
		}
		out, err = getInstanceTx(ctx, tx, instanceID)
		return err
	})
	if err != nil {
		return Instance{}, err
	}
	return out, nil
}

func (s *SQLiteStore) GetInstance(ctx context.Context, instanceID string) (Instance, error) {
	return scanInstance(s.db.QueryRowContext(ctx, instanceSelectSQL+` WHERE instance_id = ?`, instanceID))
}

func (s *SQLiteStore) ListInstances(ctx context.Context, filter InstanceFilter) ([]Instance, error) {
	query := instanceSelectSQL
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
	if len(filter.Statuses) > 0 {
		clauses = append(clauses, "status IN ("+placeholders(len(filter.Statuses))+")")
		for _, status := range filter.Statuses {
			args = append(args, status)
		}
	}
	if filter.SupervisorID != "" {
		clauses = append(clauses, "supervisor_id = ?")
		args = append(args, filter.SupervisorID)
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY scenario ASC, generation DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list runtime instances: %w", err)
	}
	defer rows.Close()
	return scanInstances(rows)
}

func (s *SQLiteStore) AcquirePortClaim(ctx context.Context, claim PortClaim) (PortClaim, error) {
	if strings.TrimSpace(claim.ClaimID) == "" {
		claim.ClaimID = newID("claim")
	}
	if strings.TrimSpace(claim.InstanceID) == "" {
		return PortClaim{}, fmt.Errorf("acquire port claim: instance_id is required")
	}
	if strings.TrimSpace(claim.Scenario) == "" {
		return PortClaim{}, fmt.Errorf("acquire port claim: scenario is required")
	}
	claim.Variant = InstanceKey{Scenario: claim.Scenario, Variant: claim.Variant}.Normalize().Variant
	if claim.Port <= 0 {
		return PortClaim{}, fmt.Errorf("acquire port claim: port must be positive")
	}
	if claim.BindHost == "" {
		claim.BindHost = defaultBindHost
	}
	if claim.Status == "" {
		claim.Status = ClaimStatusReserved
	}
	if claim.ListenerStatus == "" {
		claim.ListenerStatus = ListenerStatusUnknown
	}
	now := s.now()
	if claim.CreatedAt.IsZero() {
		claim.CreatedAt = now
	}
	claim.UpdatedAt = now

	err := s.withTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
INSERT INTO runtime_port_claims (
  claim_id, instance_id, scenario, variant, port_name, env_var, port, bind_host, url,
  status, created_at, updated_at, expires_at, last_bound_at, last_listener_check_at,
  last_listener_seen_at, first_unbound_at, consecutive_listener_misses, listener_status,
  listener_pid, listener_process_label
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			claim.ClaimID, claim.InstanceID, claim.Scenario, claim.Variant, claim.PortName, claim.EnvVar, claim.Port, claim.BindHost,
			claim.URL, claim.Status, formatTime(claim.CreatedAt), formatTime(claim.UpdatedAt),
			formatOptionalTime(claim.ExpiresAt), formatOptionalTime(claim.LastBoundAt),
			formatOptionalTime(claim.LastListenerCheckAt), formatOptionalTime(claim.LastListenerSeenAt),
			formatOptionalTime(claim.FirstUnboundAt), claim.ConsecutiveListenerMisses, claim.ListenerStatus,
			optionalIntValue(claim.ListenerPID), claim.ListenerProcessLabel)
		if err != nil {
			if isUniqueConstraint(err) {
				return ErrActiveClaimConflict
			}
			return fmt.Errorf("insert runtime port claim: %w", err)
		}
		return nil
	})
	if err != nil {
		return PortClaim{}, err
	}
	return claim, nil
}

func (s *SQLiteStore) ReleasePortClaim(ctx context.Context, claimID string) (PortClaim, error) {
	return s.updateClaimStatus(ctx, claimID, ClaimStatusReleased)
}

// RenewReservedPortClaimsForInstance pushes expires_at forward for every claim
// of the instance that is still reserved (not yet bound), returning how many
// were renewed. The lifecycle calls this alongside its instance heartbeats so
// a slow start cannot have its reservations expired-and-stolen by a concurrent
// allocation. Bound claims carry no expiry (cleared on bind) and are skipped
// by the status filter.
func (s *SQLiteStore) RenewReservedPortClaimsForInstance(ctx context.Context, instanceID string, expiresAt time.Time) (int, error) {
	if strings.TrimSpace(instanceID) == "" {
		return 0, fmt.Errorf("renew reserved port claims: instance_id is required")
	}
	now := s.now()
	var renewed int64
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
UPDATE runtime_port_claims
SET expires_at = ?, updated_at = ?
WHERE instance_id = ? AND status = ?`,
			formatTime(expiresAt.UTC()), formatTime(now), instanceID, ClaimStatusReserved)
		if err != nil {
			return fmt.Errorf("renew reserved port claims: %w", err)
		}
		renewed, err = result.RowsAffected()
		if err != nil {
			return fmt.Errorf("inspect reserved port claim renewal: %w", err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return int(renewed), nil
}

func (s *SQLiteStore) BindPortClaim(ctx context.Context, claimID string) (PortClaim, error) {
	now := s.now()
	var out PortClaim
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		// Restrict the UPDATE to rows still in `reserved`. If the row's
		// status was flipped to `expired`, `released`, or `bound` by
		// another path between acquire and bind, the UPDATE matches zero
		// rows and we return a typed error instead of blowing past the
		// partial unique index `(port, bind_host) WHERE status IN
		// ('reserved','bound')` — which would otherwise re-add this row
		// to the index and collide with whatever active row replaced it.
		result, err := tx.ExecContext(ctx, `
UPDATE runtime_port_claims
SET status = ?, updated_at = ?, expires_at = NULL, last_bound_at = ?
WHERE claim_id = ? AND status = ?`,
			ClaimStatusBound, formatTime(now), formatTime(now), claimID, ClaimStatusReserved)
		if err != nil {
			return fmt.Errorf("bind runtime port claim: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("inspect runtime port claim bind: %w", err)
		}
		if affected == 0 {
			existing, err := getPortClaimTx(ctx, tx, claimID)
			if err != nil {
				return err
			}
			return fmt.Errorf("%w: claim %s status=%s", ErrClaimNotReservable, claimID, existing.Status)
		}
		out, err = getPortClaimTx(ctx, tx, claimID)
		return err
	})
	if err != nil {
		return PortClaim{}, err
	}
	return out, nil
}

func (s *SQLiteStore) ReleaseActivePortClaimsForInstance(ctx context.Context, instanceID string) ([]PortClaim, error) {
	if strings.TrimSpace(instanceID) == "" {
		return nil, fmt.Errorf("release active port claims: instance_id is required")
	}
	now := s.now()
	var released []PortClaim
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, portClaimSelectSQL+`
WHERE instance_id = ? AND status IN (?, ?)
ORDER BY port ASC`,
			instanceID, ClaimStatusReserved, ClaimStatusBound)
		if err != nil {
			return fmt.Errorf("list active runtime port claims for release: %w", err)
		}
		released, err = scanPortClaims(rows)
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		if err != nil {
			return err
		}
		if len(released) == 0 {
			return nil
		}
		result, err := tx.ExecContext(ctx, `
UPDATE runtime_port_claims
SET status = ?, updated_at = ?
WHERE instance_id = ? AND status IN (?, ?)`,
			ClaimStatusReleased, formatTime(now), instanceID, ClaimStatusReserved, ClaimStatusBound)
		if err != nil {
			return fmt.Errorf("release active runtime port claims: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("inspect runtime port claim release: %w", err)
		}
		if int(affected) != len(released) {
			return ErrStaleGeneration
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for i := range released {
		released[i].Status = ClaimStatusReleased
		released[i].UpdatedAt = now
	}
	return released, nil
}

func (s *SQLiteStore) ExpirePortClaim(ctx context.Context, claimID string) (PortClaim, error) {
	return s.updateClaimStatus(ctx, claimID, ClaimStatusExpired)
}

func (s *SQLiteStore) ExpireInstance(ctx context.Context, instanceID string, reason string) (Instance, error) {
	if strings.TrimSpace(instanceID) == "" {
		return Instance{}, fmt.Errorf("expire runtime instance: instance_id is required")
	}
	now := s.now()
	if strings.TrimSpace(reason) == "" {
		reason = "runtime reconciliation expired stale instance"
	}
	var out Instance
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
UPDATE runtime_instances
SET status = ?, updated_at = ?, stopped_at = ?, stop_reason = ?
WHERE instance_id = ? AND status IN (?, ?)`,
			StatusExpired, formatTime(now), formatTime(now), reason, instanceID, StatusStarting, StatusRunning)
		if err != nil {
			return fmt.Errorf("expire runtime instance: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("inspect runtime instance expiry: %w", err)
		}
		if affected == 0 {
			return ErrNotFound
		}
		out, err = getInstanceTx(ctx, tx, instanceID)
		return err
	})
	if err != nil {
		return Instance{}, err
	}
	return out, nil
}

// ExpireStaleStartingLeases reaps leases stuck in status='starting'. A lease is
// only a candidate once its heartbeat deadline has elapsed, and only condemned
// when the guard can show the starter is gone (see StaleStartingTrigger) — a
// live owner mid-setup keeps its lease no matter how long the build runs.
func (s *SQLiteStore) ExpireStaleStartingLeases(ctx context.Context, at time.Time, guard StartingLeaseGuard) ([]Instance, error) {
	at = at.UTC()
	var expired []Instance
	var reasons []string
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, instanceSelectSQL+`
WHERE status = ? AND heartbeat_deadline_at IS NOT NULL AND heartbeat_deadline_at <= ?
ORDER BY scenario ASC, generation DESC`,
			StatusStarting, formatTime(at))
		if err != nil {
			return fmt.Errorf("list stale starting runtime leases: %w", err)
		}
		candidates, err := scanInstances(rows)
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		if err != nil {
			return err
		}
		expired = expired[:0]
		reasons = reasons[:0]
		for _, candidate := range candidates {
			trigger, ok := StaleStartingTrigger(candidate, guard, at)
			if !ok {
				continue
			}
			expired = append(expired, candidate)
			reasons = append(reasons, trigger)
		}
		if len(expired) == 0 {
			return nil
		}
		// Expire by explicit (instance_id, generation) rather than by re-running
		// the deadline predicate, so a row that changed under us fails loudly
		// instead of being swept up by a broad UPDATE.
		for i, instance := range expired {
			result, err := tx.ExecContext(ctx, `
UPDATE runtime_instances
SET status = ?, updated_at = ?, stop_reason = ?
WHERE instance_id = ? AND generation = ? AND status = ?`,
				StatusExpired, formatTime(s.now()), staleStartingStopReason(reasons[i]),
				instance.InstanceID, instance.Generation, StatusStarting)
			if err != nil {
				return fmt.Errorf("expire stale starting runtime lease %s: %w", instance.InstanceID, err)
			}
			affected, err := result.RowsAffected()
			if err != nil {
				return fmt.Errorf("inspect stale starting runtime lease expiry %s: %w", instance.InstanceID, err)
			}
			if affected == 0 {
				return ErrStaleGeneration
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for i := range expired {
		expired[i].Status = StatusExpired
		expired[i].UpdatedAt = s.now()
		expired[i].StopReason = staleStartingStopReason(reasons[i])
	}
	return expired, nil
}

func staleStartingStopReason(trigger string) string {
	if strings.TrimSpace(trigger) == "" {
		return staleStartingStopReasonPrefix
	}
	return staleStartingStopReasonPrefix + " (" + trigger + ")"
}

func (s *SQLiteStore) ListPortClaims(ctx context.Context, filter PortClaimFilter) ([]PortClaim, error) {
	query := portClaimSelectSQL
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
	if filter.InstanceID != "" {
		clauses = append(clauses, "instance_id = ?")
		args = append(args, filter.InstanceID)
	}
	if len(filter.Statuses) > 0 {
		clauses = append(clauses, "status IN ("+placeholders(len(filter.Statuses))+")")
		for _, status := range filter.Statuses {
			args = append(args, status)
		}
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY scenario ASC, port ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list runtime port claims: %w", err)
	}
	defer rows.Close()
	return scanPortClaims(rows)
}

func (s *SQLiteStore) ListExpiredActivePortClaims(ctx context.Context, at time.Time) ([]PortClaim, error) {
	rows, err := s.db.QueryContext(ctx, portClaimSelectSQL+`
 WHERE status IN (?, ?) AND expires_at IS NOT NULL AND expires_at <= ?
 ORDER BY scenario ASC, port ASC`,
		ClaimStatusReserved, ClaimStatusBound, formatTime(at.UTC()))
	if err != nil {
		return nil, fmt.Errorf("list expired runtime port claims: %w", err)
	}
	defer rows.Close()
	return scanPortClaims(rows)
}

// PruneTerminalPortClaims deletes expired/released claim rows whose last
// update is older than the cutoff. Terminal rows are resolved history; without
// retention they accumulate forever and every stale-claim consumer ends up
// counting tombstones.
func (s *SQLiteStore) PruneTerminalPortClaims(ctx context.Context, before time.Time) (int, error) {
	result, err := s.db.ExecContext(ctx, `
DELETE FROM runtime_port_claims
 WHERE status IN (?, ?) AND updated_at < ?`,
		ClaimStatusExpired, ClaimStatusReleased, formatTime(before.UTC()))
	if err != nil {
		return 0, fmt.Errorf("prune terminal runtime port claims: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("prune terminal runtime port claims: %w", err)
	}
	return int(affected), nil
}

func (s *SQLiteStore) UpdatePortClaimListenerEvidence(ctx context.Context, claimID string, evidence ListenerObservation) (PortClaim, error) {
	if strings.TrimSpace(claimID) == "" {
		return PortClaim{}, fmt.Errorf("update listener evidence: claim_id is required")
	}
	status := normalizeListenerStatus(evidence.Status)
	checkedAt := evidence.CheckedAt
	if checkedAt.IsZero() {
		checkedAt = s.now()
	}
	checkedAt = checkedAt.UTC()

	var out PortClaim
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		current, err := getPortClaimTx(ctx, tx, claimID)
		if err != nil {
			return err
		}
		lastSeenAt := current.LastListenerSeenAt
		firstUnboundAt := current.FirstUnboundAt
		misses := current.ConsecutiveListenerMisses
		switch status {
		case ListenerStatusListening, ListenerStatusForeignListener:
			lastSeenAt = &checkedAt
			firstUnboundAt = nil
			misses = 0
		case ListenerStatusNotListening:
			if firstUnboundAt == nil {
				firstUnboundAt = &checkedAt
			}
			misses++
		case ListenerStatusInspectionUnavailable, ListenerStatusUnknown:
			// Inspection gaps should be visible, but they are not listener misses.
		}

		result, err := tx.ExecContext(ctx, `
UPDATE runtime_port_claims
SET updated_at = ?, last_listener_check_at = ?, last_listener_seen_at = ?,
  first_unbound_at = ?, consecutive_listener_misses = ?, listener_status = ?,
  listener_pid = ?, listener_process_label = ?
WHERE claim_id = ?`,
			formatTime(s.now()), formatTime(checkedAt), formatOptionalTime(lastSeenAt),
			formatOptionalTime(firstUnboundAt), misses, status, optionalIntValue(evidence.PID),
			evidence.ProcessLabel, claimID)
		if err != nil {
			return fmt.Errorf("update runtime port listener evidence: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("inspect runtime port listener evidence update: %w", err)
		}
		if affected == 0 {
			return ErrNotFound
		}
		out, err = getPortClaimTx(ctx, tx, claimID)
		return err
	})
	if err != nil {
		return PortClaim{}, err
	}
	return out, nil
}

func (s *SQLiteStore) UpsertHealthSnapshot(ctx context.Context, snapshot HealthSnapshot) (HealthSnapshot, error) {
	if snapshot.InstanceID == "" {
		return HealthSnapshot{}, fmt.Errorf("upsert health snapshot: instance_id is required")
	}
	if snapshot.Scenario == "" {
		return HealthSnapshot{}, fmt.Errorf("upsert health snapshot: scenario is required")
	}
	if snapshot.Status == "" {
		snapshot.Status = HealthStatusUnknown
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO runtime_health_snapshots (
  instance_id, scenario, status, readiness, checked_at, latency_ms, error, response_json, schema_valid
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(instance_id) DO UPDATE SET
  scenario = excluded.scenario,
  status = excluded.status,
  readiness = excluded.readiness,
  checked_at = excluded.checked_at,
  latency_ms = excluded.latency_ms,
  error = excluded.error,
  response_json = excluded.response_json,
  schema_valid = excluded.schema_valid`,
		snapshot.InstanceID, snapshot.Scenario, snapshot.Status, optionalBoolValue(snapshot.Readiness),
		formatOptionalTime(snapshot.CheckedAt), optionalInt64Value(snapshot.LatencyMillis),
		snapshot.Error, snapshot.ResponseJSON, optionalBoolValue(snapshot.SchemaValid))
	if err != nil {
		return HealthSnapshot{}, fmt.Errorf("upsert health snapshot: %w", err)
	}
	return snapshot, nil
}

func (s *SQLiteStore) GetHealthSnapshot(ctx context.Context, instanceID string) (HealthSnapshot, error) {
	return scanHealthSnapshot(s.db.QueryRowContext(ctx, `
SELECT instance_id, scenario, status, readiness, checked_at, latency_ms, error, response_json, schema_valid
FROM runtime_health_snapshots
WHERE instance_id = ?`, instanceID))
}

func (s *SQLiteStore) AddProcessRef(ctx context.Context, ref ProcessRef) (ProcessRef, error) {
	if strings.TrimSpace(ref.RefID) == "" {
		ref.RefID = newID("proc")
	}
	if strings.TrimSpace(ref.InstanceID) == "" {
		return ProcessRef{}, fmt.Errorf("add process ref: instance_id is required")
	}
	if ref.StartedAt.IsZero() {
		ref.StartedAt = s.now()
	}
	if strings.TrimSpace(ref.Status) == "" {
		ref.Status = "running"
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO runtime_process_refs (
  ref_id, instance_id, pid, pgid, process_id, step, command, log_file, status, started_at, ended_at, host_boot_id
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ref.RefID, ref.InstanceID, optionalIntValue(ref.PID), optionalIntValue(ref.PGID),
		ref.ProcessID, ref.Step, ref.Command, ref.LogFile, ref.Status,
		formatTime(ref.StartedAt), formatOptionalTime(ref.EndedAt), ref.HostBootID)
	if err != nil {
		return ProcessRef{}, fmt.Errorf("insert runtime process ref: %w", err)
	}
	return ref, nil
}

func (s *SQLiteStore) UpdateProcessRefStatus(ctx context.Context, refID string, status string, endedAt *time.Time) (ProcessRef, error) {
	if strings.TrimSpace(refID) == "" {
		return ProcessRef{}, fmt.Errorf("update process ref: ref_id is required")
	}
	var ended any
	if endedAt != nil {
		normalized := endedAt.UTC()
		ended = formatTime(normalized)
	}
	var out ProcessRef
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
UPDATE runtime_process_refs
SET status = ?, ended_at = ?
WHERE ref_id = ?`,
			status, ended, refID)
		if err != nil {
			return fmt.Errorf("update runtime process ref: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("inspect runtime process ref update: %w", err)
		}
		if affected == 0 {
			return ErrNotFound
		}
		out, err = getProcessRefTx(ctx, tx, refID)
		return err
	})
	if err != nil {
		return ProcessRef{}, err
	}
	return out, nil
}

func (s *SQLiteStore) ListProcessRefs(ctx context.Context, instanceID string) ([]ProcessRef, error) {
	query := processRefSelectSQL
	args := []any{}
	if strings.TrimSpace(instanceID) != "" {
		query += ` WHERE instance_id = ?`
		args = append(args, instanceID)
	}
	query += ` ORDER BY started_at ASC, ref_id ASC`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list runtime process refs: %w", err)
	}
	defer rows.Close()
	return scanProcessRefs(rows)
}

func (s *SQLiteStore) RecordEvent(ctx context.Context, event Event) (Event, error) {
	if strings.TrimSpace(event.EventID) == "" {
		event.EventID = newID("evt")
	}
	if strings.TrimSpace(event.EventType) == "" {
		return Event{}, fmt.Errorf("record runtime event: event_type is required")
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = s.now()
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO runtime_events (event_id, instance_id, scenario, event_type, created_at, details_json)
VALUES (?, ?, ?, ?, ?, ?)`,
		event.EventID, nullableString(event.InstanceID), event.Scenario, event.EventType,
		formatTime(event.CreatedAt), event.DetailsJSON)
	if err != nil {
		return Event{}, fmt.Errorf("insert runtime event: %w", err)
	}
	return event, nil
}

func (s *SQLiteStore) updateClaimStatus(ctx context.Context, claimID string, status string) (PortClaim, error) {
	now := s.now()
	var out PortClaim
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
UPDATE runtime_port_claims
SET status = ?, updated_at = ?
WHERE claim_id = ?`,
			status, formatTime(now), claimID)
		if err != nil {
			return fmt.Errorf("update runtime port claim: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("inspect runtime port claim update: %w", err)
		}
		if affected == 0 {
			return ErrNotFound
		}
		out, err = getPortClaimTx(ctx, tx, claimID)
		return err
	})
	if err != nil {
		return PortClaim{}, err
	}
	return out, nil
}

func (s *SQLiteStore) now() time.Time {
	return s.clock.Now().UTC()
}

func (s *SQLiteStore) withTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin runtime registry transaction: %w", err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit runtime registry transaction: %w", err)
	}
	return nil
}

func (s *SQLiteStore) withRetryableTx(ctx context.Context, fn func(*sql.Tx) error) error {
	var err error
	delay := runtimeRegistryTxRetryBase
	for attempt := 1; attempt <= runtimeRegistryTxRetryAttempts; attempt++ {
		err = s.withTx(ctx, fn)
		if err == nil || !isSQLiteLockContention(err) || attempt == runtimeRegistryTxRetryAttempts {
			return err
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		delay *= 2
		if delay > runtimeRegistryTxRetryMax {
			delay = runtimeRegistryTxRetryMax
		}
	}
	return err
}

func isSQLiteLockContention(err error) bool {
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) {
		switch sqliteErr.Code() & 0xff {
		case sqlite3.SQLITE_BUSY, sqlite3.SQLITE_LOCKED:
			return true
		}
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") || strings.Contains(msg, "table in the database is locked")
}

func buildDSN(path string) string {
	return "file:" + url.PathEscape(path) + "?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)"
}

func buildReadOnlyDSN(path string) string {
	if _, err := os.Stat(path + "-wal"); err != nil {
		return "file:" + url.PathEscape(path) + "?mode=ro&immutable=1&_pragma=foreign_keys(ON)&_pragma=query_only(ON)&_pragma=busy_timeout(10000)&_pragma=temp_store(MEMORY)"
	}
	// A WAL-backed registry still needs SQLite's shared-memory locking. The
	// nolock URI flag makes modernc unable to open a live WAL database and
	// produces SQLITE_CANTOPEN, so keep the connection read-only while allowing
	// SQLite to coordinate with the writer.
	return "file:" + url.PathEscape(path) + "?mode=ro&_pragma=foreign_keys(ON)&_pragma=query_only(ON)&_pragma=busy_timeout(10000)&_pragma=temp_store(MEMORY)"
}

func nextGeneration(ctx context.Context, tx *sql.Tx, scenario, variant string) (int64, error) {
	var generation sql.NullInt64
	if err := tx.QueryRowContext(ctx, `SELECT MAX(generation) FROM runtime_instances WHERE scenario = ? AND variant = ?`, scenario, variant).Scan(&generation); err != nil {
		return 0, fmt.Errorf("read next runtime generation: %w", err)
	}
	if !generation.Valid {
		return 1, nil
	}
	return generation.Int64 + 1, nil
}

func newID(prefix string) string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(b[:])
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", n), ",")
}

func isUniqueConstraint(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "constraint failed") || strings.Contains(msg, "UNIQUE constraint failed")
}

func normalizeListenerStatus(status string) string {
	switch status {
	case ListenerStatusListening, ListenerStatusNotListening, ListenerStatusInspectionUnavailable, ListenerStatusForeignListener:
		return status
	default:
		return ListenerStatusUnknown
	}
}

func optionalIntValue(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

func optionalInt64Value(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

func optionalBoolValue(v *bool) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullableString(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}

func formatTime(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000000000Z07:00")
}

func formatOptionalTime(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	return formatTime(*t)
}

func parseOptionalTime(v sql.NullString) (*time.Time, error) {
	if !v.Valid || strings.TrimSpace(v.String) == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339Nano, v.String)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func parseRequiredTime(v string) (time.Time, error) {
	if strings.TrimSpace(v) == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339Nano, v)
}

func ptrInt(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	i := int(v.Int64)
	return &i
}

func ptrInt64(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	i := v.Int64
	return &i
}

func ptrBool(v sql.NullBool) *bool {
	if !v.Valid {
		return nil
	}
	b := v.Bool
	return &b
}

func mapRowErr(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
