package ingest

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
)

var (
	atomicInteger = regexp.MustCompile(`^(0|[1-9][0-9]{0,77})$`)
	evmAddress    = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)
	evmHash       = regexp.MustCompile(`^0x[0-9a-fA-F]{64}$`)
	evmNetwork    = regexp.MustCompile(`^eip155:[1-9][0-9]{0,19}$`)
)

// PreviewCryptoObservation interprets untrusted chain observations without
// promoting them into owned funds, fiat value, a posting or budget credit.
// It deliberately contains no call to any writer or configurable policy engine.
func (s *Store) PreviewCryptoObservation(ctx context.Context, in *ingestpb.PreviewCryptoObservationRequest) (*ingestpb.PreviewCryptoObservationResponse, error) {
	if in == nil {
		return nil, errors.New("crypto observation is required")
	}
	if !evmNetwork.MatchString(in.Network) {
		return nil, errors.New("canonical EVM network identity is required")
	}
	if _, err := strconv.ParseUint(strings.TrimPrefix(in.Network, "eip155:"), 10, 64); err != nil {
		return nil, errors.New("EVM chain id exceeds supported uint64 identity")
	}
	if !strings.HasPrefix(in.AssetId, in.Network+"/") {
		return nil, errors.New("asset network must match observation network")
	}
	asset := strings.TrimPrefix(in.AssetId, in.Network+"/")
	native := asset == "slip44:60"
	if !native && (!strings.HasPrefix(asset, "erc20:") || !evmAddress.MatchString(strings.TrimPrefix(asset, "erc20:"))) {
		return nil, errors.New("asset must identify native ETH or an exact ERC20 contract")
	}
	if in.Decimals > 255 || native && in.Decimals != 18 {
		return nil, errors.New("asset decimals are invalid")
	}
	if !atomicInteger.MatchString(in.AtomicAmount) {
		return nil, errors.New("atomic amount must be a canonical unsigned uint256 decimal string")
	}
	amount, ok := new(big.Int).SetString(in.AtomicAmount, 10)
	if !ok || amount.BitLen() > 256 {
		return nil, errors.New("atomic amount exceeds uint256")
	}
	if !evmAddress.MatchString(in.Address) {
		return nil, errors.New("observed EVM address is required")
	}
	if strings.TrimSpace(in.SourceReference) == "" || len(in.SourceReference) > 1024 || strings.ContainsAny(in.SourceReference, "\r\n") {
		return nil, errors.New("bounded source reference is required")
	}
	if _, ok := ingestpb.CryptoObservationState_name[int32(in.State)]; !ok || in.State == ingestpb.CryptoObservationState_CRYPTO_OBSERVATION_STATE_UNSPECIFIED {
		return nil, errors.New("explicit observation state is required")
	}
	if _, ok := ingestpb.CryptoLegKind_name[int32(in.LegKind)]; !ok || in.LegKind == ingestpb.CryptoLegKind_CRYPTO_LEG_KIND_UNSPECIFIED {
		return nil, errors.New("explicit observation leg kind is required")
	}
	if in.LegKind != ingestpb.CryptoLegKind_CRYPTO_LEG_KIND_BALANCE && !evmHash.MatchString(in.TransactionHash) {
		return nil, errors.New("transaction leg requires its exact hash")
	}
	if in.State != ingestpb.CryptoObservationState_CRYPTO_OBSERVATION_STATE_PENDING && !evmHash.MatchString(in.BlockHash) {
		return nil, errors.New("included, removed or reverted observation requires its block hash")
	}
	if in.BlockHash != "" && !evmHash.MatchString(in.BlockHash) {
		return nil, errors.New("invalid block hash")
	}
	if in.TransactionHash != "" && !evmHash.MatchString(in.TransactionHash) {
		return nil, errors.New("invalid transaction hash")
	}
	out := &ingestpb.PreviewCryptoObservationResponse{Network: in.Network, AssetId: strings.ToLower(in.AssetId), AtomicAmount: in.AtomicAmount, Decimals: in.Decimals, Address: strings.ToLower(in.Address), SourceReportedState: in.State, LegKind: in.LegKind, SourceReference: in.SourceReference, PostingState: "withheld", BudgetState: "uncredited", BookAccountStatus: "not_supplied", Holds: []string{"wallet_ownership_unverified", "authorized_valuation_absent", "canonicality_source_reported_only", "asset_metadata_source_reported_only"}}
	if in.BookId != "" || in.AccountId != "" {
		if in.BookId == "" || in.AccountId == "" {
			return nil, errors.New("book and account must be supplied together")
		}
		if s == nil || s.db == nil {
			return nil, errors.New("book readback unavailable")
		}
		var accountBook string
		err := s.db.QueryRowContext(ctx, `SELECT a.book_id FROM accounts a JOIN books b ON b.id=a.book_id WHERE a.id=? AND b.archived=0`, in.AccountId).Scan(&accountBook)
		if err != nil || accountBook != in.BookId {
			return nil, errors.New("account is not in the supplied active book")
		}
		out.BookAccountStatus = "active_book_account_readback"
	}
	if in.BlockHash != "" {
		bytes, _ := json.Marshal([]any{in.Network, out.AssetId, out.Address, strings.ToLower(in.TransactionHash), strings.ToLower(in.BlockHash), in.LegIndex, int32(in.LegKind)})
		out.OccurrenceKey = fmt.Sprintf("sha256:%x", sha256.Sum256(bytes))
	}
	switch in.State {
	case ingestpb.CryptoObservationState_CRYPTO_OBSERVATION_STATE_PENDING:
		out.Disposition = "pending_uncredited"
	case ingestpb.CryptoObservationState_CRYPTO_OBSERVATION_STATE_REMOVED:
		out.Disposition = "correction_requires_accepted_original"
		out.CorrectionRequiresOriginalReceipt = true
		out.Holds = append(out.Holds, "accepted_original_receipt_required")
	default:
		out.Disposition = "unvalued_candidate_only"
		switch {
		case in.LegKind == ingestpb.CryptoLegKind_CRYPTO_LEG_KIND_APPROVAL:
			out.CandidateEffectAtomic = "0"
		case in.LegKind == ingestpb.CryptoLegKind_CRYPTO_LEG_KIND_FEE:
			out.CandidateEffectAtomic = amount.Neg(amount).String()
		case in.State == ingestpb.CryptoObservationState_CRYPTO_OBSERVATION_STATE_REVERTED:
			out.CandidateEffectAtomic = "0"
		case in.LegKind == ingestpb.CryptoLegKind_CRYPTO_LEG_KIND_BALANCE:
			out.Disposition = "balance_snapshot_not_a_deposit"
		default:
			out.CandidateEffectAtomic = in.AtomicAmount
		}
	}
	return out, nil
}
