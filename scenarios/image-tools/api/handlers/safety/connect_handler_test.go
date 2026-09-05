package safety

import (
	"testing"

	internalsafety "image-tools/internal/safety"

	"connectrpc.com/connect"

	safetyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/safety"
)

func TestGetPolicy_Local(t *testing.T) {
	h := NewConnectHandler(internalsafety.TierLocal)
	resp, err := h.GetPolicy(t.Context(), connect.NewRequest(&safetyv1.GetPolicyRequest{}))
	if err != nil {
		t.Fatalf("GetPolicy: %v", err)
	}
	p := resp.Msg
	if p.GetTier() != safetyv1.DeploymentTier_DEPLOYMENT_TIER_LOCAL {
		t.Errorf("tier = %v, want LOCAL", p.GetTier())
	}
	if p.GetRequireConsent() || p.GetForceNsfwScan() || p.GetRequireProvenance() {
		t.Errorf("local policy should enforce nothing, got %+v", p)
	}
	if p.GetSummary() == "" {
		t.Error("summary should be set")
	}
	if len(p.GetOpWeights()) == 0 {
		t.Error("op weights table should be populated")
	}
}

func TestGetPolicy_Public(t *testing.T) {
	h := NewConnectHandler(internalsafety.TierPublic)
	resp, err := h.GetPolicy(t.Context(), connect.NewRequest(&safetyv1.GetPolicyRequest{}))
	if err != nil {
		t.Fatalf("GetPolicy: %v", err)
	}
	p := resp.Msg
	if p.GetTier() != safetyv1.DeploymentTier_DEPLOYMENT_TIER_PUBLIC {
		t.Errorf("tier = %v, want PUBLIC", p.GetTier())
	}
	if !p.GetRequireConsent() || !p.GetForceNsfwScan() || !p.GetRequireProvenance() {
		t.Errorf("public policy should enforce all gates, got %+v", p)
	}
	if p.GetRateLimitPerMin() <= 0 {
		t.Errorf("public rate limit = %d, want > 0", p.GetRateLimitPerMin())
	}
	// edit_instruct must be reported HIGH weight; text_to_image NONE.
	weights := map[string]safetyv1.ConsentWeight{}
	for _, ow := range p.GetOpWeights() {
		weights[ow.GetOperation()] = ow.GetWeight()
	}
	if weights["edit_instruct"] != safetyv1.ConsentWeight_CONSENT_WEIGHT_HIGH {
		t.Errorf("edit_instruct weight = %v, want HIGH", weights["edit_instruct"])
	}
	if weights["naturalize"] != safetyv1.ConsentWeight_CONSENT_WEIGHT_LOW {
		t.Errorf("naturalize weight = %v, want LOW", weights["naturalize"])
	}
}
