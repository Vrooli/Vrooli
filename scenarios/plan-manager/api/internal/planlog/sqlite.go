package planlog

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/vrooli/api-core/provenance"
	"plan-manager/internal/clock"
	planmodel "plan-manager/internal/planmodel"
)

// logTimeFormat matches the rest of the scenario (RFC3339Nano sorts
// lexicographically in time order for a fixed zone).
const logTimeFormat = time.RFC3339Nano

// SQLExecutor is the narrow database surface the log repository depends on.
// Declared at the consumer per seam-discovery: both *sql.DB (tests, via
// testutil/db.NewSQLite) and *database.RoutedDB (production) satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db    SQLExecutor
	clock clock.Clock
}

// NewSQLiteRepository constructs the production log Repository.
func NewSQLiteRepository(db SQLExecutor, clk clock.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

var _ Repository = (*sqliteRepository)(nil)

const (
	upsertEntrySQL = `
INSERT INTO log_entries (
  id, type, plan_id, execution_id, phase_id, title, detail, severity, triage,
  sync_status, downstream, bug_payload, record_payload, capture, source_command, evidence, attribution_run_id, verification_status, harness_session_id, harness_kind,
  idempotency_key, supersedes_id, promoted_from_id, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  type=excluded.type,
  plan_id=excluded.plan_id,
  execution_id=excluded.execution_id,
  phase_id=excluded.phase_id,
  title=excluded.title,
  detail=excluded.detail,
  severity=excluded.severity,
  triage=excluded.triage,
  sync_status=excluded.sync_status,
  downstream=excluded.downstream,
  bug_payload=excluded.bug_payload,
  record_payload=excluded.record_payload,
  capture=excluded.capture,
  source_command=excluded.source_command,
  evidence=excluded.evidence,
  attribution_run_id=excluded.attribution_run_id,
  verification_status=excluded.verification_status,
  harness_session_id=excluded.harness_session_id,
  harness_kind=excluded.harness_kind,
  idempotency_key=excluded.idempotency_key,
  supersedes_id=excluded.supersedes_id,
  promoted_from_id=excluded.promoted_from_id,
  updated_at=excluded.updated_at`

	entryColumns = `
SELECT id, type, plan_id, execution_id, phase_id, title, detail, severity, triage,
       sync_status, downstream, bug_payload, record_payload, capture, source_command, evidence, attribution_run_id, verification_status, harness_session_id, harness_kind,
       idempotency_key, supersedes_id, promoted_from_id, created_at, updated_at
FROM log_entries`

	getEntrySQL = entryColumns + ` WHERE id = ? LIMIT 1`

	findByIdempotencySQL = entryColumns + ` WHERE plan_id = ? AND idempotency_key = ? LIMIT 1`
)

func (r *sqliteRepository) SaveEntry(ctx context.Context, e Entry) error {
	_, _, _, status, runID, _ := provenance.FromContext(ctx).WriteFields()
	sessionID, sessionKind := provenance.FromContext(ctx).ObservationFields()
	e.VerificationStatus = status
	e.HarnessSessionID, e.HarnessKind = sessionID, sessionKind
	if status == provenance.VerificationVerified {
		e.AttributionRunID = runID
	} else {
		e.AttributionRunID = ""
	}
	created := e.CreatedAt
	if created == "" {
		created = r.now()
	}
	updated := e.UpdatedAt
	if updated == "" {
		updated = r.now()
	}
	downstream, err := json.Marshal(e.Downstream)
	if err != nil {
		return fmt.Errorf("marshal downstream ref %q: %w", e.ID, err)
	}
	bug, err := json.Marshal(e.Bug)
	if err != nil {
		return fmt.Errorf("marshal bug payload %q: %w", e.ID, err)
	}
	record, err := json.Marshal(e.Record)
	if err != nil {
		return fmt.Errorf("marshal record payload %q: %w", e.ID, err)
	}
	capture, err := json.Marshal(e.Capture)
	if err != nil {
		return fmt.Errorf("marshal capture disposition %q: %w", e.ID, err)
	}
	evidence, err := json.Marshal(nonNilStrings(e.Evidence))
	if err != nil {
		return fmt.Errorf("marshal evidence %q: %w", e.ID, err)
	}
	if _, err := r.db.ExecContext(ctx, upsertEntrySQL,
		e.ID, string(e.Type), e.PlanID, e.ExecutionID, e.PhaseID, e.Title, e.Detail,
		string(e.Severity), string(e.Triage), string(e.SyncStatus), string(downstream), string(bug), string(record), string(capture),
		e.SourceCommand, string(evidence), e.AttributionRunID, e.VerificationStatus, e.HarnessSessionID, e.HarnessKind, e.IdempotencyKey,
		e.SupersedesID, e.PromotedFromID, created, updated,
	); err != nil {
		return fmt.Errorf("upsert log entry %q: %w", e.ID, err)
	}
	return nil
}

func (r *sqliteRepository) GetEntry(ctx context.Context, id string) (Entry, bool, error) {
	e, err := scanEntry(r.db.QueryRowContext(ctx, getEntrySQL, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Entry{}, false, nil
	}
	if err != nil {
		return Entry{}, false, fmt.Errorf("get log entry %q: %w", id, err)
	}
	return e, true, nil
}

func (r *sqliteRepository) FindByIdempotencyKey(ctx context.Context, planID, key string) (Entry, bool, error) {
	if key == "" {
		return Entry{}, false, nil
	}
	e, err := scanEntry(r.db.QueryRowContext(ctx, findByIdempotencySQL, planID, key))
	if errors.Is(err, sql.ErrNoRows) {
		return Entry{}, false, nil
	}
	if err != nil {
		return Entry{}, false, fmt.Errorf("find log entry by idempotency key: %w", err)
	}
	return e, true, nil
}

func (r *sqliteRepository) ListEntries(ctx context.Context, f Filter) ([]Entry, error) {
	query := entryColumns
	var (
		clauses []string
		args    []any
	)
	if f.PlanID != "" {
		clauses = append(clauses, "plan_id = ?")
		args = append(args, f.PlanID)
	}
	if f.ExecutionID != "" {
		clauses = append(clauses, "execution_id = ?")
		args = append(args, f.ExecutionID)
	}
	if f.PhaseID != "" {
		clauses = append(clauses, "phase_id = ?")
		args = append(args, f.PhaseID)
	}
	if f.Type != "" {
		clauses = append(clauses, "type = ?")
		args = append(args, string(f.Type))
	}
	if f.Triage != "" {
		clauses = append(clauses, "triage = ?")
		args = append(args, string(f.Triage))
	}
	if f.SyncStatus != "" {
		clauses = append(clauses, "sync_status = ?")
		args = append(args, string(f.SyncStatus))
	}
	for i, c := range clauses {
		if i == 0 {
			query += " WHERE " + c
		} else {
			query += " AND " + c
		}
	}
	query += " ORDER BY created_at, id"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list log entries: %w", err)
	}
	defer rows.Close()
	out := make([]Entry, 0)
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("scan log entry: %w", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate log entries: %w", err)
	}
	return out, nil
}

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanEntry(s rowScanner) (Entry, error) {
	var (
		e          Entry
		typ        string
		severity   string
		triage     string
		syncStatus string
		downstream string
		bug        string
		record     string
		capture    string
		evidence   string
	)
	if err := s.Scan(
		&e.ID, &typ, &e.PlanID, &e.ExecutionID, &e.PhaseID, &e.Title, &e.Detail,
		&severity, &triage, &syncStatus, &downstream, &bug, &record, &capture, &e.SourceCommand, &evidence,
		&e.AttributionRunID, &e.VerificationStatus, &e.HarnessSessionID, &e.HarnessKind, &e.IdempotencyKey, &e.SupersedesID, &e.PromotedFromID,
		&e.CreatedAt, &e.UpdatedAt,
	); err != nil {
		return Entry{}, err
	}
	e.Type = planmodel.LogEntryType(typ)
	e.Severity = planmodel.LogSeverity(severity)
	e.Triage = planmodel.FindingTriage(triage)
	e.SyncStatus = planmodel.LogSyncStatus(syncStatus)
	if downstream != "" && downstream != "null" {
		if err := json.Unmarshal([]byte(downstream), &e.Downstream); err != nil {
			return Entry{}, fmt.Errorf("unmarshal downstream ref %q: %w", e.ID, err)
		}
	}
	if bug != "" && bug != "null" {
		if err := json.Unmarshal([]byte(bug), &e.Bug); err != nil {
			return Entry{}, fmt.Errorf("unmarshal bug payload %q: %w", e.ID, err)
		}
	}
	if record != "" && record != "null" {
		if err := json.Unmarshal([]byte(record), &e.Record); err != nil {
			return Entry{}, fmt.Errorf("unmarshal record payload %q: %w", e.ID, err)
		}
	}
	if capture != "" && capture != "null" {
		if err := json.Unmarshal([]byte(capture), &e.Capture); err != nil {
			return Entry{}, fmt.Errorf("unmarshal capture disposition %q: %w", e.ID, err)
		}
	}
	if evidence != "" && evidence != "null" {
		if err := json.Unmarshal([]byte(evidence), &e.Evidence); err != nil {
			return Entry{}, fmt.Errorf("unmarshal evidence %q: %w", e.ID, err)
		}
	}
	return e, nil
}

func (r *sqliteRepository) now() string { return r.clock.Now().UTC().Format(logTimeFormat) }

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
