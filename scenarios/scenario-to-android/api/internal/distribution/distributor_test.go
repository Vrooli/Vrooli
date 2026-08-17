package distribution

import (
	"context"
	"testing"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

func TestDistributorDoesNotInferSiblingChannelAvailability(t *testing.T) {
	result, err := (Distributor{HasDeveloperVerification: true, HasSigningReference: false}).Distribute(context.Background(), deliveryramp.DistributionRequest{Artifact: deliveryramp.Artifact{ImmutableRef: "android-debug:abc"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Disposition != deliveryramp.DispositionPass {
		t.Fatal("verified sideload should remain independently available")
	}
	if result.Targets[0].Available || !result.Targets[1].Available {
		t.Fatalf("channel availability was inferred: %#v", result.Targets)
	}
}
