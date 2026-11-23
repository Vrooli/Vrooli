package models

import (
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"
	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"

	internalbackends "image-tools/internal/backends"
	"image-tools/internal/safety"
)

type cloudTestProvider struct{ testProvider }

func (cloudTestProvider) IsCloud() bool { return true }

// The explain surface answers "does this request need a consent affirmation"
// with the same rule the submit edge enforces, so the UI shows its checkbox
// exactly when a submit would ask.
func TestExplainResolutionReportsConsentByRoute(t *testing.T) {
	h, _ := newTestHandler(t, busyGPUHost)
	policy := safety.PolicyFor(safety.TierLocal)
	h.deps.Consent = func(op, weight, tier string) (bool, string, string) {
		need := policy.ConsentFor(safety.Weight(weight), op, tier)
		return need.Required, need.Basis, need.Reason
	}
	h.deps.Backends = internalbackends.New()
	ops := []string{"edit_instruct", "image_to_image", "text_to_image"}
	for _, p := range []internalbackends.Provider{
		testProvider{name: "diffusers", ops: ops, available: true},
		cloudTestProvider{testProvider{name: "openrouter", ops: ops, available: true}},
	} {
		if err := h.deps.Backends.Register(p); err != nil {
			t.Fatal(err)
		}
	}
	explain := func(op, model string) *modelsv1.Resolution {
		t.Helper()
		out, err := h.ExplainResolution(context.Background(), connect.NewRequest(&modelsv1.ExplainResolutionRequest{Operation: op, ModelId: model, AllowByok: true}))
		if err != nil {
			t.Fatalf("ExplainResolution(%s, %s): %v", op, model, err)
		}
		return out.Msg.GetResolution()
	}

	cloud := explain("edit_instruct", "openrouter-image")
	if !cloud.GetConsentRequired() || cloud.GetConsentBasis() != safety.ConsentBasisRoute || !strings.Contains(cloud.GetConsentReason(), "outside this machine") {
		t.Fatalf("a cloud identity edit must ask for consent on the route rule: %+v", cloud)
	}
	if local := explain("image_to_image", "sd-1.5"); local.GetConsentRequired() {
		t.Fatalf("a local run needs no consent: %+v", local)
	}
	if generated := explain("text_to_image", "openrouter-image"); generated.GetConsentRequired() {
		t.Fatalf("a no-weight operation never asks, even in the cloud: %+v", generated)
	}
}
