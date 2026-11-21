package routing

import (
	"context"
	"errors"
	"strings"
	"time"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	"ai-gateway/internal/providers"
	"github.com/vrooli/ai-provider-go/policy"
)

func (s *Service) Preview(ctx context.Context, req *sharedv1.GatewayRequest) (*routingv1.PreviewRouteResponse, error) {
	issues := s.validator.Validate(req)
	if len(issues) > 0 {
		return &routingv1.PreviewRouteResponse{Valid: false, Issues: issues}, nil
	}
	plan := s.plan(ctx, req)
	return &routingv1.PreviewRouteResponse{
		Valid:            true,
		Candidates:       plan.candidates,
		SelectedProvider: plan.selectedProvider,
		PolicyReasons:    plan.policyReasons,
		FallbackAllowed:  plan.fallbackAllowed,
		RoutePlanId:      plan.id,
	}, nil
}

func (s *Service) Execute(ctx context.Context, req *sharedv1.GatewayRequest, input string) (*routingv1.ExecuteRouteResponse, error) {
	return s.execute(ctx, req, input)
}

// plannedLocalRightsAdmit revalidates the plan's local model authorization
// against the concrete role and current trusted context immediately before
// dispatch. A role or receipt change fails closed.
func (s *Service) plannedLocalRightsAdmit(ctx context.Context, candidate *routingv1.RouteCandidate, resolvedRole providers.ResolvedRole, plan routePlan) bool {
	key := candidateKey(candidate.GetProvider(), candidate.GetRole())
	planned, found := plan.rights[key]
	currentContext, contextErr := currentRightsContext(ctx, s.rightsContext)
	if !plannedCandidateIdentityMatches(planned, found, candidate, resolvedRole) || contextErr != nil {
		return false
	}
	if err := RequireLocalRight(planned.decision, policy.Model{Rights: resolvedRole.Rights}, policy.RightsUseLocalInference, currentContext); err != nil {
		return false
	}
	return plannedReceiptMatches(planned, resolvedRole, candidate, currentContext)
}

func currentRightsContext(ctx context.Context, source RightsContextSource) (policy.RightsContext, error) {
	if source == nil {
		return policy.RightsContext{}, errors.New("trusted context source unavailable")
	}
	return source(ctx)
}

func plannedCandidateIdentityMatches(planned plannedLocalRight, found bool, candidate *routingv1.RouteCandidate, resolvedRole providers.ResolvedRole) bool {
	return found && resolvedRole.Model == planned.model && strings.EqualFold(resolvedRole.Locality, candidate.GetLocality()) &&
		sameRuntimeArtifactIdentity(planned.runtimeIdentity, resolvedRole.RuntimeIdentity) &&
		planned.runtimeDigest == resolvedRole.ObservedRuntimeDigest &&
		planned.policySchemaVersion == resolvedRole.PolicySchemaVersion && planned.policyFileSHA256 == resolvedRole.PolicyFileSHA256 &&
		RuntimeArtifactIdentityMatches(resolvedRole.RuntimeIdentity, resolvedRole.Rights, candidate.GetProvider(), resolvedRole.Model, resolvedRole.ObservedRuntimeDigest)
}

func plannedReceiptMatches(planned plannedLocalRight, resolvedRole providers.ResolvedRole, candidate *routingv1.RouteCandidate, currentContext policy.RightsContext) bool {
	evaluatedAt, err := time.Parse(time.RFC3339Nano, planned.decision.EvaluatedAt)
	if err != nil {
		return false
	}
	receipt := policy.NewRightsDecisionReceipt(policy.RightsDecisionReceiptExpectation{
		Role: resolvedRole.Role, Provider: candidate.GetProvider(), Model: resolvedRole.Model,
		Use: policy.RightsUseLocalInference, PolicySchemaVersion: resolvedRole.PolicySchemaVersion,
		ArtifactRights: resolvedRole.Rights, RuntimeIdentity: resolvedRole.RuntimeIdentity, Context: currentContext,
	}, evaluatedAt)
	if err := policy.VerifyRightsDecisionReceipt(receipt, policy.RightsDecisionReceiptExpectation{
		Role: resolvedRole.Role, Provider: candidate.GetProvider(), Model: resolvedRole.Model,
		Use: policy.RightsUseLocalInference, PolicySchemaVersion: planned.policySchemaVersion,
		ArtifactRights: resolvedRole.Rights, RuntimeIdentity: resolvedRole.RuntimeIdentity, Context: currentContext,
	}); err != nil {
		return false
	}
	return receipt.ReceiptSHA256 == planned.receiptSHA256
}

// requestedTemperature applies the caller's explicit control against one
// candidate's declared support.
//
// Anything short of "honored" refuses the candidate. That is deliberate: the
// caller made a promise-shaped request, and a provider that accepts the field
// and discards it keeps the promise no better than one that rejects it — the
// difference is only whether the failure is visible. Refusing both keeps the
// gateway from returning output that silently ignored the caller's control.
func requestedTemperature(req *sharedv1.GatewayRequest, resolved providers.ResolvedRole) (*float64, error) {
	sampling := req.GetSampling()
	if sampling == nil || sampling.Temperature == nil {
		return nil, nil
	}
	if resolved.TemperatureSupport() != providers.SamplingHonored {
		return nil, errUnsupportedSampling
	}
	value := sampling.GetTemperature()
	return &value, nil
}

var errUnsupportedSampling = errors.New("resolved role does not honor an explicit temperature")
