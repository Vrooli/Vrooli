package apply_test

import (
	"context"
	"encoding/json"
	"fmt"
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
	svc := apply.NewService(brands, assets, recorder, ws, nil, nil, nil)

	res, err := svc.Preview(context.Background(), apply.Request{BrandID: "b1", Scenario: "web-console"})
	require.NoError(t, err)
	require.True(t, res.DryRun)
	require.Equal(t, 3, res.BrandVersion)
	// colors, typography and identity produce actions; icons is skipped (no
	// branding declaration/renderer here).
	require.Len(t, res.Applied, 3)
	require.Len(t, res.Skipped, 1)
	// A preview writes nothing and records nothing.
	require.Zero(t, ws.WriteCount())
	require.Empty(t, recorder.Recorded())
}

func TestApply_WritesFilesAndRecordsAssignment(t *testing.T) {
	brands, assets, recorder, ws := newDeps(t)
	brands.Seed(fullBrand("b1", 5))
	ws.SeedScenario("web-console")
	svc := apply.NewService(brands, assets, recorder, ws, nil, nil, nil)

	res, err := svc.Apply(context.Background(), apply.Request{BrandID: "b1", Scenario: "web-console"})
	require.NoError(t, err)
	require.False(t, res.DryRun)
	// colors + typography + identity applied; icons skipped.
	require.Len(t, res.Applied, 3)

	// brand.css carries both the colors block and the appended typography block.
	css := string(ws.Written("web-console", "ui/src/styles/brand.css"))
	require.Contains(t, css, "--brand-primary: #112233")
	require.Contains(t, css, "--brand-heading-font: Inter")
	// manifest.json carries the merged identity provenance + theme color.
	manifest := string(ws.Written("web-console", "ui/public/manifest.json"))
	require.Contains(t, manifest, "_brand_display_name")
	require.Contains(t, manifest, "Acme")
	// The assignment is recorded once with exactly the applied elements.
	recorded := recorder.Recorded()
	require.Len(t, recorded, 1)
	require.Equal(t, "b1", recorded[0].BrandID)
	require.Equal(t, "web-console", recorded[0].Scenario)
	require.Equal(t, []string{"colors", "typography", "identity"}, recorded[0].Elements)
}

// fakeRenderer is a deterministic stand-in for image-tools' rasterize/
// icon_container, so the apply tests do not need a live image-tools.
type fakeRenderer struct{}

func (fakeRenderer) Rasterize(_ context.Context, svg []byte, w, h int, _ string) ([]byte, error) {
	return []byte(fmt.Sprintf("PNG %dx%d of %d svg bytes", w, h, len(svg))), nil
}

func (fakeRenderer) IconContainer(_ context.Context, _ []byte, format string, sizes []int) ([]byte, error) {
	return []byte(fmt.Sprintf("%s %d entries", format, len(sizes))), nil
}

type fakeStyles struct{ view apply.ContainerStyleView }

func (f fakeStyles) ContainerStyle(context.Context, string) (apply.ContainerStyleView, bool, error) {
	return f.view, true, nil
}

func TestApply_ProfilesIconSetAndMarkerBlock(t *testing.T) {
	brands, assets, recorder, ws := newDeps(t)
	brand := fullBrand("b1", 2)
	brand.MarkAssetID = "mark1"
	brands.Seed(brand)
	assets.SeedByID("mark1", apply.AssetContent{Filename: "logo.svg", Bytes: []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 80"><path fill="#ffffff" d="M10 10H90V70H10Z"/></svg>`)})
	ws.SeedScenario("web-console")
	ws.SeedFile("web-console", ".vrooli/service.json", []byte(`{"branding":{"brand":"aquila","targets":["web-public-v1"]}}`))
	ws.SeedFile("web-console", "ui/index.html", []byte(`<html><head><link rel="icon" href="/favicon.ico"><link rel="manifest" href="/manifest.json"></head><body></body></html>`))
	svc := apply.NewService(brands, assets, recorder, ws, fakeRenderer{}, fakeStyles{}, nil)

	res, err := svc.Apply(context.Background(), apply.Request{BrandID: "b1", Scenario: "web-console", Elements: []string{"icons"}})
	require.NoError(t, err)
	require.NotEmpty(t, res.Applied)

	// Targets live under the /public/* layout, not the URL root.
	require.NotEmpty(t, ws.Written("web-console", "ui/public/public/logo.svg"))
	require.NotEmpty(t, ws.Written("web-console", "ui/public/public/favicon-16.png"))
	require.NotEmpty(t, ws.Written("web-console", "ui/public/public/maskable-icon-512.png"))
	require.NotEmpty(t, ws.Written("web-console", "ui/public/public/og-image.png"))
	require.Nil(t, ws.Written("web-console", "ui/public/favicon-16.png"), "old root-layout writer must not be used")

	manifest := string(ws.Written("web-console", "ui/public/public/site.webmanifest"))
	require.Contains(t, manifest, `"src": "icon-192.png"`, "manifest srcs are relative")
	require.Contains(t, manifest, `"purpose": "maskable"`)

	// Every key validation's manifest-completeness rule requires. Writing only
	// name/short_name/theme_color/background_color/icons left every branded
	// scenario failing that rule identically.
	var manifestObj map[string]any
	require.NoError(t, json.Unmarshal([]byte(manifest), &manifestObj))
	for _, key := range []string{
		"name", "short_name", "description",
		"theme_color", "background_color",
		"display", "start_url", "id", "icons",
	} {
		require.Contains(t, manifestObj, key, "manifest-completeness requires %q", key)
		require.NotEmpty(t, manifestObj[key], "manifest key %q must not be empty", key)
	}
	// An icons-only apply must still declare the brand: brand-markers-applied is
	// otherwise satisfied only by CSS markers, which this element never writes.
	require.Contains(t, manifestObj, "_brand")

	html := string(ws.Written("web-console", "ui/index.html"))
	require.Contains(t, html, "brand-manager:icons:start")
	require.Contains(t, html, `href="/public/logo.svg"`)
	require.NotContains(t, html, `href="/favicon.ico"`, "old root link tags are replaced")

	// Idempotent: a second apply is byte-identical.
	_, err = svc.Apply(context.Background(), apply.Request{BrandID: "b1", Scenario: "web-console", Elements: []string{"icons"}})
	require.NoError(t, err)
	require.Equal(t, manifest, string(ws.Written("web-console", "ui/public/public/site.webmanifest")))
	require.Equal(t, html, string(ws.Written("web-console", "ui/index.html")))
}

func TestApply_PartialElementsSubset(t *testing.T) {
	brands, assets, recorder, ws := newDeps(t)
	brands.Seed(fullBrand("b1", 1))
	ws.SeedScenario("web-console")
	svc := apply.NewService(brands, assets, recorder, ws, nil, nil, nil)

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
	svc := apply.NewService(brands, assets, recorder, ws, nil, nil, nil)

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
	svc := apply.NewService(brands, assets, recorder, ws, nil, nil, nil)

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
	svc := apply.NewService(brands, assets, recorder, ws, nil, nil, nil)

	_, err := svc.Apply(context.Background(), apply.Request{BrandID: "ghost", Scenario: "web-console"})
	var notFound apply.ErrBrandNotFound
	require.ErrorAs(t, err, &notFound)
}

func TestApply_MissingScenarioIsNotFound(t *testing.T) {
	brands, assets, recorder, ws := newDeps(t)
	brands.Seed(fullBrand("b1", 1))
	svc := apply.NewService(brands, assets, recorder, ws, nil, nil, nil)

	_, err := svc.Apply(context.Background(), apply.Request{BrandID: "b1", Scenario: "missing"})
	var notFound apply.ErrScenarioNotFound
	require.ErrorAs(t, err, &notFound)
}

func TestApply_MissingInputIsInvalid(t *testing.T) {
	brands, assets, recorder, ws := newDeps(t)
	svc := apply.NewService(brands, assets, recorder, ws, nil, nil, nil)

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
	svc := apply.NewService(brands, assets, recorder, ws, nil, nil, nil)

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
