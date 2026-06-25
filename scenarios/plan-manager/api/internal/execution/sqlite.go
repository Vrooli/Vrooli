package execution

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"plan-manager/internal/clock"
	internalplans "plan-manager/internal/plans"
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
	clock clock.Clock
}

// NewSQLiteRepository constructs the production execution Repository.
func NewSQLiteRepository(db SQLExecutor, clk clock.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

var _ Repository = (*sqliteRepository)(nil)

// handoffDocument is the JSON payload stored in handoffs.document — the assembled
// snapshot that round-trips with the handoff row (the queryable columns carry id/
// execution/plan/completeness/resume/assembled_at).
type handoffDocument struct {
	Decisions         []Decision       `json:"decisions"`
	CandidateFindings []Finding        `json:"candidate_findings"`
	LastValidation    ValidationResult `json:"last_validation"`
	HasValidation     bool             `json:"has_validation"`
	Staleness         string           `json:"staleness"`
	ProseHandoffRef   string           `json:"prose_handoff_ref"`
}

const (
	upsertExecutionSQL = `
INSERT INTO executions (id, plan_id, run_id, current_phase_id, complete, started_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  plan_id=excluded.plan_id,
  run_id=excluded.run_id,
  current_phase_id=excluded.current_phase_id,
  complete=excluded.complete,
  updated_at=excluded.updated_at`

	getExecutionSQL = `
SELECT id, plan_id, run_id, current_phase_id, complete, started_at, updated_at
FROM executions WHERE id = ? LIMIT 1`

	insertDecisionSQL = `
INSERT INTO decisions (id, execution_id, phase_id, summary, detail, recorded_at)
VALUES (?, ?, ?, ?, ?, ?)`

	listDecisionsSQL = `
SELECT id, execution_id, phase_id, summary, detail, recorded_at
FROM decisions WHERE execution_id = ? ORDER BY recorded_at, id`

	upsertFindingSQL = `
INSERT INTO findings (id, execution_id, phase_id, title, detail, triage, attribution_run_id, recorded_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  triage=excluded.triage`

	getFindingSQL = `
SELECT id, execution_id, phase_id, title, detail, triage, attribution_run_id, recorded_at
FROM findings WHERE id = ? LIMIT 1`

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
	started := e.StartedAt
	if started == "" {
		started = r.now()
	}
	updated := e.UpdatedAt
	if updated == "" {
		updated = r.now()
	}
	if _, err := r.db.ExecContext(ctx, upsertExecutionSQL,
		e.ID, e.PlanID, e.RunID, e.CurrentPhaseID, boolToInt(e.Complete), started, updated,
	); err != nil {
		return fmt.Errorf("upsert execution %q: %w", e.ID, err)
	}
	return nil
}

func (r *sqliteRepository) GetExecution(ctx context.Context, id string) (Execution, bool, error) {
	var (
		e        Execution
		complete int
	)
	err := r.db.QueryRowContext(ctx, getExecutionSQL, id).Scan(
		&e.ID, &e.PlanID, &e.RunID, &e.CurrentPhaseID, &complete, &e.StartedAt, &e.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Execution{}, false, nil
	}
	if err != nil {
		return Execution{}, false, fmt.Errorf("get execution %q: %w", id, err)
	}
	e.Complete = complete != 0
	return e, true, nil
}

func (r *sqliteRepository) SaveDecision(ctx context.Context, d Decision) error {
	recorded := d.RecordedAt
	if recorded == "" {
		recorded = r.now()
	}
	if _, err := r.db.ExecContext(ctx, insertDecisionSQL,
		d.ID, d.ExecutionID, d.PhaseID, d.Summary, d.Detail, recorded,
	); err != nil {
		return fmt.Errorf("insert decision %q: %w", d.ID, err)
	}
	return nil
}

func (r *sqliteRepository) ListDecisions(ctx context.Context, executionID string) ([]Decision, error) {
	rows, err := r.db.QueryContext(ctx, listDecisionsSQL, executionID)
	if err != nil {
		return nil, fmt.Errorf("list decisions: %w", err)
	}
	defer rows.Close()
	out := make([]Decision, 0)
	for rows.Next() {
		var d Decision
		if err := rows.Scan(&d.ID, &d.ExecutionID, &d.PhaseID, &d.Summary, &d.Detail, &d.RecordedAt); err != nil {
			return nil, fmt.Errorf("scan decision: %w", err)
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate decisions: %w", err)
	}
	return out, nil
}

func (r *sqliteRepository) SaveFinding(ctx context.Context, f Finding) error {
	recorded := f.RecordedAt
	if recorded == "" {
		recorded = r.now()
	}
	triage := f.Triage
	if triage == "" {
		triage = TriageCandidate
	}
	if _, err := r.db.ExecContext(ctx, upsertFindingSQL,
		f.ID, f.ExecutionID, f.PhaseID, f.Title, f.Detail, string(triage), f.AttributionRunID, recorded,
	); err != nil {
		return fmt.Errorf("upsert finding %q: %w", f.ID, err)
	}
	return nil
}

func (r *sqliteRepository) GetFinding(ctx context.Context, id string) (Finding, bool, error) {
	f, err := scanFinding(r.db.QueryRowContext(ctx, getFindingSQL, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Finding{}, false, nil
	}
	if err != nil {
		return Finding{}, false, fmt.Errorf("get finding %q: %w", id, err)
	}
	return f, true, nil
}

func (r *sqliteRepository) ListFindings(ctx context.Context, executionID string, triage FindingTriage) ([]Finding, error) {
	// Build the query dynamically so an empty scope/triage means "all". Keeps the
	// SQL parameterized — no string interpolation of values.
	query := `
SELECT id, execution_id, phase_id, title, detail, triage, attribution_run_id, recorded_at
FROM findings`
	var (
		clauses []string
		args    []any
	)
	if executionID != "" {
		clauses = append(clauses, "execution_id = ?")
		args = append(args, executionID)
	}
	if triage != "" {
		clauses = append(clauses, "triage = ?")
		args = append(args, string(triage))
	}
	for i, c := range clauses {
		if i == 0 {
			query += " WHERE " + c
		} else {
			query += " AND " + c
		}
	}
	query += " ORDER BY recorded_at, id"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list findings: %w", err)
	}
	defer rows.Close()
	out := make([]Finding, 0)
	for rows.Next() {
		f, err := scanFinding(rows)
		if err != nil {
			return nil, fmt.Errorf("scan finding: %w", err)
		}
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate findings: %w", err)
	}
	return out, nil
}

func (r *sqliteRepository) SaveHandoff(ctx context.Context, h Handoff) error {
	doc := handoffDocument{
		Decisions:         h.Decisions,
		CandidateFindings: h.CandidateFindings,
		LastValidation:    h.LastValidation,
		HasValidation:     h.HasValidation,
		Staleness:         string(h.Staleness),
		ProseHandoffRef:   h.ProseHandoffRef,
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
	h.Decisions = doc.Decisions
	h.CandidateFindings = doc.CandidateFindings
	h.LastValidation = doc.LastValidation
	h.HasValidation = doc.HasValidation
	h.Staleness = stalenessFromString(doc.Staleness)
	h.ProseHandoffRef = doc.ProseHandoffRef
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

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanFinding(s rowScanner) (Finding, error) {
	var (
		f      Finding
		triage string
	)
	if err := s.Scan(&f.ID, &f.ExecutionID, &f.PhaseID, &f.Title, &f.Detail, &triage, &f.AttributionRunID, &f.RecordedAt); err != nil {
		return Finding{}, err
	}
	f.Triage = FindingTriage(triage)
	return f, nil
}

func (r *sqliteRepository) now() string { return r.clock.Now().UTC().Format(execTimeFormat) }

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// stalenessFromString rebuilds the plans StalenessTier from its stored string.
// The empty string is the unknown/degraded tier (internalplans.StalenessUnknown).
func stalenessFromString(s string) internalplans.StalenessTier {
	return internalplans.StalenessTier(s)
}
