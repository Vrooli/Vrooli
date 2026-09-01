package catalog

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
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

func seedDuplicateNode(t *testing.T, s *Store, ctx context.Context, kind offerspb.NodeKind, name string) *offerspb.Node {
	t.Helper()
	n := &offerspb.Node{Id: uuid.NewString(), Kind: kind, Name: name, Status: offerspb.Status_IDEA, CreatedAt: timestamppb.New(time.Date(2026, time.January, 2, 12, 0, 0, 0, time.UTC))}
	_, err := s.db.ExecContext(ctx, `INSERT INTO nodes(id,kind,name,status,trigger_id,created_at,actual_account_id) VALUES(?,?,?,?,?,?,?)`, n.Id, int32(n.Kind), n.Name, int32(n.Status), "", n.CreatedAt.AsTime().UTC().Format(time.RFC3339Nano), "")
	require.NoError(t, err)
	return n
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

func TestTypedDeliverableClassesAndEnablesRules(t *testing.T) { // [REQ:GRAPH-001]
	s, ctx := testCatalog(t)
	marketed, err := s.CreateNodeWithDetails(ctx, offerspb.NodeKind_DELIVERABLE, "Marketed", offerspb.Status_IDEA, "", "", offerspb.DeliverableClass_MARKETED, offerspb.FinishBar_CUSTOMER_FACING)
	require.NoError(t, err)
	enabling, err := s.CreateNodeWithDetails(ctx, offerspb.NodeKind_DELIVERABLE, "Enabler", offerspb.Status_IDEA, "", "", offerspb.DeliverableClass_ENABLING, offerspb.FinishBar_INTERNAL)
	require.NoError(t, err)
	_, _, err = s.SetReleaseRank(ctx, enabling.Id, 1, "operator")
	require.ErrorContains(t, err, "enabling_deliverables_are_unranked")
	_, err = s.CreateEdge(ctx, &offerspb.Edge{FromId: enabling.Id, ToId: marketed.Id, Kind: "enables"})
	require.NoError(t, err)
	_, err = s.CreateEdge(ctx, &offerspb.Edge{FromId: marketed.Id, ToId: enabling.Id, Kind: "enables"})
	require.ErrorContains(t, err, "enables_is_acyclic")
	_, err = s.CreateEdge(ctx, &offerspb.Edge{FromId: marketed.Id, ToId: enabling.Id, Kind: "enables"})
	require.ErrorContains(t, err, "enables_is_acyclic")
	_, err = s.CreateEdge(ctx, &offerspb.Edge{FromId: marketed.Id, ToId: enabling.Id, Kind: "serves"})
	require.ErrorContains(t, err, "typed_edge_matrix")
}

func TestPrerequisitesWalksEnablersTransitively(t *testing.T) { // [REQ:RELEASE-003]
	s, ctx := testCatalog(t)
	stream, err := s.CreateNode(ctx, offerspb.NodeKind_STREAM, "voice_minutes", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	root, err := s.CreateNodeWithDetails(ctx, offerspb.NodeKind_DELIVERABLE, "Web console", offerspb.Status_SHIPPED, "", "", offerspb.DeliverableClass_MARKETED, offerspb.FinishBar_CUSTOMER_FACING)
	require.NoError(t, err)
	a, err := s.CreateNodeWithDetails(ctx, offerspb.NodeKind_DELIVERABLE, "AI gateway", offerspb.Status_ACTIVE, "", "", offerspb.DeliverableClass_ENABLING, offerspb.FinishBar_OPERATOR_FACING)
	require.NoError(t, err)
	b, err := s.CreateNodeWithDetails(ctx, offerspb.NodeKind_DELIVERABLE, "Desktop", offerspb.Status_ACTIVE, "", "", offerspb.DeliverableClass_ENABLING, offerspb.FinishBar_INTERNAL)
	require.NoError(t, err)
	_, err = s.CreateEdge(ctx, &offerspb.Edge{FromId: root.Id, ToId: stream.Id, Kind: "unlocks"})
	require.NoError(t, err)
	_, err = s.CreateEdge(ctx, &offerspb.Edge{FromId: a.Id, ToId: root.Id, Kind: "enables"})
	require.NoError(t, err)
	_, err = s.CreateEdge(ctx, &offerspb.Edge{FromId: b.Id, ToId: a.Id, Kind: "enables"})
	require.NoError(t, err)
	_, _, err = s.SetReleaseRank(ctx, root.Id, 1, "operator")
	require.NoError(t, err)
	got, err := s.PrerequisitesWithOptions(ctx, stream.Id, 2, false)
	require.NoError(t, err)
	require.Len(t, got.Tree, 2)
	require.Equal(t, "Web console", got.Tree[0].Node.Name)
	require.EqualValues(t, 1, got.Tree[0].Depth)
	require.EqualValues(t, 1, got.Tree[0].DerivedUrgency)
	require.Equal(t, "AI gateway", got.Tree[1].Node.Name)
	require.EqualValues(t, 2, got.Tree[1].Depth)
	require.Len(t, got.Unshipped, 1)
	got, err = s.PrerequisitesWithOptions(ctx, stream.Id, 3, false)
	require.NoError(t, err)
	require.Len(t, got.Tree, 3)
	require.Equal(t, "Desktop", got.Tree[2].Node.Name)
	require.EqualValues(t, 1, got.Tree[2].DerivedUrgency)
}

func TestReleaseLadderRanksDeliverablesAndWalksPrerequisites(t *testing.T) { // [REQ:RELEASE-001] [REQ:RELEASE-002]
	s, ctx := testCatalog(t)
	first, err := s.CreateNode(ctx, offerspb.NodeKind_DELIVERABLE, "Console", offerspb.Status_SHIPPED, "", "")
	require.NoError(t, err)
	second, err := s.CreateNode(ctx, offerspb.NodeKind_DELIVERABLE, "Monitor", offerspb.Status_ACTIVE, "", "")
	require.NoError(t, err)
	ramp, err := s.CreateNode(ctx, offerspb.NodeKind_RAMP, "desktop", offerspb.Status_ACTIVE, "", "")
	require.NoError(t, err)
	stream, err := s.CreateNode(ctx, offerspb.NodeKind_STREAM, "voice_minutes", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	audience, err := s.CreateNode(ctx, offerspb.NodeKind_AUDIENCE, "developer", offerspb.Status_ACTIVE, "", "")
	require.NoError(t, err)
	_, err = s.CreateEdge(ctx, &offerspb.Edge{FromId: first.Id, ToId: ramp.Id, Kind: "unlocks"})
	require.NoError(t, err)
	_, err = s.CreateEdge(ctx, &offerspb.Edge{FromId: second.Id, ToId: stream.Id, Kind: "unlocks"})
	require.NoError(t, err)
	_, err = s.CreateEdge(ctx, &offerspb.Edge{FromId: second.Id, ToId: audience.Id, Kind: "serves"})
	require.NoError(t, err)
	_, _, err = s.SetReleaseRank(ctx, first.Id, 1, "operator")
	require.NoError(t, err)
	_, _, err = s.SetReleaseRank(ctx, second.Id, 2, "operator")
	require.NoError(t, err)
	ladder, err := s.ReleaseLadder(ctx, false)
	require.NoError(t, err)
	require.Len(t, ladder.Entries, 2)
	require.Equal(t, "Console", ladder.Entries[0].Deliverable.Name)
	require.Equal(t, "Monitor", ladder.Entries[1].Deliverable.Name)
	require.Len(t, ladder.Entries[1].CumulativeRamps, 1)
	require.Equal(t, "desktop", ladder.Entries[1].CumulativeRamps[0].Name)
	prerequisites, err := s.Prerequisites(ctx, stream.Id)
	require.NoError(t, err)
	require.Len(t, prerequisites.Deliverables, 1)
	require.Len(t, prerequisites.Unshipped, 1)
	require.Equal(t, "Monitor", prerequisites.Unshipped[0].Name)
}

func TestEdgePricePresenceDistinguishesAbsentFromDeclaredZero(t *testing.T) { // [REQ:MIG-001]
	s, ctx := testCatalog(t)
	offer, err := s.CreateNode(ctx, offerspb.NodeKind_OFFER, "Offer", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	absolute, err := s.CreateNode(ctx, offerspb.NodeKind_VARIANT, "Absent price", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	zero, err := s.CreateNode(ctx, offerspb.NodeKind_VARIANT, "Declared zero", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	_, err = s.CreateEdge(ctx, &offerspb.Edge{FromId: offer.Id, ToId: absolute.Id, Kind: "sells_at"})
	require.NoError(t, err)
	_, err = s.CreateEdge(ctx, &offerspb.Edge{FromId: offer.Id, ToId: zero.Id, Kind: "sells_at", IntendedPriceDeclared: true, Currency: "USD"})
	require.NoError(t, err)
	edges, err := s.ListEdges(ctx, offer.Id)
	require.NoError(t, err)
	require.Len(t, edges, 2)
	byVariant := map[string]*offerspb.Edge{}
	for _, edge := range edges {
		byVariant[edge.ToId] = edge
	}
	require.False(t, byVariant[absolute.Id].IntendedPriceDeclared)
	require.Zero(t, byVariant[absolute.Id].IntendedPriceMinor)
	require.True(t, byVariant[zero.Id].IntendedPriceDeclared)
	require.Zero(t, byVariant[zero.Id].IntendedPriceMinor)
	require.Equal(t, "USD", byVariant[zero.Id].Currency)
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

func TestMergeMovesEveryReferenceThenRemovesDuplicate(t *testing.T) { // [REQ:GRAPH-005]
	s, ctx := testCatalog(t)
	survivor, err := s.CreateNode(ctx, offerspb.NodeKind_OFFER, "Same offer", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	duplicate := seedDuplicateNode(t, s, ctx, offerspb.NodeKind_OFFER, "Same offer")
	target, err := s.CreateNode(ctx, offerspb.NodeKind_VARIANT, "Target", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	edge, err := s.CreateEdge(ctx, &offerspb.Edge{FromId: duplicate.Id, ToId: target.Id, Kind: "sells_at"})
	require.NoError(t, err)
	trigger, err := s.AddTrigger(ctx, &offerspb.Trigger{NodeId: duplicate.Id, FactName: "paying_users", Operator: ">=", Threshold: 1})
	require.NoError(t, err)
	_, err = s.Proposal(ctx, duplicate.Id, "agent", offerspb.Status_ACTIVE, "ready")
	require.NoError(t, err)
	_, err = s.db.ExecContext(ctx, `INSERT INTO evaluations(id,node_id,verdict,fact_name,explanation,evaluated_at) VALUES(?,?,?,?,?,?)`, "evaluation-duplicate", duplicate.Id, int32(offerspb.Verdict_SATISFIED), trigger.FactName, "test", time.Now().UTC().Format(time.RFC3339Nano))
	require.NoError(t, err)
	_, err = s.db.ExecContext(ctx, `INSERT INTO migration_findings(id,node_id,source_file,reference,reason,created_at) VALUES(?,?,?,?,?,?)`, "finding-duplicate", duplicate.Id, "source.md", "#same", "test", time.Now().UTC().Format(time.RFC3339Nano))
	require.NoError(t, err)

	dryReport, err := s.MergeNodes(ctx, &offerspb.MergeNodesRequest{SurvivingId: survivor.Id, DuplicateId: duplicate.Id, Actor: "operator", DryRun: true})
	require.NoError(t, err)
	require.EqualValues(t, 1, dryReport.MovedEdges)
	require.EqualValues(t, 1, dryReport.MovedTriggers)
	require.EqualValues(t, 1, dryReport.MovedEvaluations)
	var count int
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM nodes WHERE id=?`, duplicate.Id).Scan(&count))
	require.Equal(t, 1, count)
	report, err := s.MergeNodes(ctx, &offerspb.MergeNodesRequest{SurvivingId: survivor.Id, DuplicateId: duplicate.Id, Actor: "operator", DryRun: false})
	require.NoError(t, err)
	require.Equal(t, survivor.Id, report.Surviving.Id)
	require.EqualValues(t, 1, report.MovedEdges)
	require.EqualValues(t, 1, report.MovedTriggers)
	require.EqualValues(t, 1, report.MovedEvaluations)
	require.EqualValues(t, 1, report.MovedProposals)
	require.EqualValues(t, 1, report.MovedFindings)
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM nodes WHERE id=?`, duplicate.Id).Scan(&count))
	require.Zero(t, count)
	for _, table := range []string{"edges", "triggers", "evaluations", "proposals", "migration_findings"} {
		require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table+` WHERE `+map[string]string{"edges": "from_id", "triggers": "node_id", "evaluations": "node_id", "proposals": "node_id", "migration_findings": "node_id"}[table]+`=?`, duplicate.Id).Scan(&count))
		require.Zero(t, count, table)
	}
	var auditReason, auditActor string
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT actor,reason FROM catalog_audit WHERE node_id=? ORDER BY created_at DESC LIMIT 1`, survivor.Id).Scan(&auditActor, &auditReason))
	require.Equal(t, "operator", auditActor)
	require.Contains(t, auditReason, duplicate.Id)
	require.NotEmpty(t, edge.Id)
}

func TestMergeRefusesAcrossKinds(t *testing.T) { // [REQ:GRAPH-005]
	s, ctx := testCatalog(t)
	offer, err := s.CreateNode(ctx, offerspb.NodeKind_OFFER, "Offer", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	channel, err := s.CreateNode(ctx, offerspb.NodeKind_CHANNEL, "Channel", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	_, err = s.MergeNodes(ctx, &offerspb.MergeNodesRequest{SurvivingId: offer.Id, DuplicateId: channel.Id, Actor: "operator"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "OFFER")
	require.Contains(t, err.Error(), "CHANNEL")
	require.Contains(t, err.Error(), offer.Id)
	require.Contains(t, err.Error(), channel.Id)
}

func TestCreateNodeRefusesDuplicateKindAndName(t *testing.T) {
	s, ctx := testCatalog(t)
	first, err := s.CreateNode(ctx, offerspb.NodeKind_OFFER, "Unique identity", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	_, err = s.CreateNode(ctx, offerspb.NodeKind_OFFER, "Unique identity", offerspb.Status_IDEA, "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), first.Id)
}

func TestEnsureMigrationsRefusesUniqueIndexWhileDuplicatesExist(t *testing.T) {
	s, ctx := testCatalog(t)
	seedDuplicateNode(t, s, ctx, offerspb.NodeKind_OFFER, "Duplicate identity")
	seedDuplicateNode(t, s, ctx, offerspb.NodeKind_OFFER, "Duplicate identity")
	err := EnsureMigrations(ctx, s.db)
	require.ErrorContains(t, err, "catalog-merge")
	var index string
	require.Error(t, s.db.QueryRowContext(ctx, `SELECT name FROM pragma_index_list('nodes') WHERE name='nodes_kind_name'`).Scan(&index))
}

func TestEnsureMigrationsCreatesUniqueIndexesForCleanCatalog(t *testing.T) {
	s, ctx := testCatalog(t)
	require.NoError(t, EnsureMigrations(ctx, s.db))
	var nodeIndex, edgeIndex string
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT name FROM pragma_index_list('nodes') WHERE name='nodes_kind_name'`).Scan(&nodeIndex))
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT name FROM pragma_index_list('edges') WHERE name='edges_from_to_kind'`).Scan(&edgeIndex))
	require.Equal(t, "nodes_kind_name", nodeIndex)
	require.Equal(t, "edges_from_to_kind", edgeIndex)
}

func TestCreateEdgeUpsertPreservesDeclaredPriceAgainstAbsentPrice(t *testing.T) {
	s, ctx := testCatalog(t)
	offer, err := s.CreateNode(ctx, offerspb.NodeKind_OFFER, "Priced offer", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	variant, err := s.CreateNode(ctx, offerspb.NodeKind_VARIANT, "Priced variant", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	first, err := s.CreateEdge(ctx, &offerspb.Edge{FromId: offer.Id, ToId: variant.Id, Kind: "sells_at", IntendedPriceMinor: 2900, Currency: "USD", IntendedPriceDeclared: true})
	require.NoError(t, err)
	second, err := s.CreateEdge(ctx, &offerspb.Edge{FromId: offer.Id, ToId: variant.Id, Kind: "sells_at"})
	require.NoError(t, err)
	require.Equal(t, first.Id, second.Id)
	require.EqualValues(t, 2900, second.IntendedPriceMinor)
	require.Equal(t, "USD", second.Currency)
	require.True(t, second.IntendedPriceDeclared)
}

func TestMergeCollapsesDuplicateEdgeAndReportsIt(t *testing.T) { // [REQ:GRAPH-005]
	s, ctx := testCatalog(t)
	survivor, err := s.CreateNode(ctx, offerspb.NodeKind_OFFER, "Same offer", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	duplicate := seedDuplicateNode(t, s, ctx, offerspb.NodeKind_OFFER, "Same offer")
	target, err := s.CreateNode(ctx, offerspb.NodeKind_VARIANT, "Target", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	kept, err := s.CreateEdge(ctx, &offerspb.Edge{FromId: survivor.Id, ToId: target.Id, Kind: "sells_at"})
	require.NoError(t, err)
	moved, err := s.CreateEdge(ctx, &offerspb.Edge{FromId: duplicate.Id, ToId: target.Id, Kind: "sells_at"})
	require.NoError(t, err)
	report, err := s.MergeNodes(ctx, &offerspb.MergeNodesRequest{SurvivingId: survivor.Id, DuplicateId: duplicate.Id, Actor: "operator"})
	require.NoError(t, err)
	require.Contains(t, report.CollapsedEdgeIds, moved.Id)
	require.NotContains(t, report.CollapsedEdgeIds, kept.Id)
	edges, err := s.ListEdges(ctx, survivor.Id)
	require.NoError(t, err)
	require.Len(t, edges, 1)
	require.Equal(t, kept.Id, edges[0].Id)
}

// An imported node carries an empty actual_account_id, and before MapAccount
// existed there was no way to set one — so the board reported "no ledger
// account mapping" for every row forever and the actuals join could not fire.
// [REQ:INT-002]
func TestMapAccountAttachesAnImportedNodeAndAuditsTheChange(t *testing.T) {
	s, ctx := testCatalog(t)
	node, err := s.CreateNode(ctx, offerspb.NodeKind_OFFER, "business", offerspb.Status_ACTIVE, "", "")
	require.NoError(t, err)
	require.Empty(t, node.ActualAccountId, "a created node starts unmapped, as the importer leaves it")

	mapped, prior, err := s.MapAccount(ctx, &offerspb.MapAccountRequest{
		NodeId: node.Id, ActualAccountId: "acct-subscription-revenue", Actor: "operator", Reason: "adoption wiring",
	})
	require.NoError(t, err)
	require.Equal(t, "acct-subscription-revenue", mapped.ActualAccountId)
	require.Empty(t, prior)

	var stored string
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT actual_account_id FROM nodes WHERE id=?`, node.Id).Scan(&stored))
	require.Equal(t, "acct-subscription-revenue", stored)

	var audits int
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM catalog_audit WHERE node_id=?`, node.Id).Scan(&audits))
	require.Equal(t, 1, audits, "the mapping change must leave an audit row")

	var reason string
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT reason FROM catalog_audit WHERE node_id=?`, node.Id).Scan(&reason))
	require.Contains(t, reason, "adoption wiring")
	require.Contains(t, reason, "(unmapped)", "the audit must record what the mapping replaced")

	// Remapping records the prior value rather than overwriting history.
	_, prior, err = s.MapAccount(ctx, &offerspb.MapAccountRequest{
		NodeId: node.Id, ActualAccountId: "acct-services-revenue", Actor: "operator",
	})
	require.NoError(t, err)
	require.Equal(t, "acct-subscription-revenue", prior)

	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM catalog_audit WHERE node_id=?`, node.Id).Scan(&audits))
	require.Equal(t, 2, audits, "corrections are new audit entries, never edits")
}

func TestMapAccountRefusesAnUnknownNodeAndRequiresAnActor(t *testing.T) {
	s, ctx := testCatalog(t)
	node, err := s.CreateNode(ctx, offerspb.NodeKind_OFFER, "business", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)

	_, _, err = s.MapAccount(ctx, &offerspb.MapAccountRequest{NodeId: "missing", ActualAccountId: "x", Actor: "operator"})
	require.Error(t, err)

	_, _, err = s.MapAccount(ctx, &offerspb.MapAccountRequest{NodeId: node.Id, ActualAccountId: "x"})
	require.ErrorContains(t, err, "actor")
}
