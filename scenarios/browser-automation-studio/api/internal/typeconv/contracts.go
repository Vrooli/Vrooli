package typeconv

import (
	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

// ToAssertionOutcome converts various types to *contracts.AssertionOutcome.
// Returns nil if conversion fails.
func ToAssertionOutcome(value any) *contracts.AssertionOutcome {
	switch v := value.(type) {
	case *contracts.AssertionOutcome:
		return v
	case contracts.AssertionOutcome:
		result := v
		return &result
	case map[string]any:
		result := &contracts.AssertionOutcome{
			Mode:          ToString(v["mode"]),
			Selector:      ToString(v["selector"]),
			Message:       ToString(v["message"]),
			Success:       ToBool(v["success"]),
			Negated:       ToBool(v["negated"]),
			CaseSensitive: ToBool(v["caseSensitive"]),
		}
		if expected, ok := v["expected"]; ok {
			result.Expected = expected
		}
		if actual, ok := v["actual"]; ok {
			result.Actual = actual
		}
		return result
	default:
		return nil
	}
}

// ToHighlightRegions converts various slice types to []*contracts.HighlightRegion.
func ToHighlightRegions(value any) []*contracts.HighlightRegion {
	regions := make([]*contracts.HighlightRegion, 0)
	switch v := value.(type) {
	case []*contracts.HighlightRegion:
		return v
	case []contracts.HighlightRegion:
		for i := range v {
			regions = append(regions, &v[i])
		}
	case []map[string]any:
		for _, item := range v {
			if region := ToHighlightRegion(item); region != nil {
				regions = append(regions, region)
			}
		}
	case []any:
		for _, item := range v {
			if region := ToHighlightRegion(item); region != nil {
				regions = append(regions, region)
			}
		}
	}
	return regions
}

// ToHighlightRegion converts a map to *contracts.HighlightRegion.
// Returns nil if conversion fails or required fields are missing.
func ToHighlightRegion(value any) *contracts.HighlightRegion {
	m, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	customRGBA := ToString(m["customRgba"])
	if customRGBA == "" {
		customRGBA = ToString(m["custom_rgba"])
	}
	if customRGBA == "" {
		// Backwards-compat alias (deprecated in proto; reserved field 4 in selectors.proto).
		customRGBA = ToString(m["color"])
	}
	region := contracts.HighlightRegion{
		Selector: ToString(m["selector"]),
		Padding:  int32(ToInt(m["padding"])),
	}
	if customRGBA != "" {
		region.CustomRgba = &customRGBA
	}
	if bbox := ToBoundingBox(m["boundingBox"]); bbox != nil {
		region.BoundingBox = bbox
	}
	if region.Selector == "" && region.BoundingBox == nil {
		return nil
	}
	return &region
}

// ToMaskRegions converts various slice types to []*contracts.MaskRegion.
func ToMaskRegions(value any) []*contracts.MaskRegion {
	regions := make([]*contracts.MaskRegion, 0)
	switch v := value.(type) {
	case []*contracts.MaskRegion:
		return v
	case []contracts.MaskRegion:
		for i := range v {
			regions = append(regions, &v[i])
		}
	case []map[string]any:
		for _, item := range v {
			if region := ToMaskRegion(item); region != nil {
				regions = append(regions, region)
			}
		}
	case []any:
		for _, item := range v {
			if region := ToMaskRegion(item); region != nil {
				regions = append(regions, region)
			}
		}
	}
	return regions
}

// ToMaskRegion converts a map to *contracts.MaskRegion.
// Returns nil if conversion fails or required fields are missing.
func ToMaskRegion(value any) *contracts.MaskRegion {
	m, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	region := contracts.MaskRegion{
		Selector: ToString(m["selector"]),
		Opacity:  ToFloat(m["opacity"]),
	}
	if bbox := ToBoundingBox(m["boundingBox"]); bbox != nil {
		region.BoundingBox = bbox
	}
	if region.Selector == "" && region.BoundingBox == nil {
		return nil
	}
	return &region
}

// ToElementFocus converts a map to *contracts.ElementFocus.
// Returns nil if conversion fails or required fields are missing.
func ToElementFocus(value any) *contracts.ElementFocus {
	m, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	focus := contracts.ElementFocus{
		Selector: ToString(m["selector"]),
	}
	if bbox := ToBoundingBox(m["boundingBox"]); bbox != nil {
		focus.BoundingBox = bbox
	}
	if focus.Selector == "" && focus.BoundingBox == nil {
		return nil
	}
	return &focus
}

// ToBoundingBox converts a map to *contracts.BoundingBox.
// Returns nil if conversion fails or map is empty.
func ToBoundingBox(value any) *contracts.BoundingBox {
	m, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	if len(m) == 0 {
		return nil
	}
	return &contracts.BoundingBox{
		X:      ToFloat(m["x"]),
		Y:      ToFloat(m["y"]),
		Width:  ToFloat(m["width"]),
		Height: ToFloat(m["height"]),
	}
}

// ToPoint converts a map to *contracts.Point.
// Returns nil if conversion fails.
func ToPoint(value any) *contracts.Point {
	m, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return &contracts.Point{
		X: ToFloat(m["x"]),
		Y: ToFloat(m["y"]),
	}
}

// ToPointSlice converts various types to []*contracts.Point.
func ToPointSlice(value any) []*contracts.Point {
	points := make([]*contracts.Point, 0)
	switch v := value.(type) {
	case []*contracts.Point:
		for _, item := range v {
			if item != nil {
				points = append(points, item)
			}
		}
		return points
	case []contracts.Point:
		for i := range v {
			points = append(points, &v[i])
		}
	case []map[string]any:
		for _, item := range v {
			if pt := ToPoint(item); pt != nil {
				points = append(points, pt)
			}
		}
	case []any:
		for _, item := range v {
			if pt := ToPoint(item); pt != nil {
				points = append(points, pt)
			}
		}
	case map[string]any:
		if pt := ToPoint(v); pt != nil {
			points = append(points, pt)
		}
	}
	return points
}
