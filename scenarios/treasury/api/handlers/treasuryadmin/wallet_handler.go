package treasuryadmin

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	authv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
	walletv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/wallet"
	"google.golang.org/protobuf/types/known/timestamppb"
	"treasury/internal/baseusdc"
	"treasury/internal/wallet"
)

func walletToProto(w wallet.Wallet) *walletv1.Wallet {
	return &walletv1.Wallet{Id: w.ID, BookId: w.BookID, OperatorIdentity: w.OperatorIdentity, Network: w.Network, Address: w.Address, PublicKey: w.PublicKey, CustodyReference: w.CustodyReference, CreatedAt: timestamppb.New(w.CreatedAt), RecoveryStatus: w.RecoveryStatus, RecoverySha256: w.RecoverySHA256, IndependentRecoveryVerified: false}
}

func walletError(err error) error {
	if err == nil {
		return nil
	}
	code := connect.CodeFailedPrecondition
	switch {
	case errors.Is(err, wallet.ErrInvalid):
		code = connect.CodeInvalidArgument
	case errors.Is(err, wallet.ErrMismatch):
		code = connect.CodePermissionDenied
	case errors.Is(err, wallet.ErrNotFound):
		code = connect.CodeNotFound
	}
	return connect.NewError(code, err)
}

func (h *connectHandler) CreateResearchWallet(ctx context.Context, request *connect.Request[authv1.CreateResearchWalletRequest]) (*connect.Response[authv1.ResearchWalletResponse], error) {
	operator, err := h.operatorIdentity(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.wallets == nil {
		return nil, walletError(wallet.ErrUnavailable)
	}
	w, err := h.wallets.Create(ctx, wallet.CreateInput{ID: request.Msg.GetId(), BookID: request.Msg.GetBookId(), Network: request.Msg.GetNetwork(), OperatorIdentity: operator.Subject})
	if err != nil {
		return nil, walletError(err)
	}
	return connect.NewResponse(&authv1.ResearchWalletResponse{Wallet: walletToProto(w)}), nil
}

func (h *connectHandler) GetResearchWallet(ctx context.Context, request *connect.Request[authv1.ResearchWalletRequest]) (*connect.Response[authv1.ResearchWalletResponse], error) {
	operator, err := h.operatorIdentity(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.wallets == nil {
		return nil, walletError(wallet.ErrUnavailable)
	}
	w, err := h.wallets.Get(ctx, request.Msg.GetId(), request.Msg.GetNetwork(), operator.Subject)
	if err != nil {
		return nil, walletError(err)
	}
	return connect.NewResponse(&authv1.ResearchWalletResponse{Wallet: walletToProto(w)}), nil
}

func (h *connectHandler) ProveResearchWallet(ctx context.Context, request *connect.Request[authv1.ProveResearchWalletRequest]) (*connect.Response[authv1.ProveResearchWalletResponse], error) {
	operator, err := h.operatorIdentity(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.wallets == nil {
		return nil, walletError(wallet.ErrUnavailable)
	}
	p, err := h.wallets.Prove(ctx, request.Msg.GetId(), request.Msg.GetNetwork(), operator.Subject, request.Msg.GetNonce())
	if err != nil {
		return nil, walletError(err)
	}
	return connect.NewResponse(&authv1.ProveResearchWalletResponse{Proof: &walletv1.ControlProof{Wallet: walletToProto(p.Wallet), Message: p.Message, Signature: p.Signature, Scheme: p.Scheme, Verified: p.Verified}}), nil
}

func (h *connectHandler) VerifyBaseUSDCReceipt(ctx context.Context, request *connect.Request[authv1.VerifyBaseUSDCReceiptRequest]) (*connect.Response[authv1.VerifyBaseUSDCReceiptResponse], error) {
	operator, err := h.operatorIdentity(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.wallets == nil {
		return nil, walletError(wallet.ErrUnavailable)
	}
	if h.baseUSDC == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("Base receipt verifier unavailable"))
	}
	if request.Msg.GetWalletId() == "" || request.Msg.GetTransactionHash() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("wallet ID and transaction hash are required"))
	}
	w, err := h.wallets.Get(ctx, request.Msg.GetWalletId(), baseusdc.Network, operator.Subject)
	if err != nil {
		return nil, walletError(err)
	}
	check, err := h.baseUSDC.Verify(ctx, operator.Subject, w, request.Msg.GetTransactionHash())
	if err != nil {
		code := connect.CodeFailedPrecondition
		if errors.Is(err, baseusdc.ErrRPCFinality) || errors.Is(err, baseusdc.ErrRPCUnavailable) {
			code = connect.CodeUnavailable
		}
		if errors.Is(err, baseusdc.ErrPersist) {
			code = connect.CodeInternal
		}
		return nil, connect.NewError(code, err)
	}
	response := &authv1.VerifyBaseUSDCReceiptResponse{
		Transfers: baseUSDCObservationsToProto(check.Transfers), CheckId: check.ID,
		CheckOutcome: check.Outcome, ProviderId: check.ProviderID, CheckedAt: timestamppb.New(check.CheckedAt),
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) ListBaseUSDCReceiptChecks(ctx context.Context, request *connect.Request[authv1.ListBaseUSDCReceiptChecksRequest]) (*connect.Response[authv1.ListBaseUSDCReceiptChecksResponse], error) {
	operator, err := h.operatorIdentity(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.wallets == nil {
		return nil, walletError(wallet.ErrUnavailable)
	}
	if h.baseUSDC == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("Base receipt evidence service unavailable"))
	}
	if request.Msg.GetWalletId() == "" || request.Msg.GetTransactionHash() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("wallet ID and transaction hash are required"))
	}
	if _, err := h.wallets.Get(ctx, request.Msg.GetWalletId(), baseusdc.Network, operator.Subject); err != nil {
		return nil, walletError(err)
	}
	checks, err := h.baseUSDC.List(ctx, operator.Subject, request.Msg.GetWalletId(), request.Msg.GetTransactionHash(), int(request.Msg.GetLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	response := &authv1.ListBaseUSDCReceiptChecksResponse{Checks: make([]*walletv1.BaseUSDCReceiptCheck, 0, len(checks))}
	for _, check := range checks {
		response.Checks = append(response.Checks, &walletv1.BaseUSDCReceiptCheck{
			Id: check.ID, WalletId: check.WalletID, Network: check.Network,
			TransactionHash: check.TransactionHash, ProviderId: check.ProviderID,
			Outcome: check.Outcome, ErrorCode: check.ErrorCode,
			CheckedAt: timestamppb.New(check.CheckedAt), Transfers: baseUSDCObservationsToProto(check.Transfers),
		})
	}
	return connect.NewResponse(response), nil
}

func baseUSDCObservationsToProto(observations []baseusdc.Observation) []*walletv1.BaseUSDCTransferObservation {
	transfers := make([]*walletv1.BaseUSDCTransferObservation, 0, len(observations))
	for _, observation := range observations {
		transfers = append(transfers, &walletv1.BaseUSDCTransferObservation{
			EconomicKey: observation.EconomicKey, InclusionRevision: observation.InclusionRevision, WalletId: observation.WalletID,
			Network: observation.Network, AssetId: observation.AssetID, Decimals: observation.Decimals,
			From: observation.From, To: observation.To, AtomicAmount: observation.AtomicAmount,
			TransactionHash: observation.TransactionHash, LogIndex: observation.LogIndex,
			BlockNumber: observation.BlockNumber, BlockHash: observation.BlockHash,
			BlockTime: timestamppb.New(observation.BlockTime), Finality: observation.Finality,
			FinalizedHeadNumber: observation.FinalizedHeadNumber, FinalizedHeadHash: observation.FinalizedHeadHash,
			ProviderId: observation.ProviderID, ObservedAt: timestamppb.New(observation.ObservedAt),
		})
	}
	return transfers
}

func (h *connectHandler) QualifyResearchWalletRecovery(ctx context.Context, request *connect.Request[authv1.ResearchWalletRequest]) (*connect.Response[authv1.ResearchWalletResponse], error) {
	operator, err := h.operatorIdentity(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.wallets == nil {
		return nil, walletError(wallet.ErrUnavailable)
	}
	w, err := h.wallets.QualifyRecovery(ctx, request.Msg.GetId(), request.Msg.GetNetwork(), operator.Subject)
	if err != nil {
		return nil, walletError(err)
	}
	return connect.NewResponse(&authv1.ResearchWalletResponse{Wallet: walletToProto(w)}), nil
}

func (h *connectHandler) RecoverResearchWallet(ctx context.Context, request *connect.Request[authv1.ResearchWalletRequest]) (*connect.Response[authv1.ResearchWalletResponse], error) {
	operator, err := h.operatorIdentity(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.wallets == nil {
		return nil, walletError(wallet.ErrUnavailable)
	}
	w, err := h.wallets.Recover(ctx, request.Msg.GetId(), request.Msg.GetNetwork(), operator.Subject)
	if err != nil {
		return nil, walletError(err)
	}
	return connect.NewResponse(&authv1.ResearchWalletResponse{Wallet: walletToProto(w)}), nil
}
