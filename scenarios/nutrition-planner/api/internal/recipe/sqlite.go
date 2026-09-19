package recipe

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

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
}

func NewSQLiteRepository(db SQLExecutor, clock schedule.Clock) Repository {
	return &sqliteRepository{db, clock}
}

const tf = time.RFC3339Nano

func (r *sqliteRepository) Create(ctx context.Context, w Recipe) (Recipe, error) {
	if w.IdempotencyKey != "" {
		var hash, existingID string
		err := r.db.QueryRowContext(ctx, `SELECT request_hash, recipe_id FROM recipe_idempotency WHERE workspace_id=? AND idempotency_key=?`, w.WorkspaceID, w.IdempotencyKey).Scan(&hash, &existingID)
		if err == nil {
			if hash != w.RequestHash {
				return Recipe{}, ErrIdempotencyConflict{w.IdempotencyKey}
			}
			return r.Get(ctx, existingID, w.WorkspaceID)
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return Recipe{}, fmt.Errorf("check idempotency key: %w", err)
		}
	}
	if w.ID == "" {
		w.ID = uuid.NewString()
	}
	now := r.clock.Now().UTC()
	if w.CreatedAt.IsZero() {
		w.CreatedAt = now
	}
	if w.UpdatedAt.IsZero() {
		w.UpdatedAt = w.CreatedAt
	}
	w.Revision = 1
	if w.Status == "" {
		w.Status = "draft"
	}
	if _, err := r.db.ExecContext(ctx, `INSERT INTO recipes(id,workspace_id,current_revision,created_at,updated_at) VALUES(?,?,?,?,?)`, w.ID, w.WorkspaceID, w.Revision, w.CreatedAt.Format(tf), w.UpdatedAt.Format(tf)); err != nil {
		return Recipe{}, fmt.Errorf("insert recipe: %w", err)
	}
	methods, _ := json.Marshal(w.Methods)
	groups, _ := json.Marshal(w.Groups)
	appliances, _ := json.Marshal(w.RequiredAppliances)
	evidence, _ := json.Marshal(w.AllergenEvidence)
	ingredients, _ := json.Marshal(w.Ingredients)
	if _, err := r.db.ExecContext(ctx, `INSERT INTO recipe_revisions(recipe_id,revision,workspace_id,name,notes,source_url,source_type,original_text,status,methods_json,groups_json,required_appliances_json,allergen_evidence_json,canonical_yield,serving_unit,ingredients_json,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, w.ID, w.Revision, w.WorkspaceID, w.Name, w.Notes, w.SourceURL, w.SourceType, w.OriginalText, w.Status, string(methods), string(groups), string(appliances), string(evidence), w.CanonicalYield, w.ServingUnit, string(ingredients), w.CreatedAt.Format(tf)); err != nil {
		return Recipe{}, fmt.Errorf("insert recipe revision: %w", err)
	}
	if w.IdempotencyKey != "" {
		if _, err := r.db.ExecContext(ctx, `INSERT INTO recipe_idempotency(workspace_id,idempotency_key,request_hash,recipe_id,created_at) VALUES(?,?,?,?,?)`, w.WorkspaceID, w.IdempotencyKey, w.RequestHash, w.ID, now.Format(tf)); err != nil {
			return Recipe{}, fmt.Errorf("record idempotency key: %w", err)
		}
	}
	return w, nil
}

func (r *sqliteRepository) List(ctx context.Context, workspaceID string) ([]Recipe, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT r.id,r.workspace_id,r.current_revision,v.name,v.notes,v.source_url,v.source_type,v.original_text,v.status,v.methods_json,v.groups_json,v.required_appliances_json,v.allergen_evidence_json,v.canonical_yield,v.serving_unit,v.ingredients_json,r.created_at,r.updated_at FROM recipes r JOIN recipe_revisions v ON v.recipe_id=r.id AND v.revision=r.current_revision WHERE r.workspace_id=? ORDER BY r.updated_at DESC,r.id DESC`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Recipe
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) Get(ctx context.Context, id, workspaceID string) (Recipe, error) {
	row := r.db.QueryRowContext(ctx, `SELECT r.id,r.workspace_id,r.current_revision,v.name,v.notes,v.source_url,v.source_type,v.original_text,v.status,v.methods_json,v.groups_json,v.required_appliances_json,v.allergen_evidence_json,v.canonical_yield,v.serving_unit,v.ingredients_json,r.created_at,r.updated_at FROM recipes r JOIN recipe_revisions v ON v.recipe_id=r.id AND v.revision=r.current_revision WHERE r.id=?`, id)
	v, err := scan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Recipe{}, ErrNotFound{id}
	}
	if err != nil {
		return Recipe{}, err
	}
	if v.WorkspaceID != workspaceID {
		return Recipe{}, ErrForbidden{id}
	}
	return v, nil
}

func (r *sqliteRepository) Update(ctx context.Context, in UpdateInput) (Recipe, error) {
	cur, err := r.Get(ctx, in.ID, in.WorkspaceID)
	if err != nil {
		return Recipe{}, err
	}
	if in.ExpectedRevision != cur.Revision {
		return Recipe{}, ErrConflict{in.ID, in.ExpectedRevision, cur.Revision}
	}
	next := cur
	next.Revision++
	next.Name = in.Name
	next.Notes = in.Notes
	next.SourceURL = in.SourceURL
	next.SourceType = in.SourceType
	next.OriginalText = in.OriginalText
	next.Methods = in.Methods
	next.Groups = in.Groups
	next.RequiredAppliances = in.RequiredAppliances
	next.AllergenEvidence = in.AllergenEvidence
	next.CanonicalYield = in.CanonicalYield
	next.ServingUnit = in.ServingUnit
	next.Ingredients = in.Ingredients
	next.UpdatedAt = r.clock.Now().UTC()
	if err := validateName(next.Name); err != nil {
		return Recipe{}, err
	}
	methods, _ := json.Marshal(next.Methods)
	groups, _ := json.Marshal(next.Groups)
	appliances, _ := json.Marshal(next.RequiredAppliances)
	evidence, _ := json.Marshal(next.AllergenEvidence)
	ingredients, _ := json.Marshal(next.Ingredients)
	if _, err := r.db.ExecContext(ctx, `INSERT INTO recipe_revisions(recipe_id,revision,workspace_id,name,notes,source_url,source_type,original_text,status,methods_json,groups_json,required_appliances_json,allergen_evidence_json,canonical_yield,serving_unit,ingredients_json,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, next.ID, next.Revision, next.WorkspaceID, next.Name, next.Notes, next.SourceURL, next.SourceType, next.OriginalText, next.Status, string(methods), string(groups), string(appliances), string(evidence), next.CanonicalYield, next.ServingUnit, string(ingredients), next.UpdatedAt.Format(tf)); err != nil {
		return Recipe{}, err
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE recipes SET current_revision=?,updated_at=? WHERE id=? AND workspace_id=? AND current_revision=?`, next.Revision, next.UpdatedAt.Format(tf), next.ID, next.WorkspaceID, cur.Revision); err != nil {
		return Recipe{}, err
	}
	return next, nil
}

func scan(s interface{ Scan(...any) error }) (Recipe, error) {
	var v Recipe
	var c, u, methodsJSON, groupsJSON, appliancesJSON, evidenceJSON, ingredientsJSON string
	if err := s.Scan(&v.ID, &v.WorkspaceID, &v.Revision, &v.Name, &v.Notes, &v.SourceURL, &v.SourceType, &v.OriginalText, &v.Status, &methodsJSON, &groupsJSON, &appliancesJSON, &evidenceJSON, &v.CanonicalYield, &v.ServingUnit, &ingredientsJSON, &c, &u); err != nil {
		return Recipe{}, err
	}
	var err error
	v.CreatedAt, err = time.Parse(tf, c)
	if err != nil {
		return Recipe{}, err
	}
	v.UpdatedAt, err = time.Parse(tf, u)
	if err != nil {
		return v, err
	}
	if methodsJSON != "" {
		if err := json.Unmarshal([]byte(methodsJSON), &v.Methods); err != nil {
			return Recipe{}, err
		}
	}
	if groupsJSON != "" {
		if err := json.Unmarshal([]byte(groupsJSON), &v.Groups); err != nil {
			return Recipe{}, err
		}
	}
	if appliancesJSON != "" {
		if err := json.Unmarshal([]byte(appliancesJSON), &v.RequiredAppliances); err != nil {
			return Recipe{}, err
		}
	}
	if evidenceJSON != "" {
		if err := json.Unmarshal([]byte(evidenceJSON), &v.AllergenEvidence); err != nil {
			return Recipe{}, err
		}
	}
	if ingredientsJSON != "" {
		if err := json.Unmarshal([]byte(ingredientsJSON), &v.Ingredients); err != nil {
			return Recipe{}, err
		}
	}
	return v, nil
}
