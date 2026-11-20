package routing

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	"ai-gateway/internal/providers"
	"github.com/vrooli/ai-provider-go/policy"
)

// probeCapacity returns a capacity verdict for a local candidate without holding
// a reservation (claim then immediate release). Used by planning/preview. A nil
// adapter or an absent footprint yields a non-blocking verdict. A declared
// footprint requires an actual fit before the route can be selected.
func (s *Service) probeCapacity(ctx context.Context, req *sharedv1.GatewayRequest) CapacityEvaluation {
	bytes := requestCapacityRequirementBytes(req)
	if bytes <= 0 {
		return CapacityEvaluation{Verdict: CapacityUnknown}
	}
	if s.capacity == nil {
		return CapacityEvaluation{Verdict: CapacityUnknown, RequiredBytes: bytes}
	}
	eval, err := s.capacity.Claim(ctx, CapacityRequest{OwnerID: capacityOwner(req), RequiredBytes: bytes, AllowReclaim: allowReclaim(req)})
	if eval.ClaimID != "" {
		s.capacity.Release(ctx, eval.ClaimID)
		eval.ClaimID = ""
	}
	if err != nil {
		return CapacityEvaluation{Verdict: CapacityUnknown, RequiredBytes: bytes}
	}
	return eval
}

// acquireCapacity holds an op-scoped claim around a local execution attempt. The
// returned evaluation carries the live ClaimID the caller must release. Non-local
// candidates and absent footprints return a zero evaluation; a missing adapter
// for a declared footprint returns unknown and is rejected before execution.
func (s *Service) acquireCapacity(ctx context.Context, req *sharedv1.GatewayRequest, candidate *routingv1.RouteCandidate) CapacityEvaluation {
	if !strings.EqualFold(candidate.GetLocality(), "local") {
		return CapacityEvaluation{}
	}
	bytes := requestCapacityRequirementBytes(req)
	if bytes <= 0 {
		return CapacityEvaluation{Verdict: CapacityUnknown}
	}
	if s.capacity == nil {
		return CapacityEvaluation{Verdict: CapacityUnknown, RequiredBytes: bytes}
	}
	eval, err := s.capacity.Claim(ctx, CapacityRequest{OwnerID: capacityOwner(req), RequiredBytes: bytes, AllowReclaim: allowReclaim(req)})
	if err != nil {
		if eval.ClaimID != "" {
			s.releaseCapacity(ctx, eval)
		}
		return CapacityEvaluation{Verdict: CapacityUnknown, RequiredBytes: bytes}
	}
	return eval
}

func (s *Service) releaseCapacity(ctx context.Context, eval CapacityEvaluation) {
	if s.capacity != nil && eval.ClaimID != "" {
		s.capacity.Release(ctx, eval.ClaimID)
	}
}

func capacityReason(eval CapacityEvaluation) string {
	switch eval.Verdict {
	case CapacityInsufficient:
		return "local capacity broker reports insufficient capacity for this route"
	case CapacityAdvisoryReclaimUnavailable:
		return "local route would need capacity reclaim but broker enforcement is advisory; treating local as unavailable"
	default:
		return "local capacity broker verdict: " + string(eval.Verdict)
	}
}

type routePlan struct {
	id               string
	candidates       []*routingv1.RouteCandidate
	selectedProvider string
	selectedLocality string
	policyReasons    []string
	fallbackAllowed  bool
	rights           map[string]plannedLocalRight
}

type plannedLocalRight struct {
	model               string
	runtimeIdentity     *policy.RuntimeArtifactIdentity
	runtimeDigest       string
	decision            policy.RightsDecision
	policySchemaVersion string
	policyFileSHA256    string
	receiptSHA256       string
}

func candidateKey(provider, role string) string { return provider + "\x00" + role }

// requiresLocalModelRights covers known local model owners even while their
// adapters lack artifact facts. Other local providers enter this path when
// their policy exposes a concrete model identity or Rights facts.
func requiresLocalModelRights(provider string, role providers.Role) bool {
	if !strings.EqualFold(role.Locality, "local") {
		return false
	}
	if provider == providers.ProviderOllama || provider == providers.ProviderOllaya || provider == providers.ProviderPiiDetector {
		return true
	}
	return strings.TrimSpace(role.Model) != "" || role.Rights != nil
}

func hasVerifiedRuntimeArtifactIdentity(provider, model string, rights *policy.ArtifactRights, binding *policy.RuntimeArtifactIdentity, observedDigest string) bool {
	return RuntimeArtifactIdentityMatches(binding, rights, provider, model, observedDigest)
}

// classifyCandidate resolves one provider into a route candidate and reports
// whether it is eligible. Rejected candidates carry a stable rejection_reason.
// The evaluation order is hard policy first (role/capability/locality), then
// provider breaker state, then local capacity — never opaque scoring.
func (s *Service) classifyCandidate(ctx context.Context, req *sharedv1.GatewayRequest, providerName string) (*routingv1.RouteCandidate, bool) {
	adapter := s.adapters[providerName]
	allowConsent := s.remoteConsentAllowed(ctx, req, providerName, adapter.Locality)
	role, rejected := policyCandidate(ctx, req, providerName, adapter, allowConsent)
	if rejected != nil {
		return rejected, false
	}
	candidate := &routingv1.RouteCandidate{
		Provider: providerName,
		Role:     role.Role,
		Locality: role.Locality,
		Reasons:  []string{"eligible by role capability and profile policy"},
	}
	if requiresLocalModelRights(providerName, role) && !s.localModelRightsAdmit(ctx, providerName, role, candidate) {
		return candidate, false
	}
	if !s.applyCandidateBreaker(ctx, req, role, candidate) {
		return candidate, false
	}
	if !s.capacityAdmits(ctx, req, candidate) {
		return candidate, false
	}
	return candidate, true
}

func (s *Service) localModelRightsAdmit(ctx context.Context, provider string, role providers.Role, candidate *routingv1.RouteCandidate) bool {
	if s.rightsContext == nil || role.Model == "" || role.Rights == nil || !hasVerifiedRuntimeArtifactIdentity(provider, role.Model, role.Rights, role.RuntimeIdentity, role.ObservedRuntimeDigest) {
		candidate.RejectionReason = "local_model_rights_unknown"
		candidate.Reasons = append(candidate.Reasons, "local model identity, artifact rights, or trusted organization context is missing")
		return false
	}
	currentContext, err := s.rightsContext(ctx)
	if err != nil {
		candidate.RejectionReason = "local_model_rights_context_unavailable"
		candidate.Reasons = append(candidate.Reasons, "trusted organization context is unavailable")
		return false
	}
	decision := EvaluateLocalModelRights(policy.Model{Rights: role.Rights}, policy.RightsUseLocalInference, currentContext, s.now())
	candidate.Reasons = append(candidate.Reasons, "local model rights: "+string(decision.Status)+" ("+decision.Reason+")")
	if decision.Status != policy.RightsAllowed {
		candidate.RejectionReason = "local_model_rights_" + string(decision.Status)
		return false
	}
	return true
}

func (s *Service) applyCandidateBreaker(ctx context.Context, req *sharedv1.GatewayRequest, role providers.Role, candidate *routingv1.RouteCandidate) bool {
	state := s.effectiveBreaker(ctx, candidate.GetProvider(), role.Role, req.GetKind())
	candidate.BreakerState = string(state)
	if state == BreakerOpen {
		candidate.RejectionReason = "provider_breaker_open"
		candidate.Reasons = []string{"provider circuit breaker is open; skipping until cooldown elapses"}
		return false
	}
	if state == BreakerHalfOpen {
		candidate.HalfOpenProbe = true
		candidate.Reasons = append(candidate.Reasons, "provider breaker is half-open; eligible as a bounded recovery probe")
	}
	return true
}

// policyCandidate applies the hard policy filters (inventory/role/capability/
// locality) and returns either the resolved role (ok) or a rejected candidate.
func policyCandidate(ctx context.Context, req *sharedv1.GatewayRequest, providerName string, adapter providers.Adapter, allowConsent bool) (providers.Role, *routingv1.RouteCandidate) {
	inventory, err := adapter.ListRoles(ctx)
	if err != nil {
		return providers.Role{}, rejectedCandidate(providerName, req.GetRole(), adapter.Locality,
			fmt.Sprintf("inventory unavailable: %v", err), "inventory_unavailable")
	}
	role, ok := findRole(inventory.Roles, req.GetRole())
	if !ok {
		return providers.Role{}, rejectedCandidate(providerName, req.GetRole(), adapter.Locality,
			"role is not exposed by provider policy", "role_not_exposed")
	}
	if !roleSupportsRequest(role, req) {
		return providers.Role{}, rejectedCandidate(providerName, role.Role, role.Locality,
			"role capabilities or input modalities do not satisfy request", "capability_mismatch")
	}
	if reason := localityRejection(req, role.Locality, allowConsent); reason != "" {
		return providers.Role{}, rejectedCandidate(providerName, role.Role, role.Locality, reason, "locality_forbidden")
	}
	return role, nil
}

// capacityAdmits evaluates local capacity for a local candidate, annotates its
// verdict, and reports whether it stays eligible. Remote candidates and
// requests without a declared footprint always admit.
func (s *Service) capacityAdmits(ctx context.Context, req *sharedv1.GatewayRequest, candidate *routingv1.RouteCandidate) bool {
	if !strings.EqualFold(candidate.GetLocality(), "local") {
		return true
	}
	if requestCapacityRequirementBytes(req) <= 0 {
		return true
	}
	eval := s.probeCapacity(ctx, req)
	candidate.CapacityVerdict = string(eval.Verdict)
	if !capacityVerdictAdmitted(eval, allowReclaim(req)) || eval.GrantedBytes < eval.RequiredBytes {
		candidate.RejectionReason = "insufficient_capacity"
		candidate.Reasons = append(candidate.Reasons, capacityReason(eval))
		return false
	}
	return true
}

func capacityVerdictAdmitted(eval CapacityEvaluation, allowReclaim bool) bool {
	return eval.Verdict == CapacityFit || (allowReclaim && eval.Verdict == CapacityReclaimRequired)
}

func capacityExecutionAdmitted(eval CapacityEvaluation, requiredBytes int64, allowReclaim bool) bool {
	return capacityVerdictAdmitted(eval, allowReclaim) && eval.ClaimID != "" && eval.GrantedBytes >= requiredBytes
}

func rejectedCandidate(provider, role, locality, reason, code string) *routingv1.RouteCandidate {
	return &routingv1.RouteCandidate{
		Provider:        provider,
		Role:            role,
		Locality:        locality,
		Reasons:         []string{reason},
		RejectionReason: code,
	}
}

func (s *Service) plan(ctx context.Context, req *sharedv1.GatewayRequest) routePlan {
	plan := routePlan{id: newID("plan"), policyReasons: []string{profileReason(req.GetProfile())}, rights: map[string]plannedLocalRight{}}
	var eligible []*routingv1.RouteCandidate
	for _, providerName := range s.order {
		candidate, ok := s.planProvider(ctx, req, providerName, &plan)
		if ok {
			eligible = append(eligible, candidate)
		} else {
			plan.candidates = append(plan.candidates, candidate)
		}
	}

	sort.SliceStable(eligible, func(i, j int) bool {
		return rank(req.GetProfile(), eligible[i].GetLocality(), eligible[i].GetProvider()) <
			rank(req.GetProfile(), eligible[j].GetLocality(), eligible[j].GetProvider())
	})
	if len(eligible) > 0 {
		eligible[0].Selected = true
		plan.selectedProvider = eligible[0].GetProvider()
		plan.selectedLocality = eligible[0].GetLocality()
		plan.policyReasons = append(plan.policyReasons, fmt.Sprintf("selected %s because it is the highest-ranked eligible %s provider", plan.selectedProvider, plan.selectedLocality))
	}
	for i, candidate := range eligible {
		if i > 0 {
			candidate.FallbackEligible = true
			plan.fallbackAllowed = true
		}
		plan.candidates = append(plan.candidates, candidate)
	}
	if len(eligible) == 0 {
		plan.policyReasons = append(plan.policyReasons, "no provider satisfied role, capability, locality, and privacy constraints")
	}
	sort.SliceStable(plan.candidates, func(i, j int) bool {
		if plan.candidates[i].GetSelected() != plan.candidates[j].GetSelected() {
			return plan.candidates[i].GetSelected()
		}
		if plan.candidates[i].GetFallbackEligible() != plan.candidates[j].GetFallbackEligible() {
			return plan.candidates[i].GetFallbackEligible()
		}
		return plan.candidates[i].GetProvider() < plan.candidates[j].GetProvider()
	})
	return plan
}

func (s *Service) planProvider(ctx context.Context, req *sharedv1.GatewayRequest, provider string, plan *routePlan) (*routingv1.RouteCandidate, bool) {
	candidate, eligible := s.classifyCandidate(ctx, req, provider)
	if !eligible || !strings.EqualFold(candidate.GetLocality(), "local") {
		return candidate, eligible
	}
	allowConsent := s.remoteConsentAllowed(ctx, req, provider, s.adapters[provider].Locality)
	role, rejected := policyCandidate(ctx, req, provider, s.adapters[provider], allowConsent)
	if rejected == nil && !requiresLocalModelRights(provider, role) {
		return candidate, true
	}
	return candidate, s.planLocalRights(ctx, provider, role, rejected, candidate, plan)
}

// planLocalRights captures an auditable rights decision for a local model
// candidate. The receipt is later rechecked immediately before provider
// dispatch, so a stale policy or changed runtime artifact fails closed.
func (s *Service) planLocalRights(ctx context.Context, provider string, role providers.Role, rejected *routingv1.RouteCandidate, candidate *routingv1.RouteCandidate, plan *routePlan) bool {
	currentContext, decision, valid := s.evaluatePlanRights(ctx, provider, role)
	if rejected != nil || !valid {
		candidate.RejectionReason = "local_model_rights_unknown"
		candidate.Reasons = append(candidate.Reasons, "local model rights or trusted context changed while planning")
		return false
	}
	if role.PolicySchemaVersion == "" || role.PolicyFileSHA256 == "" {
		candidate.RejectionReason = "local_model_policy_identity_unknown"
		candidate.Reasons = append(candidate.Reasons, "local model policy revision is missing")
		return false
	}
	evaluatedAt, err := time.Parse(time.RFC3339Nano, decision.EvaluatedAt)
	if err != nil {
		candidate.RejectionReason = "local_model_rights_unknown"
		candidate.Reasons = append(candidate.Reasons, "local model rights evaluation timestamp is invalid")
		return false
	}
	receipt := policy.NewRightsDecisionReceipt(policy.RightsDecisionReceiptExpectation{
		Role: role.Role, Provider: provider, Model: role.Model,
		Use: policy.RightsUseLocalInference, PolicySchemaVersion: role.PolicySchemaVersion,
		ArtifactRights: role.Rights, RuntimeIdentity: role.RuntimeIdentity, Context: currentContext,
	}, evaluatedAt)
	if receipt.Decision.DecisionSHA256 != decision.DecisionSHA256 || policy.RequireAllowedRightsDecisionReceipt(receipt) != nil {
		candidate.RejectionReason = "local_model_rights_unknown"
		candidate.Reasons = append(candidate.Reasons, "local model rights receipt is not allowed")
		return false
	}
	bindingCopy := *role.RuntimeIdentity
	plan.rights[candidateKey(provider, role.Role)] = plannedLocalRight{
		model: role.Model, runtimeIdentity: &bindingCopy, runtimeDigest: role.ObservedRuntimeDigest, decision: decision,
		policySchemaVersion: role.PolicySchemaVersion, policyFileSHA256: role.PolicyFileSHA256, receiptSHA256: receipt.ReceiptSHA256,
	}
	return true
}

func (s *Service) evaluatePlanRights(ctx context.Context, provider string, role providers.Role) (policy.RightsContext, policy.RightsDecision, bool) {
	currentContext, contextErr := policy.RightsContext{}, errors.New("trusted context source unavailable")
	if s.rightsContext != nil {
		currentContext, contextErr = s.rightsContext(ctx)
	}
	decision := EvaluateLocalModelRights(policy.Model{Rights: role.Rights}, policy.RightsUseLocalInference, currentContext, s.now())
	valid := contextErr == nil && role.Model != "" && role.Rights != nil &&
		hasVerifiedRuntimeArtifactIdentity(provider, role.Model, role.Rights, role.RuntimeIdentity, role.ObservedRuntimeDigest) && decision.Status == policy.RightsAllowed
	return currentContext, decision, valid
}

// applyCapacityEvidence records the capacity verdict and any acquired claim on
// route evidence. The execution claim (capEval) is preferred; when no claim was
// acquired (remote route or absent footprint) the planning verdict on the
// selected candidate is used.
func applyCapacityEvidence(ev *routingv1.RouteEvidence, candidate *routingv1.RouteCandidate, capEval CapacityEvaluation) {
	verdict := string(capEval.Verdict)
	if verdict == "" {
		verdict = candidate.GetCapacityVerdict()
	}
	ev.CapacityVerdict = verdict
	ev.CapacityClaimId = capEval.ClaimID
	ev.CapacityRequiredBytes = capEval.RequiredBytes
	ev.CapacityGrantedBytes = capEval.GrantedBytes
	ev.CapacityReclaimRequired = capEval.ReclaimRequired
}

// blockedReason derives a stable rejection code for a route that selected no
// provider. If every candidate that reached health evaluation was suppressed by
// an open breaker, it reports that specifically; otherwise the route simply had
// no eligible provider.
func blockedReason(plan routePlan) string {
	reasons := map[string]bool{}
	for _, c := range plan.candidates {
		if r := c.GetRejectionReason(); r != "" {
			reasons[r] = true
		}
	}
	// Report the most actionable cause first. A local route that failed capacity
	// or an open breaker is more informative than the expected locality rejection
	// of the remote peer that the profile forbids.
	for _, r := range []string{"insufficient_capacity", "provider_breaker_open"} {
		if reasons[r] {
			return r
		}
	}
	return "no_eligible_route"
}

// blockedCapacityVerdict returns the capacity verdict of a capacity-rejected
// candidate so blocked evidence can carry it. Empty when capacity was not the
// cause.
func blockedCapacityVerdict(plan routePlan) string {
	for _, c := range plan.candidates {
		if c.GetRejectionReason() == "insufficient_capacity" {
			return c.GetCapacityVerdict()
		}
	}
	return ""
}

func (s *Service) executionAttempts(plan routePlan) []*routingv1.RouteCandidate {
	var out []*routingv1.RouteCandidate
	for _, candidate := range plan.candidates {
		if candidate.GetSelected() {
			out = append([]*routingv1.RouteCandidate{candidate}, out...)
			continue
		}
		if candidate.GetFallbackEligible() {
			out = append(out, candidate)
		}
	}
	return out
}
