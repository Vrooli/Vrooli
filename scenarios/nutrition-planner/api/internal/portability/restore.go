package portability

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"
	"nutrition-planner/internal/decimalx"
	"nutrition-planner/internal/nutrition"
	"nutrition-planner/internal/planning"
	"nutrition-planner/internal/profile"
	"nutrition-planner/internal/shopping"
	"nutrition-planner/internal/supplement"
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

type RestoreCheckpoint struct {
	ID, CreatedAt   string
	RestoreRevision int64
	RecipeCount     int
	PlanIncluded    bool
	Omissions       []string
}

type CheckpointRecoveryResult struct {
	RestoreResult
	PlanRestored bool
	Omissions    []string
}

type Restorer interface {
	ApplyWorkspace(context.Context, string, int64, string, Export) (RestoreResult, error)
	GetRestoreCheckpoint(context.Context, string, string) (RestoreCheckpoint, error)
	RecoverRestoreCheckpoint(context.Context, string, string, int64, string) (CheckpointRecoveryResult, error)
	PreviewRecipes(context.Context, string, Export) (RecipeImportPreview, error)
	ApplyRecipes(context.Context, string, int64, string, string, Export) (RecipeImportResult, error)
}

var workspaceRestoreOmissions = []string{"catalog", "inventory", "prices", "feedback", "jobs", "provider_credentials", "authentication", "entitlements"}

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

type ErrRestoreCheckpointNotFound struct{ WorkspaceID, CheckpointID string }

func (e ErrRestoreCheckpointNotFound) Error() string { return "restore checkpoint not found" }

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
	previousHistory, err := snapshotRecipeHistory(ctx, tx, workspaceID)
	if err != nil {
		return RestoreResult{}, err
	}
	previousPlanRevision, previousPlan, err := readPlan(ctx, tx, workspaceID)
	if err != nil {
		return RestoreResult{}, err
	}
	previousShopping, err := snapshotShoppingState(ctx, tx, workspaceID)
	if err != nil {
		return RestoreResult{}, err
	}
	checkpointID := uuid.NewString()
	previousTargets, err := snapshotNutritionTargets(ctx, tx, workspaceID)
	if err != nil {
		return RestoreResult{}, err
	}
	previousProfile, err := snapshotProfile(ctx, tx, workspaceID, r.clock)
	if err != nil {
		return RestoreResult{}, err
	}
	var checkpointProfile *profileRecord
	if previousProfile != nil {
		portable := portableProfile(*previousProfile)
		checkpointProfile = &portable
	}
	previousIntake, err := snapshotIntakeEvents(ctx, tx, workspaceID)
	if err != nil {
		return RestoreResult{}, err
	}
	previousSchedules, err := snapshotSupplementSchedules(ctx, tx, workspaceID)
	if err != nil {
		return RestoreResult{}, err
	}
	previousJSON, err := json.Marshal(map[string]any{"recipes": previous, "recipeHistory": previousHistory, "planRevision": previousPlanRevision, "planJson": previousPlan, "shopping": previousShopping, "nutritionTargets": previousTargets, "profile": checkpointProfile, "intakeEvents": previousIntake, "supplementSchedules": previousSchedules})
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
	history, current, err := portableRecipeHistory(envelope)
	if err != nil {
		return RestoreResult{}, err
	}
	ids := make([]string, 0, len(history))
	for id := range history {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		versions := history[id]
		if len(versions) == 0 {
			continue
		}
		if current[id] < 1 {
			return RestoreResult{}, fmt.Errorf("recipe %q has no current revision", id)
		}
		foundCurrent := false
		seenVersions := map[int64]bool{}
		for _, item := range versions {
			if seenVersions[item.Revision] {
				return RestoreResult{}, fmt.Errorf("duplicate imported recipe revision %s:%d", id, item.Revision)
			}
			seenVersions[item.Revision] = true
			if item.Revision == current[id] {
				foundCurrent = true
			}
		}
		if !foundCurrent {
			return RestoreResult{}, fmt.Errorf("recipe %q current revision %d is missing", id, current[id])
		}
		var otherWorkspace string
		if err := tx.QueryRowContext(ctx, `SELECT workspace_id FROM recipes WHERE id=?`, id).Scan(&otherWorkspace); err == nil && otherWorkspace != workspaceID {
			remap[id] = uuid.NewString()
		}
		mappedID := id
		if remap[id] != "" {
			mappedID = remap[id]
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO recipes(id,workspace_id,current_revision,created_at,updated_at) VALUES(?,?,?,?,?)`, mappedID, workspaceID, current[id], now, now); err != nil {
			return RestoreResult{}, err
		}
		for _, item := range versions {
			item.ID = mappedID
			if err := insertPortableRevision(ctx, tx, workspaceID, item, now); err != nil {
				return RestoreResult{}, err
			}
		}
	}

	var planRecord *Record
	for i := range envelope.Records {
		if envelope.Records[i].Kind == "plan" {
			planRecord = &envelope.Records[i]
			break
		}
	}
	if err := replaceNutritionTargets(ctx, tx, workspaceID, envelope); err != nil {
		return RestoreResult{}, err
	}
	if err := replaceProfile(ctx, tx, workspaceID, envelope, now); err != nil {
		return RestoreResult{}, err
	}
	if err := replaceIntakeEvents(ctx, tx, workspaceID, envelope, remap); err != nil {
		return RestoreResult{}, err
	}
	if err := replaceSupplementSchedules(ctx, tx, workspaceID, envelope); err != nil {
		return RestoreResult{}, err
	}
	if planRecord == nil {
		if err := planning.DeletePlanTx(ctx, tx, workspaceID); err != nil {
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
		if err := planning.ReplacePlanTx(ctx, tx, workspaceID, previousPlanRevision+1, planJSON, now); err != nil {
			return RestoreResult{}, err
		}
	}
	for i := range envelope.Records {
		if envelope.Records[i].Kind == "shopping_state" {
			var state shopping.PersistedState
			if err := json.Unmarshal(envelope.Records[i].Data, &state); err != nil {
				return RestoreResult{}, err
			}
			if err := replaceShoppingState(ctx, tx, workspaceID, state, now); err != nil {
				return RestoreResult{}, err
			}
			break
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
	return RestoreResult{WorkspaceRevision: nextWorkspaceRevision, RecipesApplied: len(history), CheckpointID: checkpointID}, nil
}

func (r *sqliteRestorer) GetRestoreCheckpoint(ctx context.Context, workspaceID, checkpointID string) (RestoreCheckpoint, error) {
	if workspaceID == "" || checkpointID == "" {
		return RestoreCheckpoint{}, errors.New("workspace_id and checkpoint_id are required")
	}
	var raw, createdAt string
	var revision int64
	err := r.db.QueryRowContext(ctx, `SELECT previous_state_json,created_at,expected_revision FROM portability_restore_checkpoints WHERE workspace_id=? AND id=?`, workspaceID, checkpointID).Scan(&raw, &createdAt, &revision)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RestoreCheckpoint{}, ErrRestoreCheckpointNotFound{workspaceID, checkpointID}
		}
		return RestoreCheckpoint{}, err
	}
	var previous struct {
		Recipes          []Recipe            `json:"recipes"`
		PlanRevision     int64               `json:"planRevision"`
		PlanJSON         string              `json:"planJson"`
		NutritionTargets []nutrition.Target  `json:"nutritionTargets"`
		Profile          *profileRecord      `json:"profile"`
		IntakeEvents     []intakeEventRecord `json:"intakeEvents"`
	}
	if err := json.Unmarshal([]byte(raw), &previous); err != nil {
		return RestoreCheckpoint{}, fmt.Errorf("invalid saved checkpoint: %w", err)
	}
	return RestoreCheckpoint{ID: checkpointID, CreatedAt: createdAt, RestoreRevision: revision, RecipeCount: len(previous.Recipes), PlanIncluded: previous.PlanJSON != "", Omissions: append([]string(nil), workspaceRestoreOmissions...)}, nil
}

func (r *sqliteRestorer) RecoverRestoreCheckpoint(ctx context.Context, workspaceID, checkpointID string, expectedRevision int64, idempotencyKey string) (CheckpointRecoveryResult, error) {
	if workspaceID == "" || checkpointID == "" || idempotencyKey == "" {
		return CheckpointRecoveryResult{}, errors.New("workspace, checkpoint, and idempotency key are required")
	}
	var raw string
	var checkpointRevision int64
	if err := r.db.QueryRowContext(ctx, `SELECT previous_state_json,expected_revision FROM portability_restore_checkpoints WHERE workspace_id=? AND id=?`, workspaceID, checkpointID).Scan(&raw, &checkpointRevision); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CheckpointRecoveryResult{}, ErrRestoreCheckpointNotFound{workspaceID, checkpointID}
		}
		return CheckpointRecoveryResult{}, err
	}
	var previous struct {
		Recipes             []Recipe                 `json:"recipes"`
		RecipeHistory       []Recipe                 `json:"recipeHistory"`
		PlanRevision        int64                    `json:"planRevision"`
		PlanJSON            string                   `json:"planJson"`
		Shopping            *shopping.PersistedState `json:"shopping"`
		NutritionTargets    []nutrition.Target       `json:"nutritionTargets"`
		Profile             *profileRecord           `json:"profile"`
		IntakeEvents        []intakeEventRecord      `json:"intakeEvents"`
		SupplementSchedules []supplement.Schedule    `json:"supplementSchedules"`
	}
	if err := json.Unmarshal([]byte(raw), &previous); err != nil {
		return CheckpointRecoveryResult{}, fmt.Errorf("invalid saved checkpoint: %w", err)
	}
	envelope := Export{Format: "daily.workspace", SchemaVersion: 2, Scope: Scope{Kind: "workspace_backup"}, Manifest: Manifest{AttachmentsIncluded: false, Omissions: append([]string(nil), workspaceRestoreOmissions...), RecordKinds: []string{"workspace", "recipe", "recipe_revision", "plan", "profile", "intake_event"}}}
	workspaceData, _ := json.Marshal(map[string]any{"revision": checkpointRevision})
	envelope.Records = append(envelope.Records, Record{Kind: "workspace", ID: workspaceID, Revision: checkpointRevision, Data: workspaceData})
	for _, item := range previous.Recipes {
		identity, _ := json.Marshal(map[string]any{"currentRevision": item.Revision, "status": item.Status})
		envelope.Records = append(envelope.Records, Record{Kind: "recipe", ID: item.ID, Revision: item.Revision, Data: identity})
	}
	if len(previous.RecipeHistory) == 0 { // Checkpoints created before history was retained contain current revisions only.
		previous.RecipeHistory = previous.Recipes
	}
	for _, item := range previous.RecipeHistory {
		revision, err := json.Marshal(item)
		if err != nil {
			return CheckpointRecoveryResult{}, err
		}
		envelope.Records = append(envelope.Records, Record{Kind: "recipe_revision", ID: item.ID, Revision: item.Revision, Data: revision})
	}
	if previous.PlanJSON != "" {
		data, _ := json.Marshal(map[string]any{"revision": previous.PlanRevision, "planJson": previous.PlanJSON})
		envelope.Records = append(envelope.Records, Record{Kind: "plan", ID: workspaceID, Revision: previous.PlanRevision, Data: data})
	}
	if previous.Profile != nil {
		item := previous.Profile.domain()
		item.WorkspaceID = workspaceID
		data, err := json.Marshal(item)
		if err != nil {
			return CheckpointRecoveryResult{}, err
		}
		envelope.Records = append(envelope.Records, Record{Kind: "profile", ID: workspaceID, Revision: item.Revision, Data: data})
	}
	envelope.Manifest.RecordKinds = append(envelope.Manifest.RecordKinds, "shopping_state", "nutrition_target")
	if previous.Shopping != nil {
		shoppingData, err := json.Marshal(previous.Shopping)
		if err != nil {
			return CheckpointRecoveryResult{}, err
		}
		envelope.Records = append(envelope.Records, Record{Kind: "shopping_state", ID: workspaceID, Data: shoppingData})
	}
	for _, target := range previous.NutritionTargets {
		target.WorkspaceID = workspaceID
		data, err := json.Marshal(target)
		if err != nil {
			return CheckpointRecoveryResult{}, err
		}
		envelope.Records = append(envelope.Records, Record{Kind: "nutrition_target", ID: target.ID, Revision: target.Revision, Data: data})
	}
	for _, event := range previous.IntakeEvents {
		data, err := json.Marshal(event)
		if err != nil {
			return CheckpointRecoveryResult{}, err
		}
		envelope.Records = append(envelope.Records, Record{Kind: "intake_event", ID: event.EventID, Data: data})
	}
	envelope.Manifest.RecordKinds = append(envelope.Manifest.RecordKinds, "supplement_schedule", "supplement_schedule_revision")
	for _, item := range previous.SupplementSchedules {
		data, err := json.Marshal(portableSupplementSchedule(item))
		if err != nil {
			return CheckpointRecoveryResult{}, err
		}
		envelope.Records = append(envelope.Records, Record{Kind: "supplement_schedule_revision", ID: item.ID, Revision: item.Revision, Data: data})
	}
	currentSchedules := map[string]supplement.Schedule{}
	for _, item := range previous.SupplementSchedules {
		if old, ok := currentSchedules[item.ID]; !ok || item.Revision > old.Revision {
			currentSchedules[item.ID] = item
		}
	}
	for id, item := range currentSchedules {
		data, err := json.Marshal(supplementScheduleRecord{WorkspaceID: workspaceID, CurrentRevision: item.Revision})
		if err != nil {
			return CheckpointRecoveryResult{}, err
		}
		envelope.Records = append(envelope.Records, Record{Kind: "supplement_schedule", ID: id, Revision: item.Revision, Data: data})
	}
	envelope.Manifest.RecordCount = len(envelope.Records)
	result, err := r.ApplyWorkspace(ctx, workspaceID, expectedRevision, idempotencyKey, envelope)
	if err != nil {
		return CheckpointRecoveryResult{}, err
	}
	return CheckpointRecoveryResult{RestoreResult: result, PlanRestored: previous.PlanJSON != "", Omissions: append([]string(nil), workspaceRestoreOmissions...)}, nil
}

func snapshotNutritionTargets(ctx context.Context, tx *sql.Tx, workspaceID string) ([]nutrition.Target, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id,revision,workspace_id,nutrient_id,lower_bound,upper_bound,period,scope,enforcement,provenance,effective_from,effective_to,active FROM nutrition_targets WHERE workspace_id=? ORDER BY id,revision`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []nutrition.Target{}
	for rows.Next() {
		var target nutrition.Target
		var lower, upper, from string
		var to sql.NullString
		var active int
		if err := rows.Scan(&target.ID, &target.Revision, &target.WorkspaceID, &target.NutrientID, &lower, &upper, &target.Period, &target.Scope, &target.Enforcement, &target.Provenance, &from, &to, &active); err != nil {
			return nil, err
		}
		var parseErr error
		target.Lower, parseErr = decimalx.Parse(lower)
		if parseErr != nil {
			return nil, parseErr
		}
		target.Upper, parseErr = decimalx.Parse(upper)
		if parseErr != nil {
			return nil, parseErr
		}
		target.EffectiveFrom, parseErr = time.Parse(time.RFC3339Nano, from)
		if parseErr != nil {
			return nil, parseErr
		}
		if to.Valid {
			end, err := time.Parse(time.RFC3339Nano, to.String)
			if err != nil {
				return nil, err
			}
			target.EffectiveTo = &end
		}
		target.Active = active != 0
		out = append(out, target)
	}
	return out, rows.Err()
}

func snapshotIntakeEvents(ctx context.Context, tx *sql.Tx, workspaceID string) ([]intakeEventRecord, error) {
	rows, err := tx.QueryContext(ctx, `SELECT event_id,event_date,recipe_id,recipe_revision,nutrient_id,amount,unit,reason,correction_of,recorded_at FROM nutrition_intake_events WHERE workspace_id=? ORDER BY event_date,recorded_at,event_id`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []intakeEventRecord{}
	for rows.Next() {
		var item intakeEventRecord
		if err := rows.Scan(&item.EventID, &item.Date, &item.RecipeID, &item.RecipeRevision, &item.NutrientID, &item.Amount, &item.Unit, &item.Reason, &item.CorrectionOf, &item.RecordedAt); err != nil {
			return nil, err
		}
		item.WorkspaceID = workspaceID
		out = append(out, item)
	}
	return out, rows.Err()
}

func snapshotSupplementSchedules(ctx context.Context, tx *sql.Tx, workspaceID string) ([]supplement.Schedule, error) {
	rows, err := tx.QueryContext(ctx, `SELECT v.schedule_id,v.revision,v.workspace_id,v.product_revision_id,v.dose,v.dose_unit,v.weekdays_json,v.start_date,v.end_date,v.paused,v.confirmed,v.created_at FROM supplement_schedule_revisions v JOIN supplement_schedules s ON s.id=v.schedule_id WHERE s.workspace_id=? ORDER BY v.schedule_id,v.revision`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []supplement.Schedule
	for rows.Next() {
		var item supplement.Schedule
		var dose, weekdays, created string
		var paused, confirmed int
		if err := rows.Scan(&item.ID, &item.Revision, &item.WorkspaceID, &item.ProductRevisionID, &dose, &item.DoseUnit, &weekdays, &item.StartDate, &item.EndDate, &paused, &confirmed, &created); err != nil {
			return nil, err
		}
		item.Dose, err = decimalx.Parse(dose)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(weekdays), &item.Weekdays); err != nil {
			return nil, err
		}
		item.Paused, item.Confirmed = paused != 0, confirmed != 0
		item.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func replaceSupplementSchedules(ctx context.Context, tx *sql.Tx, workspaceID string, envelope Export) error {
	if !containsString(envelope.Manifest.RecordKinds, "supplement_schedule") {
		return nil
	}
	if !containsString(envelope.Manifest.RecordKinds, "supplement_schedule_revision") {
		return errors.New("supplement schedule manifest is missing revision history")
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM supplement_schedule_revisions WHERE workspace_id=?`, workspaceID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM supplement_schedules WHERE workspace_id=?`, workspaceID); err != nil {
		return err
	}
	ids := map[string]string{}
	for _, record := range envelope.Records {
		if record.Kind != "supplement_schedule" {
			continue
		}
		var owner string
		err := tx.QueryRowContext(ctx, `SELECT workspace_id FROM supplement_schedules WHERE id=? LIMIT 1`, record.ID).Scan(&owner)
		if err == nil && owner != workspaceID {
			ids[record.ID] = uuid.NewString()
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}
	versions := map[string][]supplement.Schedule{}
	current := map[string]int64{}
	created := map[string]string{}
	for _, record := range envelope.Records {
		if record.Kind == "supplement_schedule" {
			var pointer supplementScheduleRecord
			if err := json.Unmarshal(record.Data, &pointer); err != nil {
				return err
			}
			current[record.ID] = pointer.CurrentRevision
		}
		if record.Kind != "supplement_schedule_revision" {
			continue
		}
		var data supplementScheduleRevisionRecord
		if err := json.Unmarshal(record.Data, &data); err != nil {
			return err
		}
		dose, err := decimalx.Parse(data.Dose)
		if err != nil {
			return err
		}
		at, err := time.Parse(time.RFC3339Nano, data.CreatedAt)
		if err != nil {
			return err
		}
		item := supplement.Schedule{ID: record.ID, Revision: record.Revision, WorkspaceID: workspaceID, ProductRevisionID: data.ProductRevisionID, Dose: dose, DoseUnit: data.DoseUnit, Weekdays: data.Weekdays, StartDate: data.StartDate, EndDate: data.EndDate, Paused: data.Paused, Confirmed: data.Confirmed, CreatedAt: at}
		mappedID := ids[record.ID]
		if mappedID == "" {
			mappedID = record.ID
		}
		item.ID = mappedID
		versions[record.ID] = append(versions[record.ID], item)
		if item.Revision == 1 {
			created[record.ID] = at.UTC().Format(time.RFC3339Nano)
		}
	}
	for originalID, revision := range current {
		mappedID := ids[originalID]
		if mappedID == "" {
			mappedID = originalID
		}
		updatedAt := ""
		for _, item := range versions[originalID] {
			if item.Revision == revision {
				updatedAt = item.CreatedAt.UTC().Format(time.RFC3339Nano)
			}
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO supplement_schedules(id,workspace_id,current_revision,created_at,updated_at) VALUES(?,?,?,?,?)`, mappedID, workspaceID, revision, created[originalID], updatedAt); err != nil {
			return err
		}
		for _, item := range versions[originalID] {
			weekdays, err := json.Marshal(item.Weekdays)
			if err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO supplement_schedule_revisions(schedule_id,revision,workspace_id,product_revision_id,dose,dose_unit,weekdays_json,start_date,end_date,paused,confirmed,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, mappedID, item.Revision, workspaceID, item.ProductRevisionID, item.Dose.String(), item.DoseUnit, string(weekdays), item.StartDate, item.EndDate, portableBoolInt(item.Paused), portableBoolInt(item.Confirmed), item.CreatedAt.UTC().Format(time.RFC3339Nano)); err != nil {
				return err
			}
		}
	}
	return nil
}

func portableBoolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func replaceIntakeEvents(ctx context.Context, tx *sql.Tx, workspaceID string, envelope Export, remap map[string]string) error {
	declared := false
	for _, kind := range envelope.Manifest.RecordKinds {
		if kind == "intake_event" {
			declared = true
			break
		}
	}
	if !declared {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM nutrition_intake_events WHERE workspace_id=?`, workspaceID); err != nil {
		return err
	}
	for _, record := range envelope.Records {
		if record.Kind != "intake_event" {
			continue
		}
		var item intakeEventRecord
		if err := json.Unmarshal(record.Data, &item); err != nil {
			return err
		}
		if mapped := remap[item.RecipeID]; mapped != "" {
			item.RecipeID = mapped
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO nutrition_intake_events(workspace_id,event_id,event_date,recipe_id,recipe_revision,nutrient_id,amount,unit,reason,correction_of,recorded_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, workspaceID, item.EventID, item.Date, item.RecipeID, item.RecipeRevision, item.NutrientID, item.Amount, item.Unit, item.Reason, item.CorrectionOf, item.RecordedAt); err != nil {
			return err
		}
	}
	return nil
}

func snapshotProfile(ctx context.Context, tx *sql.Tx, workspaceID string, clock schedule.Clock) (*profile.Profile, error) {
	item, err := profile.NewSQLiteRepository(tx, clock).Get(ctx, workspaceID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func replaceProfile(ctx context.Context, tx *sql.Tx, workspaceID string, envelope Export, now string) error {
	if !containsString(envelope.Manifest.RecordKinds, "profile") {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM profiles WHERE workspace_id=?`, workspaceID); err != nil {
		return err
	}
	for _, record := range envelope.Records {
		if record.Kind != "profile" {
			continue
		}
		var data profileRecord
		if err := json.Unmarshal(record.Data, &data); err != nil {
			return err
		}
		item := data.domain()
		item.WorkspaceID = workspaceID // Imported identity is data, never restore authority.
		encode := func(values []string) (string, error) { b, err := json.Marshal(values); return string(b), err }
		rules, err := encode(item.ActiveRules)
		if err != nil {
			return err
		}
		excluded, err := encode(item.ExcludedGroups)
		if err != nil {
			return err
		}
		allergies, err := encode(item.Allergies)
		if err != nil {
			return err
		}
		appliances, err := encode(item.Appliances)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO profiles(workspace_id,revision,preset,preset_version,active_rules_json,excluded_groups_json,allergies_json,appliances_json,cost_weight,effort_weight,variety_weight,draft_json,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`, item.WorkspaceID, item.Revision, item.Preset, item.PresetVersion, rules, excluded, allergies, appliances, strconv.FormatFloat(item.CostWeight, 'f', -1, 64), strconv.FormatFloat(item.EffortWeight, 'f', -1, 64), strconv.FormatFloat(item.VarietyWeight, 'f', -1, 64), item.DraftJSON, now)
		return err
	}
	return nil // A declared family with no record means the destination uses defaults.
}

func replaceNutritionTargets(ctx context.Context, tx *sql.Tx, workspaceID string, envelope Export) error {
	if !containsString(envelope.Manifest.RecordKinds, "nutrition_target") {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM nutrition_targets WHERE workspace_id=?`, workspaceID); err != nil {
		return err
	}
	remapped := map[string]string{}
	for _, record := range envelope.Records {
		if record.Kind != "nutrition_target" {
			continue
		}
		var target nutrition.Target
		if err := json.Unmarshal(record.Data, &target); err != nil {
			return err
		}
		target.WorkspaceID = workspaceID
		if mapped := remapped[target.ID]; mapped != "" {
			target.ID = mapped
		} else {
			var owner string
			err := tx.QueryRowContext(ctx, `SELECT workspace_id FROM nutrition_targets WHERE id=? LIMIT 1`, target.ID).Scan(&owner)
			if err == nil && owner != workspaceID {
				remapped[target.ID] = uuid.NewString()
				target.ID = remapped[target.ID]
			} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return err
			}
		}
		var effectiveTo any
		if target.EffectiveTo != nil {
			effectiveTo = target.EffectiveTo.UTC().Format(time.RFC3339Nano)
		}
		active := 0
		if target.Active {
			active = 1
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO nutrition_targets(id,revision,workspace_id,nutrient_id,lower_bound,upper_bound,period,scope,enforcement,provenance,effective_from,effective_to,active) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`, target.ID, target.Revision, workspaceID, target.NutrientID, target.Lower.String(), target.Upper.String(), target.Period, target.Scope, target.Enforcement, target.Provenance, target.EffectiveFrom.UTC().Format(time.RFC3339Nano), effectiveTo, active)
		if err != nil {
			return err
		}
	}
	return nil
}

func snapshotShoppingState(ctx context.Context, tx *sql.Tx, workspaceID string) (shopping.PersistedState, error) {
	state := shopping.PersistedState{Checks: map[string]bool{}, HaveThis: map[string]bool{}, Reviews: []shopping.PersistedReview{}}
	for _, q := range []struct {
		table, column string
		dst           map[string]bool
	}{{"shopping_checks", "checked", state.Checks}, {"shopping_have_this", "asserted", state.HaveThis}} {
		rows, err := tx.QueryContext(ctx, `SELECT line_key,`+q.column+` FROM `+q.table+` WHERE workspace_id=?`, workspaceID)
		if err != nil {
			return shopping.PersistedState{}, err
		}
		for rows.Next() {
			var key string
			var value int
			if err := rows.Scan(&key, &value); err != nil {
				rows.Close()
				return shopping.PersistedState{}, err
			}
			q.dst[key] = value != 0
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return shopping.PersistedState{}, err
		}
		if err := rows.Close(); err != nil {
			return shopping.PersistedState{}, err
		}
	}
	rows, err := tx.QueryContext(ctx, `SELECT review_id,payload_hash,created_at FROM shopping_purchase_reviews WHERE workspace_id=? ORDER BY review_id`, workspaceID)
	if err != nil {
		return shopping.PersistedState{}, err
	}
	for rows.Next() {
		var review shopping.PersistedReview
		if err := rows.Scan(&review.ID, &review.PayloadHash, &review.CreatedAt); err != nil {
			rows.Close()
			return shopping.PersistedState{}, err
		}
		lines, err := tx.QueryContext(ctx, `SELECT line_key,item_id,amount,unit,price,omitted FROM shopping_purchase_review_lines WHERE workspace_id=? AND review_id=? ORDER BY line_key`, workspaceID, review.ID)
		if err != nil {
			rows.Close()
			return shopping.PersistedState{}, err
		}
		for lines.Next() {
			var line shopping.PersistedLine
			var omitted int
			if err := lines.Scan(&line.Key, &line.ItemID, &line.Amount, &line.Unit, &line.Price, &omitted); err != nil {
				lines.Close()
				rows.Close()
				return shopping.PersistedState{}, err
			}
			line.Omitted = omitted != 0
			review.Lines = append(review.Lines, line)
		}
		if err := lines.Err(); err != nil {
			lines.Close()
			rows.Close()
			return shopping.PersistedState{}, err
		}
		if err := lines.Close(); err != nil {
			rows.Close()
			return shopping.PersistedState{}, err
		}
		state.Reviews = append(state.Reviews, review)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return shopping.PersistedState{}, err
	}
	if err := rows.Close(); err != nil {
		return shopping.PersistedState{}, err
	}
	return state, nil
}

// replaceShoppingState mutates only persisted checklist and review tables.
// Purchase review replay never calls ConfirmPurchases and cannot create inventory events.
func replaceShoppingState(ctx context.Context, tx *sql.Tx, workspaceID string, state shopping.PersistedState, now string) error {
	for _, table := range []string{"shopping_purchase_review_lines", "shopping_purchase_reviews", "shopping_checks", "shopping_have_this"} {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE workspace_id=?`, workspaceID); err != nil {
			return err
		}
	}
	for key, value := range state.Checks {
		n := 0
		if value {
			n = 1
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO shopping_checks(workspace_id,line_key,checked,updated_at) VALUES(?,?,?,?)`, workspaceID, key, n, now); err != nil {
			return err
		}
	}
	for key, value := range state.HaveThis {
		n := 0
		if value {
			n = 1
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO shopping_have_this(workspace_id,line_key,asserted,updated_at) VALUES(?,?,?,?)`, workspaceID, key, n, now); err != nil {
			return err
		}
	}
	for _, review := range state.Reviews {
		if _, err := tx.ExecContext(ctx, `INSERT INTO shopping_purchase_reviews(workspace_id,review_id,payload_hash,created_at) VALUES(?,?,?,?)`, workspaceID, review.ID, review.PayloadHash, review.CreatedAt); err != nil {
			return err
		}
		for _, line := range review.Lines {
			n := 0
			if line.Omitted {
				n = 1
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO shopping_purchase_review_lines(workspace_id,review_id,line_key,item_id,amount,unit,price,omitted) VALUES(?,?,?,?,?,?,?,?)`, workspaceID, review.ID, line.Key, line.ItemID, line.Amount, line.Unit, line.Price, n); err != nil {
				return err
			}
		}
	}
	return nil
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
		versionsByID, _, historyErr := portableRecipeHistory(envelope)
		if historyErr != nil {
			return RecipeImportResult{}, historyErr
		}
		versions := versionsByID[incoming.ID]
		if len(versions) == 0 {
			versions = []Recipe{incoming}
		}
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
		} else {
			var owner string
			err := tx.QueryRowContext(ctx, `SELECT workspace_id FROM recipes WHERE id=?`, incoming.ID).Scan(&owner)
			if err == nil && owner != workspaceID {
				item.ID = uuid.NewString()
				result.RemappedIDs = append(result.RemappedIDs, incoming.ID+"="+item.ID)
			} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return RecipeImportResult{}, err
			}
		}
		if found && policy == "replace" {
			if err := insertPortableRecipe(ctx, tx, workspaceID, item, now, true); err != nil {
				return RecipeImportResult{}, err
			}
		} else {
			for i := range versions {
				versions[i].ID = item.ID
			}
			if err := insertPortableHistory(ctx, tx, workspaceID, versions, item.Revision, now); err != nil {
				return RecipeImportResult{}, err
			}
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
	if replace {
		err := insertPortableRevision(ctx, tx, workspaceID, item, now)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE recipes SET current_revision=?,updated_at=? WHERE id=? AND workspace_id=?`, item.Revision, now, item.ID, workspaceID)
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO recipes(id,workspace_id,current_revision,created_at,updated_at) VALUES(?,?,?,?,?)`, item.ID, workspaceID, item.Revision, now, now); err != nil {
		return err
	}
	return insertPortableRevision(ctx, tx, workspaceID, item, now)
}

func insertPortableHistory(ctx context.Context, tx *sql.Tx, workspaceID string, versions []Recipe, current int64, now string) error {
	if len(versions) == 0 {
		return errors.New("recipe has no revisions")
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO recipes(id,workspace_id,current_revision,created_at,updated_at) VALUES(?,?,?,?,?)`, versions[0].ID, workspaceID, current, now, now); err != nil {
		return err
	}
	for _, item := range versions {
		if err := insertPortableRevision(ctx, tx, workspaceID, item, now); err != nil {
			return err
		}
	}
	return nil
}

func insertPortableRevision(ctx context.Context, tx *sql.Tx, workspaceID string, item Recipe, now string) error {
	methods, _ := json.Marshal(item.Methods)
	groups, _ := json.Marshal(item.Groups)
	appliances, _ := json.Marshal(item.RequiredAppliances)
	evidence, _ := json.Marshal(item.AllergenEvidence)
	ingredients, _ := json.Marshal(item.Ingredients)
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

func snapshotRecipeHistory(ctx context.Context, tx *sql.Tx, workspaceID string) ([]Recipe, error) {
	rows, err := tx.QueryContext(ctx, `SELECT v.recipe_id,v.revision,v.name,v.notes,v.source_url,v.source_type,v.original_text,v.status,v.methods_json,v.groups_json,v.required_appliances_json,v.allergen_evidence_json,v.canonical_yield,v.serving_unit,v.ingredients_json FROM recipe_revisions v JOIN recipes r ON r.id=v.recipe_id AND r.workspace_id=v.workspace_id WHERE v.workspace_id=? ORDER BY v.recipe_id,v.revision`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Recipe
	for rows.Next() {
		var item storedRecipe
		if err := rows.Scan(&item.ID, &item.Revision, &item.Name, &item.Notes, &item.SourceURL, &item.SourceType, &item.OriginalText, &item.Status, &item.MethodsJSON, &item.GroupsJSON, &item.AppliancesJSON, &item.EvidenceJSON, &item.CanonicalYield, &item.ServingUnit, &item.IngredientsJSON); err != nil {
			return nil, err
		}
		portable, err := item.Portable()
		if err != nil {
			return nil, err
		}
		out = append(out, portable)
	}
	return out, rows.Err()
}

func readPlan(ctx context.Context, tx *sql.Tx, workspaceID string) (int64, string, error) {
	return planning.ReadPlanTx(ctx, tx, workspaceID)
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
