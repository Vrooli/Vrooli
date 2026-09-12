package sketch

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func renderSnapshot() Snapshot {
	return Snapshot{ContentHash: "exact-page-hash", DeclaredRegions: []Region{{ID: "inspector"}}, Document: Document{Template: &AssetRef{Asset: "templates.collection-page", Version: "1.0.7"}, Placements: []Placement{{Region: "inspector", Fills: Fill{Asset: "controls.button", Version: "2.2.9"}}}, Render: &RenderSettings{TemplateExport: "CollectionPage", Regions: []RenderRegion{{ID: "inspector", Export: "Button", Slot: []string{"data", "inspector"}}}, Bindings: map[string]any{"$template": map[string]any{}, "inspector": map[string]any{"children": "Inspect"}}}}}
}
func TestPrepareRenderUsesAuthoredAssetIdentitiesAndRevision(t *testing.T) {
	s := renderSnapshot()
	got, err := PrepareRender(s, "Missing", "Failed")
	if err != nil {
		t.Fatal(err)
	}
	if got.Composition.Revision != s.ContentHash || got.Composition.Regions[0].Asset.CatalogID != "controls.button" {
		t.Fatalf("authored identity lost: %+v", got)
	}
	s.Document.Placements[0].Fills.Version = "3.0.0"
	got, err = PrepareRender(s, "Missing", "Failed")
	if err != nil || got.Composition.Regions[0].Asset.Version != "3.0.0" {
		t.Fatal("renderer retained a parallel stale asset selection", err)
	}
	if _, ok := s.Document.Render.Bindings["$labels"]; ok {
		t.Fatal("localized runtime labels mutated saved bindings")
	}
}
func TestPrepareRenderCannotSilentlyOmitDeclaredRegions(t *testing.T) {
	s := renderSnapshot()
	s.DeclaredRegions = append(s.DeclaredRegions, Region{ID: "unmapped"})
	if _, err := PrepareRender(s, "Missing", "Failed"); err == nil || !strings.Contains(err.Error(), "unmapped") {
		t.Fatal("missing semantic region was omitted", err)
	}
}
func TestPrepareRenderRetainsEmptyRegionsAndMissingFixtures(t *testing.T) {
	s := renderSnapshot()
	s.Document.Placements = nil
	got, err := PrepareRender(s, "Missing", "Failed")
	if err != nil || len(got.Composition.Regions) != 1 || got.Composition.Regions[0].Asset != nil {
		t.Fatal("empty region disappeared", err)
	}
	s = renderSnapshot()
	delete(s.Document.Render.Bindings, "inspector")
	got, err = PrepareRender(s, "Missing", "Failed")
	if err != nil || len(got.Gaps) != 1 || got.Gaps[0].Code != "fixture_missing" {
		t.Fatal("missing fixture not represented", err)
	}
}

func TestRenderSettingsAreRevisionedAndRemovable(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "scenarios", "demo", "experience", "pages", "home.json")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"sketch":{}}`), 0600); err != nil {
		t.Fatal(err)
	}
	store := NewStore(root)
	before, err := store.Read("demo", "home")
	if err != nil {
		t.Fatal(err)
	}
	doc := renderSnapshot().Document
	saved, err := store.Save("demo", "home", before.ContentHash, doc)
	if err != nil {
		t.Fatal(err)
	}
	read, err := store.Read("demo", "home")
	if err != nil || !reflect.DeepEqual(read.Document.Render, doc.Render) || saved.ContentHash == before.ContentHash {
		t.Fatal("render settings not revisioned", err)
	}
	doc.Render = nil
	cleared, err := store.Save("demo", "home", saved.ContentHash, doc)
	if err != nil || cleared.Document.Render != nil {
		t.Fatal("render settings cannot be removed", err)
	}
	if _, err := store.Save("demo", "home", saved.ContentHash, doc); err == nil {
		t.Fatal("stale render revision accepted")
	}
}
func TestTemplateChangeInvalidatesOldRenderPorts(t *testing.T) {
	doc := renderSnapshot().Document
	got, err := ChangeTemplate(doc, AssetRef{Asset: "templates.other", Version: "1.0.0"}, []string{"inspector"}, nil)
	if err != nil || got.Document.Render != nil || doc.Render == nil {
		t.Fatal("old template ports retained or caller mutated", err)
	}
}
