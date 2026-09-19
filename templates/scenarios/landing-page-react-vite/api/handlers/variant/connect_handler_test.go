package variant_test

import (
	"context"
	"database/sql"
	"landing-page-react-vite-api/internal/testutil/pgtest"
	"landing-page-react-vite-api/internal/variantspace"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"

	variantH "landing-page-react-vite-api/handlers/variant"
	internalcontent "landing-page-react-vite-api/internal/content"

	internalvariant "landing-page-react-vite-api/internal/variant"
)

// testSpaceJSON mirrors the scenario's .vrooli/variant_space.json axes so the
// seeded control/variant-a selections and test selections validate.
var testSpaceJSON = []byte(`{
	"_name": "Test Space", "_schemaVersion": 1,
	"axes": {
		"persona": { "variants": [
			{ "id": "ops_leader", "label": "Ops Leader" },
			{ "id": "automation_freelancer", "label": "Automation Freelancer" },
			{ "id": "product_marketer", "label": "Product Marketer" }
		] },
		"jtbd": { "variants": [
			{ "id": "launch_bundle", "label": "Launch bundle" },
			{ "id": "scale_services", "label": "Scale services" },
			{ "id": "improve_conversions", "label": "Improve conversions" }
		] },
		"conversionStyle": { "variants": [
			{ "id": "self_serve", "label": "Self-serve" },
			{ "id": "demo_led", "label": "Demo-led" },
			{ "id": "founder_led", "label": "Founder-led" }
		] }
	}
}`)

func defaultAxes() map[string]string {
	return map[string]string{"persona": "automation_freelancer", "jtbd": "scale_services", "conversionStyle": "self_serve"}
}

func altAxes() map[string]string {
	return map[string]string{"persona": "ops_leader", "jtbd": "launch_bundle", "conversionStyle": "demo_led"}
}

func newHandler(t *testing.T) *variantH.Deps {
	t.Helper()
	db := pgtest.NewDB(t)
	pgtest.Apply(t, db, internalvariant.Schema, internalcontent.Schema)
	// Deterministic baseline: fresh control + variant-a with axes, like the seed.
	_, err := db.Exec(`DELETE FROM variants`)
	require.NoError(t, err)
	seedVariant(t, db, "control", "Control (Original)", 50, altAxes())
	seedVariant(t, db, "variant-a", "Variant A", 50, defaultAxes())

	space, err := variantspace.Parse(testSpaceJSON)
	require.NoError(t, err)
	content := internalcontent.NewService(db)
	return &variantH.Deps{Service: internalvariant.NewService(db, space, content)}
}

func seedVariant(t *testing.T, db *sql.DB, slug, name string, weight int, axes map[string]string) {
	t.Helper()
	var id int
	require.NoError(t, db.QueryRow(`
		INSERT INTO variants (slug, name, weight, status) VALUES ($1, $2, $3, 'active') RETURNING id`,
		slug, name, weight).Scan(&id))
	for axisID, value := range axes {
		_, err := db.Exec(`INSERT INTO variant_axes (variant_id, axis_id, variant_value) VALUES ($1, $2, $3)`, id, axisID, value)
		require.NoError(t, err)
	}
}

func TestSelectVariant(t *testing.T) {
	h := variantH.NewConnectHandler(*newHandler(t))
	resp, err := h.SelectVariant(context.Background(), connect.NewRequest(&landingv1.SelectVariantRequest{}))
	require.NoError(t, err)
	require.NotEmpty(t, resp.Msg.Variant.Slug)
	require.Equal(t, "active", resp.Msg.Variant.Status)
}

func TestGetVariant(t *testing.T) {
	h := variantH.NewConnectHandler(*newHandler(t))
	resp, err := h.GetVariant(context.Background(), connect.NewRequest(&landingv1.GetVariantRequest{Slug: "control"}))
	require.NoError(t, err)
	require.Equal(t, "control", resp.Msg.Variant.Slug)

	_, err = h.GetVariant(context.Background(), connect.NewRequest(&landingv1.GetVariantRequest{Slug: "nonexistent"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestCreateVariant(t *testing.T) {
	h := variantH.NewConnectHandler(*newHandler(t))
	resp, err := h.CreateVariant(context.Background(), connect.NewRequest(&landingv1.CreateVariantRequest{
		Slug: "test-variant", Name: "Test Variant", Description: "Test", Weight: 30, Axes: defaultAxes(),
	}))
	require.NoError(t, err)
	require.Equal(t, "test-variant", resp.Msg.Variant.Slug)
	require.EqualValues(t, 30, resp.Msg.Variant.Weight)
	require.Equal(t, "active", resp.Msg.Variant.Status)
	require.Equal(t, "automation_freelancer", resp.Msg.Variant.Axes["persona"])

	// Invalid weight rejected.
	_, err = h.CreateVariant(context.Background(), connect.NewRequest(&landingv1.CreateVariantRequest{
		Slug: "test-invalid", Name: "Invalid", Weight: 150, Axes: defaultAxes(),
	}))
	require.Error(t, err)
}

func TestCreateVariantRequiresAxes(t *testing.T) {
	h := variantH.NewConnectHandler(*newHandler(t))
	_, err := h.CreateVariant(context.Background(), connect.NewRequest(&landingv1.CreateVariantRequest{
		Slug: "axes-missing", Name: "Missing Axes", Weight: 40,
	}))
	require.Error(t, err)
}

func TestUpdateVariant(t *testing.T) {
	deps := newHandler(t)
	h := variantH.NewConnectHandler(*deps)
	_, err := h.CreateVariant(context.Background(), connect.NewRequest(&landingv1.CreateVariantRequest{
		Slug: "test-update", Name: "Update Test", Weight: 50, Axes: defaultAxes(),
	}))
	require.NoError(t, err)

	newWeight := int32(70)
	updated, err := h.UpdateVariant(context.Background(), connect.NewRequest(&landingv1.UpdateVariantRequest{Slug: "test-update", Weight: &newWeight}))
	require.NoError(t, err)
	require.EqualValues(t, 70, updated.Msg.Variant.Weight)

	newName := "Updated Name"
	updated, err = h.UpdateVariant(context.Background(), connect.NewRequest(&landingv1.UpdateVariantRequest{Slug: "test-update", Name: &newName}))
	require.NoError(t, err)
	require.Equal(t, "Updated Name", updated.Msg.Variant.Name)

	invalid := int32(150)
	_, err = h.UpdateVariant(context.Background(), connect.NewRequest(&landingv1.UpdateVariantRequest{Slug: "test-update", Weight: &invalid}))
	require.Error(t, err)

	updated, err = h.UpdateVariant(context.Background(), connect.NewRequest(&landingv1.UpdateVariantRequest{
		Slug: "test-update", Axes: &landingv1.AxesSelection{Values: altAxes()},
	}))
	require.NoError(t, err)
	require.Equal(t, "ops_leader", updated.Msg.Variant.Axes["persona"])
}

func TestArchiveVariant(t *testing.T) {
	h := variantH.NewConnectHandler(*newHandler(t))
	_, err := h.CreateVariant(context.Background(), connect.NewRequest(&landingv1.CreateVariantRequest{
		Slug: "test-archive", Name: "Archive Test", Weight: 50, Axes: defaultAxes(),
	}))
	require.NoError(t, err)

	resp, err := h.ArchiveVariant(context.Background(), connect.NewRequest(&landingv1.ArchiveVariantRequest{Slug: "test-archive"}))
	require.NoError(t, err)
	require.Equal(t, "archived", resp.Msg.Variant.Status)
	require.NotNil(t, resp.Msg.Variant.ArchivedAt)

	for i := 0; i < 10; i++ {
		selected, err := h.SelectVariant(context.Background(), connect.NewRequest(&landingv1.SelectVariantRequest{}))
		require.NoError(t, err)
		require.NotEqual(t, "test-archive", selected.Msg.Variant.Slug)
	}
}

func TestDeleteVariant(t *testing.T) {
	h := variantH.NewConnectHandler(*newHandler(t))
	_, err := h.CreateVariant(context.Background(), connect.NewRequest(&landingv1.CreateVariantRequest{
		Slug: "test-delete", Name: "Delete Test", Weight: 50, Axes: defaultAxes(),
	}))
	require.NoError(t, err)

	resp, err := h.DeleteVariant(context.Background(), connect.NewRequest(&landingv1.DeleteVariantRequest{Slug: "test-delete"}))
	require.NoError(t, err)
	require.True(t, resp.Msg.Deleted)

	got, err := h.GetVariant(context.Background(), connect.NewRequest(&landingv1.GetVariantRequest{Slug: "test-delete"}))
	require.NoError(t, err)
	require.Equal(t, "deleted", got.Msg.Variant.Status)
}

func TestListVariants(t *testing.T) {
	h := variantH.NewConnectHandler(*newHandler(t))
	resp, err := h.ListVariants(context.Background(), connect.NewRequest(&landingv1.ListVariantsRequest{}))
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(resp.Msg.Variants), 2)

	active, err := h.ListVariants(context.Background(), connect.NewRequest(&landingv1.ListVariantsRequest{StatusFilter: "active"}))
	require.NoError(t, err)
	for _, v := range active.Msg.Variants {
		require.Equal(t, "active", v.Status)
		require.NotEmpty(t, v.Axes)
	}
}

func TestWeightedSelection(t *testing.T) {
	h := variantH.NewConnectHandler(*newHandler(t))
	_, err := h.CreateVariant(context.Background(), connect.NewRequest(&landingv1.CreateVariantRequest{
		Slug: "test-heavy", Name: "Heavy", Weight: 90, Axes: defaultAxes(),
	}))
	require.NoError(t, err)
	_, err = h.CreateVariant(context.Background(), connect.NewRequest(&landingv1.CreateVariantRequest{
		Slug: "test-light", Name: "Light", Weight: 10, Axes: altAxes(),
	}))
	require.NoError(t, err)

	selections := map[string]int{}
	for i := 0; i < 1000; i++ {
		resp, err := h.SelectVariant(context.Background(), connect.NewRequest(&landingv1.SelectVariantRequest{}))
		require.NoError(t, err)
		selections[resp.Msg.Variant.Slug]++
	}
	require.NotZero(t, selections["test-heavy"])
	require.NotZero(t, selections["test-light"])
}

func TestExportImportSnapshot(t *testing.T) {
	h := variantH.NewConnectHandler(*newHandler(t))
	// Seed a section on control so the snapshot round-trips sections.
	deps := h
	_ = deps
	exp, err := h.ExportVariantSnapshot(context.Background(), connect.NewRequest(&landingv1.ExportVariantSnapshotRequest{Slug: "control"}))
	require.NoError(t, err)
	require.Equal(t, "control", exp.Msg.Snapshot.Slug)

	snap := exp.Msg.Snapshot
	snap.Name = "Control Renamed"
	imp, err := h.ImportVariantSnapshot(context.Background(), connect.NewRequest(&landingv1.ImportVariantSnapshotRequest{Slug: "control", Snapshot: snap}))
	require.NoError(t, err)
	require.Equal(t, "Control Renamed", imp.Msg.Variant.Name)
}
