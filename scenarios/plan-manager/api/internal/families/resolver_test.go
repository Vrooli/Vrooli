package families_test

import (
	"testing"

	"plan-manager/internal/families"
	"plan-manager/internal/planmodel"

	"github.com/stretchr/testify/require"
	familiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/families"
)

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
