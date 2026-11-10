package receiptfunding

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/api-core/storage"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	_ "modernc.org/sqlite"
	"treasury/internal/baseusdc"
	"treasury/internal/book"
	"treasury/internal/budget"
	"treasury/internal/wallet"
)

type walletReader struct{ value wallet.Wallet }

func (w walletReader) Get(_ context.Context, id, network, operator string) (wallet.Wallet, error) {
	if id != w.value.ID || network != w.value.Network || operator != w.value.OperatorIdentity {
		return wallet.Wallet{}, wallet.ErrMismatch
	}
	return w.value, nil
}

type acceptFunc func(context.Context, *ingestpb.AcceptCryptoReceiptRequest) (*ingestpb.AcceptedCryptoReceipt, error)

func (f acceptFunc) Accept(ctx context.Context, request *ingestpb.AcceptCryptoReceiptRequest) (*ingestpb.AcceptedCryptoReceipt, error) {
	return f(ctx, request)
}

type readFunc func(context.Context, string) (*ingestpb.AcceptedCryptoReceipt, error)
type acceptAndRead struct {
	acceptFunc
	read readFunc
}

func (f acceptAndRead) Get(ctx context.Context, id string) (*ingestpb.AcceptedCryptoReceipt, error) {
	return f.read(ctx, id)
}

type lotGrantFunc func(context.Context, *ingestpb.AcceptedCryptoReceipt) (CreditLotLink, error)

func (f lotGrantFunc) GrantAcceptedReceipt(ctx context.Context, receipt *ingestpb.AcceptedCryptoReceipt) (CreditLotLink, error) {
	return f(ctx, receipt)
}

func TestFundingRecoversLostAcceptanceAndRestartWithoutDuplicateCredit(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 8, 20, 0, 0, 0, time.UTC)
	path := filepath.Join(t.TempDir(), "treasury.sqlite")
	dsn, err := storage.SQLiteDSNAt(path, storage.SQLiteTuning{TxLock: "immediate"})
	require.NoError(t, err)
	open := func() *sql.DB {
		db, err := sql.Open("sqlite", dsn)
		require.NoError(t, err)
		db.SetMaxOpenConns(1)
		require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(book.Schema), database.SchemaProviderFunc(budget.Schema), database.SchemaProviderFunc(baseusdc.Schema), database.SchemaProviderFunc(Schema)))
		t.Cleanup(func() { _ = db.Close() })
		return db
	}
	db := open()
	_, err = book.NewService(book.NewSQLiteRepository(db), schedule.NewFake(now)).Create(ctx, book.CreateInput{ID: "book", Name: "Research", BeneficiaryIdentity: "operator"})
	require.NoError(t, err)
	repository := budget.NewSQLiteRepository(db)
	_, err = budget.NewService(repository, schedule.NewFake(now)).Create(ctx, budget.Budget{ID: "budget", BookID: "book", Currency: "USD", TotalCapMinor: 1000, PeriodicCapMinor: 1000, PerTransactionCapMinor: 1000, Period: time.Hour})
	require.NoError(t, err)
	w := wallet.Wallet{ID: "wallet", OperatorIdentity: "operator", BookID: "book", Network: baseusdc.Network, Address: "0x" + strings.Repeat("a", 40)}
	tx, block := "0x"+strings.Repeat("b", 64), "0x"+strings.Repeat("c", 64)
	check := baseusdc.Check{ID: "check", OperatorIdentity: "operator", WalletID: w.ID, Network: baseusdc.Network, TransactionHash: tx, ProviderID: "fork", Outcome: "verified", CheckedAt: now, Transfers: []baseusdc.Observation{{WalletID: w.ID, Network: baseusdc.Network, AssetID: baseusdc.TransferAsset, Decimals: 6, From: "0x" + strings.Repeat("d", 40), To: w.Address, AtomicAmount: "1000001", TransactionHash: tx, BlockHash: block, BlockTime: now, Finality: "finalized"}}}
	require.NoError(t, baseusdc.NewSQLiteStore(db).Append(ctx, check))
	in := SubmitInput{OperatorIdentity: "operator", WalletID: w.ID, TransactionHash: tx, CheckID: check.ID, BudgetID: "budget", LedgerBookID: "ledger-book", LedgerAccountID: "ledger-account", Action: ingestpb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_ORIGINAL, Valuation: &ingestpb.CryptoValuation{AmountMinor: 100, Currency: "USD", Basis: sharedpb.Basis_BASIS_OPERATOR_ASSERTED, BasisReference: "test-explicit-valuation", ValuedAt: timestamppb.New(now)}, Attribution: &ingestpb.CryptoPayerAttribution{PayerAddress: check.Transfers[0].From, PayerIdentity: "fixture:customer"}}
	var accepted *ingestpb.AcceptedCryptoReceipt
	calls := 0
	acceptor := acceptAndRead{acceptFunc: acceptFunc(func(_ context.Context, req *ingestpb.AcceptCryptoReceiptRequest) (*ingestpb.AcceptedCryptoReceipt, error) {
		calls++
		if accepted == nil {
			accepted = &ingestpb.AcceptedCryptoReceipt{Id: "ledger-receipt", PostingId: "posting", Source: proto.Clone(req.Source).(*ingestpb.TreasuryReceiptLink), Action: req.Action, AtomicDelta: req.Observation.AtomicAmount, ValuationDeltaMinor: 100, Currency: "USD", AcceptedAt: timestamppb.New(now), Observation: proto.Clone(req.Observation).(*ingestpb.PreviewCryptoObservationRequest), Valuation: proto.Clone(req.Valuation).(*ingestpb.CryptoValuation), Attribution: proto.Clone(req.Attribution).(*ingestpb.CryptoPayerAttribution), NoChargeback: true}
			return nil, errors.New("acceptance committed; response lost")
		}
		return proto.Clone(accepted).(*ingestpb.AcceptedCryptoReceipt), nil
	}), read: func(_ context.Context, id string) (*ingestpb.AcceptedCryptoReceipt, error) {
		if accepted == nil || id != accepted.Id {
			return nil, ErrNotFound
		}
		return proto.Clone(accepted).(*ingestpb.AcceptedCryptoReceipt), nil
	}}
	service := NewService(NewStore(db), baseusdc.NewSQLiteStore(db), walletReader{w}, repository, acceptor, func() time.Time { return now })
	grantCalls := 0
	service.SetCreditLotGranter(lotGrantFunc(func(_ context.Context, receipt *ingestpb.AcceptedCryptoReceipt) (CreditLotLink, error) {
		grantCalls++
		return CreditLotLink{ID: "lpbs-lot", Owner: "landing-page-business-suite", ReceiptID: receipt.GetId()}, nil
	}))
	delivery, err := service.Submit(ctx, in)
	require.NoError(t, err)
	require.Equal(t, "queued", delivery.Status)
	require.ErrorContains(t, service.DrainPending(ctx), "response lost")
	funding, err := repository.Funding(ctx, "budget")
	require.NoError(t, err)
	require.Zero(t, funding.CreditedMinor)
	require.NoError(t, db.Close())
	db = open()
	repository = budget.NewSQLiteRepository(db)
	service = NewService(NewStore(db), baseusdc.NewSQLiteStore(db), walletReader{w}, repository, acceptor, func() time.Time { return now })
	service.SetCreditLotGranter(lotGrantFunc(func(_ context.Context, receipt *ingestpb.AcceptedCryptoReceipt) (CreditLotLink, error) {
		grantCalls++
		return CreditLotLink{ID: "lpbs-lot", Owner: "landing-page-business-suite", ReceiptID: receipt.GetId()}, nil
	}))
	require.NoError(t, service.DrainPending(ctx))
	require.NoError(t, service.DrainPending(ctx))
	delivery, err = service.Get(ctx, delivery.ID, "operator")
	require.NoError(t, err)
	require.Equal(t, "accepted", delivery.Status)
	require.Equal(t, "lpbs-lot", delivery.CreditLotID)
	require.Equal(t, "landing-page-business-suite", delivery.CreditLotOwner)
	require.Equal(t, "ledger-receipt", delivery.CreditLotReceiptID)
	require.Equal(t, 2, calls)
	require.Equal(t, 1, grantCalls, "accepted receipt was granted only once across the Treasury store restart")
	funding, err = repository.Funding(ctx, "budget")
	require.NoError(t, err)
	require.EqualValues(t, 100, funding.CreditedMinor)
	require.Equal(t, "1000001", funding.AtomicAmount)
	require.EqualValues(t, 1, funding.ReceiptCount)
	_, err = service.Get(ctx, delivery.ID, "other-operator")
	require.ErrorIs(t, err, ErrNotFound)
	_, err = service.Submit(ctx, in)
	require.NoError(t, err)
	in.Valuation.AmountMinor++
	_, err = service.Submit(ctx, in)
	require.ErrorIs(t, err, ErrConflict)
	in.OperatorIdentity = "another-operator"
	_, err = service.Submit(ctx, in)
	require.ErrorIs(t, err, wallet.ErrMismatch)
}

func TestAcceptanceValidationRejectsIncompleteAndChangedTokenEvidence(t *testing.T) {
	request := &ingestpb.AcceptCryptoReceiptRequest{Observation: &ingestpb.PreviewCryptoObservationRequest{AtomicAmount: "1000001", SourceReference: "check-new"}, Source: &ingestpb.TreasuryReceiptLink{CheckId: "check-new", WalletId: "wallet", OperatorIdentity: "operator"}, Action: ingestpb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_ORIGINAL, Valuation: &ingestpb.CryptoValuation{Currency: "USD", AmountMinor: 100}, Attribution: &ingestpb.CryptoPayerAttribution{PayerAddress: "payer"}}
	receipt := &ingestpb.AcceptedCryptoReceipt{Id: "r", PostingId: "p", Source: &ingestpb.TreasuryReceiptLink{CheckId: "check-original", WalletId: "wallet", OperatorIdentity: "operator"}, Action: request.Action, AtomicDelta: "1000001", Currency: "USD", AcceptedAt: timestamppb.Now(), Observation: proto.Clone(request.Observation).(*ingestpb.PreviewCryptoObservationRequest), Valuation: request.Valuation, Attribution: request.Attribution, NoChargeback: true}
	receipt.Observation.SourceReference = "check-original"
	require.NoError(t, validateAcceptance(request, receipt))
	require.Error(t, validateAcceptance(request, nil))
	receipt.Observation.AtomicAmount = "1000002"
	require.ErrorContains(t, validateAcceptance(request, receipt), "differs")
	receipt.Observation = nil
	require.Error(t, validateAcceptance(request, receipt))
}
