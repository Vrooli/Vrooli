package workspaces

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
)

// TestSchemaIsEnsureSchemasCompatible exercises the exact boot seam main.go
// uses: api-core applies the declared schema twice and reconciles declared
// columns. A schema that only works through a raw Exec would hide a boot
// failure until production.
func TestSchemaIsEnsureSchemasCompatible(t *testing.T) {
	sqlDB := db.NewSQLite(t)
	ctx := context.Background()
	provider := apidb.SchemaProviderFunc(Schema)

	require.NoError(t, apidb.EnsureSchemas(ctx, sqlDB, provider))
	require.NoError(t, apidb.EnsureSchemas(ctx, sqlDB, provider), "re-applying the schema is a no-op")

	for table, columns := range map[string][]string{
		"workspaces":                                  {"id", "name", "summary", "provenance", "tags_json", "binding_scenario", "binding_page", "theme_json", "options_json", "origin_workspace_id", "revision", "created_at", "updated_at"},
		"workspace_pages":                             {"id", "workspace_id", "name", "route", "page_type_id", "sort_order", "spec_json"},
		"workspace_displays":                          {"id", "page_id", "device", "width", "height", "sort_order"},
		"workspace_states":                            {"id", "display_id", "name", "args_json", "scenario_data_json", "status", "capture_id", "sort_order"},
		"workspace_state_capture_evidence":            {"workspace_id", "state_id", "request_token", "source_revision", "capture_id", "evidence_json", "captured_at"},
		"workspace_capture_receipts":                  {"id", "workspace_id", "state_id", "source_revision", "capture_id", "attempt_id", "request_token", "evidence_json", "captured_at"},
		"workspace_visual_comparisons":                {"id", "workspace_id", "state_id", "baseline_receipt_id", "current_receipt_id", "result_json", "heatmap_ref", "created_at"},
		"workspace_state_source_baselines":            {"id", "workspace_id", "state_id", "asset_id", "sha256", "source_locator", "created_at"},
		"workspace_state_source_baseline_acceptances": {"baseline_id", "workspace_id", "state_id", "accepted_by", "accepted_at", "acceptance_method"},
		"workspace_source_visual_comparisons":         {"id", "workspace_id", "state_id", "source_baseline_id", "current_receipt_id", "result_json", "heatmap_ref", "created_at"},
		"workspace_state_capture_attempts":            {"workspace_id", "state_id", "source_revision", "attempt_id", "request_token", "state", "error", "created_at", "updated_at"},
		"workspace_revisions":                         {"id", "workspace_id", "revision", "snapshot_json", "actor", "created_at"},
		"workspace_assets":                            {"id", "workspace_id", "name", "mime", "bytes", "width", "height", "fidelity", "data", "created_at"},
		"workspace_migrations": {
			"source", "source_id", "workspace_id",
			"migrated_at",
		},
	} {
		live, err := tableColumns(ctx, sqlDB, table)
		require.NoError(t, err)
		for _, column := range columns {
			require.Truef(t, live[column], "%s must declare %s", table, column)
		}
	}
}

func TestSchemaEnforcesCascadeAndRevisionUniqueness(t *testing.T) {
	sqlDB := db.NewSQLite(t)
	ctx := context.Background()
	require.NoError(t, apidb.EnsureSchemas(ctx, sqlDB, apidb.SchemaProviderFunc(Schema)))

	repo := NewSQLiteRepository(sqlDB)
	created, err := repo.Create(ctx, &workspacev1.CreateWorkspaceRequest{Name: "Cascade"})
	require.NoError(t, err)

	// A second snapshot at the same revision is not history.
	_, err = sqlDB.ExecContext(ctx, `INSERT INTO workspace_revisions
		(id, workspace_id, revision, snapshot_json, actor, created_at)
		VALUES ('rv_duplicate', ?, 1, '{}', '', '2026-11-19T12:00:00.000Z')`, created.Id)
	require.Error(t, err)

	require.NoError(t, repo.Delete(ctx, created.Id))
	var pages int
	require.NoError(t, sqlDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM workspace_pages WHERE workspace_id = ?`, created.Id).Scan(&pages))
	require.Zero(t, pages)
}
