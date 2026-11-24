package workspaces

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	db "github.com/vrooli/api-core/databasetest"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
)

// currentLegacySchema is studio_designs as internal/designs ships it, including
// the binding and options columns. The importer must read all of them.
const currentLegacySchema = `
CREATE TABLE studio_designs (
  id           TEXT PRIMARY KEY,
  name         TEXT NOT NULL,
  revision     INTEGER NOT NULL DEFAULT 1,
  frame_count  INTEGER NOT NULL DEFAULT 0,
  frames_json  TEXT NOT NULL DEFAULT '[]',
  theme_json   TEXT NOT NULL DEFAULT '{}',
  options_json TEXT NOT NULL DEFAULT '{}',
  scenario     TEXT,
  page         TEXT,
  created_at   TEXT NOT NULL,
  updated_at   TEXT NOT NULL
);
`

// oldLegacySchema predates scenario/page/options_json. The importer assembles
// its select list from the live table shape, so this must work too.
const oldLegacySchema = `
CREATE TABLE studio_designs (
  id           TEXT PRIMARY KEY,
  name         TEXT NOT NULL,
  revision     INTEGER NOT NULL DEFAULT 1,
  frame_count  INTEGER NOT NULL DEFAULT 0,
  frames_json  TEXT NOT NULL DEFAULT '[]',
  theme_json   TEXT NOT NULL DEFAULT '{}',
  created_at   TEXT NOT NULL,
  updated_at   TEXT NOT NULL
);
`

func TestMigrateImportsStudioDesignsWithFrameSpecs(t *testing.T) {
	sqlDB := db.NewSQLite(t)
	ctx := context.Background()
	_, err := sqlDB.ExecContext(ctx, currentLegacySchema)
	require.NoError(t, err)

	frames := `[
		{"id":"frame-desktop","name":"Desktop","device":"desktop","spec":{"root":{"kind":"page","children":[{"id":"hero","type":"surface","props":{"label":"Welcome"},"unknownKey":{"keep":true}}]}}},
		{"id":"frame-phone","name":"Phone","device":"phone","spec":{"root":{"kind":"page","children":[{"id":"cta","type":"button","props":{"label":"Start"}}]}}}
	]`
	_, err = sqlDB.ExecContext(ctx, `INSERT INTO studio_designs
		(id, name, revision, frame_count, frames_json, theme_json, options_json, scenario, page, created_at, updated_at)
		VALUES (?, ?, 4, 2, ?, ?, ?, ?, ?, ?, ?)`,
		"design-harbor", "Harbor onboarding", frames,
		`{"ink":"#111111","accent":"#ff5500"}`, `{"radius":18,"display":"serif"}`,
		"agent-inbox", "inbox", "2026-11-18T10:00:00.000Z", "2026-11-18T11:00:00.000Z")
	require.NoError(t, err)

	require.NoError(t, Migrate(ctx, sqlDB))

	repo := NewSQLiteRepository(sqlDB)
	list, err := repo.List(ctx, workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_IMPORTED, "", 0)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "Harbor onboarding", list[0].Name)
	require.Equal(t, "agent-inbox", list[0].BindingScenario)
	require.Equal(t, "inbox", list[0].BindingPage)
	require.Equal(t, int32(1), list[0].PageCount)
	require.Equal(t, int32(2), list[0].DisplayCount)
	require.Equal(t, int32(2), list[0].StateCount)

	workspace, err := repo.Get(ctx, list[0].Id)
	require.NoError(t, err)
	require.Equal(t, workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_IMPORTED, workspace.Provenance)
	require.Equal(t, "2026-11-18T10:00:00.000Z", workspace.CreatedAt)
	require.Equal(t, "2026-11-18T11:00:00.000Z", workspace.UpdatedAt)
	require.JSONEq(t, `{"ink":"#111111","accent":"#ff5500"}`, workspace.ThemeJson)
	require.JSONEq(t, `{"radius":18,"display":"serif"}`, workspace.OptionsJson)

	require.Len(t, workspace.Pages, 1)
	page := workspace.Pages[0]
	require.Equal(t, "Harbor onboarding", page.Name)

	var frameSpecs []json.RawMessage
	var rawFrames []map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(frames), &rawFrames))
	for _, raw := range rawFrames {
		frameSpecs = append(frameSpecs, raw["spec"])
	}
	require.JSONEq(t, string(frameSpecs[0]), page.SpecJson, "page spec is the first frame spec")

	require.Len(t, page.Displays, 2)
	desktop := page.Displays[0]
	require.Equal(t, workspacev1.DisplayDevice_DISPLAY_DEVICE_DESKTOP, desktop.Device)
	require.Equal(t, int32(1180), desktop.Width)
	require.Equal(t, int32(780), desktop.Height)
	phone := page.Displays[1]
	require.Equal(t, workspacev1.DisplayDevice_DISPLAY_DEVICE_PHONE, phone.Device)
	require.Equal(t, int32(390), phone.Width)
	require.Equal(t, int32(844), phone.Height)

	for index, display := range page.Displays {
		require.Len(t, display.States, 1)
		state := display.States[0]
		require.Equal(t, "Default", state.Name)
		require.Equal(t, workspacev1.StateStatus_STATE_STATUS_DRAFT, state.Status)

		var args map[string]json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(state.ArgsJson), &args))
		require.Len(t, args, 1)
		require.JSONEq(t, string(frameSpecs[index]), string(args["spec"]),
			"display %d default state must preserve its frame spec losslessly", index)
	}

	// The import receipt makes a second boot a no-op.
	var migrations int
	require.NoError(t, sqlDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM workspace_migrations WHERE source = 'studio_designs' AND source_id = 'design-harbor'`).Scan(&migrations))
	require.Equal(t, 1, migrations)
	requireLegacyTablesRetired(t, ctx, sqlDB)

	require.NoError(t, Migrate(ctx, sqlDB))
	list, err = repo.List(ctx, workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_UNSPECIFIED, "", 0)
	require.NoError(t, err)
	require.Len(t, list, 1, "a second Migrate must not duplicate the import")
}

func TestMigrateHandlesPreBindingStudioDesigns(t *testing.T) {
	sqlDB := db.NewSQLite(t)
	ctx := context.Background()
	_, err := sqlDB.ExecContext(ctx, oldLegacySchema)
	require.NoError(t, err)
	_, err = sqlDB.ExecContext(ctx, `INSERT INTO studio_designs
		(id, name, revision, frame_count, frames_json, theme_json, created_at, updated_at)
		VALUES ('design-old', 'Old design', 2, 1, '[{"id":"f","name":"Desk","device":"tablet","spec":{"root":{"kind":"page"}}}]', '{"ink":"#000"}', '2026-11-17T09:00:00.000Z', '2026-11-17T09:30:00.000Z')`)
	require.NoError(t, err)

	require.NoError(t, Migrate(ctx, sqlDB))

	repo := NewSQLiteRepository(sqlDB)
	list, err := repo.List(ctx, workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_IMPORTED, "", 0)
	require.NoError(t, err)
	require.Len(t, list, 1)

	workspace, err := repo.Get(ctx, list[0].Id)
	require.NoError(t, err)
	require.Empty(t, workspace.BindingScenario)
	require.Empty(t, workspace.BindingPage)
	require.JSONEq(t, `{}`, workspace.OptionsJson)
	require.Len(t, workspace.Pages[0].Displays, 1)
	require.Equal(t, workspacev1.DisplayDevice_DISPLAY_DEVICE_TABLET, workspace.Pages[0].Displays[0].Device)
	require.Equal(t, int32(834), workspace.Pages[0].Displays[0].Width)
	require.Equal(t, int32(900), workspace.Pages[0].Displays[0].Height)
}

func TestMigrateWithoutStudioDesignsIsANoOp(t *testing.T) {
	sqlDB := db.NewSQLite(t)
	ctx := context.Background()

	require.NoError(t, Migrate(ctx, sqlDB))

	var tables int
	require.NoError(t, sqlDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'workspaces'`).Scan(&tables))
	require.Equal(t, 1, tables, "Migrate applies the schema even when there is nothing to import")

	require.NoError(t, Migrate(ctx, sqlDB), "Migrate is idempotent")
}

func TestMigrateBackfillsLatestCaptureEvidenceIntoStableReceiptOnce(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	created, err := repo.Create(ctx, &workspacev1.CreateWorkspaceRequest{Name: "Existing capture"})
	require.NoError(t, err)
	stateID := created.Pages[0].Displays[0].States[0].Id
	_, err = repo.db.ExecContext(ctx, `INSERT INTO workspace_state_capture_evidence
		(workspace_id,state_id,request_token,source_revision,capture_id,evidence_json,captured_at)
		VALUES (?,?,?,?,?,?,?)`, created.Id, stateID, "legacy-token", created.Revision, "legacy-capture",
		`{"schemaVersion":"rcl-workspace-state-capture/v1","attemptId":"legacy-attempt"}`, "2026-11-20T12:00:00Z")
	require.NoError(t, err)
	primaryDB := repo.db.(*sql.DB)
	require.NoError(t, Migrate(ctx, primaryDB))
	require.NoError(t, Migrate(ctx, primaryDB))
	receipts, err := repo.ListCaptureReceipts(ctx, created.Id, stateID)
	require.NoError(t, err)
	require.Len(t, receipts, 1)
	require.Equal(t, "legacy-capture", receipts[0].CaptureID)
	require.Equal(t, "legacy-attempt", receipts[0].AttemptID)
}

func TestMigrateImportsLegacyTemplateAsOpenableCuratedWorkspace(t *testing.T) {
	sqlDB := db.NewSQLite(t)
	ctx := context.Background()
	_, err := sqlDB.ExecContext(ctx, `CREATE TABLE studio_templates (
		id TEXT PRIMARY KEY, name TEXT NOT NULL, tagline TEXT NOT NULL DEFAULT '',
		category TEXT NOT NULL DEFAULT '', palette_json TEXT NOT NULL DEFAULT '{}',
		display TEXT NOT NULL DEFAULT 'sans', radius INTEGER NOT NULL DEFAULT 14,
		spec_json TEXT NOT NULL DEFAULT '{}', created_at TEXT NOT NULL, updated_at TEXT NOT NULL
	)`)
	require.NoError(t, err)
	spec := `{"id":"legacy-page","type":"page","props":{"pad":24},"children":[{"id":"legacy-title","type":"text","props":{"text":"Welcome"}}]}`
	document := `{"spec":` + spec + `,"slotHint":"Main content","idea":"A saved page"}`
	_, err = sqlDB.ExecContext(ctx, `INSERT INTO studio_templates
		(id, name, tagline, category, palette_json, display, radius, spec_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"custom-harbor", "Harbor", "A saved legacy template", "marketing",
		`{"ink":"#112233","accent":"#ff5500"}`, "serif", 22, document,
		"2026-11-18T10:00:00.000Z", "2026-11-18T11:00:00.000Z")
	require.NoError(t, err)

	require.NoError(t, Migrate(ctx, sqlDB))
	repo := NewSQLiteRepository(sqlDB)
	workspace, err := repo.Get(ctx, "ws_curated_custom-harbor")
	require.NoError(t, err, "bookmark-compatible workspace id must open after migration")
	require.Equal(t, "Harbor", workspace.GetName())
	require.Equal(t, "A saved legacy template", workspace.GetSummary())
	require.Equal(t, workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_CURATED, workspace.GetProvenance())
	require.Equal(t, "2026-11-18T10:00:00.000Z", workspace.GetCreatedAt())
	require.Equal(t, "2026-11-18T11:00:00.000Z", workspace.GetUpdatedAt())
	require.Equal(t, []string{"marketing"}, workspace.GetTags())
	require.JSONEq(t, `{"ink":"#112233","accent":"#ff5500"}`, workspace.GetThemeJson())
	require.JSONEq(t, `{"display":"serif","radius":22}`, workspace.GetOptionsJson())
	require.Len(t, workspace.GetPages(), 1)
	require.Len(t, workspace.GetPages()[0].GetDisplays(), 1)
	state := workspace.GetPages()[0].GetDisplays()[0].GetStates()[0]
	require.JSONEq(t, `{"spec":`+spec+`}`, state.GetArgsJson())

	var receipts, workspaces int
	require.NoError(t, sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM workspace_migrations WHERE source = 'studio_templates' AND source_id = 'custom-harbor'`).Scan(&receipts))
	require.Equal(t, 1, receipts)
	requireLegacyTablesRetired(t, ctx, sqlDB)
	require.NoError(t, sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM workspaces WHERE id = 'ws_curated_custom-harbor'`).Scan(&workspaces))
	require.Equal(t, 1, workspaces)
	require.NoError(t, Migrate(ctx, sqlDB))
	require.NoError(t, sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM workspaces WHERE id = 'ws_curated_custom-harbor'`).Scan(&workspaces))
	require.Equal(t, 1, workspaces, "second boot must not duplicate imported templates")
	_, err = sqlDB.ExecContext(ctx, `DELETE FROM workspaces WHERE id = 'ws_curated_custom-harbor'`)
	require.NoError(t, err)
	require.NoError(t, Migrate(ctx, sqlDB))
	require.NoError(t, sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM workspaces WHERE id = 'ws_curated_custom-harbor'`).Scan(&workspaces))
	require.Zero(t, workspaces, "a migration receipt prevents a deleted workspace from being resurrected")
}

func TestMigrateRollsBackTemplateImportOnWorkspaceIDCollision(t *testing.T) {
	sqlDB := db.NewSQLite(t)
	ctx := context.Background()
	_, err := sqlDB.ExecContext(ctx, `CREATE TABLE studio_templates (
		id TEXT PRIMARY KEY, name TEXT NOT NULL, tagline TEXT NOT NULL DEFAULT '',
		category TEXT NOT NULL DEFAULT '', palette_json TEXT NOT NULL DEFAULT '{}',
		display TEXT NOT NULL DEFAULT 'sans', radius INTEGER NOT NULL DEFAULT 14,
		spec_json TEXT NOT NULL DEFAULT '{}', created_at TEXT NOT NULL, updated_at TEXT NOT NULL
	)`)
	require.NoError(t, err)
	_, err = sqlDB.ExecContext(ctx, `INSERT INTO studio_templates (id, name, palette_json, spec_json, created_at, updated_at)
		VALUES ('collision', 'Legacy', '{}', '{}', '2026-11-18T10:00:00.000Z', '2026-11-18T10:00:00.000Z')`)
	require.NoError(t, err)
	_, err = sqlDB.ExecContext(ctx, schemaSQL)
	require.NoError(t, err)
	_, err = sqlDB.ExecContext(ctx, `INSERT INTO workspaces (id, name, provenance, created_at, updated_at)
		VALUES ('ws_curated_collision', 'Existing', 'curated', '2026-11-18T10:00:00.000Z', '2026-11-18T10:00:00.000Z')`)
	require.NoError(t, err)

	err = Migrate(ctx, sqlDB)
	require.ErrorContains(t, err, "conflicts with an existing workspace")
	var receipts int
	require.NoError(t, sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM workspace_migrations WHERE source = 'studio_templates'`).Scan(&receipts))
	require.Zero(t, receipts, "collision must not leave a false receipt")
	requireLegacyTablePresent(t, ctx, sqlDB, "studio_templates")
}

func TestMigrateReportsDuplicateLegacyBindingsBeforeInstallingIndex(t *testing.T) {
	sqlDB := db.NewSQLite(t)
	ctx := context.Background()
	_, err := sqlDB.ExecContext(ctx, Schema())
	require.NoError(t, err)
	for _, id := range []string{"ws_old_a", "ws_old_b"} {
		_, err := sqlDB.ExecContext(ctx, `INSERT INTO workspaces
			(id, name, binding_scenario, binding_page, created_at, updated_at)
			VALUES (?, 'Legacy', 'harbor', 'welcome', '2026-11-19T00:00:00.000Z', '2026-11-19T00:00:00.000Z')`, id)
		require.NoError(t, err)
	}

	err = Migrate(ctx, sqlDB)
	require.ErrorContains(t, err, "duplicate workspace experience bindings")
	require.ErrorContains(t, err, "harbor/welcome")
	require.ErrorContains(t, err, "ws_old_a")
	require.ErrorContains(t, err, "ws_old_b")
	require.ErrorContains(t, err, "resolve these rows explicitly")

	var indexes int
	require.NoError(t, sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'workspaces_binding_unique'`).Scan(&indexes))
	require.Zero(t, indexes, "ambiguous legacy ownership must not be silently selected")
}

func TestMigrateRejectsUnreadableFrames(t *testing.T) {
	sqlDB := db.NewSQLite(t)
	ctx := context.Background()
	_, err := sqlDB.ExecContext(ctx, currentLegacySchema)
	require.NoError(t, err)
	_, err = sqlDB.ExecContext(ctx, `INSERT INTO studio_designs
		(id, name, frames_json, created_at, updated_at)
		VALUES ('design-bad', 'Bad frames', '{"not":"an array"}', '2026-11-18T10:00:00.000Z', '2026-11-18T10:00:00.000Z')`)
	require.NoError(t, err)

	require.Error(t, Migrate(ctx, sqlDB))

	var count int
	require.NoError(t, sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM workspaces`).Scan(&count))
	require.Zero(t, count, "a failed migration must not leave a partial import")
	requireLegacyTablePresent(t, ctx, sqlDB, "studio_designs")
}

func TestMigrateRejectsUnknownFrameDevice(t *testing.T) {
	sqlDB := db.NewSQLite(t)
	ctx := context.Background()
	_, err := sqlDB.ExecContext(ctx, currentLegacySchema)
	require.NoError(t, err)
	_, err = sqlDB.ExecContext(ctx, `INSERT INTO studio_designs
		(id, name, frames_json, created_at, updated_at)
		VALUES ('design-device', 'Unknown device', '[{"name":"F","device":"watch","spec":{}}]', '2026-11-18T10:00:00.000Z', '2026-11-18T10:00:00.000Z')`)
	require.NoError(t, err)

	require.ErrorContains(t, Migrate(ctx, sqlDB), "unsupported device")
}

// The legacy asset table is the other half of a design. If a person uploaded
// references before the switch, the imported workspace must own the same bytes
// under the same ids rather than presenting an empty Assets pane.
func TestMigrateMovesLegacyDesignAssetsToTheWorkspace(t *testing.T) {
	sqlDB := db.NewSQLite(t)
	ctx := context.Background()
	_, err := sqlDB.ExecContext(ctx, currentLegacySchema+`
CREATE TABLE studio_design_assets (
  id          TEXT PRIMARY KEY,
  design_id   TEXT NOT NULL,
  name        TEXT NOT NULL,
  mime        TEXT NOT NULL,
  bytes       INTEGER NOT NULL DEFAULT 0,
  width       INTEGER NOT NULL DEFAULT 0,
  height      INTEGER NOT NULL DEFAULT 0,
  fidelity    TEXT NOT NULL DEFAULT 'reference',
  data        BLOB NOT NULL,
  created_at  TEXT NOT NULL
);`)
	require.NoError(t, err)
	_, err = sqlDB.ExecContext(ctx, `INSERT INTO studio_designs
		(id, name, revision, frame_count, frames_json, theme_json, created_at, updated_at)
		VALUES ('design-a', 'Asset design', 1, 0, '[]', '{}', '2026-11-18T10:00:00.000Z', '2026-11-18T10:00:00.000Z')`)
	require.NoError(t, err)
	_, err = sqlDB.ExecContext(ctx, `INSERT INTO studio_design_assets
		(id, design_id, name, mime, bytes, width, height, fidelity, data, created_at)
		VALUES ('asset-1', 'design-a', 'hero.png', 'image/png', 4, 10, 20, 'captured', ?, '2026-11-18T09:00:00.000Z')`,
		[]byte{1, 2, 3, 4})
	require.NoError(t, err)

	require.NoError(t, Migrate(ctx, sqlDB))
	repo := NewSQLiteRepository(sqlDB)
	list, err := repo.List(ctx, workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_IMPORTED, "", 0)
	require.NoError(t, err)
	require.Len(t, list, 1)

	assets, err := repo.ListWorkspaceAssets(ctx, list[0].Id)
	require.NoError(t, err)
	require.Len(t, assets, 1)
	require.Equal(t, "asset-1", assets[0].ID)
	require.Equal(t, AssetFidelityCaptured, assets[0].Fidelity)

	_, data, err := repo.GetWorkspaceAsset(ctx, list[0].Id, "asset-1")
	require.NoError(t, err)
	require.Equal(t, []byte{1, 2, 3, 4}, data)

	// A second boot must not duplicate the asset.
	require.NoError(t, Migrate(ctx, sqlDB))
	assets, err = repo.ListWorkspaceAssets(ctx, list[0].Id)
	require.NoError(t, err)
	require.Len(t, assets, 1)
}

func TestMigrateBackfillsAssetsFromAnEarlierImportReceiptBeforeRetiringSources(t *testing.T) {
	sqlDB := db.NewSQLite(t)
	ctx := context.Background()
	_, err := sqlDB.ExecContext(ctx, currentLegacySchema+`
CREATE TABLE studio_design_assets (
  id          TEXT PRIMARY KEY,
  design_id   TEXT NOT NULL,
  name        TEXT NOT NULL,
  mime        TEXT NOT NULL,
  bytes       INTEGER NOT NULL DEFAULT 0,
  width       INTEGER NOT NULL DEFAULT 0,
  height      INTEGER NOT NULL DEFAULT 0,
  fidelity    TEXT NOT NULL DEFAULT 'reference',
  data        BLOB NOT NULL,
  created_at  TEXT NOT NULL
);`)
	require.NoError(t, err)
	_, err = sqlDB.ExecContext(ctx, `INSERT INTO studio_designs
		(id, name, revision, frame_count, frames_json, theme_json, created_at, updated_at)
		VALUES ('design-late', 'Late assets', 1, 0, '[]', '{}', '2026-11-18T10:00:00.000Z', '2026-11-18T10:00:00.000Z')`)
	require.NoError(t, err)
	// Simulate a release that already wrote the workspace and receipt but left
	// the legacy upload table for the compatibility window.
	_, err = sqlDB.ExecContext(ctx, schemaSQL)
	require.NoError(t, err)
	workspace := &workspacev1.Workspace{
		Id: "ws_design-late", Name: "Late assets",
		Provenance: workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_IMPORTED,
		Revision:   1, ThemeJson: "{}", OptionsJson: "{}",
		CreatedAt: "2026-11-18T10:00:00.000Z", UpdatedAt: "2026-11-18T10:00:00.000Z",
	}
	require.NoError(t, insertWorkspace(ctx, sqlDB, workspace))
	_, err = sqlDB.ExecContext(ctx, `INSERT INTO workspace_migrations
		(source, source_id, workspace_id, migrated_at)
		VALUES ('studio_designs', 'design-late', 'ws_design-late', '2026-11-18T10:00:00.000Z')`)
	require.NoError(t, err)
	_, err = sqlDB.ExecContext(ctx, `INSERT INTO studio_design_assets
		(id, design_id, name, mime, bytes, width, height, fidelity, data, created_at)
		VALUES ('asset-late', 'design-late', 'late.png', 'image/png', 2, 1, 1, 'reference', ?, '2026-11-18T11:00:00.000Z')`,
		[]byte{7, 8})
	require.NoError(t, err)
	require.NoError(t, Migrate(ctx, sqlDB))

	repo := NewSQLiteRepository(sqlDB)
	assets, err := repo.ListWorkspaceAssets(ctx, workspace.Id)
	require.NoError(t, err)
	require.Len(t, assets, 1)
	require.Equal(t, "asset-late", assets[0].ID)
	requireLegacyTablesRetired(t, ctx, sqlDB)
}

func requireLegacyTablesRetired(t *testing.T, ctx context.Context, sqlDB *sql.DB) {
	t.Helper()
	for _, table := range []string{"studio_designs", "studio_design_assets", "studio_templates"} {
		var count int
		require.NoError(t, sqlDB.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&count))
		require.Zero(t, count, "legacy table %s should be dropped after import", table)
	}
}

func requireLegacyTablePresent(t *testing.T, ctx context.Context, sqlDB *sql.DB, table string) {
	t.Helper()
	var count int
	require.NoError(t, sqlDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&count))
	require.Equal(t, 1, count, "failed import must preserve legacy table %s", table)
}
