package reset_test

import (
	"context"
	"database/sql"
	"landing-page-react-vite-api/internal/adminreset"
	"landing-page-react-vite-api/internal/testutil/pgtest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"

	resetH "landing-page-react-vite-api/handlers/reset"

	contentschema "landing-page-react-vite-api/internal/content"
	downloadschema "landing-page-react-vite-api/internal/download"
	planschema "landing-page-react-vite-api/internal/plan"

	variantschema "landing-page-react-vite-api/internal/variant"
)

func setup(t *testing.T) *sql.DB {
	t.Helper()
	db := pgtest.NewDB(t)
	pgtest.Apply(t, db, variantschema.Schema, contentschema.Schema, planschema.Schema, downloadschema.Schema)
	for _, table := range []string{"content_sections", "variant_axes", "variants", "download_assets", "download_apps", "bundle_prices", "bundle_products"} {
		_, err := db.Exec("DELETE FROM " + table)
		require.NoError(t, err)
	}
	return db
}

// reseed installs the minimal default fixtures the reset restores.
func reseed(db *sql.DB) func(context.Context) error {
	return func(ctx context.Context) error {
		if _, err := db.ExecContext(ctx, `INSERT INTO variants (slug, name, status) VALUES ('control', 'Control', 'active') ON CONFLICT (slug) DO NOTHING`); err != nil {
			return err
		}
		_, err := db.ExecContext(ctx, `INSERT INTO download_apps (bundle_key, app_key, name) VALUES ('business_suite', 'app', 'App') ON CONFLICT (bundle_key, app_key) DO NOTHING`)
		return err
	}
}

func TestResetDisabled(t *testing.T) {
	db := setup(t)
	t.Setenv("ENABLE_ADMIN_RESET", "false")
	h := resetH.NewConnectHandler(resetH.Deps{Service: adminreset.NewService(db, reseed(db))})
	_, err := h.ResetDemoData(context.Background(), connect.NewRequest(&landingv1.ResetDemoDataRequest{}))
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
}

func TestResetSuccess(t *testing.T) {
	db := setup(t)
	t.Setenv("ENABLE_ADMIN_RESET", "true")

	var customID int64
	require.NoError(t, db.QueryRow(`INSERT INTO variants (slug, name, status) VALUES ('custom-reset', 'Custom', 'active') RETURNING id`).Scan(&customID))
	_, err := db.Exec(`INSERT INTO content_sections (variant_id, section_type, content, "order", enabled) VALUES ($1, 'hero', '{"headline":"Temp"}'::jsonb, 1, TRUE)`, customID)
	require.NoError(t, err)

	h := resetH.NewConnectHandler(resetH.Deps{Service: adminreset.NewService(db, reseed(db))})
	resp, err := h.ResetDemoData(context.Background(), connect.NewRequest(&landingv1.ResetDemoDataRequest{}))
	require.NoError(t, err)
	require.True(t, resp.Msg.Reset_)
	require.NotEmpty(t, resp.Msg.Timestamp)

	var customCount, controlCount, downloadCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM variants WHERE slug = 'custom-reset'`).Scan(&customCount))
	require.Equal(t, 0, customCount, "custom variant should be removed after reset")
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM variants WHERE slug = 'control'`).Scan(&controlCount))
	require.Equal(t, 1, controlCount, "control variant should be reseeded")
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM download_apps`).Scan(&downloadCount))
	require.Greater(t, downloadCount, 0, "download apps should be reseeded")
}
