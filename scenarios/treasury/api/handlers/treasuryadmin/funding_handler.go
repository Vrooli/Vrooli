package treasuryadmin

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	fundingpb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/funding"
	"google.golang.org/protobuf/types/known/timestamppb"
	"treasury/internal/baseusdc"
	"treasury/internal/budget"
	"treasury/internal/receiptfunding"
)

func (h *connectHandler) SubmitReceiptFunding(ctx context.Context, request *connect.Request[fundingpb.SubmitReceiptFundingRequest]) (*connect.Response[fundingpb.ReceiptFundingResponse], error) {
	operator, err := h.operatorIdentity(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.funding == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("receipt funding service unavailable"))
	}
	m := request.Msg
	value, err := h.funding.Submit(ctx, receiptfunding.SubmitInput{OperatorIdentity: operator.Subject,
		WalletID: m.GetWalletId(), TransactionHash: m.GetTransactionHash(), CheckID: m.GetCheckId(), LogIndex: m.GetLogIndex(),
		BudgetID: m.GetBudgetId(), LedgerBookID: m.GetLedgerBookId(), LedgerAccountID: m.GetLedgerAccountId(), Action: m.GetAction(),
		OriginalReceiptID: m.GetOriginalReceiptId(), PreviousReceiptID: m.GetPreviousReceiptId(), Reason: m.GetReason(),
		Valuation: m.GetValuation(), Attribution: m.GetAttribution(),
	})
	if err != nil {
		return nil, fundingError(err)
	}
	// The durable queue is the result. Dependency outages remain visible and
	// retryable; no budget credit is created before downstream acceptance.
	_ = h.funding.DrainPending(ctx)
	if current, err := h.funding.Get(ctx, value.ID, operator.Subject); err == nil {
		value = current
	}
	return connect.NewResponse(&fundingpb.ReceiptFundingResponse{Delivery: fundingDeliveryToProto(value)}), nil
}

func (h *connectHandler) GetReceiptFunding(ctx context.Context, request *connect.Request[fundingpb.GetReceiptFundingRequest]) (*connect.Response[fundingpb.ReceiptFundingResponse], error) {
	operator, err := h.operatorIdentity(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.funding == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("receipt funding service unavailable"))
	}
	value, err := h.funding.Get(ctx, request.Msg.GetId(), operator.Subject)
	if err != nil {
		return nil, fundingError(err)
	}
	return connect.NewResponse(&fundingpb.ReceiptFundingResponse{Delivery: fundingDeliveryToProto(value)}), nil
}

func fundingError(err error) error {
	code := connect.CodeInternal
	switch {
	case errors.Is(err, receiptfunding.ErrNotFound), errors.Is(err, baseusdc.ErrPriorCheckNotFound), errors.Is(err, budget.ErrNotFound):
		code = connect.CodeNotFound
	case errors.Is(err, receiptfunding.ErrInvalid), errors.Is(err, budget.ErrInvalid):
		code = connect.CodeInvalidArgument
	case errors.Is(err, receiptfunding.ErrConflict):
		code = connect.CodeAlreadyExists
	}
	return connect.NewError(code, err)
}

func fundingDeliveryToProto(value receiptfunding.Delivery) *fundingpb.ReceiptFundingDelivery {
	state := fundingpb.ReceiptFundingState_RECEIPT_FUNDING_STATE_QUEUED
	if value.Status == "accepted" {
		state = fundingpb.ReceiptFundingState_RECEIPT_FUNDING_STATE_ACCEPTED
	}
	out := &fundingpb.ReceiptFundingDelivery{Id: value.ID, WalletId: value.WalletID, BudgetId: value.BudgetID, SourceCheckId: value.SourceCheckID, State: state, Attempts: uint32(value.Attempts), LastError: value.LastError, CreatedAt: timestamppb.New(value.CreatedAt), Receipt: value.Receipt, CreditLotId: value.CreditLotID, CreditLotOwner: value.CreditLotOwner, CreditLotReceiptId: value.CreditLotReceiptID}
	if !value.AcceptedAt.IsZero() {
		out.AcceptedAt = timestamppb.New(value.AcceptedAt)
	}
	return out
}
