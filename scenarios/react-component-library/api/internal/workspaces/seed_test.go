package workspaces

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	db "github.com/vrooli/api-core/databasetest"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
)

func newSeedDB(t *testing.T) (*SQLiteRepository, string) {
	t.Helper()
	sqlDB := db.NewSQLite(t)
	ctx := context.Background()
	_, err := sqlDB.ExecContext(ctx, Schema())
	require.NoError(t, err)
	path := filepath.Join("..", "..", "..", "library", "workspace-seeds.json")
	raw, err := os.ReadFile(path)
	require.NoError(t, err, "the reviewed seed file is part of the scenario")
	result, err := Seed(ctx, sqlDB, raw)
	require.NoError(t, err, "the reviewed seed file must satisfy the aggregate schema")
	require.Equal(t, 16, result.Inserted)
	require.Zero(t, result.Skipped)
	return NewSQLiteRepository(sqlDB), string(raw)
}

func TestSeedIsIdempotentAndStocksTheTemplateShelf(t *testing.T) {
	repo, raw := newSeedDB(t)
	ctx := context.Background()

	second, err := Seed(ctx, repo.db.(*sql.DB), []byte(raw))
	require.NoError(t, err)
	require.Zero(t, second.Inserted, "a second boot must not duplicate the shelf")
	require.Equal(t, 16, second.Skipped)

	list, err := repo.List(ctx, workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_CURATED, "", 0)
	require.NoError(t, err)
	require.Len(t, list, 16)
	for _, summary := range list {
		require.Greater(t, summary.GetPageCount(), int32(0), summary.GetId())
		require.Greater(t, summary.GetDisplayCount(), int32(0), summary.GetId())
		require.Greater(t, summary.GetStateCount(), int32(0), summary.GetId())
	}

	// The shelf is content: every seed must render a real composition.
	for _, summary := range list {
		workspace, err := repo.Get(ctx, summary.GetId())
		require.NoError(t, err)
		for _, page := range workspace.GetPages() {
			require.NotEqual(t, "{}", page.GetSpecJson(), "%s must carry a composition", summary.GetId())
			for _, display := range page.GetDisplays() {
				for _, state := range display.GetStates() {
					require.Contains(t, state.GetArgsJson(), `"spec"`, "%s/%s must carry a state spec",
						summary.GetId(), display.GetId())
				}
			}
		}
	}
}

func TestCuratedWorkspaceForksButRefusesDirectWrites(t *testing.T) {
	repo, _ := newSeedDB(t)
	ctx := context.Background()
	original, err := repo.Get(ctx, "ws_curated_harbor")
	require.NoError(t, err)

	_, err = repo.Save(ctx, &workspacev1.SaveWorkspaceRequest{
		Workspace:    original,
		BaseRevision: original.GetRevision(),
	})
	require.ErrorIs(t, err, ErrCurated, "the shelf cannot drift under existing forks")

	require.ErrorIs(t, repo.Delete(ctx, original.GetId()), ErrCurated)

	fork, err := repo.Fork(ctx, original.GetId(), "Harbor, ours")
	require.NoError(t, err)
	require.Equal(t, original.GetId(), fork.GetOriginWorkspaceId())
	require.Equal(t, workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_FORKED, fork.GetProvenance())

	fork.Name = "Harbor, ours"
	saved, err := repo.Save(ctx, &workspacev1.SaveWorkspaceRequest{Workspace: fork, BaseRevision: fork.GetRevision()})
	require.NoError(t, err)
	require.Equal(t, int32(2), saved.GetRevision())

	stored, err := repo.Get(ctx, original.GetId())
	require.NoError(t, err)
	require.Equal(t, original.GetName(), stored.GetName(), "editing the fork never touches the template")
}

func TestSeedRejectsNonCuratedOrMalformedContent(t *testing.T) {
	sqlDB := db.NewSQLite(t)
	_, err := sqlDB.ExecContext(context.Background(), Schema())
	require.NoError(t, err)

	cases := map[string]string{
		"authored": `{"schemaVersion":1,"workspaces":[{"id":"ws_x","name":"x","provenance":"WORKSPACE_PROVENANCE_AUTHORED","revision":1,"pages":[{"id":"pg_x","name":"x","specJson":"{}","displays":[{"id":"dp_x","device":"DISPLAY_DEVICE_DESKTOP","states":[{"id":"st_x","name":"Default"}]}]}]}]}`,
		"revision": `{"schemaVersion":1,"workspaces":[{"id":"ws_x","name":"x","provenance":"WORKSPACE_PROVENANCE_CURATED","revision":4,"pages":[{"id":"pg_x","name":"x","specJson":"{}","displays":[{"id":"dp_x","device":"DISPLAY_DEVICE_DESKTOP","states":[{"id":"st_x","name":"Default"}]}]}]}]}`,
		"schema":   `{"schemaVersion":9,"workspaces":[]}`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Seed(context.Background(), sqlDB, []byte(raw))
			require.Error(t, err)
		})
	}
}

func TestSeedVersionAppliesReviewedChangesAndRetiresUnlistedTemplates(t *testing.T) {
	sqlDB := db.NewSQLite(t)
	ctx := context.Background()
	_, err := sqlDB.ExecContext(ctx, Schema())
	require.NoError(t, err)
	repo := NewSQLiteRepository(sqlDB)

	seed := func(version int, name string, includeSecond bool) string {
		second := ""
		if includeSecond {
			second = `,{"id":"ws_tpl_2","name":"Second","provenance":"WORKSPACE_PROVENANCE_CURATED","revision":1,` +
				`"createdAt":"2026-11-20T00:00:00.000Z","updatedAt":"2026-11-20T00:00:00.000Z",` +
				`"pages":[{"id":"pg_2","name":"Two","specJson":"{}","displays":[{"id":"dp_2","device":"DISPLAY_DEVICE_DESKTOP",` +
				`"states":[{"id":"st_2","name":"Default","argsJson":"{}"}]}]}]}`
		}
		return fmt.Sprintf(`{"schemaVersion":1,"seedVersion":%d,"workspaces":[`+
			`{"id":"ws_tpl_1","name":%q,"provenance":"WORKSPACE_PROVENANCE_CURATED","revision":1,`+
			`"createdAt":"2026-11-20T00:00:00.000Z","updatedAt":"2026-11-20T00:00:00.000Z",`+
			`"pages":[{"id":"pg_1","name":"One","specJson":"{}","displays":[{"id":"dp_1","device":"DISPLAY_DEVICE_DESKTOP",`+
			`"states":[{"id":"st_1","name":"Default","argsJson":"{}"}]}]}]}%s]}`, version, name, second)
	}

	first, err := Seed(ctx, sqlDB, []byte(seed(1, "First", true)))
	require.NoError(t, err)
	require.Equal(t, 2, first.Inserted)

	same, err := Seed(ctx, sqlDB, []byte(seed(1, "First", true)))
	require.NoError(t, err)
	require.Equal(t, 2, same.Skipped, "the same version is a no-op")

	changed, err := Seed(ctx, sqlDB, []byte(seed(2, "Reviewed name", false)))
	require.NoError(t, err)
	require.Equal(t, 1, changed.Updated)
	require.Equal(t, 1, changed.Removed, "an unlisted template is retired by the version bump")

	updated, err := repo.Get(ctx, "ws_tpl_1")
	require.NoError(t, err)
	require.Equal(t, "Reviewed name", updated.GetName())
	require.Equal(t, int32(2), updated.GetRevision(), "a reviewed change is a real revision")

	_, err = repo.Get(ctx, "ws_tpl_2")
	require.ErrorIs(t, err, ErrNotFound)

	_, err = Seed(ctx, sqlDB, []byte(seed(1, "Older", false)))
	require.Error(t, err, "an older seed file must not overwrite newer content")
}

func TestSeedUpdateConflictRollsBackWholeVersion(t *testing.T) {
	sqlDB := db.NewSQLite(t)
	ctx := context.Background()
	_, err := sqlDB.ExecContext(ctx, Schema())
	require.NoError(t, err)
	_, err = sqlDB.ExecContext(ctx, `CREATE UNIQUE INDEX workspaces_binding_unique
		ON workspaces (binding_scenario, binding_page)
		WHERE binding_scenario <> '' AND binding_page <> ''`)
	require.NoError(t, err)
	repo := NewSQLiteRepository(sqlDB)
	authored, err := repo.Create(ctx, &workspacev1.CreateWorkspaceRequest{
		Name: "Bound workspace", BindingScenario: "harbor", BindingPage: "welcome",
	})
	require.NoError(t, err)

	seed := func(version int, name string, binding bool, includeNew bool) string {
		bindingFields := `"bindingScenario":"","bindingPage":"",`
		if binding {
			bindingFields = `"bindingScenario":"harbor","bindingPage":"welcome",`
		}
		newEntry := ""
		if includeNew {
			newEntry = `,{"id":"ws_new","name":"New template","provenance":"WORKSPACE_PROVENANCE_CURATED","revision":1,` +
				`"createdAt":"2026-11-20T00:00:00.000Z","updatedAt":"2026-11-20T00:00:00.000Z",` +
				`"pages":[{"id":"pg_new","name":"New","specJson":"{}","displays":[{"id":"dp_new","device":"DISPLAY_DEVICE_DESKTOP",` +
				`"states":[{"id":"st_new","name":"Default","argsJson":"{}"}]}]}]}`
		}
		return fmt.Sprintf(`{"schemaVersion":1,"seedVersion":%d,"workspaces":[`+
			`{"id":"ws_bound_template","name":%q,"provenance":"WORKSPACE_PROVENANCE_CURATED","revision":1,`+
			`"createdAt":"2026-11-20T00:00:00.000Z","updatedAt":"2026-11-20T00:00:00.000Z",%s`+
			`"pages":[{"id":"pg_template","name":"Template","specJson":"{}","displays":[{"id":"dp_template","device":"DISPLAY_DEVICE_DESKTOP",`+
			`"states":[{"id":"st_template","name":"Default","argsJson":"{}"}]}]}]}%s]}`,
			version, name, bindingFields, newEntry)
	}
	_, err = Seed(ctx, sqlDB, []byte(seed(1, "Original", false, false)))
	require.NoError(t, err)

	_, err = Seed(ctx, sqlDB, []byte(seed(2, "Changed", true, true)))
	require.Error(t, err, "the binding update conflicts with an authored workspace")

	stored, err := repo.Get(ctx, "ws_bound_template")
	require.NoError(t, err)
	require.Equal(t, "Original", stored.GetName(), "failed seed updates must roll back")
	require.Equal(t, int32(1), stored.GetRevision())
	_, err = repo.Get(ctx, "ws_new")
	require.ErrorIs(t, err, ErrNotFound, "earlier inserts in the failed seed pass must roll back too")
	var applied int
	require.NoError(t, sqlDB.QueryRowContext(ctx, `SELECT version FROM workspace_seed_state WHERE id=1`).Scan(&applied))
	require.Equal(t, 1, applied, "failed version must not be recorded")
	authoredAfter, err := repo.Get(ctx, authored.GetId())
	require.NoError(t, err)
	require.Equal(t, "harbor", authoredAfter.GetBindingScenario())
	require.Equal(t, "welcome", authoredAfter.GetBindingPage())
}
