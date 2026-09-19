// Package candidates owns logo candidates: explore, import, refine with
// lineage, pick, reject and restore. A pick promotes a candidate (vectorizing a
// raster pick first) to the brand's canonical mark.
package candidates

import "time"

// Origin records how a candidate was produced.
type Origin string

const (
	OriginGenerated         Origin = "GENERATED"
	OriginImported          Origin = "IMPORTED"
	OriginEdited            Origin = "EDITED"
	OriginObjectRemoved     Origin = "OBJECT_REMOVED"
	OriginBackgroundRemoved Origin = "BACKGROUND_REMOVED"
	OriginVectorized        Origin = "VECTORIZED"
)

// Status is the candidate lifecycle state.
type Status string

const (
	StatusProposed   Status = "PROPOSED"
	StatusPicked     Status = "PICKED"
	StatusRejected   Status = "REJECTED"
	StatusSuperseded Status = "SUPERSEDED"
)

// Candidate is one proposed mark.
type Candidate struct {
	ID            string
	BrandID       string
	AssetID       string
	MediaType     string
	Concept       string
	Prompt        string
	Role          string
	Model         string
	Seed          int64
	Origin        Origin
	ParentID      string
	Status        Status
	Note          string
	GenerationRef string
	CreatedAt     time.Time
}

// RefineAction selects the refinement.
type RefineAction struct {
	Instruction      string
	MaskAssetID      string
	RemoveBackground bool
	Vectorize        *VectorizeOptions
}

// VectorizeOptions mirrors the image-tools vectorize parameters.
type VectorizeOptions struct {
	Colors                     int
	KeepColors                 []string
	DropBackgroundLayers       bool
	ClipToLargestRoundedRegion bool
	InsetPx                    float64
	TolerancePx                float64
	Smoothing                  bool
	MinAreaPx                  float64
}

// ExploreInput is one explore request.
type ExploreInput struct {
	BrandID        string
	Brief          string
	Concepts       []string
	Variations     int
	PreferVector   bool
	QualityPolicy  string
	FallbackPolicy string
	// AllowBYOK is accepted for older callers; the cloud tier is permitted unless
	// FallbackPolicy is local_only (see generation.BrandImageAllowBYOK).
	AllowBYOK bool
	// Role overrides the OpenRouter role concepts render with.
	Role string
	// StyleReferenceBrand (id or slug) makes that brand's approved mark the style
	// reference every concept is rendered to match.
	StyleReferenceBrand string
	// StyleReferenceAssetID uses one asset as the style reference instead.
	StyleReferenceAssetID string
}
