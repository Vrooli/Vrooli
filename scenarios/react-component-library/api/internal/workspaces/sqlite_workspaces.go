package workspaces

import (
	"context"
	"fmt"
	"strings"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
)

func (r *SQLiteRepository) List(ctx context.Context, provenance workspacev1.WorkspaceProvenance, query string, limit int32) ([]*workspacev1.WorkspaceSummary, error) {
	clauses := make([]string, 0, 2)
	args := make([]any, 0, 4)
	if provenance != workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_UNSPECIFIED {
		stable, ok := provenanceToSQL(provenance)
		if !ok {
			return nil, &InvalidError{Issues: []*workspacev1.ValidationIssue{{Path: "provenance", Message: "provenance is not a known value"}}}
		}
		clauses = append(clauses, "w.provenance = ?")
		args = append(args, stable)
	}
	if search := strings.TrimSpace(query); search != "" {
		pattern := "%" + escapeLike(strings.ToLower(search)) + "%"
		clauses = append(clauses, "(LOWER(w.name) LIKE ? ESCAPE '\\' OR LOWER(w.summary) LIKE ? ESCAPE '\\' OR LOWER(w.tags_json) LIKE ? ESCAPE '\\')")
		args = append(args, pattern, pattern, pattern)
	}

	statement := `SELECT w.id, w.name, w.summary, w.provenance, w.tags_json,
			w.binding_scenario, w.binding_page, w.revision, w.created_at,
			w.updated_at, w.origin_workspace_id,
			(SELECT COUNT(*) FROM workspace_pages p WHERE p.workspace_id = w.id),
			(SELECT COUNT(*) FROM workspace_displays d
				JOIN workspace_pages p ON p.id = d.page_id
				WHERE p.workspace_id = w.id),
			(SELECT COUNT(*) FROM workspace_states s
				JOIN workspace_displays d ON d.id = s.display_id
				JOIN workspace_pages p ON p.id = d.page_id
				WHERE p.workspace_id = w.id)
		FROM workspaces w`
	if len(clauses) > 0 {
		statement += " WHERE " + strings.Join(clauses, " AND ")
	}
	statement += " ORDER BY w.updated_at DESC, w.rowid DESC LIMIT ?"
	args = append(args, listLimit(limit))

	rows, err := r.db.QueryContext(ctx, statement, args...)
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	defer rows.Close()

	summaries := make([]*workspacev1.WorkspaceSummary, 0)
	for rows.Next() {
		var (
			summary    workspacev1.WorkspaceSummary
			provenance string
			tagsJSON   string
			pageCount  int64
			displayCnt int64
			stateCount int64
		)
		if err := rows.Scan(
			&summary.Id, &summary.Name, &summary.Summary, &provenance, &tagsJSON,
			&summary.BindingScenario, &summary.BindingPage, &summary.Revision,
			&summary.CreatedAt, &summary.UpdatedAt, &summary.OriginWorkspaceId,
			&pageCount, &displayCnt, &stateCount,
		); err != nil {
			return nil, fmt.Errorf("scan workspace summary: %w", err)
		}
		decoded, err := provenanceFromSQL(provenance)
		if err != nil {
			return nil, fmt.Errorf("workspace %s: %w", summary.Id, err)
		}
		tags, err := parseTags(tagsJSON)
		if err != nil {
			return nil, fmt.Errorf("workspace %s: %w", summary.Id, err)
		}
		summary.Provenance = decoded
		summary.Tags = tags
		summary.PageCount = int32(pageCount)
		summary.DisplayCount = int32(displayCnt)
		summary.StateCount = int32(stateCount)
		summaries = append(summaries, &summary)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspaces: %w", err)
	}
	return summaries, nil
}

func (r *SQLiteRepository) Get(ctx context.Context, id string) (*workspacev1.Workspace, error) {
	return r.get(ctx, r.db, id)
}

func (r *SQLiteRepository) Create(ctx context.Context, req *workspacev1.CreateWorkspaceRequest) (*workspacev1.Workspace, error) {
	if req == nil {
		return nil, &InvalidError{Issues: []*workspacev1.ValidationIssue{{Path: "workspace", Message: "request is required"}}}
	}
	id, err := newID("ws_")
	if err != nil {
		return nil, fmt.Errorf("assign workspace id: %w", err)
	}
	now := r.timestamp()
	workspace := &workspacev1.Workspace{
		Id:              id,
		Name:            req.GetName(),
		Summary:         req.GetSummary(),
		Provenance:      req.GetProvenance(),
		Tags:            append([]string(nil), req.GetTags()...),
		BindingScenario: req.GetBindingScenario(),
		BindingPage:     req.GetBindingPage(),
		ThemeJson:       req.GetThemeJson(),
		OptionsJson:     req.GetOptionsJson(),
		Revision:        1,
		CreatedAt:       now,
		UpdatedAt:       now,
		Pages:           clonePages(req.GetPages()),
	}
	if len(workspace.Pages) == 0 {
		page, err := seedDefaultComposition()
		if err != nil {
			return nil, err
		}
		workspace.Pages = []*workspacev1.WorkspacePage{page}
	}
	if err := normalizeAggregate(workspace, true); err != nil {
		return nil, err
	}
	if err := invalidFromIssues(validationIssues(workspace)); err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("create workspace: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := insertWorkspace(ctx, tx, workspace); err != nil {
		if workspace.BindingScenario != "" && workspace.BindingPage != "" {
			var existing string
			lookupErr := tx.QueryRowContext(ctx, `SELECT id FROM workspaces
				WHERE binding_scenario = ? AND binding_page = ? LIMIT 1`,
				workspace.BindingScenario, workspace.BindingPage).Scan(&existing)
			if lookupErr == nil {
				return nil, fmt.Errorf("%w: %s", ErrBindingConflict, existing)
			}
		}
		return nil, err
	}
	if err := insertNested(ctx, tx, workspace); err != nil {
		return nil, err
	}
	snapshot, err := marshalWorkspace(workspace)
	if err != nil {
		return nil, err
	}
	if err := insertRevision(ctx, tx, workspace.Id, workspace.Revision, snapshot, "", now); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("create workspace commit: %w", err)
	}
	if err := r.record(ctx, "workspace.created", workspace.Id, nil, workspace); err != nil {
		return nil, err
	}
	return workspace, nil
}

func (r *SQLiteRepository) Save(ctx context.Context, req *workspacev1.SaveWorkspaceRequest) (*workspacev1.Workspace, error) {
	if req == nil || req.GetWorkspace() == nil {
		return nil, &InvalidError{Issues: []*workspacev1.ValidationIssue{{Path: "workspace", Message: "workspace is required"}}}
	}
	workspace := cloneWorkspace(req.GetWorkspace())
	if workspace.Id == "" {
		return nil, &InvalidError{Issues: []*workspacev1.ValidationIssue{{Path: "workspace.id", Message: "id is required"}}}
	}
	if req.GetBaseRevision() <= 0 {
		return nil, &InvalidError{Issues: []*workspacev1.ValidationIssue{{Path: "base_revision", Message: "base_revision is required"}}}
	}
	if err := normalizeAggregate(workspace, true); err != nil {
		return nil, err
	}
	if err := invalidFromIssues(validationIssues(workspace)); err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("save workspace: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	current, err := r.get(ctx, tx, workspace.Id)
	if err != nil {
		return nil, err
	}
	if current.GetProvenance() == workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_CURATED {
		return nil, ErrCurated
	}
	if current.Revision != req.GetBaseRevision() {
		return nil, &ConflictError{Current: current}
	}
	compositionChanged, err := workspaceCompositionChanged(current, workspace)
	if err != nil {
		return nil, err
	}
	if compositionChanged {
		invalidateWorkspaceCaptures(workspace)
		if _, err := tx.ExecContext(ctx, `DELETE FROM workspace_state_capture_evidence WHERE workspace_id = ?`, workspace.Id); err != nil {
			return nil, fmt.Errorf("invalidate workspace capture evidence: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM workspace_state_capture_attempts WHERE workspace_id = ?`, workspace.Id); err != nil {
			return nil, fmt.Errorf("invalidate workspace capture attempts: %w", err)
		}
	}

	now := r.timestamp()
	workspace.Revision = current.Revision + 1
	workspace.CreatedAt = current.CreatedAt
	workspace.UpdatedAt = now
	// Origin is fork metadata owned by Fork, not an editable client field. A
	// save that tried to clear or repoint it would rewrite provenance history.
	workspace.OriginWorkspaceId = current.OriginWorkspaceId

	provenance, ok := provenanceToSQL(workspace.Provenance)
	if !ok {
		return nil, &InvalidError{Issues: []*workspacev1.ValidationIssue{{Path: "provenance", Message: "provenance is not a known value"}}}
	}
	result, err := tx.ExecContext(ctx, `UPDATE workspaces
		SET name = ?, summary = ?, provenance = ?, tags_json = ?,
			binding_scenario = ?, binding_page = ?, theme_json = ?, options_json = ?,
			revision = ?, updated_at = ?
		WHERE id = ? AND revision = ?`,
		workspace.Name, workspace.Summary, provenance, tagsJSON(workspace.Tags),
		workspace.BindingScenario, workspace.BindingPage, workspace.ThemeJson, workspace.OptionsJson,
		workspace.Revision, workspace.UpdatedAt, workspace.Id, req.GetBaseRevision())
	if err != nil {
		if existing := conflictingBinding(ctx, tx, workspace.BindingScenario, workspace.BindingPage, workspace.Id); existing != "" {
			return nil, fmt.Errorf("%w: %s", ErrBindingConflict, existing)
		}
		return nil, fmt.Errorf("save workspace row: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("save workspace rows affected: %w", err)
	}
	if affected == 0 {
		// The compare-and-set is in SQL, so this only happens when another
		// writer won between the read above and this update.
		raced, err := r.get(ctx, tx, workspace.Id)
		if err != nil {
			return nil, err
		}
		return nil, &ConflictError{Current: raced}
	}

	if err := deleteNested(ctx, tx, workspace.Id); err != nil {
		return nil, err
	}
	if err := insertNested(ctx, tx, workspace); err != nil {
		return nil, err
	}
	saved, err := r.get(ctx, tx, workspace.Id)
	if err != nil {
		return nil, err
	}
	snapshot, err := marshalWorkspace(saved)
	if err != nil {
		return nil, err
	}
	if err := insertRevision(ctx, tx, saved.Id, saved.Revision, snapshot, strings.TrimSpace(req.GetActor()), now); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("save workspace commit: %w", err)
	}
	if err := r.record(ctx, "workspace.saved", saved.Id, current, saved); err != nil {
		return nil, err
	}
	return saved, nil
}

func conflictingBinding(ctx context.Context, q queryer, scenario, page, exceptID string) string {
	if scenario == "" || page == "" {
		return ""
	}
	var existing string
	if err := q.QueryRowContext(ctx, `SELECT id FROM workspaces
		WHERE binding_scenario = ? AND binding_page = ? AND id <> ? LIMIT 1`,
		scenario, page, exceptID).Scan(&existing); err != nil {
		return ""
	}
	return existing
}

func (r *SQLiteRepository) Delete(ctx context.Context, id string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("delete workspace: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	current, err := r.get(ctx, tx, id)
	if err != nil {
		return err
	}
	if current.GetProvenance() == workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_CURATED {
		return ErrCurated
	}
	// Source baselines protect their reference assets from independent deletes.
	// Remove the workspace-owned audit chain first so the aggregate cascade can
	// then remove all of its assets and captures atomically.
	if _, err := tx.ExecContext(ctx, `DELETE FROM workspace_source_visual_comparisons WHERE workspace_id = ?`, id); err != nil {
		return fmt.Errorf("delete workspace source visual comparisons: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM workspace_state_source_baseline_acceptances WHERE workspace_id = ?`, id); err != nil {
		return fmt.Errorf("delete workspace source baseline acceptances: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM workspace_state_source_baselines WHERE workspace_id = ?`, id); err != nil {
		return fmt.Errorf("delete workspace source baselines: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM workspaces WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete workspace %q: %w", id, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("delete workspace commit: %w", err)
	}
	return r.record(ctx, "workspace.deleted", id, current, nil)
}

func (r *SQLiteRepository) Fork(ctx context.Context, id, name string) (*workspacev1.Workspace, error) {
	source, err := r.get(ctx, r.db, id)
	if err != nil {
		return nil, err
	}
	newWorkspaceID, err := newID("ws_")
	if err != nil {
		return nil, fmt.Errorf("assign fork workspace id: %w", err)
	}
	now := r.timestamp()
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Copy of " + source.Name
	}
	fork := &workspacev1.Workspace{
		Id:         newWorkspaceID,
		Name:       name,
		Summary:    source.Summary,
		Provenance: workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_FORKED,
		Tags:       append([]string(nil), source.Tags...),
		// A fork is an independent composition and cannot take over the
		// experience page's singular route binding from its source workspace.
		BindingScenario:   "",
		BindingPage:       "",
		ThemeJson:         source.ThemeJson,
		OptionsJson:       source.OptionsJson,
		OriginWorkspaceId: source.Id,
		Revision:          1,
		CreatedAt:         now,
		UpdatedAt:         now,
		Pages:             reidentifyPages(source.Pages),
	}
	// Captures are workspace-scoped evidence. Fork the authored composition,
	// but do not carry a captured marker or capture id into a workspace whose
	// evidence ledger is empty.
	invalidateWorkspaceCaptures(fork)
	if err := normalizeAggregate(fork, true); err != nil {
		return nil, err
	}
	if err := invalidFromIssues(validationIssues(fork)); err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("fork workspace: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := insertWorkspace(ctx, tx, fork); err != nil {
		return nil, err
	}
	if err := insertNested(ctx, tx, fork); err != nil {
		return nil, err
	}
	snapshot, err := marshalWorkspace(fork)
	if err != nil {
		return nil, err
	}
	if err := insertRevision(ctx, tx, fork.Id, fork.Revision, snapshot, "", now); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("fork workspace commit: %w", err)
	}
	if err := r.record(ctx, "workspace.forked", fork.Id, nil, fork); err != nil {
		return nil, err
	}
	return fork, nil
}

func (r *SQLiteRepository) Revisions(ctx context.Context, id string, limit int32) ([]*workspacev1.WorkspaceRevision, error) {
	var exists int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM workspaces WHERE id = ?`, id).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check workspace %q: %w", id, err)
	}
	if exists == 0 {
		return nil, ErrNotFound
	}

	rows, err := r.db.QueryContext(ctx, `SELECT revision, snapshot_json, actor, created_at
		FROM workspace_revisions WHERE workspace_id = ?
		ORDER BY revision DESC LIMIT ?`, id, revisionLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list workspace revisions: %w", err)
	}
	defer rows.Close()

	revisions := make([]*workspacev1.WorkspaceRevision, 0)
	for rows.Next() {
		var revision workspacev1.WorkspaceRevision
		if err := rows.Scan(&revision.Revision, &revision.SnapshotJson, &revision.Actor, &revision.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan workspace revision: %w", err)
		}
		revisions = append(revisions, &revision)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspace revisions: %w", err)
	}
	return revisions, nil
}

// get loads one aggregate with a bounded number of queries: four, independent
// of page/display/state count. Nested rows are assembled in memory by parent
// id, so an aggregate read never becomes a query per node.
