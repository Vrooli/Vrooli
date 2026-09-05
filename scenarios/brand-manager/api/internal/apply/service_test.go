package apply_test

import (
	"context"
	"strings"
	"testing"

	"brand-manager/internal/apply"
	mocks "brand-manager/internal/apply/mocks"

	"github.com/stretchr/testify/require"
)

// fullBrand is a brand with every facet populated, so a default (all-elements)
// apply produces an action for colors, typography, and identity.
func fullBrand(id string, version int) apply.BrandView {
	return apply.BrandView{
		ID:          id,
		Version:     version,
		DisplayName: "Acme",
		Tagline:     "We make things",
		Colors:      apply.Colors{Primary: "#112233", Secondary: "#445566"},
		Typography:  apply.Typography{HeadingFont: "Inter", BodyFont: "Inter"},
	}
}

func newDeps(t *testing.T) (*mocks.FakeBrandStore, *mocks.FakeAssetStore, *mocks.FakeAssignmentRecorder, *mocks.FakeWorkspace) {
	t.Helper()
	return &mocks.FakeBrandStore{}, &mocks.FakeAssetStore{}, &mocks.FakeAssignmentRecorder{}, &mocks.FakeWorkspace{}
}

func TestPreview_PlansWithoutWriting(t *testing.T) {
	brands, assets, recorder, ws := newDeps(t)
	brands.Seed(fullBrand("b1", 3))
	ws.SeedScenario("web-console")
	svc := apply.NewService(brands, assets, recorder, ws, nil)

	res, err := svc.Preview(context.Background(), apply.Request{BrandID: "b1", Scenario: "web-console"})
	require.NoError(t, err)
	require.True(t, res.DryRun)
	require.Equal(t, 3, res.BrandVersion)
	// colors, typography, identity, and icons (manifest metadata from the brand
	// colors) produce actions; favicon + logo are skipped (no assets seeded).
	require.Len(t, res.Applied, 4)
	require.Len(t, res.Skipped, 2)
	// A preview writes nothing and records nothing.
	require.Zero(t, ws.WriteCount())
	require.Empty(t, recorder.Recorded())
}

func TestApply_WritesFilesAndRecordsAssignment(t *testing.T) {
	brands, assets, recorder, ws := newDeps(t)
	brands.Seed(fullBrand("b1", 5))
	assets.Seed("b1", apply.ElementLogo, apply.AssetContent{Filename: "logo.png", Bytes: []byte("PNGDATA")})
	ws.SeedScenario("web-console")
	svc := apply.NewService(brands, assets, recorder, ws, nil)

	res, err := svc.Apply(context.Background(), apply.Request{BrandID: "b1", Scenario: "web-console"})
	require.NoError(t, err)
	require.False(t, res.DryRun)
	// colors + typography + identity + icons(manifest) + logo applied; favicon skipped.
	require.Len(t, res.Applied, 5)

	// brand.css carries both the colors block and the appended typography block.
	css := string(ws.Written("web-console", "ui/src/styles/brand.css"))
	require.Contains(t, css, "--brand-primary: #112233")
	require.Contains(t, css, "--brand-heading-font: Inter")
	// manifest.json carries the merged identity provenance + theme color.
	manifest := string(ws.Written("web-console", "ui/public/manifest.json"))
	require.Contains(t, manifest, "_brand_display_name")
	require.Contains(t, manifest, "Acme")
	require.Contains(t, manifest, "theme_color")
	// logo bytes copied verbatim into ui/public.
	require.Equal(t, []byte("PNGDATA"), ws.Written("web-console", "ui/public/logo.png"))

	// The assignment is recorded once with exactly the applied elements.
	recorded := recorder.Recorded()
	require.Len(t, recorded, 1)
	require.Equal(t, "b1", recorded[0].BrandID)
	require.Equal(t, "web-console", recorded[0].Scenario)
	require.Equal(t, []string{"colors", "typography", "identity", "icons", "logo"}, recorded[0].Elements)
}

func TestApply_InstallsIconSetAndManifestIconsIdempotently(t *testing.T) {
	brands, assets, recorder, ws := newDeps(t)
	brands.Seed(fullBrand("b1", 2))
	// A derived icon set: two favicons (transparent) + one maskable (solid).
	assets.Seed("b1", "favicon-16", apply.AssetContent{Filename: "favicon-16.png", Bytes: []byte("F16")})
	assets.Seed("b1", "favicon-32", apply.AssetContent{Filename: "favicon-32.png", Bytes: []byte("F32")})
	assets.Seed("b1", "maskable-icon-192", apply.AssetContent{Filename: "maskable-icon-192.png", Bytes: []byte("M192")})
	ws.SeedScenario("web-console")
	svc := apply.NewService(brands, assets, recorder, ws, nil)

	res, err := svc.Apply(context.Background(), apply.Request{BrandID: "b1", Scenario: "web-console", Elements: []string{"icons"}})
	require.NoError(t, err)
	// three icon file copies + one manifest write.
	require.Len(t, res.Applied, 4)
	require.Equal(t, []byte("F16"), ws.Written("web-console", "ui/public/favicon-16.png"))
	require.Equal(t, []byte("M192"), ws.Written("web-console", "ui/public/maskable-icon-192.png"))

	manifest := string(ws.Written("web-console", "ui/public/manifest.json"))
	require.Contains(t, manifest, `"src": "/favicon-16.png"`)
	require.Contains(t, manifest, `"sizes": "192x192"`)
	require.Contains(t, manifest, `"purpose": "maskable"`)
	require.Contains(t, manifest, "theme_color")

	// Idempotent: a second apply produces a byte-identical manifest.
	first := ws.Written("web-console", "ui/public/manifest.json")
	_, err = svc.Apply(context.Background(), apply.Request{BrandID: "b1", Scenario: "web-console", Elements: []string{"icons"}})
	require.NoError(t, err)
	require.Equal(t, first, ws.Written("web-console", "ui/public/manifest.json"), "re-apply is byte-identical (no duplicate icon entries)")
}

func TestApply_PartialElementsSubset(t *testing.T) {
	brands, assets, recorder, ws := newDeps(t)
	brands.Seed(fullBrand("b1", 1))
	ws.SeedScenario("web-console")
	svc := apply.NewService(brands, assets, recorder, ws, nil)

	res, err := svc.Apply(context.Background(), apply.Request{
		BrandID:  "b1",
		Scenario: "web-console",
		Elements: []string{"colors"},
	})
	require.NoError(t, err)
	require.Len(t, res.Applied, 1)
	require.Equal(t, "colors", res.Applied[0].Element)
	require.Equal(t, 1, ws.WriteCount())
	require.NotEmpty(t, recorder.Recorded())
	require.Equal(t, []string{"colors"}, recorder.Recorded()[0].Elements)
}

func TestApply_UnknownElementIsSkipped(t *testing.T) {
	brands, assets, recorder, ws := newDeps(t)
	brands.Seed(fullBrand("b1", 1))
	ws.SeedScenario("web-console")
	svc := apply.NewService(brands, assets, recorder, ws, nil)

	res, err := svc.Apply(context.Background(), apply.Request{
		BrandID:  "b1",
		Scenario: "web-console",
		Elements: []string{"bogus"},
	})
	require.NoError(t, err)
	require.Empty(t, res.Applied)
	require.Len(t, res.Skipped, 1)
	require.Equal(t, "unknown element", res.Skipped[0].Reason)
	// Nothing applied → no assignment recorded.
	require.Empty(t, recorder.Recorded())
}

func TestApply_NoFacetIsSkippedNotFailed(t *testing.T) {
	brands, assets, recorder, ws := newDeps(t)
	brands.Seed(apply.BrandView{ID: "b1", Version: 1}) // empty brand: no colors/typography/identity
	ws.SeedScenario("web-console")
	svc := apply.NewService(brands, assets, recorder, ws, nil)

	res, err := svc.Apply(context.Background(), apply.Request{BrandID: "b1", Scenario: "web-console"})
	require.NoError(t, err)
	require.Empty(t, res.Applied)
	require.Len(t, res.Skipped, len(apply.AllElements))
	require.Zero(t, ws.WriteCount())
	require.Empty(t, recorder.Recorded())
}

func TestApply_UnknownBrandIsNotFound(t *testing.T) {
	brands, assets, recorder, ws := newDeps(t)
	ws.SeedScenario("web-console")
	svc := apply.NewService(brands, assets, recorder, ws, nil)

	_, err := svc.Apply(context.Background(), apply.Request{BrandID: "ghost", Scenario: "web-console"})
	var notFound apply.ErrBrandNotFound
	require.ErrorAs(t, err, &notFound)
}

func TestApply_MissingScenarioIsNotFound(t *testing.T) {
	brands, assets, recorder, ws := newDeps(t)
	brands.Seed(fullBrand("b1", 1))
	svc := apply.NewService(brands, assets, recorder, ws, nil)

	_, err := svc.Apply(context.Background(), apply.Request{BrandID: "b1", Scenario: "missing"})
	var notFound apply.ErrScenarioNotFound
	require.ErrorAs(t, err, &notFound)
}

func TestApply_MissingInputIsInvalid(t *testing.T) {
	brands, assets, recorder, ws := newDeps(t)
	svc := apply.NewService(brands, assets, recorder, ws, nil)

	_, err := svc.Apply(context.Background(), apply.Request{Scenario: "web-console"})
	var invalid apply.ErrInvalidApply
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "brand_id", invalid.Field)

	_, err = svc.Apply(context.Background(), apply.Request{BrandID: "b1"})
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "scenario_name", invalid.Field)
}

func TestApply_ReapplyConverges(t *testing.T) {
	brands, assets, recorder, ws := newDeps(t)
	brands.Seed(fullBrand("b1", 1))
	ws.SeedScenario("web-console")
	svc := apply.NewService(brands, assets, recorder, ws, nil)

	_, err := svc.Apply(context.Background(), apply.Request{BrandID: "b1", Scenario: "web-console", Elements: []string{"colors"}})
	require.NoError(t, err)
	first := string(ws.Written("web-console", "ui/src/styles/brand.css"))

	_, err = svc.Apply(context.Background(), apply.Request{BrandID: "b1", Scenario: "web-console", Elements: []string{"colors"}})
	require.NoError(t, err)
	second := string(ws.Written("web-console", "ui/src/styles/brand.css"))

	// Re-applying colors overwrites the same managed file — no accumulation.
	require.Equal(t, first, second)
	require.Equal(t, 1, strings.Count(second, "brand-manager:colors"))
}
