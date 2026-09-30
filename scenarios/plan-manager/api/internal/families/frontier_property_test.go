package families

import (
	"fmt"
	"math/rand"
	"testing"

	familiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/families"
)

func TestFrontierPropertyNeverCoSchedulesDependenciesOrConflicts(t *testing.T) { // [REQ:PM-FAMILY-002]
	random := rand.New(rand.NewSource(42))
	for sample := 0; sample < 100; sample++ {
		family := &familiesv1.PlanFamily{Policy: &familiesv1.FamilyPolicy{MaximumParallelPlans: 4}}
		for i := 0; i < 8; i++ {
			family.Members = append(family.Members, &familiesv1.FamilyMember{PlanId: fmt.Sprintf("p%d", i), State: familiesv1.MemberState_MEMBER_STATE_PENDING})
		}
		edges := []*familiesv1.FamilyEdge{}
		for i := 1; i < 8; i++ {
			if random.Intn(2) == 0 {
				edges = append(edges, &familiesv1.FamilyEdge{FromPlanId: fmt.Sprintf("p%d", i), ToPlanId: fmt.Sprintf("p%d", random.Intn(i)), Kind: familiesv1.EdgeKind_EDGE_KIND_DEPENDENCY})
			}
		}
		for i := 0; i < 7; i++ {
			if random.Intn(3) == 0 {
				edges = append(edges, &familiesv1.FamilyEdge{FromPlanId: fmt.Sprintf("p%d", i), ToPlanId: fmt.Sprintf("p%d", i+1), Kind: familiesv1.EdgeKind_EDGE_KIND_CONFLICT})
			}
		}
		batches, cyclic, _ := compileFrontier(family, edges)
		if cyclic {
			t.Fatalf("generated acyclic graph reported cyclic in sample %d", sample)
		}
		ordinal := map[string]uint32{}
		for _, batch := range batches {
			for _, id := range batch.GetPlanIds() {
				ordinal[id] = batch.GetOrdinal()
			}
		}
		for _, edge := range edges {
			switch edge.GetKind() {
			case familiesv1.EdgeKind_EDGE_KIND_DEPENDENCY:
				if ordinal[edge.GetToPlanId()] >= ordinal[edge.GetFromPlanId()] {
					t.Fatalf("dependency co-scheduled or ordered after dependent: %v", edge)
				}
			case familiesv1.EdgeKind_EDGE_KIND_CONFLICT:
				if ordinal[edge.GetFromPlanId()] == ordinal[edge.GetToPlanId()] {
					t.Fatalf("conflict co-scheduled: %v", edge)
				}
			}
		}
	}
}

func TestFrontierPartialFailureKeepsIndependentWorkRunnable(t *testing.T) {
	family := &familiesv1.PlanFamily{Policy: &familiesv1.FamilyPolicy{MaximumParallelPlans: 2}, Members: []*familiesv1.FamilyMember{
		{PlanId: "failed", State: familiesv1.MemberState_MEMBER_STATE_FAILED},
		{PlanId: "dependent", State: familiesv1.MemberState_MEMBER_STATE_PENDING},
		{PlanId: "independent", State: familiesv1.MemberState_MEMBER_STATE_PENDING},
	}}
	edges := []*familiesv1.FamilyEdge{{FromPlanId: "dependent", ToPlanId: "failed", Kind: familiesv1.EdgeKind_EDGE_KIND_DEPENDENCY}}
	batches, cyclic, diagnostics := compileFrontier(family, edges)
	if cyclic {
		t.Fatal("a failed dependency is not a cycle")
	}
	if len(batches) != 1 || len(batches[0].GetPlanIds()) != 1 || batches[0].GetPlanIds()[0] != "independent" {
		t.Fatalf("independent frontier = %v", batches)
	}
	if len(diagnostics) == 0 {
		t.Fatal("blocked dependent must remain explainable")
	}
}

func TestRunningMemberRetainsClaimsAndCapacity(t *testing.T) {
	family := &familiesv1.PlanFamily{Policy: &familiesv1.FamilyPolicy{MaximumParallelPlans: 2}, Members: []*familiesv1.FamilyMember{
		{PlanId: "a", State: familiesv1.MemberState_MEMBER_STATE_RUNNING},
		{PlanId: "b", State: familiesv1.MemberState_MEMBER_STATE_PENDING},
		{PlanId: "c", State: familiesv1.MemberState_MEMBER_STATE_PENDING},
		{PlanId: "d", State: familiesv1.MemberState_MEMBER_STATE_PENDING},
	}}
	edges := []*familiesv1.FamilyEdge{{FromPlanId: "b", ToPlanId: "a", Kind: familiesv1.EdgeKind_EDGE_KIND_CONFLICT}, {FromPlanId: "c", ToPlanId: "a", Kind: familiesv1.EdgeKind_EDGE_KIND_DEPENDENCY}}
	batches, cyclic, _ := compileFrontier(family, edges)
	if cyclic || len(batches) != 1 || len(batches[0].GetPlanIds()) != 1 || batches[0].GetPlanIds()[0] != "d" {
		t.Fatalf("unsafe frontier: %v cycle=%v", batches, cyclic)
	}
	family.Policy.MaximumParallelPlans = 1
	batches, cyclic, _ = compileFrontier(family, edges)
	if cyclic || len(batches) != 0 {
		t.Fatalf("running member must consume capacity: %v cycle=%v", batches, cyclic)
	}
}
