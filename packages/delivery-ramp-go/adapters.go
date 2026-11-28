package deliveryramp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/targetmodel"
)

type TransportKind = targetmodel.TransportKind

const (
	TransportLocal  = targetmodel.TransportLocal
	TransportBridge = targetmodel.TransportBridge
)

type Transport = targetmodel.Transport

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
	Cell      Cell   `json:"cell"`
	SourceRef string `json:"source_ref"`
	// Format describes the delivery representation. It is intentionally not
	// encoded in Cell.Target, which remains an execution target identity.
	Format     DeliveryFormat    `json:"format,omitempty"`
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
	Cell     Cell           `json:"cell"`
	Artifact Artifact       `json:"artifact"`
	Format   DeliveryFormat `json:"format,omitempty"`
}

func (r DistributionRequest) ValidateResult(result DistributionResult) error {
	if err := result.Validate(); err != nil {
		return err
	}
	if result.EffectReceipt != nil && strings.TrimSpace(r.Artifact.ImmutableRef) != result.EffectReceipt.ArtifactRef {
		return fmt.Errorf("distribution effect receipt artifact %q does not match requested artifact %q", result.EffectReceipt.ArtifactRef, r.Artifact.ImmutableRef)
	}
	return nil
}

type DistributionTarget struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
}

// DistributionEffectReceipt is producer-attributed evidence that a
// distribution owner performed an external effect. Capability readiness and
// target availability do not populate this value.
type DistributionEffectReceipt struct {
	TargetID        string    `json:"target_id"`
	ArtifactRef     string    `json:"artifact_ref"`
	ExternalReceipt string    `json:"external_receipt"`
	Outcome         string    `json:"outcome"`
	ObservedAt      time.Time `json:"observed_at"`
}

type DistributionResult struct {
	Disposition     Disposition                `json:"disposition"`
	Targets         []DistributionTarget       `json:"targets,omitempty"`
	References      []EvidenceReference        `json:"references,omitempty"`
	CapabilityReady bool                       `json:"capability_ready,omitempty"`
	EffectReceipt   *DistributionEffectReceipt `json:"effect_receipt,omitempty"`
	Reason          string                     `json:"reason,omitempty"`
}

// Validate enforces the boundary between a ready capability and an external
// distribution effect. A distributor may report degraded readiness without a
// receipt, but a passing or effectful result must identify the exact target,
// artifact, and owner receipt that observed the effect.
func (r DistributionResult) Validate() error {
	if !r.Disposition.Valid() {
		return fmt.Errorf("invalid distribution disposition %q", r.Disposition)
	}
	if r.Disposition != DispositionPass && strings.TrimSpace(r.Reason) == "" {
		return fmt.Errorf("distribution disposition %q requires a reason", r.Disposition)
	}
	if r.Disposition == DispositionPass && r.EffectReceipt == nil {
		return fmt.Errorf("passing distribution requires an effect receipt")
	}
	for _, target := range r.Targets {
		if strings.TrimSpace(target.ID) == "" || strings.TrimSpace(target.Kind) == "" {
			return fmt.Errorf("distribution target id and kind are required")
		}
		if !target.Available && strings.TrimSpace(target.Reason) == "" {
			return fmt.Errorf("unavailable distribution target %q requires a reason", target.ID)
		}
		if r.Disposition == DispositionPass && !target.Available {
			return fmt.Errorf("passing distribution cannot include unavailable target %q", target.ID)
		}
	}
	if r.EffectReceipt == nil {
		return nil
	}
	receipt := r.EffectReceipt
	if r.Disposition != DispositionPass && r.Disposition != DispositionDegraded {
		return fmt.Errorf("distribution disposition %q cannot carry an effect receipt", r.Disposition)
	}
	if strings.TrimSpace(receipt.TargetID) == "" || strings.TrimSpace(receipt.ArtifactRef) == "" || strings.TrimSpace(receipt.ExternalReceipt) == "" || strings.TrimSpace(receipt.Outcome) == "" {
		return fmt.Errorf("distribution effect receipt identity and outcome are required")
	}
	if receipt.ObservedAt.IsZero() {
		return fmt.Errorf("distribution effect receipt observed_at is required")
	}
	if len(r.Targets) > 0 {
		for _, target := range r.Targets {
			if target.ID == receipt.TargetID {
				return nil
			}
		}
		return fmt.Errorf("distribution effect receipt target %q is not declared", receipt.TargetID)
	}
	return nil
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
