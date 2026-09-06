package sketch

import (
	"fmt"
	"strings"
)

func Place(doc Document, placement Placement) (Document, error) {
	if strings.TrimSpace(placement.Region) == "" || strings.TrimSpace(placement.Fills.Asset) == "" {
		return doc, fmt.Errorf("region and asset are required")
	}
	if placement.Fills.Placeholder != "" {
		return doc, fmt.Errorf("asset placement cannot also be a placeholder")
	}
	return insertPlacement(doc, placement)
}

func insertPlacement(doc Document, placement Placement) (Document, error) {
	for i := range doc.Placements {
		if doc.Placements[i].Region != placement.Region {
			continue
		}
		if placementsEqual(doc.Placements[i], placement) {
			return doc, nil
		}
		return doc, fmt.Errorf("region %q already has an occupant", placement.Region)
	}
	doc.Placements = append(doc.Placements, placement)
	return doc, nil
}

func AddPlaceholder(doc Document, placement Placement) (Document, error) {
	if len(strings.TrimSpace(placement.Fills.Intent)) < 20 {
		return doc, fmt.Errorf("placeholder intent must be at least 20 characters")
	}
	if strings.TrimSpace(placement.Fills.Placeholder) == "" {
		return doc, fmt.Errorf("placeholder name is required")
	}
	if strings.TrimSpace(placement.Region) == "" {
		return doc, fmt.Errorf("region is required")
	}
	if placement.Fills.Asset != "" || placement.Fills.Version != "" {
		return doc, fmt.Errorf("placeholder cannot select a published asset or version")
	}
	return insertPlacement(doc, placement)
}

func AddNote(doc Document, note Note) Document {
	for _, existing := range doc.Notes {
		if existing == note {
			return doc
		}
	}
	doc.Notes = append(doc.Notes, note)
	return doc
}

func Unplace(doc Document, region, reason string) (Document, error) {
	if len(strings.TrimSpace(reason)) < 10 {
		return doc, fmt.Errorf("unplace reason must be at least 10 characters")
	}
	for i, placement := range doc.Placements {
		if placement.Region != region {
			continue
		}
		was := placement.Fills.Asset
		if was == "" {
			was = placement.Fills.Placeholder
		}
		entry := Unplaced{Was: was, Reason: reason, Region: region, Placement: &placement}
		for _, existing := range doc.Unplaced {
			if existing.Was == entry.Was && existing.Region == entry.Region {
				return doc, nil
			}
		}
		doc.Placements = removePlacement(doc.Placements, i)
		doc.Unplaced = append(doc.Unplaced, entry)
		return doc, nil
	}
	return doc, fmt.Errorf("region %q has no placement", region)
}

func Restore(doc Document, was, region string) (Document, error) {
	for i, item := range doc.Unplaced {
		if item.Was != was {
			continue
		}
		if item.Placement == nil {
			return doc, fmt.Errorf("unplaced item %q has no restorable placement", was)
		}
		placement := *item.Placement
		placement.Region = region
		place := Place
		if placement.Fills.Placeholder != "" {
			place = AddPlaceholder
		}
		next, err := place(doc, placement)
		if err != nil {
			return doc, err
		}
		next.Unplaced = removeUnplaced(next.Unplaced, i)
		return next, nil
	}
	return doc, fmt.Errorf("unplaced item %q not found", was)
}

func removePlacement(items []Placement, index int) []Placement {
	if len(items) == 1 {
		return nil
	}
	return append(items[:index:index], items[index+1:]...)
}

func removeUnplaced(items []Unplaced, index int) []Unplaced {
	if len(items) == 1 {
		return nil
	}
	return append(items[:index:index], items[index+1:]...)
}

func placementsEqual(a, b Placement) bool {
	return a.Region == b.Region && a.Fills == b.Fills && a.State == b.State && a.Note == b.Note && a.Intent == b.Intent
}
