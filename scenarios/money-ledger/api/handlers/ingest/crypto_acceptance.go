package ingest

import (
	"context"
	"database/sql"
	"errors"

	"money-ledger/internal/ledger"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
)

func requireTreasury(ctx context.Context) (identity.Principal, error) {
	principal, err := authn.RequireService(ctx)
	if err != nil {
		return identity.Principal{}, connect.NewError(connect.CodeUnauthenticated, err)
	}
	if principal.Subject != "treasury" || principal.Source != identity.SourceScenarioAuthenticator {
		return identity.Principal{}, connect.NewError(connect.CodePermissionDenied, errors.New("Treasury service identity required"))
	}
	if _, err := authn.RequireCapability(ctx, "money-ledger:write"); err != nil {
		return identity.Principal{}, connect.NewError(connect.CodePermissionDenied, err)
	}
	return principal, nil
}

func (s *Service) AcceptCryptoReceipt(ctx context.Context, req *connect.Request[pb.AcceptCryptoReceiptRequest]) (*connect.Response[pb.AcceptCryptoReceiptResponse], error) {
	principal, err := requireTreasury(ctx)
	if err != nil {
		return nil, err
	}
	out, err := s.store.AcceptCryptoReceipt(ctx, req.Msg, principal.Subject)
	if err != nil {
		return nil, cryptoError(err)
	}
	return connect.NewResponse(out), nil
}

func (s *Service) GetCryptoReceipt(ctx context.Context, req *connect.Request[pb.GetCryptoReceiptRequest]) (*connect.Response[pb.GetCryptoReceiptResponse], error) {
	if _, err := requireTreasury(ctx); err != nil {
		return nil, err
	}
	out, err := s.store.GetCryptoReceipt(ctx, req.Msg.GetId())
	if err != nil {
		return nil, cryptoError(err)
	}
	return connect.NewResponse(&pb.GetCryptoReceiptResponse{Receipt: out}), nil
}

func (s *Service) ListCryptoReceipts(ctx context.Context, req *connect.Request[pb.ListCryptoReceiptsRequest]) (*connect.Response[pb.ListCryptoReceiptsResponse], error) {
	if _, err := requireTreasury(ctx); err != nil {
		return nil, err
	}
	out, err := s.store.ListCryptoReceipts(ctx, req.Msg)
	if err != nil {
		return nil, cryptoError(err)
	}
	return connect.NewResponse(out), nil
}

func cryptoError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return notFound(err)
	}
	if errors.Is(err, ledger.ErrCryptoConflict) {
		return connect.NewError(connect.CodeAlreadyExists, err)
	}
	if errors.Is(err, ledger.ErrCryptoPredecessor) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return invalid(err)
}
