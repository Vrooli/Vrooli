package execution

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	planmodel "plan-manager/internal/planmodel"

	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/api-core/schedule"
)

// execTimeFormat matches the rest of the scenario (RFC3339Nano sorts
// lexicographically in time order for a fixed zone).
const execTimeFormat = time.RFC3339Nano

// SQLExecutor is the narrow database surface the execution repository depends
// on. Declared at the consumer per seam-discovery: both *sql.DB (tests, via
// testutil/db.NewSQLite) and *database.RoutedDB (production) satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
}

// NewSQLiteRepository constructs the production execution Repository.
func NewSQLiteRepository(db SQLExecutor, clk schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

var _ Repository = (*sqliteRepository)(nil)

// txBeginner is the optional transaction capability of the underlying DB. Both
// *sql.DB (tests) and *database.RoutedDB (production) satisfy it; *sql.Tx does
// not, so a WithTx already inside a transaction safely runs inline.
type txBeginner interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// WithTx runs fn against a repository bound to one transaction so a multi-write
// operation commits atomically or rolls back as a unit. When the underlying DB
// does not expose BeginTx, fn runs against this repository directly — a safe,
// non-atomic fallback that never blocks the operation.
func (r *sqliteRepository) WithTx(ctx context.Context, fn func(Repository) error) error {
	beginner, ok := r.db.(txBeginner)
	if !ok {
		return fn(r)
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	if err := fn(&sqliteRepository{db: tx, clock: r.clock}); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// handoffDocument is the JSON payload stored in handoffs.document — the assembled
// snapshot that round-trips with the handoff row (the queryable columns carry id/
// execution/plan/completeness/resume/assembled_at). The log ledger snapshot
// (summary + captured entries) is read from the log domain at Complete time and
// stored here so the handoff is a durable point-in-time record.
type handoffDocument struct {
	LogSummary      planmodel.LogSummary     `json:"log_summary"`
	LogEntries      []planmodel.LogEntry     `json:"log_entries"`
	LastValidation  ValidationResult         `json:"last_validation"`
	HasValidation   bool                     `json:"has_validation"`
	Staleness       string                   `json:"staleness"`
	ProseHandoffRef string                   `json:"prose_handoff_ref"`
	ChangeBoundary  planmodel.ChangeBoundary `json:"change_boundary"`
}

// executionScopeDocument intentionally owns only execution-local policy. It
// round-trips alongside the execution row so schema evolution is additive for
// existing SQLite databases.
type executionScopeDocument struct {
	PhaseValidationGenerations map[string]int          `json:"phase_validation_generations"`
	ScopeAmendments            []ScopeAmendment        `json:"scope_amendments"`
	BoundaryExtensions         []BoundaryExtension     `json:"boundary_extensions,omitempty"`
	DegradedReason             string                  `json:"degraded_reason"`
	LifecycleState             ExecutionLifecycleState `json:"lifecycle_state,omitempty"`
	AbandonedReason            string                  `json:"abandoned_reason,omitempty"`
	AbandonedAt                string                  `json:"abandoned_at,omitempty"`
	AbandonedBy                string                  `json:"abandoned_by,omitempty"`
}

const (
	upsertExecutionSQL = `
	INSERT INTO executions (id, plan_id, run_id, verification_status, harness_session_id, harness_kind, current_phase_id, complete, started_at, updated_at, inputs_freshened_at, freshen_status, freshen_detail)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  plan_id=excluded.plan_id,
  run_id=excluded.run_id,
	verification_status=excluded.verification_status,
	harness_session_id=excluded.harness_session_id,
	harness_kind=excluded.harness_kind,
  current_phase_id=excluded.current_phase_id,
  complete=excluded.complete,
  updated_at=excluded.updated_at,
  inputs_freshened_at=excluded.inputs_freshened_at,
  freshen_status=excluded.freshen_status,
  freshen_detail=excluded.freshen_detail`

	getExecutionSQL = `
	SELECT id, plan_id, run_id, verification_status, harness_session_id, harness_kind, current_phase_id, complete, started_at, updated_at, inputs_freshened_at, freshen_status, freshen_detail
FROM executions WHERE id = ? LIMIT 1`

	latestExecutionForPlanSQL = `
	SELECT id, plan_id, run_id, verification_status, harness_session_id, harness_kind, current_phase_id, complete, started_at, updated_at, inputs_freshened_at, freshen_status, freshen_detail
FROM executions WHERE plan_id = ? ORDER BY updated_at DESC, started_at DESC, id DESC LIMIT 1`

	listExecutionsForPlanSQL = `
	SELECT id, plan_id, run_id, verification_status, harness_session_id, harness_kind, current_phase_id, complete, started_at, updated_at, inputs_freshened_at, freshen_status, freshen_detail
FROM executions WHERE plan_id = ? ORDER BY updated_at DESC, started_at DESC, id DESC`

	upsertBaselineSetSQL = `
INSERT INTO execution_baseline_sets (execution_id, document, updated_at)
VALUES (?, ?, ?)
ON CONFLICT(execution_id) DO UPDATE SET document=excluded.document, updated_at=excluded.updated_at`

	getBaselineSetSQL = `SELECT document FROM execution_baseline_sets WHERE execution_id = ? LIMIT 1`

	upsertScopeStateSQL = `
INSERT INTO execution_scope_states (execution_id, document, updated_at)
VALUES (?, ?, ?)
ON CONFLICT(execution_id) DO UPDATE SET document=excluded.document, updated_at=excluded.updated_at`

	getScopeStateSQL = `SELECT document FROM execution_scope_states WHERE execution_id = ? LIMIT 1`

	upsertHandoffSQL = `
INSERT INTO handoffs (id, execution_id, plan_id, completeness, resume_phase_id, document, assembled_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  plan_id=excluded.plan_id,
  completeness=excluded.completeness,
  resume_phase_id=excluded.resume_phase_id,
  document=excluded.document,
  assembled_at=excluded.assembled_at`

	getHandoffSQL = `
SELECT id, execution_id, plan_id, completeness, resume_phase_id, document, assembled_at
FROM handoffs WHERE execution_id = ? ORDER BY assembled_at DESC, id LIMIT 1`

	insertVelocitySQL = `
INSERT INTO velocity_points (id, plan_id, run_id, wall_time_seconds, tokens, iterations, completeness, recorded_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	listVelocitySQL = `
SELECT id, plan_id, run_id, wall_time_seconds, tokens, iterations, completeness, recorded_at
FROM velocity_points WHERE plan_id = ? ORDER BY recorded_at, id`
)

func (r *sqliteRepository) SaveExecution(ctx context.Context, e Execution) error {
	prov := provenance.FromContext(ctx)
	_, _, _, status, runID, _ := prov.WriteFields()
	sessionID, sessionKind := prov.ObservationFields()
	if status == provenance.VerificationVerified {
		e.RunID = runID
	} else {
		e.RunID = ""
	}
	e.VerificationStatus = status
	e.HarnessSessionID, e.HarnessKind = sessionID, sessionKind
	started := e.StartedAt
	if started == "" {
		started = r.now()
	}
	updated := e.UpdatedAt
	if updated == "" {
		updated = r.now()
	}
	if _, err := r.db.ExecContext(ctx, upsertExecutionSQL,
		e.ID, e.PlanID, e.RunID, e.VerificationStatus, e.HarnessSessionID, e.HarnessKind, e.CurrentPhaseID, boolToInt(e.Complete), started, updated,
		e.InputsFreshenedAt, e.FreshenStatus, e.FreshenDetail,
	); err != nil {
		return fmt.Errorf("upsert execution %q: %w", e.ID, err)
	}
	if e.BaselineSet.Name != "" || e.BaselineSet.LegacyAdoptionRequired {
		raw, err := json.Marshal(e.BaselineSet)
		if err != nil {
			return fmt.Errorf("marshal execution baseline set %q: %w", e.ID, err)
		}
		if _, err := r.db.ExecContext(ctx, upsertBaselineSetSQL, e.ID, string(raw), updated); err != nil {
			return fmt.Errorf("upsert execution baseline set %q: %w", e.ID, err)
		}
	}
	scopeRaw, err := json.Marshal(executionScopeDocument{
		PhaseValidationGenerations: e.PhaseValidationGenerations,
		ScopeAmendments:            e.ScopeAmendments,
		BoundaryExtensions:         e.BoundaryExtensions,
		DegradedReason:             e.DegradedReason,
		LifecycleState:             e.LifecycleState,
		AbandonedReason:            e.AbandonedReason,
		AbandonedAt:                e.AbandonedAt,
		AbandonedBy:                e.AbandonedBy,
	})
	if err != nil {
		return fmt.Errorf("marshal execution scope state %q: %w", e.ID, err)
	}
	if _, err := r.db.ExecContext(ctx, upsertScopeStateSQL, e.ID, string(scopeRaw), updated); err != nil {
		return fmt.Errorf("upsert execution scope state %q: %w", e.ID, err)
	}
	return nil
}

func (r *sqliteRepository) ListExecutionsForPlan(ctx context.Context, planID string) ([]Execution, error) {
	rows, err := r.db.QueryContext(ctx, listExecutionsForPlanSQL, planID)
	if err != nil {
		return nil, fmt.Errorf("list executions for plan %q: %w", planID, err)
	}
	defer rows.Close()
	var out []Execution
	for rows.Next() {
		var e Execution
		var complete int
		if err := rows.Scan(&e.ID, &e.PlanID, &e.RunID, &e.VerificationStatus, &e.HarnessSessionID, &e.HarnessKind, &e.CurrentPhaseID, &complete, &e.StartedAt, &e.UpdatedAt, &e.InputsFreshenedAt, &e.FreshenStatus, &e.FreshenDetail); err != nil {
			return nil, fmt.Errorf("scan execution for plan %q: %w", planID, err)
		}
		e.Complete = complete != 0
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate executions for plan %q: %w", planID, err)
	}
	for i := range out {
		if err := r.loadBaselineSet(ctx, &out[i]); err != nil {
			return nil, err
		}
		if err := r.loadScopeState(ctx, &out[i]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (r *sqliteRepository) GetExecution(ctx context.Context, id string) (Execution, bool, error) {
	var (
		e        Execution
		complete int
	)
	err := r.db.QueryRowContext(ctx, getExecutionSQL, id).Scan(
		&e.ID, &e.PlanID, &e.RunID, &e.VerificationStatus, &e.HarnessSessionID, &e.HarnessKind, &e.CurrentPhaseID, &complete, &e.StartedAt, &e.UpdatedAt,
		&e.InputsFreshenedAt, &e.FreshenStatus, &e.FreshenDetail,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Execution{}, false, nil
	}
	if err != nil {
		return Execution{}, false, fmt.Errorf("get execution %q: %w", id, err)
	}
	e.Complete = complete != 0
	if err := r.loadBaselineSet(ctx, &e); err != nil {
		return Execution{}, false, err
	}
	if err := r.loadScopeState(ctx, &e); err != nil {
		return Execution{}, false, err
	}
	return e, true, nil
}

func (r *sqliteRepository) LatestExecutionForPlan(ctx context.Context, planID string) (Execution, bool, error) {
	var (
		e        Execution
		complete int
	)
	err := r.db.QueryRowContext(ctx, latestExecutionForPlanSQL, planID).Scan(
		&e.ID, &e.PlanID, &e.RunID, &e.VerificationStatus, &e.HarnessSessionID, &e.HarnessKind, &e.CurrentPhaseID, &complete, &e.StartedAt, &e.UpdatedAt,
		&e.InputsFreshenedAt, &e.FreshenStatus, &e.FreshenDetail,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Execution{}, false, nil
	}
	if err != nil {
		return Execution{}, false, fmt.Errorf("latest execution for plan %q: %w", planID, err)
	}
	e.Complete = complete != 0
	if err := r.loadBaselineSet(ctx, &e); err != nil {
		return Execution{}, false, err
	}
	if err := r.loadScopeState(ctx, &e); err != nil {
		return Execution{}, false, err
	}
	return e, true, nil
}

func (r *sqliteRepository) loadBaselineSet(ctx context.Context, e *Execution) error {
	var raw string
	err := r.db.QueryRowContext(ctx, getBaselineSetSQL, e.ID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil // legacy execution: no baseline-set checkpoint
	}
	if err != nil {
		return fmt.Errorf("get execution baseline set %q: %w", e.ID, err)
	}
	if err := json.Unmarshal([]byte(raw), &e.BaselineSet); err != nil {
		return fmt.Errorf("unmarshal execution baseline set %q: %w", e.ID, err)
	}
	return nil
}

func (r *sqliteRepository) loadScopeState(ctx context.Context, e *Execution) error {
	var raw string
	err := r.db.QueryRowContext(ctx, getScopeStateSQL, e.ID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil // legacy execution: scope defaults to the authored plan
	}
	if err != nil {
		return fmt.Errorf("get execution scope state %q: %w", e.ID, err)
	}
	var state executionScopeDocument
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return fmt.Errorf("unmarshal execution scope state %q: %w", e.ID, err)
	}
	e.PhaseValidationGenerations = state.PhaseValidationGenerations
	e.ScopeAmendments = state.ScopeAmendments
	e.BoundaryExtensions = state.BoundaryExtensions
	e.DegradedReason = state.DegradedReason
	e.LifecycleState = state.LifecycleState
	e.AbandonedReason = state.AbandonedReason
	e.AbandonedAt = state.AbandonedAt
	e.AbandonedBy = state.AbandonedBy
	return nil
}

func (r *sqliteRepository) SaveHandoff(ctx context.Context, h Handoff) error {
	doc := handoffDocument{
		LogSummary:      h.LogSummary,
		LogEntries:      h.LogEntries,
		LastValidation:  h.LastValidation,
		HasValidation:   h.HasValidation,
		Staleness:       string(h.Staleness),
		ProseHandoffRef: h.ProseHandoffRef,
		ChangeBoundary:  h.ChangeBoundary,
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal handoff document %q: %w", h.ID, err)
	}
	assembled := h.AssembledAt
	if assembled == "" {
		assembled = r.now()
	}
	if _, err := r.db.ExecContext(ctx, upsertHandoffSQL,
		h.ID, h.ExecutionID, h.PlanID, string(h.Completeness), h.ResumePhaseID, string(raw), assembled,
	); err != nil {
		return fmt.Errorf("upsert handoff %q: %w", h.ID, err)
	}
	return nil
}

func (r *sqliteRepository) GetHandoff(ctx context.Context, executionID string) (Handoff, bool, error) {
	var (
		h            Handoff
		completeness string
		document     string
	)
	err := r.db.QueryRowContext(ctx, getHandoffSQL, executionID).Scan(
		&h.ID, &h.ExecutionID, &h.PlanID, &completeness, &h.ResumePhaseID, &document, &h.AssembledAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Handoff{}, false, nil
	}
	if err != nil {
		return Handoff{}, false, fmt.Errorf("get handoff for execution %q: %w", executionID, err)
	}
	h.Completeness = Completeness(completeness)
	var doc handoffDocument
	if err := json.Unmarshal([]byte(document), &doc); err != nil {
		return Handoff{}, false, fmt.Errorf("unmarshal handoff document %q: %w", h.ID, err)
	}
	h.LogSummary = doc.LogSummary
	h.LogEntries = doc.LogEntries
	h.LastValidation = doc.LastValidation
	h.HasValidation = doc.HasValidation
	h.Staleness = stalenessFromString(doc.Staleness)
	h.ProseHandoffRef = doc.ProseHandoffRef
	h.ChangeBoundary = doc.ChangeBoundary
	return h, true, nil
}

func (r *sqliteRepository) SaveVelocity(ctx context.Context, v VelocityPoint) error {
	recorded := v.RecordedAt
	if recorded == "" {
		recorded = r.now()
	}
	if _, err := r.db.ExecContext(ctx, insertVelocitySQL,
		v.ID, v.PlanID, v.RunID, v.WallTimeSeconds, v.Tokens, v.Iterations, string(v.Completeness), recorded,
	); err != nil {
		return fmt.Errorf("insert velocity point %q: %w", v.ID, err)
	}
	return nil
}

func (r *sqliteRepository) ListVelocity(ctx context.Context, planID string) ([]VelocityPoint, error) {
	rows, err := r.db.QueryContext(ctx, listVelocitySQL, planID)
	if err != nil {
		return nil, fmt.Errorf("list velocity: %w", err)
	}
	defer rows.Close()
	out := make([]VelocityPoint, 0)
	for rows.Next() {
		var (
			v            VelocityPoint
			completeness string
		)
		if err := rows.Scan(&v.ID, &v.PlanID, &v.RunID, &v.WallTimeSeconds, &v.Tokens, &v.Iterations, &completeness, &v.RecordedAt); err != nil {
			return nil, fmt.Errorf("scan velocity point: %w", err)
		}
		v.Completeness = Completeness(completeness)
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate velocity: %w", err)
	}
	return out, nil
}

func (r *sqliteRepository) now() string { return r.clock.Now().UTC().Format(execTimeFormat) }

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// stalenessFromString rebuilds the plans StalenessTier from its stored string.
// The empty string is the unknown/degraded tier (planmodel.StalenessUnknown).
func stalenessFromString(s string) planmodel.StalenessTier {
	return planmodel.StalenessTier(s)
}
