package distribution

import (
	"context"
	"testing"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

func TestChannelsRemainIndependent(t *testing.T) {
	result, err := (Distributor{DeveloperProgram: true, SigningReference: true, TestFlightAccess: true}).Distribute(context.Background(), deliveryramp.DistributionRequest{Artifact: deliveryramp.Artifact{ImmutableRef: "ios:debug"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Disposition != deliveryramp.DispositionPass {
		t.Fatalf("result = %+v", result)
	}
	if result.Targets[1].Available {
		t.Fatal("App Store became available from TestFlight access")
	}
}

func TestNoCredentialsIsUnavailable(t *testing.T) {
	result, err := (Distributor{}).Distribute(context.Background(), deliveryramp.DistributionRequest{Artifact: deliveryramp.Artifact{ImmutableRef: "ios:debug"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Disposition != deliveryramp.DispositionUnavailable {
		t.Fatalf("result = %+v", result)
	}
}
