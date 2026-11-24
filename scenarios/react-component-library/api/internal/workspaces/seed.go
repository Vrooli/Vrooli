package workspaces

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
)

// SeedFile is the reviewed content that stocks the template shelf. It is JSON,
// not code, so a copy change does not need a build and a reviewer can diff the
// exact composition that will be offered.
type SeedFile struct {
	SchemaVersion int `json:"schemaVersion"`
	// SeedVersion gates the whole shelf. A boot applies the file only when this
	// is higher than the applied version, so a restart is a no-op and a
	// reviewed content change is an explicit version bump.
	SeedVersion int               `json:"seedVersion"`
	Workspaces  []json.RawMessage `json:"workspaces"`
}

// SeedResult reports what one seed pass did. Inserted + Skipped always equals
// the number of workspaces in the file; an error aborts the whole pass.
type SeedResult struct {
	Inserted int
	Updated  int
	Removed  int
	Skipped  int
}

// Seed inserts curated workspaces from a reviewed file. It is additive and
// idempotent: a workspace whose id already exists is left exactly as stored, so
// a seed edit never overwrites a workspace someone may already be looking at.
// Retiring a seed therefore requires deleting the row deliberately.
func Seed(ctx context.Context, db Database, raw []byte) (SeedResult, error) {
	var file SeedFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return SeedResult{}, fmt.Errorf("workspace seed is not valid JSON: %w", err)
	}
	if file.SchemaVersion != 1 {
		return SeedResult{}, fmt.Errorf("workspace seed schemaVersion %d is not supported", file.SchemaVersion)
	}
	if file.SeedVersion <= 0 {
		return SeedResult{}, fmt.Errorf("workspace seed must declare a positive seedVersion")
	}
	applied, err := appliedSeedVersion(ctx, db)
	if err != nil {
		return SeedResult{}, err
	}
	switch {
	case file.SeedVersion < applied:
		return SeedResult{}, fmt.Errorf("workspace seed version %d is older than the applied version %d", file.SeedVersion, applied)
	case file.SeedVersion == applied:
		return SeedResult{Skipped: len(file.Workspaces)}, nil
	}

	workspaces := make([]*workspacev1.Workspace, 0, len(file.Workspaces))
	ids := make(map[string]struct{}, len(file.Workspaces))
	for index, entry := range file.Workspaces {
		var workspace workspacev1.Workspace
		if err := protojson.Unmarshal(entry, &workspace); err != nil {
			return SeedResult{}, fmt.Errorf("workspace seed %d: %w", index, err)
		}
		if workspace.GetProvenance() != workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_CURATED {
			return SeedResult{}, fmt.Errorf("workspace seed %q must be curated, not %s",
				workspace.GetId(), workspace.GetProvenance())
		}
		if err := normalizeAggregate(&workspace, false); err != nil {
			return SeedResult{}, fmt.Errorf("workspace seed %q: %w", workspace.GetId(), err)
		}
		if err := invalidFromIssues(validationIssues(&workspace)); err != nil {
			return SeedResult{}, fmt.Errorf("workspace seed %q: %w", workspace.GetId(), err)
		}
		if workspace.GetId() == "" {
			return SeedResult{}, fmt.Errorf("workspace seed %d has no id", index)
		}
		if workspace.GetRevision() != 1 {
			return SeedResult{}, fmt.Errorf("workspace seed %q must start at revision 1", workspace.GetId())
		}
		if _, duplicate := ids[workspace.GetId()]; duplicate {
			return SeedResult{}, fmt.Errorf("workspace seed %q is declared twice", workspace.GetId())
		}
		ids[workspace.GetId()] = struct{}{}
		workspaces = append(workspaces, &workspace)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return SeedResult{}, fmt.Errorf("begin seed: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var result SeedResult
	for _, workspace := range workspaces {
		inserted, err := upsertSeed(ctx, tx, workspace)
		if err != nil {
			return result, fmt.Errorf("workspace seed %q: %w", workspace.GetId(), err)
		}
		if inserted {
			result.Inserted++
		} else {
			result.Updated++
		}
	}
	removed, err := retireUnlistedSeeds(ctx, tx, ids)
	if err != nil {
		return result, err
	}
	result.Removed = removed
	if _, err := tx.ExecContext(ctx, `INSERT INTO workspace_seed_state (id, version, applied_at)
		VALUES (1, ?, ?) ON CONFLICT(id) DO UPDATE SET version = excluded.version, applied_at = excluded.applied_at`,
		file.SeedVersion, nowTimestamp()); err != nil {
		return result, fmt.Errorf("record seed version: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("seed commit: %w", err)
	}
	return result, nil
}

func appliedSeedVersion(ctx context.Context, db Database) (int, error) {
	var version int
	err := db.QueryRowContext(ctx, `SELECT version FROM workspace_seed_state WHERE id = 1`).Scan(&version)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read seed version: %w", err)
	}
	return version, nil
}

// upsertSeed inserts a missing template or replaces the reviewed content of an
// existing one. A curated replacement bumps the revision and appends a snapshot,
// so the change has the same history as any other aggregate write. A non-curated
// row with the same id is a collision and aborts the pass.
func upsertSeed(ctx context.Context, tx *sql.Tx, workspace *workspacev1.Workspace) (bool, error) {
	var (
		provenance string
		revision   int32
	)
	err := tx.QueryRowContext(ctx,
		`SELECT provenance, revision FROM workspaces WHERE id = ?`, workspace.GetId()).Scan(&provenance, &revision)
	switch {
	case err == sql.ErrNoRows:
		if err := insertWorkspace(ctx, tx, workspace); err != nil {
			return false, err
		}
		if err := insertNested(ctx, tx, workspace); err != nil {
			return false, err
		}
		snapshot, err := marshalWorkspace(workspace)
		if err != nil {
			return false, err
		}
		if err := insertRevision(ctx, tx, workspace.GetId(), workspace.GetRevision(), snapshot, "seed", workspace.GetUpdatedAt()); err != nil {
			return false, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO workspace_migrations
			(source, source_id, workspace_id, migrated_at) VALUES ('seed', ?, ?, ?)`,
			workspace.GetId(), workspace.GetId(), nowTimestamp()); err != nil {
			return false, fmt.Errorf("record seed receipt: %w", err)
		}
		return true, nil
	case err != nil:
		return false, fmt.Errorf("inspect seeded workspace: %w", err)
	}
	if provenance != provenanceCurated {
		return false, fmt.Errorf("id %q is already used by a %s workspace", workspace.GetId(), provenance)
	}

	workspace.Revision = revision + 1
	if err := tx.QueryRowContext(ctx,
		`SELECT created_at FROM workspaces WHERE id = ?`, workspace.GetId()).Scan(&workspace.CreatedAt); err != nil {
		return false, fmt.Errorf("read seeded created_at: %w", err)
	}
	provenanceSQL, ok := provenanceToSQL(workspace.GetProvenance())
	if !ok {
		return false, fmt.Errorf("provenance is not a known value")
	}
	if _, err := tx.ExecContext(ctx, `UPDATE workspaces
		SET name = ?, summary = ?, provenance = ?, tags_json = ?, binding_scenario = ?,
			binding_page = ?, theme_json = ?, options_json = ?, revision = ?, updated_at = ?
		WHERE id = ?`,
		workspace.GetName(), workspace.GetSummary(), provenanceSQL, tagsJSON(workspace.GetTags()),
		workspace.GetBindingScenario(), workspace.GetBindingPage(), workspace.GetThemeJson(),
		workspace.GetOptionsJson(), workspace.GetRevision(), workspace.GetUpdatedAt(), workspace.GetId()); err != nil {
		return false, fmt.Errorf("update seeded workspace: %w", err)
	}
	if err := deleteNested(ctx, tx, workspace.GetId()); err != nil {
		return false, err
	}
	if err := insertNested(ctx, tx, workspace); err != nil {
		return false, err
	}
	snapshot, err := marshalWorkspace(workspace)
	if err != nil {
		return false, err
	}
	if err := insertRevision(ctx, tx, workspace.GetId(), workspace.GetRevision(), snapshot, "seed", workspace.GetUpdatedAt()); err != nil {
		return false, err
	}
	return false, nil
}

// retireUnlistedSeeds removes curated templates that the reviewed file no
// longer lists. Only a version bump reaches here, so deleting a shelf entry is
// an explicit reviewed change rather than a restart side effect.
func retireUnlistedSeeds(ctx context.Context, tx *sql.Tx, listed map[string]struct{}) (int, error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT id FROM workspaces WHERE provenance = ?`, provenanceCurated)
	if err != nil {
		return 0, fmt.Errorf("list curated workspaces: %w", err)
	}
	retired := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan curated workspace: %w", err)
		}
		if _, keep := listed[id]; !keep {
			retired = append(retired, id)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("iterate curated workspaces: %w", err)
	}
	rows.Close()
	for _, id := range retired {
		if _, err := tx.ExecContext(ctx, `DELETE FROM workspaces WHERE id = ?`, id); err != nil {
			return 0, fmt.Errorf("retire seeded workspace %q: %w", id, err)
		}
	}
	return len(retired), nil
}
