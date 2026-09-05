package catalogsearch

import (
	"path/filepath"
	"testing"
)

func TestLayerRankOrdersBeforeScore(t *testing.T) {
	i := &Index{docs:[]Document{{CatalogID:"component.perfect", Kind:"component", Layer:LayerRank("component"), Description:"dashboard metrics recent activity page template"},{CatalogID:"templates.dashboard-page", Kind:"page-template", Layer:LayerRank("page-template"), Description:"dashboard"}}}
	got := i.Search("dashboard metrics recent activity", 10, "", "", "")
	if len(got) != 2 || got[0].CatalogID != "templates.dashboard-page" { t.Fatalf("results = %#v", got) }
}

func TestEveryDeclarationIsIndexed(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	i := New(); if err := i.Reindex(root); err != nil { t.Fatal(err) }
	discovered, err := filepath.Glob(filepath.Join(root,"catalog","assets","*","*.json")); if err != nil { t.Fatal(err) }
	indexed, _ := i.Status(); if indexed != len(discovered) { t.Fatalf("discovered=%d indexed=%d",len(discovered),indexed) }
}
