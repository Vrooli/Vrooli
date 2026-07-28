package facets

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

type SQLiteRepository struct{ db *sql.DB }

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository { return &SQLiteRepository{db: db} }

var seeds = []Definition{
	{ID: "standing-rule", Label: "Standing rule", RetentionPolicy: "pinned-or-review"},
	{ID: "environment-fact", Label: "Environment fact", RetentionPolicy: "retain"},
	{ID: "gotcha", Label: "Gotcha", RetentionPolicy: "retain"},
	{ID: "episode", Label: "Episode", RetentionPolicy: "compact", CompactionEligible: true},
	{ID: "thread", Label: "Thread", RetentionPolicy: "expire-on-resolution"},
	{ID: "entity-record", Label: "Entity record", RetentionPolicy: "retain"},
}

func (r *SQLiteRepository) Seed(ctx context.Context) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, d := range seeds {
		if _, err = tx.ExecContext(ctx, `INSERT INTO facet_definitions(id,label,created_at) VALUES(?,?,?) ON CONFLICT(id) DO NOTHING`, d.ID, d.Label, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO facet_policies(facet_id,retention_policy,compaction_eligible) VALUES(?,?,?) ON CONFLICT(facet_id) DO NOTHING`, d.ID, d.RetentionPolicy, d.CompactionEligible); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *SQLiteRepository) List(ctx context.Context) ([]Definition, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT d.id,d.label,p.retention_policy,p.compaction_eligible FROM facet_definitions d JOIN facet_policies p ON p.facet_id=d.id ORDER BY d.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Definition
	for rows.Next() {
		var d Definition
		if err = rows.Scan(&d.ID, &d.Label, &d.RetentionPolicy, &d.CompactionEligible); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) Validate(ctx context.Context, id string) error {
	if id == UnclassifiedFacet {
		return nil
	}
	var found string
	err := r.db.QueryRowContext(ctx, `SELECT id FROM facet_definitions WHERE id=?`, id).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUnknownFacet{ID: id}
	}
	return err
}

func (r *SQLiteRepository) CompactionEligible(ctx context.Context, entryID string) (bool, error) {
	var eligible, pinned bool
	err := r.db.QueryRowContext(ctx, `SELECT p.compaction_eligible, EXISTS(SELECT 1 FROM pins WHERE entry_id=?) FROM facet_assignments a JOIN facet_policies p ON p.facet_id=a.facet_id WHERE a.entry_id=? ORDER BY a.assigned_at DESC,a.id DESC LIMIT 1`, entryID, entryID).Scan(&eligible, &pinned)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return eligible && !pinned, err
}

func (r *SQLiteRepository) SetPin(ctx context.Context, entryID string, pinned bool) error {
	if !pinned {
		_, err := r.db.ExecContext(ctx, `DELETE FROM pins WHERE entry_id=?`, entryID)
		return err
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO pins(entry_id,pinned_at) VALUES(?,?) ON CONFLICT(entry_id) DO NOTHING`, entryID, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (r *SQLiteRepository) Pinned(ctx context.Context, entryID string) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pins WHERE entry_id=?)`, entryID).Scan(&ok)
	return ok, err
}

func (r *SQLiteRepository) Assign(ctx context.Context, a Assignment) (Assignment, error) {
	if err := r.Validate(ctx, a.FacetID); err != nil {
		return Assignment{}, err
	}
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	if a.AssignedAt.IsZero() {
		a.AssignedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO facet_assignments(id,entry_id,facet_id,assigned_at,actor_id) VALUES(?,?,?,?,?)`, a.ID, a.EntryID, a.FacetID, a.AssignedAt.Format(time.RFC3339Nano), a.ActorID)
	return a, err
}

func (r *SQLiteRepository) Assignments(ctx context.Context, entryID string) ([]Assignment, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,entry_id,facet_id,assigned_at,actor_id FROM facet_assignments WHERE entry_id=? ORDER BY assigned_at,id`, entryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Assignment
	for rows.Next() {
		var a Assignment
		var raw string
		if err = rows.Scan(&a.ID, &a.EntryID, &a.FacetID, &raw, &a.ActorID); err != nil {
			return nil, err
		}
		a.AssignedAt, _ = time.Parse(time.RFC3339Nano, raw)
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) MarkSuperseded(ctx context.Context, entryID, replacementID string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO marks(id,entry_id,kind,replacement_entry_id,created_at) VALUES(?,?,?,?,?)`, uuid.NewString(), entryID, "superseded", replacementID, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}
