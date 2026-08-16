package offers

import (
	"context"
	"log"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"
	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	"google.golang.org/protobuf/types/known/timestamppb"
	"offer-desk/internal/catalog"
)

type drillTicker struct{ ch chan time.Time }

func (t *drillTicker) C() <-chan time.Time { return t.ch }
func (t *drillTicker) Stop()               {}
func (t *drillTicker) Reset(time.Duration) {}

type drillClock struct {
	now    time.Time
	ticker *drillTicker
}

func (c *drillClock) Now() time.Time { return c.now }
func (c *drillClock) NewTimer(d time.Duration) schedule.Timer {
	return schedule.System().NewTimer(d)
}

func (c *drillClock) NewTicker(time.Duration) schedule.Ticker {
	c.ticker = &drillTicker{ch: make(chan time.Time, 1)}
	return c.ticker
}
func (c *drillClock) Sleep(d time.Duration) { time.Sleep(d) }

func testOfferService(t *testing.T, clock *drillClock) (*Service, context.Context) {
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
	clock := &drillClock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	s, ctx := testOfferService(t, clock)
	n, err := s.store.CreateNode(ctx, offerspb.NodeKind_OFFER, "Scheduled offer", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	_, err = s.store.AddTrigger(ctx, &offerspb.Trigger{NodeId: n.Id, FactName: "activation_rate", Operator: ">=", Threshold: 0.5})
	require.NoError(t, err)
	_, err = s.store.Transition(ctx, n.Id, offerspb.Status_CANDIDATE, "operator")
	require.NoError(t, err)
	_, err = s.store.AddFact(ctx, &offerspb.Fact{Name: "activation_rate", Value: 0.8, ObservedAt: timestamppb.New(clock.now), StaleAfterDays: 10})
	require.NoError(t, err)
	s.startScheduler()
	clock.ticker.ch <- clock.now

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
	clock := &drillClock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
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
	clock := &drillClock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
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
	clock := &drillClock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
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
	require.Equal(t, "active and earning nothing", response.Msg.Entries[0].RankReason)
}
