package offers

import (
	"context"
	"log"
	"testing"
	"time"

	"offer-desk/internal/catalog"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"
	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func testOfferService(t *testing.T, clock *schedule.Fake) (*Service, context.Context) {
	t.Helper()
	sqlDB := databasetest.NewSQLite(t)
	db := database.NewFromPrimary(sqlDB)
	ctx := context.Background()
	store := catalog.NewStore(db, clock.Now)
	require.NoError(t, database.EnsureSchemas(ctx, sqlDB, database.SchemaProviderFunc(store.Schema)))
	return &Service{store: store, logger: log.Default(), clock: clock}, ctx
}

func TestSchedulerPromotesSatisfiedCandidateWithoutManualEvaluate(t *testing.T) { // [REQ:GATE-003]
	t.Setenv("OFFER_EVALUATION_INTERVAL", "1ms")
	clock := schedule.NewFake(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	s, ctx := testOfferService(t, clock)
	n, err := s.store.CreateNode(ctx, offerspb.NodeKind_OFFER, "Scheduled offer", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	_, err = s.store.AddTrigger(ctx, &offerspb.Trigger{NodeId: n.Id, FactName: "activation_rate", Operator: ">=", Threshold: 0.5})
	require.NoError(t, err)
	_, err = s.store.Transition(ctx, n.Id, offerspb.Status_CANDIDATE, "operator")
	require.NoError(t, err)
	_, err = s.store.AddFact(ctx, &offerspb.Fact{Name: "activation_rate", Value: 0.8, ObservedAt: timestamppb.New(clock.Now()), StaleAfterDays: 10})
	require.NoError(t, err)
	s.startScheduler()
	clock.Tick()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		nodes, listErr := s.store.ListNodes(ctx, offerspb.NodeKind_NODE_KIND_UNSPECIFIED, offerspb.Status_STATUS_UNSPECIFIED)
		if listErr == nil && len(nodes) == 1 && nodes[0].Status == offerspb.Status_TRIGGER_MET {
			var evaluations int
			require.NoError(t, s.store.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM evaluations WHERE node_id=?`, n.Id).Scan(&evaluations))
			require.Equal(t, 1, evaluations)
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("scheduler did not promote the satisfied candidate")
}

func TestAgentPromotionCreatesListableProposalWithoutActivatingNode(t *testing.T) { // [REQ:GATE-005]
	clock := schedule.NewFake(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	s, ctx := testOfferService(t, clock)
	node, err := s.store.CreateNode(ctx, offerspb.NodeKind_OFFER, "Proposal offer", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)

	response, err := s.Promote(ctx, connect.NewRequest(&offerspb.PromoteRequest{NodeId: node.Id, Actor: "agent-7", Role: "agent"}))
	require.NoError(t, err)
	require.Equal(t, offerspb.Status_ACTIVE, response.Msg.Proposal.RequestedStatus)
	require.NotEmpty(t, response.Msg.Proposal.CreatedAt)
	require.NotEmpty(t, response.Msg.Proposal.EvidenceReference)

	nodes, err := s.store.ListNodes(ctx, offerspb.NodeKind_NODE_KIND_UNSPECIFIED, offerspb.Status_STATUS_UNSPECIFIED)
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	require.NotEqual(t, offerspb.Status_ACTIVE, nodes[0].Status)

	listed, err := s.ListProposals(ctx, connect.NewRequest(&offerspb.ListProposalsRequest{NodeId: node.Id}))
	require.NoError(t, err)
	require.Len(t, listed.Msg.Proposals, 1)
	require.Equal(t, response.Msg.Proposal.Id, listed.Msg.Proposals[0].Id)
}

func TestOperatorRetirementRecordsProposalDeclineHistory(t *testing.T) { // [REQ:GATE-005]
	clock := schedule.NewFake(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	s, ctx := testOfferService(t, clock)
	node, err := s.store.CreateNode(ctx, offerspb.NodeKind_OFFER, "Declined offer", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	_, err = s.store.Proposal(ctx, node.Id, "agent-7", offerspb.Status_ACTIVE, "ready for review")
	require.NoError(t, err)
	_, err = s.store.Transition(ctx, node.Id, offerspb.Status_RETIRED, "operator:decline:Not ready for this audience")
	require.NoError(t, err)

	listed, err := s.store.ListProposals(ctx, node.Id, offerspb.Status_STATUS_UNSPECIFIED)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Len(t, listed[0].DeclineHistory, 1)
	require.Equal(t, "operator", listed[0].DeclineHistory[0].Actor)
}

func TestBoardReportsLedgerUnavailableWithoutInventingActuals(t *testing.T) { // [REQ:INT-002] [REQ:INT-003] [REQ:INT-005]
	clock := schedule.NewFake(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	s, ctx := testOfferService(t, clock)
	_, err := s.store.CreateNode(ctx, offerspb.NodeKind_OFFER, "Active but unmeasured", offerspb.Status_ACTIVE, "", "")
	require.NoError(t, err)
	response, err := s.GetBoard(ctx, nil)
	require.NoError(t, err)
	require.Len(t, response.Msg.Entries, 1)
	require.Equal(t, int64(0), response.Msg.Entries[0].ActualMinor)
	require.False(t, response.Msg.Entries[0].ActualsAvailable)
	require.Len(t, response.Msg.Availability, 1)
	require.Contains(t, response.Msg.Availability[0].Reason, "actuals unavailable")
	require.Contains(t, response.Msg.Entries[0].RankReason, "earnings unknown")
	require.NotContains(t, response.Msg.Entries[0].RankReason, "earning nothing")
}

func TestEveryStatusHasAnExplicitRankReason(t *testing.T) {
	for _, status := range []offerspb.Status{
		offerspb.Status_STATUS_UNSPECIFIED, offerspb.Status_IDEA, offerspb.Status_CANDIDATE,
		offerspb.Status_TRIGGER_MET, offerspb.Status_ACTIVE, offerspb.Status_SHIPPED, offerspb.Status_RETIRED, offerspb.Status_PROPOSED,
	} {
		reason := rankReason(status, false, 0, "money-ledger.actuals")
		require.NotEmpty(t, reason, status.String())
	}
	require.NotEqual(t, rankReason(offerspb.Status_IDEA, false, 0, "money-ledger.actuals"), rankReason(offerspb.Status_RETIRED, false, 0, "money-ledger.actuals"))
}

func TestProposedStatusNamesPendingOperatorDecision(t *testing.T) {
	require.Equal(t, "awaiting operator decision", rankReason(offerspb.Status_PROPOSED, false, 0, "money-ledger.actuals"))
}

func TestActiveWithZeroActualsStillSaysEarningNothing(t *testing.T) {
	require.Equal(t, "active and earning nothing", rankReason(offerspb.Status_ACTIVE, true, 0, "money-ledger.actuals"))
	require.Equal(t, "active and earning", rankReason(offerspb.Status_ACTIVE, true, 100, "money-ledger.actuals"))
}

// The board's declared first priority is "which triggers fired since I last
// looked". Ordering must express that: a fired trigger has to be reachable
// without scrolling past retired drill rows. [REQ:INT-003]
func TestBoardOrdersFiredTriggersFirstAndRetiredLast(t *testing.T) {
	cases := []struct {
		name             string
		status           offerspb.Status
		actualsAvailable bool
		actualMinor      int64
	}{
		{"retired", offerspb.Status_RETIRED, false, 0},
		{"idea", offerspb.Status_IDEA, false, 0},
		{"candidate", offerspb.Status_CANDIDATE, false, 0},
		{"active earning", offerspb.Status_ACTIVE, true, 500},
		{"active unknown earnings", offerspb.Status_ACTIVE, false, 0},
		{"active earning nothing", offerspb.Status_ACTIVE, true, 0},
		{"proposed", offerspb.Status_PROPOSED, false, 0},
		{"trigger met", offerspb.Status_TRIGGER_MET, false, 0},
	}
	ranks := make([]int, len(cases))
	for i, c := range cases {
		ranks[i] = boardRank(c.status, c.actualsAvailable, c.actualMinor)
	}
	byName := map[string]int{}
	for i, c := range cases {
		byName[c.name] = ranks[i]
	}

	require.Less(t, byName["trigger met"], byName["proposed"], "a fired trigger outranks a pending proposal")
	require.Less(t, byName["proposed"], byName["active earning nothing"], "a decision waiting on the operator outranks a standing condition")
	require.Less(t, byName["active earning nothing"], byName["active unknown earnings"],
		"a confirmed zero outranks an unavailable read; missing evidence is not a business finding")
	require.Less(t, byName["active unknown earnings"], byName["active earning"])
	require.Less(t, byName["active earning"], byName["candidate"])
	require.Less(t, byName["candidate"], byName["idea"])
	require.Less(t, byName["idea"], byName["retired"], "retired rows must never outrank live work")
}
