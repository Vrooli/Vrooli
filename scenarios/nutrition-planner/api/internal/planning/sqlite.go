package planning

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/vrooli/api-core/schedule"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

type Repository interface {
	Apply(context.Context, string, int64, string) (int64, error)
	Get(context.Context, string) (int64, string, error)
}

type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
}

func NewSQLiteRepository(db SQLExecutor, clock schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clock}
}

type ErrStaleInputs struct {
	WorkspaceID      string
	Expected, Actual int64
}

func (e ErrStaleInputs) Error() string {
	return fmt.Sprintf("plan inputs are stale for workspace %q: expected revision %d, actual %d", e.WorkspaceID, e.Expected, e.Actual)
}

func (r *sqliteRepository) Get(ctx context.Context, id string) (int64, string, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return 0, "", err
	}
	defer tx.Rollback()
	draft, revision, err := readDraft(ctx, tx, id)
	if err != nil || revision == 0 {
		return revision, "", err
	}
	encoded, err := json.Marshal(draft)
	if err != nil {
		return 0, "", err
	}
	return revision, string(encoded), nil
}

func (r *sqliteRepository) Apply(ctx context.Context, id string, expected int64, raw string) (int64, error) {
	var draft Draft
	if err := json.Unmarshal([]byte(raw), &draft); err != nil {
		return 0, fmt.Errorf("invalid plan draft: %w", err)
	}
	canonicalizeDraft(&draft)
	canonical, err := json.Marshal(draft)
	if err != nil {
		return 0, err
	}
	payloadHash := fmt.Sprintf("%x", sha256.Sum256(canonical))
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var receiptRevision int64
	err = tx.QueryRowContext(ctx, `SELECT applied_revision FROM plan_apply_receipts WHERE workspace_id=? AND expected_revision=? AND payload_hash=?`, id, expected, payloadHash).Scan(&receiptRevision)
	if err == nil {
		return receiptRevision, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	current, actual, err := readDraft(ctx, tx, id)
	if err != nil {
		return 0, err
	}
	if actual != expected {
		return 0, ErrStaleInputs{id, expected, actual}
	}
	touched := make(map[string]struct{}, len(draft.Occurrences)+len(draft.Unresolved))
	for _, occurrence := range draft.Occurrences {
		touched[occurrence.Date] = struct{}{}
	}
	for _, unresolved := range draft.Unresolved {
		touched[unresolved.Date] = struct{}{}
	}
	if len(touched) == 0 {
		return 0, errors.New("plan draft must include an occurrence or an unresolved date")
	}
	next := expected + 1
	updated := r.clock.Now().UTC().Format(time.RFC3339Nano)
	metadata := draft
	metadata.Occurrences = nil
	metadata.Unresolved = nil
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return 0, err
	}
	mergedUnresolved := make([]Unresolved, 0, len(current.Unresolved)+len(draft.Unresolved))
	for _, unresolved := range current.Unresolved {
		if _, replace := touched[unresolved.Date]; !replace {
			mergedUnresolved = append(mergedUnresolved, unresolved)
		}
	}
	mergedUnresolved = append(mergedUnresolved, draft.Unresolved...)
	unresolvedJSON, err := json.Marshal(mergedUnresolved)
	if err != nil {
		return 0, err
	}
	if actual == 0 {
		_, err = tx.ExecContext(ctx, `INSERT INTO plan_state(workspace_id,revision,metadata_json,unresolved_json,updated_at) VALUES(?,?,?,?,?)`, id, next, string(metadataJSON), string(unresolvedJSON), updated)
	} else {
		result, updateErr := tx.ExecContext(ctx, `UPDATE plan_state SET revision=?,metadata_json=?,unresolved_json=?,updated_at=? WHERE workspace_id=? AND revision=?`, next, string(metadataJSON), string(unresolvedJSON), updated, id, expected)
		err = updateErr
		if err == nil {
			var count int64
			count, err = result.RowsAffected()
			if err == nil && count != 1 {
				return 0, ErrStaleInputs{id, expected, actual}
			}
		}
	}
	if err != nil {
		return 0, err
	}
	for date := range touched {
		if _, err = tx.ExecContext(ctx, `DELETE FROM plan_occurrences WHERE workspace_id=? AND date=?`, id, date); err != nil {
			return 0, err
		}
	}
	for _, occurrence := range draft.Occurrences {
		if occurrence.SlotName == "" {
			occurrence.SlotName = "meal"
		}
		locked := 0
		if occurrence.Locked {
			locked = 1
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO plan_occurrences(workspace_id,date,slot_name,mode,quantity,recipe_id,recipe_revision,recipe_name,reason,locked) VALUES(?,?,?,?,?,?,?,?,?,?)`, id, occurrence.Date, occurrence.SlotName, occurrence.Mode, occurrence.Quantity, occurrence.RecipeID, occurrence.RecipeRevision, occurrence.RecipeName, occurrence.Reason, locked); err != nil {
			return 0, err
		}
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO plan_apply_receipts(workspace_id,expected_revision,payload_hash,applied_revision) VALUES(?,?,?,?)`, id, expected, payloadHash, next); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return next, nil
}

func canonicalizeDraft(draft *Draft) {
	for i := range draft.Occurrences {
		if draft.Occurrences[i].SlotName == "" {
			draft.Occurrences[i].SlotName = "meal"
		}
	}
	sort.SliceStable(draft.Occurrences, func(i, j int) bool {
		if draft.Occurrences[i].Date == draft.Occurrences[j].Date {
			return draft.Occurrences[i].SlotName < draft.Occurrences[j].SlotName
		}
		return draft.Occurrences[i].Date < draft.Occurrences[j].Date
	})
	for i := range draft.Unresolved {
		sort.Strings(draft.Unresolved[i].CandidateIDs)
	}
	sort.SliceStable(draft.Unresolved, func(i, j int) bool {
		if draft.Unresolved[i].Date == draft.Unresolved[j].Date {
			return draft.Unresolved[i].Code < draft.Unresolved[j].Code
		}
		return draft.Unresolved[i].Date < draft.Unresolved[j].Date
	})
}

type rowQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func readDraft(ctx context.Context, q rowQuerier, id string) (Draft, int64, error) {
	var revision int64
	var metadataJSON, unresolvedJSON string
	err := q.QueryRowContext(ctx, `SELECT revision,metadata_json,unresolved_json FROM plan_state WHERE workspace_id=?`, id).Scan(&revision, &metadataJSON, &unresolvedJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return Draft{}, 0, nil
	}
	if err != nil {
		return Draft{}, 0, err
	}
	var draft Draft
	if err := json.Unmarshal([]byte(metadataJSON), &draft); err != nil {
		return Draft{}, 0, fmt.Errorf("decode plan metadata for workspace %q: %w", id, err)
	}
	if err := json.Unmarshal([]byte(unresolvedJSON), &draft.Unresolved); err != nil {
		return Draft{}, 0, fmt.Errorf("decode unresolved plan slots for workspace %q: %w", id, err)
	}
	rows, err := q.QueryContext(ctx, `SELECT date,slot_name,mode,quantity,recipe_id,recipe_revision,recipe_name,reason,locked FROM plan_occurrences WHERE workspace_id=? ORDER BY date,slot_name`, id)
	if err != nil {
		return Draft{}, 0, err
	}
	defer rows.Close()
	for rows.Next() {
		var occurrence Occurrence
		var locked int
		if err := rows.Scan(&occurrence.Date, &occurrence.SlotName, &occurrence.Mode, &occurrence.Quantity, &occurrence.RecipeID, &occurrence.RecipeRevision, &occurrence.RecipeName, &occurrence.Reason, &locked); err != nil {
			return Draft{}, 0, err
		}
		occurrence.Locked = locked != 0
		draft.Occurrences = append(draft.Occurrences, occurrence)
	}
	if err := rows.Err(); err != nil {
		return Draft{}, 0, err
	}
	if draft.Occurrences == nil {
		draft.Occurrences = []Occurrence{}
	}
	return draft, revision, nil
}

// ReadPlanTx returns the normalized plan from an existing transaction so
// restore and checkpoint code can share the planning persistence contract.
func ReadPlanTx(ctx context.Context, tx *sql.Tx, workspaceID string) (int64, string, error) {
	draft, revision, err := readDraft(ctx, tx, workspaceID)
	if err != nil || revision == 0 {
		return revision, "", err
	}
	encoded, err := json.Marshal(draft)
	return revision, string(encoded), err
}

// ReplacePlanTx replaces a complete workspace plan within the caller's
// transaction. It is reserved for workspace restore; normal edits use Apply
// so range preservation, conflict checks, and idempotency receipts are kept.
func ReplacePlanTx(ctx context.Context, tx *sql.Tx, workspaceID string, revision int64, raw, updatedAt string) error {
	var draft Draft
	if err := json.Unmarshal([]byte(raw), &draft); err != nil {
		return fmt.Errorf("invalid restored plan: %w", err)
	}
	canonicalizeDraft(&draft)
	metadata := draft
	metadata.Occurrences = nil
	metadata.Unresolved = nil
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	unresolvedJSON, err := json.Marshal(draft.Unresolved)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO plan_state(workspace_id,revision,metadata_json,unresolved_json,updated_at) VALUES(?,?,?,?,?) ON CONFLICT(workspace_id) DO UPDATE SET revision=excluded.revision,metadata_json=excluded.metadata_json,unresolved_json=excluded.unresolved_json,updated_at=excluded.updated_at`, workspaceID, revision, string(metadataJSON), string(unresolvedJSON), updatedAt); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM plan_occurrences WHERE workspace_id=?`, workspaceID); err != nil {
		return err
	}
	for _, occurrence := range draft.Occurrences {
		locked := 0
		if occurrence.Locked {
			locked = 1
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO plan_occurrences(workspace_id,date,slot_name,mode,quantity,recipe_id,recipe_revision,recipe_name,reason,locked) VALUES(?,?,?,?,?,?,?,?,?,?)`, workspaceID, occurrence.Date, occurrence.SlotName, occurrence.Mode, occurrence.Quantity, occurrence.RecipeID, occurrence.RecipeRevision, occurrence.RecipeName, occurrence.Reason, locked); err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `DELETE FROM plan_apply_receipts WHERE workspace_id=?`, workspaceID)
	return err
}

// DeletePlanTx removes the normalized state during an explicit workspace
// restore that contains no plan.
func DeletePlanTx(ctx context.Context, tx *sql.Tx, workspaceID string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM plan_occurrences WHERE workspace_id=?`, workspaceID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM plan_apply_receipts WHERE workspace_id=?`, workspaceID); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `DELETE FROM plan_state WHERE workspace_id=?`, workspaceID)
	return err
}
