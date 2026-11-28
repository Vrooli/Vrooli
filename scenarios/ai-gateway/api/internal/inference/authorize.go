package inference

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	"ai-gateway/internal/providers"
	"github.com/vrooli/ai-provider-go/policy"
	profilepolicy "github.com/vrooli/ai-provider-go/routing"
)

func (r *ResourceRepository) admitsRequest(ctx context.Context, request ProviderRequest, provider, locality string) bool {
	if profileAdmitsRequest(request.Profile, request.PrivacyClass, locality) {
		return true
	}
	return r.admitsConfidentialRemoteRequest(ctx, request, provider, locality)
}

func (r *ResourceRepository) admitsConfidentialRemoteRequest(ctx context.Context, request ProviderRequest, provider, locality string) bool {
	if request.PrivacyClass != sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL || !strings.EqualFold(locality, "remote") || r == nil || r.consent == nil {
		return false
	}
	selectedSource := strings.ToLower(strings.TrimSpace(providers.Source(ctx)))
	if !confidentialRemoteSourceAllows(selectedSource, provider) || !confidentialRemoteProfileAllows(request.Profile, locality) {
		return false
	}
	return r.consent.Verify(ctx, providers.PrivacyConsent(ctx), selectedSource) == nil
}

func confidentialRemoteSourceAllows(source, provider string) bool {
	provider = strings.ToLower(strings.TrimSpace(provider))
	switch source {
	case "own-key":
		return provider != providers.ProviderMetered && provider != providers.ProviderOllama && provider != providers.ProviderOllaya
	case "vrooli-credits":
		return provider == providers.ProviderMetered
	default:
		return false
	}
}

func confidentialRemoteProfileAllows(profile sharedv1.Profile, locality string) bool {
	if profile == sharedv1.Profile_PROFILE_LOCAL_ONLY || profile == sharedv1.Profile_PROFILE_PRIVACY_SENSITIVE {
		return false
	}
	profileAllowed, _ := profilepolicy.Admits(profile.String(), sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL.String(), locality)
	return profileAllowed
}

func localModelRequiresRights(provider string, resolved providers.ResolvedRole) bool {
	if !strings.EqualFold(strings.TrimSpace(resolved.Locality), "local") {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case providers.ProviderOllama, providers.ProviderOllaya, providers.ProviderPiiDetector:
		return true
	default:
		return strings.TrimSpace(resolved.Model) != "" || resolved.Rights != nil
	}
}

func sameRuntimeIdentity(left, right *policy.RuntimeArtifactIdentity) bool {
	return left != nil && right != nil && *left == *right
}

// authorizeLocalCandidate binds one candidate's rights to its current role,
// artifact identity, use and trusted context, then repeats the binding against
// a fresh resource resolution immediately before command dispatch.
func (r *ResourceRepository) authorizeLocalCandidate(ctx context.Context, adapter providers.Adapter, resourceRole string, resolved providers.ResolvedRole) (providers.ResolvedRole, string, policy.RightsDecision, error) {
	if !localModelRequiresRights(adapter.Provider, resolved) {
		return resolved, "", policy.RightsDecision{}, nil
	}
	decision, evaluatedAt, currentContext, receipt, err := r.evaluateLocalCandidateRights(ctx, adapter, resolved)
	if err != nil {
		return providers.ResolvedRole{}, "", decision, err
	}
	refreshed, err := r.revalidateLocalCandidateRights(ctx, adapter, resourceRole, resolved, decision, evaluatedAt, currentContext, receipt)
	if err != nil {
		return providers.ResolvedRole{}, "", decision, err
	}
	return refreshed, refreshed.RuntimeIdentity.Digest, decision, nil
}

func (r *ResourceRepository) evaluateLocalCandidateRights(ctx context.Context, adapter providers.Adapter, resolved providers.ResolvedRole) (policy.RightsDecision, time.Time, policy.RightsContext, policy.RightsDecisionReceipt, error) {
	if r.rightsContext == nil || strings.TrimSpace(resolved.Model) == "" || resolved.Rights == nil {
		return policy.RightsDecision{}, time.Time{}, policy.RightsContext{}, policy.RightsDecisionReceipt{}, errors.New("local model rights, model identity, or trusted organization context is missing")
	}
	currentContext, err := r.rightsContext(ctx)
	if err != nil {
		return policy.RightsDecision{}, time.Time{}, policy.RightsContext{}, policy.RightsDecisionReceipt{}, fmt.Errorf("trusted rights context is unavailable: %w", err)
	}
	decision, evaluatedAt, err := evaluateLocalRightsDecision(adapter, resolved, currentContext)
	if err != nil {
		return decision, time.Time{}, currentContext, policy.RightsDecisionReceipt{}, err
	}
	receipt := policy.NewRightsDecisionReceipt(policy.RightsDecisionReceiptExpectation{
		Role: resolved.Role, Provider: adapter.Provider, Model: resolved.Model,
		Use: policy.RightsUseLocalInference, PolicySchemaVersion: resolved.PolicySchemaVersion,
		ArtifactRights: resolved.Rights, RuntimeIdentity: resolved.RuntimeIdentity, Context: currentContext,
	}, evaluatedAt)
	if receipt.Decision.DecisionSHA256 != decision.DecisionSHA256 || receipt.Decision.EvaluatedAt != decision.EvaluatedAt {
		return decision, time.Time{}, currentContext, policy.RightsDecisionReceipt{}, errors.New("local model rights receipt does not match the evaluated decision")
	}
	if err := policy.RequireAllowedRightsDecisionReceipt(receipt); err != nil {
		return decision, time.Time{}, currentContext, policy.RightsDecisionReceipt{}, err
	}
	return decision, evaluatedAt, currentContext, receipt, nil
}

func evaluateLocalRightsDecision(adapter providers.Adapter, resolved providers.ResolvedRole, currentContext policy.RightsContext) (policy.RightsDecision, time.Time, error) {
	decision := policy.EvaluateRights(resolved.Rights, policy.RightsUseLocalInference, currentContext, time.Now())
	if decision.Status != policy.RightsAllowed {
		return decision, time.Time{}, fmt.Errorf("local model rights decision %s (%s)", decision.Status, decision.Reason)
	}
	if !policy.RuntimeArtifactIdentityMatches(resolved.RuntimeIdentity, resolved.Rights, adapter.Provider, resolved.Model, resolved.ObservedRuntimeDigest) {
		return decision, time.Time{}, errors.New("local model runtime artifact identity is missing or differs from the observed digest")
	}
	if strings.TrimSpace(resolved.PolicyFileSHA256) == "" || strings.TrimSpace(resolved.PolicySchemaVersion) == "" {
		return decision, time.Time{}, errors.New("local model policy file identity is missing")
	}
	evaluatedAt, err := time.Parse(time.RFC3339Nano, decision.EvaluatedAt)
	if err != nil {
		return decision, time.Time{}, errors.New("local model rights decision timestamp is invalid")
	}
	return decision, evaluatedAt, nil
}

func (r *ResourceRepository) revalidateLocalCandidateRights(ctx context.Context, adapter providers.Adapter, resourceRole string, resolved providers.ResolvedRole, decision policy.RightsDecision, evaluatedAt time.Time, currentContext policy.RightsContext, receipt policy.RightsDecisionReceipt) (providers.ResolvedRole, error) {
	refreshed, err := adapter.ResolveRole(ctx, resourceRole)
	if err != nil {
		return providers.ResolvedRole{}, fmt.Errorf("re-resolve local model rights identity before execution: %w", err)
	}
	refreshedContext, err := r.rightsContext(ctx)
	if err != nil {
		return providers.ResolvedRole{}, fmt.Errorf("trusted rights context is unavailable before execution: %w", err)
	}
	if !sameLocalRightsIdentity(refreshed, resolved) {
		return providers.ResolvedRole{}, errors.New("local model policy or exact runtime artifact identity changed before execution")
	}
	if err := policy.RequireCurrentRightsDecision(decision, refreshed.Rights, policy.RightsUseLocalInference, refreshedContext); err != nil {
		return providers.ResolvedRole{}, err
	}
	if !policy.RuntimeArtifactIdentityMatches(refreshed.RuntimeIdentity, refreshed.Rights, adapter.Provider, refreshed.Model, refreshed.ObservedRuntimeDigest) {
		return providers.ResolvedRole{}, errors.New("local model runtime artifact identity changed before execution")
	}
	currentReceipt := policy.NewRightsDecisionReceipt(policy.RightsDecisionReceiptExpectation{
		Role: refreshed.Role, Provider: adapter.Provider, Model: refreshed.Model,
		Use: policy.RightsUseLocalInference, PolicySchemaVersion: refreshed.PolicySchemaVersion,
		ArtifactRights: refreshed.Rights, RuntimeIdentity: refreshed.RuntimeIdentity, Context: refreshedContext,
	}, evaluatedAt)
	if err := policy.VerifyRightsDecisionReceipt(currentReceipt, policy.RightsDecisionReceiptExpectation{
		Role: resolved.Role, Provider: adapter.Provider, Model: resolved.Model,
		Use: policy.RightsUseLocalInference, PolicySchemaVersion: resolved.PolicySchemaVersion,
		ArtifactRights: resolved.Rights, RuntimeIdentity: resolved.RuntimeIdentity, Context: currentContext,
	}); err != nil {
		return providers.ResolvedRole{}, fmt.Errorf("local model rights receipt changed before execution: %w", err)
	}
	if _, err := currentReceipt.MarshalCanonical(); err != nil {
		return providers.ResolvedRole{}, err
	}
	refreshed.RightsDecisionSHA256 = currentReceipt.Decision.DecisionSHA256
	refreshed.RightsEvaluatedAt = currentReceipt.Decision.EvaluatedAt
	refreshed.RightsReceiptSHA256 = currentReceipt.ReceiptSHA256
	refreshed.RightsContextSHA256 = currentReceipt.Decision.ContextSHA256
	return refreshed, nil
}

func sameLocalRightsIdentity(refreshed, resolved providers.ResolvedRole) bool {
	return refreshed.Provider == resolved.Provider && refreshed.Role == resolved.Role && refreshed.Model == resolved.Model &&
		strings.EqualFold(refreshed.Locality, resolved.Locality) && refreshed.ObservedRuntimeDigest == resolved.ObservedRuntimeDigest &&
		sameRuntimeIdentity(refreshed.RuntimeIdentity, resolved.RuntimeIdentity) &&
		refreshed.PolicyFileSHA256 == resolved.PolicyFileSHA256 && refreshed.PolicySchemaVersion == resolved.PolicySchemaVersion
}
