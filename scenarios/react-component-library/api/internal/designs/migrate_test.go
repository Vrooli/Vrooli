package designs

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	db "github.com/vrooli/api-core/databasetest"
)

// oldSchema is the pre-binding, pre-assets schema exactly as it shipped
// before this change. Migrate runs against a database created this way.
const oldSchema = `
CREATE TABLE IF NOT EXISTS studio_designs (
  id           TEXT PRIMARY KEY,
  name         TEXT NOT NULL,
  revision     INTEGER NOT NULL DEFAULT 1,
  frame_count  INTEGER NOT NULL DEFAULT 0,
  frames_json  TEXT NOT NULL DEFAULT '[]',
  theme_json   TEXT NOT NULL DEFAULT '{}',
  created_at   TEXT NOT NULL,
  updated_at   TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS studio_designs_updated_at
  ON studio_designs (updated_at DESC);
`

func TestMigrateAddsBindingColumnsWithoutLosingData(t *testing.T) {
	sqlDB := db.NewSQLite(t)
	ctx := context.Background()
	_, err := sqlDB.ExecContext(ctx, oldSchema)
	require.NoError(t, err)
	_, err = sqlDB.ExecContext(ctx,
		`INSERT INTO studio_designs (id, name, revision, frame_count, frames_json, theme_json, created_at, updated_at)
		 VALUES ('design-old', 'Old design', 4, 2, '[{"id":"a"},{"id":"b"}]', '{"ink":"#111"}', '2026-11-18T10:00:00.000Z', '2026-11-18T11:00:00.000Z')`)
	require.NoError(t, err)

	require.NoError(t, Migrate(ctx, sqlDB))

	for _, column := range []string{"scenario", "page", "options_json"} {
		var count int
		require.NoError(t, sqlDB.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM pragma_table_info('studio_designs') WHERE name = ?`, column).Scan(&count))
		require.Equal(t, 1, count, "migration must add studio_designs.%s", column)
	}
	var assetTable int
	require.NoError(t, sqlDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'studio_design_assets'`).Scan(&assetTable))
	require.Equal(t, 1, assetTable, "migration must create the asset table on an old database")

	repo := NewSQLiteRepository(sqlDB).(*sqliteRepository)
	doc, err := repo.Get(ctx, "design-old")
	require.NoError(t, err)
	require.Equal(t, "Old design", doc.Name)
	require.Equal(t, 4, doc.Revision)
	require.Equal(t, 2, len(mustFrames(t, doc)))
	require.JSONEq(t, `{"ink":"#111"}`, string(doc.Theme))
	require.JSONEq(t, `{}`, string(doc.Options), "an old row reads the column default")
	require.Nil(t, doc.Scenario)
	require.Nil(t, doc.Page)

	// The migrated row still saves through the new seam, including a binding
	// and design-owned chrome.
	scenario := "legacy"
	revision := doc.Revision
	updated, err := repo.Save(ctx, doc.ID, UpdateInput{
		Name:     doc.Name,
		Frames:   doc.Frames,
		Theme:    doc.Theme,
		Options:  json.RawMessage(`{"radius":20,"display":"sans"}`),
		Scenario: &scenario,
		Revision: &revision,
	})
	require.NoError(t, err)
	require.Equal(t, 5, updated.Revision)
	require.NotNil(t, updated.Scenario)
	require.Equal(t, scenario, *updated.Scenario)
	require.JSONEq(t, `{"radius":20,"display":"sans"}`, string(updated.Options))

	// Idempotent: a second boot must not fail or duplicate anything.
	require.NoError(t, Migrate(ctx, sqlDB))
}

func mustFrames(t *testing.T, doc Document) []map[string]any {
	t.Helper()
	var frames []map[string]any
	require.NoError(t, json.Unmarshal(doc.Frames, &frames))
	return frames
}
