package sketch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"react-component-library/internal/components"
	"testing"
)

type templatePortReader struct {
	components.DependencyReader
	version components.ComponentVersion
}

func (r templatePortReader) Get(context.Context, string) (components.Component, error) {
	return components.Component{ID: "id", CatalogID: "templates.page", Slug: "Page"}, nil
}
func (r templatePortReader) GetVersion(context.Context, string, string) (components.ComponentVersion, error) {
	return r.version, nil
}
func TestPublishedTemplatePortsPrepareWireframeWithoutInventingBindings(t *testing.T) {
	story := `{"schemaVersion":5,"kind":"component","args":{"fields":[{"path":"regions.main","region":"main","kind":"text"},{"path":"title","kind":"text","required":true,"default":"Records"}]},"environment":{"fixtures":[]},"stories":[{"id":"default","name":"Default","role":"anatomy","args":{"data":{"items":["One record"]}}}]}`
	digest := sha256.Sum256([]byte(story))
	reader := templatePortReader{version: components.ComponentVersion{ComponentID: "id", Version: "1.0.0", Status: components.VersionStatusReleased, DependencyLockPresent: true, Files: []components.ComponentVersionFile{{Path: "story.json", Content: story, ContentSHA256: hex.EncodeToString(digest[:])}}}}
	doc := Document{Template: &AssetRef{Asset: "templates.page", Version: "1.0.0"}, Regions: []Region{{ID: "main"}}}
	configured, obligations, err := ConfigureTemplateRender(context.Background(), doc, reader)
	if err != nil || len(obligations) != 0 || configured.Render.Regions[0].Slot[0] != "regions" || configured.Render.TemplateExport != "Page" {
		t.Fatal("declared ports not consumed", err, obligations)
	}
	if configured.Render.Bindings["$template"].(map[string]any)["title"] != "Records" {
		t.Fatal("template field default omitted")
	}
	prepared, err := PrepareRender(Snapshot{ContentHash: "candidate-hash", Document: configured}, "Missing", "Failed")
	if err != nil || len(prepared.Composition.Regions) != 1 || prepared.Composition.Regions[0].Asset != nil {
		t.Fatal("wireframe configuration failed or invented asset", err)
	}
	if doc.Render != nil {
		t.Fatal("input mutated")
	}
	doc.Regions = append(doc.Regions, Region{ID: "authored"})
	configured, obligations, err = ConfigureTemplateRender(context.Background(), doc, reader)
	if err != nil || len(obligations) != 1 {
		t.Fatal("unmapped authored region lost", err, obligations)
	}
	if _, err := PrepareRender(Snapshot{ContentHash: "candidate-hash", Document: configured}, "Missing", "Failed"); err == nil {
		t.Fatal("unmapped region disappeared from render")
	}
	reader.version.Files[0].Content += " "
	if _, _, err := ConfigureTemplateRender(context.Background(), doc, reader); err == nil {
		t.Fatal("tampered story supplied ports")
	}
}
