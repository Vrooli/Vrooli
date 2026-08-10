package deliveryramp

import (
	"context"
	"time"
)

type TransportKind string

const (
	TransportLocal  TransportKind = "local"
	TransportBridge TransportKind = "bridge"
)

type Transport struct {
	Kind      TransportKind `json:"kind"`
	ID        string        `json:"id"`
	Trust     string        `json:"trust,omitempty"`
	Endpoint  string        `json:"-"` // credentials and endpoints never enter evidence JSON
	Available bool          `json:"available"`
	Reason    string        `json:"reason,omitempty"`
}

type Cell struct {
	ID         string `json:"id"`
	Target     Target `json:"target"`
	ProfileID  string `json:"profile_id"`
	Capability string `json:"capability"`
	Required   bool   `json:"required"`
}

type Artifact struct {
	ImmutableRef string            `json:"immutable_ref"`
	LocalPath    string            `json:"local_path,omitempty"`
	Kind         string            `json:"kind"`
	Checksum     string            `json:"checksum"`
	SizeBytes    int64             `json:"size_bytes"`
	Width        int               `json:"width,omitempty"`
	Height       int               `json:"height,omitempty"`
	DurationMs   int64             `json:"duration_ms,omitempty"`
	Container    string            `json:"container,omitempty"`
	Codec        string            `json:"codec,omitempty"`
	UsefulFrames bool              `json:"useful_frames"`
	CreatedAt    time.Time         `json:"created_at"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type BuildRequest struct {
	Cell       Cell              `json:"cell"`
	SourceRef  string            `json:"source_ref"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

type JourneyEvidenceSink interface {
	Capture(context.Context, CaptureRequest) (EvidenceReference, error)
}

type CaptureRequest struct {
	RunID    string `json:"run_id"`
	Scenario string `json:"scenario"`
	TargetID string `json:"target_id"`
	Label    string `json:"label"`
	Display  string `json:"display,omitempty"`
}

type DriverRequest struct {
	Cell     Cell                `json:"cell"`
	Artifact Artifact            `json:"artifact"`
	Plan     JourneyPlan         `json:"plan"`
	Evidence JourneyEvidenceSink `json:"-"`
	RunID    string              `json:"run_id"`
}

type DistributionRequest struct {
	Cell     Cell     `json:"cell"`
	Artifact Artifact `json:"artifact"`
}

type DistributionTarget struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
}

type DistributionResult struct {
	Disposition Disposition          `json:"disposition"`
	Targets     []DistributionTarget `json:"targets,omitempty"`
	References  []EvidenceReference  `json:"references,omitempty"`
	Reason      string               `json:"reason,omitempty"`
}

type Builder interface {
	Build(context.Context, BuildRequest) (Artifact, error)
}

type Driver interface {
	Execute(context.Context, DriverRequest) (JourneyResult, error)
}

type Distributor interface {
	Distribute(context.Context, DistributionRequest) (DistributionResult, error)
}
