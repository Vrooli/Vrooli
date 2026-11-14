package ledger

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TreasuryAdapterID is reserved for authenticated, atomically recorded receipts.
const TreasuryAdapterID = "treasury-base-usdc"

var (
	ErrCryptoConflict    = errors.New("crypto receipt conflicts with accepted evidence")
	ErrCryptoPredecessor = errors.New("crypto receipt must reference the current accepted predecessor")
)

// CryptoReceiptCandidate is normalized by the ingestion adapter. Exact token
// evidence belongs to the journal and commits with the same MoneyEvent writer.
type CryptoReceiptCandidate struct {
	Request                                        *pb.AcceptCryptoReceiptRequest
	OccurrenceKey, InclusionRevision, SemanticHash string
}

type cryptoRowReader interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func readCryptoReceipt(ctx context.Context, db cryptoRowReader, query string, args ...any) (*pb.AcceptedCryptoReceipt, error) {
	var payload []byte
	if err := db.QueryRowContext(ctx, query, args...).Scan(&payload); err != nil {
		return nil, err
	}
	out := &pb.AcceptedCryptoReceipt{}
	if err := proto.Unmarshal(payload, out); err != nil {
		return nil, fmt.Errorf("decode durable crypto receipt: %w", err)
	}
	return out, nil
}

// AcceptCryptoReceipt owns the atomic source submission, token evidence, journal
// posting and audit commit. The initial insert acquires SQLite's writer lock
// before any predecessor read, including across independent processes.
func (s *Store) AcceptCryptoReceipt(ctx context.Context, candidate CryptoReceiptCandidate, actor string) (*pb.AcceptCryptoReceiptResponse, error) {
	in := candidate.Request
	id := uuid.NewString()
	at := s.now().UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, `INSERT INTO crypto_submissions(check_id,occurrence_key,semantic_hash,source_reference,receipt_id,observed_at) VALUES(?,?,?,?,?,?) ON CONFLICT(check_id,occurrence_key) DO NOTHING`, in.Source.CheckId, candidate.OccurrenceKey, candidate.SemanticHash, in.Source.SourceReference, id, at.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if inserted == 0 {
		var hash, receiptID, sourceReference string
		if err := tx.QueryRowContext(ctx, `SELECT semantic_hash,receipt_id,source_reference FROM crypto_submissions WHERE check_id=? AND occurrence_key=?`, in.Source.CheckId, candidate.OccurrenceKey).Scan(&hash, &receiptID, &sourceReference); err != nil {
			return nil, err
		}
		if hash != candidate.SemanticHash || sourceReference != in.Source.SourceReference {
			return nil, ErrCryptoConflict
		}
		return finishCryptoReplay(ctx, tx, receiptID)
	}
	// One Treasury check may contain several transfer legs, but its owner,
	// wallet and transaction identity cannot be rebound by another submission.
	check, err := readCryptoReceipt(ctx, tx, `SELECT r.payload FROM crypto_submissions s JOIN crypto_receipts r ON r.id=s.receipt_id WHERE s.check_id=? LIMIT 1`, in.Source.CheckId)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if check != nil && (check.Source.WalletId != in.Source.WalletId || check.Source.OperatorIdentity != in.Source.OperatorIdentity || check.Source.SourceReference != in.Source.SourceReference || check.Observation.Network != in.Observation.Network || check.Observation.TransactionHash != in.Observation.TransactionHash) {
		return nil, fmt.Errorf("%w: Treasury check identity cannot be rebound", ErrCryptoConflict)
	}
	var replayID string
	err = tx.QueryRowContext(ctx, `SELECT id FROM crypto_receipts WHERE occurrence_key=? AND semantic_hash=? ORDER BY sequence DESC LIMIT 1`, candidate.OccurrenceKey, candidate.SemanticHash).Scan(&replayID)
	if err == nil {
		if _, err := tx.ExecContext(ctx, `UPDATE crypto_submissions SET receipt_id=? WHERE check_id=? AND occurrence_key=?`, replayID, in.Source.CheckId, candidate.OccurrenceKey); err != nil {
			return nil, err
		}
		return finishCryptoReplay(ctx, tx, replayID)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	previous, err := readCryptoReceipt(ctx, tx, `SELECT payload FROM crypto_receipts WHERE occurrence_key=? ORDER BY sequence DESC LIMIT 1`, candidate.OccurrenceKey)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	atomicDelta, valuationDelta, rootID, reversalOf, err := cryptoEffect(in, candidate.InclusionRevision, previous, id)
	if err != nil {
		return nil, err
	}
	if in.Attribution.CreditLotId != "" && previous == nil {
		var existing string
		err := tx.QueryRowContext(ctx, `SELECT occurrence_key FROM crypto_receipts WHERE action=1 AND credit_lot_owner=? AND credit_lot_id=?`, in.Attribution.CreditLotOwner, in.Attribution.CreditLotId).Scan(&existing)
		if err == nil {
			return nil, fmt.Errorf("%w: credit lot is already attributed to another payment", ErrCryptoConflict)
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}
	postingID := uuid.NewString()
	out := &pb.AcceptedCryptoReceipt{
		Id: id, PostingId: postingID, Source: in.Source, OccurrenceKey: candidate.OccurrenceKey,
		InclusionRevision: candidate.InclusionRevision, AtomicDelta: atomicDelta, ValuationDeltaMinor: valuationDelta,
		Currency: in.Valuation.Currency, AcceptedAt: timestamppb.New(at), OriginalReceiptId: rootID,
		Action: in.Action, Attribution: in.Attribution, NoChargeback: true, PreviousReceiptId: in.PreviousReceiptId,
		Observation: in.Observation, Valuation: in.Valuation, OccurredAt: in.OccurredAt,
		CorrectionReason: in.CorrectionReason, AcceptedBy: actor,
	}
	event := &sharedpb.MoneyEvent{
		Id: postingID, ExternalId: id, AdapterId: TreasuryAdapterID, AccountId: in.Observation.AccountId,
		BookId: in.Observation.BookId, AmountMinor: valuationDelta, Currency: in.Valuation.Currency,
		OccurredAt: in.OccurredAt, FetchedAt: timestamppb.New(at), Basis: in.Valuation.Basis,
		Description: "Base USDC " + strings.TrimPrefix(in.Action.String(), "CRYPTO_RECEIPT_ACTION_"), Category: "crypto_receipt",
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO adapters(id,name,kind,enabled,created_at) VALUES(?,?,?,?,?) ON CONFLICT(id) DO NOTHING`, TreasuryAdapterID, "Treasury Base USDC", int32(pb.AdapterKind_ADAPTER_KIND_COMMERCE), true, at.Format(time.RFC3339Nano)); err != nil {
		return nil, err
	}
	if _, err := s.appendMoneyEvent(ctx, tx, event, actor, reversalOf, at.Format(time.RFC3339Nano)); err != nil {
		return nil, err
	}
	encoded, err := proto.Marshal(out)
	if err != nil {
		return nil, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO crypto_receipts(id,posting_id,occurrence_key,inclusion_revision,original_receipt_id,previous_receipt_id,action,book_id,account_id,wallet_id,source_check_id,network,asset_id,decimals,atomic_amount,atomic_delta,valuation_amount_minor,valuation_delta_minor,currency,valuation_basis_reference,payer_address,payer_identity,credit_lot_id,credit_lot_owner,no_chargeback,accepted_at,semantic_hash,payload) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, out.Id, out.PostingId, out.OccurrenceKey, out.InclusionRevision, out.OriginalReceiptId, out.PreviousReceiptId, int32(out.Action), in.Observation.BookId, in.Observation.AccountId, in.Source.WalletId, in.Source.CheckId, in.Observation.Network, in.Observation.AssetId, in.Observation.Decimals, in.Observation.AtomicAmount, atomicDelta, in.Valuation.AmountMinor, valuationDelta, out.Currency, in.Valuation.BasisReference, in.Attribution.PayerAddress, in.Attribution.PayerIdentity, in.Attribution.CreditLotId, in.Attribution.CreditLotOwner, true, at.Format(time.RFC3339Nano), candidate.SemanticHash, encoded)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE adapters SET last_success_at=?,availability_reason='' WHERE id=?`, at.Format(time.RFC3339Nano), TreasuryAdapterID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &pb.AcceptCryptoReceiptResponse{Receipt: out}, nil
}

func finishCryptoReplay(ctx context.Context, tx *sql.Tx, id string) (*pb.AcceptCryptoReceiptResponse, error) {
	receipt, err := readCryptoReceipt(ctx, tx, `SELECT payload FROM crypto_receipts WHERE id=?`, id)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &pb.AcceptCryptoReceiptResponse{Receipt: receipt, Duplicate: true}, nil
}

func cryptoEffect(in *pb.AcceptCryptoReceiptRequest, revision string, previous *pb.AcceptedCryptoReceipt, id string) (atomicDelta string, valuationDelta int64, rootID, reversalOf string, err error) {
	if previous == nil {
		if in.Action != pb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_ORIGINAL {
			err = ErrCryptoPredecessor
			return
		}
		return in.Observation.AtomicAmount, in.Valuation.AmountMinor, id, "", nil
	}
	if in.Action == pb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_ORIGINAL {
		err = ErrCryptoConflict
		return
	}
	if in.PreviousReceiptId != previous.Id || in.OriginalReceiptId != previous.OriginalReceiptId {
		err = ErrCryptoPredecessor
		return
	}
	rootID = previous.OriginalReceiptId
	if in.Observation.BookId != previous.Observation.BookId || in.Observation.AccountId != previous.Observation.AccountId || in.Observation.Decimals != previous.Observation.Decimals || in.Source.WalletId != previous.Source.WalletId || in.Source.OperatorIdentity != previous.Source.OperatorIdentity || in.Valuation.Currency != previous.Currency || !proto.Equal(in.Attribution, previous.Attribution) {
		err = fmt.Errorf("%w: scope, asset metadata, currency and payer/lot attribution are immutable", ErrCryptoConflict)
		return
	}
	priorAmount, _ := new(big.Int).SetString(previous.Observation.AtomicAmount, 10)
	targetAmount, _ := new(big.Int).SetString(in.Observation.AtomicAmount, 10)
	priorValue := previous.Valuation.AmountMinor
	targetValue := in.Valuation.AmountMinor
	removed := previous.Action == pb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_REMOVAL
	switch in.Action {
	case pb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_REINCLUSION:
		if revision == previous.InclusionRevision || in.Observation.AtomicAmount != previous.Observation.AtomicAmount || !proto.Equal(in.Valuation, previous.Valuation) {
			err = fmt.Errorf("%w: re-inclusion requires a new block and the accepted quantity/valuation", ErrCryptoConflict)
			return
		}
		if removed {
			priorAmount.SetInt64(0)
			priorValue = 0
		}
	case pb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_REMOVAL:
		if removed || revision != previous.InclusionRevision || in.Observation.AtomicAmount != previous.Observation.AtomicAmount || !proto.Equal(in.Valuation, previous.Valuation) {
			err = fmt.Errorf("%w: removal requires the active accepted inclusion, quantity and valuation", ErrCryptoConflict)
			return
		}
		targetAmount.SetInt64(0)
		targetValue = 0
		reversalOf = previous.PostingId
	case pb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_CORRECTION:
		if removed || revision != previous.InclusionRevision {
			err = fmt.Errorf("%w: correction requires an active accepted inclusion", ErrCryptoConflict)
			return
		}
	default:
		err = ErrCryptoConflict
		return
	}
	// Both targets are bounded, nonnegative int64 values; their difference fits
	// int64 without multiplication or token-to-fiat coercion.
	return new(big.Int).Sub(targetAmount, priorAmount).String(), targetValue - priorValue, rootID, reversalOf, nil
}

func (s *Store) GetCryptoReceipt(ctx context.Context, id string) (*pb.AcceptedCryptoReceipt, error) {
	return readCryptoReceipt(ctx, s.db, `SELECT payload FROM crypto_receipts WHERE id=?`, id)
}

func (s *Store) ListCryptoReceipts(ctx context.Context, in *pb.ListCryptoReceiptsRequest) (*pb.ListCryptoReceiptsResponse, error) {
	limit := in.GetLimit()
	if limit == 0 || limit > 500 {
		limit = 100
	}
	var after int64
	if in.GetAfterReceiptId() != "" {
		if err := s.db.QueryRowContext(ctx, `SELECT sequence FROM crypto_receipts WHERE id=?`, in.AfterReceiptId).Scan(&after); err != nil {
			return nil, err
		}
	}
	query := `SELECT payload FROM crypto_receipts WHERE sequence>?`
	args := []any{after}
	for _, filter := range []struct{ column, value string }{{"book_id", in.GetBookId()}, {"wallet_id", in.GetWalletId()}, {"credit_lot_id", in.GetCreditLotId()}} {
		if filter.value != "" {
			query += " AND " + filter.column + "=?"
			args = append(args, filter.value)
		}
	}
	query += ` ORDER BY sequence LIMIT ?`
	args = append(args, limit+1)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := &pb.ListCryptoReceiptsResponse{}
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		if len(out.Receipts) == int(limit) {
			out.NextAfterReceiptId = out.Receipts[len(out.Receipts)-1].Id
			break
		}
		receipt := &pb.AcceptedCryptoReceipt{}
		if err := proto.Unmarshal(payload, receipt); err != nil {
			return nil, err
		}
		out.Receipts = append(out.Receipts, receipt)
	}
	return out, rows.Err()
}
