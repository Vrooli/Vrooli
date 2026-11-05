package contracts

import (
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
)

// ToAssertionOutcome converts various types to *AssertionOutcome.
// Returns nil if conversion fails.
func ToAssertionOutcome(value any) *AssertionOutcome {
	switch v := value.(type) {
	case *AssertionOutcome:
		return v
	case AssertionOutcome:
		result := v
		return &result
	case map[string]any:
		result := &AssertionOutcome{
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

// ToHighlightRegions converts various slice types to []*HighlightRegion.
func ToHighlightRegions(value any) []*HighlightRegion {
	regions := make([]*HighlightRegion, 0)
	switch v := value.(type) {
	case []*HighlightRegion:
		return v
	case []HighlightRegion:
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

// ToHighlightRegion converts a map to *HighlightRegion.
// Returns nil if conversion fails or required fields are missing.
func ToHighlightRegion(value any) *HighlightRegion {
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
	region := HighlightRegion{
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

// ToMaskRegions converts various slice types to []*MaskRegion.
func ToMaskRegions(value any) []*MaskRegion {
	regions := make([]*MaskRegion, 0)
	switch v := value.(type) {
	case []*MaskRegion:
		return v
	case []MaskRegion:
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

// ToMaskRegion converts a map to *MaskRegion.
// Returns nil if conversion fails or required fields are missing.
func ToMaskRegion(value any) *MaskRegion {
	m, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	region := MaskRegion{
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

// ToElementFocus converts a map to *ElementFocus.
// Returns nil if conversion fails or required fields are missing.
func ToElementFocus(value any) *ElementFocus {
	m, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	focus := ElementFocus{
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

// ToBoundingBox converts a map to *BoundingBox.
// Returns nil if conversion fails or map is empty.
func ToBoundingBox(value any) *BoundingBox {
	m, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	if len(m) == 0 {
		return nil
	}
	return &BoundingBox{
		X:      ToFloat(m["x"]),
		Y:      ToFloat(m["y"]),
		Width:  ToFloat(m["width"]),
		Height: ToFloat(m["height"]),
	}
}

// ToPoint converts a map to *Point.
// Returns nil if conversion fails.
func ToPoint(value any) *Point {
	m, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return &Point{
		X: ToFloat(m["x"]),
		Y: ToFloat(m["y"]),
	}
}

// ToPointSlice converts various types to []*Point.
func ToPointSlice(value any) []*Point {
	points := make([]*Point, 0)
	switch v := value.(type) {
	case []*Point:
		for _, item := range v {
			if item != nil {
				points = append(points, item)
			}
		}
		return points
	case []Point:
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

// ConditionOutcomeToProto converts a native ConditionOutcome to proto.
func ConditionOutcomeToProto(condition *ConditionOutcome) *basbase.ConditionOutcome {
	if condition == nil {
		return nil
	}

	pb := &basbase.ConditionOutcome{
		Outcome: condition.Outcome,
		Negated: condition.Negated,
	}

	if condition.Type != "" {
		pb.Type = &condition.Type
	}
	if condition.Operator != "" {
		pb.Operator = &condition.Operator
	}
	if condition.Variable != "" {
		pb.Variable = &condition.Variable
	}
	if condition.Selector != "" {
		pb.Selector = &condition.Selector
	}
	if condition.Expression != "" {
		pb.Expression = &condition.Expression
	}
	if condition.Actual != nil {
		pb.Actual = AnyToJsonValue(condition.Actual)
	}
	if condition.Expected != nil {
		pb.Expected = AnyToJsonValue(condition.Expected)
	}

	return pb
}

// ProtoToConditionOutcome converts a proto ConditionOutcome to native.
func ProtoToConditionOutcome(pb *basbase.ConditionOutcome) *ConditionOutcome {
	if pb == nil {
		return nil
	}

	return &ConditionOutcome{
		Type: pb.GetType(), Outcome: pb.GetOutcome(), Negated: pb.GetNegated(),
		Operator: pb.GetOperator(), Variable: pb.GetVariable(), Selector: pb.GetSelector(),
		Expression: pb.GetExpression(), Actual: JsonValueToAny(pb.GetActual()),
		Expected: JsonValueToAny(pb.GetExpected()),
	}
}
