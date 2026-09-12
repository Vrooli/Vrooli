// Package distribution owns independent iOS channel availability.
package distribution

import (
	"context"
	"fmt"
	"strings"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

// Distributor models App Store, TestFlight, and ad-hoc distribution without
// claiming that one channel's credentials satisfy another's requirements.
type Distributor struct {
	DeveloperProgram bool
	SigningReference bool
	TestFlightAccess bool
	AppStoreListing  bool
}

var _ deliveryramp.Distributor = Distributor{}

func (d Distributor) Distribute(_ context.Context, request deliveryramp.DistributionRequest) (result deliveryramp.DistributionResult, err error) {
	defer func() {
		if err != nil {
			return
		}
		if validationErr := request.ValidateResult(result); validationErr != nil {
			err = fmt.Errorf("validate iOS distribution result: %w", validationErr)
			result = deliveryramp.DistributionResult{}
		}
	}()
	if strings.TrimSpace(request.Artifact.ImmutableRef) == "" {
		return deliveryramp.DistributionResult{Disposition: deliveryramp.DispositionUnavailable, Reason: "artifact has no immutable identity"}, nil
	}
	targets := []deliveryramp.DistributionTarget{
		{ID: "testflight", Kind: "testflight", Available: d.DeveloperProgram && d.SigningReference && d.TestFlightAccess, Reason: channelReason(d.DeveloperProgram && d.SigningReference && d.TestFlightAccess, "TestFlight requires an enrolled developer, signing reference, and TestFlight access")},
		{ID: "app-store", Kind: "app-store", Available: d.DeveloperProgram && d.SigningReference && d.AppStoreListing, Reason: channelReason(d.DeveloperProgram && d.SigningReference && d.AppStoreListing, "App Store requires enrollment, signing, and a completed listing")},
		{ID: "ad-hoc", Kind: "ad-hoc", Available: d.DeveloperProgram && d.SigningReference, Reason: channelReason(d.DeveloperProgram && d.SigningReference, "ad-hoc distribution requires enrollment and signing")},
	}
	for _, target := range targets {
		if target.Available {
			return deliveryramp.DistributionResult{
				Disposition: deliveryramp.DispositionDegraded, Targets: targets, CapabilityReady: true,
				Reason: "distribution prerequisites are ready; no App Store, TestFlight, or ad-hoc publication effect was performed",
			}, nil
		}
	}
	return deliveryramp.DistributionResult{Disposition: deliveryramp.DispositionUnavailable, Targets: targets, Reason: "no iOS distribution channel is currently available"}, nil
}

func channelReason(available bool, reason string) string {
	if available {
		return "channel prerequisites are present"
	}
	return reason
}
