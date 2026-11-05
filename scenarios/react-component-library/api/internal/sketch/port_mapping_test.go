package sketch

import "testing"

func TestTemplatePortMappingPreservesNestedControls(t *testing.T) {
	source := Snapshot{Document: Document{Regions: []Region{{ID: "header", Origin: "template"}, {ID: "start", Note: "Start action"}, {ID: "collection", Origin: "template"}, {ID: "rows", Note: "Records"}}, Render: &RenderSettings{Regions: []RenderRegion{{ID: "header", Slot: []string{"header"}}, {ID: "start", Parent: "header", Slot: []string{"actions"}}, {ID: "collection", Slot: []string{"collection"}}}}}}
	if _, err := MapRegionPorts(source, []PortMapping{{Region: "rows", TemplateRegion: "header"}}); err == nil {
		t.Fatal("a containing region with authored controls was overwritten")
	}
	if _, err := MapRegionPorts(source, []PortMapping{{Region: "rows", TemplateRegion: "start"}}); err == nil {
		t.Fatal("nested control was treated as a template port")
	}
	result, err := MapRegionPorts(source, []PortMapping{{Region: "rows", TemplateRegion: "collection"}})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range result.Render.Regions {
		if r.ID == "start" && r.Parent == "header" {
			found = true
		}
	}
	if !found {
		t.Fatal("unrelated port mapping lost the nested control")
	}
}

func TestAuthoredRegionsReplaceOnlyGeneratedTemplatePlaceholders(t *testing.T) {
	source := Snapshot{ContentHash: "revision", DeclaredRegions: []Region{{ID: "thread-list-region", Note: "Preserve thread context"}}, Document: Document{Template: &AssetRef{Asset: "templates.collection-page", Version: "1.1.0"}, Regions: []Region{{ID: "collection", Origin: "template"}, {ID: "thread-list-region", Note: "Preserve thread context"}}, Render: &RenderSettings{TemplateExport: "CollectionPage", Regions: []RenderRegion{{ID: "collection", TemplateRegion: "collection", Slot: []string{"regions", "collection"}}}, Bindings: map[string]any{"$template": map[string]any{}}}}}
	mapped, err := MapRegionPorts(source, []PortMapping{{Region: "thread-list-region", TemplateRegion: "collection"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(mapped.Regions) != 1 || mapped.Regions[0].ID != "thread-list-region" || mapped.Regions[0].Note != "Preserve thread context" {
		t.Fatal("authored region was renamed or lost")
	}
	if mapped.Render.Regions[0].ID != "thread-list-region" || mapped.Render.Regions[0].TemplateRegion != "collection" {
		t.Fatal("semantic and template identities conflated")
	}
	source.Document = mapped
	if _, err := PrepareRender(source, "Missing", "Failed"); err != nil {
		t.Fatal("explicitly mapped semantic region could not render", err)
	}
	if source.DeclaredRegions[0].ID != "thread-list-region" {
		t.Fatal("root contract was renamed")
	}
}
func TestPortMappingRefusesAuthoredDisplacementAndCollisions(t *testing.T) {
	source := Snapshot{Document: Document{Regions: []Region{{ID: "collection", Note: "Existing authored purpose"}, {ID: "threads"}, {ID: "gate"}}, Render: &RenderSettings{Regions: []RenderRegion{{ID: "collection", TemplateRegion: "collection", Slot: []string{"regions", "collection"}}}}}}
	for _, mapping := range [][]PortMapping{{{Region: "threads", TemplateRegion: "collection"}}, {{Region: "threads", TemplateRegion: "collection"}, {Region: "gate", TemplateRegion: "collection"}}, {{Region: "threads", TemplateRegion: "unknown"}}} {
		if _, err := MapRegionPorts(source, mapping); err == nil {
			t.Fatal("unsafe mapping accepted", mapping)
		}
	}
	if source.Document.Render.Regions[0].ID != "collection" {
		t.Fatal("failed mapping mutated caller")
	}
}

func TestMovingAnAuthoredRegionLeavesAnExplicitVacantPort(t *testing.T) {
	source := Snapshot{Document: Document{Regions: []Region{{ID: "threads", Note: "Authored list"}, {ID: "inspector", Origin: "template"}}, Render: &RenderSettings{Regions: []RenderRegion{{ID: "threads", TemplateRegion: "collection", Slot: []string{"regions", "collection"}}, {ID: "inspector", TemplateRegion: "inspector", Slot: []string{"regions", "inspector"}}}}}}
	got, err := MapRegionPorts(source, []PortMapping{{Region: "threads", TemplateRegion: "inspector"}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Render.Regions[1].ID != "threads" || got.Render.Regions[0].ID == "threads" || len(got.Regions) != 2 {
		t.Fatal("moving a region duplicated or removed a port")
	}
}
