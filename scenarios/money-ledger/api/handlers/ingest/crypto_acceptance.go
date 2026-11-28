package ingest

import (
	"context"
	"database/sql"
	"errors"

	"money-ledger/internal/access"
	"money-ledger/internal/ledger"

	"connectrpc.com/connect"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
)

func (s *Service) AcceptCryptoReceipt(ctx context.Context, req *connect.Request[pb.AcceptCryptoReceiptRequest]) (*connect.Response[pb.AcceptCryptoReceiptResponse], error) {
	out, err := s.store.AcceptCryptoReceipt(ctx, req.Msg, string(access.ActorFrom(ctx)))
	if err != nil {
		return nil, cryptoError(err)
	}
	return connect.NewResponse(out), nil
}

func (s *Service) GetCryptoReceipt(ctx context.Context, req *connect.Request[pb.GetCryptoReceiptRequest]) (*connect.Response[pb.GetCryptoReceiptResponse], error) {
	out, err := s.store.GetCryptoReceipt(ctx, req.Msg.GetId())
	if err != nil {
		return nil, cryptoError(err)
	}
	return connect.NewResponse(&pb.GetCryptoReceiptResponse{Receipt: out}), nil
}

func (s *Service) ListCryptoReceipts(ctx context.Context, req *connect.Request[pb.ListCryptoReceiptsRequest]) (*connect.Response[pb.ListCryptoReceiptsResponse], error) {
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
