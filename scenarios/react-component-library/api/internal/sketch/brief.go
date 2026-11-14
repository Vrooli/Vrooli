package sketch

import (
	"fmt"
	"strings"
)

// BuildBrief derives a deterministic, read-only implementation checklist from
// the exact authored sketch. Tier numbers are stable ordering groups: page
// shell, region coverage, unresolved work, and render verification.
func BuildBrief(scenario, page string, doc Document) (string, []BriefItem, string) {
	items := make([]BriefItem, 0, 2+len(doc.Regions)+len(doc.Placements)+len(doc.Unplaced))
	add := func(tier int32, region, action, asset, constraint string) {
		items = append(items, BriefItem{Order: int32(len(items) + 1), Tier: tier, Region: region, Action: action, Asset: asset, Constraint: constraint})
	}
	if doc.Template != nil && doc.Template.Asset != "" {
		add(1, "$page", "Use the declared page template", assetRef(doc.Template.Asset, doc.Template.Version), "Keep this exact catalog identity unless a reviewed revision changes it.")
	} else {
		add(1, "$page", "Choose and configure a published page template", "", "The sketch has no template; map every required page region before rendering.")
	}
	for _, region := range doc.Regions {
		if region.Origin == "template" {
			continue
		}
		add(2, region.ID, "Map the authored region to a template port", "", region.Note)
	}
	for _, placement := range doc.Placements {
		fill := placement.Fills
		asset := assetRef(fill.Asset, fill.Version)
		action := "Place the selected catalog asset"
		constraint := placement.Intent
		if fill.Asset == "" {
			action = "Resolve the region placeholder"
			asset = fill.Placeholder
			if constraint == "" {
				constraint = fill.Intent
			}
		}
		add(2, placement.Region, action, asset, constraint)
	}
	for _, unplaced := range doc.Unplaced {
		region := unplaced.Region
		if region == "" && unplaced.Placement != nil {
			region = unplaced.Placement.Region
		}
		add(3, region, "Resolve or explicitly retain unplaced content: "+unplaced.Was, "", unplaced.Reason)
	}
	if len(items) == 0 {
		add(1, "$page", "Create a page composition from the brief", "", "Select catalog-backed assets and map them to declared page regions.")
	}
	closing := "Render the exact page revision at phone and desktop sizes, inspect both captures, resolve every blocking gap, and record a critique against the captured render hash."
	var markdown strings.Builder
	fmt.Fprintf(&markdown, "# Implementation brief: %s/%s\n\n", scenario, page)
	markdown.WriteString("Follow these steps in order. Catalog references include their pinned versions. This brief is derived from the saved sketch and does not modify it.\n\n")
	for _, item := range items {
		fmt.Fprintf(&markdown, "%d. **%s**", item.Order, item.Action)
		if item.Region != "" {
			fmt.Fprintf(&markdown, " — region `%s`", item.Region)
		}
		if item.Asset != "" {
			fmt.Fprintf(&markdown, " — `%s`", item.Asset)
		}
		markdown.WriteString(".\n")
		if item.Constraint != "" {
			fmt.Fprintf(&markdown, "   - Constraint: %s\n", item.Constraint)
		}
	}
	fmt.Fprintf(&markdown, "\n## Closing gate\n\n%s\n", closing)
	return markdown.String(), items, closing
}

func assetRef(asset, version string) string {
	if asset == "" {
		return ""
	}
	if version == "" {
		return asset
	}
	return asset + "@" + version
}

type BriefItem struct {
	Order      int32
	Tier       int32
	Region     string
	Action     string
	Asset      string
	Constraint string
}
