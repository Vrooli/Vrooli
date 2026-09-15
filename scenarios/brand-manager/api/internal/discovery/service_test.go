package discovery_test

import (
	"context"
	"testing"

	"brand-manager/internal/discovery"
	mocks "brand-manager/internal/discovery/mocks"

	"github.com/stretchr/testify/require"
)

// seedBranded wires a scanner that "exists" for the scenario and carries a full
// spread of branding sources, so a scan matches every probe.
func seedBranded(t *testing.T, scenario string) *mocks.FakeScanner {
	t.Helper()
	sc := &mocks.FakeScanner{}
	sc.SeedScenario(scenario)
	sc.SeedFile(scenario, ".vrooli/service.json", []byte(`{"name":"Acme","description":"Acme app","tags":["a"]}`))
	sc.SeedFile(scenario, ".vrooli/branding.json", []byte(`{"site_name":"Acme","tagline":"We build","theme":{"primary":"#112233","accent":"#445566"},"logo_url":"https://x/logo.png"}`))
	sc.SeedFile(scenario, "ui/public/manifest.json", []byte(`{"name":"Acme PWA","background_color":"#000000","theme_color":"#ffffff"}`))
	sc.SeedFile(scenario, "ui/src/styles/theme.css", []byte(":root{--brand-primary:#112233;--brand-accent:#445566;}"))
	sc.SeedDir(scenario, "ui/public", []string{"favicon.ico", "logo.png", "other.txt"})
	return sc
}

func TestDiscover_FindsAllSources(t *testing.T) {
	sc := seedBranded(t, "web-console")
	svc := discovery.NewService(sc, &mocks.FakeBrandStore{}, nil)

	res, err := svc.Discover(context.Background(), "web-console")
	require.NoError(t, err)
	require.Equal(t, "web-console", res.Scenario)

	// service_json, branding_json, manifest, theme_css, favicon asset, logo asset.
	require.Len(t, res.Sources, 6)

	// Higher-priority branding.json wins display name + tagline; service.json
	// provides the description.
	require.Equal(t, "Acme", res.Draft.Identity.DisplayName)
	require.Equal(t, "We build", res.Draft.Identity.Tagline)
	require.Equal(t, "Acme app", res.Draft.Description)
	// branding.json colors take precedence; manifest only fills empty slots.
	require.Equal(t, "#112233", res.Draft.Colors.Primary)
	require.Equal(t, "#445566", res.Draft.Colors.Accent)
	require.Equal(t, "#000000", res.Draft.Colors.Background)
	// branding.json logo_url is higher priority than the asset scan.
	require.Equal(t, "https://x/logo.png", res.Draft.Identity.LogoPath)
	require.Equal(t, "ui/public/favicon.ico", res.Draft.Identity.FaviconPath)

	require.Greater(t, res.Confidence, 0.0)
	require.LessOrEqual(t, res.Confidence, 1.0)
}

func TestDiscover_AssetLogoFillsWhenNoLogoURL(t *testing.T) {
	sc := &mocks.FakeScanner{}
	sc.SeedScenario("web-console")
	sc.SeedDir("web-console", "ui/public", []string{"logo.svg"})
	svc := discovery.NewService(sc, &mocks.FakeBrandStore{}, nil)

	res, err := svc.Discover(context.Background(), "web-console")
	require.NoError(t, err)
	require.Len(t, res.Sources, 1)
	require.Equal(t, "ui/public/logo.svg", res.Draft.Identity.LogoPath)
}

func TestDiscover_EmptyScenarioYieldsNoSourcesAndSuggestions(t *testing.T) {
	sc := &mocks.FakeScanner{}
	sc.SeedScenario("blank")
	svc := discovery.NewService(sc, &mocks.FakeBrandStore{}, nil)

	res, err := svc.Discover(context.Background(), "blank")
	require.NoError(t, err)
	require.Empty(t, res.Sources)
	require.Zero(t, res.Confidence)
	require.False(t, res.Draft.Colors.HasAny())
	require.False(t, res.Draft.Identity.HasAny())
	// All three suggestions surface when nothing is discovered.
	require.Len(t, res.Suggestions, 3)
}

func TestDiscover_MalformedJSONIsSkippedNotFailed(t *testing.T) {
	sc := &mocks.FakeScanner{}
	sc.SeedScenario("web-console")
	sc.SeedFile("web-console", ".vrooli/service.json", []byte("{not json"))
	svc := discovery.NewService(sc, &mocks.FakeBrandStore{}, nil)

	res, err := svc.Discover(context.Background(), "web-console")
	require.NoError(t, err)
	require.Empty(t, res.Sources)
}

func TestDiscover_MissingScenarioIsNotFound(t *testing.T) {
	sc := &mocks.FakeScanner{}
	svc := discovery.NewService(sc, &mocks.FakeBrandStore{}, nil)

	_, err := svc.Discover(context.Background(), "ghost")
	var notFound discovery.ErrScenarioNotFound
	require.ErrorAs(t, err, &notFound)
}

func TestDiscover_MissingInputIsInvalid(t *testing.T) {
	svc := discovery.NewService(&mocks.FakeScanner{}, &mocks.FakeBrandStore{}, nil)

	_, err := svc.Discover(context.Background(), "   ")
	var invalid discovery.ErrInvalidDiscovery
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "scenario_name", invalid.Field)
}

func TestImport_CreatesBrandFromDiscoveredDraft(t *testing.T) {
	sc := seedBranded(t, "web-console")
	store := &mocks.FakeBrandStore{}
	svc := discovery.NewService(sc, store, nil)

	res, err := svc.Import(context.Background(), "web-console")
	require.NoError(t, err)
	require.Equal(t, "brand-1", res.Brand.ID)
	require.Equal(t, 1, res.Brand.Version)
	require.NotEmpty(t, res.Sources)

	recorded := store.Recorded()
	require.Len(t, recorded, 1)
	require.Equal(t, "Acme", recorded[0].Identity.DisplayName)
	require.Equal(t, "#112233", recorded[0].Colors.Primary)
}

func TestImport_DefaultsBrandNameToScenarioWhenUnnamed(t *testing.T) {
	// A scenario whose only signal is an asset → no display name discovered, so
	// the brand name defaults to the scenario name.
	sc := &mocks.FakeScanner{}
	sc.SeedScenario("web-console")
	sc.SeedDir("web-console", "ui/public", []string{"logo.png"})
	store := &mocks.FakeBrandStore{}
	svc := discovery.NewService(sc, store, nil)

	res, err := svc.Import(context.Background(), "web-console")
	require.NoError(t, err)
	require.Equal(t, "web-console", res.Brand.Name)
	require.Equal(t, "web-console", store.Recorded()[0].Name)
}

func TestImport_NoBrandingIsFailedPrecondition(t *testing.T) {
	sc := &mocks.FakeScanner{}
	sc.SeedScenario("blank")
	svc := discovery.NewService(sc, &mocks.FakeBrandStore{}, nil)

	_, err := svc.Import(context.Background(), "blank")
	var noBranding discovery.ErrNoBrandingFound
	require.ErrorAs(t, err, &noBranding)
}

func TestImport_MissingScenarioIsNotFound(t *testing.T) {
	svc := discovery.NewService(&mocks.FakeScanner{}, &mocks.FakeBrandStore{}, nil)

	_, err := svc.Import(context.Background(), "ghost")
	var notFound discovery.ErrScenarioNotFound
	require.ErrorAs(t, err, &notFound)
}
