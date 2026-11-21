package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"
	"nutrition-planner/internal/decimalx"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
}

func NewSQLiteRepository(db SQLExecutor, clock schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clock}
}

func (r *sqliteRepository) Create(ctx context.Context, v Revision) (Revision, error) {
	if err := Validate(v); err != nil {
		return Revision{}, err
	}
	if v.ID == "" {
		v.ID = uuid.NewString()
	}
	if v.Revision == 0 {
		v.Revision = 1
	}
	if v.CreatedAt.IsZero() {
		v.CreatedAt = r.clock.Now().UTC()
	}
	timestamp := v.CreatedAt.Format(time.RFC3339Nano)
	if _, err := r.db.ExecContext(ctx, `INSERT INTO catalog_concepts(id,workspace_id,created_at) VALUES(?,?,?) ON CONFLICT(id) DO NOTHING`, v.ConceptID, v.WorkspaceID, timestamp); err != nil {
		return Revision{}, fmt.Errorf("insert catalog concept: %w", err)
	}
	nutrients, err := json.Marshal(v.Nutrients)
	if err != nil {
		return Revision{}, err
	}
	evidence, err := json.Marshal(v.AllergenEvidence)
	if err != nil {
		return Revision{}, err
	}
	if _, err = r.db.ExecContext(ctx, `INSERT INTO catalog_revisions(id,revision,workspace_id,concept_id,name,product_name,preparation,serving_quantity,serving_unit,nutrients_json,allergen_evidence_json,source_type,source_ref,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, v.ID, v.Revision, v.WorkspaceID, v.ConceptID, v.Name, v.ProductName, v.Preparation, v.ServingQuantity.String(), v.ServingUnit, string(nutrients), string(evidence), v.SourceType, v.SourceRef, timestamp); err != nil {
		return Revision{}, fmt.Errorf("insert catalog revision: %w", err)
	}
	return v, nil
}

func (r *sqliteRepository) List(ctx context.Context, workspaceID string) ([]Revision, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,revision,workspace_id,concept_id,name,product_name,preparation,serving_quantity,serving_unit,nutrients_json,allergen_evidence_json,source_type,source_ref,created_at FROM catalog_revisions WHERE workspace_id=? ORDER BY created_at DESC,id,revision DESC`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Revision
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) Get(ctx context.Context, id, workspaceID string, revision int64) (Revision, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id,revision,workspace_id,concept_id,name,product_name,preparation,serving_quantity,serving_unit,nutrients_json,allergen_evidence_json,source_type,source_ref,created_at FROM catalog_revisions WHERE id=? AND revision=?`, id, revision)
	v, err := scan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Revision{}, ErrNotFound{id, revision}
	}
	if err != nil {
		return Revision{}, err
	}
	if v.WorkspaceID != workspaceID {
		return Revision{}, ErrNotFound{id, revision}
	}
	return v, nil
}

func scan(s interface{ Scan(...any) error }) (Revision, error) {
	var v Revision
	var quantity, nutrients, evidence, created string
	if err := s.Scan(&v.ID, &v.Revision, &v.WorkspaceID, &v.ConceptID, &v.Name, &v.ProductName, &v.Preparation, &quantity, &v.ServingUnit, &nutrients, &evidence, &v.SourceType, &v.SourceRef, &created); err != nil {
		return Revision{}, err
	}
	var err error
	v.ServingQuantity, err = decimalx.Parse(quantity)
	if err != nil {
		return Revision{}, err
	}
	if err = json.Unmarshal([]byte(nutrients), &v.Nutrients); err != nil {
		return Revision{}, err
	}
	if err = json.Unmarshal([]byte(evidence), &v.AllergenEvidence); err != nil {
		return Revision{}, err
	}
	v.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return Revision{}, err
	}
	return v, nil
}
