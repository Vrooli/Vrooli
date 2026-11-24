package search

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"react-component-library/internal/availability"
	"react-component-library/internal/components"
)

type catalogAvailabilityFixture struct {
	asset   components.Component
	version components.ComponentVersion
}

func (f catalogAvailabilityFixture) List(context.Context, components.SearchQuery) ([]components.Component, error) {
	return []components.Component{f.asset}, nil
}

func (catalogAvailabilityFixture) Get(context.Context, string) (components.Component, error) {
	panic("snapshot owns identity reads")
}

func (catalogAvailabilityFixture) GetByLibraryID(context.Context, string) (components.Component, error) {
	panic("snapshot owns identity reads")
}

func (f catalogAvailabilityFixture) GetVersion(context.Context, string, string) (components.ComponentVersion, error) {
	return f.version, nil
}

func TestUnifiedProjectionSupportsExactCatalogLookupAndRankedRetrieval(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "catalog", "assets", "controls", "button.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	write := func(value string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(`{"kind":"catalog-asset","asset":{"id":"controls.button","name":{"en":"Button"},"kind":"component","domain":"controls","description":"Primary action","targets":["react-vite"],"kits":["vrooli-default"]},"requiredCapabilities":["keyboard"],"regions":[{"id":"main"}]}`)
	index := newServiceAt(t, root, nil)
	if _, _, err := index.Reindex(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	content := "export const Button = () => null;"
	versionHash := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))
	availabilitySource := catalogAvailabilityFixture{
		asset:   components.Component{ID: "uuid", CatalogID: "controls.button", LibraryID: "button", LatestVersion: "2.3.4"},
		version: components.ComponentVersion{ComponentID: "uuid", Version: "2.3.4", Status: components.VersionStatusReleased, DependencyLockPresent: true, SourcePath: "Button.tsx", Content: content, ContentSHA256: versionHash},
	}
	factoryCalls := 0
	index.WithCatalogAvailability(func(ctx context.Context) (*availability.Snapshot, error) {
		factoryCalls++
		return availability.NewSnapshot(ctx, availabilitySource, func(context.Context, string, string) (string, error) { return "bundle", nil })
	})
	got, ok, err := index.LookupCatalogID(context.Background(), "controls.button")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || got.Kind != "component" || got.CatalogID != "controls.button" || len(got.RequiredCapabilities) != 1 || len(got.Targets) != 1 || len(got.Kits) != 1 {
		t.Fatalf("exact declaration lookup = %+v, %v", got, ok)
	}
	for _, id := range []string{"button", " controls.button "} {
		if _, ok, err := index.LookupCatalogID(context.Background(), id); err != nil {
			t.Fatal(err)
		} else if ok {
			t.Fatalf("lookup treated %q as an exact ID", id)
		}
	}
	results, err := index.SearchCatalog(context.Background(), "primary action", 10, "component", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].CatalogID != "controls.button" || !results[0].Availability.IsBuilt() || results[0].Availability.Version != "2.3.4" || factoryCalls != 1 {
		t.Fatalf("ranked catalog results = %+v", results)
	}
	if got, err := index.SearchCatalog(context.Background(), "primary action", 10, "page-template", "", ""); err != nil {
		t.Fatal(err)
	} else if len(got) != 0 {
		t.Fatalf("authored catalog kind filter ignored: %+v", got)
	}
}

func TestSearchCatalogReturnsPageTemplateMetadataFromUnifiedProjection(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "catalog", "assets", "templates", "landing-page.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	declaration := `{"kind":"catalog-asset","asset":{"id":"templates.landing-page","name":{"en":"LandingPage"},"kind":"page-template","domain":"templates","description":"A product landing page","targets":["react-vite"],"kits":["vrooli-default"]},"requiredCapabilities":["keyboard"],"regions":[{"id":"hero"},{"id":"features"}]}`
	if err := os.WriteFile(path, []byte(declaration), 0o600); err != nil {
		t.Fatal(err)
	}
	index := newServiceAt(t, root, nil)
	if _, _, err := index.Reindex(context.Background(), true); err != nil {
		t.Fatal(err)
	}

	results, err := index.SearchCatalog(context.Background(), "product landing page", 3, "page-template", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("page-template results = %+v", results)
	}
	template := results[0]
	if template.CatalogID != "templates.landing-page" || template.Name != "LandingPage" || template.Kind != "page-template" ||
		template.Domain != "templates" || len(template.Regions) != 2 || len(template.Targets) != 1 || len(template.Kits) != 1 || len(template.RequiredCapabilities) != 1 {
		t.Fatalf("page-template projection lost authored proposal metadata: %+v", template)
	}
	if template.Availability.CatalogID != template.CatalogID || template.Availability.IsBuilt() {
		t.Fatalf("unconfigured template availability should remain declared: %+v", template.Availability)
	}
}

func TestCatalogProjectionRefreshFailureRetainsLastGoodRowsAndHonorsCancellation(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "catalog", "assets", "controls", "button.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	write := func(value string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(`{"kind":"catalog-asset","asset":{"id":"controls.button","name":{"en":"Button"},"kind":"component"}}`)
	index := newServiceAt(t, root, nil)
	if _, _, err := index.Reindex(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	write(`{"kind":`)
	if _, _, err := index.Reindex(context.Background(), true); err == nil {
		t.Fatal("malformed refresh succeeded")
	}
	if got, ok, err := index.LookupCatalogID(context.Background(), "controls.button"); err != nil {
		t.Fatal(err)
	} else if !ok || got.Name != "Button" {
		t.Fatalf("last good projection lost: %+v, %v", got, ok)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := index.Reindex(ctx, true); err == nil {
		t.Fatal("cancelled reindex succeeded")
	}
}
