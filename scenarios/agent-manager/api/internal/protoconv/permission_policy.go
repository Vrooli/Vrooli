package protoconv

import (
	"agent-manager/internal/permissionpolicy"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
)

func PermissionPolicyStatusToProto(status permissionpolicy.Status) *apipb.PermissionPolicyStatus {
	result := &apipb.PermissionPolicyStatus{
		Path: status.Path,
		Requirement: &apipb.PermissionPolicyRequirement{
			Required: status.Requirement.Required,
			Reason:   status.Requirement.Reason,
		},
		Ready:        status.Ready,
		ActiveDigest: status.ActiveDigest,
	}
	if status.ActivatedAt != nil {
		result.ActivatedAt = TimestampToProto(*status.ActivatedAt)
	}
	if attempt := status.LastReloadAttempt; attempt != nil {
		result.LastReloadAttempt = &apipb.PermissionPolicyReloadAttempt{
			AttemptedAt: TimestampToProto(attempt.AttemptedAt),
			Succeeded:   attempt.Succeeded,
			Digest:      attempt.Digest,
			Diagnostic:  PermissionPolicyDiagnosticToProto(attempt.Diagnostic),
		}
	}
	return result
}

func PermissionPolicyDiagnosticToProto(diagnostic *permissionpolicy.Diagnostic) *apipb.PermissionPolicyDiagnostic {
	if diagnostic == nil {
		return nil
	}
	return &apipb.PermissionPolicyDiagnostic{Code: diagnostic.Code, Message: diagnostic.Message, Cause: diagnostic.Cause}
}

func PermissionPolicyCatalogToProto(catalog *permissionpolicy.Catalog) *apipb.PermissionPolicyCatalog {
	if catalog == nil {
		return nil
	}
	result := &apipb.PermissionPolicyCatalog{
		SchemaVersion: int32(catalog.SchemaVersion),
		Metadata:      &apipb.PermissionPolicyCatalogMetadata{CatalogId: catalog.Metadata.CatalogID, UpdatedAt: catalog.Metadata.UpdatedAt},
		TargetScopes:  catalog.Scopes(),
	}
	for _, rule := range catalog.Rules {
		result.Rules = append(result.Rules, &apipb.PermissionPolicyRule{
			Id:                      rule.ID,
			Action:                  rule.Action,
			Matcher:                 PermissionPolicyMatcherToProto(rule.Matcher),
			Rationale:               rule.Rationale,
			Owner:                   rule.Owner,
			TargetScope:             rule.TargetScope,
			RequiresHardEnforcement: rule.RequiresHardEnforcement,
		})
	}
	return result
}

func PermissionPolicyMatcherToProto(matcher permissionpolicy.Matcher) *apipb.PermissionPolicyMatcher {
	return &apipb.PermissionPolicyMatcher{Kind: matcher.Kind, Pattern: matcher.Pattern}
}

func PermissionPolicyResourcePlanToProto(resource permissionpolicy.ResourcePlan) *apipb.PermissionPolicyResourceResult {
	result := &apipb.PermissionPolicyResourceResult{
		RunnerType:         RunnerTypeToProto(resource.Runner),
		Scope:              resource.Scope,
		Installed:          resource.Installed,
		Status:             resource.Status,
		Error:              resource.Error,
		DesiredDigest:      resource.DesiredDigest,
		DesiredFingerprint: resource.DesiredFingerprint,
		LiveFingerprint:    resource.LiveFingerprint,
		Drift:              resource.Drift,
		Changes:            append([]string(nil), resource.Changes...),
		NativePaths:        append([]string(nil), resource.NativePaths...),
		Enforcement: &apipb.PermissionPolicyEnforcement{
			Permissions: resource.Enforcement.Permissions,
			Caveats:     append([]string(nil), resource.Enforcement.Caveats...),
		},
	}
	for _, matcher := range resource.UnsupportedMatchers {
		result.UnsupportedMatchers = append(result.UnsupportedMatchers, PermissionPolicyMatcherToProto(matcher))
	}
	return result
}

func PermissionPolicyPlanToProto(plan permissionpolicy.AggregatePlan) *apipb.PermissionPolicyPlan {
	result := &apipb.PermissionPolicyPlan{
		CatalogDigest:                 plan.CatalogDigest,
		HardEnforcementSatisfied:      plan.HardEnforcementSatisfied,
		MissingHardEnforcementRuleIds: append([]string(nil), plan.MissingHardEnforcementRuleIDs...),
	}
	for _, resource := range plan.Resources {
		result.Resources = append(result.Resources, PermissionPolicyResourcePlanToProto(resource))
	}
	return result
}

func PermissionPolicyReconcileResultToProto(reconcile *permissionpolicy.ReconcileResult) *apipb.PermissionPolicyReconcileResult {
	if reconcile == nil {
		return nil
	}
	result := &apipb.PermissionPolicyReconcileResult{
		CatalogDigest:                 reconcile.CatalogDigest,
		StartedAt:                     TimestampToProto(reconcile.StartedAt),
		FinishedAt:                    TimestampToProto(reconcile.FinishedAt),
		ExplicitlyAuthorized:          reconcile.ExplicitlyAuthorized,
		Success:                       reconcile.Success,
		HardEnforcementSatisfied:      reconcile.HardEnforcementSatisfied,
		MissingHardEnforcementRuleIds: append([]string(nil), reconcile.MissingHardEnforcementRuleIDs...),
	}
	for _, resource := range reconcile.Resources {
		result.Resources = append(result.Resources, PermissionPolicyResourcePlanToProto(resource))
	}
	return result
}
