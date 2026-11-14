package ingest

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"money-ledger/internal/ledger"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func cryptoAcceptance(t *testing.T, s *Store) *pb.AcceptCryptoReceiptRequest {
	t.Helper()
	ctx := context.Background()
	book, err := s.journal.CreateBook(ctx, "Base rehearsal", "USD")
	require.NoError(t, err)
	account, err := s.journal.CreateAccount(ctx, book.Id, "USDC", ledgerpb.AccountKind_ASSET)
	require.NoError(t, err)
	observation := cryptoObservation()
	observation.BookId, observation.AccountId = book.Id, account.Id
	at := timestamppb.New(time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC))
	return &pb.AcceptCryptoReceiptRequest{
		Observation: observation, Source: &pb.TreasuryReceiptLink{CheckId: "check-1", WalletId: "test-wallet", OperatorIdentity: "operator", SourceReference: "treasury:check-1"},
		Action:      pb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_ORIGINAL,
		Valuation:   &pb.CryptoValuation{AmountMinor: 1000, Currency: "USD", BasisReference: "operator:test-valuation-no-implied-peg", ValuedAt: at, Basis: sharedpb.Basis_BASIS_OPERATOR_ASSERTED},
		Attribution: &pb.CryptoPayerAttribution{PayerAddress: "0x" + strings.Repeat("2", 40)}, OccurredAt: at,
	}
}

func revision(in *pb.AcceptCryptoReceiptRequest, previous *pb.AcceptedCryptoReceipt, action pb.CryptoReceiptAction, check string) *pb.AcceptCryptoReceiptRequest {
	out := proto.Clone(in).(*pb.AcceptCryptoReceiptRequest)
	out.Action, out.OriginalReceiptId, out.PreviousReceiptId = action, previous.OriginalReceiptId, previous.Id
	out.Source.CheckId, out.Source.SourceReference, out.Observation.SourceReference = check, "treasury:"+check, "treasury:"+check
	out.CorrectionReason = "provider evidence revision " + check
	return out
}

func TestCryptoAcceptanceExactFiveReceiptClasses(t *testing.T) { // [REQ:CRYP-002] [REQ:CRYP-003] [REQ:CRYP-004]
	s, ctx := newIngestStore(t)
	in := cryptoAcceptance(t, s)
	one, err := s.AcceptCryptoReceipt(ctx, in, "treasury")
	require.NoError(t, err)
	require.False(t, one.Duplicate)
	require.Equal(t, in.Observation.AtomicAmount, one.Receipt.AtomicDelta, "quantity exceeds int64 and must remain exact")
	require.EqualValues(t, 1000, one.Receipt.ValuationDeltaMinor)
	require.Equal(t, one.Receipt.Id, one.Receipt.OriginalReceiptId)
	for _, request := range []*pb.AcceptCryptoReceiptRequest{in, proto.Clone(in).(*pb.AcceptCryptoReceiptRequest)} {
		if request != in {
			request.Source.CheckId = "fresh-verify-check"
			request.Source.SourceReference = "treasury:fresh-verify-check"
		}
		replay, err := s.AcceptCryptoReceipt(ctx, request, "treasury")
		require.NoError(t, err)
		require.True(t, replay.Duplicate)
		require.True(t, proto.Equal(one.Receipt, replay.Receipt))
	}
	reincluded := revision(in, one.Receipt, pb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_REINCLUSION, "check-reincluded")
	reincluded.Observation.BlockHash = "0x" + strings.Repeat("c", 64)
	two, err := s.AcceptCryptoReceipt(ctx, reincluded, "treasury")
	require.NoError(t, err)
	require.Equal(t, "0", two.Receipt.AtomicDelta)
	require.Zero(t, two.Receipt.ValuationDeltaMinor)
	removed := revision(reincluded, two.Receipt, pb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_REMOVAL, "check-removed")
	removed.Observation.State = pb.CryptoObservationState_CRYPTO_OBSERVATION_STATE_REMOVED
	three, err := s.AcceptCryptoReceipt(ctx, removed, "treasury")
	require.NoError(t, err)
	require.Equal(t, "-"+in.Observation.AtomicAmount, three.Receipt.AtomicDelta)
	require.EqualValues(t, -1000, three.Receipt.ValuationDeltaMinor)
	restored := revision(reincluded, three.Receipt, pb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_REINCLUSION, "check-restored")
	restored.Observation.BlockHash = "0x" + strings.Repeat("d", 64)
	four, err := s.AcceptCryptoReceipt(ctx, restored, "treasury")
	require.NoError(t, err)
	require.Equal(t, in.Observation.AtomicAmount, four.Receipt.AtomicDelta)
	require.EqualValues(t, 1000, four.Receipt.ValuationDeltaMinor)
	corrected := revision(restored, four.Receipt, pb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_CORRECTION, "check-corrected")
	corrected.Observation.AtomicAmount = "5901522149285533025182"
	corrected.Valuation.AmountMinor = 950
	corrected.Valuation.BasisReference = "operator:corrected-valuation"
	five, err := s.AcceptCryptoReceipt(ctx, corrected, "treasury")
	require.NoError(t, err)
	require.Equal(t, "1", five.Receipt.AtomicDelta)
	require.EqualValues(t, -50, five.Receipt.ValuationDeltaMinor)
	customer := proto.Clone(in).(*pb.AcceptCryptoReceiptRequest)
	customer.Source.CheckId = "customer-payment"
	customer.Observation.TransactionHash = "0x" + strings.Repeat("e", 64)
	customer.Observation.AtomicAmount = "2500000"
	customer.Valuation.AmountMinor = 250
	customer.Attribution.PayerIdentity = "customer:42"
	customer.Attribution.CreditLotId = "lot-42"
	customer.Attribution.CreditLotOwner = "landing-page-business-suite"
	six, err := s.AcceptCryptoReceipt(ctx, customer, "treasury")
	require.NoError(t, err)
	require.True(t, six.Receipt.NoChargeback)
	require.True(t, proto.Equal(customer.Attribution, six.Receipt.Attribution))
	lot, err := s.ListCryptoReceipts(ctx, &pb.ListCryptoReceiptsRequest{CreditLotId: "lot-42"})
	require.NoError(t, err)
	require.Len(t, lot.Receipts, 1)
	position, err := s.journal.Position(ctx, in.Observation.BookId)
	require.NoError(t, err)
	require.EqualValues(t, 1200, position.CashMinor, "fiat position uses explicit valuation deltas only")
	postings, err := s.journal.ListPostings(ctx, "", in.Observation.BookId, "", "", 100)
	require.NoError(t, err)
	require.Len(t, postings, 6)
	for _, posting := range postings {
		require.NotEmpty(t, posting.CryptoReceiptId)
		require.Len(t, posting.Audit, 1)
		stored, err := s.GetCryptoReceipt(ctx, posting.CryptoReceiptId)
		require.NoError(t, err)
		require.Equal(t, posting.Id, stored.PostingId)
		require.Equal(t, posting.Event.AmountMinor, stored.ValuationDeltaMinor)
	}
	var submissions int
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM crypto_submissions`).Scan(&submissions))
	require.Equal(t, 7, submissions, "fresh verification is retained without adding a posting")
	page, err := s.ListCryptoReceipts(ctx, &pb.ListCryptoReceiptsRequest{BookId: in.Observation.BookId, Limit: 2})
	require.NoError(t, err)
	require.Len(t, page.Receipts, 2)
	require.Equal(t, two.Receipt.Id, page.NextAfterReceiptId)
	next, err := s.ListCryptoReceipts(ctx, &pb.ListCryptoReceiptsRequest{BookId: in.Observation.BookId, AfterReceiptId: page.NextAfterReceiptId})
	require.NoError(t, err)
	require.Len(t, next.Receipts, 4)
}

func TestCryptoAcceptanceRejectsConflictingAndStaleCorrections(t *testing.T) { // [REQ:CRYP-004]
	s, ctx := newIngestStore(t)
	in := cryptoAcceptance(t, s)
	one, err := s.AcceptCryptoReceipt(ctx, in, "treasury")
	require.NoError(t, err)
	changed := proto.Clone(in).(*pb.AcceptCryptoReceiptRequest)
	changed.Valuation.AmountMinor++
	_, err = s.AcceptCryptoReceipt(ctx, changed, "treasury")
	require.ErrorIs(t, err, ledger.ErrCryptoConflict)
	changed.Source.CheckId = "different-source"
	_, err = s.AcceptCryptoReceipt(ctx, changed, "treasury")
	require.ErrorIs(t, err, ledger.ErrCryptoConflict)
	corrected := revision(in, one.Receipt, pb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_CORRECTION, "correction")
	corrected.Valuation.AmountMinor = 800
	two, err := s.AcceptCryptoReceipt(ctx, corrected, "treasury")
	require.NoError(t, err)
	stale := revision(in, one.Receipt, pb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_REMOVAL, "stale-removal")
	stale.Observation.State = pb.CryptoObservationState_CRYPTO_OBSERVATION_STATE_REMOVED
	_, err = s.AcceptCryptoReceipt(ctx, stale, "treasury")
	require.ErrorIs(t, err, ledger.ErrCryptoPredecessor)
	wrongRoot := revision(corrected, two.Receipt, pb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_CORRECTION, "wrong-root")
	wrongRoot.OriginalReceiptId = "not-accepted"
	_, err = s.AcceptCryptoReceipt(ctx, wrongRoot, "treasury")
	require.ErrorIs(t, err, ledger.ErrCryptoPredecessor)
	wrongPayer := revision(corrected, two.Receipt, pb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_CORRECTION, "wrong-payer")
	wrongPayer.Attribution.PayerAddress = "0x" + strings.Repeat("3", 40)
	_, err = s.AcceptCryptoReceipt(ctx, wrongPayer, "treasury")
	require.ErrorIs(t, err, ledger.ErrCryptoConflict)
	_, err = s.journal.Reverse(ctx, one.Receipt.PostingId, "bypass token correction", "operator")
	require.ErrorContains(t, err, "accepted crypto correction")
	posting, err := s.journal.GetPosting(ctx, one.Receipt.PostingId)
	require.NoError(t, err)
	_, _, err = s.journal.Ingest(ctx, posting.Event, "operator")
	require.ErrorContains(t, err, "authenticated receipt acceptance")
	position, err := s.journal.Position(ctx, in.Observation.BookId)
	require.NoError(t, err)
	require.EqualValues(t, 800, position.CashMinor)
}

func TestCryptoAcceptanceRollsBackEveryOwnedFact(t *testing.T) { // [REQ:CRYP-004]
	s, ctx := newIngestStore(t)
	in := cryptoAcceptance(t, s)
	_, err := s.db.ExecContext(ctx, `CREATE TRIGGER reject_crypto_receipt BEFORE INSERT ON crypto_receipts BEGIN SELECT RAISE(ABORT, 'injected receipt write failure'); END;`)
	require.NoError(t, err)
	_, err = s.AcceptCryptoReceipt(ctx, in, "treasury")
	require.ErrorContains(t, err, "injected receipt write failure")
	for _, table := range []string{"crypto_submissions", "crypto_receipts", "postings", "ledger_audit", "adapters"} {
		var count int
		require.NoError(t, s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count))
		require.Zero(t, count, table)
	}
}

func openCryptoStore(t *testing.T, path string) (*Store, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(func() string { return (&ledger.Store{}).Schema() })))
	return NewStore(database.NewFromPrimary(db), nil), db
}

func TestCryptoAcceptanceSurvivesReopenAndConcurrentChecks(t *testing.T) { // [REQ:CRYP-003]
	path := filepath.Join(t.TempDir(), "ledger.sqlite")
	a, db := openCryptoStore(t, path)
	in := cryptoAcceptance(t, a)
	b, _ := openCryptoStore(t, path)
	start := make(chan struct{})
	type result struct {
		response *pb.AcceptCryptoReceiptResponse
		err      error
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for i, store := range []*Store{a, b} {
		wg.Add(1)
		go func(i int, store *Store) {
			defer wg.Done()
			request := proto.Clone(in).(*pb.AcceptCryptoReceiptRequest)
			if i == 1 {
				request.Source.CheckId = "concurrent-check"
			}
			<-start
			response, err := store.AcceptCryptoReceipt(context.Background(), request, "treasury")
			results <- result{response, err}
		}(i, store)
	}
	close(start)
	wg.Wait()
	close(results)
	var accepted *pb.AcceptedCryptoReceipt
	duplicates := 0
	for result := range results {
		require.NoError(t, result.err)
		if accepted != nil {
			require.Equal(t, accepted.Id, result.response.Receipt.Id)
		}
		accepted = result.response.Receipt
		if result.response.Duplicate {
			duplicates++
		}
	}
	require.Equal(t, 1, duplicates)
	require.NoError(t, db.Close())
	fresh, _ := openCryptoStore(t, path)
	stored, err := fresh.GetCryptoReceipt(context.Background(), accepted.Id)
	require.NoError(t, err)
	require.True(t, proto.Equal(accepted, stored))
	replayed, err := fresh.AcceptCryptoReceipt(context.Background(), in, "treasury")
	require.NoError(t, err)
	require.True(t, replayed.Duplicate)
	require.Equal(t, accepted.Id, replayed.Receipt.Id)
}

func TestCryptoAcceptanceRejectsCheckRebindingAndDuplicateCreditLot(t *testing.T) { // [REQ:CRYP-002]
	s, ctx := newIngestStore(t)
	in := cryptoAcceptance(t, s)
	in.Attribution.PayerIdentity, in.Attribution.CreditLotId, in.Attribution.CreditLotOwner = "customer-1", "lot-1", "landing-page-business-suite"
	_, err := s.AcceptCryptoReceipt(ctx, in, "treasury")
	require.NoError(t, err)
	other := proto.Clone(in).(*pb.AcceptCryptoReceiptRequest)
	other.Observation.TransactionHash = "0x" + strings.Repeat("f", 64)
	_, err = s.AcceptCryptoReceipt(ctx, other, "treasury")
	require.ErrorContains(t, err, "check identity cannot be rebound")
	other.Source.CheckId = "another-check"
	_, err = s.AcceptCryptoReceipt(ctx, other, "treasury")
	require.ErrorContains(t, err, "credit lot is already attributed")
	other.Attribution.CreditLotId = "lot-2"
	_, err = s.AcceptCryptoReceipt(ctx, other, "treasury")
	require.NoError(t, err)
	_, err = s.RegisterAdapter(ctx, &pb.Adapter{Id: ledger.TreasuryAdapterID, Name: "override", Kind: pb.AdapterKind_ADAPTER_KIND_MANUAL})
	require.ErrorContains(t, err, "authenticated receipt acceptance")
}

func TestCryptoAcceptanceAllowsCustomerPayerBeforeLPBSCreatesLot(t *testing.T) { // [REQ:CRYP-002]
	s, ctx := newIngestStore(t)
	in := cryptoAcceptance(t, s)
	in.Attribution.PayerIdentity = "customer:cf2-local"
	// The authoritative LPBS lot does not exist until after this immutable
	// Money Ledger acceptance. Its ID is therefore intentionally absent here.
	in.Attribution.CreditLotId = ""
	in.Attribution.CreditLotOwner = ""
	accepted, err := s.AcceptCryptoReceipt(ctx, in, "treasury")
	require.NoError(t, err)
	require.True(t, accepted.Receipt.NoChargeback)
	require.Equal(t, in.Attribution.PayerAddress, accepted.Receipt.Attribution.PayerAddress)
	require.Equal(t, "customer:cf2-local", accepted.Receipt.Attribution.PayerIdentity)
	require.Empty(t, accepted.Receipt.Attribution.CreditLotId)
	require.Empty(t, accepted.Receipt.Attribution.CreditLotOwner)
	require.Equal(t, in.Observation.AtomicAmount, accepted.Receipt.AtomicDelta)
}

func TestCryptoAcceptanceRejectsUnfinalizedAmbiguousOrUnvaluedEvidence(t *testing.T) { // [REQ:CRYP-001] [REQ:CRYP-002]
	s, ctx := newIngestStore(t)
	in := cryptoAcceptance(t, s)
	for _, test := range []struct {
		name   string
		change func(*pb.AcceptCryptoReceiptRequest)
	}{
		{"missing observation", func(r *pb.AcceptCryptoReceiptRequest) { r.Observation = nil }},
		{"missing source", func(r *pb.AcceptCryptoReceiptRequest) { r.Source = nil }},
		{"missing valuation", func(r *pb.AcceptCryptoReceiptRequest) { r.Valuation = nil }},
		{"missing payer", func(r *pb.AcceptCryptoReceiptRequest) { r.Attribution = nil }},
		{"pending", func(r *pb.AcceptCryptoReceiptRequest) {
			r.Observation.State = pb.CryptoObservationState_CRYPTO_OBSERVATION_STATE_PENDING
		}},
		{"included", func(r *pb.AcceptCryptoReceiptRequest) {
			r.Observation.State = pb.CryptoObservationState_CRYPTO_OBSERVATION_STATE_INCLUDED
		}},
		{"wrong decimals", func(r *pb.AcceptCryptoReceiptRequest) { r.Observation.Decimals = 18 }},
		{"wrong token", func(r *pb.AcceptCryptoReceiptRequest) {
			r.Observation.AssetId = "eip155:8453/erc20:0x" + strings.Repeat("1", 40)
		}},
		{"approval leg", func(r *pb.AcceptCryptoReceiptRequest) {
			r.Observation.LegKind = pb.CryptoLegKind_CRYPTO_LEG_KIND_APPROVAL
		}},
		{"zero", func(r *pb.AcceptCryptoReceiptRequest) { r.Observation.AtomicAmount = "0" }},
		{"missing action", func(r *pb.AcceptCryptoReceiptRequest) {
			r.Action = pb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_UNSPECIFIED
		}},
		{"negative value", func(r *pb.AcceptCryptoReceiptRequest) { r.Valuation.AmountMinor = -1 }},
		{"missing basis", func(r *pb.AcceptCryptoReceiptRequest) { r.Valuation.Basis = sharedpb.Basis_BASIS_UNSPECIFIED }},
		{"missing basis reference", func(r *pb.AcceptCryptoReceiptRequest) { r.Valuation.BasisReference = "" }},
		{"wrong currency", func(r *pb.AcceptCryptoReceiptRequest) { r.Valuation.Currency = "EUR" }},
		{"missing valuation time", func(r *pb.AcceptCryptoReceiptRequest) { r.Valuation.ValuedAt = nil }},
		{"missing occurrence time", func(r *pb.AcceptCryptoReceiptRequest) { r.OccurredAt = nil }},
		{"invalid payer", func(r *pb.AcceptCryptoReceiptRequest) { r.Attribution.PayerAddress = "customer-1" }},
		{"partial credit lot id", func(r *pb.AcceptCryptoReceiptRequest) { r.Attribution.CreditLotId = "lot-only" }},
		{"partial credit lot owner", func(r *pb.AcceptCryptoReceiptRequest) { r.Attribution.CreditLotOwner = "landing-page-business-suite" }},
		{"original with predecessor", func(r *pb.AcceptCryptoReceiptRequest) { r.PreviousReceiptId = "not-original" }},
		{"source injection", func(r *pb.AcceptCryptoReceiptRequest) { r.Source.CheckId = "check\nforged" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := proto.Clone(in).(*pb.AcceptCryptoReceiptRequest)
			test.change(request)
			_, err := s.AcceptCryptoReceipt(ctx, request, "treasury")
			require.Error(t, err)
		})
	}
	for _, table := range []string{"crypto_receipts", "crypto_submissions", "postings", "ledger_audit"} {
		var count int
		require.NoError(t, s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count))
		require.Zero(t, count, table)
	}
}
