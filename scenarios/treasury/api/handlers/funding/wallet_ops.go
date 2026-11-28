package funding

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	fundingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/funding"
	"google.golang.org/protobuf/types/known/timestamppb"
	"treasury/internal/baseusdc"
	"treasury/internal/budget"
	"treasury/internal/identity"
	receiptfundingdomain "treasury/internal/receiptfunding"
	"treasury/internal/walletops"
)

func (h *connectHandler) SubmitReceiptFunding(ctx context.Context, request *connect.Request[fundingv1.SubmitReceiptFundingRequest]) (*connect.Response[fundingv1.ReceiptFundingResponse], error) {
	operator, err := h.authorize(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.receipts == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("receipt funding service unavailable"))
	}
	m := request.Msg
	value, err := h.receipts.Submit(ctx, receiptfundingdomain.SubmitInput{OperatorIdentity: operator.Subject,
		WalletID: m.GetWalletId(), TransactionHash: m.GetTransactionHash(), CheckID: m.GetCheckId(), LogIndex: m.GetLogIndex(),
		BudgetID: m.GetBudgetId(), LedgerBookID: m.GetLedgerBookId(), LedgerAccountID: m.GetLedgerAccountId(), Action: m.GetAction(),
		OriginalReceiptID: m.GetOriginalReceiptId(), PreviousReceiptID: m.GetPreviousReceiptId(), Reason: m.GetReason(),
		Valuation: m.GetValuation(), Attribution: m.GetAttribution(),
	})
	if err != nil {
		return nil, fundingError(err)
	}
	_ = h.receipts.DrainPending(ctx)
	if current, err := h.receipts.Get(ctx, value.ID, operator.Subject); err == nil {
		value = current
	}
	return connect.NewResponse(&fundingv1.ReceiptFundingResponse{Delivery: fundingDeliveryToProto(value)}), nil
}

func (h *connectHandler) GetReceiptFunding(ctx context.Context, request *connect.Request[fundingv1.GetReceiptFundingRequest]) (*connect.Response[fundingv1.ReceiptFundingResponse], error) {
	operator, err := h.authorize(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.receipts == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("receipt funding service unavailable"))
	}
	value, err := h.receipts.Get(ctx, request.Msg.GetId(), operator.Subject)
	if err != nil {
		return nil, fundingError(err)
	}
	return connect.NewResponse(&fundingv1.ReceiptFundingResponse{Delivery: fundingDeliveryToProto(value)}), nil
}

func fundingDeliveryToProto(value receiptfundingdomain.Delivery) *fundingv1.ReceiptFundingDelivery {
	state := fundingv1.ReceiptFundingState_RECEIPT_FUNDING_STATE_QUEUED
	if value.Status == "accepted" {
		state = fundingv1.ReceiptFundingState_RECEIPT_FUNDING_STATE_ACCEPTED
	}
	out := &fundingv1.ReceiptFundingDelivery{Id: value.ID, WalletId: value.WalletID, BudgetId: value.BudgetID, SourceCheckId: value.SourceCheckID, State: state, Attempts: uint32(value.Attempts), LastError: value.LastError, CreatedAt: timestamppb.New(value.CreatedAt), Receipt: value.Receipt, CreditLotId: value.CreditLotID, CreditLotOwner: value.CreditLotOwner, CreditLotReceiptId: value.CreditLotReceiptID}
	if !value.AcceptedAt.IsZero() {
		out.AcceptedAt = timestamppb.New(value.AcceptedAt)
	}
	return out
}

func (h *connectHandler) ProvisionBudgetWallet(ctx context.Context, request *connect.Request[fundingv1.ProvisionBudgetWalletRequest]) (*connect.Response[fundingv1.BudgetWalletResponse], error) {
	operator, err := h.authorize(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.walletOps == nil {
		return nil, walletOpsUnavailable()
	}
	value, err := h.walletOps.Provision(ctx, operator.Subject, request.Msg)
	if err != nil {
		return nil, walletOpsError(err)
	}
	return connect.NewResponse(&fundingv1.BudgetWalletResponse{Wallet: value}), nil
}

func (h *connectHandler) GetBudgetWallet(ctx context.Context, request *connect.Request[fundingv1.GetBudgetWalletRequest]) (*connect.Response[fundingv1.BudgetWalletResponse], error) {
	operator, err := h.authorize(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.walletOps == nil {
		return nil, walletOpsUnavailable()
	}
	value, err := h.walletOps.GetForOperator(ctx, operator.Subject, request.Msg.GetBudgetId(), request.Msg.GetMandateId())
	if err != nil {
		return nil, walletOpsError(err)
	}
	return connect.NewResponse(&fundingv1.BudgetWalletResponse{Wallet: value}), nil
}

func (h *connectHandler) SetWithdrawalDestination(ctx context.Context, request *connect.Request[fundingv1.SetWithdrawalDestinationRequest]) (*connect.Response[fundingv1.WithdrawalDestinationResponse], error) {
	operator, err := h.authorize(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.walletOps == nil {
		return nil, walletOpsUnavailable()
	}
	value, err := h.walletOps.SetWithdrawalDestination(ctx, operator.Subject, request.Msg)
	if err != nil {
		return nil, walletOpsError(err)
	}
	return connect.NewResponse(&fundingv1.WithdrawalDestinationResponse{Destination: value}), nil
}

func (h *connectHandler) PreviewWithdrawal(ctx context.Context, request *connect.Request[fundingv1.PreviewWithdrawalRequest]) (*connect.Response[fundingv1.WithdrawalQuoteResponse], error) {
	operator, err := h.authorize(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.walletOps == nil {
		return nil, walletOpsUnavailable()
	}
	value, err := h.walletOps.PreviewWithdrawal(ctx, operator.Subject, request.Msg)
	if err != nil {
		return nil, walletOpsError(err)
	}
	return connect.NewResponse(&fundingv1.WithdrawalQuoteResponse{Quote: value}), nil
}

func (h *connectHandler) ExecuteWithdrawal(ctx context.Context, request *connect.Request[fundingv1.ExecuteWithdrawalRequest]) (*connect.Response[fundingv1.WithdrawalResponse], error) {
	operator, err := h.authorize(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.walletOps == nil {
		return nil, walletOpsUnavailable()
	}
	value, err := h.walletOps.ExecuteWithdrawal(ctx, operator.Subject, request.Msg)
	if err != nil {
		return nil, walletOpsError(err)
	}
	return connect.NewResponse(&fundingv1.WithdrawalResponse{Withdrawal: value}), nil
}

func (h *connectHandler) GetWithdrawal(ctx context.Context, request *connect.Request[fundingv1.GetWithdrawalRequest]) (*connect.Response[fundingv1.WithdrawalResponse], error) {
	operator, err := h.authorize(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.walletOps == nil {
		return nil, walletOpsUnavailable()
	}
	value, err := h.walletOps.GetWithdrawal(ctx, operator.Subject, request.Msg.GetId())
	if err != nil {
		return nil, walletOpsError(err)
	}
	return connect.NewResponse(&fundingv1.WithdrawalResponse{Withdrawal: value}), nil
}

func (h *agentWalletHandler) GetBudgetWallet(ctx context.Context, request *connect.Request[fundingv1.GetBudgetWalletRequest]) (*connect.Response[fundingv1.BudgetWalletResponse], error) {
	if request == nil || request.Msg == nil || strings.TrimSpace(request.Msg.GetBudgetId()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("budget ID is required"))
	}
	if h.wallets == nil || h.verifier == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("budget wallet reporting is unavailable"))
	}
	claims, err := identity.VerifyRequest(ctx, h.verifier, request.Header())
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	if mandateID := request.Msg.GetMandateId(); mandateID != "" {
		if publicKey := strings.TrimSpace(request.Header().Get(walletops.HeaderAgentSessionPublicKey)); publicKey != "" {
			if err = h.wallets.BindAgentSessionPublicKey(ctx, request.Msg.GetBudgetId(), mandateID, claims, publicKey); err != nil {
				return nil, connect.NewError(connect.CodePermissionDenied, err)
			}
		}
	}
	wallet, err := h.wallets.GetForAgent(ctx, request.Msg.GetBudgetId(), request.Msg.GetMandateId(), claims)
	if err != nil {
		switch {
		case errors.Is(err, identity.ErrUnverifiable):
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		case errors.Is(err, walletops.ErrNotFound):
			return nil, connect.NewError(connect.CodeNotFound, err)
		default:
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	return connect.NewResponse(&fundingv1.BudgetWalletResponse{Wallet: wallet}), nil
}

func walletOpsUnavailable() error {
	return connect.NewError(connect.CodeFailedPrecondition, errors.New("funding wallet operations are unavailable"))
}

func walletOpsError(err error) error {
	return fundingError(err)
}

func fundingError(err error) error {
	code := connect.CodeInternal
	switch {
	case errors.Is(err, walletops.ErrInvalid):
		code = connect.CodeInvalidArgument
	case errors.Is(err, walletops.ErrNotFound):
		code = connect.CodeNotFound
	case errors.Is(err, walletops.ErrConflict):
		code = connect.CodeAlreadyExists
	case errors.Is(err, receiptfundingdomain.ErrNotFound), errors.Is(err, baseusdc.ErrPriorCheckNotFound), errors.Is(err, budget.ErrNotFound):
		code = connect.CodeNotFound
	case errors.Is(err, receiptfundingdomain.ErrInvalid), errors.Is(err, budget.ErrInvalid):
		code = connect.CodeInvalidArgument
	case errors.Is(err, receiptfundingdomain.ErrConflict):
		code = connect.CodeAlreadyExists
	}
	return connect.NewError(code, err)
}
