package offers

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/schedule"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	ledgerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger/ledger_v1connect"
	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
)

type posturePositionService struct {
	ledgerconnect.UnimplementedPositionServiceHandler
	position *ledgerpb.PositionResponse
}

func (s *posturePositionService) GetPosition(context.Context, *connect.Request[ledgerpb.PositionRequest]) (*connect.Response[ledgerpb.PositionResponse], error) {
	return connect.NewResponse(s.position), nil
}

func (*posturePositionService) ListGoals(context.Context, *connect.Request[ledgerpb.ListGoalsRequest]) (*connect.Response[ledgerpb.ListGoalsResponse], error) {
	return connect.NewResponse(&ledgerpb.ListGoalsResponse{}), nil
}

func TestBoardPostureTimestampTracksOldestLedgerInputAndRetainsPartialEvidence(t *testing.T) {
	now := time.Date(2026, 11, 21, 16, 0, 0, 0, time.UTC)
	clock := schedule.NewFake(now)
	service, ctx := testOfferService(t, clock)
	service.bookID = "operator-book"
	ledger := &posturePositionService{position: &ledgerpb.PositionResponse{
		CashMinor: 82000, RevenueMinor: 12000, BurnMinor: 4000, RunwayMonths: 20.5,
		Currency: "USD", RunwayAvailable: true,
		Inputs: []*ledgerpb.PositionInput{{Source: "bank-import", AgeSeconds: 95, Available: true}},
	}}
	_, handler := ledgerconnect.NewPositionServiceHandler(ledger)
	peer := httptest.NewServer(handler)
	t.Cleanup(peer.Close)
	service.resolveLedgerURL = func(context.Context) (string, error) { return peer.URL, nil }

	response, err := service.GetBoard(ctx, connect.NewRequest(&offerspb.ProjectionRequest{Projection: "ledger"}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.ObservedAt == nil || !response.Msg.ObservedAt.AsTime().Equal(now.Add(-95*time.Second)) {
		t.Fatalf("observed_at=%v, want oldest ledger input at %s", response.Msg.ObservedAt, now.Add(-95*time.Second))
	}
	if response.Msg.Position == nil || response.Msg.Position.Partial || response.Msg.PostureAgeSeconds != 95 {
		t.Fatalf("complete posture=%+v age=%d, want complete position with age 95s", response.Msg.Position, response.Msg.PostureAgeSeconds)
	}

	ledger.position = &ledgerpb.PositionResponse{
		CashMinor: 82000, RevenueMinor: 12000, BurnMinor: 4000, RunwayMonths: 20.5,
		Currency: "USD", Partial: true, RunwayAvailable: true,
		Inputs: []*ledgerpb.PositionInput{{Source: "bank-import", AgeSeconds: 95, Available: true}},
	}
	response, err = service.GetBoard(ctx, connect.NewRequest(&offerspb.ProjectionRequest{Projection: "ledger"}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.Position == nil || !response.Msg.Position.Partial {
		t.Fatalf("partial position=%+v, want Offer Desk to preserve the Money Ledger partial marker", response.Msg.Position)
	}
	if !response.Msg.ObservedAt.AsTime().Equal(now.Add(-95 * time.Second)) {
		t.Fatalf("partial observed_at=%v, want source input timestamp retained", response.Msg.ObservedAt)
	}
}
