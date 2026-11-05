package review

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	OutcomePassed  = "passed"
	OutcomeBlocked = "blocked"
)

type (
	Verdict struct {
		Mode     string
		Passed   bool
		Evidence string
		Finding  string
	}
	Run struct {
		ID, DraftID, Outcome string
		Verdicts             []Verdict
	}
	SQLExecutor interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
		QueryRowContext(context.Context, string, ...any) *sql.Row
	}
	Service interface {
		Record(context.Context, string, []Verdict) (Run, error)
		List(context.Context) ([]Run, error)
	}
	service struct{ db SQLExecutor }
)

func NewService(db SQLExecutor) Service { return &service{db} }
func (s *service) Record(ctx context.Context, draftID string, verdicts []Verdict) (Run, error) {
	if len(verdicts) == 0 {
		return Run{}, fmt.Errorf("review run requires declared failure-mode verdicts")
	}
	if err := s.validateDeclaredModes(ctx, draftID, verdicts); err != nil {
		return Run{}, err
	}
	priorRunIDs, err := s.unsupersededRunIDs(ctx, draftID)
	if err != nil {
		return Run{}, err
	}
	run := Run{ID: uuid.NewString(), DraftID: draftID, Outcome: OutcomePassed, Verdicts: verdicts}
	for _, v := range verdicts {
		if !v.Passed {
			run.Outcome = OutcomeBlocked
		}
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO review_runs (id,draft_id,outcome,created_at) VALUES (?,?,?,?)`, run.ID, run.DraftID, run.Outcome, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return Run{}, err
	}
	for _, v := range verdicts {
		if _, err := s.db.ExecContext(ctx, `INSERT INTO review_verdicts (run_id,mode,passed,evidence,finding) VALUES (?,?,?,?,?)`, run.ID, v.Mode, boolInt(v.Passed), v.Evidence, v.Finding); err != nil {
			return Run{}, err
		}
	}
	for _, priorRunID := range priorRunIDs {
		if _, err := s.db.ExecContext(ctx, `INSERT INTO review_supersessions (superseded_run_id, superseding_run_id) VALUES (?, ?)`, priorRunID, run.ID); err != nil {
			return Run{}, err
		}
	}
	return run, nil
}

func (s *service) unsupersededRunIDs(ctx context.Context, draftID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM review_runs WHERE draft_id = ? AND id NOT IN (SELECT superseded_run_id FROM review_supersessions)`, draftID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *service) validateDeclaredModes(ctx context.Context, draftID string, verdicts []Verdict) error {
	var postTypeID string
	if err := s.db.QueryRowContext(ctx, `SELECT post_type_id FROM drafts WHERE id = ?`, draftID).Scan(&postTypeID); err != nil {
		return fmt.Errorf("load draft post type: %w", err)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT failure_mode FROM post_type_failure_modes WHERE post_type_id = ? UNION SELECT mode FROM review_policy_failure_modes ORDER BY 1`, postTypeID)
	if err != nil {
		return err
	}
	defer rows.Close()
	expected := map[string]struct{}{}
	for rows.Next() {
		var mode string
		if err := rows.Scan(&mode); err != nil {
			return err
		}
		expected[mode] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(expected) == 0 {
		return fmt.Errorf("post type %q has no declared failure modes", postTypeID)
	}
	seen := map[string]struct{}{}
	for _, verdict := range verdicts {
		if verdict.Mode == "" {
			return fmt.Errorf("review verdict mode is required")
		}
		if _, duplicate := seen[verdict.Mode]; duplicate {
			return fmt.Errorf("duplicate review verdict mode %q", verdict.Mode)
		}
		seen[verdict.Mode] = struct{}{}
		if _, declared := expected[verdict.Mode]; !declared {
			return fmt.Errorf("review verdict mode %q is not declared by post type %q", verdict.Mode, postTypeID)
		}
	}
	for mode := range expected {
		if _, present := seen[mode]; !present {
			return fmt.Errorf("review is missing declared failure mode %q", mode)
		}
	}
	return nil
}

func (s *service) List(ctx context.Context) ([]Run, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,draft_id,outcome FROM review_runs ORDER BY created_at DESC,id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Run
	for rows.Next() {
		var run Run
		if err := rows.Scan(&run.ID, &run.DraftID, &run.Outcome); err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// A SQLite :memory: database exposes one connection in tests. Close the
	// parent cursor before issuing child queries so List cannot self-deadlock.
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i := range out {
		verdicts, err := s.verdicts(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Verdicts = verdicts
	}
	return out, nil
}

func (s *service) verdicts(ctx context.Context, id string) ([]Verdict, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT mode,passed,evidence,finding FROM review_verdicts WHERE run_id=? ORDER BY passed ASC, mode`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Verdict
	for rows.Next() {
		var v Verdict
		var passed int
		if err := rows.Scan(&v.Mode, &passed, &v.Evidence, &v.Finding); err != nil {
			return nil, err
		}
		v.Passed = passed != 0
		out = append(out, v)
	}
	return out, rows.Err()
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
