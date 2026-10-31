package ingest

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	"google.golang.org/protobuf/proto"
)

func cryptoObservation() *ingestpb.PreviewCryptoObservationRequest {
	return &ingestpb.PreviewCryptoObservationRequest{Network: "eip155:8453", AssetId: "eip155:8453/erc20:0x833589fcd6edb6e08f4c7c32d4f71b54bda02913", Address: "0x1111111111111111111111111111111111111111", AtomicAmount: "5901522149285533025181", Decimals: 6, State: ingestpb.CryptoObservationState_CRYPTO_OBSERVATION_STATE_FINALIZED, LegKind: ingestpb.CryptoLegKind_CRYPTO_LEG_KIND_DEPOSIT, SourceReference: "fixture:explicitly-synthetic-deposit", TransactionHash: "0x" + strings.Repeat("a", 64), BlockHash: "0x" + strings.Repeat("b", 64), LegIndex: 3}
}

func TestCryptoPreviewPreservesExactUnitsAndWithholdsPosting(t *testing.T) {
	s, ctx := newIngestStore(t)
	in := cryptoObservation()
	out, err := s.PreviewCryptoObservation(ctx, in)
	require.NoError(t, err)
	require.Equal(t, in.AtomicAmount, out.AtomicAmount)
	require.Equal(t, in.AtomicAmount, out.CandidateEffectAtomic)
	require.Equal(t, "withheld", out.PostingState)
	require.Equal(t, "uncredited", out.BudgetState)
	require.Contains(t, out.Holds, "wallet_ownership_unverified")
	require.Contains(t, out.Holds, "authorized_valuation_absent")
	require.Contains(t, out.Holds, "canonicality_source_reported_only")
	for _, table := range []string{"postings", "ingest_receipts", "ledger_audit", "adapters"} {
		var count int
		require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table).Scan(&count))
		require.Zero(t, count)
	}
}

func TestCryptoPreviewRejectsWrongNetworkAndNonCanonicalAmounts(t *testing.T) {
	s, ctx := newIngestStore(t)
	for _, bad := range []string{strings.Repeat("9", 1000), "-1", "01", "1.0", "1e18", "+1", "", "115792089237316195423570985008687907853269984665640564039457584007913129639936"} {
		in := cryptoObservation()
		in.AtomicAmount = bad
		_, err := s.PreviewCryptoObservation(ctx, in)
		require.Error(t, err, bad)
	}
	in := cryptoObservation()
	in.Network = "eip155:84532"
	_, err := s.PreviewCryptoObservation(ctx, in)
	require.Error(t, err)
	in = cryptoObservation()
	in.Decimals = 256
	_, err = s.PreviewCryptoObservation(ctx, in)
	require.Error(t, err)
	in = cryptoObservation()
	in.AtomicAmount = "115792089237316195423570985008687907853269984665640564039457584007913129639935"
	out, err := s.PreviewCryptoObservation(ctx, in)
	require.NoError(t, err)
	require.Equal(t, in.AtomicAmount, out.AtomicAmount)
}

func TestCryptoPreviewPendingApprovalReversionAndRemovalHaveDistinctEffects(t *testing.T) {
	s, ctx := newIngestStore(t)
	in := cryptoObservation()
	pending := proto.Clone(in).(*ingestpb.PreviewCryptoObservationRequest)
	pending.State = ingestpb.CryptoObservationState_CRYPTO_OBSERVATION_STATE_PENDING
	pending.BlockHash = ""
	out, err := s.PreviewCryptoObservation(ctx, pending)
	require.NoError(t, err)
	require.Empty(t, out.CandidateEffectAtomic)
	approval := proto.Clone(in).(*ingestpb.PreviewCryptoObservationRequest)
	approval.LegKind = ingestpb.CryptoLegKind_CRYPTO_LEG_KIND_APPROVAL
	out, err = s.PreviewCryptoObservation(ctx, approval)
	require.NoError(t, err)
	require.Equal(t, "0", out.CandidateEffectAtomic)
	reverted := proto.Clone(in).(*ingestpb.PreviewCryptoObservationRequest)
	reverted.State = ingestpb.CryptoObservationState_CRYPTO_OBSERVATION_STATE_REVERTED
	out, err = s.PreviewCryptoObservation(ctx, reverted)
	require.NoError(t, err)
	require.Equal(t, "0", out.CandidateEffectAtomic)
	reverted.LegKind = ingestpb.CryptoLegKind_CRYPTO_LEG_KIND_FEE
	reverted.AssetId = "eip155:8453/slip44:60"
	reverted.Decimals = 18
	out, err = s.PreviewCryptoObservation(ctx, reverted)
	require.NoError(t, err)
	require.Equal(t, "-"+reverted.AtomicAmount, out.CandidateEffectAtomic)
	removed := proto.Clone(in).(*ingestpb.PreviewCryptoObservationRequest)
	removed.State = ingestpb.CryptoObservationState_CRYPTO_OBSERVATION_STATE_REMOVED
	out, err = s.PreviewCryptoObservation(ctx, removed)
	require.NoError(t, err)
	require.True(t, out.CorrectionRequiresOriginalReceipt)
	require.Empty(t, out.CandidateEffectAtomic)
	original, err := s.PreviewCryptoObservation(ctx, in)
	require.NoError(t, err)
	require.Equal(t, original.OccurrenceKey, out.OccurrenceKey)
	reincluded := proto.Clone(in).(*ingestpb.PreviewCryptoObservationRequest)
	reincluded.BlockHash = "0x" + strings.Repeat("c", 64)
	again, err := s.PreviewCryptoObservation(ctx, reincluded)
	require.NoError(t, err)
	require.NotEqual(t, original.OccurrenceKey, again.OccurrenceKey)
}

func TestCryptoPreviewBookReadbackDoesNotProveWalletOwnership(t *testing.T) {
	s, ctx := newIngestStore(t)
	b, err := s.journal.CreateBook(ctx, "Fixture", "USD")
	require.NoError(t, err)
	a, err := s.journal.CreateAccount(ctx, b.Id, "Cash", ledgerpb.AccountKind_ASSET)
	require.NoError(t, err)
	in := cryptoObservation()
	in.BookId = b.Id
	in.AccountId = a.Id
	out, err := s.PreviewCryptoObservation(ctx, in)
	require.NoError(t, err)
	require.Equal(t, "active_book_account_readback", out.BookAccountStatus)
	require.Contains(t, out.Holds, "wallet_ownership_unverified")
	require.Equal(t, "withheld", out.PostingState)
	other, err := s.journal.CreateBook(ctx, "Other", "USD")
	require.NoError(t, err)
	in.BookId = other.Id
	_, err = s.PreviewCryptoObservation(ctx, in)
	require.Error(t, err)
}
