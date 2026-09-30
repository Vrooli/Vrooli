package commitments

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
	id    func() string
}

func NewSQLiteRepository(db SQLExecutor, clock schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clock, id: uuid.NewString}
}

const commitmentSelect = `SELECT id,result,definition_of_done,promised_boundary,timezone,beneficiary,assumptions,scope_exclusions,state,risk,acknowledgment_status,created_at,updated_at,revision FROM commitments`

func scanCommitment(row interface{ Scan(...any) error }) (Commitment, error) {
	var x Commitment
	err := row.Scan(&x.ID, &x.Result, &x.DefinitionOfDone, &x.PromisedBoundary, &x.Timezone, &x.Beneficiary, &x.Assumptions, &x.ScopeExclusions, &x.State, &x.Risk, &x.AcknowledgmentStatus, &x.CreatedAt, &x.UpdatedAt, &x.Revision)
	return x, err
}

func (r *sqliteRepository) List(ctx context.Context) ([]Commitment, error) {
	rows, err := r.db.QueryContext(ctx, commitmentSelect+` ORDER BY CASE WHEN state='active' THEN 0 WHEN state='proposed' THEN 1 ELSE 2 END,promised_boundary,updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Commitment{}
	for rows.Next() {
		x, err := scanCommitment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) Create(ctx context.Context, x Commitment) (Commitment, error) {
	if x.ID == "" {
		x.ID = r.id()
	}
	now := r.clock.Now().Unix()
	_, err := r.db.ExecContext(ctx, `INSERT INTO commitments (id,result,definition_of_done,promised_boundary,timezone,beneficiary,assumptions,scope_exclusions,state,risk,acknowledgment_status,created_at,updated_at,revision) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, x.ID, x.Result, x.DefinitionOfDone, x.PromisedBoundary, x.Timezone, x.Beneficiary, x.Assumptions, x.ScopeExclusions, x.State, x.Risk, x.AcknowledgmentStatus, now, now, x.Revision)
	if err != nil {
		return Commitment{}, fmt.Errorf("insert commitment: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO commitment_revisions (id,commitment_id,promised_boundary,assumptions,scope_exclusions,reason,acknowledgment_status,created_at,revision) VALUES (?,?,?,?,?,?,?,?,?)`, r.id(), x.ID, x.PromisedBoundary, x.Assumptions, x.ScopeExclusions, "initial promise", x.AcknowledgmentStatus, now, x.Revision)
	if err != nil {
		return Commitment{}, fmt.Errorf("insert commitment revision: %w", err)
	}
	x.CreatedAt, x.UpdatedAt = now, now
	return x, nil
}

func (r *sqliteRepository) UpdateState(ctx context.Context, id, state string, revision int64) (Commitment, error) {
	now := r.clock.Now().Unix()
	res, err := r.db.ExecContext(ctx, `UPDATE commitments SET state=?,updated_at=?,revision=revision+1 WHERE id=? AND revision=?`, state, now, id, revision)
	if err != nil {
		return Commitment{}, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return Commitment{}, err
	}
	if n != 1 {
		return Commitment{}, ErrRevisionConflict{id}
	}
	x, err := scanCommitment(r.db.QueryRowContext(ctx, commitmentSelect+` WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Commitment{}, ErrCommitmentNotFound{id}
	}
	return x, err
}

func (r *sqliteRepository) Revise(ctx context.Context, in ReviseInput) (Commitment, Revision, error) {
	now := r.clock.Now().Unix()
	res, err := r.db.ExecContext(ctx, `UPDATE commitments SET promised_boundary=?,assumptions=?,scope_exclusions=?,acknowledgment_status=?,updated_at=?,revision=revision+1 WHERE id=? AND revision=?`, in.PromisedBoundary, in.Assumptions, in.ScopeExclusions, in.AcknowledgmentStatus, now, in.ID, in.ExpectedRevision)
	if err != nil {
		return Commitment{}, Revision{}, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return Commitment{}, Revision{}, err
	}
	if n != 1 {
		return Commitment{}, Revision{}, ErrRevisionConflict{in.ID}
	}
	x, err := scanCommitment(r.db.QueryRowContext(ctx, commitmentSelect+` WHERE id=?`, in.ID))
	if errors.Is(err, sql.ErrNoRows) {
		return Commitment{}, Revision{}, ErrCommitmentNotFound{in.ID}
	}
	if err != nil {
		return Commitment{}, Revision{}, err
	}
	rev := Revision{ID: r.id(), CommitmentID: x.ID, PromisedBoundary: x.PromisedBoundary, Assumptions: x.Assumptions, ScopeExclusions: x.ScopeExclusions, Reason: in.Reason, AcknowledgmentStatus: x.AcknowledgmentStatus, CreatedAt: now, Revision: x.Revision}
	if _, err := r.db.ExecContext(ctx, `INSERT INTO commitment_revisions (id,commitment_id,promised_boundary,assumptions,scope_exclusions,reason,acknowledgment_status,created_at,revision) VALUES (?,?,?,?,?,?,?,?,?)`, rev.ID, rev.CommitmentID, rev.PromisedBoundary, rev.Assumptions, rev.ScopeExclusions, rev.Reason, rev.AcknowledgmentStatus, rev.CreatedAt, rev.Revision); err != nil {
		return Commitment{}, Revision{}, fmt.Errorf("insert commitment revision: %w", err)
	}
	return x, rev, nil
}
