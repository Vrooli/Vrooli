// Package sketch owns the authored page-sketch document stored inside a
// scenario's experience page file.
package sketch

import (
	"encoding/json"
	"fmt"
)

type Document struct {
	Intent     *DesignIntent   `json:"intent,omitempty"`
	Render     *RenderSettings `json:"render,omitempty"`
	Viewport   string          `json:"viewport,omitempty"`
	Template   *AssetRef       `json:"template,omitempty"`
	Placements []Placement     `json:"placements,omitempty"`
	Unplaced   []Unplaced      `json:"unplaced,omitempty"`
	Notes      []Note          `json:"notes,omitempty"`
	Regions    []Region        `json:"regions,omitempty"`
}

type AssetRef struct {
	Asset   string `json:"asset"`
	Version string `json:"version,omitempty"`
}

type Fill struct {
	Asset       string `json:"asset,omitempty"`
	Version     string `json:"version,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	Intent      string `json:"intent,omitempty"`
}

type Placement struct {
	Region string `json:"region"`
	Fills  Fill   `json:"fills"`
	State  string `json:"state"`
	Note   string `json:"note,omitempty"`
	Intent string `json:"intent,omitempty"`
}

type Unplaced struct {
	Was       string     `json:"was"`
	Reason    string     `json:"reason"`
	Region    string     `json:"region,omitempty"`
	Placement *Placement `json:"placement,omitempty"`
}

type Note struct {
	Scope string `json:"scope"`
	Text  string `json:"text"`
}

type Region struct {
	Locked   bool     `json:"locked,omitempty"`
	Origin   string   `json:"origin,omitempty"`
	ID       string   `json:"id"`
	Note     string   `json:"note,omitempty"`
	Elements []string `json:"elements,omitempty"`
	Grid     *Grid    `json:"grid,omitempty"`
}

type Grid struct {
	X int `json:"x,omitempty"`
	Y int `json:"y,omitempty"`
	W int `json:"w,omitempty"`
	H int `json:"h,omitempty"`
}

// LockedRegionError requires a separate unlock revision before content changes.
type LockedRegionError struct{ Region string }

func (e *LockedRegionError) Error() string {
	return fmt.Sprintf("region %s is locked; unlock it in a separate revision before changing its content", e.Region)
}

// regionContent includes shared rendering inputs because changing a template or
// interaction graph can alter a locked region without changing its asset props.
func regionContent(d Document, id string) []byte {
	value := map[string]any{"self": json.RawMessage(regionOwnContent(d, id))}
	if d.Render != nil {
		parents := map[string]string{}
		for _, r := range d.Render.Regions {
			parents[r.ID] = r.Parent
		}
		related := map[string]json.RawMessage{}
		for child := range parents {
			seen := map[string]bool{}
			for parent := parents[child]; parent != "" && !seen[parent]; parent = parents[parent] {
				seen[parent] = true
				if parent == id {
					related[child] = regionOwnContent(d, child)
					break
				}
			}
		}
		value["descendants"] = related
		ancestors := map[string]json.RawMessage{}
		for parent := parents[id]; parent != ""; parent = parents[parent] {
			if _, seen := ancestors[parent]; seen {
				break
			}
			ancestors[parent] = regionOwnContent(d, parent)
		}
		value["ancestors"] = ancestors
	}
	raw, _ := json.Marshal(value)
	return raw
}

func regionOwnContent(d Document, id string) []byte {
	value := map[string]any{"template": d.Template, "viewport": d.Viewport, "intent": d.Intent}
	for _, r := range d.Regions {
		if r.ID == id {
			r.Locked = false
			value["region"] = r
		}
	}
	var placements []Placement
	for _, p := range d.Placements {
		if p.Region == id {
			placements = append(placements, p)
		}
	}
	value["placements"] = placements
	var notes []Note
	for _, n := range d.Notes {
		if n.Scope == id {
			notes = append(notes, n)
		}
	}
	value["notes"] = notes
	if d.Render != nil {
		value["templateExport"] = d.Render.TemplateExport
		var regions []RenderRegion
		for _, r := range d.Render.Regions {
			if r.ID == id {
				regions = append(regions, r)
			}
		}
		value["renderRegions"] = regions
		value["bindings"] = d.Render.Bindings[id]
		value["templateBindings"] = d.Render.Bindings["$template"]
		value["interactions"] = d.Render.Bindings["$preview"]
		var fixtures []RenderFixture
		for _, f := range d.Render.Fixtures {
			if f.Target == id || f.Target == "$template" {
				fixtures = append(fixtures, f)
			}
		}
		value["fixtures"] = fixtures
	}
	raw, _ := json.Marshal(value)
	return raw
}
func validateRegionLocks(before, after Document) error {
	for _, r := range before.Regions {
		if r.Locked && string(regionContent(before, r.ID)) != string(regionContent(after, r.ID)) {
			return &LockedRegionError{Region: r.ID}
		}
	}
	return nil
}
