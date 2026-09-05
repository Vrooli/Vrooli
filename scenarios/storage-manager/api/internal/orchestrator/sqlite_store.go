package orchestrator

// DOC: docs/reference/providers.md#policy-profiles

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"storage-manager/internal/cleanup"
)

//go:embed schema.sql
var schemaSQL string

// Schema returns the tables this package owns. Registered in
// modules.AllSchemas. It lives beside the code that reads and writes these
// tables so a domain stays a one-folder unit.
func Schema() string { return schemaSQL }

// DB is the subset of a database handle this store needs.
//
// Declaring the dependency as an interface rather than a *sql.DB is what keeps
// the scenario eligible for per-request routing: production passes
// *database.RoutedDB, which routes each request to the right pool, while tests
// pass a plain *sql.DB. Capturing the concrete pool type here would pin every
// caller to a single pool.
type DB interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// SQLiteStore persists the operator's durable decisions and keeps transient
// request state in memory.
//
// The split is deliberate rather than an unfinished migration:
//
//   - The active POLICY is an operator decision. It must survive a restart.
//     While it lived only in memory, every profile change silently reverted to
//     the shipped default on the next restart — indistinguishable from cleanup
//     never having been enabled at all, which is the exact class of bug that
//     let a disk fill while three safeguards reported healthy.
//   - The AUDIT trail is evidence. A record of what was deleted is worthless if
//     it disappears with the process that deleted it.
//   - PLANS and APPLY reports are transient. A plan is a measurement of the
//     filesystem at one instant, consumed within seconds by the apply that
//     follows it, and meaningless after a restart because the filesystem has
//     moved on. They are also large: one plan on this project's development
//     host serialised to 6.9 MB. Persisting those in a scenario whose purpose
//     is reclaiming disk space would be a self-defeating design.
type SQLiteStore struct {
	db DB

	// Transient state, same semantics as MemoryStore.
	mu      sync.Mutex
	plans   map[string]Plan
	applies map[string]ApplyReport
}

// NewSQLiteStore builds a store backed by db for policy and audit.
func NewSQLiteStore(db DB) *SQLiteStore {
	// Existing installations predate the typed byte column. SQLite's additive
	// migration is idempotent; the duplicate-column error is intentionally
	// ignored so construction remains compatible with fresh schema bootstrap.
	if db != nil {
		_, _ = db.ExecContext(context.Background(), `ALTER TABLE cleanup_audit ADD COLUMN bytes_reclaimed INTEGER NOT NULL DEFAULT 0`)
		_, _ = db.ExecContext(context.Background(), `ALTER TABLE recovery_actions ADD COLUMN authority TEXT NOT NULL DEFAULT 'class'`)
	}
	return &SQLiteStore{
		db:      db,
		plans:   map[string]Plan{},
		applies: map[string]ApplyReport{},
	}
}

// Compile-time guarantee that SQLiteStore satisfies the orchestrator contract.
var _ Store = (*SQLiteStore)(nil)

// SavePolicy replaces the single active policy row.
//
// There is exactly one active policy, so the row is keyed by a constant. Using
// a fixed key rather than appending versions means a reader can never pick the
// wrong one, and the policy's own Version field still records which profile
// generation is live.
func (s *SQLiteStore) SavePolicy(ctx context.Context, policy Policy) error {
	payload, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("marshal cleanup policy: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO cleanup_policy (id, payload, updated_at) VALUES (1, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET payload = excluded.payload, updated_at = excluded.updated_at`,
		string(payload), time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("save cleanup policy: %w", err)
	}
	return nil
}

// CurrentPolicy returns the stored policy, reporting false when none has been
// saved yet so the caller applies its default profile.
func (s *SQLiteStore) CurrentPolicy(ctx context.Context) (Policy, bool, error) {
	var payload string
	err := s.db.QueryRowContext(ctx, `SELECT payload FROM cleanup_policy WHERE id = 1`).Scan(&payload)
	if err == sql.ErrNoRows {
		return Policy{}, false, nil
	}
	if err != nil {
		return Policy{}, false, fmt.Errorf("read cleanup policy: %w", err)
	}

	var policy Policy
	if err := json.Unmarshal([]byte(payload), &policy); err != nil {
		// A policy row that cannot be decoded must not be treated as "no
		// policy": silently falling back to the default would quietly change
		// what gets deleted. Report it and let the caller decide.
		return Policy{}, false, fmt.Errorf("decode stored cleanup policy: %w", err)
	}
	return policy, true, nil
}

func (s *SQLiteStore) SavePlan(_ context.Context, plan Plan) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.plans[plan.ID] = plan
	return nil
}

func (s *SQLiteStore) GetPlan(_ context.Context, id string) (Plan, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	plan, ok := s.plans[id]
	return plan, ok, nil
}

func (s *SQLiteStore) SaveApply(_ context.Context, report ApplyReport) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.applies[report.IdempotencyKey] = report
	return nil
}

func (s *SQLiteStore) ApplyByKey(_ context.Context, key string) (ApplyReport, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	report, ok := s.applies[key]
	return report, ok, nil
}

// AddAudit appends an audit event.
func (s *SQLiteStore) AddAudit(ctx context.Context, event AuditEvent) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO cleanup_audit (id, occurred_at, type, plan_id, provider_id, idempotency_key, message, bytes_reclaimed, redacted)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO NOTHING`,
		event.ID, event.Time.UTC().Format(time.RFC3339Nano), event.Type,
		event.PlanID, event.ProviderID, event.IdempotencyKey, event.Message, event.ReclaimedBytes, event.Redacted,
	)
	if err != nil {
		return fmt.Errorf("append cleanup audit event: %w", err)
	}
	return nil
}

// ListAudit returns audit events oldest first, matching MemoryStore's order.
func (s *SQLiteStore) ListAudit(ctx context.Context) ([]AuditEvent, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, occurred_at, type, plan_id, provider_id, idempotency_key, message, bytes_reclaimed, redacted
		 FROM cleanup_audit ORDER BY occurred_at, id`)
	if err != nil {
		return nil, fmt.Errorf("list cleanup audit events: %w", err)
	}
	defer rows.Close()

	var out []AuditEvent
	for rows.Next() {
		var (
			event    AuditEvent
			occurred string
			redacted int
			planID   sql.NullString
			provider sql.NullString
			idemKey  sql.NullString
			message  sql.NullString
			bytes    int64
		)
		if err := rows.Scan(&event.ID, &occurred, &event.Type, &planID, &provider, &idemKey, &message, &bytes, &redacted); err != nil {
			return nil, fmt.Errorf("scan cleanup audit event: %w", err)
		}
		event.Time, _ = time.Parse(time.RFC3339Nano, occurred)
		event.PlanID = planID.String
		event.ProviderID = provider.String
		event.IdempotencyKey = idemKey.String
		event.Message = message.String
		event.ReclaimedBytes = bytes
		event.Redacted = redacted != 0
		out = append(out, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cleanup audit events: %w", err)
	}
	return out, nil
}

// SaveRecoveryRun persists the server-owned recovery lifecycle. The completion
// channel is process-local and intentionally omitted; a restarted server can
// still report the last durable terminal state through the ledger.
func (s *SQLiteStore) SaveRecoveryRun(ctx context.Context, run RecoveryRun) error {
	var completed any
	if !run.CompletedAt.IsZero() {
		completed = run.CompletedAt.UTC().Format(time.RFC3339Nano)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO recovery_runs (id, started_at, completed_at, trigger, mount, target_free_bytes, reclaimed_bytes, result, stopped_because)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET completed_at=excluded.completed_at, target_free_bytes=excluded.target_free_bytes, reclaimed_bytes=excluded.reclaimed_bytes, result=excluded.result, stopped_because=excluded.stopped_because`,
		run.ID, run.StartedAt.UTC().Format(time.RFC3339Nano), completed, run.Trigger, run.Partition, run.TargetFreeBytes, run.ReclaimedBytes, run.Status, run.StoppedBecause)
	if err != nil {
		return fmt.Errorf("save recovery run: %w", err)
	}
	return nil
}

// SaveRecoveryAction records one bounded provider application. Byte accounting
// is kept separate from prose audit messages so readers can aggregate it.
func (s *SQLiteStore) SaveRecoveryAction(ctx context.Context, runID, providerID, rung string, reclaimed, before, after int64, result cleanup.ApplyResult) error {
	return s.SaveRecoveryActionMetrics(ctx, runID, providerID, rung, reclaimed, before, after, len(result.AppliedItems), 0, result)
}

// SaveRecoveryActionMetrics records one bounded provider application with the
// measured item count and apply duration needed to audit recovery throughput.
func (s *SQLiteStore) SaveRecoveryActionMetrics(ctx context.Context, runID, providerID, rung string, reclaimed, before, after int64, filesRemoved int, duration time.Duration, result cleanup.ApplyResult) error {
	authority := "class"
	if rung == "R2" {
		authority = "owner_budget"
	} else if rung == "R3" {
		authority = "standing_approval"
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO recovery_actions (id, run_id, occurred_at, provider_id, rung, authority, bytes_reclaimed, files_removed, duration_ms, free_before, free_after, result)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		fmt.Sprintf("%s|%s|%d", runID, providerID, time.Now().UnixNano()), runID, time.Now().UTC().Format(time.RFC3339Nano), providerID, rung,
		authority, reclaimed, filesRemoved, duration.Milliseconds(), before, after, "applied")
	if err != nil {
		return fmt.Errorf("save recovery action: %w", err)
	}
	return nil
}

// ListRecoveryRuns reads terminal and in-flight runs from the durable ledger,
// allowing history to survive an API restart.
func (s *SQLiteStore) ListRecoveryRuns(ctx context.Context, limit int) ([]RecoveryRun, error) {
	query := `SELECT id, started_at, completed_at, trigger, mount, target_free_bytes, reclaimed_bytes, result, stopped_because FROM recovery_runs ORDER BY started_at DESC`
	args := []any{}
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list recovery runs: %w", err)
	}
	defer rows.Close()
	var out []RecoveryRun
	for rows.Next() {
		var run RecoveryRun
		var started, completed, stopped sql.NullString
		if err := rows.Scan(&run.ID, &started, &completed, &run.Trigger, &run.Partition, &run.TargetFreeBytes, &run.ReclaimedBytes, &run.Status, &stopped); err != nil {
			return nil, fmt.Errorf("scan recovery run: %w", err)
		}
		run.StartedAt, _ = time.Parse(time.RFC3339Nano, started.String)
		if completed.Valid {
			run.CompletedAt, _ = time.Parse(time.RFC3339Nano, completed.String)
		}
		run.StoppedBecause = stopped.String
		run.Reason = stopped.String
		out = append(out, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recovery runs: %w", err)
	}
	return out, nil
}

func (s *SQLiteStore) ListWriterSnapshots(ctx context.Context, limit int) ([]WriterSnapshot, error) {
	// This is the operational "top writers" view, not the historical ledger:
	// rank only the newest observation for each root so a cooled writer cannot
	// remain an apparently active H6 alarm forever. The underlying table keeps
	// every bounded snapshot for growth analysis.
	query := `SELECT id, observed_at, root, mount, bytes, delta_bytes, delta_hours, bytes_per_hour, partial, hot
		FROM (SELECT w.*, ROW_NUMBER() OVER (PARTITION BY root ORDER BY observed_at DESC, id DESC) AS root_rank
		      FROM writer_snapshots w
		      WHERE observed_at >= ?)
		WHERE root_rank = 1
		ORDER BY bytes_per_hour DESC, observed_at DESC`
	args := []any{time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339Nano)}
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list writer snapshots: %w", err)
	}
	defer rows.Close()
	var out []WriterSnapshot
	for rows.Next() {
		var snapshot WriterSnapshot
		var observed string
		var partial, hot int
		if err := rows.Scan(&snapshot.ID, &observed, &snapshot.Root, &snapshot.Mount, &snapshot.Bytes, &snapshot.DeltaBytes, &snapshot.DeltaHours, &snapshot.BytesPerHour, &partial, &hot); err != nil {
			return nil, fmt.Errorf("scan writer snapshot: %w", err)
		}
		snapshot.ObservedAt, _ = time.Parse(time.RFC3339Nano, observed)
		snapshot.Partial, snapshot.Hot = partial != 0, hot != 0
		out = append(out, snapshot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate writer snapshots: %w", err)
	}
	return out, nil
}

// SaveWriterSnapshot stores the sender's bounded growth observation without
// requiring the receiver to walk the filesystem on the pressure path.
func (s *SQLiteStore) SaveWriterSnapshot(ctx context.Context, id, sampledAt, mount, root string, bytes, deltaBytes int64, deltaHours float64, rate int64, partial, hot bool) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO writer_snapshots (id, observed_at, root, mount, bytes, delta_bytes, delta_hours, bytes_per_hour, partial, hot)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, sampledAt, root, mount, bytes, deltaBytes, deltaHours, rate, boolInt(partial), boolInt(hot))
	if err != nil {
		return fmt.Errorf("save writer snapshot: %w", err)
	}
	// Keep the attribution ledger bounded. This is indexed and runs after the
	// write, so an old evidence backlog cannot delay the pressure sender.
	_, _ = s.db.ExecContext(ctx, `DELETE FROM writer_snapshots WHERE observed_at < ?`, time.Now().UTC().Add(-30*24*time.Hour).Format(time.RFC3339Nano))
	return nil
}

// MarkWritersCooled reconciles the latest pressure report's complete hot-root
// set with the durable operational view. A sender omits cooled roots from
// HotWriters, so retaining their last hot row would make the read surface
// report a stale alarm indefinitely.
func (s *SQLiteStore) MarkWritersCooled(ctx context.Context, sampledAt string, activeRoots map[string]struct{}) error {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, observed_at, root, mount, bytes, delta_bytes, delta_hours, bytes_per_hour, partial
		FROM (
			SELECT w.*, ROW_NUMBER() OVER (PARTITION BY root ORDER BY observed_at DESC, id DESC) AS root_rank
			FROM writer_snapshots w
			WHERE hot = 1
		)
		WHERE root_rank = 1`)
	if err != nil {
		return fmt.Errorf("list hot writer snapshots: %w", err)
	}
	var hotSnapshots []WriterSnapshot
	for rows.Next() {
		var snapshot WriterSnapshot
		var observed string
		var partial int
		if err := rows.Scan(&snapshot.ID, &observed, &snapshot.Root, &snapshot.Mount, &snapshot.Bytes, &snapshot.DeltaBytes, &snapshot.DeltaHours, &snapshot.BytesPerHour, &partial); err != nil {
			return fmt.Errorf("scan hot writer snapshot: %w", err)
		}
		if _, stillHot := activeRoots[snapshot.Root]; stillHot {
			continue
		}
		snapshot.Partial = partial != 0
		hotSnapshots = append(hotSnapshots, snapshot)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate hot writer snapshots: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close hot writer snapshots: %w", err)
	}
	for _, snapshot := range hotSnapshots {
		if err := s.SaveWriterSnapshot(ctx, "cooled|"+snapshot.Root+"|"+sampledAt, sampledAt, snapshot.Mount, snapshot.Root, snapshot.Bytes, snapshot.DeltaBytes, snapshot.DeltaHours, snapshot.BytesPerHour, snapshot.Partial, false); err != nil {
			return err
		}
	}
	return nil
}

// PruneRecoveryLedger removes only derived evidence. It is intentionally
// separate from provider cleanup so retention can bound storage-manager's own
// tables without ever touching the source filesystem or durable owner data.
func (s *SQLiteStore) PruneRecoveryLedger(ctx context.Context, now time.Time) error {
	if s == nil || s.db == nil {
		return nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM recovery_actions WHERE run_id IN (SELECT id FROM recovery_runs WHERE started_at < ?)`, now.Add(-90*24*time.Hour).Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("prune recovery actions: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM recovery_runs WHERE started_at < ?`, now.Add(-90*24*time.Hour).Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("prune recovery runs: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM writer_snapshots WHERE observed_at < ?`, now.Add(-30*24*time.Hour).Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("prune writer snapshots: %w", err)
	}
	return nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
