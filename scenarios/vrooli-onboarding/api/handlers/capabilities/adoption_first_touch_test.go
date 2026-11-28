package capabilities

import (
	"encoding/json"
	"testing"

	capabilitiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/capabilities"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestAdoptionFirstTouchUsesLocalControlPlaneAndSeparateAdoptionIdentity(t *testing.T) {
	inputs, err := structpb.NewStruct(map[string]any{"instance_id": "adopted-vps:fixture-1", "ssh_password": "transient"})
	if err != nil {
		t.Fatal(err)
	}
	request, err := actionRequest("local", &capabilitiesv1.ActionRequest{CapabilityId: "compute-manager-adoption-first-touch", IdempotencyKey: "independent-request-key-123", Inputs: inputs})
	if err != nil {
		t.Fatal(err)
	}
	if request.TargetID != "local" {
		t.Fatalf("target = %q, want local so Onboarding does not proxy to Bridge", request.TargetID)
	}
	var instanceID, requestKey string
	if err := json.Unmarshal(request.Inputs["instance_id"], &instanceID); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(request.Inputs["idempotency_key"], &requestKey); err != nil {
		t.Fatal(err)
	}
	if instanceID != "adopted-vps:fixture-1" || requestKey != "independent-request-key-123" {
		t.Fatalf("first-touch inputs instance=%q request_key=%q", instanceID, requestKey)
	}
}

func TestAdoptionFirstTouchRejectsNonlocalExecutionAndInvalidAdoptionID(t *testing.T) {
	for _, test := range []struct {
		name, target, instance string
	}{
		{name: "remote target", target: "bridge-machine-1", instance: "adopted-vps:fixture-1"},
		{name: "not adopted", target: "local", instance: "local"},
	} {
		t.Run(test.name, func(t *testing.T) {
			inputs, err := structpb.NewStruct(map[string]any{"instance_id": test.instance, "ssh_password": "transient"})
			if err != nil {
				t.Fatal(err)
			}
			_, err = actionRequest(test.target, &capabilitiesv1.ActionRequest{CapabilityId: "compute-manager-adoption-first-touch", IdempotencyKey: "independent-request-key-123", Inputs: inputs})
			if err == nil {
				t.Fatal("invalid first-touch route/instance was accepted")
			}
		})
	}
}
