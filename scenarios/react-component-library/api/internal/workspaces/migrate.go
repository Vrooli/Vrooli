package workspaces

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
)

// Migrate applies the workspaces schema and imports every legacy
// studio_designs row that has not been imported yet.
//
// It is deliberately additive and idempotent: the schema text is re-applied
// (every statement is IF NOT EXISTS) and workspace_migrations records each
// source row by (source, source_id), so a second boot is a no-op. Nothing is
// deleted here — the old table stays readable until the deletion ledger retires
// it, and that is also the rollback window for this import.
func Migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, schemaSQL); err != nil {
		return fmt.Errorf("apply workspaces schema: %w", err)
	}
	exists, err := tableExists(ctx, db, "studio_designs")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("migrate studio_designs: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	designs, err := loadLegacyDesigns(ctx, tx)
	if err != nil {
		return err
	}
	now := nowTimestamp()
	for _, design := range designs {
		done, err := migrationRecorded(ctx, tx, "studio_designs", design.ID)
		if err != nil {
			return err
		}
		if done {
			continue
		}
		if err := importLegacyDesign(ctx, tx, design, now); err != nil {
			return fmt.Errorf("migrate studio_designs %q: %w", design.ID, err)
		}
	}
	// Repairs an install that imported a design before this package owned
	// assets: the design receipt already exists, so the loop above skips it and
	// its uploads would otherwise never arrive. The insert is idempotent.
	if err := backfillLegacyAssets(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migrate studio_designs commit: %w", err)
	}
	return nil
}

// backfillLegacyAssets copies uploads for every already-imported design. It
// collects receipts before reading the asset table so it never holds two
// cursors on SQLite's single connection.
func backfillLegacyAssets(ctx context.Context, q queryer) error {
	exists, err := tableExists(ctx, q, "studio_design_assets")
	if err != nil || !exists {
		return err
	}
	rows, err := q.QueryContext(ctx,
		`SELECT source_id, workspace_id FROM workspace_migrations WHERE source = 'studio_designs'`)
	if err != nil {
		return fmt.Errorf("read design receipts: %w", err)
	}
	type receipt struct{ designID, workspaceID string }
	receipts := make([]receipt, 0)
	for rows.Next() {
		var entry receipt
		if err := rows.Scan(&entry.designID, &entry.workspaceID); err != nil {
			rows.Close()
			return fmt.Errorf("scan design receipt: %w", err)
		}
		receipts = append(receipts, entry)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate design receipts: %w", err)
	}
	rows.Close()
	for _, entry := range receipts {
		if err := importLegacyAssets(ctx, q, entry.designID, entry.workspaceID); err != nil {
			return err
		}
	}
	return nil
}

func tableExists(ctx context.Context, q queryer, table string) (bool, error) {
	var count int
	if err := q.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&count); err != nil {
		return false, fmt.Errorf("inspect table %q: %w", table, err)
	}
	return count > 0, nil
}

func tableColumns(ctx context.Context, q queryer, table string) (map[string]bool, error) {
	// The table name is a compile-time literal supplied by this package, never
	// user input; SQLite's pragma does not accept a bound identifier.
	rows, err := q.QueryContext(ctx, fmt.Sprintf("SELECT name FROM pragma_table_info('%s')", table))
	if err != nil {
		return nil, fmt.Errorf("inspect %s columns: %w", table, err)
	}
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan %s column: %w", table, err)
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate %s columns: %w", table, err)
	}
	return columns, nil
}

type legacyDesign struct {
	ID         string
	Name       string
	FramesJSON string
	ThemeJSON  string
	Options    string
	Scenario   string
	Page       string
	CreatedAt  string
	UpdatedAt  string
}

// loadLegacyDesigns reads only the columns that exist. Older studio_designs
// tables predate scenario/page/options_json; sqlite cannot reference a missing
// column, so the select list is assembled from the live shape instead of
// assuming the current one.
func loadLegacyDesigns(ctx context.Context, q queryer) ([]legacyDesign, error) {
	columns, err := tableColumns(ctx, q, "studio_designs")
	if err != nil {
		return nil, err
	}
	scenarioExpr := "''"
	if columns["scenario"] {
		scenarioExpr = "COALESCE(scenario, '')"
	}
	pageExpr := "''"
	if columns["page"] {
		pageExpr = "COALESCE(page, '')"
	}
	optionsExpr := "'{}'"
	if columns["options_json"] {
		optionsExpr = "COALESCE(options_json, '{}')"
	}
	statement := fmt.Sprintf(`SELECT id, name, COALESCE(frames_json, '[]'),
		COALESCE(theme_json, '{}'), COALESCE(created_at, ''), COALESCE(updated_at, ''),
		%s, %s, %s
		FROM studio_designs ORDER BY rowid`, scenarioExpr, pageExpr, optionsExpr)

	rows, err := q.QueryContext(ctx, statement)
	if err != nil {
		return nil, fmt.Errorf("read studio_designs: %w", err)
	}
	defer rows.Close()

	designs := make([]legacyDesign, 0)
	for rows.Next() {
		var design legacyDesign
		if err := rows.Scan(&design.ID, &design.Name, &design.FramesJSON, &design.ThemeJSON,
			&design.CreatedAt, &design.UpdatedAt, &design.Scenario, &design.Page, &design.Options); err != nil {
			return nil, fmt.Errorf("scan studio_designs: %w", err)
		}
		designs = append(designs, design)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate studio_designs: %w", err)
	}
	return designs, nil
}

func migrationRecorded(ctx context.Context, q queryer, source, sourceID string) (bool, error) {
	var count int
	if err := q.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM workspace_migrations WHERE source = ? AND source_id = ?`,
		source, sourceID).Scan(&count); err != nil {
		return false, fmt.Errorf("inspect migration record: %w", err)
	}
	return count > 0, nil
}

type legacyFrame struct {
	Device string          `json:"device"`
	Spec   json.RawMessage `json:"spec"`
	Width  int32           `json:"width"`
	Height int32           `json:"height"`
}

// importLegacyDesign maps the old multi-frame document onto the aggregate:
// one workspace, one page, one display per frame, and one default state per
// display. The frame spec is the only composition data the old store had, so
// it becomes both the page spec (first frame) and the default state's
// {"spec": …} args. Nothing else about a frame is invented.
func importLegacyDesign(ctx context.Context, q queryer, design legacyDesign, now string) error {
	name := strings.TrimSpace(design.Name)
	if name == "" {
		return fmt.Errorf("name is required")
	}
	frames, err := parseLegacyFrames(design.FramesJSON)
	if err != nil {
		return err
	}

	workspaceID, err := newID("ws_")
	if err != nil {
		return err
	}
	pageID, err := newID("pg_")
	if err != nil {
		return err
	}

	pageSpec := "{}"
	if len(frames) > 0 {
		if spec := strings.TrimSpace(string(frames[0].Spec)); spec != "" {
			pageSpec = spec
		}
	}

	displays := make([]*workspacev1.WorkspaceDisplay, 0, len(frames))
	for frameIndex, frame := range frames {
		device, ok := legacyDevice(frame.Device)
		if !ok {
			return fmt.Errorf("frame %d has unsupported device %q", frameIndex, frame.Device)
		}
		displayID, err := newID("dp_")
		if err != nil {
			return err
		}
		stateID, err := newID("st_")
		if err != nil {
			return err
		}
		width, height := frame.Width, frame.Height
		if width <= 0 || height <= 0 {
			defaultWidth, defaultHeight := deviceMetrics(device)
			if width <= 0 {
				width = defaultWidth
			}
			if height <= 0 {
				height = defaultHeight
			}
		}
		argsJSON, err := frameArgsJSON(frame.Spec)
		if err != nil {
			return fmt.Errorf("frame %d spec: %w", frameIndex, err)
		}
		displays = append(displays, &workspacev1.WorkspaceDisplay{
			Id:        displayID,
			Device:    device,
			Width:     width,
			Height:    height,
			SortOrder: int32(frameIndex),
			States: []*workspacev1.WorkspaceState{{
				Id:        stateID,
				Name:      "Default",
				ArgsJson:  argsJSON,
				Status:    workspacev1.StateStatus_STATE_STATUS_DRAFT,
				SortOrder: 0,
			}},
		})
	}

	createdAt := design.CreatedAt
	if strings.TrimSpace(createdAt) == "" {
		createdAt = now
	}
	updatedAt := design.UpdatedAt
	if strings.TrimSpace(updatedAt) == "" {
		updatedAt = createdAt
	}
	workspace := &workspacev1.Workspace{
		Id:              workspaceID,
		Name:            name,
		Provenance:      workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_IMPORTED,
		BindingScenario: strings.TrimSpace(design.Scenario),
		BindingPage:     strings.TrimSpace(design.Page),
		ThemeJson:       design.ThemeJSON,
		OptionsJson:     design.Options,
		Revision:        1,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
		Pages: []*workspacev1.WorkspacePage{{
			Id:        pageID,
			Name:      name,
			SortOrder: 0,
			SpecJson:  pageSpec,
			Displays:  displays,
		}},
	}
	if strings.TrimSpace(workspace.ThemeJson) == "" {
		workspace.ThemeJson = "{}"
	}
	if strings.TrimSpace(workspace.OptionsJson) == "" {
		workspace.OptionsJson = "{}"
	}
	if err := normalizeAggregate(workspace, false); err != nil {
		return err
	}
	if err := invalidFromIssues(validationIssues(workspace)); err != nil {
		return err
	}

	if err := insertWorkspace(ctx, q, workspace); err != nil {
		return err
	}
	if err := insertNested(ctx, q, workspace); err != nil {
		return err
	}
	snapshot, err := marshalWorkspace(workspace)
	if err != nil {
		return err
	}
	if err := insertRevision(ctx, q, workspace.Id, workspace.Revision, snapshot, "migration", updatedAt); err != nil {
		return err
	}
	if err := importLegacyAssets(ctx, q, design.ID, workspace.Id); err != nil {
		return err
	}
	if _, err := q.ExecContext(ctx, `INSERT INTO workspace_migrations
		(source, source_id, workspace_id, migrated_at) VALUES ('studio_designs', ?, ?, ?)`,
		design.ID, workspace.Id, now); err != nil {
		return fmt.Errorf("record migration: %w", err)
	}
	return nil
}

func parseLegacyFrames(raw string) ([]legacyFrame, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}
	var entries []json.RawMessage
	if err := json.Unmarshal([]byte(trimmed), &entries); err != nil {
		return nil, fmt.Errorf("frames_json is not an array: %w", err)
	}
	frames := make([]legacyFrame, 0, len(entries))
	for index, entry := range entries {
		var frame legacyFrame
		if err := json.Unmarshal(entry, &frame); err != nil {
			return nil, fmt.Errorf("frame %d is not an object: %w", index, err)
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

func legacyDevice(value string) (workspacev1.DisplayDevice, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case deviceDesktop:
		return workspacev1.DisplayDevice_DISPLAY_DEVICE_DESKTOP, true
	case deviceTablet:
		return workspacev1.DisplayDevice_DISPLAY_DEVICE_TABLET, true
	case devicePhone:
		return workspacev1.DisplayDevice_DISPLAY_DEVICE_PHONE, true
	default:
		return workspacev1.DisplayDevice_DISPLAY_DEVICE_UNSPECIFIED, false
	}
}

// frameArgsJSON wraps the frame spec without re-modelling it. RawMessage keeps
// the object byte-for-byte, so a frame whose spec carries keys this package has
// never heard of still survives the import.
func frameArgsJSON(spec json.RawMessage) (string, error) {
	args := map[string]json.RawMessage{"spec": spec}
	encoded, err := json.Marshal(args)
	if err != nil {
		return "", fmt.Errorf("encode args: %w", err)
	}
	return string(encoded), nil
}

// importLegacyAssets moves a legacy design's uploaded references beside its
// imported workspace. Ids are preserved so an in-flight studio tab that still
// holds a legacy asset URL keeps resolving after the switch. The insert is
// OR IGNORE because the import is idempotent by receipt: a rerun after a
// partial failure must not fail on the rows it already copied.
func importLegacyAssets(ctx context.Context, q queryer, designID, workspaceID string) error {
	exists, err := tableExists(ctx, q, "studio_design_assets")
	if err != nil || !exists {
		return err
	}
	columns, err := tableColumns(ctx, q, "studio_design_assets")
	if err != nil {
		return err
	}
	for _, required := range []string{"id", "name", "mime", "data", "created_at"} {
		if !columns[required] {
			// A shape this import does not understand is left alone rather than
			// half-copied; the legacy table still holds the bytes.
			return nil
		}
	}
	bytesExpr := intColumn(columns, "bytes")
	widthExpr := intColumn(columns, "width")
	heightExpr := intColumn(columns, "height")
	fidelityExpr := stringColumn(columns, "fidelity")
	rows, err := q.QueryContext(ctx, `SELECT id, name, mime, `+bytesExpr+`, `+widthExpr+`,
		`+heightExpr+`, `+fidelityExpr+`, data, created_at
		FROM studio_design_assets WHERE design_id = ? ORDER BY created_at, rowid`, designID)
	if err != nil {
		return fmt.Errorf("read legacy design assets: %w", err)
	}
	defer rows.Close()
	type legacyAsset struct {
		id, name, mime, fidelity, createdAt string
		bytes, width, height                int64
		data                                []byte
	}
	assets := make([]legacyAsset, 0)
	for rows.Next() {
		var asset legacyAsset
		if err := rows.Scan(&asset.id, &asset.name, &asset.mime, &asset.bytes,
			&asset.width, &asset.height, &asset.fidelity, &asset.data, &asset.createdAt); err != nil {
			return fmt.Errorf("scan legacy design asset: %w", err)
		}
		assets = append(assets, asset)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate legacy design assets: %w", err)
	}
	for _, asset := range assets {
		fidelity := strings.TrimSpace(asset.fidelity)
		if fidelity != AssetFidelityCaptured {
			fidelity = AssetFidelityReference
		}
		if _, err := q.ExecContext(ctx, `INSERT INTO workspace_assets
			(id, workspace_id, name, mime, bytes, width, height, fidelity, data, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO NOTHING`,
			asset.id, workspaceID, asset.name, asset.mime, asset.bytes,
			asset.width, asset.height, fidelity, asset.data, asset.createdAt); err != nil {
			return fmt.Errorf("import legacy design asset %q: %w", asset.id, err)
		}
	}
	return nil
}

// stringColumn/intColumn render a column reference only when the live table has
// it. The table name and column name are compile-time literals from this
// package; only the presence test is dynamic.
func stringColumn(columns map[string]bool, name string) string {
	if columns[name] {
		return "COALESCE(" + name + ", '')"
	}
	return "''"
}

func intColumn(columns map[string]bool, name string) string {
	if columns[name] {
		return "COALESCE(" + name + ", 0)"
	}
	return "0"
}
