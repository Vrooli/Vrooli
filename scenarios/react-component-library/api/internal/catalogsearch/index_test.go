package catalogsearch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRelevantComponentOutranksBarelyRelevantTemplate(t *testing.T) {
	i := &Index{docs: []Document{{CatalogID: "component.perfect", Kind: "component", Layer: LayerRank("component"), Description: "dashboard metrics recent activity page template"}, {CatalogID: "templates.dashboard-page", Kind: "page-template", Layer: LayerRank("page-template"), Description: "dashboard"}}}
	got := i.Search("dashboard metrics recent activity", 10, "", "", "")
	if len(got) != 2 || got[0].CatalogID != "component.perfect" {
		t.Fatalf("results = %#v", got)
	}
}

func TestEveryDeclarationIsIndexed(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	i := New()
	if err := i.Reindex(root); err != nil {
		t.Fatal(err)
	}
	discovered, err := filepath.Glob(filepath.Join(root, "catalog", "assets", "*", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	indexed, _ := i.Status()
	if indexed != len(discovered) {
		t.Fatalf("discovered=%d indexed=%d", len(discovered), indexed)
	}
}

func TestRefreshFailurePreservesLastGoodProjection(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "catalog", "assets", "controls", "button.json")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	write := func(value string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(`{"kind":"catalog-asset","asset":{"id":"controls.button","name":{"en":"Button","fr":"Bouton"},"kind":"component","domain":"controls"}}`)
	index := New()
	if err := index.Reindex(root); err != nil {
		t.Fatal(err)
	}
	before := index.Diagnostics()
	write(`{"kind":`)
	if err := index.Reindex(root); err == nil {
		t.Fatal("malformed refresh succeeded")
	}
	after := index.Diagnostics()
	if !after.Stale || after.LastError == "" || after.Count != 1 || !after.IndexedAt.Equal(before.IndexedAt) {
		t.Fatalf("lost last-good state: %+v", after)
	}
	results := index.Search("Button", 10, "", "", "")
	if len(results) != 1 || results[0].CatalogID != "controls.button" {
		t.Fatalf("lost results: %+v", results)
	}
	write(`{"kind":"catalog-asset","asset":{"id":"controls.button","name":"Renamed","kind":"component"}}`)
	if err := index.Reindex(root); err != nil {
		t.Fatal(err)
	}
	if index.Diagnostics().Stale {
		t.Fatal("successful refresh remains stale")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := index.ReindexContext(ctx, root); err == nil {
		t.Fatal("cancelled refresh succeeded")
	}
	if got := index.Search("Renamed", 10, "", "", ""); len(got) != 1 {
		t.Fatalf("cancel discarded results: %+v", got)
	}
}

func TestMissingCatalogCannotEraseSuccessfulIndex(t *testing.T) {
	i := &Index{docs: []Document{{CatalogID: "controls.button", Name: "Button"}}}
	if err := i.Reindex(t.TempDir()); err == nil {
		t.Fatal("missing catalog accepted")
	}
	if got := i.Search("Button", 1, "", "", ""); len(got) != 1 {
		t.Fatalf("last good result erased: %v", got)
	}
}

func TestSearchLocalizedTermsAndTemplateTieBreak(t *testing.T) {
	i := &Index{docs: []Document{
		{CatalogID: "component.button", Name: "操作", Kind: "component", Layer: LayerRank("component")},
		{CatalogID: "templates.actions", Name: "操作", Kind: "page-template", Layer: LayerRank("page-template")},
	}}
	got := i.Search("操作", 10, "", "", "")
	if len(got) != 2 || got[0].CatalogID != "templates.actions" {
		t.Fatalf("localized ranking: %+v", got)
	}
}
