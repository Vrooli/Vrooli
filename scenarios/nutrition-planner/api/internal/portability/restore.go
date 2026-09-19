package portability

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"
	"nutrition-planner/internal/planning"
)

type SQLBeginner interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type RestoreResult struct {
	WorkspaceRevision int64
	RecipesApplied    int
	CheckpointID      string
}

type Restorer interface {
	ApplyWorkspace(context.Context, string, int64, string, Export) (RestoreResult, error)
	PreviewRecipes(context.Context, string, Export) (RecipeImportPreview, error)
	ApplyRecipes(context.Context, string, int64, string, string, Export) (RecipeImportResult, error)
}

type RecipeImportPreview struct {
	RecipeCount, DuplicateCount, ConflictCount int
	Errors                                     []string
}

type RecipeImportResult struct {
	WorkspaceRevision int64
	RecipesApplied    int
	RecipesSkipped    int
	RemappedIDs       []string
}

type ErrRestoreRevision struct {
	WorkspaceID      string
	Expected, Actual int64
}

func (e ErrRestoreRevision) Error() string {
	return fmt.Sprintf("workspace %q is stale: expected revision %d, actual %d", e.WorkspaceID, e.Expected, e.Actual)
}

type sqliteRestorer struct {
	db    SQLBeginner
	clock schedule.Clock
}

func NewSQLiteRestorer(db SQLBeginner, clock schedule.Clock) *sqliteRestorer {
	return &sqliteRestorer{db: db, clock: clock}
}

// ApplyWorkspace replaces only the domains declared by the native workspace
// envelope. It creates a complete pre-restore checkpoint inside the same
// transaction and never reads or writes owner, authentication, entitlement,
// provider-secret, or billing fields from the imported content.
func (r *sqliteRestorer) ApplyWorkspace(ctx context.Context, workspaceID string, expectedRevision int64, idempotencyKey string, envelope Export) (RestoreResult, error) {
	if _, err := ImportWorkspaceMust(envelope); err != nil {
		return RestoreResult{}, err
	}
	if workspaceID == "" || expectedRevision < 0 || idempotencyKey == "" {
		return RestoreResult{}, errors.New("workspace restore requires workspace, expected revision, and idempotency key")
	}
	content, err := json.Marshal(envelope)
	if err != nil {
		return RestoreResult{}, err
	}
	hash := sha256.Sum256(content)
	requestHash := hex.EncodeToString(hash[:])
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return RestoreResult{}, err
	}
	defer tx.Rollback()

	var existingHash, existingCheckpoint string
	var existingRevision int64
	err = tx.QueryRowContext(ctx, `SELECT request_hash,workspace_revision,checkpoint_id FROM portability_restore_operations WHERE workspace_id=? AND idempotency_key=?`, workspaceID, idempotencyKey).Scan(&existingHash, &existingRevision, &existingCheckpoint)
	if err == nil {
		if existingHash != requestHash {
			return RestoreResult{}, errors.New("workspace restore idempotency key was reused with different content")
		}
		return RestoreResult{WorkspaceRevision: existingRevision, CheckpointID: existingCheckpoint}, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return RestoreResult{}, err
	}

	var actualRevision int64
	if err := tx.QueryRowContext(ctx, `SELECT revision FROM workspaces WHERE id=?`, workspaceID).Scan(&actualRevision); err != nil {
		return RestoreResult{}, err
	}
	if actualRevision != expectedRevision {
		return RestoreResult{}, ErrRestoreRevision{workspaceID, expectedRevision, actualRevision}
	}

	previous, err := snapshotCurrent(ctx, tx, workspaceID)
	if err != nil {
		return RestoreResult{}, err
	}
	previousPlanRevision, previousPlan, err := readPlan(ctx, tx, workspaceID)
	if err != nil {
		return RestoreResult{}, err
	}
	checkpointID := uuid.NewString()
	previousJSON, err := json.Marshal(map[string]any{"recipes": previous, "planRevision": previousPlanRevision, "planJson": previousPlan})
	if err != nil {
		return RestoreResult{}, err
	}
	now := r.clock.Now().UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, `INSERT INTO portability_restore_checkpoints(id,workspace_id,expected_revision,previous_state_json,created_at) VALUES(?,?,?,?,?)`, checkpointID, workspaceID, expectedRevision, string(previousJSON), now); err != nil {
		return RestoreResult{}, err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM recipe_revisions WHERE workspace_id=?`, workspaceID); err != nil {
		return RestoreResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM recipes WHERE workspace_id=?`, workspaceID); err != nil {
		return RestoreResult{}, err
	}

	remap := map[string]string{}
	imported := make([]Recipe, 0)
	seen := map[string]bool{}
	for _, record := range envelope.Records {
		if record.Kind != "recipe_revision" {
			continue
		}
		var item Recipe
		if err := json.Unmarshal(record.Data, &item); err != nil || item.ID == "" || item.Revision < 1 || item.Name == "" {
			return RestoreResult{}, fmt.Errorf("invalid imported recipe record %q", record.ID)
		}
		if seen[item.ID] {
			return RestoreResult{}, fmt.Errorf("duplicate imported recipe %q", item.ID)
		}
		seen[item.ID] = true
		var otherWorkspace string
		if err := tx.QueryRowContext(ctx, `SELECT workspace_id FROM recipes WHERE id=?`, item.ID).Scan(&otherWorkspace); err == nil && otherWorkspace != workspaceID {
			remap[item.ID] = uuid.NewString()
		}
		if remap[item.ID] != "" {
			item.ID = remap[item.ID]
		}
		imported = append(imported, item)
	}
	for _, item := range imported {
		methods, _ := json.Marshal(item.Methods)
		groups, _ := json.Marshal(item.Groups)
		appliances, _ := json.Marshal(item.RequiredAppliances)
		evidence, _ := json.Marshal(item.AllergenEvidence)
		ingredients, _ := json.Marshal(item.Ingredients)
		if _, err := tx.ExecContext(ctx, `INSERT INTO recipes(id,workspace_id,current_revision,created_at,updated_at) VALUES(?,?,?,?,?)`, item.ID, workspaceID, item.Revision, now, now); err != nil {
			return RestoreResult{}, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO recipe_revisions(recipe_id,revision,workspace_id,name,notes,source_url,source_type,original_text,status,methods_json,groups_json,required_appliances_json,allergen_evidence_json,canonical_yield,serving_unit,ingredients_json,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, item.ID, item.Revision, workspaceID, item.Name, item.Notes, item.SourceURL, item.SourceType, item.OriginalText, item.Status, string(methods), string(groups), string(appliances), string(evidence), item.CanonicalYield, item.ServingUnit, string(ingredients), now); err != nil {
			return RestoreResult{}, err
		}
	}

	var planRecord *Record
	for i := range envelope.Records {
		if envelope.Records[i].Kind == "plan" {
			planRecord = &envelope.Records[i]
			break
		}
	}
	if planRecord == nil {
		if _, err := tx.ExecContext(ctx, `DELETE FROM plans WHERE workspace_id=?`, workspaceID); err != nil {
			return RestoreResult{}, err
		}
	} else {
		var data struct {
			PlanJSON string `json:"planJson"`
		}
		if err := json.Unmarshal(planRecord.Data, &data); err != nil {
			return RestoreResult{}, errors.New("invalid imported plan record")
		}
		planJSON, err := remapPlan(data.PlanJSON, remap)
		if err != nil {
			return RestoreResult{}, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO plans(workspace_id,revision,plan_json,updated_at) VALUES(?,?,?,?) ON CONFLICT(workspace_id) DO UPDATE SET revision=excluded.revision,plan_json=excluded.plan_json,updated_at=excluded.updated_at`, workspaceID, previousPlanRevision+1, planJSON, now); err != nil {
			return RestoreResult{}, err
		}
	}

	nextWorkspaceRevision := actualRevision + 1
	if _, err := tx.ExecContext(ctx, `UPDATE workspaces SET revision=?,updated_at=? WHERE id=? AND revision=?`, nextWorkspaceRevision, now, workspaceID, actualRevision); err != nil {
		return RestoreResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO portability_restore_operations(workspace_id,idempotency_key,request_hash,workspace_revision,checkpoint_id,created_at) VALUES(?,?,?,?,?,?)`, workspaceID, idempotencyKey, requestHash, nextWorkspaceRevision, checkpointID, now); err != nil {
		return RestoreResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return RestoreResult{}, err
	}
	return RestoreResult{WorkspaceRevision: nextWorkspaceRevision, RecipesApplied: len(imported), CheckpointID: checkpointID}, nil
}

func (r *sqliteRestorer) PreviewRecipes(ctx context.Context, workspaceID string, envelope Export) (RecipeImportPreview, error) {
	if workspaceID == "" {
		return RecipeImportPreview{}, errors.New("workspace_id is required")
	}
	preview := RecipeImportPreview{RecipeCount: len(envelope.Recipes)}
	seen := map[string]bool{}
	for _, incoming := range envelope.Recipes {
		if seen[incoming.ID] {
			preview.DuplicateCount++
			continue
		}
		seen[incoming.ID] = true
		var existing storedRecipe
		err := r.db.QueryRowContext(ctx, `SELECT recipe_id,revision,name,notes,source_url,source_type,original_text,status,methods_json,groups_json,required_appliances_json,allergen_evidence_json,canonical_yield,serving_unit,ingredients_json FROM recipe_revisions WHERE recipe_id=? AND revision=(SELECT current_revision FROM recipes WHERE id=? AND workspace_id=?)`, incoming.ID, incoming.ID, workspaceID).Scan(&existing.ID, &existing.Revision, &existing.Name, &existing.Notes, &existing.SourceURL, &existing.SourceType, &existing.OriginalText, &existing.Status, &existing.MethodsJSON, &existing.GroupsJSON, &existing.AppliancesJSON, &existing.EvidenceJSON, &existing.CanonicalYield, &existing.ServingUnit, &existing.IngredientsJSON)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return RecipeImportPreview{}, err
		}
		current, err := existing.Portable()
		if err != nil {
			return RecipeImportPreview{}, err
		}
		if canonicalRecipe(current) != canonicalRecipe(incoming) {
			preview.ConflictCount++
		}
	}
	return preview, nil
}

func (r *sqliteRestorer) ApplyRecipes(ctx context.Context, workspaceID string, expectedRevision int64, idempotencyKey, policy string, envelope Export) (RecipeImportResult, error) {
	if _, err := ImportRecipesMust(envelope); err != nil {
		return RecipeImportResult{}, err
	}
	if workspaceID == "" || expectedRevision < 0 || idempotencyKey == "" {
		return RecipeImportResult{}, errors.New("recipe import requires workspace, expected revision, and idempotency key")
	}
	if policy == "" {
		policy = "copy_incoming"
	}
	if policy != "keep_existing" && policy != "copy_incoming" && policy != "replace" {
		return RecipeImportResult{}, errors.New("unsupported recipe conflict policy")
	}
	content, err := json.Marshal(envelope)
	if err != nil {
		return RecipeImportResult{}, err
	}
	hash := sha256.Sum256(append(content, []byte("\x00"+policy)...))
	requestHash := hex.EncodeToString(hash[:])
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return RecipeImportResult{}, err
	}
	defer tx.Rollback()
	var oldHash string
	var prior RecipeImportResult
	var remappedJSON string
	err = tx.QueryRowContext(ctx, `SELECT request_hash,workspace_revision,recipes_applied,recipes_skipped,remapped_ids_json FROM portability_recipe_import_operations WHERE workspace_id=? AND idempotency_key=?`, workspaceID, idempotencyKey).Scan(&oldHash, &prior.WorkspaceRevision, &prior.RecipesApplied, &prior.RecipesSkipped, &remappedJSON)
	if err == nil {
		if oldHash != requestHash {
			return RecipeImportResult{}, errors.New("recipe import idempotency key was reused with different content")
		}
		_ = json.Unmarshal([]byte(remappedJSON), &prior.RemappedIDs)
		return prior, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return RecipeImportResult{}, err
	}
	var actualRevision int64
	if err := tx.QueryRowContext(ctx, `SELECT revision FROM workspaces WHERE id=?`, workspaceID).Scan(&actualRevision); err != nil {
		return RecipeImportResult{}, err
	}
	if actualRevision != expectedRevision {
		return RecipeImportResult{}, ErrRestoreRevision{workspaceID, expectedRevision, actualRevision}
	}
	seen := map[string]bool{}
	result := RecipeImportResult{}
	now := r.clock.Now().UTC().Format(time.RFC3339Nano)
	for _, incoming := range envelope.Recipes {
		if seen[incoming.ID] {
			result.RecipesSkipped++
			continue
		}
		seen[incoming.ID] = true
		current, found, err := loadPortableRecipe(ctx, tx, workspaceID, incoming.ID)
		if err != nil {
			return RecipeImportResult{}, err
		}
		if found && canonicalRecipe(current) == canonicalRecipe(incoming) {
			result.RecipesSkipped++
			continue
		}
		item := incoming
		if found {
			switch policy {
			case "keep_existing":
				result.RecipesSkipped++
				continue
			case "copy_incoming":
				item.ID = uuid.NewString()
				result.RemappedIDs = append(result.RemappedIDs, incoming.ID+"="+item.ID)
			case "replace":
				item.Revision = current.Revision + 1
			}
		}
		if err := insertPortableRecipe(ctx, tx, workspaceID, item, now, found && policy == "replace"); err != nil {
			return RecipeImportResult{}, err
		}
		result.RecipesApplied++
	}
	result.WorkspaceRevision = actualRevision + 1
	if _, err := tx.ExecContext(ctx, `UPDATE workspaces SET revision=?,updated_at=? WHERE id=? AND revision=?`, result.WorkspaceRevision, now, workspaceID, actualRevision); err != nil {
		return RecipeImportResult{}, err
	}
	remapped, _ := json.Marshal(result.RemappedIDs)
	if _, err := tx.ExecContext(ctx, `INSERT INTO portability_recipe_import_operations(workspace_id,idempotency_key,request_hash,workspace_revision,recipes_applied,recipes_skipped,remapped_ids_json,created_at) VALUES(?,?,?,?,?,?,?,?)`, workspaceID, idempotencyKey, requestHash, result.WorkspaceRevision, result.RecipesApplied, result.RecipesSkipped, string(remapped), now); err != nil {
		return RecipeImportResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return RecipeImportResult{}, err
	}
	return result, nil
}

func ImportRecipesMust(envelope Export) (Export, error) {
	data, err := json.Marshal(envelope)
	if err != nil {
		return Export{}, err
	}
	return ImportRecipes(data)
}

type storedRecipe struct {
	ID, Name, Notes, SourceURL, SourceType, OriginalText, Status                                        string
	Revision                                                                                            int64
	MethodsJSON, GroupsJSON, AppliancesJSON, EvidenceJSON, CanonicalYield, ServingUnit, IngredientsJSON string
}

func (s storedRecipe) Portable() (Recipe, error) {
	item := Recipe{ID: s.ID, Revision: s.Revision, Name: s.Name, Notes: s.Notes, SourceURL: s.SourceURL, SourceType: s.SourceType, OriginalText: s.OriginalText, Status: s.Status, CanonicalYield: s.CanonicalYield, ServingUnit: s.ServingUnit}
	if err := json.Unmarshal([]byte(s.MethodsJSON), &item.Methods); err != nil {
		return Recipe{}, err
	}
	if err := json.Unmarshal([]byte(s.GroupsJSON), &item.Groups); err != nil {
		return Recipe{}, err
	}
	if err := json.Unmarshal([]byte(s.AppliancesJSON), &item.RequiredAppliances); err != nil {
		return Recipe{}, err
	}
	if err := json.Unmarshal([]byte(s.EvidenceJSON), &item.AllergenEvidence); err != nil {
		return Recipe{}, err
	}
	if err := json.Unmarshal([]byte(s.IngredientsJSON), &item.Ingredients); err != nil {
		return Recipe{}, err
	}
	return item, nil
}

func loadPortableRecipe(ctx context.Context, tx *sql.Tx, workspaceID, id string) (Recipe, bool, error) {
	var s storedRecipe
	err := tx.QueryRowContext(ctx, `SELECT v.recipe_id,v.name,v.notes,v.source_url,v.source_type,v.original_text,v.status,v.revision,v.methods_json,v.groups_json,v.required_appliances_json,v.allergen_evidence_json,v.canonical_yield,v.serving_unit,v.ingredients_json FROM recipe_revisions v JOIN recipes r ON r.id=v.recipe_id AND r.current_revision=v.revision WHERE r.id=? AND r.workspace_id=?`, id, workspaceID).Scan(&s.ID, &s.Name, &s.Notes, &s.SourceURL, &s.SourceType, &s.OriginalText, &s.Status, &s.Revision, &s.MethodsJSON, &s.GroupsJSON, &s.AppliancesJSON, &s.EvidenceJSON, &s.CanonicalYield, &s.ServingUnit, &s.IngredientsJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return Recipe{}, false, nil
	}
	if err != nil {
		return Recipe{}, false, err
	}
	item, err := s.Portable()
	return item, true, err
}

func insertPortableRecipe(ctx context.Context, tx *sql.Tx, workspaceID string, item Recipe, now string, replace bool) error {
	methods, _ := json.Marshal(item.Methods)
	groups, _ := json.Marshal(item.Groups)
	appliances, _ := json.Marshal(item.RequiredAppliances)
	evidence, _ := json.Marshal(item.AllergenEvidence)
	ingredients, _ := json.Marshal(item.Ingredients)
	if replace {
		_, err := tx.ExecContext(ctx, `INSERT INTO recipe_revisions(recipe_id,revision,workspace_id,name,notes,source_url,source_type,original_text,status,methods_json,groups_json,required_appliances_json,allergen_evidence_json,canonical_yield,serving_unit,ingredients_json,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, item.ID, item.Revision, workspaceID, item.Name, item.Notes, item.SourceURL, item.SourceType, item.OriginalText, item.Status, string(methods), string(groups), string(appliances), string(evidence), item.CanonicalYield, item.ServingUnit, string(ingredients), now)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE recipes SET current_revision=?,updated_at=? WHERE id=? AND workspace_id=?`, item.Revision, now, item.ID, workspaceID)
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO recipes(id,workspace_id,current_revision,created_at,updated_at) VALUES(?,?,?,?,?)`, item.ID, workspaceID, item.Revision, now, now); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO recipe_revisions(recipe_id,revision,workspace_id,name,notes,source_url,source_type,original_text,status,methods_json,groups_json,required_appliances_json,allergen_evidence_json,canonical_yield,serving_unit,ingredients_json,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, item.ID, item.Revision, workspaceID, item.Name, item.Notes, item.SourceURL, item.SourceType, item.OriginalText, item.Status, string(methods), string(groups), string(appliances), string(evidence), item.CanonicalYield, item.ServingUnit, string(ingredients), now)
	return err
}

func canonicalRecipe(item Recipe) string {
	b, _ := json.Marshal(item)
	return string(b)
}

func ImportWorkspaceMust(envelope Export) (Export, error) {
	data, err := json.Marshal(envelope)
	if err != nil {
		return Export{}, err
	}
	return ImportWorkspace(data)
}

func snapshotCurrent(ctx context.Context, tx *sql.Tx, workspaceID string) ([]Recipe, error) {
	rows, err := tx.QueryContext(ctx, `SELECT r.id,r.current_revision,v.name,v.notes,v.source_url,v.source_type,v.original_text,v.status,v.methods_json,v.groups_json,v.required_appliances_json,v.allergen_evidence_json,v.canonical_yield,v.serving_unit,v.ingredients_json FROM recipes r JOIN recipe_revisions v ON v.recipe_id=r.id AND v.revision=r.current_revision WHERE r.workspace_id=?`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Recipe
	for rows.Next() {
		var item Recipe
		var methods, groups, appliances, evidence, ingredientsJSON string
		if err := rows.Scan(&item.ID, &item.Revision, &item.Name, &item.Notes, &item.SourceURL, &item.SourceType, &item.OriginalText, &item.Status, &methods, &groups, &appliances, &evidence, &item.CanonicalYield, &item.ServingUnit, &ingredientsJSON); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(methods), &item.Methods); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(groups), &item.Groups); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(appliances), &item.RequiredAppliances); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(evidence), &item.AllergenEvidence); err != nil {
			return nil, err
		}
		if ingredientsJSON != "" {
			if err := json.Unmarshal([]byte(ingredientsJSON), &item.Ingredients); err != nil {
				return nil, err
			}
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func readPlan(ctx context.Context, tx *sql.Tx, workspaceID string) (int64, string, error) {
	var revision int64
	var plan string
	err := tx.QueryRowContext(ctx, `SELECT revision,plan_json FROM plans WHERE workspace_id=?`, workspaceID).Scan(&revision, &plan)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", nil
	}
	return revision, plan, err
}

func remapPlan(raw string, remap map[string]string) (string, error) {
	if raw == "" {
		return raw, nil
	}
	var draft planning.Draft
	if err := json.Unmarshal([]byte(raw), &draft); err != nil {
		return "", errors.New("imported plan is not valid JSON")
	}
	for i := range draft.Occurrences {
		if replacement := remap[draft.Occurrences[i].RecipeID]; replacement != "" {
			draft.Occurrences[i].RecipeID = replacement
		}
	}
	return stringMustJSON(draft)
}

func stringMustJSON(value any) (string, error) {
	b, err := json.Marshal(value)
	return string(b), err
}
