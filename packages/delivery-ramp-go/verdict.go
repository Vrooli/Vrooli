package deliveryramp

import (
	"fmt"
	"strings"
	"time"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TargetVerdictInput struct {
	Producer    string
	Target      Target
	Disposition Disposition
	RunID       string
	Detail      string
	References  []EvidenceReference
	CreatedAt   time.Time
}

// NewTargetVerdict emits the deployment-manager reference-only contract. It
// never embeds capture bytes, local paths, credentials, or endpoint details.
func NewTargetVerdict(input TargetVerdictInput) (*commonv1.TargetVerdict, error) {
	if err := input.Target.Validate(); err != nil {
		return nil, err
	}
	if !input.Disposition.Valid() {
		return nil, fmt.Errorf("invalid target disposition %q", input.Disposition)
	}
	if strings.TrimSpace(input.Producer) == "" || strings.TrimSpace(input.RunID) == "" {
		return nil, fmt.Errorf("target verdict producer and run id are required")
	}
	if input.Disposition == DispositionPass && len(input.References) == 0 {
		return nil, fmt.Errorf("passing target verdict requires evidence references")
	}
	refs := make([]*commonv1.EvidenceRef, 0, len(input.References))
	for _, reference := range input.References {
		if strings.TrimSpace(reference.ID) == "" || strings.TrimSpace(reference.Checksum) == "" || !reference.Redacted {
			return nil, fmt.Errorf("evidence reference %q is incomplete or not redacted", reference.ID)
		}
		refs = append(refs, &commonv1.EvidenceRef{Producer: input.Producer, ArtifactId: reference.ID, Kind: reference.Kind, Checksum: reference.Checksum, CreatedAt: timestamp(input.CreatedAt)})
	}
	verdict := &commonv1.TargetVerdict{
		Target:      &commonv1.EvidenceTarget{Ramp: input.Target.Ramp, Platform: input.Target.Platform, Os: input.Target.OS, DeviceKind: deviceKind(input.Target.DeviceKind), BridgeNodeId: optional(input.Target.NodeID)},
		Disposition: protoDisposition(input.Disposition), Refs: refs, RunId: input.RunID, Detail: input.Detail,
	}
	if input.Target.Transport.Kind == TransportBridge {
		verdict.Target.BridgeJobId = optional(input.RunID)
	}
	return verdict, nil
}

func timestamp(value time.Time) *timestamppb.Timestamp {
	if value.IsZero() {
		return nil
	}
	return timestamppb.New(value.UTC())
}

func optional(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func deviceKind(value string) commonv1.DeviceKind {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "emulator":
		return commonv1.DeviceKind_DEVICE_KIND_EMULATOR
	case "physical":
		return commonv1.DeviceKind_DEVICE_KIND_PHYSICAL
	default:
		return commonv1.DeviceKind_DEVICE_KIND_HOST
	}
}

func protoDisposition(value Disposition) commonv1.Disposition {
	switch value {
	case DispositionPass:
		return commonv1.Disposition_DISPOSITION_PASSED
	case DispositionNotRun:
		return commonv1.Disposition_DISPOSITION_SKIPPED
	default:
		return commonv1.Disposition_DISPOSITION_FAILED
	}
}
