package sketch

import "fmt"

type RegionRemap struct{ From, To string }
type TemplateChange struct {
	Document      Document
	NewlyUnplaced []Unplaced
}

// ChangeTemplate computes a reviewable result without mutating its input. The
// caller decides whether to publish after inspecting every displaced occupant.
func ChangeTemplate(doc Document, template AssetRef, slots []string, remaps []RegionRemap) (TemplateChange, error) {
	out := TemplateChange{Document: doc}
	if template.Asset == "" || len(slots) == 0 {
		return out, fmt.Errorf("template identity and declared regions are required")
	}
	targets := map[string]bool{}
	for _, slot := range slots {
		if slot == "" || targets[slot] {
			return out, fmt.Errorf("template contains empty or duplicate region %q", slot)
		}
		targets[slot] = true
	}
	source := map[string]bool{}
	for _, r := range doc.Regions {
		source[r.ID] = true
	}
	for _, p := range doc.Placements {
		source[p.Region] = true
	}
	mapping := map[string]string{}
	assigned := map[string]string{}
	for _, remap := range remaps {
		if !source[remap.From] {
			return out, fmt.Errorf("unknown source region %q", remap.From)
		}
		if !targets[remap.To] {
			return out, fmt.Errorf("unknown template region %q", remap.To)
		}
		if _, ok := mapping[remap.From]; ok {
			return out, fmt.Errorf("duplicate remap for %q", remap.From)
		}
		if previous, ok := assigned[remap.To]; ok {
			return out, fmt.Errorf("regions %q and %q both target %q", previous, remap.From, remap.To)
		}
		mapping[remap.From] = remap.To
		assigned[remap.To] = remap.From
	}
	for id := range source {
		if _, ok := mapping[id]; !ok && targets[id] {
			if previous, ok := assigned[id]; ok && previous != id {
				return out, fmt.Errorf("region %q already has an implicit assignment", id)
			}
			mapping[id] = id
			assigned[id] = id
		}
	}
	// Port paths and template props are contracts of an exact template version.
	// A new template must configure those contracts again; history retains the
	// previous settings for recovery. Renamed regions likewise need new ports.
	resetRender := doc.Template == nil || *doc.Template != template
	for from, to := range mapping {
		if from != to {
			resetRender = true
		}
	}
	if resetRender {
		out.Document.Render = nil
	}
	out.Document.Template = &template
	out.Document.Placements = nil
	out.Document.Unplaced = append([]Unplaced(nil), doc.Unplaced...)
	occupied := map[string]bool{}
	for _, placement := range doc.Placements {
		if next, ok := mapping[placement.Region]; ok {
			if occupied[next] {
				return TemplateChange{}, fmt.Errorf("multiple occupants target region %q", next)
			}
			occupied[next] = true
			placement.Region = next
			out.Document.Placements = append(out.Document.Placements, placement)
		} else {
			copy := placement
			was := placement.Fills.Asset
			if was == "" {
				was = placement.Fills.Placeholder
			}
			item := Unplaced{Was: was, Region: placement.Region, Placement: &copy, Reason: "No region mapping into template " + template.Asset}
			out.NewlyUnplaced = append(out.NewlyUnplaced, item)
			out.Document.Unplaced = append(out.Document.Unplaced, item)
		}
	}
	// Retain unmatched semantic definitions as authored intent. They are not
	// template occupants and cannot silently disappear when the shell changes.
	out.Document.Regions = append([]Region(nil), doc.Regions...)
	defined := map[string]bool{}
	for i, region := range out.Document.Regions {
		if next, ok := mapping[region.ID]; ok {
			out.Document.Regions[i].ID = next
		}
		id := out.Document.Regions[i].ID
		if defined[id] {
			return TemplateChange{}, fmt.Errorf("duplicate region definition %q", id)
		}
		defined[id] = true
	}
	for _, slot := range slots {
		if !defined[slot] {
			out.Document.Regions = append(out.Document.Regions, Region{ID: slot})
		}
	}
	out.Document.Notes = append([]Note(nil), doc.Notes...)
	for i, note := range out.Document.Notes {
		if next, ok := mapping[note.Scope]; ok {
			out.Document.Notes[i].Scope = next
		}
	}
	return out, nil
}
