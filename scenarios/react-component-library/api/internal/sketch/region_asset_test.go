package sketch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/components"
	"testing"
)

type regionAssetReader struct {
	components.DependencyReader
	story string
}

func (r regionAssetReader) Get(_ context.Context, id string) (components.Component, error) {
	return components.Component{ID: id, CatalogID: id, Slug: "Card"}, nil
}
func (r regionAssetReader) GetVersion(_ context.Context, id, version string) (components.ComponentVersion, error) {
	h := sha256.Sum256([]byte(r.story))
	return components.ComponentVersion{ComponentID: id, Version: version, Status: components.VersionStatusReleased, DependencyLockPresent: true, Files: []components.ComponentVersionFile{{Path: "story.json", Content: r.story, ContentSHA256: hex.EncodeToString(h[:])}}}, nil
}
func TestRegionAssetUsesPublishedStoryAndPreservesCandidate(t *testing.T) {
	reader := regionAssetReader{story: `{"schemaVersion":5,"kind":"component","args":{"fields":[{"path":"children","kind":"text","required":true,"default":"A record"}]},"environment":{"fixtures":[]},"stories":[{"id":"ready","name":"Ready","role":"anatomy","args":{}}]}`}
	source := Snapshot{ContentHash: "candidate", Document: Document{Template: &AssetRef{Asset: "templates.page", Version: "1.0.0"}, Regions: []Region{{ID: "records"}}, Render: &RenderSettings{TemplateExport: "Page", Regions: []RenderRegion{{ID: "records", TemplateRegion: "body", Slot: []string{"regions", "body"}}}, Bindings: map[string]any{"$template": map[string]any{}}}}}
	catalog := []catalogcoverage.Asset{{ID: "templates.page", RegionAccepts: map[string]string{"body": "data-display"}}, {ID: "data-display.card", Kind: "component", Domain: "data-display"}}
	ref := AssetRef{Asset: "data-display.card", Version: "1.0.0"}
	got, err := SelectRegionAsset(context.Background(), source, "records", ref, "ready", reader, catalog)
	if err != nil {
		t.Fatal(err)
	}
	if got.Placements[0].Fills.Version != "1.0.0" || got.Render.Regions[0].Export != "Card" || got.Render.Regions[0].Story != "ready" {
		t.Fatal("exact selection provenance lost")
	}
	if got.Render.Bindings["records"].(map[string]any)["children"] != "A record" {
		t.Fatal("published fixture not used")
	}
	if len(source.Document.Placements) != 0 || source.Document.Render.Regions[0].Export != "" {
		t.Fatal("input candidate mutated")
	}
	if _, err := SelectRegionAsset(context.Background(), source, "records", ref, "invented", reader, catalog); err == nil {
		t.Fatal("invented fixture accepted")
	}
	catalog[1].Domain = "controls"
	if _, err := SelectRegionAsset(context.Background(), source, "records", ref, "ready", reader, catalog); err == nil {
		t.Fatal("incompatible filler accepted")
	}
}
