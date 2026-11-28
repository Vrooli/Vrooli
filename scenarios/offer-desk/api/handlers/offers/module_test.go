package offers

import (
	"context"
	"errors"
	"log"
	"os"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func testOfferService(t *testing.T, clock *schedule.Fake) (*Service, context.Context) {
	service, ctx := testOfferServiceWithEnv(t, clock, os.Getenv)
	// Most unit tests exercise local persistence and should not discover a live
	// Money Ledger from the host environment.
	service.resolveLedgerURL = nil
	return service, ctx
}

func testOfferServiceWithEnv(t *testing.T, clock *schedule.Fake, getenv func(string) string) (*Service, context.Context) {
	t.Helper()
	sqlDB := databasetest.NewSQLite(t)
	db := database.NewFromPrimary(sqlDB)
	ctx := context.Background()
	service := NewServiceWithEnv(db, log.Default(), clock, getenv)
	require.NoError(t, database.EnsureSchemas(ctx, sqlDB, database.SchemaProviderFunc(service.store.Schema)))
	return service, ctx
}

func TestNewServiceWithEnvUsesInjectedMoneyLedgerOverride(t *testing.T) {
	clock := schedule.NewFake(time.Date(2026, 11, 24, 0, 0, 0, 0, time.UTC))
	service, ctx := testOfferServiceWithEnv(t, clock, func(key string) string {
		if key == "MONEY_LEDGER_API_URL" {
			return "http://ledger.test"
		}
		return ""
	})

	baseURL, err := service.resolveLedgerURL(ctx)
	require.NoError(t, err)
	require.Equal(t, "http://ledger.test", baseURL)
}

// A consumer that requires producer time must be able to judge the ladder's
// freshness, so both ladder projections carry the service clock's reading.
func TestReleaseLadderProjectionsCarryProducerObservationTime(t *testing.T) { // [REQ:LADDER-011]
	now := time.Date(2026, 9, 15, 5, 48, 0, 0, time.UTC)
	s, ctx := testOfferService(t, schedule.NewFake(now))
	ladder, err := s.GetReleaseLadder(ctx, connect.NewRequest(&offerspb.ReleaseLadderRequest{}))
	require.NoError(t, err)
	require.NotNil(t, ladder.Msg.GeneratedAt)
	require.True(t, ladder.Msg.GeneratedAt.AsTime().Equal(now))
	enabling, err := s.GetEnablingDeliverables(ctx, connect.NewRequest(&offerspb.ReleaseLadderRequest{}))
	require.NoError(t, err)
	require.NotNil(t, enabling.Msg.GeneratedAt)
	require.True(t, enabling.Msg.GeneratedAt.AsTime().Equal(now))
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

func TestListFactsAndTriggersExposeStoredGateDefinitionsAndFreshness(t *testing.T) {
	clock := schedule.NewFake(time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC))
	s, ctx := testOfferService(t, clock)
	node, err := s.store.CreateNode(ctx, offerspb.NodeKind_OFFER, "Read surfaces", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	staleAt := clock.Now().Add(-24*time.Hour - time.Second)
	_, err = s.store.AddFact(ctx, &offerspb.Fact{Name: "a_stale", Value: 0.7, ObservedAt: timestamppb.New(staleAt), StaleAfterDays: 1, Dimension: "activation"})
	require.NoError(t, err)
	boundaryAt := clock.Now().Add(-24 * time.Hour)
	_, err = s.store.AddFact(ctx, &offerspb.Fact{Name: "b_boundary", Value: 0.7, ObservedAt: timestamppb.New(boundaryAt), StaleAfterDays: 1})
	require.NoError(t, err)
	_, err = s.store.AddFact(ctx, &offerspb.Fact{Name: "c_bad_clock", Value: 0.7, ObservedAt: timestamppb.New(clock.Now()), StaleAfterDays: 1})
	require.NoError(t, err)
	_, err = s.store.DB().ExecContext(ctx, `UPDATE facts SET observed_at='bad-time' WHERE name='c_bad_clock'`)
	require.NoError(t, err)
	_, err = s.store.AddTrigger(ctx, &offerspb.Trigger{NodeId: node.Id, FactName: "activation_rate", Operator: ">=", Threshold: 0.5, Clauses: []*offerspb.TriggerClause{{FactName: "activation_rate", Operator: ">=", Threshold: 0.5}}, Composition: offerspb.TriggerComposition_ALL})
	require.NoError(t, err)

	facts, err := s.ListFacts(ctx, connect.NewRequest(&offerspb.ListFactsRequest{}))
	require.NoError(t, err)
	require.Len(t, facts.Msg.Facts, 3)
	require.Equal(t, clock.Now(), facts.Msg.EvaluatedAt.AsTime())
	require.Equal(t, "a_stale", facts.Msg.Facts[0].Fact.Name)
	require.Equal(t, offerspb.FactFreshness_FACT_FRESHNESS_STALE, facts.Msg.Facts[0].Freshness)
	require.Equal(t, staleAt, facts.Msg.Facts[0].Fact.ObservedAt.AsTime())
	require.Equal(t, int32(1), facts.Msg.Facts[0].Fact.StaleAfterDays)
	require.Equal(t, "activation", facts.Msg.Facts[0].Fact.Dimension)
	require.Equal(t, "b_boundary", facts.Msg.Facts[1].Fact.Name)
	require.Equal(t, offerspb.FactFreshness_FACT_FRESHNESS_FRESH, facts.Msg.Facts[1].Freshness, "expiry equality remains fresh by the evaluator's strict-after rule")
	require.Equal(t, "c_bad_clock", facts.Msg.Facts[2].Fact.Name)
	require.Equal(t, offerspb.FactFreshness_FACT_FRESHNESS_UNKNOWN, facts.Msg.Facts[2].Freshness)

	triggers, err := s.ListTriggers(ctx, connect.NewRequest(&offerspb.ListTriggersRequest{}))
	require.NoError(t, err)
	require.Len(t, triggers.Msg.Triggers, 1)
	require.Equal(t, node.Id, triggers.Msg.Triggers[0].NodeId)
	require.Equal(t, offerspb.TriggerComposition_ALL, triggers.Msg.Triggers[0].Composition)
	require.Equal(t, "activation_rate", triggers.Msg.Triggers[0].Clauses[0].FactName)
}

func TestListAuditReturnsTypedNewestFirstRowsWithBoundedNodeFiltering(t *testing.T) {
	now := time.Date(2026, 11, 24, 8, 0, 0, 0, time.UTC)
	s, ctx := testOfferService(t, schedule.NewFake(now))
	node, err := s.store.CreateNodeWithDetailsAudited(ctx, offerspb.NodeKind_OFFER, "Audit target", offerspb.Status_IDEA, "", "", offerspb.DeliverableClass_DELIVERABLE_CLASS_UNSPECIFIED, offerspb.FinishBar_FINISH_BAR_UNSPECIFIED, "creator", "initial reason")
	require.NoError(t, err)
	_, _, err = s.store.RenameNode(ctx, node.Id, "Renamed audit target", "reviewer", "reviewed rename")
	require.NoError(t, err)
	_, err = s.store.CreateNodeWithDetailsAudited(ctx, offerspb.NodeKind_OFFER, "Other node", offerspb.Status_IDEA, "", "", offerspb.DeliverableClass_DELIVERABLE_CLASS_UNSPECIFIED, offerspb.FinishBar_FINISH_BAR_UNSPECIFIED, "other", "other reason")
	require.NoError(t, err)

	page, err := s.ListAudit(ctx, connect.NewRequest(&offerspb.ListAuditRequest{NodeId: node.Id, Limit: 1}))
	require.NoError(t, err)
	require.Len(t, page.Msg.Records, 1)
	require.True(t, page.Msg.Truncated)
	require.Equal(t, node.Id, page.Msg.Records[0].NodeId)
	require.Equal(t, "reviewer", page.Msg.Records[0].Actor)
	require.Equal(t, offerspb.Status_IDEA, page.Msg.Records[0].NextStatus)
	require.Equal(t, "reviewed rename; prior_name=Audit target", page.Msg.Records[0].Reason)
	require.Equal(t, now, page.Msg.Records[0].CreatedAt.AsTime())

	all, err := s.ListAudit(ctx, connect.NewRequest(&offerspb.ListAuditRequest{NodeId: node.Id}))
	require.NoError(t, err)
	require.Len(t, all.Msg.Records, 2)
	require.False(t, all.Msg.Truncated)
	require.Equal(t, "creator", all.Msg.Records[1].Actor)

	for _, limit := range []int32{-1, 501} {
		_, err = s.ListAudit(ctx, connect.NewRequest(&offerspb.ListAuditRequest{Limit: limit}))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	}
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

func TestBoardKeepsMappedAccountVisibleWhenLedgerCannotBeResolved(t *testing.T) {
	clock := schedule.NewFake(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	s, ctx := testOfferService(t, clock)
	node, err := s.store.CreateNode(ctx, offerspb.NodeKind_OFFER, "Mapped but unavailable", offerspb.Status_ACTIVE, "", "acct-revenue")
	require.NoError(t, err)
	s.resolveLedgerURL = func(context.Context) (string, error) { return "", errors.New("money-ledger is restarting") }

	response, err := s.GetBoard(ctx, nil)
	require.NoError(t, err)
	require.Len(t, response.Msg.Entries, 1)
	entry := response.Msg.Entries[0]
	require.Equal(t, node.Id, entry.NodeId)
	require.False(t, entry.ActualsAvailable)
	require.Len(t, entry.Availability, 1)
	require.Equal(t, "money-ledger.actuals", entry.Availability[0].Source)
	require.Contains(t, entry.Availability[0].Reason, "dependency could not be resolved")
	require.NotContains(t, entry.Availability[0].Reason, "no ledger account mapping")
	require.Contains(t, entry.RankReason, "earnings unknown")
}

func TestDeclaredDefaultAliveBufferNeverFallsBackToLocalThreshold(t *testing.T) { // [REQ:INT-005]
	buffer, available := declaredDefaultAliveBuffer(nil)
	require.Zero(t, buffer)
	require.False(t, available)

	buffer, available = declaredDefaultAliveBuffer([]*ledgerpb.GoalVerdict{
		{Goal: &ledgerpb.Goal{Name: "default-alive", BufferMultiple: 1.4}},
	})
	require.True(t, available)
	require.Equal(t, 1.4, buffer)
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
