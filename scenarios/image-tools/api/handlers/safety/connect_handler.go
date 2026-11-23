package safety

import (
	"context"
	"sort"

	internalsafety "image-tools/internal/safety"

	"connectrpc.com/connect"

	safetyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/safety"
)

// connectHandler implements SafetyServiceHandler — the policy discovery surface.
type connectHandler struct {
	policy internalsafety.Policy
}

// NewConnectHandler builds the SafetyService discovery handler from the RESOLVED
// policy (preset + per-control overrides). Taking the resolved policy, rather
// than a tier, guarantees discovery reports the same controls the submit edge
// enforces — a tier alone would report the preset and hide operator overrides.
func NewConnectHandler(policy internalsafety.Policy) *connectHandler {
	return &connectHandler{policy: policy}
}

func (h *connectHandler) GetPolicy(_ context.Context, _ *connect.Request[safetyv1.GetPolicyRequest]) (*connect.Response[safetyv1.SafetyPolicy], error) {
	return connect.NewResponse(policyToProto(h.policy)), nil
}

// policyToProto maps the internal policy + the op-weight table to the proto.
func policyToProto(p internalsafety.Policy) *safetyv1.SafetyPolicy {
	weights := internalsafety.OpWeights()
	ops := make([]string, 0, len(weights))
	for op := range weights {
		ops = append(ops, op)
	}
	sort.Strings(ops)

	out := &safetyv1.SafetyPolicy{
		Tier:              tierToProto(p.Tier),
		RequireConsent:    p.RequireConsent,
		ConsentNonLocal:   p.ConsentNonLocal,
		ForceNsfwScan:     p.ForceNSFWScan,
		RequireProvenance: p.RequireProvenance,
		RateLimitPerMin:   int32(p.RateLimitPerMin),
		Summary:           p.Summary(),
	}
	for _, op := range ops {
		out.OpWeights = append(out.OpWeights, &safetyv1.OpWeight{
			Operation: op,
			Weight:    weightToProto(weights[op]),
		})
	}
	return out
}

func tierToProto(t internalsafety.Tier) safetyv1.DeploymentTier {
	if t == internalsafety.TierPublic {
		return safetyv1.DeploymentTier_DEPLOYMENT_TIER_PUBLIC
	}
	return safetyv1.DeploymentTier_DEPLOYMENT_TIER_LOCAL
}

func weightToProto(w internalsafety.Weight) safetyv1.ConsentWeight {
	switch w {
	case internalsafety.WeightHigh:
		return safetyv1.ConsentWeight_CONSENT_WEIGHT_HIGH
	case internalsafety.WeightLow:
		return safetyv1.ConsentWeight_CONSENT_WEIGHT_LOW
	default:
		return safetyv1.ConsentWeight_CONSENT_WEIGHT_NONE
	}
}
