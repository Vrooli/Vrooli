package sketch

import (
	"context"
	"fmt"

	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/components"
)

// SelectRegionAsset fills a mapped empty region using one exact published
// asset and explicit story. Existing authored occupants require unplacement.
func SelectRegionAsset(ctx context.Context, snapshot Snapshot, region string, ref AssetRef, storyID string, reader components.DependencyReader, catalog []catalogcoverage.Asset) (Document, error) {
	doc := snapshot.Document
	if doc.Render == nil || doc.Template == nil || reader == nil || storyID == "" {
		return doc, fmt.Errorf("mapped template, published reader, and explicit story are required")
	}
	index := -1
	for i, r := range doc.Render.Regions {
		if r.ID == region {
			index = i
		}
	}
	if index < 0 {
		return doc, fmt.Errorf("region %s has no template port", region)
	}
	var selected, template *catalogcoverage.Asset
	for i := range catalog {
		if catalog[i].ID == ref.Asset {
			selected = &catalog[i]
		}
		if catalog[i].ID == doc.Template.Asset {
			template = &catalog[i]
		}
	}
	if selected == nil || template == nil {
		return doc, fmt.Errorf("selected asset and template must have exact catalog declarations")
	}
	port := doc.Render.Regions[index].TemplateRegion
	if port == "" {
		port = doc.Render.Regions[index].ID
	}
	if accepts := template.RegionAccepts[port]; accepts != "" && accepts != selected.Domain {
		return doc, fmt.Errorf("template port %s accepts %s, not %s", port, accepts, selected.Domain)
	}
	switch selected.Kind {
	case "component", "primitive", "pattern", "navigation":
	default:
		return doc, fmt.Errorf("asset kind %s is not a renderable region filler", selected.Kind)
	}
	asset, contract, err := readPublishedStory(ctx, ref, reader)
	if err != nil {
		return doc, err
	}
	props, err := components.ResolveStoryArgs(contract, storyID)
	if err != nil {
		return doc, err
	}
	// Copy before insertion: a failed validation must not mutate the candidate.
	doc.Placements = append([]Placement(nil), doc.Placements...)
	doc, err = Place(doc, Placement{Region: region, Fills: Fill{Asset: ref.Asset, Version: ref.Version}, State: "planned"})
	if err != nil {
		return snapshot.Document, err
	}
	settings := *doc.Render
	settings.Regions = append([]RenderRegion(nil), settings.Regions...)
	settings.Regions[index].Export = asset.Slug
	settings.Regions[index].Story = storyID
	settings.Bindings = map[string]any{}
	for key, value := range doc.Render.Bindings {
		settings.Bindings[key] = value
	}
	settings.Bindings[region] = props
	// Old fixture projections must not overwrite the selected story's props.
	settings.Fixtures = nil
	for _, fixture := range doc.Render.Fixtures {
		if fixture.Target != region {
			settings.Fixtures = append(settings.Fixtures, fixture)
		}
	}
	doc.Render = &settings
	snapshot.Document = doc
	if _, err := PrepareRender(snapshot, "Missing fixture", "Region failed"); err != nil {
		return Document{}, err
	}
	return doc, ctx.Err()
}
