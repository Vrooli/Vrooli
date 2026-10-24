package selection

import (
	internalselection "image-tools/internal/selection"

	selectionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/selection"
)

// editToProto maps an internal SuggestedEdit to its proto shape.
func editToProto(e internalselection.SuggestedEdit) *selectionv1.SuggestedEdit {
	return &selectionv1.SuggestedEdit{
		Id:             e.ID,
		Label:          e.Label,
		Description:    e.Description,
		Operation:      e.Operation,
		Prompt:         e.Prompt,
		RequiresPrompt: e.RequiresPrompt,
		RequiresMask:   e.RequiresMask,
		Params:         e.Params,
	}
}

func editsToProto(in []internalselection.SuggestedEdit) []*selectionv1.SuggestedEdit {
	out := make([]*selectionv1.SuggestedEdit, 0, len(in))
	for _, e := range in {
		out = append(out, editToProto(e))
	}
	return out
}

func classToProto(c internalselection.RegionClass) *selectionv1.RegionClassInfo {
	return &selectionv1.RegionClassInfo{
		Name:    c.Name,
		Summary: c.Summary,
		Edits:   editsToProto(c.Edits),
	}
}

func boxToProto(b internalselection.Box) *selectionv1.Box {
	return &selectionv1.Box{X: b.X, Y: b.Y, Width: b.W, Height: b.H}
}

// parseMode maps the proto SegmentMode to the internal Mode, defaulting an
// unspecified mode by what the caller supplied (points → point, box → box, else
// auto).
func parseMode(m selectionv1.SegmentMode, hasPoints, hasBox bool) internalselection.Mode {
	switch m {
	case selectionv1.SegmentMode_SEGMENT_MODE_POINT:
		return internalselection.ModePoint
	case selectionv1.SegmentMode_SEGMENT_MODE_BOX:
		return internalselection.ModeBox
	case selectionv1.SegmentMode_SEGMENT_MODE_AUTO:
		return internalselection.ModeAuto
	default:
		switch {
		case hasPoints:
			return internalselection.ModePoint
		case hasBox:
			return internalselection.ModeBox
		default:
			return internalselection.ModeAuto
		}
	}
}

// paramsFromProto converts a SegmentParams message to internal Params.
func paramsFromProto(p *selectionv1.SegmentParams) internalselection.Params {
	if p == nil {
		return internalselection.Params{Mode: internalselection.ModeAuto}
	}
	points := make([]internalselection.Point, 0, len(p.GetPoints()))
	for _, pt := range p.GetPoints() {
		points = append(points, internalselection.Point{X: pt.GetX(), Y: pt.GetY(), Negative: pt.GetNegative()})
	}
	var box *internalselection.Box
	if b := p.GetBox(); b != nil {
		box = &internalselection.Box{X: b.GetX(), Y: b.GetY(), W: b.GetWidth(), H: b.GetHeight()}
	}
	return internalselection.Params{
		Mode:          parseMode(p.GetMode(), len(points) > 0, box != nil),
		Points:        points,
		Box:           box,
		Tolerance:     p.GetTolerance(),
		ModelOverride: p.GetModelOverride(),
	}
}
