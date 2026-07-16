package reconcile

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// SQLExecutor is the narrow database surface the repository depends on.
// Both *sql.DB in tests and *database.RoutedDB in production satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db SQLExecutor
}

// NewSQLiteRepository constructs the production evidence repository.
func NewSQLiteRepository(db SQLExecutor) EvidenceRepository {
	return &sqliteRepository{db: db}
}

var _ EvidenceRepository = (*sqliteRepository)(nil)

const evidenceTimeFormat = time.RFC3339Nano

const (
	insertEvidenceSQL = `
INSERT INTO reconcile_evidence (
  id, scenario, document_kind, page_id, component_id, component_title, example_name, route, state_id, claim_id, claim_type, verdict,
  capture_ref, ax_node_json, message, checked_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  scenario=excluded.scenario,
  document_kind=excluded.document_kind,
  page_id=excluded.page_id,
  component_id=excluded.component_id,
  component_title=excluded.component_title,
  example_name=excluded.example_name,
  route=excluded.route,
  state_id=excluded.state_id,
  claim_id=excluded.claim_id,
  claim_type=excluded.claim_type,
  verdict=excluded.verdict,
  capture_ref=excluded.capture_ref,
  ax_node_json=excluded.ax_node_json,
  message=excluded.message,
  checked_at=excluded.checked_at`

	upsertEvidenceViewportSQL = `
INSERT INTO reconcile_evidence_viewports (
  evidence_id, viewport_id, viewport_width, viewport_height)
VALUES (?, ?, ?, ?)
ON CONFLICT(evidence_id) DO UPDATE SET
  viewport_id=excluded.viewport_id,
  viewport_width=excluded.viewport_width,
  viewport_height=excluded.viewport_height`

	evidenceColumns = `
reconcile_evidence.id, scenario, document_kind, page_id, component_id, component_title, example_name, route, state_id,
COALESCE(viewport_id, ''), COALESCE(viewport_width, 0), COALESCE(viewport_height, 0),
claim_id, claim_type, verdict, capture_ref, ax_node_json, message, checked_at`
)

func (r *sqliteRepository) SaveEvidence(ctx context.Context, evidence Evidence) error {
	if strings.TrimSpace(evidence.ID) == "" {
		return fmt.Errorf("reconcile evidence requires id")
	}
	if _, err := r.db.ExecContext(ctx, insertEvidenceSQL,
		evidence.ID,
		evidence.Scenario,
		evidence.DocumentKind,
		evidence.PageID,
		evidence.ComponentID,
		evidence.ComponentTitle,
		evidence.ExampleName,
		evidence.Route,
		evidence.StateID,
		evidence.ClaimID,
		evidence.ClaimType,
		evidence.Verdict,
		evidence.CaptureRef,
		evidence.AXNodeJSON,
		evidence.Message,
		evidence.CheckedAt,
	); err != nil {
		return fmt.Errorf("save reconcile evidence %q: %w", evidence.ID, err)
	}
	if _, err := r.db.ExecContext(ctx, upsertEvidenceViewportSQL,
		evidence.ID,
		evidence.ViewportID,
		evidence.ViewportWidth,
		evidence.ViewportHeight,
	); err != nil {
		return fmt.Errorf("save reconcile evidence viewport %q: %w", evidence.ID, err)
	}
	return nil
}

func (r *sqliteRepository) ListEvidence(ctx context.Context, filter EvidenceFilter) ([]Evidence, error) {
	query := `SELECT ` + evidenceColumns + ` FROM reconcile_evidence
LEFT JOIN reconcile_evidence_viewports ON reconcile_evidence_viewports.evidence_id = reconcile_evidence.id`
	var clauses []string
	var args []any
	if filter.Scenario != "" {
		clauses = append(clauses, "scenario = ?")
		args = append(args, filter.Scenario)
	}
	if filter.PageID != "" {
		clauses = append(clauses, "page_id = ?")
		args = append(args, filter.PageID)
	}
	if filter.ComponentID != "" {
		clauses = append(clauses, "component_id = ?")
		args = append(args, filter.ComponentID)
	}
	if filter.ClaimID != "" {
		clauses = append(clauses, "claim_id = ?")
		args = append(args, filter.ClaimID)
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY checked_at DESC, id DESC"
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list reconcile evidence: %w", err)
	}
	defer rows.Close()

	var out []Evidence
	for rows.Next() {
		var e Evidence
		if err := rows.Scan(
			&e.ID,
			&e.Scenario,
			&e.DocumentKind,
			&e.PageID,
			&e.ComponentID,
			&e.ComponentTitle,
			&e.ExampleName,
			&e.Route,
			&e.StateID,
			&e.ViewportID,
			&e.ViewportWidth,
			&e.ViewportHeight,
			&e.ClaimID,
			&e.ClaimType,
			&e.Verdict,
			&e.CaptureRef,
			&e.AXNodeJSON,
			&e.Message,
			&e.CheckedAt,
		); err != nil {
			return nil, fmt.Errorf("scan reconcile evidence: %w", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate reconcile evidence: %w", err)
	}
	return out, nil
}
