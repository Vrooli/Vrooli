package workspaces

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
)

// timestampLayout is fixed-width UTC so lexical ordering matches time order
// and the value matches the JavaScript ISO shape already stored by the studio.
const timestampLayout = "2006-01-02T15:04:05.000Z"

const (
	defaultListLimit     = 100
	maxListLimit         = 500
	defaultRevisionLimit = 50
	maxRevisionLimit     = 200
)

// queryer is the subset shared by *sql.DB and *sql.Tx. Reads run through the
// pool; aggregate reads inside a save run through the transaction that owns
// the write.
type queryer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// SQLiteRepository is the SQLite-backed workspace aggregate store.
type SQLiteRepository struct {
	db       *sql.DB
	now      func() time.Time
	recorder Recorder
}

var _ Repository = (*SQLiteRepository)(nil)

// NewSQLiteRepository returns the workspace repository over the scenario's
// primary pool. Callers that need activity wiring chain WithRecorder.
func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db, now: time.Now}
}

// WithRecorder attaches the activity seam. A nil recorder leaves activity
// logging disabled without changing any mutation behaviour.
func (r *SQLiteRepository) WithRecorder(recorder Recorder) *SQLiteRepository {
	r.recorder = recorder
	return r
}

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
		Id:                newWorkspaceID,
		Name:              name,
		Summary:           source.Summary,
		Provenance:        workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_FORKED,
		Tags:              append([]string(nil), source.Tags...),
		BindingScenario:   source.BindingScenario,
		BindingPage:       source.BindingPage,
		ThemeJson:         source.ThemeJson,
		OptionsJson:       source.OptionsJson,
		OriginWorkspaceId: source.Id,
		Revision:          1,
		CreatedAt:         now,
		UpdatedAt:         now,
		Pages:             reidentifyPages(source.Pages),
	}
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
func (r *SQLiteRepository) get(ctx context.Context, q queryer, id string) (*workspacev1.Workspace, error) {
	var (
		workspace  workspacev1.Workspace
		provenance string
		tags       string
	)
	err := q.QueryRowContext(ctx, `SELECT id, name, summary, provenance, tags_json,
			binding_scenario, binding_page, theme_json, options_json,
			origin_workspace_id, revision, created_at, updated_at
		FROM workspaces WHERE id = ?`, id).Scan(
		&workspace.Id, &workspace.Name, &workspace.Summary, &provenance, &tags,
		&workspace.BindingScenario, &workspace.BindingPage, &workspace.ThemeJson,
		&workspace.OptionsJson, &workspace.OriginWorkspaceId, &workspace.Revision,
		&workspace.CreatedAt, &workspace.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get workspace %q: %w", id, err)
	}
	decodedProvenance, err := provenanceFromSQL(provenance)
	if err != nil {
		return nil, fmt.Errorf("workspace %s: %w", id, err)
	}
	decodedTags, err := parseTags(tags)
	if err != nil {
		return nil, fmt.Errorf("workspace %s: %w", id, err)
	}
	workspace.Provenance = decodedProvenance
	workspace.Tags = decodedTags

	pages, err := r.loadPages(ctx, q, id)
	if err != nil {
		return nil, err
	}
	workspace.Pages = pages
	return &workspace, nil
}

func (r *SQLiteRepository) loadPages(ctx context.Context, q queryer, workspaceID string) ([]*workspacev1.WorkspacePage, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, name, route, page_type_id, sort_order, spec_json
		FROM workspace_pages WHERE workspace_id = ? ORDER BY sort_order, id`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("load workspace pages: %w", err)
	}
	defer rows.Close()

	pages := make([]*workspacev1.WorkspacePage, 0)
	byID := make(map[string]*workspacev1.WorkspacePage)
	for rows.Next() {
		page := &workspacev1.WorkspacePage{}
		if err := rows.Scan(&page.Id, &page.Name, &page.Route, &page.PageTypeId, &page.SortOrder, &page.SpecJson); err != nil {
			return nil, fmt.Errorf("scan workspace page: %w", err)
		}
		pages = append(pages, page)
		byID[page.Id] = page
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspace pages: %w", err)
	}

	// ORDER BY sort_order, id keeps each page's displays ascending even though
	// the scan is workspace-wide; grouping is by parent id only.
	displayRows, err := q.QueryContext(ctx, `SELECT d.id, d.page_id, d.device, d.width, d.height, d.sort_order
		FROM workspace_displays d
		JOIN workspace_pages p ON p.id = d.page_id
		WHERE p.workspace_id = ? ORDER BY d.sort_order, d.id`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("load workspace displays: %w", err)
	}
	defer displayRows.Close()

	displaysByID := make(map[string]*workspacev1.WorkspaceDisplay)
	displayParent := make(map[string]string)
	for displayRows.Next() {
		var (
			display workspacev1.WorkspaceDisplay
			pageID  string
			device  string
		)
		if err := displayRows.Scan(&display.Id, &pageID, &device, &display.Width, &display.Height, &display.SortOrder); err != nil {
			return nil, fmt.Errorf("scan workspace display: %w", err)
		}
		decodedDevice, err := deviceFromSQL(device)
		if err != nil {
			return nil, fmt.Errorf("display %s: %w", display.Id, err)
		}
		display.Device = decodedDevice
		page, ok := byID[pageID]
		if !ok {
			return nil, fmt.Errorf("display %s references unknown page %q", display.Id, pageID)
		}
		page.Displays = append(page.Displays, &display)
		displaysByID[display.Id] = &display
		displayParent[display.Id] = pageID
	}
	if err := displayRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspace displays: %w", err)
	}

	stateRows, err := q.QueryContext(ctx, `SELECT s.id, s.display_id, s.name, s.args_json,
			s.scenario_data_json, s.status, s.capture_id, s.sort_order
		FROM workspace_states s
		JOIN workspace_displays d ON d.id = s.display_id
		JOIN workspace_pages p ON p.id = d.page_id
		WHERE p.workspace_id = ? ORDER BY s.sort_order, s.id`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("load workspace states: %w", err)
	}
	defer stateRows.Close()

	for stateRows.Next() {
		var (
			state     workspacev1.WorkspaceState
			displayID string
			status    string
		)
		if err := stateRows.Scan(&state.Id, &displayID, &state.Name, &state.ArgsJson,
			&state.ScenarioDataJson, &status, &state.CaptureId, &state.SortOrder); err != nil {
			return nil, fmt.Errorf("scan workspace state: %w", err)
		}
		decodedStatus, err := statusFromSQL(status)
		if err != nil {
			return nil, fmt.Errorf("state %s: %w", state.Id, err)
		}
		state.Status = decodedStatus
		display, ok := displaysByID[displayID]
		if !ok {
			return nil, fmt.Errorf("state %s references unknown display %q", state.Id, displayID)
		}
		display.States = append(display.States, &state)
	}
	if err := stateRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspace states: %w", err)
	}
	return pages, nil
}

func insertWorkspace(ctx context.Context, q queryer, workspace *workspacev1.Workspace) error {
	provenance, ok := provenanceToSQL(workspace.Provenance)
	if !ok {
		return fmt.Errorf("insert workspace %q: unsupported provenance %s", workspace.Id, workspace.Provenance)
	}
	_, err := q.ExecContext(ctx, `INSERT INTO workspaces
		(id, name, summary, provenance, tags_json, binding_scenario, binding_page,
		 theme_json, options_json, origin_workspace_id, revision, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		workspace.Id, workspace.Name, workspace.Summary, provenance,
		tagsJSON(workspace.Tags), workspace.BindingScenario, workspace.BindingPage,
		workspace.ThemeJson, workspace.OptionsJson, workspace.OriginWorkspaceId,
		workspace.Revision, workspace.CreatedAt, workspace.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert workspace %q: %w", workspace.Id, err)
	}
	return nil
}

func insertNested(ctx context.Context, q queryer, workspace *workspacev1.Workspace) error {
	for pageIndex, page := range workspace.Pages {
		if page == nil {
			return fmt.Errorf("insert workspace %q: page %d is nil", workspace.Id, pageIndex)
		}
		if _, err := q.ExecContext(ctx, `INSERT INTO workspace_pages
			(id, workspace_id, name, route, page_type_id, sort_order, spec_json)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			page.Id, workspace.Id, page.Name, page.Route, page.PageTypeId, page.SortOrder, page.SpecJson); err != nil {
			return fmt.Errorf("insert workspace page %q: %w", page.Id, err)
		}
		for displayIndex, display := range page.Displays {
			if display == nil {
				return fmt.Errorf("insert page %q: display %d is nil", page.Id, displayIndex)
			}
			device, ok := deviceToSQL(display.Device)
			if !ok {
				return fmt.Errorf("insert display %q: unsupported device %s", display.Id, display.Device)
			}
			if _, err := q.ExecContext(ctx, `INSERT INTO workspace_displays
				(id, page_id, device, width, height, sort_order)
				VALUES (?, ?, ?, ?, ?, ?)`,
				display.Id, page.Id, device, display.Width, display.Height, display.SortOrder); err != nil {
				return fmt.Errorf("insert workspace display %q: %w", display.Id, err)
			}
			for stateIndex, state := range display.States {
				if state == nil {
					return fmt.Errorf("insert display %q: state %d is nil", display.Id, stateIndex)
				}
				status, ok := statusToSQL(state.Status)
				if !ok {
					return fmt.Errorf("insert state %q: unsupported status %s", state.Id, state.Status)
				}
				if _, err := q.ExecContext(ctx, `INSERT INTO workspace_states
					(id, display_id, name, args_json, scenario_data_json, status, capture_id, sort_order)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
					state.Id, display.Id, state.Name, state.ArgsJson, state.ScenarioDataJson,
					status, state.CaptureId, state.SortOrder); err != nil {
					return fmt.Errorf("insert workspace state %q: %w", state.Id, err)
				}
			}
		}
	}
	return nil
}

func deleteNested(ctx context.Context, q queryer, workspaceID string) error {
	// Displays and states cascade from pages, so one delete clears the subtree.
	if _, err := q.ExecContext(ctx, `DELETE FROM workspace_pages WHERE workspace_id = ?`, workspaceID); err != nil {
		return fmt.Errorf("delete workspace pages: %w", err)
	}
	return nil
}

func insertRevision(ctx context.Context, q queryer, workspaceID string, revision int32, snapshot []byte, actor, createdAt string) error {
	id, err := newID("rv_")
	if err != nil {
		return fmt.Errorf("assign revision id: %w", err)
	}
	if _, err := q.ExecContext(ctx, `INSERT INTO workspace_revisions
		(id, workspace_id, revision, snapshot_json, actor, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		id, workspaceID, revision, string(snapshot), actor, createdAt); err != nil {
		return fmt.Errorf("insert workspace revision %d: %w", revision, err)
	}
	return nil
}

func (r *SQLiteRepository) record(ctx context.Context, action, subjectID string, before, after *workspacev1.Workspace) error {
	if r.recorder == nil {
		return nil
	}
	beforeJSON, err := marshalWorkspace(before)
	if err != nil {
		return err
	}
	afterJSON, err := marshalWorkspace(after)
	if err != nil {
		return err
	}
	if err := r.recorder.Record(ctx, action, subjectID, beforeJSON, afterJSON); err != nil {
		// The mutation is already committed. Reporting this as a failed write
		// would be a lie that invites a duplicate retry, so the message states
		// the committed state explicitly and the handler logs it as internal.
		return fmt.Errorf("record %s for workspace %s after commit: %w", action, subjectID, err)
	}
	return nil
}

// marshalWorkspace is the one snapshot encoding. Snake_case proto names keep
// revision snapshots readable and stable if this package's Go types change.
func marshalWorkspace(workspace *workspacev1.Workspace) ([]byte, error) {
	if workspace == nil {
		return nil, nil
	}
	encoded, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(workspace)
	if err != nil {
		return nil, fmt.Errorf("encode workspace snapshot: %w", err)
	}
	return encoded, nil
}

func (r *SQLiteRepository) timestamp() string {
	now := r.now
	if now == nil {
		now = time.Now
	}
	return now().UTC().Format(timestampLayout)
}

// nowTimestamp is the migration path's clock: the importer has no repository
// instance and must not depend on one just to stamp a receipt.
func nowTimestamp() string {
	return time.Now().UTC().Format(timestampLayout)
}

func listLimit(limit int32) int32 {
	switch {
	case limit <= 0:
		return defaultListLimit
	case limit > maxListLimit:
		return maxListLimit
	default:
		return limit
	}
}

func revisionLimit(limit int32) int32 {
	switch {
	case limit <= 0:
		return defaultRevisionLimit
	case limit > maxRevisionLimit:
		return maxRevisionLimit
	default:
		return limit
	}
}

func tagsJSON(tags []string) string {
	if len(tags) == 0 {
		return "[]"
	}
	encoded, err := json.Marshal(tags)
	if err != nil {
		// A []string cannot fail to marshal; returning [] keeps the column
		// valid rather than writing an empty string.
		return "[]"
	}
	return string(encoded)
}

func parseTags(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return []string{}, nil
	}
	var tags []string
	if err := json.Unmarshal([]byte(raw), &tags); err != nil {
		return nil, fmt.Errorf("decode tags_json: %w", err)
	}
	if tags == nil {
		tags = []string{}
	}
	return tags, nil
}

// escapeLike makes a user's substring a literal. Without this, a query of
// "100%" would match every name and "_" would match any character.
func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	value = strings.ReplaceAll(value, `_`, `\_`)
	return value
}

func cloneWorkspace(workspace *workspacev1.Workspace) *workspacev1.Workspace {
	if workspace == nil {
		return nil
	}
	return proto.Clone(workspace).(*workspacev1.Workspace)
}

func clonePages(pages []*workspacev1.WorkspacePage) []*workspacev1.WorkspacePage {
	out := make([]*workspacev1.WorkspacePage, 0, len(pages))
	for _, page := range pages {
		if page == nil {
			out = append(out, nil)
			continue
		}
		out = append(out, proto.Clone(page).(*workspacev1.WorkspacePage))
	}
	return out
}

// reidentifyPages deep-copies the source composition with fresh page, display
// and state ids. A fork must not share nested identity with its origin: ids are
// URLs and foreign keys, and a captured state on the fork must not overwrite
// the origin's evidence.
func reidentifyPages(pages []*workspacev1.WorkspacePage) []*workspacev1.WorkspacePage {
	out := make([]*workspacev1.WorkspacePage, 0, len(pages))
	for _, sourcePage := range pages {
		if sourcePage == nil {
			out = append(out, nil)
			continue
		}
		page := proto.Clone(sourcePage).(*workspacev1.WorkspacePage)
		page.Id = "" // assigned by normalizeAggregate
		for _, display := range page.Displays {
			if display == nil {
				continue
			}
			display.Id = ""
			for _, state := range display.States {
				if state == nil {
					continue
				}
				state.Id = ""
			}
		}
		out = append(out, page)
	}
	return out
}
