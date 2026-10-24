package main

import (
	"context"
	"strings"

	internalartifacts "vrooli-bridge/internal/artifacts"
	internalcompanion "vrooli-bridge/internal/companion"
)

// companionArtifactDistributor adapts the existing durable artifacts domain
// without allowing companion lifecycle code to move or store bytes itself.
type companionArtifactDistributor struct {
	service internalartifacts.Service
}

func (d companionArtifactDistributor) Distribute(ctx context.Context, in internalcompanion.ArtifactRequest) (internalcompanion.ArtifactDecision, error) {
	decision, err := d.service.Distribute(ctx, internalartifacts.DistributeInput{
		Actor:           "owner",
		NodeID:          in.NodeID,
		Name:            in.Name,
		SourceRef:       in.SourceRef,
		DestinationPath: in.DestinationPath,
	})
	if err != nil {
		return internalcompanion.ArtifactDecision{}, err
	}
	out := internalcompanion.ArtifactDecision{DistributionID: decision.DistributionID, Reason: decision.DeliveryRef}
	switch decision.Status {
	case internalartifacts.StatusPending:
		out.Pending = true
	case internalartifacts.StatusFailed:
		out.Failed = true
		out.Reason = "artifact delivery failed"
	case internalartifacts.StatusDelivered:
		// A synchronous delivery receipt is already terminal; lifecycle may start.
	default:
		out.Failed = true
		out.Reason = "artifact delivery returned an unknown status"
	}
	if strings.TrimSpace(out.DistributionID) == "" && out.Pending {
		out.Failed = true
		out.Pending = false
		out.Reason = "artifact delivery did not return a distribution id"
	}
	return out, nil
}
