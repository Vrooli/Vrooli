// Package distribution owns Android channel availability and distribution.
package distribution

import (
	"context"
	"strings"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

// Channel describes one Android destination independently from its siblings.
type Channel struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	ArtifactKey string `json:"artifact_key"`
	Available   bool   `json:"available"`
	Reason      string `json:"reason"`
	NextAction  string `json:"next_action"`
}

// Distributor reports only channels whose prerequisites were actually probed.
type Distributor struct {
	HasDeveloperVerification bool
	HasSigningReference      bool
	ADB                      func(context.Context, deliveryramp.DistributionRequest) (deliveryramp.DistributionTarget, error)
}

var _ deliveryramp.Distributor = Distributor{}

func (d Distributor) Distribute(ctx context.Context, request deliveryramp.DistributionRequest) (deliveryramp.DistributionResult, error) {
	if strings.TrimSpace(request.Artifact.ImmutableRef) == "" {
		return deliveryramp.DistributionResult{Disposition: deliveryramp.DispositionUnavailable, Reason: "artifact has no immutable identity"}, nil
	}
	targets := []deliveryramp.DistributionTarget{
		{ID: "play", Kind: "play", Available: d.HasDeveloperVerification && d.HasSigningReference, Reason: channelReason(d.HasDeveloperVerification && d.HasSigningReference, "Play requires developer verification and a secrets-manager signing reference")},
		{ID: "verified-sideload", Kind: "verified-sideload", Available: d.HasDeveloperVerification, Reason: channelReason(d.HasDeveloperVerification, "verified sideload requires developer verification")},
	}
	if d.ADB == nil {
		targets = append(targets, deliveryramp.DistributionTarget{ID: "adb-internal", Kind: "adb-internal", Available: false, Reason: "ADB distribution client is not configured"})
	} else {
		target, err := d.ADB(ctx, request)
		if err != nil {
			return deliveryramp.DistributionResult{}, err
		}
		targets = append(targets, target)
	}
	for _, target := range targets {
		if target.Available {
			return deliveryramp.DistributionResult{Disposition: deliveryramp.DispositionPass, Targets: targets}, nil
		}
	}
	return deliveryramp.DistributionResult{Disposition: deliveryramp.DispositionUnavailable, Targets: targets, Reason: "no Android distribution channel is currently available"}, nil
}

func channelReason(available bool, reason string) string {
	if available {
		return "channel prerequisites are present"
	}
	return reason
}
