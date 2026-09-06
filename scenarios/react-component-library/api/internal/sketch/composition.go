package sketch

import (
	"fmt"
	"react-component-library/internal/preview"
)

// RenderSettings contains ports and preview fixtures, never a second set of
// selected assets. Asset identities always come from Template and Placements.
type RenderSettings struct {
	TemplateExport string          `json:"templateExport"`
	Regions        []RenderRegion  `json:"regions,omitempty"`
	Bindings       map[string]any  `json:"bindings,omitempty"`
	Fixtures       []RenderFixture `json:"fixtures,omitempty"`
}
type RenderRegion struct {
	Parent         string   `json:"parent,omitempty"`
	Story          string   `json:"story,omitempty"`
	TemplateRegion string   `json:"templateRegion,omitempty"`
	ID             string   `json:"id"`
	Export         string   `json:"export,omitempty"`
	Slot           []string `json:"slot"`
	Optional       bool     `json:"optional,omitempty"`
}
type RenderFixture struct {
	Target  string   `json:"target"`
	Asset   string   `json:"asset"`
	Version string   `json:"version"`
	State   string   `json:"state"`
	Field   string   `json:"field"`
	Prop    []string `json:"prop"`
}

// PrepareRender derives the executable layout from the exact authored snapshot.
// Unmapped semantic regions are errors: silently omitting them would present an
// incomplete page as a complete candidate. Mapped empty regions remain visible
// wireframes in the shared renderer.
func PrepareRender(snapshot Snapshot, missingLabel, failedLabel string) (preview.PreparedComposition, error) {
	doc := snapshot.Document
	if doc.Template == nil || doc.Render == nil {
		return preview.PreparedComposition{}, fmt.Errorf("select a published template and configure its rendering ports")
	}
	c := preview.Composition{Revision: snapshot.ContentHash, Template: preview.CompositionAsset{CatalogID: doc.Template.Asset, Version: doc.Template.Version, Export: doc.Render.TemplateExport}}
	regions := map[string]bool{}
	for _, list := range [][]Region{snapshot.DeclaredRegions, doc.Regions} {
		for _, r := range list {
			regions[r.ID] = true
		}
	}
	placements := map[string]Placement{}
	for _, p := range doc.Placements {
		if _, exists := placements[p.Region]; exists {
			return preview.PreparedComposition{}, fmt.Errorf("duplicate placement for %s", p.Region)
		}
		placements[p.Region] = p
		regions[p.Region] = true
	}
	mapped := map[string]bool{}
	for _, r := range doc.Render.Regions {
		if !regions[r.ID] || mapped[r.ID] {
			return preview.PreparedComposition{}, fmt.Errorf("unknown or duplicate rendering region %s", r.ID)
		}
		mapped[r.ID] = true
		item := preview.CompositionRegion{ID: r.ID, Parent: r.Parent, Slot: r.Slot, Required: !r.Optional}
		if p, exists := placements[r.ID]; exists && p.Fills.Asset != "" {
			item.Asset = &preview.CompositionAsset{CatalogID: p.Fills.Asset, Version: p.Fills.Version, Export: r.Export}
		}
		c.Regions = append(c.Regions, item)
	}
	for id := range regions {
		if !mapped[id] {
			return preview.PreparedComposition{}, fmt.Errorf("region %s has no template slot mapping", id)
		}
	}
	bindings := map[string]any{}
	for key, value := range doc.Render.Bindings {
		bindings[key] = value
	}
	bindings["$labels"] = map[string]any{"missing": missingLabel, "failed": failedLabel}
	var fixtures []preview.CompositionFixture
	for _, f := range doc.Render.Fixtures {
		fixtures = append(fixtures, preview.CompositionFixture{Target: f.Target, Asset: f.Asset, Version: f.Version, State: f.State, Field: f.Field, Prop: f.Prop})
	}
	return preview.PrepareComposition(c, bindings, fixtures)
}
