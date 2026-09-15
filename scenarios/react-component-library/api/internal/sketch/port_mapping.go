package sketch

import "fmt"

type PortMapping struct{ Region, TemplateRegion string }

// MapRegionPorts preserves semantic IDs. Only empty, generated template
// placeholders can be replaced by an authored page region.
func MapRegionPorts(snapshot Snapshot, mappings []PortMapping) (Document, error) {
	doc := snapshot.Document
	if doc.Render == nil {
		return doc, fmt.Errorf("declared template ports are required")
	}
	if len(mappings) == 0 || len(mappings) > 64 {
		return doc, fmt.Errorf("specify one to sixty-four mappings")
	}
	known := map[string]bool{}
	protected := map[string]bool{}
	for _, r := range snapshot.DeclaredRegions {
		known[r.ID] = true
		protected[r.ID] = true
	}
	for _, r := range doc.Regions {
		known[r.ID] = true
		if r.Origin != "template" || r.Note != "" || len(r.Elements) > 0 || r.Grid != nil {
			protected[r.ID] = true
		}
	}
	for _, p := range doc.Placements {
		known[p.Region] = true
		protected[p.Region] = true
	}
	for _, n := range doc.Notes {
		protected[n.Scope] = true
	}
	for key := range doc.Render.Bindings {
		if key != "$template" && key != "$labels" {
			protected[key] = true
		}
	}
	for _, fixture := range doc.Render.Fixtures {
		protected[fixture.Target] = true
	}
	ports := map[string]RenderRegion{}
	for _, r := range doc.Render.Regions {
		if r.Parent != "" {
			protected[r.Parent] = true
			continue
		}
		id := r.TemplateRegion
		if id == "" {
			id = r.ID
		}
		if _, exists := ports[id]; exists {
			return doc, fmt.Errorf("duplicate template port %s", id)
		}
		ports[id] = r
	}
	assigned := map[string]string{}
	sources := map[string]bool{}
	for _, m := range mappings {
		if !known[m.Region] {
			return doc, fmt.Errorf("unknown authored region %s", m.Region)
		}
		if _, exists := ports[m.TemplateRegion]; !exists {
			return doc, fmt.Errorf("unknown template port %s", m.TemplateRegion)
		}
		if sources[m.Region] || assigned[m.TemplateRegion] != "" {
			return doc, fmt.Errorf("region and port mappings must be one-to-one")
		}
		sources[m.Region] = true
		assigned[m.TemplateRegion] = m.Region
	}
	settings := *doc.Render
	settings.Regions = nil
	removed := map[string]bool{}
	rendered := map[string]bool{}
	var vacant []Region
	for _, port := range doc.Render.Regions {
		if port.Parent != "" {
			settings.Regions = append(settings.Regions, port)
			rendered[port.ID] = true
			continue
		}
		target := port.TemplateRegion
		if target == "" {
			target = port.ID
		}
		if region := assigned[target]; region != "" && region != port.ID {
			if protected[port.ID] && !sources[port.ID] {
				return doc, fmt.Errorf("port %s contains authored region %s; explicitly remap that region first", target, port.ID)
			}
			if !sources[port.ID] {
				removed[port.ID] = true
			}
			port.ID = region
			// An export describes the old region asset, not this port. Retain an
			// existing source region's export when it moves; otherwise require one.
			port.Export = ""
			port.Story = ""
			for _, old := range doc.Render.Regions {
				if old.ID == region {
					port.Export = old.Export
					port.Story = old.Story
				}
			}
		}
		if assigned[target] == "" && sources[port.ID] {
			// A moved semantic region leaves a visible empty template slot behind.
			id := "slot-" + target
			for suffix := 1; known[id]; suffix++ {
				id = fmt.Sprintf("slot-%s-%d", target, suffix)
			}
			known[id] = true
			vacant = append(vacant, Region{ID: id, Origin: "template"})
			port.ID = id
			port.Export = ""
			port.Story = ""
		}
		port.TemplateRegion = target
		if rendered[port.ID] {
			return doc, fmt.Errorf("region %s would occupy multiple ports", port.ID)
		}
		rendered[port.ID] = true
		settings.Regions = append(settings.Regions, port)
	}
	doc.Regions = nil
	for _, r := range snapshot.Document.Regions {
		if !removed[r.ID] {
			doc.Regions = append(doc.Regions, r)
		}
	}
	doc.Regions = append(doc.Regions, vacant...)
	doc.Render = &settings
	return doc, nil
}
