package workspaces

import (
	"context"
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

func TestMigrateBackfillsAssetsForAnAlreadyImportedDesign(t *testing.T) {
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
	require.NoError(t, Migrate(ctx, sqlDB))

	// The upload arrives after the design was imported. The receipt exists, so
	// only a backfill pass can carry it over.
	_, err = sqlDB.ExecContext(ctx, `INSERT INTO studio_design_assets
		(id, design_id, name, mime, bytes, width, height, fidelity, data, created_at)
		VALUES ('asset-late', 'design-late', 'late.png', 'image/png', 2, 1, 1, 'reference', ?, '2026-11-18T11:00:00.000Z')`,
		[]byte{7, 8})
	require.NoError(t, err)
	require.NoError(t, Migrate(ctx, sqlDB))

	repo := NewSQLiteRepository(sqlDB)
	list, err := repo.List(ctx, workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_IMPORTED, "", 0)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assets, err := repo.ListWorkspaceAssets(ctx, list[0].Id)
	require.NoError(t, err)
	require.Len(t, assets, 1)
	require.Equal(t, "asset-late", assets[0].ID)
}
