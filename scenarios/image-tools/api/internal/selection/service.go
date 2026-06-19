package selection

import (
	"context"
	"fmt"
)

// Tier labels where a segmentation ran (honest tier reporting, matching the AI
// engine's vocabulary). Only the built-in tier ships today; the model-backed
// SAM tiers are the documented upgrade (see selection.proto).
const (
	TierBuiltinCPU = "builtin-cpu"
	TierLocalCPU   = "local-cpu"
	TierLocalGPU   = "local-gpu"
)

// SegmentResult is the full outcome of one smart-select: the produced mask, the
// classified region, and its contextual edit menu — everything the UI needs to
// render the overlay + the context menu from one click.
type SegmentResult struct {
	MaskPNG      []byte
	Box          Box
	RegionClass  string
	Confidence   float64
	AreaFraction float64
	Tier         string
	ModelID      string
	Edits        []SuggestedEdit
	Warnings     []string
}

// Service runs the smart-select pipeline (segment → classify → suggest). It is
// stateless: the built-in region-grow is the always-runnable segmenter. A
// model-backed SAM path (mobilesam/sam-2.1/hq-sam, seeded) is the documented
// accuracy upgrade — when a caller requests one via ModelOverride today, the
// service falls back to the built-in and says so plainly rather than pretending
// a model ran (the no-vaporware bar Phase 1 established).
type Service struct{}

// NewService returns a Service.
func NewService() *Service { return &Service{} }

// Segment decodes src, produces the mask with the built-in region-grow,
// classifies the region, and attaches the contextual edit menu. The result's
// MaskPNG is the caller's to persist as a blob.
func (s *Service) Segment(_ context.Context, src []byte, p Params) (*SegmentResult, error) {
	img, err := DecodeInput(src)
	if err != nil {
		return nil, err
	}

	var warnings []string
	if p.ModelOverride != "" {
		warnings = append(warnings, fmt.Sprintf(
			"model-backed segmentation (%q) is not wired yet — used the built-in region-grow", p.ModelOverride))
	}

	c, err := compute(img, p)
	if err != nil {
		return nil, err
	}
	maskPNG, err := c.encodeMask()
	if err != nil {
		return nil, err
	}
	class, conf := Classify(c.work, c.mask, c.area)
	resolved, edits := Suggest(class)

	return &SegmentResult{
		MaskPNG:      maskPNG,
		Box:          c.box,
		RegionClass:  resolved,
		Confidence:   conf,
		AreaFraction: c.area,
		Tier:         TierBuiltinCPU,
		ModelID:      "",
		Edits:        edits,
		Warnings:     warnings,
	}, nil
}
