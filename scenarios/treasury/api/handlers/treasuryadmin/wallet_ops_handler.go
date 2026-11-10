package treasuryadmin

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	fundingpb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/funding"
	"treasury/internal/walletops"
)

// HeaderWithdrawalMandateIdentityToken carries the agent principal scoped by a
// withdrawal mandate. Operator authentication remains the admin realm; the
// separate header avoids mixing the standard agent and operator credentials.
const HeaderWithdrawalMandateIdentityToken = "X-Treasury-Mandate-Identity-Token"

func (h *connectHandler) ProvisionBudgetWallet(ctx context.Context, request *connect.Request[fundingpb.ProvisionBudgetWalletRequest]) (*connect.Response[fundingpb.BudgetWalletResponse], error) {
	operator, err := h.operatorIdentity(ctx, request)
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
	return connect.NewResponse(&fundingpb.BudgetWalletResponse{Wallet: value}), nil
}

func (h *connectHandler) GetBudgetWallet(ctx context.Context, request *connect.Request[fundingpb.GetBudgetWalletRequest]) (*connect.Response[fundingpb.BudgetWalletResponse], error) {
	operator, err := h.operatorIdentity(ctx, request)
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
	return connect.NewResponse(&fundingpb.BudgetWalletResponse{Wallet: value}), nil
}

func (h *connectHandler) RevokeMandateSession(ctx context.Context, request *connect.Request[fundingpb.RevokeMandateSessionRequest]) (*connect.Response[fundingpb.RevokeMandateSessionResponse], error) {
	operator, err := h.operatorIdentity(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.walletOps == nil {
		return nil, walletOpsUnavailable()
	}
	if request == nil || request.Msg == nil || request.Msg.GetMandateId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("mandate ID is required"))
	}
	if err = h.walletOps.RevokeMandateSession(ctx, operator.Subject, request.Msg.GetMandateId()); err != nil {
		return nil, walletOpsError(err)
	}
	return connect.NewResponse(&fundingpb.RevokeMandateSessionResponse{Revoked: true}), nil
}

func (h *connectHandler) CreateFundingIntent(ctx context.Context, request *connect.Request[fundingpb.CreateFundingIntentRequest]) (*connect.Response[fundingpb.FundingIntentResponse], error) {
	operator, err := h.operatorIdentity(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.walletOps == nil {
		return nil, walletOpsUnavailable()
	}
	value, err := h.walletOps.CreateFundingIntent(ctx, operator.Subject, request.Msg)
	if err != nil {
		return nil, walletOpsError(err)
	}
	return connect.NewResponse(&fundingpb.FundingIntentResponse{Intent: value}), nil
}

func (h *connectHandler) GetFundingIntent(ctx context.Context, request *connect.Request[fundingpb.FundingIntentRequest]) (*connect.Response[fundingpb.FundingIntentResponse], error) {
	operator, err := h.operatorIdentity(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.walletOps == nil {
		return nil, walletOpsUnavailable()
	}
	value, err := h.walletOps.GetFundingIntent(ctx, operator.Subject, request.Msg.GetId())
	if err != nil {
		return nil, walletOpsError(err)
	}
	return connect.NewResponse(&fundingpb.FundingIntentResponse{Intent: value}), nil
}

func (h *connectHandler) ConfirmFundingTransfer(ctx context.Context, request *connect.Request[fundingpb.ConfirmFundingTransferRequest]) (*connect.Response[fundingpb.FundingIntentResponse], error) {
	operator, err := h.operatorIdentity(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.walletOps == nil {
		return nil, walletOpsUnavailable()
	}
	value, err := h.walletOps.ConfirmFundingTransfer(ctx, operator.Subject, request.Msg)
	if err != nil {
		return nil, walletOpsError(err)
	}
	return connect.NewResponse(&fundingpb.FundingIntentResponse{Intent: value}), nil
}

func (h *connectHandler) ListFundingIntents(ctx context.Context, request *connect.Request[fundingpb.ListFundingIntentsRequest]) (*connect.Response[fundingpb.ListFundingIntentsResponse], error) {
	operator, err := h.operatorIdentity(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.walletOps == nil {
		return nil, walletOpsUnavailable()
	}
	values, err := h.walletOps.ListFundingIntents(ctx, operator.Subject, request.Msg.GetBudgetId())
	if err != nil {
		return nil, walletOpsError(err)
	}
	return connect.NewResponse(&fundingpb.ListFundingIntentsResponse{Intents: values}), nil
}

func (h *connectHandler) SetWithdrawalDestination(ctx context.Context, request *connect.Request[fundingpb.SetWithdrawalDestinationRequest]) (*connect.Response[fundingpb.WithdrawalDestinationResponse], error) {
	operator, err := h.operatorIdentity(ctx, request)
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
	return connect.NewResponse(&fundingpb.WithdrawalDestinationResponse{Destination: value}), nil
}

func (h *connectHandler) PreviewWithdrawal(ctx context.Context, request *connect.Request[fundingpb.PreviewWithdrawalRequest]) (*connect.Response[fundingpb.WithdrawalQuoteResponse], error) {
	operator, err := h.operatorIdentity(ctx, request)
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
	return connect.NewResponse(&fundingpb.WithdrawalQuoteResponse{Quote: value}), nil
}

func (h *connectHandler) ExecuteWithdrawal(ctx context.Context, request *connect.Request[fundingpb.ExecuteWithdrawalRequest]) (*connect.Response[fundingpb.WithdrawalResponse], error) {
	operator, err := h.operatorIdentity(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.walletOps == nil {
		return nil, walletOpsUnavailable()
	}
	value, err := h.walletOps.ExecuteWithdrawal(ctx, operator.Subject, request.Header().Get(HeaderWithdrawalMandateIdentityToken), request.Msg)
	if err != nil {
		return nil, walletOpsError(err)
	}
	return connect.NewResponse(&fundingpb.WithdrawalResponse{Withdrawal: value}), nil
}

func (h *connectHandler) GetWithdrawal(ctx context.Context, request *connect.Request[fundingpb.GetWithdrawalRequest]) (*connect.Response[fundingpb.WithdrawalResponse], error) {
	operator, err := h.operatorIdentity(ctx, request)
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
	return connect.NewResponse(&fundingpb.WithdrawalResponse{Withdrawal: value}), nil
}

func walletOpsUnavailable() error {
	return connect.NewError(connect.CodeFailedPrecondition, errors.New("budget wallet operations are unavailable"))
}

func walletOpsError(err error) error {
	code := connect.CodeInternal
	switch {
	case errors.Is(err, walletops.ErrInvalid):
		code = connect.CodeInvalidArgument
	case errors.Is(err, walletops.ErrNotFound):
		code = connect.CodeNotFound
	case errors.Is(err, walletops.ErrConflict):
		code = connect.CodeAlreadyExists
	}
	return connect.NewError(code, err)
}
