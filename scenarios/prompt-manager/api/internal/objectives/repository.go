package objectives

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// queryExecutor is the routed storage seam. Passing the routed database (not a
// captured primary pool) keeps per-domain reads and writes on the same
// context-routed connection production and tests use.
type queryExecutor interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type txBeginner interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

// Repository is the engine-independent persistence seam for the objective
// domain. WithTx gives compound mutations one transaction; a repository that
// cannot begin transactions runs the callback directly.
type Repository interface {
	ListObjectives(ctx context.Context) ([]Objective, error)
	GetObjective(ctx context.Context, id string) (Objective, bool, error)
	PutObjective(ctx context.Context, o Objective) error
	DeleteObjective(ctx context.Context, id string) error

	ListAttachments(ctx context.Context, teamID string) ([]Attachment, error)
	ListTeamIDs(ctx context.Context) ([]string, error)
	GetAttachment(ctx context.Context, objectiveID, teamID string) (Attachment, bool, error)
	PutAttachment(ctx context.Context, a Attachment) error
	DeleteAttachment(ctx context.Context, objectiveID, teamID string) error
	DeleteAttachmentsForObjective(ctx context.Context, objectiveID string) error
	ReplaceAttachmentOrder(ctx context.Context, teamID string, objectiveIDs []string) error

	GetTeamAttachmentRevision(ctx context.Context, teamID string) (string, bool, error)
	PutTeamAttachmentRevision(ctx context.Context, teamID, revision string) error

	ListRelations(ctx context.Context) ([]Relation, error)
	PutRelation(ctx context.Context, r Relation) error
	DeleteRelation(ctx context.Context, from, to string) error

	PutImportSnapshot(ctx context.Context, sourceDigest, payload string) error
	GetImportSnapshot(ctx context.Context, sourceDigest string) (string, bool, error)
	PutImportReceipt(ctx context.Context, sourceDigest, payload string) error
	GetImportReceipt(ctx context.Context, sourceDigest string) (string, bool, error)

	WithTx(ctx context.Context, fn func(Repository) error) error
}

// SQLRepository is the SQL-backed Repository.
type SQLRepository struct {
	db       queryExecutor
	beginner txBeginner
}

// NewRepository builds a Repository over a routed database handle.
func NewRepository(db queryExecutor) *SQLRepository {
	repo := &SQLRepository{db: db}
	if b, ok := db.(txBeginner); ok {
		repo.beginner = b
	}
	return repo
}

// WithTx runs fn inside one transaction when the handle supports it.
func (r *SQLRepository) WithTx(ctx context.Context, fn func(Repository) error) error {
	if r.beginner == nil {
		return fn(r)
	}
	tx, err := r.beginner.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("objectives: begin transaction: %w", err)
	}
	if err := fn(&SQLRepository{db: tx}); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("objectives: commit transaction: %w", err)
	}
	return nil
}

func (r *SQLRepository) ListObjectives(ctx context.Context) ([]Objective, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, title, class, evidence_source, has_evidence, gap_marker, global_order, meaning_revision FROM objectives ORDER BY global_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Objective
	for rows.Next() {
		o, err := scanObjective(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (r *SQLRepository) GetObjective(ctx context.Context, id string) (Objective, bool, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, title, class, evidence_source, has_evidence, gap_marker, global_order, meaning_revision FROM objectives WHERE id = ?`, id)
	o, err := scanObjective(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Objective{}, false, nil
	}
	if err != nil {
		return Objective{}, false, err
	}
	return o, true, nil
}

func (r *SQLRepository) PutObjective(ctx context.Context, o Objective) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO objectives (id, title, class, evidence_source, has_evidence, gap_marker, global_order, meaning_revision)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET title = excluded.title, class = excluded.class, evidence_source = excluded.evidence_source, has_evidence = excluded.has_evidence, gap_marker = excluded.gap_marker, global_order = excluded.global_order, meaning_revision = excluded.meaning_revision`,
		o.ID, o.Title, string(o.Class), o.EvidenceSource, boolToInt(o.HasEvidence), o.GapMarker, o.GlobalOrder, o.MeaningRevision)
	return err
}

func (r *SQLRepository) DeleteObjective(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM objectives WHERE id = ?`, id)
	return err
}

func (r *SQLRepository) ListAttachments(ctx context.Context, teamID string) ([]Attachment, error) {
	query := `SELECT objective_id, team_id, role, coverage, note, priority, acknowledged_revision FROM objective_attachments`
	args := []any{}
	if teamID != "" {
		query += ` WHERE team_id = ?`
		args = append(args, teamID)
	}
	query += ` ORDER BY team_id, priority, objective_id`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Attachment
	for rows.Next() {
		a, err := scanAttachment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *SQLRepository) ListTeamIDs(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT DISTINCT team_id FROM objective_attachments ORDER BY team_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var teamID string
		if err := rows.Scan(&teamID); err != nil {
			return nil, err
		}
		out = append(out, teamID)
	}
	return out, rows.Err()
}

func (r *SQLRepository) GetAttachment(ctx context.Context, objectiveID, teamID string) (Attachment, bool, error) {
	row := r.db.QueryRowContext(ctx, `SELECT objective_id, team_id, role, coverage, note, priority, acknowledged_revision FROM objective_attachments WHERE objective_id = ? AND team_id = ?`, objectiveID, teamID)
	a, err := scanAttachment(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Attachment{}, false, nil
	}
	if err != nil {
		return Attachment{}, false, err
	}
	return a, true, nil
}

func (r *SQLRepository) PutAttachment(ctx context.Context, a Attachment) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO objective_attachments (objective_id, team_id, role, coverage, note, priority, acknowledged_revision)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(objective_id, team_id) DO UPDATE SET role = excluded.role, coverage = excluded.coverage, note = excluded.note, priority = excluded.priority, acknowledged_revision = excluded.acknowledged_revision`,
		a.ObjectiveID, a.TeamID, a.Role, a.Coverage, a.Note, a.Priority, a.AcknowledgedRevision)
	return err
}

func (r *SQLRepository) DeleteAttachment(ctx context.Context, objectiveID, teamID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM objective_attachments WHERE objective_id = ? AND team_id = ?`, objectiveID, teamID)
	return err
}

func (r *SQLRepository) DeleteAttachmentsForObjective(ctx context.Context, objectiveID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM objective_attachments WHERE objective_id = ?`, objectiveID)
	return err
}

func (r *SQLRepository) ReplaceAttachmentOrder(ctx context.Context, teamID string, objectiveIDs []string) error {
	for priority, id := range objectiveIDs {
		if _, err := r.db.ExecContext(ctx, `UPDATE objective_attachments SET priority = ? WHERE team_id = ? AND objective_id = ?`, priority, teamID, id); err != nil {
			return err
		}
	}
	return nil
}

func (r *SQLRepository) GetTeamAttachmentRevision(ctx context.Context, teamID string) (string, bool, error) {
	var revision string
	err := r.db.QueryRowContext(ctx, `SELECT revision FROM team_attachment_revisions WHERE team_id = ?`, teamID).Scan(&revision)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return revision, true, nil
}

func (r *SQLRepository) PutTeamAttachmentRevision(ctx context.Context, teamID, revision string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO team_attachment_revisions (team_id, revision) VALUES (?, ?) ON CONFLICT(team_id) DO UPDATE SET revision = excluded.revision`, teamID, revision)
	return err
}

func (r *SQLRepository) ListRelations(ctx context.Context) ([]Relation, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT from_objective_id, to_objective_id FROM objective_relations ORDER BY from_objective_id, to_objective_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Relation
	for rows.Next() {
		var rel Relation
		if err := rows.Scan(&rel.FromObjectiveID, &rel.ToObjectiveID); err != nil {
			return nil, err
		}
		out = append(out, rel)
	}
	return out, rows.Err()
}

func (r *SQLRepository) PutRelation(ctx context.Context, rel Relation) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO objective_relations (from_objective_id, to_objective_id) VALUES (?, ?) ON CONFLICT(from_objective_id, to_objective_id) DO NOTHING`, rel.FromObjectiveID, rel.ToObjectiveID)
	return err
}

func (r *SQLRepository) DeleteRelation(ctx context.Context, from, to string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM objective_relations WHERE from_objective_id = ? AND to_objective_id = ?`, from, to)
	return err
}

func (r *SQLRepository) PutImportSnapshot(ctx context.Context, sourceDigest, payload string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO objective_import_snapshots (source_digest, payload) VALUES (?, ?) ON CONFLICT(source_digest) DO NOTHING`, sourceDigest, payload)
	return err
}

func (r *SQLRepository) GetImportSnapshot(ctx context.Context, sourceDigest string) (string, bool, error) {
	var payload string
	err := r.db.QueryRowContext(ctx, `SELECT payload FROM objective_import_snapshots WHERE source_digest = ?`, sourceDigest).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return payload, true, nil
}

func (r *SQLRepository) PutImportReceipt(ctx context.Context, sourceDigest, payload string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO objective_import_receipts (source_digest, payload) VALUES (?, ?) ON CONFLICT(source_digest) DO UPDATE SET payload = excluded.payload`, sourceDigest, payload)
	return err
}

func (r *SQLRepository) GetImportReceipt(ctx context.Context, sourceDigest string) (string, bool, error) {
	var payload string
	err := r.db.QueryRowContext(ctx, `SELECT payload FROM objective_import_receipts WHERE source_digest = ?`, sourceDigest).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return payload, true, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanObjective(row rowScanner) (Objective, error) {
	var (
		o           Objective
		class       string
		hasEvidence int
	)
	if err := row.Scan(&o.ID, &o.Title, &class, &o.EvidenceSource, &hasEvidence, &o.GapMarker, &o.GlobalOrder, &o.MeaningRevision); err != nil {
		return Objective{}, err
	}
	o.Class = Class(class)
	o.HasEvidence = hasEvidence != 0
	return o, nil
}

func scanAttachment(row rowScanner) (Attachment, error) {
	var a Attachment
	if err := row.Scan(&a.ObjectiveID, &a.TeamID, &a.Role, &a.Coverage, &a.Note, &a.Priority, &a.AcknowledgedRevision); err != nil {
		return Attachment{}, err
	}
	return a, nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
