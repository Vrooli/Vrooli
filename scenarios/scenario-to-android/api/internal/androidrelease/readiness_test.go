package androidrelease

import (
	"context"
	"testing"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

func TestGoogleReadinessHasSixIndependentRungs(t *testing.T) {
	readiness := GoogleReadiness(false, false, false, true, false, false)
	if err := readiness.Validate(); err != nil {
		t.Fatal(err)
	}
	if readiness.Rungs[3].State != RungReady || readiness.Rungs[0].State != RungUnavailable {
		t.Fatalf("unexpected readiness: %#v", readiness)
	}
	if readiness.Rungs[1].Obligation == "" {
		t.Fatal("developer-verification rung omitted its dated obligation")
	}
	if readiness.Rungs[3].Obligation == "" {
		t.Fatal("target-api rung omitted its dated obligation")
	}
}

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
