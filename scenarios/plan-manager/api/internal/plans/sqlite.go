package plans

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"plan-manager/internal/clock"
)

// planTimeFormat matches the rest of the scenario (RFC3339Nano sorts
// lexicographically in time order for a fixed zone).
const planTimeFormat = time.RFC3339Nano

// SQLExecutor is the narrow database surface the plans repository depends on.
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

// NewSQLiteRepository constructs the production plans Repository.
func NewSQLiteRepository(db SQLExecutor, clk clock.Clock) Repository {
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

// planDocument is the JSON payload stored in plans.document — every structured
// field that isn't a first-class queryable column. Phases and references live
// here because they round-trip with the plan and are never queried across plans.
type planDocument struct {
	Purpose          string                `json:"purpose"`
	Scope            string                `json:"scope"`
	Constraints      string                `json:"constraints"`
	NonGoals         string                `json:"non_goals"`
	DefinitionOfDone string                `json:"definition_of_done"`
	References       []Reference           `json:"references"`
	RegressionAnchor RegressionAnchor      `json:"regression_anchor"`
	Phases           []Phase               `json:"phases"`
	Supersedes       []string              `json:"supersedes"`
	SupersededBy     []string              `json:"superseded_by"`
	RelevantContext  []RelevantContextItem `json:"relevant_context"`
	// Professional plan structure (see docs/concepts/PLAN-MODEL.md). New fields
	// persist transparently because the whole document is one JSON blob.
	ProblemStatement        string            `json:"problem_statement,omitempty"`
	TargetOutcome           string            `json:"target_outcome,omitempty"`
	Assumptions             string            `json:"assumptions,omitempty"`
	TechnicalApproach       string            `json:"technical_approach,omitempty"`
	ValidationStrategy      string            `json:"validation_strategy,omitempty"`
	FinalValidationCommands []string          `json:"final_validation_commands,omitempty"`
	RisksHazards            string            `json:"risks_hazards,omitempty"`
	ProhibitedApproaches    string            `json:"prohibited_approaches,omitempty"`
	WorkPosture             WorkPosture       `json:"work_posture,omitempty"`
	WorkPostureSource       WorkPostureSource `json:"work_posture_source,omitempty"`
	WorkPostureDetail       string            `json:"work_posture_detail,omitempty"`
	ImportProvenance        *ImportProvenance `json:"import_provenance,omitempty"`
	PreservedLegacySections []LegacySection   `json:"preserved_legacy_sections,omitempty"`
}

const (
	upsertPlanSQL = `
INSERT INTO plans (id, slug, title, status, content_hash, document, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  slug=excluded.slug,
  title=excluded.title,
  status=excluded.status,
  content_hash=excluded.content_hash,
  document=excluded.document,
  updated_at=excluded.updated_at`

	getPlanSQL = `
SELECT id, slug, title, status, content_hash, document, created_at, updated_at
FROM plans WHERE id = ? OR slug = ? LIMIT 1`

	listPlansSQL = `
SELECT id, slug, title, status, content_hash, document, created_at, updated_at
FROM plans ORDER BY created_at DESC, id`

	upsertEdgeSQL = `
INSERT INTO plan_edges (from_plan_id, to_plan_id, kind)
VALUES (?, ?, ?)
ON CONFLICT(from_plan_id, to_plan_id, kind) DO NOTHING`

	listAllEdgesSQL = `
SELECT from_plan_id, to_plan_id, kind FROM plan_edges ORDER BY from_plan_id, to_plan_id, kind`

	listEdgesForPlanSQL = `
SELECT from_plan_id, to_plan_id, kind FROM plan_edges
WHERE from_plan_id = ? OR to_plan_id = ?
ORDER BY from_plan_id, to_plan_id, kind`
)

func (r *sqliteRepository) Save(ctx context.Context, p Plan) error {
	doc := planDocument{
		Purpose:          p.Purpose,
		Scope:            p.Scope,
		Constraints:      p.Constraints,
		NonGoals:         p.NonGoals,
		DefinitionOfDone: p.DefinitionOfDone,
		References:       p.References,
		RegressionAnchor: p.RegressionAnchor,
		Phases:           p.Phases,
		Supersedes:       p.Supersedes,
		SupersededBy:     p.SupersededBy,
		RelevantContext:  p.RelevantContext,

		ProblemStatement:        p.ProblemStatement,
		TargetOutcome:           p.TargetOutcome,
		Assumptions:             p.Assumptions,
		TechnicalApproach:       p.TechnicalApproach,
		ValidationStrategy:      p.ValidationStrategy,
		FinalValidationCommands: p.FinalValidationCommands,
		RisksHazards:            p.RisksHazards,
		ProhibitedApproaches:    p.ProhibitedApproaches,
		WorkPosture:             p.WorkPosture,
		WorkPostureSource:       p.WorkPostureSource,
		WorkPostureDetail:       p.WorkPostureDetail,
		ImportProvenance:        p.ImportProvenance,
		PreservedLegacySections: p.PreservedLegacySections,
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal plan document %q: %w", p.ID, err)
	}
	created := p.CreatedAt
	if created == "" {
		created = r.clock.Now().UTC().Format(planTimeFormat)
	}
	updated := p.UpdatedAt
	if updated == "" {
		updated = r.clock.Now().UTC().Format(planTimeFormat)
	}
	if _, err := r.db.ExecContext(ctx, upsertPlanSQL,
		p.ID, p.Slug, p.Title, string(p.Status), p.ContentHash, string(raw), created, updated,
	); err != nil {
		return fmt.Errorf("upsert plan %q: %w", p.ID, err)
	}
	return nil
}

func (r *sqliteRepository) Get(ctx context.Context, idOrSlug string) (Plan, bool, error) {
	p, err := scanPlan(r.db.QueryRowContext(ctx, getPlanSQL, idOrSlug, idOrSlug))
	if errors.Is(err, sql.ErrNoRows) {
		return Plan{}, false, nil
	}
	if err != nil {
		return Plan{}, false, fmt.Errorf("get plan %q: %w", idOrSlug, err)
	}
	return p, true, nil
}

func (r *sqliteRepository) List(ctx context.Context, filter ListFilter) ([]Plan, error) {
	rows, err := r.db.QueryContext(ctx, listPlansSQL)
	if err != nil {
		return nil, fmt.Errorf("list plans: %w", err)
	}
	defer rows.Close()
	out := make([]Plan, 0)
	for rows.Next() {
		p, err := scanPlan(rows)
		if err != nil {
			return nil, err
		}
		if filter.Status != "" && p.Status != filter.Status {
			continue
		}
		if !filter.IncludeArchived && filter.Status == "" && p.Status == PlanStatusArchived {
			continue
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate plans: %w", err)
	}
	return out, nil
}

func (r *sqliteRepository) ListEdges(ctx context.Context, planID string) ([]PlanEdge, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if planID == "" {
		rows, err = r.db.QueryContext(ctx, listAllEdgesSQL)
	} else {
		rows, err = r.db.QueryContext(ctx, listEdgesForPlanSQL, planID, planID)
	}
	if err != nil {
		return nil, fmt.Errorf("list plan edges: %w", err)
	}
	defer rows.Close()
	out := make([]PlanEdge, 0)
	for rows.Next() {
		var e PlanEdge
		if err := rows.Scan(&e.FromPlanID, &e.ToPlanID, &e.Kind); err != nil {
			return nil, fmt.Errorf("scan plan edge: %w", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate plan edges: %w", err)
	}
	return out, nil
}

func (r *sqliteRepository) SaveEdge(ctx context.Context, e PlanEdge) error {
	if e.Kind == "" {
		e.Kind = EdgeKindSupersedes
	}
	if _, err := r.db.ExecContext(ctx, upsertEdgeSQL, e.FromPlanID, e.ToPlanID, e.Kind); err != nil {
		return fmt.Errorf("save plan edge %s->%s: %w", e.FromPlanID, e.ToPlanID, err)
	}
	return nil
}

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanPlan(s rowScanner) (Plan, error) {
	var (
		p        Plan
		status   string
		document string
	)
	if err := s.Scan(&p.ID, &p.Slug, &p.Title, &status, &p.ContentHash, &document, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return Plan{}, err
	}
	p.Status = PlanStatus(status)
	var doc planDocument
	if err := json.Unmarshal([]byte(document), &doc); err != nil {
		return Plan{}, fmt.Errorf("unmarshal plan document %q: %w", p.ID, err)
	}
	p.Purpose = doc.Purpose
	p.Scope = doc.Scope
	p.Constraints = doc.Constraints
	p.NonGoals = doc.NonGoals
	p.DefinitionOfDone = doc.DefinitionOfDone
	p.References = doc.References
	p.RegressionAnchor = doc.RegressionAnchor
	p.Phases = doc.Phases
	p.Supersedes = doc.Supersedes
	p.SupersededBy = doc.SupersededBy
	p.RelevantContext = doc.RelevantContext
	p.ProblemStatement = doc.ProblemStatement
	p.TargetOutcome = doc.TargetOutcome
	p.Assumptions = doc.Assumptions
	p.TechnicalApproach = doc.TechnicalApproach
	p.ValidationStrategy = doc.ValidationStrategy
	p.FinalValidationCommands = doc.FinalValidationCommands
	p.RisksHazards = doc.RisksHazards
	p.ProhibitedApproaches = doc.ProhibitedApproaches
	p.WorkPosture = doc.WorkPosture
	p.WorkPostureSource = doc.WorkPostureSource
	p.WorkPostureDetail = doc.WorkPostureDetail
	p.ImportProvenance = doc.ImportProvenance
	p.PreservedLegacySections = doc.PreservedLegacySections
	return p, nil
}
