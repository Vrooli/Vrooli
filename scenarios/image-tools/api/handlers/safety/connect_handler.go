package safety

import (
	"context"
	"log"
	"sort"

	internalquarantine "image-tools/internal/quarantine"
	internalsafety "image-tools/internal/safety"

	"connectrpc.com/connect"

	safetyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/safety"
)

// connectHandler implements SafetyServiceHandler — the policy discovery surface
// plus the reviewer-feedback corpus.
type connectHandler struct {
	policy     internalsafety.Policy
	health     func() internalsafety.ReviewHealth
	feedback   *internalsafety.FeedbackStore
	quarantine *internalquarantine.Store
	jobs       JobSource
	logger     *log.Logger
	// requireAdmin gates quarantine review on a public/cloud deployment. Local
	// deployments leave it false: the local operator is already the admin.
	requireAdmin bool
	adminToken   string
}

// NewConnectHandler builds the SafetyService discovery handler from the RESOLVED
// policy (preset + per-control overrides). An optional health provider lets the
// summary report the reviewer's live state without changing the wire schema.
func NewConnectHandler(policy internalsafety.Policy, health ...func() internalsafety.ReviewHealth) *connectHandler {
	h := &connectHandler{policy: policy}
	if len(health) > 0 {
		h.health = health[0]
	}
	return h
}

// NewConnectHandlerWithDeps builds the full safety handler, including the
// feedback corpus. It is what production wires; the policy-only constructor
// remains for callers (and tests) that only read the resolved policy.
func NewConnectHandlerWithDeps(d Deps) *connectHandler {
	h := &connectHandler{
		policy:       d.Policy,
		health:       d.Health,
		feedback:     d.Feedback,
		quarantine:   d.Quarantine,
		jobs:         d.Jobs,
		logger:       d.Logger,
		requireAdmin: d.RequireAdmin,
		adminToken:   d.AdminToken,
	}
	if h.logger == nil {
		h.logger = log.Default()
	}
	return h
}

func (h *connectHandler) GetPolicy(_ context.Context, _ *connect.Request[safetyv1.GetPolicyRequest]) (*connect.Response[safetyv1.SafetyPolicy], error) {
	return connect.NewResponse(policyToProto(h.policy, h.summary())), nil
}

func (h *connectHandler) summary() string {
	if h.health == nil {
		return h.policy.Summary()
	}
	out := h.policy.Summary()
	if health := h.health(); health.Configured {
		out += " " + health.Summary() + "."
	}
	return out
}

// policyToProto maps the internal policy + the op-weight table to the proto.
func policyToProto(p internalsafety.Policy, summary string) *safetyv1.SafetyPolicy {
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
		Summary:           summary,
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
