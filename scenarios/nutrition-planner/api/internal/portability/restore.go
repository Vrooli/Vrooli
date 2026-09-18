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
}

type RestoreResult struct {
	WorkspaceRevision int64
	RecipesApplied    int
	CheckpointID      string
}

type Restorer interface {
	ApplyWorkspace(context.Context, string, int64, string, Export) (RestoreResult, error)
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
		if _, err := tx.ExecContext(ctx, `INSERT INTO recipes(id,workspace_id,current_revision,created_at,updated_at) VALUES(?,?,?,?,?)`, item.ID, workspaceID, item.Revision, now, now); err != nil {
			return RestoreResult{}, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO recipe_revisions(recipe_id,revision,workspace_id,name,notes,source_url,source_type,original_text,status,methods_json,groups_json,required_appliances_json,allergen_evidence_json,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, item.ID, item.Revision, workspaceID, item.Name, item.Notes, item.SourceURL, item.SourceType, item.OriginalText, item.Status, string(methods), string(groups), string(appliances), string(evidence), now); err != nil {
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

func ImportWorkspaceMust(envelope Export) (Export, error) {
	data, err := json.Marshal(envelope)
	if err != nil {
		return Export{}, err
	}
	return ImportWorkspace(data)
}

func snapshotCurrent(ctx context.Context, tx *sql.Tx, workspaceID string) ([]Recipe, error) {
	rows, err := tx.QueryContext(ctx, `SELECT r.id,r.current_revision,v.name,v.notes,v.source_url,v.source_type,v.original_text,v.status,v.methods_json,v.groups_json,v.required_appliances_json,v.allergen_evidence_json FROM recipes r JOIN recipe_revisions v ON v.recipe_id=r.id AND v.revision=r.current_revision WHERE r.workspace_id=?`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Recipe
	for rows.Next() {
		var item Recipe
		var methods, groups, appliances, evidence string
		if err := rows.Scan(&item.ID, &item.Revision, &item.Name, &item.Notes, &item.SourceURL, &item.SourceType, &item.OriginalText, &item.Status, &methods, &groups, &appliances, &evidence); err != nil {
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
