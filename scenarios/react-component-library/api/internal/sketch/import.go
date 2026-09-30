package sketch

import (
	"context"
	"fmt"
	"strings"

	"react-component-library/internal/catalogsearch"
)

type ImportSearch interface {
	SearchContext(context.Context, string, int, string, string, string) []catalogsearch.Result
}
type ImportCheck struct {
	Tier    int
	Matches []string
}
type ImportItem struct {
	Region       string
	Tier         int
	Match, State string
	Checks       []ImportCheck
}
type ImportResult struct {
	Document  Document
	Items     []ImportItem
	Templates []catalogsearch.Result
}

// Import proposes a composition from authored intent. Search candidates remain
// candidates; lexical similarity never proves an adopted implementation.
func Import(ctx context.Context, snapshot Snapshot, search ImportSearch) (ImportResult, error) {
	out := ImportResult{Document: snapshot.Document}
	if search == nil {
		return out, fmt.Errorf("catalog search is required before classifying reuse gaps")
	}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	regions := append([]Region(nil), snapshot.DeclaredRegions...)
	if len(regions) == 0 && len(out.Document.Regions) == 0 {
		purpose := strings.TrimSpace(snapshot.PagePurpose)
		if purpose == "" {
			purpose = "Support the primary task described by this page's experience contract."
		}
		regions = []Region{{ID: "primary-task", Note: purpose, Elements: append([]string(nil), snapshot.PageElements...)}}
	}
	known := map[string]bool{}
	for _, r := range out.Document.Regions {
		known[r.ID] = true
	}
	for _, r := range regions {
		if r.ID != "" && !known[r.ID] {
			out.Document.Regions = append(out.Document.Regions, r)
			known[r.ID] = true
		}
	}
	placements := map[string]Placement{}
	for _, p := range out.Document.Placements {
		placements[p.Region] = p
	}
	mapped := map[string]bool{}
	for _, region := range out.Document.Regions {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		for _, element := range region.Elements {
			mapped[element] = true
		}
		item := ImportItem{Region: region.ID, State: "unresolved"}
		fill := placements[region.ID].Fills
		exactQuery := fill.Asset
		if exactQuery == "" {
			exactQuery = region.ID
		}
		exact := search.SearchContext(ctx, exactQuery, 20, "", "", "")
		first := ImportCheck{Tier: 1}
		for _, candidate := range exact {
			if candidate.CatalogID == exactQuery {
				first.Matches = append(first.Matches, candidate.CatalogID)
			}
		}
		item.Checks = append(item.Checks, first)
		if len(first.Matches) > 0 {
			item.Tier = 1
			item.Match = first.Matches[0]
			item.State = "declared"
		} else {
			nearby := search.SearchContext(ctx, strings.TrimSpace(region.Note+" "+strings.ReplaceAll(region.ID, "-", " ")), 10, "", "", "")
			second := ImportCheck{Tier: 2}
			for _, candidate := range nearby {
				if candidate.Score > 0 && candidate.Kind != "page-template" {
					second.Matches = append(second.Matches, candidate.CatalogID)
				}
			}
			item.Checks = append(item.Checks, second)
			if len(second.Matches) > 0 {
				item.Tier = 2
				item.State = "candidate"
			} else {
				item.Tier = 3
				item.State = "gap"
			}
		}
		if fill.Asset != "" || fill.Placeholder != "" {
			item.State = "preserved"
		}
		out.Items = append(out.Items, item)
	}
	// Unassociated semantic elements remain explicit import obligations. Import
	// retries do not duplicate these records or remove earlier operator decisions.
	unplaced := map[string]bool{}
	for _, item := range out.Document.Unplaced {
		unplaced[item.Was] = true
	}
	for _, element := range snapshot.PageElements {
		if element != "" && !mapped[element] && !unplaced[element] {
			out.Document.Unplaced = append(out.Document.Unplaced, Unplaced{Was: element, Reason: "Authored element has no explicit composition region association"})
			unplaced[element] = true
		}
	}
	out.Templates = search.SearchContext(ctx, snapshot.PagePurpose, 3, "page-template", "", "")
	if err := ctx.Err(); err != nil {
		return out, err
	}
	return out, nil
}
