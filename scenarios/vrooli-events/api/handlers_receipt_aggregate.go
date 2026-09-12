package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	domainconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain/domainconnect"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
)

type receiptAggregateService struct {
	domainconnect.UnimplementedReceiptAggregateServiceHandler
	store store.ReceiptAggregateStore
	now   func() time.Time
}

func newReceiptAggregateService(events store.ReceiptAggregateStore, now func() time.Time) *receiptAggregateService {
	if now == nil {
		now = time.Now
	}
	return &receiptAggregateService{store: events, now: now}
}

func (s *receiptAggregateService) AggregateReceipts(ctx context.Context, req *connect.Request[domain.ReceiptAggregateRequest]) (*connect.Response[domain.ReceiptAggregateResponse], error) {
	if s.store == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("receipt aggregate store is unavailable"))
	}
	now := s.now().UTC()
	since := now.Add(-24 * time.Hour)
	until := now
	if req.Msg.GetSince() != nil {
		since = req.Msg.GetSince().AsTime().UTC()
	}
	if req.Msg.GetUntil() != nil {
		until = req.Msg.GetUntil().AsTime().UTC()
	}
	if !since.Before(until) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("since must be before until"))
	}
	aggregates, err := s.store.AggregateReceipts(ctx, store.ReceiptAggregateFilter{
		Since: since, Until: until, TargetScenario: req.Msg.GetTargetScenario(), Operation: req.Msg.GetOperation(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&domain.ReceiptAggregateResponse{
		Aggregates:  aggregates,
		WindowSince: since.Format(time.RFC3339Nano),
		WindowUntil: until.Format(time.RFC3339Nano),
	}), nil
}

func receiptAggregateHandler(events store.ReceiptAggregateStore) (string, http.Handler) {
	return domainconnect.NewReceiptAggregateServiceHandler(newReceiptAggregateService(events, time.Now))
}
