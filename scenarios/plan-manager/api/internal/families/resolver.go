package families

import (
	"fmt"
	"strings"

	"plan-manager/internal/planmodel"

	familiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/families"
)

// ClaimsFromPlan converts the existing detailed plan boundary and references
// into conservative typed claims. It is an adapter: the child Plan remains the
// source of its own identity and execution contract.
func ClaimsFromPlan(plan planmodel.Plan) []*familiesv1.ResourceClaim {
	claims := make([]*familiesv1.ResourceClaim, 0, len(plan.ChangeBoundary.AcceptanceAllow)+len(plan.References))
	for index, path := range plan.ChangeBoundary.Normalized().AcceptanceAllow {
		kind := claimKindForPath(path)
		claims = append(claims, &familiesv1.ResourceClaim{ClaimId: fmt.Sprintf("%s:boundary:%d", plan.ID, index+1), PlanId: plan.ID, Kind: kind, Access: familiesv1.ClaimAccess_CLAIM_ACCESS_EXCLUSIVE_WRITE, Resource: path, Source: "change_boundary", Resolved: !strings.Contains(path, "<")})
	}
	for index, ref := range plan.References {
		target := strings.TrimSpace(ref.Target)
		if target == "" {
			continue
		}
		kind := familiesv1.ClaimKind_CLAIM_KIND_PATH
		if strings.Contains(target, ".proto") || strings.HasPrefix(target, "api:") {
			kind = familiesv1.ClaimKind_CLAIM_KIND_API_CONTRACT
		}
		claims = append(claims, &familiesv1.ResourceClaim{ClaimId: fmt.Sprintf("%s:reference:%d", plan.ID, index+1), PlanId: plan.ID, Kind: kind, Access: familiesv1.ClaimAccess_CLAIM_ACCESS_READ, Resource: target, Source: "reference", Resolved: ref.Resolution == planmodel.ResolutionResolved})
	}
	if len(claims) == 0 {
		claims = append(claims, &familiesv1.ResourceClaim{ClaimId: plan.ID + ":unknown", PlanId: plan.ID, Kind: familiesv1.ClaimKind_CLAIM_KIND_UNKNOWN_INTERACTION, Access: familiesv1.ClaimAccess_CLAIM_ACCESS_EXCLUSIVE_WRITE, Resource: "repository", Source: "missing_change_boundary", Detail: "no machine-readable interaction boundary", Resolved: false})
	}
	return claims
}

func claimKindForPath(path string) familiesv1.ClaimKind {
	lower := strings.ToLower(path)
	switch {
	case strings.Contains(lower, "packages/proto/gen/") || strings.Contains(lower, ".vrooli/generated/"):
		return familiesv1.ClaimKind_CLAIM_KIND_GENERATED_OUTPUT
	case strings.HasSuffix(lower, ".proto") || strings.Contains(lower, "packages/proto/schemas/"):
		return familiesv1.ClaimKind_CLAIM_KIND_API_CONTRACT
	case strings.HasSuffix(lower, ".sql") || strings.Contains(lower, "/schema") || strings.Contains(lower, "migrations/"):
		return familiesv1.ClaimKind_CLAIM_KIND_SCHEMA
	case strings.Contains(lower, "/runtime/") || strings.Contains(lower, "service.json") || strings.Contains(lower, "lifecycle/"):
		return familiesv1.ClaimKind_CLAIM_KIND_RUNTIME_PLANT
	case strings.Contains(lower, "test") || strings.Contains(lower, "requirements/"):
		return familiesv1.ClaimKind_CLAIM_KIND_VALIDATION_TARGET
	default:
		return familiesv1.ClaimKind_CLAIM_KIND_PATH
	}
}

// FamilyEdgesFromPlanGraph preserves the existing PlanEdge meaning: FromPlanID
// is the depending plan and ToPlanID is its prerequisite.
func FamilyEdgesFromPlanGraph(edges []planmodel.PlanEdge) []*familiesv1.FamilyEdge {
	out := make([]*familiesv1.FamilyEdge, 0, len(edges))
	for _, edge := range edges {
		if edge.Kind != planmodel.EdgeKindDependsOn {
			continue
		}
		out = append(out, &familiesv1.FamilyEdge{FromPlanId: edge.FromPlanID, ToPlanId: edge.ToPlanID, Kind: familiesv1.EdgeKind_EDGE_KIND_DEPENDENCY, Provenance: familiesv1.EdgeProvenance_EDGE_PROVENANCE_MIGRATED, Reason: "migrated from Plan Manager depends_on edge"})
	}
	return out
}
