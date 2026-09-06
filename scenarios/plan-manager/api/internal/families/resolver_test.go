package families_test

import (
	"context"
	"testing"

	"plan-manager/internal/families"
	"plan-manager/internal/planmodel"

	"github.com/stretchr/testify/require"
	familiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/families"
)

func TestPhysicalClaimOverlapSurvivesSemanticKindsAndGlobs(t *testing.T) {
	for _, pair := range [][2]string{
		{"scenarios/foo/**", "scenarios/foo/api/schema.sql"},
		{"scenarios/*/api/**", "scenarios/foo/api/schema.sql"},
		{"scenarios/foo/api/*_test.go", "scenarios/foo/api/service_test.go"},
		{"./scenarios/foo/../bar/api", "scenarios/bar/api/schema.sql"},
	} {
		t.Run(pair[0], func(t *testing.T) {
			svc := newService(t)
			ctx := context.Background()
			family, err := svc.Create(ctx, &familiesv1.CreateFamilyRequest{Slug: "overlap", Outcome: "serialize physical conflicts", Policy: &familiesv1.FamilyPolicy{MaximumParallelPlans: 2}})
			require.NoError(t, err)
			for index, id := range []string{"a", "b"} {
				family = putMember(t, svc, family, id)
				for _, claim := range families.ClaimsFromPlan(planmodel.Plan{ID: id, ChangeBoundary: planmodel.ChangeBoundary{AcceptanceAllow: []string{pair[index]}}}) {
					family, err = svc.PutClaim(ctx, &familiesv1.PutClaimRequest{FamilyId: family.GetFamilyId(), ExpectedRevision: family.GetRevision(), Claim: claim})
					require.NoError(t, err)
				}
			}
			family, err = svc.ProposeGraph(ctx, &familiesv1.ProposeGraphRequest{FamilyId: family.GetFamilyId(), ExpectedRevision: family.GetRevision()})
			require.NoError(t, err)
			require.Len(t, family.GetGraph().GetEdges(), 1)
			require.Len(t, family.GetGraph().GetProposedFrontier(), 2)
		})
	}
}

func TestClaimsFromPlanClassifiesOwnedSurfacesAndUnknownInteraction(t *testing.T) {
	plan := planmodel.Plan{ID: "plan-a", ChangeBoundary: planmodel.ChangeBoundary{AcceptanceAllow: []string{"packages/proto/schemas/demo/**", "scenarios/demo/api/internal/runtime/**", "scenarios/demo/api/schema.sql"}}}
	claims := families.ClaimsFromPlan(plan)
	require.Len(t, claims, 3)
	require.Equal(t, familiesv1.ClaimKind_CLAIM_KIND_API_CONTRACT, claims[0].GetKind())
	require.Equal(t, familiesv1.ClaimKind_CLAIM_KIND_RUNTIME_PLANT, claims[1].GetKind())
	require.Equal(t, familiesv1.ClaimKind_CLAIM_KIND_SCHEMA, claims[2].GetKind())

	unknown := families.ClaimsFromPlan(planmodel.Plan{ID: "plan-b"})
	require.Equal(t, familiesv1.ClaimKind_CLAIM_KIND_UNKNOWN_INTERACTION, unknown[0].GetKind())
	require.False(t, unknown[0].GetResolved())
}

func TestFamilyEdgesFromPlanGraphPreservesDependsOnDirection(t *testing.T) {
	edges := families.FamilyEdgesFromPlanGraph([]planmodel.PlanEdge{{FromPlanID: "downstream", ToPlanID: "prerequisite", Kind: planmodel.EdgeKindDependsOn}, {FromPlanID: "new", ToPlanID: "old", Kind: planmodel.EdgeKindSupersedes}})
	require.Len(t, edges, 1)
	require.Equal(t, "downstream", edges[0].GetFromPlanId())
	require.Equal(t, "prerequisite", edges[0].GetToPlanId())
	require.Equal(t, familiesv1.EdgeProvenance_EDGE_PROVENANCE_MIGRATED, edges[0].GetProvenance())
}
