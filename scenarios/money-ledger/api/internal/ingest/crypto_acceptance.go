package ingest

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	"money-ledger/internal/ledger"

	pb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	"google.golang.org/protobuf/proto"
)

const baseUSDCAsset = "eip155:8453/erc20:0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"

func boundedReference(value string) bool {
	return strings.TrimSpace(value) == value && value != "" && len(value) <= 1024 && !strings.ContainsAny(value, "\r\n\x00")
}

// AcceptCryptoReceipt normalizes Treasury's adapter event before the journal
// computes its signed effect. Only the authenticated service handler calls it.
func (s *Store) AcceptCryptoReceipt(ctx context.Context, request *pb.AcceptCryptoReceiptRequest, actor string) (*pb.AcceptCryptoReceiptResponse, error) {
	if request == nil || request.Observation == nil || request.Source == nil || request.Valuation == nil || request.Attribution == nil {
		return nil, errors.New("observation, Treasury source, explicit valuation and payer attribution are required")
	}
	in := proto.Clone(request).(*pb.AcceptCryptoReceiptRequest)
	o := in.Observation
	preview, err := s.PreviewCryptoObservation(ctx, o)
	if err != nil {
		return nil, err
	}
	if preview.AssetId != baseUSDCAsset || o.Decimals != 6 || o.LegKind != pb.CryptoLegKind_CRYPTO_LEG_KIND_DEPOSIT {
		return nil, errors.New("the Treasury adapter accepts native USDC deposits on Base with six decimals")
	}
	if o.BookId == "" || o.AccountId == "" || o.AtomicAmount == "0" {
		return nil, errors.New("a nonzero deposit and its ledger book/account are required")
	}
	if in.Action < pb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_ORIGINAL || in.Action > pb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_CORRECTION {
		return nil, errors.New("an explicit crypto receipt action is required")
	}
	if in.Action == pb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_REMOVAL {
		if o.State != pb.CryptoObservationState_CRYPTO_OBSERVATION_STATE_REMOVED {
			return nil, errors.New("removal requires removed chain evidence")
		}
	} else if o.State != pb.CryptoObservationState_CRYPTO_OBSERVATION_STATE_FINALIZED {
		return nil, errors.New("acceptance requires finalized chain evidence")
	}
	for _, value := range []string{in.Source.CheckId, in.Source.WalletId, in.Source.OperatorIdentity, in.Source.SourceReference, in.Valuation.BasisReference} {
		if !boundedReference(value) {
			return nil, errors.New("bounded Treasury identifiers and valuation basis reference are required")
		}
	}
	if in.OccurredAt == nil || in.OccurredAt.CheckValid() != nil || in.Valuation.ValuedAt == nil || in.Valuation.ValuedAt.CheckValid() != nil {
		return nil, errors.New("valid occurrence and valuation timestamps are required")
	}
	if in.Valuation.AmountMinor < 0 || len(in.Valuation.Currency) != 3 || strings.Trim(in.Valuation.Currency, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") != "" {
		return nil, errors.New("valuation requires a nonnegative minor amount and uppercase currency")
	}
	switch in.Valuation.Basis {
	case sharedpb.Basis_BASIS_AUTHORITATIVE, sharedpb.Basis_BASIS_DERIVED, sharedpb.Basis_BASIS_OPERATOR_ASSERTED:
	default:
		return nil, errors.New("valuation requires an explicit authoritative, derived or operator-asserted basis")
	}
	if !evmAddress.MatchString(in.Attribution.PayerAddress) {
		return nil, errors.New("the payer's chain address is required")
	}
	if in.Attribution.PayerIdentity != "" && !boundedReference(in.Attribution.PayerIdentity) {
		return nil, errors.New("payer identity must be a bounded reference")
	}
	lotID, lotOwner := in.Attribution.CreditLotId, in.Attribution.CreditLotOwner
	if (lotID == "") != (lotOwner == "") {
		return nil, errors.New("credit lot id and owner must be supplied together")
	}
	if lotID != "" && (in.Attribution.PayerIdentity == "" || !boundedReference(lotID) || !boundedReference(lotOwner)) {
		return nil, errors.New("a pre-existing credit lot association requires bounded payer, lot and owner references")
	}
	if in.Action == pb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_ORIGINAL {
		if in.OriginalReceiptId != "" || in.PreviousReceiptId != "" || in.CorrectionReason != "" {
			return nil, errors.New("an original cannot reference another acceptance or a correction reason")
		}
	} else {
		if !boundedReference(in.OriginalReceiptId) || !boundedReference(in.PreviousReceiptId) {
			return nil, errors.New("root accepted original and immediate accepted predecessor are required")
		}
		if !boundedReference(in.CorrectionReason) {
			return nil, errors.New("an evidence revision requires a bounded reason")
		}
	}
	o.AssetId, o.Address = preview.AssetId, preview.Address
	o.TransactionHash, o.BlockHash = strings.ToLower(o.TransactionHash), strings.ToLower(o.BlockHash)
	in.Attribution.PayerAddress = strings.ToLower(in.Attribution.PayerAddress)
	stable := proto.Clone(in).(*pb.AcceptCryptoReceiptRequest)
	// A new verification check does not turn the same economic fact into money.
	stable.Source.CheckId, stable.Source.SourceReference, stable.Observation.SourceReference = "", "", ""
	encoded, err := (proto.MarshalOptions{Deterministic: true}).Marshal(stable)
	if err != nil {
		return nil, err
	}
	return s.journal.AcceptCryptoReceipt(ctx, ledger.CryptoReceiptCandidate{Request: in, OccurrenceKey: preview.OccurrenceKey, InclusionRevision: preview.InclusionRevision, SemanticHash: fmt.Sprintf("%x", sha256.Sum256(encoded))}, actor)
}

func (s *Store) GetCryptoReceipt(ctx context.Context, id string) (*pb.AcceptedCryptoReceipt, error) {
	return s.journal.GetCryptoReceipt(ctx, id)
}

func (s *Store) ListCryptoReceipts(ctx context.Context, in *pb.ListCryptoReceiptsRequest) (*pb.ListCryptoReceiptsResponse, error) {
	return s.journal.ListCryptoReceipts(ctx, in)
}
