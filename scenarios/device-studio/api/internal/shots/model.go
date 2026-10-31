package shots

import (
	"time"

	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
)

type Shot struct {
	ID           string
	ProjectID    string
	Name         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	HeadRevision int64
}

type Revision struct {
	ShotID              string
	Revision            int64
	CreatedAt           time.Time
	Name                string
	Snapshot            *shotsv1.ShotSnapshot
	RecipeSHA256        string
	SceneSHA256         string
	PresentationSHA256  string
	SceneVersion        string
	PresentationVersion string
	CompositorProfile   string
	Plate               *shotsv1.ResolvedSourceMedia
	ActiveMask          *shotsv1.ResolvedSourceMedia
	HardwareMask        *shotsv1.ResolvedSourceMedia
	Screen              *shotsv1.ResolvedSourceMedia
	RenderEligible      bool
	Issues              []*shotsv1.ShotIssue
	Physical            *PhysicalResolution
}

type ReadabilityOutput struct {
	Width, Height     int
	SHA256, MediaType string
}

type ReadabilityReview struct {
	ID, ProjectID, ShotID, RenderID, OutputSHA256, ProofSHA256, Note string
	ShotRevision                                                     int64
	Verdict                                                          int32
	OutputWidth, OutputHeight, RegionWidth, RegionHeight             uint32
	ReviewedAt                                                       time.Time
}

type Page struct {
	Items []Shot
	Next  string
}

type RevisionPage struct {
	Items []Revision
	Next  string
}

// ResolvedShot is the renderer-facing binding for one immutable revision.
// It contains metadata and digests only; workers receive verified bytes after
// the renderer adapter opens them through the project media owner.
type ResolvedShot struct {
	ProjectID string
	Shot      Shot
	Revision  Revision
}

func Schema() string { return schema }
