package workspaces

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
)

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
