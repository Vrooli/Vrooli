package families_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"plan-manager/internal/families"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	familiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/families"
	"google.golang.org/protobuf/proto"
	_ "modernc.org/sqlite"
)

func newService(t *testing.T) *families.Service {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(families.Schema)))
	return families.NewService(families.NewSQLiteRepository(db))
}

func member(id string) *familiesv1.FamilyMember {
	return &familiesv1.FamilyMember{PlanId: id, Role: familiesv1.MemberRole_MEMBER_ROLE_SUPPORTING, State: familiesv1.MemberState_MEMBER_STATE_PENDING, SourceRevision: 7}
}

func putMember(t *testing.T, svc *families.Service, family *familiesv1.PlanFamily, id string) *familiesv1.PlanFamily {
	t.Helper()
	updated, err := svc.PutMember(context.Background(), &familiesv1.PutMemberRequest{FamilyId: family.GetFamilyId(), ExpectedRevision: family.GetRevision(), Member: member(id)})
	require.NoError(t, err)
	return updated
}

func TestFamilyRoundTripsIdentityContextClaimsGraphAndReview(t *testing.T) { // [REQ:PM-FAMILY-001]
	ctx := context.Background()
	svc := newService(t)
	family, err := svc.Create(ctx, &familiesv1.CreateFamilyRequest{Slug: "validation-cutover", Outcome: "one validation owner", SharedContext: "receipt migration", Policy: &familiesv1.FamilyPolicy{MaximumParallelPlans: 2, UnknownInteractionsSequential: true, RequireReviewBeforeLaunch: true}})
	require.NoError(t, err)
	family = putMember(t, svc, family, "plan-a")
	family = putMember(t, svc, family, "plan-b")

	claim := func(id, plan string) *familiesv1.ResourceClaim {
		return &familiesv1.ResourceClaim{ClaimId: id, PlanId: plan, Kind: familiesv1.ClaimKind_CLAIM_KIND_SCHEMA, Access: familiesv1.ClaimAccess_CLAIM_ACCESS_EXCLUSIVE_WRITE, Resource: "validation_receipts", Source: "plan", Resolved: true}
	}
	family, err = svc.PutClaim(ctx, &familiesv1.PutClaimRequest{FamilyId: family.GetFamilyId(), ExpectedRevision: family.GetRevision(), Claim: claim("claim-a", "plan-a")})
	require.NoError(t, err)
	family, err = svc.PutClaim(ctx, &familiesv1.PutClaimRequest{FamilyId: family.GetFamilyId(), ExpectedRevision: family.GetRevision(), Claim: claim("claim-b", "plan-b")})
	require.NoError(t, err)

	family, err = svc.ProposeGraph(ctx, &familiesv1.ProposeGraphRequest{FamilyId: family.GetFamilyId(), ExpectedRevision: family.GetRevision()})
	require.NoError(t, err)
	require.Len(t, family.GetGraph().GetEdges(), 1, "overlapping exclusive claims infer a conflict")
	require.Len(t, family.GetGraph().GetProposedFrontier(), 2, "conflicting plans cannot share a batch")
	frontier, err := svc.Frontier(ctx, family.GetFamilyId())
	require.NoError(t, err)
	require.False(t, frontier.GetLaunchable(), "unreviewed graph must not launch")

	family, err = svc.ReviewGraph(ctx, &familiesv1.ReviewGraphRequest{FamilyId: family.GetFamilyId(), ExpectedRevision: family.GetRevision(), GraphRevision: family.GetGraph().GetRevision(), Decision: familiesv1.ReviewDecision_REVIEW_DECISION_APPROVED, Reviewer: "operator", Rationale: "claims verified"})
	require.NoError(t, err)
	frontier, err = svc.Frontier(ctx, family.GetFamilyId())
	require.NoError(t, err)
	require.True(t, frontier.GetLaunchable())
	require.Equal(t, "validation-cutover", family.GetSlug())
	require.Equal(t, "receipt migration", family.GetSharedContext())
	require.Equal(t, uint64(7), family.GetMembers()[0].GetSourceRevision(), "child plan identity metadata round-trips unchanged")

	loaded, err := svc.Get(ctx, family.GetFamilyId())
	require.NoError(t, err)
	require.True(t, proto.Equal(family, loaded), "durable aggregate must round-trip semantically")
	renderedFamily, markdown, err := svc.Render(ctx, family.GetFamilyId())
	require.NoError(t, err)
	require.True(t, proto.Equal(family, renderedFamily))
	require.Contains(t, markdown, "# Plan family: validation-cutover")
	require.Contains(t, markdown, "Graph revision:")
	require.Contains(t, markdown, "REVIEW_DECISION_APPROVED")
}

func TestFamilyExecutionStateIsComputedFromMemberState(t *testing.T) {
	ctx := context.Background()
	svc := newService(t)
	family, err := svc.Create(ctx, &familiesv1.CreateFamilyRequest{Slug: "execution-state", Outcome: "derive aggregate state"})
	require.NoError(t, err)
	family = putMember(t, svc, family, "plan-a")
	require.Equal(t, familiesv1.FamilyExecutionState_FAMILY_EXECUTION_STATE_PENDING, family.GetExecutionState())

	family, err = svc.ProposeGraph(ctx, &familiesv1.ProposeGraphRequest{FamilyId: family.GetFamilyId(), ExpectedRevision: family.GetRevision()})
	require.NoError(t, err)
	family, err = svc.ReviewGraph(ctx, &familiesv1.ReviewGraphRequest{FamilyId: family.GetFamilyId(), ExpectedRevision: family.GetRevision(), GraphRevision: family.GetGraph().GetRevision(), Decision: familiesv1.ReviewDecision_REVIEW_DECISION_APPROVED, Reviewer: "operator", Rationale: "independent child"})
	require.NoError(t, err)
	running := member("plan-a")
	running.State = familiesv1.MemberState_MEMBER_STATE_RUNNING
	running.ExecutionId = "child-run-1"
	running.AdmissionKey = "claim-child-run-1"
	family, err = svc.PutMember(ctx, &familiesv1.PutMemberRequest{FamilyId: family.GetFamilyId(), ExpectedRevision: family.GetRevision(), Member: running})
	require.NoError(t, err)
	require.Equal(t, familiesv1.FamilyExecutionState_FAMILY_EXECUTION_STATE_RUNNING, family.GetExecutionState())
	require.NotNil(t, family.GetReview(), "execution progress must preserve topology approval")
	require.NotNil(t, family.GetLastActivityAt())

	running.State = familiesv1.MemberState_MEMBER_STATE_SUCCEEDED
	family, err = svc.PutMember(ctx, &familiesv1.PutMemberRequest{FamilyId: family.GetFamilyId(), ExpectedRevision: family.GetRevision(), Member: running})
	require.NoError(t, err)
	require.Equal(t, familiesv1.FamilyExecutionState_FAMILY_EXECUTION_STATE_SUCCEEDED, family.GetExecutionState())
}

func TestFamilySafetyPolicyCannotBeDisabledByPartialProtoPolicy(t *testing.T) {
	ctx := context.Background()
	svc := newService(t)
	family, err := svc.Create(ctx, &familiesv1.CreateFamilyRequest{
		Slug:    "safe-defaults",
		Outcome: "preserve launch safety",
		Policy:  &familiesv1.FamilyPolicy{MaximumParallelPlans: 3},
	})
	require.NoError(t, err)
	require.True(t, family.GetPolicy().GetRequireReviewBeforeLaunch())
	require.True(t, family.GetPolicy().GetUnknownInteractionsSequential())

	family, err = svc.Update(ctx, &familiesv1.UpdateFamilyRequest{
		FamilyId:         family.GetFamilyId(),
		ExpectedRevision: family.GetRevision(),
		Outcome:          family.GetOutcome(),
		Policy:           &familiesv1.FamilyPolicy{MaximumParallelPlans: 4},
	})
	require.NoError(t, err)
	require.True(t, family.GetPolicy().GetRequireReviewBeforeLaunch())
	require.True(t, family.GetPolicy().GetUnknownInteractionsSequential())
}

func TestSkippedMemberPreventsSuccessfulAggregateState(t *testing.T) {
	ctx := context.Background()
	svc := newService(t)
	family, err := svc.Create(ctx, &familiesv1.CreateFamilyRequest{Slug: "skipped", Outcome: "report incomplete work"})
	require.NoError(t, err)
	skipped := member("plan-a")
	skipped.State = familiesv1.MemberState_MEMBER_STATE_SKIPPED
	family, err = svc.PutMember(ctx, &familiesv1.PutMemberRequest{FamilyId: family.GetFamilyId(), ExpectedRevision: family.GetRevision(), Member: skipped})
	require.NoError(t, err)
	require.Equal(t, familiesv1.FamilyExecutionState_FAMILY_EXECUTION_STATE_FAILED, family.GetExecutionState())
}

func TestFamilyRejectsDanglingEdgesCyclesAndStaleWrites(t *testing.T) { // [REQ:PM-FAMILY-002]
	ctx := context.Background()
	svc := newService(t)
	family, err := svc.Create(ctx, &familiesv1.CreateFamilyRequest{Slug: "safe-frontier", Outcome: "never launch unsafe work"})
	require.NoError(t, err)
	family = putMember(t, svc, family, "a")
	family = putMember(t, svc, family, "b")

	_, err = svc.ProposeGraph(ctx, &familiesv1.ProposeGraphRequest{FamilyId: family.GetFamilyId(), ExpectedRevision: family.GetRevision(), ExplicitEdges: []*familiesv1.FamilyEdge{{FromPlanId: "a", ToPlanId: "missing", Kind: familiesv1.EdgeKind_EDGE_KIND_DEPENDENCY}}})
	require.ErrorContains(t, err, "unknown member")

	cyclic, err := svc.ProposeGraph(ctx, &familiesv1.ProposeGraphRequest{FamilyId: family.GetFamilyId(), ExpectedRevision: family.GetRevision(), ExplicitEdges: []*familiesv1.FamilyEdge{{FromPlanId: "a", ToPlanId: "b", Kind: familiesv1.EdgeKind_EDGE_KIND_DEPENDENCY}, {FromPlanId: "b", ToPlanId: "a", Kind: familiesv1.EdgeKind_EDGE_KIND_DEPENDENCY}}})
	require.NoError(t, err)
	require.True(t, cyclic.GetGraph().GetCyclic())
	frontier, err := svc.Frontier(ctx, family.GetFamilyId())
	require.NoError(t, err)
	require.False(t, frontier.GetLaunchable())
	_, err = svc.ReviewGraph(ctx, &familiesv1.ReviewGraphRequest{FamilyId: cyclic.GetFamilyId(), ExpectedRevision: cyclic.GetRevision(), GraphRevision: cyclic.GetGraph().GetRevision(), Decision: familiesv1.ReviewDecision_REVIEW_DECISION_APPROVED, Reviewer: "operator", Rationale: "unsafe"})
	require.ErrorContains(t, err, "cyclic graph")

	_, err = svc.Update(ctx, &familiesv1.UpdateFamilyRequest{FamilyId: cyclic.GetFamilyId(), ExpectedRevision: family.GetRevision(), Outcome: "stale"})
	require.True(t, errors.Is(err, families.ErrConflict))
}
