package catalog

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func testCatalog(t *testing.T) (*Store, context.Context) {
	t.Helper()
	sqlDB := databasetest.NewSQLite(t)
	db := database.NewFromPrimary(sqlDB)
	ctx := context.Background()
	require.NoError(t, database.EnsureSchemas(ctx, sqlDB, database.SchemaProviderFunc(func() string { return (&Store{}).Schema() })))
	now := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	return NewStore(db, func() time.Time { return now }), ctx
}

func TestLifecycleReachesTriggerMetAndUnknownIsNotFalse(t *testing.T) { // [REQ:GATE-003] [REQ:GATE-004]
	s, ctx := testCatalog(t)
	n, err := s.CreateNode(ctx, offerspb.NodeKind_OFFER, "Launch offer", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	trigger, err := s.AddTrigger(ctx, &offerspb.Trigger{NodeId: n.Id, FactName: "activation_rate", Operator: ">=", Threshold: 0.5})
	require.NoError(t, err)
	n, err = s.Transition(ctx, n.Id, offerspb.Status_CANDIDATE, "operator")
	require.NoError(t, err)
	require.Equal(t, offerspb.Status_CANDIDATE, n.Status)
	evals, err := s.Evaluate(ctx, false)
	require.NoError(t, err)
	require.Len(t, evals, 1)
	require.Equal(t, offerspb.Verdict_UNKNOWN, evals[0].Verdict)
	_, err = s.AddFact(ctx, &offerspb.Fact{Name: trigger.FactName, Value: 0.75, ObservedAt: timestamppb.New(time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)), StaleAfterDays: 10})
	require.NoError(t, err)
	evals, err = s.Evaluate(ctx, false)
	require.NoError(t, err)
	require.Equal(t, offerspb.Verdict_SATISFIED, evals[0].Verdict)
	nodes, err := s.ListNodes(ctx, offerspb.NodeKind_NODE_KIND_UNSPECIFIED, offerspb.Status_STATUS_UNSPECIFIED)
	require.NoError(t, err)
	require.Equal(t, offerspb.Status_TRIGGER_MET, nodes[0].Status)
}

func TestIllegalTransitionNamesRuleAndTypedEdgesRejectInvalidPairs(t *testing.T) { // [REQ:GRAPH-002] [REQ:GRAPH-003]
	s, ctx := testCatalog(t)
	offer, err := s.CreateNode(ctx, offerspb.NodeKind_OFFER, "Offer", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	_, err = s.Transition(ctx, offer.Id, offerspb.Status_ACTIVE, "operator")
	require.ErrorContains(t, err, "legal_lifecycle_transition")
	variant, err := s.CreateNode(ctx, offerspb.NodeKind_VARIANT, "Variant", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	_, err = s.CreateEdge(ctx, &offerspb.Edge{FromId: offer.Id, ToId: variant.Id, Kind: "feeds"})
	require.ErrorContains(t, err, "typed_edge_matrix")
	edge, err := s.CreateEdge(ctx, &offerspb.Edge{FromId: offer.Id, ToId: variant.Id, Kind: "sells_at"})
	require.NoError(t, err)
	require.NotEmpty(t, edge.Id)
}

func TestTriggerCompositionAndFactFreshnessAreExplicit(t *testing.T) { // [REQ:GATE-002] [REQ:GATE-006] [REQ:GATE-008]
	s, ctx := testCatalog(t)
	n, err := s.CreateNode(ctx, offerspb.NodeKind_OFFER, "Compound offer", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	_, err = s.AddTrigger(ctx, &offerspb.Trigger{
		NodeId:      n.Id,
		FactName:    "activation_rate",
		Operator:    ">=",
		Threshold:   0.8,
		Composition: offerspb.TriggerComposition_ANY,
		Clauses: []*offerspb.TriggerClause{
			{FactName: "activation_rate", Operator: ">=", Threshold: 0.8},
			{FactName: "paying_users", Operator: ">=", Threshold: 10},
		},
	})
	require.NoError(t, err)
	_, err = s.Transition(ctx, n.Id, offerspb.Status_CANDIDATE, "operator")
	require.NoError(t, err)
	fact, err := s.AddFact(ctx, &offerspb.Fact{Name: "activation_rate", Value: 0.9, Dimension: "activation", ObservedAt: timestamppb.New(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))})
	require.NoError(t, err)
	require.EqualValues(t, 180, fact.StaleAfterDays)
	evals, err := s.Evaluate(ctx, true)
	require.NoError(t, err)
	require.Len(t, evals, 1)
	require.Equal(t, offerspb.Verdict_SATISFIED, evals[0].Verdict)
	require.ElementsMatch(t, []string{"activation_rate", "paying_users"}, evals[0].FactNames)
}

func TestCandidateRequiresTriggerAndPromotionIsOperatorOnly(t *testing.T) { // [REQ:GATE-001] [REQ:GATE-005]
	s, ctx := testCatalog(t)
	n, err := s.CreateNode(ctx, offerspb.NodeKind_OFFER, "Guarded offer", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	_, err = s.Transition(ctx, n.Id, offerspb.Status_CANDIDATE, "operator")
	require.ErrorContains(t, err, "candidate_requires_trigger")
	trigger, err := s.AddTrigger(ctx, &offerspb.Trigger{NodeId: n.Id, FactName: "paying_users", Operator: ">=", Threshold: 1})
	require.NoError(t, err)
	_, err = s.Transition(ctx, n.Id, offerspb.Status_CANDIDATE, "operator")
	require.NoError(t, err)
	_, err = s.Transition(ctx, n.Id, offerspb.Status_TRIGGER_MET, "scheduler")
	require.NoError(t, err)
	_, err = s.Proposal(ctx, n.Id, "agent-7", offerspb.Status_ACTIVE, "ready for review")
	require.NoError(t, err)
	_, err = s.Transition(ctx, n.Id, offerspb.Status_ACTIVE, "agent-7")
	require.ErrorContains(t, err, "operator_only_promotion")
	var status int32
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT status FROM nodes WHERE id=?`, n.Id).Scan(&status))
	require.Equal(t, int32(offerspb.Status_TRIGGER_MET), status)
	require.NotEmpty(t, trigger.Id)
}

func TestStaleFactIsUnknownAndLeavesCandidateInPlace(t *testing.T) { // [REQ:GATE-004] [REQ:GATE-008]
	s, ctx := testCatalog(t)
	n, err := s.CreateNode(ctx, offerspb.NodeKind_OFFER, "Stale benchmark offer", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	_, err = s.AddTrigger(ctx, &offerspb.Trigger{NodeId: n.Id, FactName: "pricing_comp", Operator: ">=", Threshold: 100})
	require.NoError(t, err)
	_, err = s.Transition(ctx, n.Id, offerspb.Status_CANDIDATE, "operator")
	require.NoError(t, err)
	_, err = s.AddFact(ctx, &offerspb.Fact{Name: "pricing_comp", Value: 200, Dimension: "pricing", ObservedAt: timestamppb.New(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))})
	require.NoError(t, err)
	evals, err := s.Evaluate(ctx, false)
	require.NoError(t, err)
	require.Len(t, evals, 1)
	require.Equal(t, offerspb.Verdict_UNKNOWN, evals[0].Verdict)
	require.Contains(t, evals[0].Explanation, "stale")
	nodes, err := s.ListNodes(ctx, offerspb.NodeKind_NODE_KIND_UNSPECIFIED, offerspb.Status_STATUS_UNSPECIFIED)
	require.NoError(t, err)
	require.Equal(t, offerspb.Status_CANDIDATE, nodes[0].Status)
}

func TestTransitionAuditIsAppendOnly(t *testing.T) { // [REQ:GRAPH-004]
	s, ctx := testCatalog(t)
	n, err := s.CreateNode(ctx, offerspb.NodeKind_OFFER, "Audited offer", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	_, err = s.Transition(ctx, n.Id, offerspb.Status_RETIRED, "operator")
	require.NoError(t, err)
	var count int
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM catalog_audit WHERE node_id=?`, n.Id).Scan(&count))
	require.Equal(t, 1, count)
	var prior, next int32
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT prior_status,next_status FROM catalog_audit WHERE node_id=?`, n.Id).Scan(&prior, &next))
	require.Equal(t, int32(offerspb.Status_IDEA), prior)
	require.Equal(t, int32(offerspb.Status_RETIRED), next)
}
