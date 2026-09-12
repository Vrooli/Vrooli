package posttypes

import (
	"context"
	"database/sql"
	"fmt"
)

const (
	StatusV0     = "v0"
	StatusActive = "active"
)

type (
	PostType struct {
		ID, Status, PairedSkill                      string
		SkillExists, DocV1, ResponsibilitiesDeclared bool
		FailureModes                                 []string
	}
	Criterion struct {
		ID     string
		Passed bool
		Reason string
	}
	Evaluation struct {
		Active   bool
		Criteria []Criterion
	}
	SQLExecutor interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
		QueryRowContext(context.Context, string, ...any) *sql.Row
	}
	Registry interface {
		Upsert(context.Context, PostType) error
		List(context.Context) ([]PostType, error)
		Evaluate(context.Context, string) (Evaluation, error)
	}
	registry struct{ db SQLExecutor }
)

func NewRegistry(db SQLExecutor) Registry { return &registry{db: db} }
func (r *registry) Upsert(ctx context.Context, postType PostType) error {
	seen := make(map[string]struct{}, len(postType.FailureModes))
	for _, mode := range postType.FailureModes {
		if mode == "" {
			return fmt.Errorf("failure mode cannot be empty")
		}
		if _, duplicate := seen[mode]; duplicate {
			return fmt.Errorf("duplicate failure mode %q", mode)
		}
		seen[mode] = struct{}{}
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO post_types (id,status,paired_skill,skill_exists,doc_v1,responsibilities_declared) VALUES (?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET status=excluded.status,paired_skill=excluded.paired_skill,skill_exists=excluded.skill_exists,doc_v1=excluded.doc_v1,responsibilities_declared=excluded.responsibilities_declared`, postType.ID, postType.Status, postType.PairedSkill, boolInt(postType.SkillExists), boolInt(postType.DocV1), boolInt(postType.ResponsibilitiesDeclared))
	if err != nil {
		return err
	}
	if _, err = r.db.ExecContext(ctx, `DELETE FROM post_type_failure_modes WHERE post_type_id = ?`, postType.ID); err != nil {
		return err
	}
	for _, mode := range postType.FailureModes {
		if _, err = r.db.ExecContext(ctx, `INSERT INTO post_type_failure_modes (post_type_id, failure_mode) VALUES (?, ?)`, postType.ID, mode); err != nil {
			return err
		}
	}
	return nil
}

func (r *registry) List(ctx context.Context) ([]PostType, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,status,paired_skill,skill_exists,doc_v1,responsibilities_declared FROM post_types ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PostType
	for rows.Next() {
		var p PostType
		var skill, doc, resp int
		if err := rows.Scan(&p.ID, &p.Status, &p.PairedSkill, &skill, &doc, &resp); err != nil {
			return nil, err
		}
		p.SkillExists = skill != 0
		p.DocV1 = doc != 0
		p.ResponsibilitiesDeclared = resp != 0
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for index := range out {
		modes, err := r.failureModes(ctx, out[index].ID)
		if err != nil {
			return nil, err
		}
		out[index].FailureModes = modes
	}
	return out, nil
}

func (r *registry) failureModes(ctx context.Context, id string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT failure_mode FROM post_type_failure_modes WHERE post_type_id = ? ORDER BY failure_mode`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var modes []string
	for rows.Next() {
		var mode string
		if err := rows.Scan(&mode); err != nil {
			return nil, err
		}
		modes = append(modes, mode)
	}
	return modes, rows.Err()
}

func (r *registry) Evaluate(ctx context.Context, id string) (Evaluation, error) {
	var p PostType
	var skill, doc, resp int
	err := r.db.QueryRowContext(ctx, `SELECT id,status,paired_skill,skill_exists,doc_v1,responsibilities_declared FROM post_types WHERE id=?`, id).Scan(&p.ID, &p.Status, &p.PairedSkill, &skill, &doc, &resp)
	if err != nil {
		return Evaluation{}, fmt.Errorf("load post type %q: %w", id, err)
	}
	p.SkillExists = skill != 0
	p.DocV1 = doc != 0
	p.ResponsibilitiesDeclared = resp != 0
	criteria := []Criterion{{ID: "paired_skill_declared", Passed: p.PairedSkill != "", Reason: "paired skill is required"}, {ID: "paired_skill_exists", Passed: p.SkillExists, Reason: "paired skill must exist"}, {ID: "doc_is_v1", Passed: p.DocV1, Reason: "canon document must be v1"}, {ID: "responsibilities_declared", Passed: p.ResponsibilitiesDeclared, Reason: "member responsibility acknowledgement must be declared"}}
	active := p.Status == StatusActive
	for _, c := range criteria {
		active = active && c.Passed
	}
	return Evaluation{Active: active, Criteria: criteria}, nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
