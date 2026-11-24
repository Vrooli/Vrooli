package workspaces

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
)

// Migrate applies the workspaces schema, imports legacy studio_designs and
// studio_templates rows, installs the singular experience page binding index
// after a diagnostic duplicate check, and retires their source tables.
//
// It is deliberately additive and idempotent: the schema text is re-applied
// (every statement is IF NOT EXISTS) and workspace_migrations records each
// source row by (source, source_id). Legacy tables are removed only inside the
// same transaction that commits all successful imports and asset backfills, so
// malformed rows or collisions leave the old data intact for repair.
func Migrate(ctx context.Context, db Database) error {
	if _, err := db.ExecContext(ctx, schemaSQL); err != nil {
		return fmt.Errorf("apply workspaces schema: %w", err)
	}
	// Preserve the latest evidence already present on installs upgrading from
	// the projection-only schema. The unique capture-operation key makes this
	// backfill idempotent even though the generated receipt ID is fresh per run.
	if _, err := db.ExecContext(ctx, `INSERT OR IGNORE INTO workspace_capture_receipts
		(id,workspace_id,state_id,source_revision,capture_id,attempt_id,request_token,evidence_json,captured_at)
		SELECT 'legacy_' || lower(hex(randomblob(16))),workspace_id,state_id,source_revision,capture_id,
			COALESCE(NULLIF(json_extract(evidence_json,'$.attemptId'),''),'legacy'),request_token,evidence_json,captured_at
		FROM workspace_state_capture_evidence`); err != nil {
		return fmt.Errorf("backfill workspace capture receipts: %w", err)
	}
	exists, err := tableExists(ctx, db, "studio_designs")
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("migrate studio_designs: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	now := nowTimestamp()
	if exists {
		designs, err := loadLegacyDesigns(ctx, tx)
		if err != nil {
			return err
		}
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
	}
	templates, err := loadLegacyTemplates(ctx, tx)
	if err != nil {
		return err
	}
	for _, template := range templates {
		done, err := migrationRecorded(ctx, tx, "studio_templates", template.ID)
		if err != nil {
			return err
		}
		if done {
			continue
		}
		if err := importLegacyTemplate(ctx, tx, template, now); err != nil {
			return fmt.Errorf("migrate studio_templates %q: %w", template.ID, err)
		}
	}
	// Repairs an install that imported a design before this package owned
	// assets: the design receipt already exists, so the loop above skips it and
	// its uploads would otherwise never arrive. The insert is idempotent.
	if err := backfillLegacyAssets(ctx, tx); err != nil {
		return err
	}
	if err := enforceUniqueExperienceBinding(ctx, tx); err != nil {
		return err
	}
	if err := retireLegacyTables(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migrate studio_designs commit: %w", err)
	}
	return nil
}

func retireLegacyTables(ctx context.Context, tx *sql.Tx) error {
	for _, table := range []string{"studio_design_assets", "studio_designs", "studio_templates"} {
		exists, err := tableExists(ctx, tx, table)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		if _, err := tx.ExecContext(ctx, "DROP TABLE "+table); err != nil {
			return fmt.Errorf("retire legacy table %s: %w", table, err)
		}
	}
	return nil
}

type legacyTemplate struct {
	ID, Name, Tagline, Category string
	Palette, Display            string
	Radius                      int
	Spec, CreatedAt, UpdatedAt  string
}

func loadLegacyTemplates(ctx context.Context, q queryer) ([]legacyTemplate, error) {
	exists, err := tableExists(ctx, q, "studio_templates")
	if err != nil || !exists {
		return nil, err
	}
	rows, err := q.QueryContext(ctx, `SELECT id, name, COALESCE(tagline, ''), COALESCE(category, ''),
		COALESCE(palette_json, '{}'), COALESCE(display, 'sans'), COALESCE(radius, 14),
		COALESCE(spec_json, '{}'), COALESCE(created_at, ''), COALESCE(updated_at, '')
		FROM studio_templates ORDER BY rowid`)
	if err != nil {
		return nil, fmt.Errorf("read studio_templates: %w", err)
	}
	defer rows.Close()
	templates := make([]legacyTemplate, 0)
	for rows.Next() {
		var template legacyTemplate
		if err := rows.Scan(&template.ID, &template.Name, &template.Tagline, &template.Category,
			&template.Palette, &template.Display, &template.Radius, &template.Spec,
			&template.CreatedAt, &template.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan studio_templates: %w", err)
		}
		templates = append(templates, template)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate studio_templates: %w", err)
	}
	return templates, nil
}

func importLegacyTemplate(ctx context.Context, q queryer, template legacyTemplate, now string) error {
	name := strings.TrimSpace(template.Name)
	if template.ID == "" || name == "" {
		return fmt.Errorf("id and name are required")
	}
	if !jsonObject(template.Palette) {
		return fmt.Errorf("palette_json and spec_json must be JSON objects")
	}
	spec, _, idea, err := unwrapLegacyTemplateSpec(template.Spec)
	if err != nil {
		return err
	}
	createdAt := template.CreatedAt
	if strings.TrimSpace(createdAt) == "" {
		createdAt = now
	}
	updatedAt := template.UpdatedAt
	if strings.TrimSpace(updatedAt) == "" {
		updatedAt = createdAt
	}
	workspaceID := "ws_curated_" + template.ID
	display := strings.ToLower(strings.TrimSpace(template.Display))
	if display != "serif" {
		display = "sans"
	}
	radius := template.Radius
	if radius < 0 || radius > 48 {
		radius = 14
	}
	args, err := frameArgsJSON(spec)
	if err != nil {
		return err
	}
	workspace := &workspacev1.Workspace{
		Id: workspaceID, Name: name, Summary: firstNonEmpty(strings.TrimSpace(template.Tagline), strings.TrimSpace(idea)),
		Provenance: workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_CURATED,
		Tags:       []string{strings.TrimSpace(template.Category)}, ThemeJson: template.Palette,
		OptionsJson: fmt.Sprintf(`{"display":%q,"radius":%d}`, display, radius), Revision: 1,
		CreatedAt: createdAt, UpdatedAt: updatedAt,
		Pages: []*workspacev1.WorkspacePage{{Id: "pg_legacy_template_" + template.ID, Name: name, SpecJson: "{}", Displays: []*workspacev1.WorkspaceDisplay{{
			Id: "dp_legacy_template_" + template.ID, Device: workspacev1.DisplayDevice_DISPLAY_DEVICE_DESKTOP,
			Width: 1280, Height: 800, States: []*workspacev1.WorkspaceState{{
				Id: "st_legacy_template_" + template.ID, Name: "Default", ArgsJson: args,
				Status: workspacev1.StateStatus_STATE_STATUS_DRAFT,
			}},
		}}}},
	}
	if err := normalizeAggregate(workspace, false); err != nil {
		return err
	}
	if err := invalidFromIssues(validationIssues(workspace)); err != nil {
		return err
	}
	if err := insertWorkspace(ctx, q, workspace); err != nil {
		return fmt.Errorf("workspace id %q conflicts with an existing workspace (legacy id %q): %w", workspaceID, template.ID, err)
	}
	if err := insertNested(ctx, q, workspace); err != nil {
		return err
	}
	snapshot, err := marshalWorkspace(workspace)
	if err != nil {
		return err
	}
	if err := insertRevision(ctx, q, workspaceID, 1, snapshot, "migration", updatedAt); err != nil {
		return err
	}
	if _, err := q.ExecContext(ctx, `INSERT INTO workspace_migrations
		(source, source_id, workspace_id, migrated_at) VALUES ('studio_templates', ?, ?, ?)`, template.ID, workspaceID, now); err != nil {
		return fmt.Errorf("record template migration: %w", err)
	}
	return nil
}

func jsonObject(raw string) bool {
	var value map[string]json.RawMessage
	return json.Unmarshal([]byte(raw), &value) == nil && value != nil
}

func unwrapLegacyTemplateSpec(raw string) (json.RawMessage, string, string, error) {
	var document struct {
		Spec     json.RawMessage `json:"spec"`
		SlotHint string          `json:"slotHint"`
		Idea     string          `json:"idea"`
	}
	if err := json.Unmarshal([]byte(raw), &document); err != nil {
		return nil, "", "", fmt.Errorf("spec_json is not valid JSON: %w", err)
	}
	if document.Spec == nil {
		if !jsonObject(raw) {
			return nil, "", "", fmt.Errorf("spec_json must be a JSON object")
		}
		return json.RawMessage(raw), "", "", nil
	}
	if !jsonObject(string(document.Spec)) {
		return nil, "", "", fmt.Errorf("spec_json.spec must be a JSON object")
	}
	return document.Spec, document.SlotHint, document.Idea, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// enforceUniqueExperienceBinding installs the one-composition-owner invariant
// only after legacy imports have completed. Checking first produces a useful
// startup diagnostic for existing duplicate bindings instead of SQLite's
// opaque unique-index error during schema initialization.
func enforceUniqueExperienceBinding(ctx context.Context, db queryer) error {
	rows, err := db.QueryContext(ctx, `SELECT binding_scenario, binding_page, GROUP_CONCAT(id), COUNT(*)
		FROM workspaces
		WHERE binding_scenario <> '' AND binding_page <> ''
		GROUP BY binding_scenario, binding_page
		HAVING COUNT(*) > 1
		ORDER BY binding_scenario, binding_page`)
	if err != nil {
		return fmt.Errorf("check workspace experience bindings: %w", err)
	}
	type duplicate struct {
		scenario, page, ids string
		count               int
	}
	duplicates := make([]duplicate, 0)
	for rows.Next() {
		var item duplicate
		if err := rows.Scan(&item.scenario, &item.page, &item.ids, &item.count); err != nil {
			rows.Close()
			return fmt.Errorf("read duplicate workspace experience bindings: %w", err)
		}
		duplicates = append(duplicates, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate duplicate workspace experience bindings: %w", err)
	}
	rows.Close()
	if len(duplicates) > 0 {
		parts := make([]string, 0, len(duplicates))
		for _, item := range duplicates {
			parts = append(parts, fmt.Sprintf("%s/%s (%d workspaces: %s)", item.scenario, item.page, item.count, item.ids))
		}
		return fmt.Errorf("duplicate workspace experience bindings prevent one-owner index: %s; resolve these rows explicitly, then restart", strings.Join(parts, "; "))
	}
	if _, err := db.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS workspaces_binding_unique
		ON workspaces (binding_scenario, binding_page)
		WHERE binding_scenario <> '' AND binding_page <> ''`); err != nil {
		return fmt.Errorf("install unique workspace experience binding index: %w", err)
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
			// The caller retires the legacy table only after a full successful
			// import. Fail closed so unsupported shapes are never silently dropped.
			return fmt.Errorf("legacy design asset table is missing required column %q", required)
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
