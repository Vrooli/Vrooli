package treasuryadmin

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	authv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
	walletv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/wallet"
	"google.golang.org/protobuf/types/known/timestamppb"
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
